package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

// AIGenerationService 统一的AI生成任务管理服务
// 负责记录AI能力使用数据，用于任务管理和Token计费
type AIGenerationService struct {
	repo         domain.Repository
	geminiClient *gemini.Client
	genAPI       *genapi.GenAPI
	logger       *zap.Logger
}

// NewAIGenerationService 创建AI生成服务
func NewAIGenerationService(repo domain.Repository, geminiClient *gemini.Client, genAPI *genapi.GenAPI, logger *zap.Logger) *AIGenerationService {
	return &AIGenerationService{
		repo:         repo,
		geminiClient: geminiClient,
		genAPI:       genAPI,
		logger:       logger,
	}
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
