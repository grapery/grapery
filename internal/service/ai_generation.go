package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

// AIGenerationService 统一的AI生成任务管理服务
// 负责记录AI能力使用数据，用于任务管理和Token计费
type AIGenerationService struct {
	repo                       domain.Repository
	geminiClient               *gemini.Client
	genAPI                     *genapi.GenAPI
	logger                     *zap.Logger
	metrics                    *telemetry.Metrics // Prometheus metrics (optional)
	quotaReservation           *RedisQuotaReservationService
	redisDistributedLock       *RedisDistributedLock
	asyncVideoCompletion       *AsyncVideoCompletionService // 异步视频完成处理服务
	enableQuotaReservation      bool // 是否启用配额预留（默认关闭，逐步迁移）
}

// NewAIGenerationService 创建AI生成服务
func NewAIGenerationService(repo domain.Repository, geminiClient *gemini.Client, genAPI *genapi.GenAPI, logger *zap.Logger) *AIGenerationService {
	svc := &AIGenerationService{
		repo:                   repo,
		geminiClient:           geminiClient,
		genAPI:                 genAPI,
		logger:                 logger,
		quotaReservation:       nil, // 在 SetRedisClient 中设置
		redisDistributedLock:   nil, // 在 SetRedisClient 中设置
		enableQuotaReservation: false, // 默认关闭，可通过配置启用
	}

	// 创建异步视频完成处理服务
	svc.asyncVideoCompletion = NewAsyncVideoCompletionService(svc, repo, logger)

	return svc
}

// SetRedisClient 设置 Redis 客户端（用于分布式锁和配额预留）
func (s *AIGenerationService) SetRedisClient(client *redis.Client) {
	s.redisDistributedLock = NewRedisDistributedLock(client, s.logger)
	s.quotaReservation = NewRedisQuotaReservationService(s.logger, s.repo, client)
	s.logger.Info("Redis services initialized for distributed locking and quota reservation")
}

// SetMetrics 设置 Prometheus metrics
func (s *AIGenerationService) SetMetrics(metrics *telemetry.Metrics) {
	s.metrics = metrics
}

// SetQuotaReservationEnabled 设置是否启用配额预留机制
func (s *AIGenerationService) SetQuotaReservationEnabled(enabled bool) {
	s.enableQuotaReservation = enabled
	s.logger.Info("quota reservation mechanism updated",
		zap.Bool("enabled", enabled))
}

// GetAsyncVideoCompletionService 获取异步视频完成处理服务
func (s *AIGenerationService) GetAsyncVideoCompletionService() *AsyncVideoCompletionService {
	return s.asyncVideoCompletion
}

// GenerateTextRequest 文本生成请求
type GenerateTextRequest struct {
	UserID            string
	OriginalPrompt    string
	SystemPrompt      string
	Model             string
	Temperature       float32
	MaxTokens         int32
	RelatedEntityID   string
	RelatedEntityType string
	Metadata          map[string]interface{}
}

// GenerateTextResult 文本生成结果
type GenerateTextResult struct {
	Text       string
	RecordID   string // AI生成记录ID
	TokensUsed int
	DurationMs int64
	Metadata   map[string]interface{}
}

// GenerateText 使用Gemini生成文本内容，并记录AI使用数据
func (s *AIGenerationService) GenerateText(ctx context.Context, req *GenerateTextRequest) (*GenerateTextResult, error) {
	// Check if gemini client is configured
	if s.geminiClient == nil {
		return nil, fmt.Errorf("Gemini client not configured")
	}

	startTime := time.Now()

	// ============== 配额预留和分布式锁 ==============
	// 估算 token 数量（使用 maxTokens 或默认值 1000）
	estimatedTokens := int(req.MaxTokens)
	if estimatedTokens == 0 {
		estimatedTokens = 1000 // 默认预留 1000 tokens
	}

	// 生成请求 ID 用于分布式锁
	requestID := fmt.Sprintf("text_gen_%s_%d", req.UserID, time.Now().UnixNano())
	lockKey := fmt.Sprintf("ai_generation_lock:text:%s", req.UserID)

	var reservation *QuotaReservation
	var lockAcquired bool

	// 1. 获取分布式锁（防止同一用户的并发请求）
	if s.enableQuotaReservation && s.redisDistributedLock != nil {
		var err error
		lockAcquired, err = s.redisDistributedLock.AcquireLock(ctx, lockKey, requestID, 30*time.Second)
		if err != nil {
			s.logger.Error("failed to acquire distributed lock",
				zap.String("userID", req.UserID),
				zap.Error(err))
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}
		if !lockAcquired {
			s.logger.Warn("another request is in progress, please wait",
				zap.String("userID", req.UserID))
			return nil, fmt.Errorf("another AI generation request is in progress, please try again later")
		}
		defer func() {
			if lockAcquired {
				s.redisDistributedLock.ReleaseLock(ctx, lockKey, requestID)
			}
		}()
	}

	// 2. 预留配额
	if s.enableQuotaReservation && s.quotaReservation != nil {
		var err error
		metadata := map[string]interface{}{
			"model":    req.Model,
			"prompt":   truncateString(req.OriginalPrompt, 100),
			"type":     "text",
			"requestID": requestID,
		}
		reservation, err = s.quotaReservation.ReserveQuota(ctx, req.UserID, estimatedTokens, "ai_text_generation", metadata)
		if err != nil {
			s.logger.Error("failed to reserve quota",
				zap.String("userID", req.UserID),
				zap.Int("estimatedTokens", estimatedTokens),
				zap.Error(err))
			return nil, fmt.Errorf("failed to reserve quota: %w", err)
		}
		s.logger.Info("quota reserved successfully",
			zap.String("reservationID", reservation.ReservationID),
			zap.String("userID", req.UserID),
			zap.Int("estimatedTokens", estimatedTokens))
	}

	// 确保在失败时释放预留
	defer func() {
		if reservation != nil && reservation.Status == "pending" {
			if err := s.quotaReservation.ReleaseQuota(ctx, reservation.ReservationID); err != nil {
				s.logger.Error("failed to release quota reservation",
					zap.String("reservationID", reservation.ReservationID),
					zap.Error(err))
			}
		}
	}()
	// ============== 配额预留和分布式锁结束 ==============

	// 1. 创建AI生成记录（状态：pending）
	record := &domain.AIGenerationRecord{
		UserID:            req.UserID,
		Type:              "text",
		Provider:          "gemini",
		Model:             req.Model,
		OriginalPrompt:    req.OriginalPrompt,
		SystemPrompt:      req.SystemPrompt,
		Status:            domain.AITaskStatusPending,
		RelatedEntityID:   req.RelatedEntityID,
		RelatedEntityType: req.RelatedEntityType,
		Metadata:          req.Metadata,
		CreatedAt:         startTime.Unix(),
		OutputResult:      "{}", // Initialize with valid empty JSON for MySQL
	}

	// 保存输入参数
	inputParams := map[string]interface{}{
		"prompt":      req.OriginalPrompt,
		"system":      req.SystemPrompt,
		"model":       req.Model,
		"temperature": req.Temperature,
		"maxTokens":   req.MaxTokens,
	}
	if inputJSON, err := json.Marshal(inputParams); err == nil {
		record.InputParams = string(inputJSON)
	}

	// 保存到数据库
	if err := s.repo.CreateAIGenerationRecord(ctx, record); err != nil {
		s.logger.Error("failed to create AI generation record", zap.Error(err))
		return nil, fmt.Errorf("failed to create AI record: %w", err)
	}

	// 2. 更新状态为processing
	processingTime := time.Now().Unix()
	record.Status = domain.AITaskStatusProcessing
	record.StartedAt = &processingTime
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)

	// 3. 调用AI生成
	genConfig := &genai.GenerateContentConfig{
		Temperature:     &req.Temperature,
		MaxOutputTokens: req.MaxTokens,
	}
	if req.SystemPrompt != "" {
		genConfig.SystemInstruction = &genai.Content{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{genai.NewPartFromText(req.SystemPrompt)},
		}
	}

	text, geminiResp, err := s.geminiClient.GenerateText(ctx, req.Model, req.OriginalPrompt, genConfig)

	completedTime := time.Now()
	durationMs := completedTime.Sub(startTime).Milliseconds()

	// 4. 更新记录状态
	if err != nil {
		// 生成失败
		record.Status = domain.AITaskStatusFailed
		record.ErrorMessage = err.Error()
		record.DurationMs = durationMs
		completedUnix := completedTime.Unix()
		record.CompletedAt = &completedUnix
		_ = s.repo.UpdateAIGenerationRecord(ctx, record)

		s.logger.Error("AI text generation failed",
			zap.String("recordId", record.ID),
			zap.Error(err))
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// 生成成功
	record.Status = domain.AITaskStatusCompleted
	record.Progress = 100
	record.DurationMs = durationMs
	completedUnix := completedTime.Unix()
	record.CompletedAt = &completedUnix

	// 记录Token使用量
	if geminiResp != nil && geminiResp.UsageMetadata != nil {
		record.InputTokens = int(geminiResp.UsageMetadata.PromptTokenCount)
		record.OutputTokens = int(geminiResp.UsageMetadata.CandidatesTokenCount)
		record.TotalTokens = int(geminiResp.UsageMetadata.TotalTokenCount)
	}

	// 保存输出结果
	outputResult := map[string]interface{}{
		"text":   text,
		"tokens": record.TotalTokens,
	}
	if outputJSON, err := json.Marshal(outputResult); err == nil {
		record.OutputResult = string(outputJSON)
	}

	// 更新记录
	if err := s.repo.UpdateAIGenerationRecord(ctx, record); err != nil {
		s.logger.Warn("failed to update AI generation record", zap.Error(err))
	}

	// ============== 确认配额预留 ==============
	// 如果启用了配额预留，确认实际使用量
	if s.enableQuotaReservation && reservation != nil && reservation.Status == "pending" {
		actualTokens := record.TotalTokens
		if actualTokens == 0 {
			actualTokens = 1 // 至少扣减 1 个 token
		}

		if err := s.quotaReservation.ConfirmQuota(ctx, reservation.ReservationID, actualTokens); err != nil {
			s.logger.Error("failed to confirm quota reservation",
				zap.String("reservationID", reservation.ReservationID),
				zap.String("recordId", record.ID),
				zap.Int("actualTokens", actualTokens),
				zap.Error(err))
			// 确认失败不影响返回结果，但需要记录警告
			// 预留会在过期时自动释放，多余 token 会退还
		} else {
			s.logger.Info("quota reservation confirmed",
				zap.String("reservationID", reservation.ReservationID),
				zap.String("recordId", record.ID),
				zap.Int("estimatedTokens", reservation.EstimatedTokens),
				zap.Int("actualTokens", actualTokens))
			// 预留已确认，清除引用以避免 defer 中释放
			reservation = nil
		}
	}
	// ============== 配额预留确认结束 ==============

	// Record metrics
	if s.metrics != nil {
		s.metrics.RecordAIGeneration("gemini", "text")
	}

	s.logger.Info("AI text generation completed",
		zap.String("recordId", record.ID),
		zap.Int("tokensUsed", record.TotalTokens),
		zap.Int64("durationMs", durationMs))

	return &GenerateTextResult{
		Text:       text,
		RecordID:   record.ID,
		TokensUsed: record.TotalTokens,
		DurationMs: durationMs,
	}, nil
}

// GenerateImageRequest 图片生成请求
type GenerateImageRequest struct {
	UserID            string
	Prompt            string
	Provider          string // gemini, hailuo, huoshan
	Model             string
	AspectRatio       string
	Size              string // Image dimensions (e.g., "1024x1024", "1280x720") - used by huoshan
	Quality           string
	OutputCount       int
	ReferenceImages   []string // Character or scene reference images for image-to-image generation
	RelatedEntityID   string
	RelatedEntityType string
	Metadata          map[string]interface{}
}

// GenerateImageResult 图片生成结果
type GenerateImageResult struct {
	ImageURLs  []string
	RecordID   string
	TokensUsed int
	DurationMs int64
}

// GenerateImage 使用GenAPI生成图片，并记录AI使用数据
func (s *AIGenerationService) GenerateImage(ctx context.Context, req *GenerateImageRequest) (*GenerateImageResult, error) {
	// Check if genAPI client is configured
	if s.genAPI == nil {
		return nil, fmt.Errorf("GenAPI client not configured")
	}

	startTime := time.Now()

	// ============== 配额预留和分布式锁 ==============
	// 估算 token 数量（图片生成通常按次数计费，每张图片约 1000 tokens）
	estimatedTokens := req.OutputCount * 1000
	if estimatedTokens == 0 {
		estimatedTokens = 1000 // 默认预留 1000 tokens（1张图片）
	}

	// 生成请求 ID 用于分布式锁
	requestID := fmt.Sprintf("image_gen_%s_%d", req.UserID, time.Now().UnixNano())
	lockKey := fmt.Sprintf("ai_generation_lock:image:%s", req.UserID)

	var reservation *QuotaReservation
	var lockAcquired bool

	// 1. 获取分布式锁（防止同一用户的并发请求）
	if s.enableQuotaReservation && s.redisDistributedLock != nil {
		var err error
		lockAcquired, err = s.redisDistributedLock.AcquireLock(ctx, lockKey, requestID, 60*time.Second)
		if err != nil {
			s.logger.Error("failed to acquire distributed lock for image generation",
				zap.String("userID", req.UserID),
				zap.Error(err))
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}
		if !lockAcquired {
			s.logger.Warn("another image generation request is in progress, please wait",
				zap.String("userID", req.UserID))
			return nil, fmt.Errorf("another AI generation request is in progress, please try again later")
		}
		defer func() {
			if lockAcquired {
				s.redisDistributedLock.ReleaseLock(ctx, lockKey, requestID)
			}
		}()
	}

	// 2. 预留配额
	if s.enableQuotaReservation && s.quotaReservation != nil {
		metadata := map[string]interface{}{
			"provider":   req.Provider,
			"model":      req.Model,
			"prompt":     truncateString(req.Prompt, 100),
			"type":       "image",
			"outputCount": req.OutputCount,
			"requestID":  requestID,
		}
		var err error
		reservation, err = s.quotaReservation.ReserveQuota(ctx, req.UserID, estimatedTokens, "ai_image_generation", metadata)
		if err != nil {
			s.logger.Error("failed to reserve quota for image generation",
				zap.String("userID", req.UserID),
				zap.Int("estimatedTokens", estimatedTokens),
				zap.Error(err))
			return nil, fmt.Errorf("failed to reserve quota: %w", err)
		}
		s.logger.Info("quota reserved successfully for image generation",
			zap.String("reservationID", reservation.ReservationID),
			zap.String("userID", req.UserID),
			zap.Int("estimatedTokens", estimatedTokens))
	}

	// 确保在失败时释放预留
	defer func() {
		if reservation != nil && reservation.Status == "pending" {
			if err := s.quotaReservation.ReleaseQuota(ctx, reservation.ReservationID); err != nil {
				s.logger.Error("failed to release quota reservation",
					zap.String("reservationID", reservation.ReservationID),
					zap.Error(err))
			}
		}
	}()
	// ============== 配额预留和分布式锁结束 ==============

	// 1. 创建AI生成记录
	record := &domain.AIGenerationRecord{
		UserID:            req.UserID,
		Type:              "image",
		Provider:          req.Provider,
		Model:             req.Model,
		OriginalPrompt:    req.Prompt,
		Status:            domain.AITaskStatusPending,
		RelatedEntityID:   req.RelatedEntityID,
		RelatedEntityType: req.RelatedEntityType,
		Metadata:          req.Metadata,
		CreatedAt:         startTime.Unix(),
		OutputResult:      "{}", // Initialize with valid empty JSON for MySQL
	}

	// 保存输入参数
	inputParams := map[string]interface{}{
		"prompt":          req.Prompt,
		"provider":        req.Provider,
		"model":           req.Model,
		"aspectRatio":     req.AspectRatio,
		"size":            req.Size,
		"quality":         req.Quality,
		"outputCount":     req.OutputCount,
		"referenceImages": req.ReferenceImages,
	}
	if inputJSON, err := json.Marshal(inputParams); err == nil {
		record.InputParams = string(inputJSON)
	}

	if err := s.repo.CreateAIGenerationRecord(ctx, record); err != nil {
		s.logger.Error("failed to create AI generation record", zap.Error(err))
		return nil, fmt.Errorf("failed to create AI record: %w", err)
	}

	// 2. 更新状态为processing
	processingTime := time.Now().Unix()
	record.Status = domain.AITaskStatusProcessing
	record.StartedAt = &processingTime
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)

	// 3. 调用GenAPI生成图片
	genReq := &genapi.GenerateRequest{
		Prompt:          req.Prompt,
		AspectRatio:     req.AspectRatio,
		Size:            req.Size,
		Quality:         req.Quality,
		OutputCount:     req.OutputCount,
		Model:           req.Model,
		ReferenceImages: req.ReferenceImages,
	}

	// Choose operation type based on whether reference images are provided
	if len(req.ReferenceImages) > 0 {
		genReq.Operation = genapi.OperationImageToImage
		// Use first reference image as the primary reference
		genReq.ReferenceImageURL = req.ReferenceImages[0]
	} else {
		genReq.Operation = genapi.OperationTextToImage
	}

	resp, err := s.genAPI.GenerateImage(ctx, req.Provider, genReq)

	completedTime := time.Now()
	durationMs := completedTime.Sub(startTime).Milliseconds()

	// 4. 更新记录
	if err != nil {
		record.Status = domain.AITaskStatusFailed
		record.ErrorMessage = err.Error()
		record.DurationMs = durationMs
		completedUnix := completedTime.Unix()
		record.CompletedAt = &completedUnix
		_ = s.repo.UpdateAIGenerationRecord(ctx, record)

		return nil, fmt.Errorf("AI image generation failed: %w", err)
	}

	// 生成成功
	record.Status = domain.AITaskStatusCompleted
	record.Progress = 100
	record.DurationMs = durationMs
	record.ImageCount = len(resp.ImageURLs)
	completedUnix := completedTime.Unix()
	record.CompletedAt = &completedUnix

	// 记录Token/图片使用量
	if resp.Usage != nil {
		record.TotalTokens = resp.Usage.TotalTokens
		record.ImageCount = resp.Usage.ImageCount
	}

	// Record metrics
	if s.metrics != nil {
		provider := req.Provider
		if provider == "" {
			provider = "unknown"
		}
		s.metrics.RecordAIGeneration(provider, "image")
	}

	// 上传图片到OSS并替换URL
	ossImageURLs := make([]string, 0, len(resp.ImageURLs))
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil && len(resp.ImageURLs) > 0 {
		for i, imageURL := range resp.ImageURLs {
			// 生成OSS object key
			objectKey := fmt.Sprintf("ai-generated/images/%s/%d.jpg", record.ID, i)

			// 上传到OSS
			ossURL, err := ossClient.UploadFileFromURL(objectKey, imageURL)
			if err != nil {
				s.logger.Warn("failed to upload image to OSS, using original URL",
					zap.String("recordId", record.ID),
					zap.Int("index", i),
					zap.String("originalURL", imageURL),
					zap.Error(err))
				// 上传失败时使用原始URL
				ossImageURLs = append(ossImageURLs, imageURL)
			} else {
				// 清理URL：移除查询参数，确保HTTPS
				ossURL = strings.Split(ossURL, "?")[0]
				ossURL = strings.ReplaceAll(ossURL, "http://", "https://")
				ossImageURLs = append(ossImageURLs, ossURL)
				s.logger.Debug("image uploaded to OSS",
					zap.String("recordId", record.ID),
					zap.Int("index", i),
					zap.String("ossURL", ossURL))
			}
		}
	} else {
		// OSS客户端不可用，使用原始URL
		if ossClient == nil {
			s.logger.Warn("OSS client not available, using original image URLs",
				zap.String("recordId", record.ID))
		}
		ossImageURLs = resp.ImageURLs
	}

	// 保存输出结果（使用OSS URL）
	outputResult := map[string]interface{}{
		"imageURLs":  ossImageURLs,
		"imageCount": record.ImageCount,
		"tokens":     record.TotalTokens,
	}
	if outputJSON, err := json.Marshal(outputResult); err == nil {
		record.OutputResult = string(outputJSON)
	}

	if err := s.repo.UpdateAIGenerationRecord(ctx, record); err != nil {
		s.logger.Warn("failed to update AI generation record", zap.Error(err))
	}

	// ============== 确认配额预留 ==============
	// 计算实际使用的 token 数量
	actualTokens := record.TotalTokens
	if actualTokens == 0 && record.ImageCount > 0 {
		actualTokens = record.ImageCount * 1000 // 每张图片 1000 tokens
	}
	if actualTokens == 0 {
		actualTokens = 1000 // 至少扣减 1000 tokens
	}

	// 如果启用了配额预留，确认实际使用量
	if s.enableQuotaReservation && reservation != nil && reservation.Status == "pending" {
		if err := s.quotaReservation.ConfirmQuota(ctx, reservation.ReservationID, actualTokens); err != nil {
			s.logger.Error("failed to confirm quota reservation for image generation",
				zap.String("reservationID", reservation.ReservationID),
				zap.String("recordId", record.ID),
				zap.Int("actualTokens", actualTokens),
				zap.Error(err))
			// 确认失败不影响返回结果，但需要记录警告
		} else {
			s.logger.Info("quota reservation confirmed for image generation",
				zap.String("reservationID", reservation.ReservationID),
				zap.String("recordId", record.ID),
				zap.Int("estimatedTokens", reservation.EstimatedTokens),
				zap.Int("actualTokens", actualTokens),
				zap.Int("imageCount", record.ImageCount))
			// 预留已确认，清除引用以避免 defer 中释放
			reservation = nil
		}
	} else {
		// 如果没有启用配额预留，直接扣减（向后兼容）
		_, err := s.repo.UpdateTokenBalance(ctx, req.UserID, -actualTokens, "ai_image_generation", fmt.Sprintf("AI image generation consumed %d tokens (%d images)", actualTokens, record.ImageCount))
		if err != nil {
			s.logger.Error("failed to deduct token balance for image generation",
				zap.String("userID", req.UserID),
				zap.String("recordId", record.ID),
				zap.Int("imageCount", record.ImageCount),
				zap.Int("tokensUsed", actualTokens),
				zap.Error(err))
		}
	}
	// ============== 配额预留确认结束 ==============

	s.logger.Info("AI image generation completed",
		zap.String("recordId", record.ID),
		zap.Int("imageCount", record.ImageCount),
		zap.Int64("durationMs", durationMs))

	return &GenerateImageResult{
		ImageURLs:  ossImageURLs,
		RecordID:   record.ID,
		TokensUsed: record.TotalTokens,
		DurationMs: durationMs,
	}, nil
}

// GenerateVideoRequest 视频生成请求
type GenerateVideoRequest struct {
	UserID            string
	Prompt            string
	Provider          string
	Model             string
	DurationSeconds   int
	AspectRatio       string
	ReferenceImageURL string // Start keyframe image (FirstFrameURL)
	EndFrameURL       string // End keyframe image (LastFrameURL) for keyframe video generation
	RelatedEntityID   string
	RelatedEntityType string
	Metadata          map[string]interface{}
}

// GenerateVideoResult 视频生成结果
type GenerateVideoResult struct {
	TaskID     string // 异步任务ID
	VideoURL   string // 如果同步完成
	RecordID   string
	DurationMs int64
}

// GenerateVideo 使用GenAPI生成视频，并记录AI使用数据
func (s *AIGenerationService) GenerateVideo(ctx context.Context, req *GenerateVideoRequest) (*GenerateVideoResult, error) {
	// Check if genAPI client is configured
	if s.genAPI == nil {
		return nil, fmt.Errorf("GenAPI client not configured")
	}

	startTime := time.Now()

	// ============== 配额预留和分布式锁 ==============
	// 估算 token 数量（视频生成通常按时长和复杂度计费，每个视频约 5000 tokens）
	estimatedTokens := 5000 // 默认预留 5000 tokens

	// 生成请求 ID 用于分布式锁
	requestID := fmt.Sprintf("video_gen_%s_%d", req.UserID, time.Now().UnixNano())
	lockKey := fmt.Sprintf("ai_generation_lock:video:%s", req.UserID)

	var reservation *QuotaReservation
	var lockAcquired bool

	// 1. 获取分布式锁（防止同一用户的并发请求）
	if s.enableQuotaReservation && s.redisDistributedLock != nil {
		var err error
		lockAcquired, err = s.redisDistributedLock.AcquireLock(ctx, lockKey, requestID, 120*time.Second)
		if err != nil {
			s.logger.Error("failed to acquire distributed lock for video generation",
				zap.String("userID", req.UserID),
				zap.Error(err))
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}
		if !lockAcquired {
			s.logger.Warn("another video generation request is in progress, please wait",
				zap.String("userID", req.UserID))
			return nil, fmt.Errorf("another AI generation request is in progress, please try again later")
		}
		defer func() {
			if lockAcquired {
				s.redisDistributedLock.ReleaseLock(ctx, lockKey, requestID)
			}
		}()
	}

	// 2. 预留配额
	if s.enableQuotaReservation && s.quotaReservation != nil {
		metadata := map[string]interface{}{
			"provider":          req.Provider,
			"model":             req.Model,
			"prompt":            truncateString(req.Prompt, 100),
			"type":              "video",
			"durationSeconds":   req.DurationSeconds,
			"hasReferenceImage": req.ReferenceImageURL != "",
			"hasEndFrame":       req.EndFrameURL != "",
			"requestID":         requestID,
		}
		var err error
		reservation, err = s.quotaReservation.ReserveQuota(ctx, req.UserID, estimatedTokens, "ai_video_generation", metadata)
		if err != nil {
			s.logger.Error("failed to reserve quota for video generation",
				zap.String("userID", req.UserID),
				zap.Int("estimatedTokens", estimatedTokens),
				zap.Error(err))
			return nil, fmt.Errorf("failed to reserve quota: %w", err)
		}
		s.logger.Info("quota reserved successfully for video generation",
			zap.String("reservationID", reservation.ReservationID),
			zap.String("userID", req.UserID),
			zap.Int("estimatedTokens", estimatedTokens))
	}

	// 保存 reservation ID 到 metadata，供异步完成时使用
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	if reservation != nil {
		req.Metadata["reservationID"] = reservation.ReservationID
		req.Metadata["estimatedTokens"] = estimatedTokens
	}
	// ============== 配额预留和分布式锁结束 ==============

	// 1. 创建AI生成记录
	record := &domain.AIGenerationRecord{
		UserID:            req.UserID,
		Type:              "video",
		Provider:          req.Provider,
		Model:             req.Model,
		OriginalPrompt:    req.Prompt,
		Status:            domain.AITaskStatusPending,
		RelatedEntityID:   req.RelatedEntityID,
		RelatedEntityType: req.RelatedEntityType,
		Metadata:          req.Metadata,
		CreatedAt:         startTime.Unix(),
		OutputResult:      "{}", // Initialize with valid empty JSON for MySQL
	}

	// 保存输入参数
	inputParams := map[string]interface{}{
		"prompt":            req.Prompt,
		"provider":          req.Provider,
		"model":             req.Model,
		"durationSeconds":   req.DurationSeconds,
		"aspectRatio":       req.AspectRatio,
		"referenceImageURL": req.ReferenceImageURL,
		"endFrameURL":       req.EndFrameURL,
	}
	if inputJSON, err := json.Marshal(inputParams); err == nil {
		record.InputParams = string(inputJSON)
	}

	if err := s.repo.CreateAIGenerationRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create AI record: %w", err)
	}

	// 2. 更新状态为processing
	processingTime := time.Now().Unix()
	record.Status = domain.AITaskStatusProcessing
	record.StartedAt = &processingTime
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)

	// 3. 调用GenAPI生成视频
	genReq := &genapi.GenerateRequest{
		Prompt:          req.Prompt,
		DurationSeconds: req.DurationSeconds,
		AspectRatio:     req.AspectRatio,
		Model:           req.Model,
	}

	// 根据参考图片选择操作类型
	// Priority: keyframe (both frames) > image_to_video (start frame only) > text_to_video (no frames)
	if req.ReferenceImageURL != "" && req.EndFrameURL != "" {
		// Keyframe video generation with start and end frames
		genReq.Operation = genapi.OperationKeyframeToVideo
		genReq.FirstFrameURL = req.ReferenceImageURL
		genReq.LastFrameURL = req.EndFrameURL
	} else if req.ReferenceImageURL != "" {
		// Image-to-video with start frame only
		genReq.Operation = genapi.OperationImageToVideo
		genReq.ReferenceImageURL = req.ReferenceImageURL
	} else {
		// Text-to-video with no reference images
		genReq.Operation = genapi.OperationTextToVideo
	}

	resp, err := s.genAPI.GenerateVideo(ctx, req.Provider, genReq)

	completedTime := time.Now()
	durationMs := completedTime.Sub(startTime).Milliseconds()

	// 4. 更新记录
	if err != nil {
		record.Status = domain.AITaskStatusFailed
		record.ErrorMessage = err.Error()
		record.DurationMs = durationMs
		completedUnix := completedTime.Unix()
		record.CompletedAt = &completedUnix
		_ = s.repo.UpdateAIGenerationRecord(ctx, record)

		return nil, fmt.Errorf("AI video generation failed: %w", err)
	}

	// 视频生成通常是异步的
	record.DurationMs = durationMs

	// 处理视频URL（同步完成时上传到OSS）
	ossVideoURL := resp.VideoURL
	if resp.Status == "completed" && resp.VideoURL != "" {
		// 同步完成，上传视频到OSS
		ossClient := aliyun.GetGlobalClient()
		if ossClient != nil {
			// 生成OSS object key
			objectKey := fmt.Sprintf("ai-generated/videos/%s.mp4", record.ID)

			// 上传到OSS
			uploadedURL, err := ossClient.UploadFileFromURL(objectKey, resp.VideoURL)
			if err != nil {
				s.logger.Warn("failed to upload video to OSS, using original URL",
					zap.String("recordId", record.ID),
					zap.String("originalURL", resp.VideoURL),
					zap.Error(err))
				// 上传失败时使用原始URL
			} else {
				// 清理URL：移除查询参数，确保HTTPS
				ossVideoURL = strings.Split(uploadedURL, "?")[0]
				ossVideoURL = strings.ReplaceAll(ossVideoURL, "http://", "https://")
				s.logger.Debug("video uploaded to OSS",
					zap.String("recordId", record.ID),
					zap.String("ossURL", ossVideoURL))
			}
		} else {
			s.logger.Warn("OSS client not available, using original video URL",
				zap.String("recordId", record.ID))
		}

		// 同步完成
		record.Status = domain.AITaskStatusCompleted
		record.Progress = 100
		record.VideoCount = 1
		completedUnix := completedTime.Unix()
		record.CompletedAt = &completedUnix
	} else {
		// 异步任务，保持processing状态
		// TaskID存储在metadata中，后续可以查询状态
		if record.Metadata == nil {
			record.Metadata = make(map[string]interface{})
		}
		record.Metadata["taskId"] = resp.TaskID
		record.Progress = resp.Progress
	}

	// 记录使用量
	if resp.Usage != nil {
		record.TotalTokens = resp.Usage.TotalTokens
		record.VideoCount = resp.Usage.VideoCount
	}

	// 保存输出结果（使用OSS URL）
	outputResult := map[string]interface{}{
		"taskId":   resp.TaskID,
		"videoURL": ossVideoURL,
		"status":   resp.Status,
		"progress": resp.Progress,
	}
	if outputJSON, err := json.Marshal(outputResult); err == nil {
		record.OutputResult = string(outputJSON)
	}

	if err := s.repo.UpdateAIGenerationRecord(ctx, record); err != nil {
		s.logger.Warn("failed to update AI generation record", zap.Error(err))
	}

	// ============== 确认配额预留 ==============
	// 计算实际使用的 token 数量
	actualTokens := record.TotalTokens
	if actualTokens == 0 && record.VideoCount > 0 {
		actualTokens = record.VideoCount * 5000 // 每个视频 5000 tokens
	}
	if actualTokens == 0 {
		actualTokens = 5000 // 至少扣减 5000 tokens
	}

	if resp.Status == "completed" {
		// 同步完成：确认配额预留
		if s.enableQuotaReservation && reservation != nil && reservation.Status == "pending" {
			if err := s.quotaReservation.ConfirmQuota(ctx, reservation.ReservationID, actualTokens); err != nil {
				s.logger.Error("failed to confirm quota reservation for video generation",
					zap.String("reservationID", reservation.ReservationID),
					zap.String("recordId", record.ID),
					zap.Int("actualTokens", actualTokens),
					zap.Error(err))
			} else {
				s.logger.Info("quota reservation confirmed for video generation",
					zap.String("reservationID", reservation.ReservationID),
					zap.String("recordId", record.ID),
					zap.Int("estimatedTokens", reservation.EstimatedTokens),
					zap.Int("actualTokens", actualTokens))
				// 预留已确认，清除引用以避免 defer 中释放
				reservation = nil
			}
		} else {
			// 如果没有启用配额预留，直接扣减（向后兼容）
			_, err := s.repo.UpdateTokenBalance(ctx, req.UserID, -actualTokens, "ai_video_generation", fmt.Sprintf("AI video generation consumed %d tokens", actualTokens))
			if err != nil {
				s.logger.Error("failed to deduct token balance for video generation",
					zap.String("userID", req.UserID),
					zap.String("recordId", record.ID),
					zap.Int("tokensUsed", actualTokens),
					zap.Error(err))
			}
		}
	} else {
		// 异步任务 - 配额将在任务完成时确认
		// 注册到轮询服务
		reservationID := ""
		estimatedTokens := 0
		if reservation != nil {
			reservationID = reservation.ReservationID
			estimatedTokens = reservation.EstimatedTokens
			// 防止 defer 释放预留，因为异步任务还需要确认
			reservation = nil
		}

		s.asyncVideoCompletion.RegisterTask(
			resp.TaskID,
			record.ID,
			req.UserID,
			req.Provider,
			reservationID,
			estimatedTokens,
		)

		s.logger.Info("async video task registered for polling",
			zap.String("taskId", resp.TaskID),
			zap.String("recordId", record.ID),
			zap.String("reservationID", reservationID),
			zap.Int("estimatedTokens", estimatedTokens))
	}
	// ============== 配额预留确认结束 ==============

	s.logger.Info("AI video generation initiated",
		zap.String("recordId", record.ID),
		zap.String("taskId", resp.TaskID),
		zap.Int64("durationMs", durationMs))

	return &GenerateVideoResult{
		TaskID:     resp.TaskID,
		VideoURL:   ossVideoURL,
		RecordID:   record.ID,
		DurationMs: durationMs,
	}, nil
}

// GetAIGenerationRecord 获取AI生成记录
func (s *AIGenerationService) GetAIGenerationRecord(ctx context.Context, recordID string) (*domain.AIGenerationRecord, error) {
	return s.repo.GetAIGenerationRecord(ctx, recordID)
}

// ListUserAIGenerationRecords 获取用户的AI生成记录列表
func (s *AIGenerationService) ListUserAIGenerationRecords(ctx context.Context, userID string, limit, offset int) ([]*domain.AIGenerationRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListAIGenerationRecords(ctx, userID, limit, offset)
}

// GetUserTokenUsage 获取用户的Token使用统计
func (s *AIGenerationService) GetUserTokenUsage(ctx context.Context, userID string, startTime, endTime int64) (map[string]interface{}, error) {
	records, err := s.repo.ListAIGenerationRecordsByTimeRange(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// 统计
	stats := map[string]interface{}{
		"totalTokens":   0,
		"totalImages":   0,
		"totalVideos":   0,
		"totalRequests": len(records),
		"byProvider":    make(map[string]int),
		"byType":        make(map[string]int),
	}

	totalTokens := 0
	totalImages := 0
	totalVideos := 0
	byProvider := make(map[string]int)
	byType := make(map[string]int)

	for _, record := range records {
		totalTokens += record.TotalTokens
		totalImages += record.ImageCount
		totalVideos += record.VideoCount
		byProvider[record.Provider] += record.TotalTokens
		byType[record.Type] += record.TotalTokens
	}

	stats["totalTokens"] = totalTokens
	stats["totalImages"] = totalImages
	stats["totalVideos"] = totalVideos
	stats["byProvider"] = byProvider
	stats["byType"] = byType

	return stats, nil
}

// ============== Keyframe Subdivision Functions ==============

// EvaluateFrameGap uses VLM to assess whether two frames can transition smoothly in a short video clip.
// Returns: gapTooLarge (bool), evaluation details, error
func (s *AIGenerationService) EvaluateFrameGap(ctx context.Context, frameA, frameB string) (*genapi.FrameGapEvaluation, error) {
	if s.geminiClient == nil {
		return nil, fmt.Errorf("Gemini client not configured for VLM evaluation")
	}

	s.logger.Info("evaluating frame gap with VLM",
		zap.String("frameA", truncateURL(frameA)),
		zap.String("frameB", truncateURL(frameB)))

	// Download both images
	imageAData, mimeTypeA, err := downloadImageFromURL(ctx, frameA)
	if err != nil {
		s.logger.Error("failed to download frame A",
			zap.String("url", frameA),
			zap.Error(err))
		return nil, fmt.Errorf("download frame A: %w", err)
	}

	imageBData, mimeTypeB, err := downloadImageFromURL(ctx, frameB)
	if err != nil {
		s.logger.Error("failed to download frame B",
			zap.String("url", frameB),
			zap.Error(err))
		return nil, fmt.Errorf("download frame B: %w", err)
	}

	// Build VLM prompt for gap evaluation
	evaluationPrompt := `Analyze the visual difference between Image A and Image B to determine if a smooth video transition is possible.

Evaluation criteria:
1. Character pose change magnitude (standing/sitting/lying down - major posture changes)
2. Character position movement distance (whether crossing the scene)
3. Viewpoint/angle change (front to side, etc.)
4. Scene change (whether the location changed)

Question: Can the action between these two images be naturally completed in a single 5-second continuous shot?

IMPORTANT: Respond ONLY with valid JSON in this exact format (no markdown, no explanation):
{"feasible": true, "reason": "brief reason", "middleAction": "description of needed middle state if not feasible, empty string if feasible"}`

	// Build multimodal content with both images
	partA, err := encodeImagePart(imageAData, mimeTypeA)
	if err != nil {
		return nil, fmt.Errorf("encode frame A: %w", err)
	}
	partB, err := encodeImagePart(imageBData, mimeTypeB)
	if err != nil {
		return nil, fmt.Errorf("encode frame B: %w", err)
	}

	contents := []*genai.Content{{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			genai.NewPartFromText("Image A (Start Frame):"),
			partA,
			genai.NewPartFromText("Image B (End Frame):"),
			partB,
			genai.NewPartFromText(evaluationPrompt),
		},
	}}

	// Call Gemini with multimodal content
	config := &genai.GenerateContentConfig{
		Temperature:      floatPtr(0.3), // Low temperature for consistent evaluation
		ResponseMIMEType: "application/json",
	}

	resp, err := s.geminiClient.SDK().Models.GenerateContent(ctx, "gemini-2.5-flash", contents, config)
	if err != nil {
		s.logger.Error("VLM evaluation failed",
			zap.Error(err))
		return nil, fmt.Errorf("VLM evaluation failed: %w", err)
	}

	// Extract text response
	responseText := strings.TrimSpace(resp.Text())
	s.logger.Debug("VLM evaluation raw response",
		zap.String("response", responseText))

	// Parse JSON response
	var evaluation genapi.FrameGapEvaluation
	if err := json.Unmarshal([]byte(responseText), &evaluation); err != nil {
		// Try to extract JSON from response
		cleanedResponse := extractJSON(responseText)
		if err2 := json.Unmarshal([]byte(cleanedResponse), &evaluation); err2 != nil {
			s.logger.Warn("failed to parse VLM evaluation response, defaulting to feasible",
				zap.String("response", responseText),
				zap.Error(err2))
			// Default to feasible to avoid unnecessary subdivision
			return &genapi.FrameGapEvaluation{
				Feasible:     true,
				Reason:       "Unable to parse VLM response, defaulting to feasible",
				MiddleAction: "",
			}, nil
		}
	}

	s.logger.Info("frame gap evaluation completed",
		zap.Bool("feasible", evaluation.Feasible),
		zap.String("reason", evaluation.Reason),
		zap.String("middleAction", evaluation.MiddleAction))

	return &evaluation, nil
}

// GenerateMiddleFrame generates an intermediate frame image based on start frame and action description.
func (s *AIGenerationService) GenerateMiddleFrame(ctx context.Context, req *genapi.MiddleFrameRequest) (string, error) {
	if s.genAPI == nil {
		return "", fmt.Errorf("GenAPI not configured for middle frame generation")
	}

	s.logger.Info("generating middle frame",
		zap.String("startFrame", truncateURL(req.StartFrameURL)),
		zap.String("middleAction", req.MiddleAction))

	// Build image generation prompt
	prompt := fmt.Sprintf(`Generate an intermediate state image based on the reference image.

Middle action to depict: %s

Requirements:
1. Maintain exact same character appearance, clothing, and features
2. Keep the scene environment consistent
3. Only change the pose/position to the described middle state
4. Preserve lighting and color palette`, req.MiddleAction)

	// Determine provider
	provider := req.Provider
	if provider == "" {
		provider = "gemini" // Default to gemini for image generation
	}

	// Build image-to-image request
	genReq := &genapi.GenerateRequest{
		Operation:         genapi.OperationImageToImage,
		Prompt:            prompt,
		ReferenceImageURL: req.StartFrameURL,
		AspectRatio:       req.AspectRatio,
		OutputCount:       1,
	}

	resp, err := s.genAPI.GenerateImage(ctx, provider, genReq)
	if err != nil {
		s.logger.Error("middle frame generation failed",
			zap.Error(err))
		return "", fmt.Errorf("middle frame generation failed: %w", err)
	}

	if len(resp.ImageURLs) == 0 {
		return "", fmt.Errorf("no image generated for middle frame")
	}

	imageURL := resp.ImageURLs[0]

	// Upload to OSS for persistence
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil {
		objectKey := fmt.Sprintf("keyframe-subdivision/middle-frames/%d.png", time.Now().UnixNano())
		ossURL, err := ossClient.UploadFileFromURL(objectKey, imageURL)
		if err != nil {
			s.logger.Warn("failed to upload middle frame to OSS, using original URL",
				zap.Error(err))
		} else {
			imageURL = ossURL
		}
	}

	s.logger.Info("middle frame generated successfully",
		zap.String("imageURL", truncateURL(imageURL)))

	return imageURL, nil
}

// helper functions

func truncateURL(url string) string {
	if len(url) > 80 {
		return url[:80] + "..."
	}
	return url
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func floatPtr(f float32) *float32 {
	return &f
}

func encodeImagePart(data []byte, mimeType string) (*genai.Part, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("image data is empty")
	}
	return &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: mimeType,
			Data:     data,
		},
	}, nil
}

func extractJSON(text string) string {
	// Try to find JSON object in text
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return text
}

func downloadImageFromURL(ctx context.Context, url string) ([]byte, string, error) {
	if url == "" {
		return nil, "", fmt.Errorf("empty URL")
	}

	// Use http.Get with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read image data: %w", err)
	}

	// Determine MIME type
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		// Try to detect from content
		mimeType = http.DetectContentType(data)
	}

	return data, mimeType, nil
}

// ============== Keyframe Subdivision Core Logic ==============

const (
	defaultMaxSubdivisionDepth = 1 // Maximum recursion depth (1 = up to 2 segments, 2 = up to 4)
	defaultSegmentDuration     = 5 // Default duration per segment in seconds
	maxAllowedSegments         = 3 // Hard limit on segments to control cost
)

// GenerateVideoWithSubdivision generates video with automatic keyframe subdivision.
// When the gap between start and end frames is too large, it recursively generates
// middle frames and produces multiple video segments for smooth transitions.
func (s *AIGenerationService) GenerateVideoWithSubdivision(ctx context.Context, req *genapi.SubdivisionVideoRequest) (*genapi.KeyframeSubdivisionResult, error) {
	if s.genAPI == nil {
		return nil, fmt.Errorf("GenAPI not configured")
	}

	// Set defaults
	maxDepth := req.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxSubdivisionDepth
	}
	durationSecs := req.DurationSeconds
	if durationSecs <= 0 {
		durationSecs = defaultSegmentDuration
	}

	s.logger.Info("starting video generation with subdivision",
		zap.String("firstFrame", truncateURL(req.FirstFrameURL)),
		zap.String("lastFrame", truncateURL(req.LastFrameURL)),
		zap.String("prompt", req.Prompt),
		zap.Int("maxDepth", maxDepth),
		zap.Int("maxSegments", maxAllowedSegments))

	// Initialize result
	result := &genapi.KeyframeSubdivisionResult{
		Segments:      []genapi.VideoSegment{},
		MiddleFrames:  []string{},
		TotalDuration: 0,
		IsSubdivided:  false,
	}

	// Start recursive subdivision with segment limit
	segments, middleFrames, err := s.subdivideAndGenerate(ctx, subdivisionParams{
		startFrame:      req.FirstFrameURL,
		endFrame:        req.LastFrameURL,
		prompt:          req.Prompt,
		durationSecs:    durationSecs,
		provider:        req.Provider,
		aspectRatio:     req.AspectRatio,
		currentDepth:    0,
		maxDepth:        maxDepth,
		segmentIndex:    0,
		currentSegments: 0,
		maxSegments:     maxAllowedSegments, // Limit to 3 segments max
	})

	if err != nil {
		return nil, err
	}

	result.Segments = segments
	result.MiddleFrames = middleFrames
	result.IsSubdivided = len(middleFrames) > 0
	for _, seg := range segments {
		result.TotalDuration += seg.DurationSecs
	}

	s.logger.Info("video generation with subdivision completed",
		zap.Int("segmentCount", len(result.Segments)),
		zap.Int("middleFrameCount", len(result.MiddleFrames)),
		zap.Int("totalDuration", result.TotalDuration),
		zap.Bool("isSubdivided", result.IsSubdivided))

	return result, nil
}

type subdivisionParams struct {
	startFrame      string
	endFrame        string
	prompt          string
	durationSecs    int
	provider        string
	aspectRatio     string
	currentDepth    int
	maxDepth        int
	segmentIndex    int
	currentSegments int // Track current segment count to enforce limit
	maxSegments     int // Maximum allowed segments
}

// subdivideAndGenerate recursively evaluates frame gaps and generates video segments.
// Returns: video segments, middle frame URLs, error
func (s *AIGenerationService) subdivideAndGenerate(ctx context.Context, params subdivisionParams) ([]genapi.VideoSegment, []string, error) {
	s.logger.Debug("subdivision step",
		zap.Int("depth", params.currentDepth),
		zap.Int("maxDepth", params.maxDepth),
		zap.Int("currentSegments", params.currentSegments),
		zap.Int("maxSegments", params.maxSegments),
		zap.String("startFrame", truncateURL(params.startFrame)),
		zap.String("endFrame", truncateURL(params.endFrame)))

	// Check segment limit - if we would exceed max segments, generate directly
	// With subdivision, we'd create at least 2 segments, so check if we have room
	remainingSlots := params.maxSegments - params.currentSegments
	if remainingSlots <= 1 {
		s.logger.Debug("segment limit reached, generating video directly",
			zap.Int("currentSegments", params.currentSegments),
			zap.Int("maxSegments", params.maxSegments))
		segment, err := s.generateSingleVideoSegment(ctx, params)
		if err != nil {
			return nil, nil, err
		}
		return []genapi.VideoSegment{segment}, nil, nil
	}

	// Check recursion limit
	if params.currentDepth >= params.maxDepth {
		s.logger.Debug("max depth reached, generating video directly",
			zap.Int("depth", params.currentDepth))
		segment, err := s.generateSingleVideoSegment(ctx, params)
		if err != nil {
			return nil, nil, err
		}
		return []genapi.VideoSegment{segment}, nil, nil
	}

	// Evaluate frame gap using VLM
	evaluation, err := s.EvaluateFrameGap(ctx, params.startFrame, params.endFrame)
	if err != nil {
		s.logger.Warn("frame gap evaluation failed, generating video directly",
			zap.Error(err))
		// Fall back to direct generation
		segment, err := s.generateSingleVideoSegment(ctx, params)
		if err != nil {
			return nil, nil, err
		}
		return []genapi.VideoSegment{segment}, nil, nil
	}

	// If gap is feasible, generate video directly
	if evaluation.Feasible {
		s.logger.Debug("frame gap is feasible, generating video directly")
		segment, err := s.generateSingleVideoSegment(ctx, params)
		if err != nil {
			return nil, nil, err
		}
		return []genapi.VideoSegment{segment}, nil, nil
	}

	// Gap is too large, need to subdivide
	s.logger.Info("frame gap too large, generating middle frame",
		zap.String("reason", evaluation.Reason),
		zap.String("middleAction", evaluation.MiddleAction))

	// Generate middle frame
	middleFrameURL, err := s.GenerateMiddleFrame(ctx, &genapi.MiddleFrameRequest{
		StartFrameURL: params.startFrame,
		EndFrameURL:   params.endFrame,
		MiddleAction:  evaluation.MiddleAction,
		Provider:      params.provider,
		AspectRatio:   params.aspectRatio,
	})
	if err != nil {
		s.logger.Warn("middle frame generation failed, generating video directly",
			zap.Error(err))
		// Fall back to direct generation
		segment, err := s.generateSingleVideoSegment(ctx, params)
		if err != nil {
			return nil, nil, err
		}
		return []genapi.VideoSegment{segment}, nil, nil
	}

	// Recursively process first half: startFrame -> middleFrame
	firstHalfParams := subdivisionParams{
		startFrame:      params.startFrame,
		endFrame:        middleFrameURL,
		prompt:          params.prompt,
		durationSecs:    params.durationSecs,
		provider:        params.provider,
		aspectRatio:     params.aspectRatio,
		currentDepth:    params.currentDepth + 1,
		maxDepth:        params.maxDepth,
		segmentIndex:    params.segmentIndex,
		currentSegments: params.currentSegments,
		maxSegments:     params.maxSegments,
	}
	firstSegments, firstMiddleFrames, err := s.subdivideAndGenerate(ctx, firstHalfParams)
	if err != nil {
		return nil, nil, fmt.Errorf("first half subdivision failed: %w", err)
	}

	// Recursively process second half: middleFrame -> endFrame
	// Update currentSegments to include segments from first half
	secondHalfParams := subdivisionParams{
		startFrame:      middleFrameURL,
		endFrame:        params.endFrame,
		prompt:          params.prompt,
		durationSecs:    params.durationSecs,
		provider:        params.provider,
		aspectRatio:     params.aspectRatio,
		currentDepth:    params.currentDepth + 1,
		maxDepth:        params.maxDepth,
		segmentIndex:    params.segmentIndex + len(firstSegments),
		currentSegments: params.currentSegments + len(firstSegments), // Track segments generated so far
		maxSegments:     params.maxSegments,
	}
	secondSegments, secondMiddleFrames, err := s.subdivideAndGenerate(ctx, secondHalfParams)
	if err != nil {
		return nil, nil, fmt.Errorf("second half subdivision failed: %w", err)
	}

	// Combine results
	allSegments := append(firstSegments, secondSegments...)
	allMiddleFrames := append([]string{middleFrameURL}, firstMiddleFrames...)
	allMiddleFrames = append(allMiddleFrames, secondMiddleFrames...)

	// Re-index segments
	for i := range allSegments {
		allSegments[i].Index = i
	}

	return allSegments, allMiddleFrames, nil
}

// generateSingleVideoSegment generates a single video segment using keyframe-to-video.
// For async providers (like hailuo), it polls until video generation completes.
func (s *AIGenerationService) generateSingleVideoSegment(ctx context.Context, params subdivisionParams) (genapi.VideoSegment, error) {
	s.logger.Debug("generating single video segment",
		zap.String("startFrame", truncateURL(params.startFrame)),
		zap.String("endFrame", truncateURL(params.endFrame)),
		zap.Int("duration", params.durationSecs))

	provider := params.provider
	if provider == "" {
		provider = "hailuo" // Default video provider
	}

	genReq := &genapi.GenerateRequest{
		Operation:       genapi.OperationKeyframeToVideo,
		FirstFrameURL:   params.startFrame,
		LastFrameURL:    params.endFrame,
		Prompt:          params.prompt,
		DurationSeconds: params.durationSecs,
		AspectRatio:     params.aspectRatio,
	}

	resp, err := s.genAPI.GenerateVideo(ctx, provider, genReq)
	if err != nil {
		return genapi.VideoSegment{}, fmt.Errorf("video generation failed: %w", err)
	}

	videoURL := resp.VideoURL

	// Handle async video generation (TaskID returned instead of VideoURL)
	if videoURL == "" && resp.TaskID != "" {
		s.logger.Info("video generation is async, polling for completion",
			zap.String("taskId", resp.TaskID),
			zap.String("provider", provider))

		videoURL, err = s.pollForVideoCompletion(ctx, provider, resp.TaskID)
		if err != nil {
			return genapi.VideoSegment{}, fmt.Errorf("async video generation polling failed: %w", err)
		}
	}

	// Check if we got a video URL
	if videoURL == "" {
		return genapi.VideoSegment{}, fmt.Errorf("video generation did not return a video URL")
	}

	// Upload to OSS for persistence
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil && videoURL != "" {
		objectKey := fmt.Sprintf("keyframe-subdivision/videos/%d.mp4", time.Now().UnixNano())
		ossURL, err := ossClient.UploadFileFromURL(objectKey, videoURL)
		if err != nil {
			s.logger.Warn("failed to upload video segment to OSS, using original URL",
				zap.Error(err))
		} else {
			videoURL = ossURL
		}
	}

	segment := genapi.VideoSegment{
		Index:        params.segmentIndex,
		VideoURL:     videoURL,
		StartFrame:   params.startFrame,
		EndFrame:     params.endFrame,
		DurationSecs: params.durationSecs,
	}

	s.logger.Debug("video segment generated",
		zap.Int("index", segment.Index),
		zap.String("videoURL", truncateURL(videoURL)))

	return segment, nil
}

// pollForVideoCompletion polls the video generation task until it completes.
// This is used for async video providers like hailuo.
func (s *AIGenerationService) pollForVideoCompletion(ctx context.Context, provider, taskID string) (string, error) {
	const (
		maxPollAttempts = 120             // Max poll attempts (10 minutes at 5s interval)
		pollInterval    = 5 * time.Second // Polling interval
	)

	for attempt := 1; attempt <= maxPollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		s.logger.Debug("polling video generation status",
			zap.String("taskId", taskID),
			zap.String("provider", provider),
			zap.Int("attempt", attempt))

		resp, err := s.genAPI.GetVideoStatus(ctx, provider, taskID)
		if err != nil {
			s.logger.Warn("poll request failed, will retry",
				zap.String("taskId", taskID),
				zap.Int("attempt", attempt),
				zap.Error(err))
			time.Sleep(pollInterval)
			continue
		}

		// Check if completed
		if resp.VideoURL != "" {
			s.logger.Info("video generation completed",
				zap.String("taskId", taskID),
				zap.String("videoURL", truncateURL(resp.VideoURL)))
			return resp.VideoURL, nil
		}

		// Check if failed
		if resp.Status == "failed" || resp.Status == "error" {
			errMsg := resp.Error
			if errMsg == "" {
				errMsg = resp.Message
			}
			return "", fmt.Errorf("video generation failed: %s", errMsg)
		}

		// Still processing, wait and retry
		time.Sleep(pollInterval)
	}

	return "", fmt.Errorf("video generation timed out after %d attempts", maxPollAttempts)
}
