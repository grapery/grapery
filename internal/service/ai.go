package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	huoshanclient "github.com/grapestree/fgrapery/grapery/internal/genai/providers/huoshan"
)

// AIService AI 生成服务
type AIService struct {
	genAPI               *genapi.GenAPI       // 用于直连能力探测（如火山对话）
	geminiClient         *gemini.Client       // 文本回退
	aiGen                *AIGenerationService // 碎片配图等：统一配额与 AIGenerationRecord
	defaultImageProvider string               // 与 cfg.AI.ImageProvider 对齐
	defaultVideoProvider string               // 与 cfg.AI.VideoProvider 对齐
	repo                 domain.Repository
	logger               *zap.Logger
	textAdmission        *AITextAdmissionGate // global text LLM concurrency (optional; shared with AIGenerationService)
}

// NewAIService 创建 AI 服务
func NewAIService(genAPI *genapi.GenAPI, geminiClient *gemini.Client, aiGen *AIGenerationService, defaultImageProvider, defaultVideoProvider string, repo domain.Repository, logger *zap.Logger, admission *AITextAdmissionGate) *AIService {
	dp := strings.TrimSpace(defaultImageProvider)
	if dp == "" {
		dp = "huoshan"
	}
	vp := strings.TrimSpace(defaultVideoProvider)
	if vp == "" {
		vp = "huoshan"
	}
	return &AIService{
		genAPI:               genAPI,
		geminiClient:         geminiClient,
		aiGen:                aiGen,
		defaultImageProvider: dp,
		defaultVideoProvider: vp,
		repo:                 repo,
		logger:               logger,
		textAdmission:        admission,
	}
}

func (s *AIService) acquireTextAdmission(ctx context.Context) (func(), error) {
	if s == nil || s.textAdmission == nil {
		return func() {}, nil
	}
	return s.textAdmission.Acquire(ctx)
}

// ============== 故事生成 ==============

// GenerateStory 生成故事内容
func (s *AIService) GenerateStory(ctx context.Context, userID string, req *domain.AIStoryGenerationRequest) (*domain.AITask, error) {
	// 创建 AI 任务
	task := &domain.AITask{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      domain.AITaskGenerateStory,
		Status:    domain.AITaskStatusPending,
		Provider:  "", // 使用默认提供商
		Progress:  0,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 序列化输入参数
	inputJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}
	task.Input = string(inputJSON)

	// 保存任务到数据库
	if err := s.repo.CreateAITask(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// 启动异步生成
	go s.processStoryGeneration(context.Background(), task, req)

	return task, nil
}

// processStoryGeneration 处理故事生成（异步）
func (s *AIService) processStoryGeneration(ctx context.Context, task *domain.AITask, req *domain.AIStoryGenerationRequest) {
	// 更新任务状态为处理中
	task.Status = domain.AITaskStatusProcessing
	task.Progress = 10
	startTime := time.Now().Unix()
	task.StartedAt = &startTime
	task.UpdatedAt = time.Now().Unix()
	if err := s.repo.UpdateAITask(ctx, task); err != nil {
		s.logger.Error("更新任务状态失败", zap.String("taskId", task.ID), zap.Error(err))
	}

	s.logger.Info("开始生成故事",
		zap.String("taskId", task.ID),
		zap.String("userId", task.UserID),
		zap.String("prompt", req.Prompt),
	)

	// 构建 AI 提示词
	prompt := s.buildStoryPrompt(req)

	// 配置生成参数
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7 // 默认温度
	}
	tempFloat32 := float32(temperature)
	maxTokens := int32(2000)

	genConfig := &genai.GenerateContentConfig{
		Temperature:     &tempFloat32,
		MaxOutputTokens: maxTokens,
	}

	task.Progress = 30
	task.UpdatedAt = time.Now().Unix()
	_ = s.repo.UpdateAITaskProgress(ctx, task.ID, task.Progress)

	// 调用 Gemini 生成文本
	text, geminiResp, err := s.geminiClient.GenerateText(ctx, "", prompt, genConfig)
	if err != nil {
		s.logger.Error("AI 生成失败",
			zap.String("taskId", task.ID),
			zap.Error(err),
		)
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = err.Error()
		task.Progress = 0
		task.UpdatedAt = time.Now().Unix()
		_ = s.repo.UpdateAITask(ctx, task)
		return
	}

	task.Progress = 80

	// 计算 token 使用量
	tokensUsed := 0
	if geminiResp != nil && geminiResp.UsageMetadata != nil {
		tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
	}

	// 解析生成结果
	result := s.parseStoryResult(text, req)
	result.TokensUsed = tokensUsed

	// 序列化输出结果
	outputJSON, err := json.Marshal(result)
	if err != nil {
		s.logger.Error("序列化输出失败",
			zap.String("taskId", task.ID),
			zap.Error(err),
		)
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = fmt.Sprintf("marshal output: %v", err)
		task.UpdatedAt = time.Now().Unix()
		_ = s.repo.UpdateAITask(ctx, task)
		return
	}

	// 更新任务完成
	task.Status = domain.AITaskStatusCompleted
	task.Progress = 100
	task.Output = string(outputJSON)
	task.TokensUsed = tokensUsed
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime
	task.UpdatedAt = time.Now().Unix()
	_ = s.repo.UpdateAITask(ctx, task)

	s.logger.Info("故事生成完成",
		zap.String("taskId", task.ID),
		zap.Int("tokensUsed", tokensUsed),
		zap.Duration("duration", time.Duration(time.Now().Unix()-startTime)*time.Second),
	)
}

// buildStoryPrompt 构建故事生成提示词
func (s *AIService) buildStoryPrompt(req *domain.AIStoryGenerationRequest) string {
	prompt := "作为一位专业的故事创作者，请根据以下要求创作一个精彩的故事：\n\n"
	prompt += fmt.Sprintf("核心提示: %s\n\n", req.Prompt)

	if req.Context != "" {
		prompt += fmt.Sprintf("背景信息: %s\n\n", req.Context)
	}

	if len(req.Characters) > 0 {
		prompt += fmt.Sprintf("涉及角色: %v\n\n", req.Characters)
	}

	if req.Style != "" {
		prompt += fmt.Sprintf("风格: %s\n", req.Style)
	}

	lengthGuide := map[string]string{
		"short":  "简短（500-800字）",
		"medium": "中等（1000-1500字）",
		"long":   "长篇（2000-3000字）",
	}
	if guide, ok := lengthGuide[req.Length]; ok {
		prompt += fmt.Sprintf("长度要求: %s\n", guide)
	}

	prompt += "\n请以JSON格式返回，包含以下字段：\n"
	prompt += "- title: 故事标题\n"
	prompt += "- content: 完整的故事内容\n"
	prompt += "- summary: 故事摘要（100字以内）\n"
	prompt += "- scenes: 场景列表（每个场景包含title, description, location, timeOfDay）\n"
	prompt += "- suggestedTags: 建议的标签（3-5个）\n"

	return prompt
}

// parseStoryResult 解析故事生成结果
func (s *AIService) parseStoryResult(text string, req *domain.AIStoryGenerationRequest) *domain.AIStoryGenerationResult {
	var result domain.AIStoryGenerationResult

	// 尝试解析 JSON
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		// 如果不是 JSON，使用原始文本
		result.Title = "生成的故事"
		result.Content = text
		result.Summary = s.extractSummary(text)
	}

	return &result
}

// extractSummary 从文本中提取摘要
func (s *AIService) extractSummary(text string) string {
	if len(text) <= 100 {
		return text
	}
	return text[:100] + "..."
}

// ============== 提示词增强 ==============

// EnhancePrompt 增强提示词
func (s *AIService) EnhancePrompt(ctx context.Context, userID string, req *domain.AIPromptEnhanceRequest) (*domain.AITask, error) {
	if s == nil {
		return nil, fmt.Errorf("ai service is not initialized")
	}
	// 创建 AI 任务
	task := &domain.AITask{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      domain.AITaskEnhancePrompt,
		Status:    domain.AITaskStatusPending,
		Provider:  "",
		Progress:  0,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 序列化输入参数
	inputJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}
	task.Input = string(inputJSON)

	// 启动异步处理
	go s.processPromptEnhancement(context.Background(), task, req)

	return task, nil
}

// processPromptEnhancement 处理提示词增强（异步）
func (s *AIService) processPromptEnhancement(ctx context.Context, task *domain.AITask, req *domain.AIPromptEnhanceRequest) {
	task.Status = domain.AITaskStatusProcessing
	task.Progress = 20
	startTime := time.Now().Unix()
	task.StartedAt = &startTime
	task.UpdatedAt = time.Now().Unix()

	s.logger.Info("开始增强提示词",
		zap.String("taskId", task.ID),
		zap.String("originalPrompt", req.OriginalPrompt),
	)

	// 构建增强提示词
	prompt := s.buildEnhancePrompt(req)

	temperature := float32(0.5) // 提示词增强使用较低温度
	maxTokens := int32(500)
	genConfig := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: maxTokens,
	}

	task.Progress = 50

	// 调用 Gemini 生成文本
	text, geminiResp, err := s.geminiClient.GenerateText(ctx, "", prompt, genConfig)
	if err != nil {
		s.logger.Error("提示词增强失败",
			zap.String("taskId", task.ID),
			zap.Error(err),
		)
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = err.Error()
		task.UpdatedAt = time.Now().Unix()
		return
	}

	// 计算 token 使用量
	tokensUsed := 0
	if geminiResp != nil && geminiResp.UsageMetadata != nil {
		tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
	}

	// 解析结果
	result := &domain.AIPromptEnhanceResult{
		EnhancedPrompt: s.extractEnhancedPrompt(text),
		Improvements:   s.extractImprovements(text),
		TokensUsed:     tokensUsed,
	}

	outputJSON, err := json.Marshal(result)
	if err != nil {
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = fmt.Sprintf("marshal output: %v", err)
		task.UpdatedAt = time.Now().Unix()
		return
	}

	task.Status = domain.AITaskStatusCompleted
	task.Progress = 100
	task.Output = string(outputJSON)
	task.TokensUsed = tokensUsed
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime
	task.UpdatedAt = time.Now().Unix()

	s.logger.Info("提示词增强完成",
		zap.String("taskId", task.ID),
		zap.Int("tokensUsed", tokensUsed),
	)
}

// buildEnhancePrompt 构建增强提示词的提示
func (s *AIService) buildEnhancePrompt(req *domain.AIPromptEnhanceRequest) string {
	targetTypeDesc := map[string]string{
		"image":      "图片生成",
		"video":      "视频生成",
		"storyboard": "故事板分支的剧情走向（续写多格漫画前的文字说明，叙事性、可拍成连续分镜，不要写成图像提示词）",
	}

	targetLabel := targetTypeDesc[req.TargetType]
	if targetLabel == "" {
		targetLabel = req.TargetType
	}

	prompt := "作为专业的AI提示词工程师，请帮我优化以下输入：\n\n"
	prompt += fmt.Sprintf("原始内容: %s\n\n", req.OriginalPrompt)
	prompt += fmt.Sprintf("目标用途: %s\n", targetLabel)

	if req.TargetType == "storyboard" {
		prompt += "\n专项要求：润色为一段连贯、具体的剧情走向描述；突出冲突、动机或转折；适合作为多格漫画分镜的文字基础；使用与原文一致的语言；总长度建议不超过 200 个字符（中文按字计）。\n"
	}

	if req.Style != "" {
		prompt += fmt.Sprintf("期望风格: %s\n", req.Style)
	}

	detailLevelDesc := map[string]string{
		"low":    "简洁明了，关注核心要素",
		"medium": "中等细节，平衡描述",
		"high":   "极致细节，包含环境、光影、氛围等",
	}
	if desc, ok := detailLevelDesc[req.DetailLevel]; ok {
		prompt += fmt.Sprintf("细节程度: %s\n", desc)
	}

	prompt += "\n请返回JSON格式：\n"
	prompt += "{\n"
	prompt += "  \"enhancedPrompt\": \"增强后的提示词\",\n"
	prompt += "  \"improvements\": \"改进说明\"\n"
	prompt += "}\n"

	return prompt
}

// extractEnhancedPrompt 提取增强后的提示词
func (s *AIService) extractEnhancedPrompt(text string) string {
	var result struct {
		EnhancedPrompt string `json:"enhancedPrompt"`
	}
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result.EnhancedPrompt
	}
	// 如果解析失败，返回原文
	return text
}

// extractImprovements 提取改进说明
func (s *AIService) extractImprovements(text string) string {
	var result struct {
		Improvements string `json:"improvements"`
	}
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result.Improvements
	}
	return ""
}

// ============== 图片生成 ==============

// GenerateImage 生成图片
func (s *AIService) GenerateImage(ctx context.Context, userID string, req *domain.AIImageGenerationRequest) (*domain.AITask, error) {
	// 创建 AI 任务
	task := &domain.AITask{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      domain.AITaskGenerateImage,
		Status:    domain.AITaskStatusPending,
		Provider:  "",
		Progress:  0,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	inputJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}
	task.Input = string(inputJSON)

	// 启动异步处理
	go s.processImageGeneration(context.Background(), task, req)

	return task, nil
}

// processImageGeneration 处理图片生成（异步）
func (s *AIService) processImageGeneration(ctx context.Context, task *domain.AITask, req *domain.AIImageGenerationRequest) {
	task.Status = domain.AITaskStatusProcessing
	task.Progress = 20
	startTime := time.Now().Unix()
	task.StartedAt = &startTime
	task.UpdatedAt = time.Now().Unix()

	s.logger.Info("开始生成图片",
		zap.String("taskId", task.ID),
		zap.String("prompt", req.Prompt),
	)

	// 构建图片生成请求
	genReq := &genapi.GenerateRequest{
		Operation:   genapi.OperationTextToImage,
		Prompt:      req.Prompt,
		Size:        req.Size,
		Quality:     req.Quality,
		Style:       req.Style,
		OutputCount: req.N,
	}

	if genReq.OutputCount == 0 {
		genReq.OutputCount = 1 // 默认生成1张
	}

	task.Progress = 50

	providerName := strings.TrimSpace(task.Provider)
	if providerName == "" {
		providerName = s.defaultImageProvider
	}
	if s.genAPI != nil {
		providerName = CoalesceRegisteredImageProvider(s.genAPI, providerName)
	}

	if strings.EqualFold(strings.TrimSpace(providerName), "huoshan") {
		PrepareHuoshanGenAPIImageRequest(genReq)
	}

	resp, err := s.genAPI.GenerateImage(ctx, providerName, genReq)
	if err != nil {
		s.logger.Error("图片生成失败",
			zap.String("taskId", task.ID),
			zap.Error(err),
		)
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = err.Error()
		task.UpdatedAt = time.Now().Unix()
		return
	}

	// 检查响应错误
	if resp.Error != "" {
		s.logger.Error("图片生成返回错误",
			zap.String("taskId", task.ID),
			zap.String("error", resp.Error),
		)
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = resp.Error
		task.UpdatedAt = time.Now().Unix()
		return
	}

	// 计算 token 使用量
	tokensUsed := 0
	if resp.Usage != nil {
		tokensUsed = resp.Usage.TotalTokens
	}

	// 上传图片到 OSS 并生成多级别缩略图
	processedURLs := make([]string, 0, len(resp.ImageURLs))
	ossClient := aliyun.GetGlobalClient()

	for i, imageURL := range resp.ImageURLs {
		if ossClient != nil {
			// 生成 OSS object key（包含多级别）
			objectKey := fmt.Sprintf("ai-generated/images/%s/original.jpg", task.ID)

			// 上传到 OSS（会自动生成不同 level 的图片）
			ossURL, err := ossClient.UploadFileFromURL(objectKey, imageURL)
			if err != nil {
				s.logger.Warn("failed to upload image to OSS, using original URL",
					zap.String("taskId", task.ID),
					zap.Int("index", i),
					zap.String("originalURL", imageURL),
					zap.Error(err))
				// 上传失败时使用原始 URL
				processedURLs = append(processedURLs, imageURL)
			} else {
				// 清理 URL：移除查询参数，确保 HTTPS
				ossURL = strings.Split(ossURL, "?")[0]
				ossURL = strings.ReplaceAll(ossURL, "http://", "https://")
				processedURLs = append(processedURLs, ossURL)
				s.logger.Debug("image uploaded to OSS with multi-level support",
					zap.String("taskId", task.ID),
					zap.Int("index", i),
					zap.String("ossURL", ossURL))
			}
		} else {
			s.logger.Warn("OSS client not available, using original image URL",
				zap.String("taskId", task.ID))
			processedURLs = append(processedURLs, imageURL)
		}
	}

	// 构建结果
	result := &domain.AIImageGenerationResult{
		URLs:       processedURLs,
		TokensUsed: tokensUsed,
	}

	outputJSON, err := json.Marshal(result)
	if err != nil {
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = fmt.Sprintf("marshal output: %v", err)
		task.UpdatedAt = time.Now().Unix()
		return
	}

	task.Status = domain.AITaskStatusCompleted
	task.Progress = 100
	task.Output = string(outputJSON)
	task.TokensUsed = tokensUsed
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime
	task.UpdatedAt = time.Now().Unix()

	s.logger.Info("图片生成完成",
		zap.String("taskId", task.ID),
		zap.Int("imagesCount", len(resp.ImageURLs)),
		zap.Int("tokensUsed", tokensUsed),
	)
}

// ============== 视频生成 ==============

// GenerateVideo 生成视频
func (s *AIService) GenerateVideo(ctx context.Context, userID string, req *domain.AIVideoGenerationRequest) (*domain.AITask, error) {
	// 创建 AI 任务
	task := &domain.AITask{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      domain.AITaskGenerateVideo,
		Status:    domain.AITaskStatusPending,
		Provider:  "",
		Progress:  0,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	inputJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}
	task.Input = string(inputJSON)

	// 启动异步处理
	go s.processVideoGeneration(context.Background(), task, req)

	return task, nil
}

// processVideoGeneration 处理视频生成（异步）
func (s *AIService) processVideoGeneration(ctx context.Context, task *domain.AITask, req *domain.AIVideoGenerationRequest) {
	task.Status = domain.AITaskStatusProcessing
	task.Progress = 10
	startTime := time.Now().Unix()
	task.StartedAt = &startTime
	task.UpdatedAt = time.Now().Unix()

	s.logger.Info("开始生成视频",
		zap.String("taskId", task.ID),
		zap.String("prompt", req.Prompt),
		zap.Int("duration", req.Duration),
	)

	// 构建视频生成请求
	genReq := &genapi.GenerateRequest{
		Operation:       genapi.OperationTextToVideo,
		Prompt:          req.Prompt,
		DurationSeconds: req.Duration,
		Resolution:      req.Resolution,
		Style:           req.Style,
	}

	// 设置默认值
	if genReq.DurationSeconds == 0 {
		genReq.DurationSeconds = 6 // 默认6秒
	}
	if genReq.Resolution == "" {
		genReq.Resolution = "1080p"
	}

	task.Progress = 30

	providerName := strings.TrimSpace(task.Provider)
	if providerName == "" {
		providerName = s.defaultVideoProvider
	}
	if s.genAPI != nil {
		providerName = CoalesceRegisteredVideoProvider(s.genAPI, providerName)
	}

	resp, err := s.genAPI.GenerateVideo(ctx, providerName, genReq)
	if err != nil {
		s.logger.Error("视频生成失败",
			zap.String("taskId", task.ID),
			zap.Error(err),
		)
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = err.Error()
		task.UpdatedAt = time.Now().Unix()
		return
	}

	// 检查响应错误
	if resp.Error != "" {
		s.logger.Error("视频生成返回错误",
			zap.String("taskId", task.ID),
			zap.String("error", resp.Error),
		)
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = resp.Error
		task.UpdatedAt = time.Now().Unix()
		return
	}

	task.Progress = 90

	// 计算 token 使用量
	tokensUsed := 0
	if resp.Usage != nil {
		tokensUsed = resp.Usage.TotalTokens
	}

	// 上传视频到 OSS 并生成多级别缩略图
	processedVideoURL := resp.VideoURL
	processedThumbnailURL := resp.ThumbnailURL
	ossClient := aliyun.GetGlobalClient()

	if ossClient != nil {
		// 上传视频文件
		if resp.VideoURL != "" {
			videoObjectKey := fmt.Sprintf("ai-generated/videos/%s/original.mp4", task.ID)
			ossVideoURL, err := ossClient.UploadFileFromURL(videoObjectKey, resp.VideoURL)
			if err != nil {
				s.logger.Warn("failed to upload video to OSS, using original URL",
					zap.String("taskId", task.ID),
					zap.String("originalURL", resp.VideoURL),
					zap.Error(err))
			} else {
				// 清理 URL
				ossVideoURL = strings.Split(ossVideoURL, "?")[0]
				ossVideoURL = strings.ReplaceAll(ossVideoURL, "http://", "https://")
				processedVideoURL = ossVideoURL
				s.logger.Debug("video uploaded to OSS",
					zap.String("taskId", task.ID),
					zap.String("ossURL", ossVideoURL))
			}
		}

		// 上传缩略图
		if resp.ThumbnailURL != "" {
			thumbnailObjectKey := fmt.Sprintf("ai-generated/videos/%s/thumbnail.jpg", task.ID)
			ossThumbnailURL, err := ossClient.UploadFileFromURL(thumbnailObjectKey, resp.ThumbnailURL)
			if err != nil {
				s.logger.Warn("failed to upload thumbnail to OSS, using original URL",
					zap.String("taskId", task.ID),
					zap.String("originalURL", resp.ThumbnailURL),
					zap.Error(err))
			} else {
				// 清理 URL
				ossThumbnailURL = strings.Split(ossThumbnailURL, "?")[0]
				ossThumbnailURL = strings.ReplaceAll(ossThumbnailURL, "http://", "https://")
				processedThumbnailURL = ossThumbnailURL
				s.logger.Debug("thumbnail uploaded to OSS",
					zap.String("taskId", task.ID),
					zap.String("ossURL", ossThumbnailURL))
			}
		}
	} else {
		s.logger.Warn("OSS client not available, using original video URLs",
			zap.String("taskId", task.ID))
	}

	// 构建生成结果
	result := &domain.AIVideoGenerationResult{
		VideoURL:     processedVideoURL,
		ThumbnailURL: processedThumbnailURL,
		Duration:     req.Duration,
		TokensUsed:   tokensUsed,
	}

	outputJSON, err := json.Marshal(result)
	if err != nil {
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = fmt.Sprintf("marshal output: %v", err)
		task.UpdatedAt = time.Now().Unix()
		return
	}

	task.Status = domain.AITaskStatusCompleted
	task.Progress = 100
	task.Output = string(outputJSON)
	task.TokensUsed = result.TokensUsed
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime
	task.UpdatedAt = time.Now().Unix()

	s.logger.Info("视频生成完成",
		zap.String("taskId", task.ID),
		zap.Int("tokensUsed", result.TokensUsed),
		zap.Duration("duration", time.Duration(time.Now().Unix()-startTime)*time.Second),
	)
}

// ============== 任务查询 ==============

// GetTaskStatus 获取任务状态
func (s *AIService) GetTaskStatus(ctx context.Context, taskID string) (*domain.AITask, error) {
	task, err := s.repo.GetAITask(ctx, taskID)
	if err != nil {
		s.logger.Error("获取任务状态失败",
			zap.String("taskId", taskID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get task: %w", err)
	}

	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// GetTaskResult 获取任务结果
func (s *AIService) GetTaskResult(ctx context.Context, taskID string) (*domain.AITask, error) {
	task, err := s.GetTaskStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.Status != domain.AITaskStatusCompleted {
		return nil, fmt.Errorf("task not completed yet, current status: %s", task.Status)
	}

	return task, nil
}

// CancelTask 取消任务
func (s *AIService) CancelTask(ctx context.Context, taskID, userID string) error {
	// 获取任务
	task, err := s.repo.GetAITask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 验证用户权限
	if task.UserID != userID {
		return fmt.Errorf("unauthorized: task belongs to another user")
	}

	// 只能取消等待中或处理中的任务
	if task.Status != domain.AITaskStatusPending && task.Status != domain.AITaskStatusProcessing {
		return fmt.Errorf("cannot cancel task with status: %s", task.Status)
	}

	// 更新任务状态为已取消
	task.Status = domain.AITaskStatusCancelled
	task.UpdatedAt = time.Now().Unix()
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime

	if err := s.repo.UpdateAITask(ctx, task); err != nil {
		s.logger.Error("取消任务失败",
			zap.String("taskId", taskID),
			zap.Error(err),
		)
		return fmt.Errorf("update task: %w", err)
	}

	s.logger.Info("任务已取消",
		zap.String("taskId", taskID),
		zap.String("userId", userID),
	)

	return nil
}

// ptrInt32 返回 int32 指针
func ptrInt32(v int) *int32 {
	i := int32(v)
	return &i
}

// ============== Fragment Generation Helpers ==============

// fragmentTextPayload 碎片文案生成输入（支持纯字符串或 JSON：prompt + 可选参考图 URL）。
type fragmentTextPayload struct {
	Prompt    string   `json:"prompt"`
	ImageURLs []string `json:"imageUrls,omitempty"`
}

func (s *AIService) parseFragmentTextInput(aiTask *domain.AITask) (prompt string, imageURLs []string, err error) {
	if aiTask == nil {
		return "", nil, fmt.Errorf("ai task is nil")
	}
	raw := strings.TrimSpace(aiTask.Input)
	if raw == "" {
		return "", nil, fmt.Errorf("empty input")
	}
	var p fragmentTextPayload
	if json.Unmarshal([]byte(raw), &p) == nil && strings.TrimSpace(p.Prompt) != "" {
		return strings.TrimSpace(p.Prompt), fragmentPrefillHTTPImageURLs(p.ImageURLs, 10), nil
	}
	var single string
	if json.Unmarshal([]byte(raw), &single) == nil && strings.TrimSpace(single) != "" {
		return strings.TrimSpace(single), nil, nil
	}
	return raw, nil, nil
}

// fragmentImagePayload 碎片配图生成输入（prompt + 可选 aspectRatio + 可选多参考图）。
type fragmentImagePayload struct {
	Prompt          string                 `json:"prompt"`
	AspectRatio     string                 `json:"aspectRatio,omitempty"`
	ReferenceImages []string               `json:"referenceImages,omitempty"`
	Seed            int                    `json:"seed,omitempty"`
	Options         map[string]interface{} `json:"options,omitempty"`
	GuidanceScale   float64                `json:"guidanceScale,omitempty"`
}

func parseFragmentImageInput(aiTask *domain.AITask) (prompt string, aspectRatio string, referenceImages []string, seed int, options map[string]interface{}, guidanceScale float64, err error) {
	if aiTask == nil {
		return "", "", nil, 0, nil, 0, fmt.Errorf("ai task is nil")
	}
	raw := strings.TrimSpace(aiTask.Input)
	if raw == "" {
		return "", "", nil, 0, nil, 0, fmt.Errorf("empty input")
	}
	var p fragmentImagePayload
	if json.Unmarshal([]byte(raw), &p) == nil && strings.TrimSpace(p.Prompt) != "" {
		return strings.TrimSpace(p.Prompt), strings.TrimSpace(p.AspectRatio), fragmentPrefillHTTPImageURLs(p.ReferenceImages, 12), p.Seed, p.Options, p.GuidanceScale, nil
	}
	var single string
	if json.Unmarshal([]byte(raw), &single) == nil {
		return strings.TrimSpace(single), "", nil, 0, nil, 0, nil
	}
	return raw, "", nil, 0, nil, 0, nil
}

func parseFragmentMultimodalStoryJSON(raw string) (content string, inferredAspect string, err error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	var out struct {
		Content     string `json:"content"`
		AspectRatio string `json:"aspectRatio"`
	}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return "", "", err
	}
	content = strings.TrimSpace(out.Content)
	if content == "" {
		return "", "", fmt.Errorf("empty content in JSON")
	}
	return content, domain.NormalizeFragmentAspectRatio(out.AspectRatio), nil
}

func (s *AIService) generateFragmentStoryMultimodal(ctx context.Context, prompt string, imageURLs []string) (content string, tokens int, inferredAspect string, err error) {
	hc := s.genAPI.HuoshanInternalClient()
	if hc != nil {
		resp, err := hc.GenerateText(ctx, &huoshanclient.TextGenerationRequest{
			Prompt:       prompt,
			ImageURLs:    imageURLs,
			MaxTokens:    1200,
			Temperature:  0.7,
			JSONResponse: true,
		})
		if err == nil && resp != nil && strings.TrimSpace(resp.Text) != "" {
			c, ar, perr := parseFragmentMultimodalStoryJSON(resp.Text)
			if perr == nil {
				tok := resp.TotalTokens
				if tok == 0 {
					tok = resp.InputTokens + resp.OutputTokens
				}
				return c, tok, ar, nil
			}
			s.logger.Warn("fragment multimodal: JSON parse failed", zap.Error(perr))
		} else if err != nil {
			s.logger.Warn("fragment multimodal: huoshan failed", zap.Error(err))
		}
	}

	if s.geminiClient == nil {
		if hc == nil {
			return "", 0, "", fmt.Errorf("no multimodal provider: configure HUOSHAN_API_KEY or GEMINI_API_KEY")
		}
		return "", 0, "", fmt.Errorf("huoshan multimodal failed and gemini is not configured")
	}

	s.logger.Warn("fragment multimodal: falling back to gemini text-only (no vision)")
	temperature := float32(0.7)
	maxTokens := int32(1200)
	cfg := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: maxTokens,
	}
	text, gemResp, err := s.geminiClient.GenerateText(ctx, "", prompt, cfg)
	if err != nil {
		return "", 0, "", fmt.Errorf("gemini multimodal fallback: %w", err)
	}
	c, ar, perr := parseFragmentMultimodalStoryJSON(text)
	if perr != nil {
		return "", 0, "", fmt.Errorf("gemini returned invalid JSON story: %w", perr)
	}
	tokensUsed := 0
	if gemResp != nil && gemResp.UsageMetadata != nil {
		tokensUsed = int(gemResp.UsageMetadata.TotalTokenCount)
	}
	return c, tokensUsed, ar, nil
}

// generateFragmentExtractionMultimodalRaw 多模态提取：返回完整 JSON 文本（含 visualBible），不裁剪为 content-only。
func (s *AIService) generateFragmentExtractionMultimodalRaw(ctx context.Context, prompt string, imageURLs []string, relatedEntityType, relatedEntityID string) (string, int, error) {
	resp, err := s.GenerateFragmentVisionJSON(ctx, &FragmentVisionJSONRequest{
		Prompt:            prompt,
		ImageURLs:         imageURLs,
		MaxTokens:         4096,
		Temperature:       0.55,
		RelatedEntityID:   relatedEntityID,
		RelatedEntityType: relatedEntityType,
		Step:              "fragment_extraction",
	})
	if err != nil {
		return "", 0, err
	}
	return resp.Raw, resp.TokensUsed, nil
}

// FragmentVisionJSONRequest describes a multimodal JSON helper call used by fragment generation.
type FragmentVisionJSONRequest struct {
	Prompt            string
	ImageURLs         []string
	ProviderHint      string
	MaxTokens         int32
	Temperature       float32
	RelatedEntityID   string
	RelatedEntityType string
	Step              string
}

type FragmentVisionJSONResult struct {
	Raw        string
	TokensUsed int
	DurationMs int64
	Provider   string
	Model      string
}

func (s *AIService) recordFragmentPromptAudit(ctx context.Context, relatedType, relatedID, step, promptKind, provider, model string, temperature float64, maxTokens int, prompt string, imageURLs []string, output string, tokens int, metadata map[string]any) {
	if s == nil || s.repo == nil || strings.TrimSpace(prompt) == "" {
		return
	}
	step = strings.TrimSpace(step)
	if step == "" {
		step = "fragment_ai"
	}
	promptKind = strings.TrimSpace(promptKind)
	if promptKind == "" {
		promptKind = step
	}
	relatedType = strings.TrimSpace(relatedType)
	if relatedType == "" {
		relatedType = "fragment_generation"
	}
	tokenUsage := map[string]any{"totalTokens": tokens}
	md := map[string]any{"source": "fragment_ai_service"}
	for k, v := range metadata {
		md[k] = v
	}
	audit := &domain.AIPromptAuditRecord{
		ID:                    uuid.NewString(),
		RelatedEntityType:     relatedType,
		RelatedEntityID:       strings.TrimSpace(relatedID),
		Step:                  step,
		PromptKind:            promptKind,
		PromptTemplateVersion: "fragment_generation_v1",
		FullPromptHash:        stableSHA256(prompt),
		Provider:              strings.TrimSpace(provider),
		Model:                 strings.TrimSpace(model),
		Temperature:           temperature,
		MaxTokens:             maxTokens,
		UserPrompt:            prompt,
		FinalPrompt:           prompt,
		ReferenceImageURLs:    fragmentPrefillHTTPImageURLs(imageURLs, 12),
		Output:                output,
		TokenUsageJSON:        mustJSON(tokenUsage, "{}"),
		MetadataJSON:          mustJSON(md, "{}"),
		CreatedAt:             time.Now().Unix(),
	}
	if err := s.repo.CreateAIPromptAuditRecord(ctx, audit); err != nil {
		s.logger.Warn("failed to create fragment prompt audit",
			zap.String("relatedEntityType", relatedType),
			zap.String("relatedEntityID", relatedID),
			zap.String("step", step),
			zap.Error(err))
	}
}

// GenerateFragmentVisionJSON runs a JSON-mode multimodal analysis/audit with Huoshan or Gemini.
func (s *AIService) GenerateFragmentVisionJSON(ctx context.Context, req *FragmentVisionJSONRequest) (*FragmentVisionJSONResult, error) {
	rel, admErr := s.acquireTextAdmission(ctx)
	if admErr != nil {
		return nil, admErr
	}
	if rel != nil {
		defer rel()
	}
	if req == nil || strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("empty vision prompt")
	}
	imageURLs := fragmentPrefillHTTPImageURLs(req.ImageURLs, 12)
	if len(imageURLs) == 0 {
		return nil, fmt.Errorf("vision JSON requires at least one image URL")
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	temp := req.Temperature
	if temp == 0 {
		temp = 0.35
	}
	providers := fragmentVisionProviderOrder(req.ProviderHint)
	var lastErr error
	for _, p := range providers {
		start := time.Now()
		switch p {
		case "huoshan":
			raw, tokens, model, err := s.generateFragmentVisionJSONHuoshan(ctx, req.Prompt, imageURLs, maxTokens, temp)
			if err == nil && strings.TrimSpace(raw) != "" {
				s.recordFragmentPromptAudit(ctx, req.RelatedEntityType, req.RelatedEntityID, req.Step, "vision_json", "huoshan", model, float64(temp), int(maxTokens), req.Prompt, imageURLs, strings.TrimSpace(raw), tokens, nil)
				return &FragmentVisionJSONResult{Raw: strings.TrimSpace(raw), TokensUsed: tokens, DurationMs: time.Since(start).Milliseconds(), Provider: "huoshan", Model: model}, nil
			}
			if err != nil {
				lastErr = err
				s.logger.Warn("fragment vision JSON: huoshan failed", zap.String("step", req.Step), zap.Error(err))
			}
		case "gemini":
			raw, tokens, model, err := s.generateFragmentVisionJSONGemini(ctx, req.Prompt, imageURLs, maxTokens, temp)
			if err == nil && strings.TrimSpace(raw) != "" {
				s.recordFragmentPromptAudit(ctx, req.RelatedEntityType, req.RelatedEntityID, req.Step, "vision_json", "gemini", model, float64(temp), int(maxTokens), req.Prompt, imageURLs, strings.TrimSpace(raw), tokens, nil)
				return &FragmentVisionJSONResult{Raw: strings.TrimSpace(raw), TokensUsed: tokens, DurationMs: time.Since(start).Milliseconds(), Provider: "gemini", Model: model}, nil
			}
			if err != nil {
				lastErr = err
				s.logger.Warn("fragment vision JSON: gemini failed", zap.String("step", req.Step), zap.Error(err))
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no multimodal provider available")
}

func fragmentVisionProviderOrder(hint string) []string {
	switch strings.ToLower(strings.TrimSpace(hint)) {
	case "gemini":
		return []string{"gemini", "huoshan"}
	case "huoshan":
		return []string{"huoshan", "gemini"}
	default:
		return []string{"huoshan", "gemini"}
	}
}

func (s *AIService) generateFragmentVisionJSONHuoshan(ctx context.Context, prompt string, imageURLs []string, maxTokens int32, temp float32) (string, int, string, error) {
	if s.genAPI == nil || s.genAPI.HuoshanInternalClient() == nil {
		return "", 0, "", fmt.Errorf("huoshan client not available")
	}
	resp, err := s.genAPI.HuoshanInternalClient().GenerateText(ctx, &huoshanclient.TextGenerationRequest{
		Prompt:       prompt,
		ImageURLs:    imageURLs,
		MaxTokens:    int(maxTokens),
		Temperature:  temp,
		JSONResponse: true,
	})
	if err != nil {
		return "", 0, "", err
	}
	if resp == nil || strings.TrimSpace(resp.Text) == "" {
		return "", 0, "", fmt.Errorf("huoshan returned empty vision response")
	}
	tok := resp.TotalTokens
	if tok == 0 {
		tok = resp.InputTokens + resp.OutputTokens
	}
	model := strings.TrimSpace(resp.Model)
	if model == "" {
		model = "huoshan-ark"
	}
	return strings.TrimSpace(resp.Text), tok, model, nil
}

func (s *AIService) generateFragmentVisionJSONGemini(ctx context.Context, prompt string, imageURLs []string, maxTokens int32, temp float32) (string, int, string, error) {
	if s.geminiClient == nil {
		return "", 0, "", fmt.Errorf("gemini client not configured")
	}
	parts := make([]*genai.Part, 0, len(imageURLs)+1)
	parts = append(parts, genai.NewPartFromText(prompt))
	for _, u := range imageURLs {
		data, mime, err := downloadImageFromURL(ctx, u)
		if err != nil {
			return "", 0, "", fmt.Errorf("download image for gemini vision: %w", err)
		}
		part, err := encodeImagePart(data, mime)
		if err != nil {
			return "", 0, "", fmt.Errorf("encode image for gemini vision: %w", err)
		}
		parts = append(parts, part)
	}
	contents := []*genai.Content{{Role: genai.RoleUser, Parts: parts}}
	cfg := &genai.GenerateContentConfig{
		Temperature:      &temp,
		MaxOutputTokens:  maxTokens,
		ResponseMIMEType: "application/json",
	}
	const model = "gemini-2.5-flash"
	resp, err := s.geminiClient.SDK().Models.GenerateContent(ctx, model, contents, cfg)
	if err != nil {
		return "", 0, "", err
	}
	text := strings.TrimSpace(resp.Text())
	if text == "" {
		return "", 0, "", fmt.Errorf("gemini returned empty vision response")
	}
	tokens := 0
	if resp != nil && resp.UsageMetadata != nil {
		tokens = int(resp.UsageMetadata.TotalTokenCount)
	}
	return text, tokens, model, nil
}

func (s *AIService) generateFragmentExtractionTextHuoshan(ctx context.Context, prompt string) (string, int, error) {
	if s.genAPI == nil {
		return "", 0, fmt.Errorf("genAPI not available")
	}
	hc := s.genAPI.HuoshanInternalClient()
	if hc == nil {
		return "", 0, fmt.Errorf("huoshan client not available")
	}
	resp, err := hc.GenerateText(ctx, &huoshanclient.TextGenerationRequest{
		Prompt:       prompt,
		MaxTokens:    4096,
		Temperature:  0.55,
		JSONResponse: true,
	})
	if err != nil {
		return "", 0, fmt.Errorf("huoshan extraction JSON text failed: %w", err)
	}
	if resp == nil {
		return "", 0, fmt.Errorf("huoshan returned empty response")
	}
	tokens := resp.TotalTokens
	if tokens == 0 {
		tokens = resp.InputTokens + resp.OutputTokens
	}
	return strings.TrimSpace(resp.Text), tokens, nil
}

// GenerateFragmentExtractionJSON Step1 专用：保留完整结构化 JSON（含 visualBible），供普通碎片链路解析。
func (s *AIService) GenerateFragmentExtractionJSON(ctx context.Context, aiTask *domain.AITask) (raw string, tokens int, err error) {
	rel, admErr := s.acquireTextAdmission(ctx)
	if admErr != nil {
		return "", 0, admErr
	}
	if rel != nil {
		defer rel()
	}

	prompt, imageURLs, err := s.parseFragmentTextInput(aiTask)
	if err != nil {
		return "", 0, err
	}
	if len(imageURLs) > 0 {
		return s.generateFragmentExtractionMultimodalRaw(ctx, prompt, imageURLs, aiTask.RelatedEntityType, aiTask.RelatedEntityID)
	}
	if s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil {
		if t, tok, err := s.generateFragmentExtractionTextHuoshan(ctx, prompt); err == nil && strings.TrimSpace(t) != "" {
			s.recordFragmentPromptAudit(ctx, aiTask.RelatedEntityType, aiTask.RelatedEntityID, "fragment_extraction", "text_json", "huoshan", "", 0.55, 4096, prompt, nil, strings.TrimSpace(t), tok, nil)
			return t, tok, nil
		} else if err != nil {
			s.logger.Warn("fragment extraction JSON: huoshan failed", zap.Error(err))
		}
	}
	if s.geminiClient != nil {
		temp := float32(0.55)
		maxTok := int32(4096)
		cfg := &genai.GenerateContentConfig{
			Temperature:     &temp,
			MaxOutputTokens: maxTok,
		}
		text, gemResp, err := s.geminiClient.GenerateText(ctx, "", prompt, cfg)
		if err != nil {
			return "", 0, fmt.Errorf("gemini extraction JSON: %w", err)
		}
		tokensUsed := 0
		if gemResp != nil && gemResp.UsageMetadata != nil {
			tokensUsed = int(gemResp.UsageMetadata.TotalTokenCount)
		}
		s.recordFragmentPromptAudit(ctx, aiTask.RelatedEntityType, aiTask.RelatedEntityID, "fragment_extraction", "text_json", "gemini", "", 0.55, 4096, prompt, nil, strings.TrimSpace(text), tokensUsed, nil)
		return strings.TrimSpace(text), tokensUsed, nil
	}
	return "", 0, fmt.Errorf("no text generation provider available (configure HUOSHAN_API_KEY or GEMINI_API_KEY)")
}

// GenerateFragmentAuxJSON 无参考图的 JSON 模式文本（一致性检查等辅助步骤）。
func (s *AIService) GenerateFragmentAuxJSON(ctx context.Context, prompt string) (raw string, tokens int, err error) {
	rel, admErr := s.acquireTextAdmission(ctx)
	if admErr != nil {
		return "", 0, admErr
	}
	if rel != nil {
		defer rel()
	}

	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", 0, fmt.Errorf("empty prompt")
	}
	if s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil {
		hc := s.genAPI.HuoshanInternalClient()
		resp, err := hc.GenerateText(ctx, &huoshanclient.TextGenerationRequest{
			Prompt:       prompt,
			MaxTokens:    2048,
			Temperature:  0.35,
			JSONResponse: true,
		})
		if err == nil && resp != nil && strings.TrimSpace(resp.Text) != "" {
			tok := resp.TotalTokens
			if tok == 0 {
				tok = resp.InputTokens + resp.OutputTokens
			}
			return strings.TrimSpace(resp.Text), tok, nil
		}
		if err != nil {
			s.logger.Warn("fragment aux JSON: huoshan failed", zap.Error(err))
		}
	}
	if s.geminiClient == nil {
		return "", 0, fmt.Errorf("no provider for fragment aux JSON")
	}
	temp := float32(0.35)
	maxTok := int32(2048)
	cfg := &genai.GenerateContentConfig{
		Temperature:     &temp,
		MaxOutputTokens: maxTok,
	}
	text, gemResp, err := s.geminiClient.GenerateText(ctx, "", prompt, cfg)
	if err != nil {
		return "", 0, err
	}
	tokensUsed := 0
	if gemResp != nil && gemResp.UsageMetadata != nil {
		tokensUsed = int(gemResp.UsageMetadata.TotalTokenCount)
	}
	return strings.TrimSpace(text), tokensUsed, nil
}

// GenerateTextForFragment 生成文本内容（为 FragmentGenerationService 提供简化接口）。
// 有参考图时使用多模态 + JSON（content + aspectRatio）；否则纯文本。
// inferredAspect 仅在有参考图且模型成功返回 JSON 时可能非空；需由调用方与用户指定比例合并。
func (s *AIService) GenerateTextForFragment(ctx context.Context, aiTask *domain.AITask) (string, int, string, error) {
	rel, admErr := s.acquireTextAdmission(ctx)
	if admErr != nil {
		return "", 0, "", admErr
	}
	if rel != nil {
		defer rel()
	}

	prompt, imageURLs, err := s.parseFragmentTextInput(aiTask)
	if err != nil {
		return "", 0, "", err
	}
	auditStep := "fragment_text"
	var inputMeta map[string]any
	if json.Unmarshal([]byte(aiTask.Input), &inputMeta) == nil {
		if step, ok := inputMeta["step"].(string); ok && strings.TrimSpace(step) != "" {
			auditStep = strings.TrimSpace(step)
		}
	}

	if len(imageURLs) > 0 {
		content, tokens, inferredAspect, err := s.generateFragmentStoryMultimodal(ctx, prompt, imageURLs)
		if err == nil {
			s.recordFragmentPromptAudit(ctx, aiTask.RelatedEntityType, aiTask.RelatedEntityID, auditStep, "text_multimodal_json", "auto", "", 0.55, 1200, prompt, imageURLs, content, tokens, map[string]any{"inferredAspectRatio": inferredAspect})
		}
		return content, tokens, inferredAspect, err
	}

	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil

	if huoshanOK {
		text, n, err := s.generateFragmentTextHuoshan(ctx, prompt)
		if err == nil {
			s.recordFragmentPromptAudit(ctx, aiTask.RelatedEntityType, aiTask.RelatedEntityID, auditStep, "text", "huoshan", "", 0.7, 1000, prompt, nil, text, n, nil)
			return text, n, "", nil
		}
		if s.geminiClient != nil {
			s.logger.Warn("fragment text: huoshan failed, falling back to gemini",
				zap.String("userId", aiTask.UserID), zap.Error(err))
			t, tok, err2 := s.generateFragmentTextGemini(ctx, prompt)
			if err2 == nil {
				s.recordFragmentPromptAudit(ctx, aiTask.RelatedEntityType, aiTask.RelatedEntityID, auditStep, "text", "gemini", "", 0.7, 1000, prompt, nil, t, tok, map[string]any{"fallbackFrom": "huoshan"})
			}
			return t, tok, "", err2
		}
		return "", 0, "", err
	}

	if s.geminiClient != nil {
		t, tok, err := s.generateFragmentTextGemini(ctx, prompt)
		if err == nil {
			s.recordFragmentPromptAudit(ctx, aiTask.RelatedEntityType, aiTask.RelatedEntityID, auditStep, "text", "gemini", "", 0.7, 1000, prompt, nil, t, tok, nil)
		}
		return t, tok, "", err
	}

	return "", 0, "", fmt.Errorf("no text generation provider available (configure HUOSHAN_API_KEY or GEMINI_API_KEY)")
}

func (s *AIService) fragmentTextUserRegion(ctx context.Context, userID string) string {
	if s.repo == nil || strings.TrimSpace(userID) == "" {
		return ""
	}
	st, err := s.repo.UserSettings(ctx, userID)
	if err != nil || st == nil {
		return ""
	}
	return st.Region
}

func (s *AIService) generateFragmentTextGemini(ctx context.Context, prompt string) (string, int, error) {
	if s.geminiClient == nil {
		return "", 0, fmt.Errorf("gemini client not available")
	}
	temperature := float32(0.7)
	maxTokens := int32(1000)
	genConfig := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: maxTokens,
	}
	text, geminiResp, err := s.geminiClient.GenerateText(ctx, "", prompt, genConfig)
	if err != nil {
		return "", 0, fmt.Errorf("gemini generate text failed: %w", err)
	}
	tokensUsed := 0
	if geminiResp != nil && geminiResp.UsageMetadata != nil {
		tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
	}
	return text, tokensUsed, nil
}

func (s *AIService) generateFragmentTextHuoshan(ctx context.Context, prompt string) (string, int, error) {
	if s.genAPI == nil {
		return "", 0, fmt.Errorf("genAPI not available")
	}
	hc := s.genAPI.HuoshanInternalClient()
	if hc == nil {
		return "", 0, fmt.Errorf("huoshan client not available")
	}
	resp, err := hc.GenerateText(ctx, &huoshanclient.TextGenerationRequest{
		Prompt:      prompt,
		MaxTokens:   1000,
		Temperature: 0.7,
	})
	if err != nil {
		return "", 0, fmt.Errorf("huoshan generate text failed: %w", err)
	}
	if resp == nil {
		return "", 0, fmt.Errorf("huoshan returned empty response")
	}
	tokens := resp.TotalTokens
	if tokens == 0 {
		tokens = resp.InputTokens + resp.OutputTokens
	}
	return strings.TrimSpace(resp.Text), tokens, nil
}

// fragmentPrefillHTTPImageURLs 仅保留公网 http(s) URL，供火山多模态对话使用（最多 maxN 张）。
func fragmentPrefillHTTPImageURLs(urls []string, maxN int) []string {
	if maxN <= 0 {
		return nil
	}
	var out []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		low := strings.ToLower(u)
		if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
			continue
		}
		out = append(out, u)
		if len(out) >= maxN {
			break
		}
	}
	return out
}

// GenerateTextForFragmentStoryPrefill 碎片「生成故事」预填：优先火山方舟（JSON 模式 + 可选参考图），失败回退 Gemini。
func (s *AIService) GenerateTextForFragmentStoryPrefill(ctx context.Context, prompt string, referenceImageURLs []string) (string, error) {
	rel, admErr := s.acquireTextAdmission(ctx)
	if admErr != nil {
		return "", admErr
	}
	if rel != nil {
		defer rel()
	}

	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	imgURLs := fragmentPrefillHTTPImageURLs(referenceImageURLs, 4)

	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	if huoshanOK {
		hc := s.genAPI.HuoshanInternalClient()
		req := &huoshanclient.TextGenerationRequest{
			Prompt:       prompt,
			MaxTokens:    2048,
			Temperature:  0.35,
			JSONResponse: true,
			ImageURLs:    imgURLs,
		}
		resp, err := hc.GenerateText(ctx, req)
		if err == nil && resp != nil {
			text := strings.TrimSpace(resp.Text)
			if text != "" {
				return text, nil
			}
		}
		if err != nil {
			s.logger.Warn("fragment story prefill: huoshan failed, falling back to gemini",
				zap.Error(err))
		} else {
			s.logger.Warn("fragment story prefill: huoshan returned empty text, falling back to gemini")
		}
	}

	if s.geminiClient == nil {
		if !huoshanOK {
			return "", fmt.Errorf("no text generation provider available (configure HUOSHAN_API_KEY or GEMINI_API_KEY)")
		}
		return "", fmt.Errorf("huoshan text generation failed and gemini is not configured")
	}

	temp := float32(0.35)
	maxTok := int32(2048)
	cfg := &genai.GenerateContentConfig{
		Temperature:     &temp,
		MaxOutputTokens: maxTok,
	}
	raw, _, err := s.geminiClient.GenerateText(ctx, "", prompt, cfg)
	if err != nil {
		return "", fmt.Errorf("gemini generate text failed: %w", err)
	}
	return raw, nil
}

// ResolveFragmentImageProvider 与 GenerateImageForFragment 相同的配图路由规则（Huoshan / Gemini 等）。
func (s *AIService) ResolveFragmentImageProvider(ctx context.Context, userID, taskProviderOverride string) string {
	region := s.fragmentTextUserRegion(ctx, userID)
	_, preferred := ResolvePanelGenerationAIProviders(region, s.defaultImageProvider, s.aiGen)
	p := strings.TrimSpace(taskProviderOverride)
	if p != "" {
		return CoalesceRegisteredImageProvider(s.genAPI, p)
	}
	return CoalesceRegisteredImageProvider(s.genAPI, preferred)
}

// GenerateBatchImagesForFragment 叙事碎片 Huoshan 组图：一次请求多场景（仅当 provider 为 huoshan 时由调用方保证）。
func (s *AIService) GenerateBatchImagesForFragment(ctx context.Context, userID, relatedEntityID, relatedEntityType, prompt, aspectRatio string, refImgs []string, sceneCount int, options map[string]interface{}, seed int, guidanceScale float64, metadata map[string]interface{}) ([]string, int, error) {
	if s.aiGen == nil {
		return nil, 0, fmt.Errorf("AI generation service not configured")
	}
	if sceneCount < 1 {
		return nil, 0, fmt.Errorf("sceneCount must be >= 1")
	}
	ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	provider := s.ResolveFragmentImageProvider(ctx, userID, "")
	if !strings.EqualFold(provider, "huoshan") {
		return nil, 0, fmt.Errorf("batch narrative images requires huoshan provider, got %s", provider)
	}
	if s.genAPI != nil && s.genAPI.GetImageProvider(provider) == nil {
		return nil, 0, fmt.Errorf("image provider %q is not registered", provider)
	}
	md := map[string]interface{}{
		"source":       "fragment_generation_batch_huoshan",
		"aspectRatio":  ar,
		"scene_count":  sceneCount,
		"batch_huoshan": true,
	}
	for k, v := range metadata {
		md[k] = v
	}
	imgReq := &GenerateImageRequest{
		UserID:            strings.TrimSpace(userID),
		Prompt:            prompt,
		NegativePrompt:    fragmentImageNegativePrompt(),
		Provider:          provider,
		Quality:           "standard",
		OutputCount:       sceneCount,
		ReferenceImages:   refImgs,
		Seed:              seed,
		Options:           options,
		GuidanceScale:     guidanceScale,
		RelatedEntityID:   strings.TrimSpace(relatedEntityID),
		RelatedEntityType: strings.TrimSpace(relatedEntityType),
		Metadata:          md,
	}
	imgReq.Size = domain.FragmentImagePixelSizeForAspectRatio(ar)

	imgOut, err := s.aiGen.GenerateImage(ctx, imgReq)
	if err != nil {
		return nil, 0, err
	}
	got := 0
	if imgOut != nil {
		got = len(imgOut.ImageURLs)
	}
	expected := imgReq.OutputCount
	if imgOut == nil || got < expected {
		tok := 0
		if imgOut != nil {
			tok = imgOut.TokensUsed
		}
		return nil, tok, fmt.Errorf("batch images insufficient: got %d need %d", got, expected)
	}
	return imgOut.ImageURLs[:expected], imgOut.TokensUsed, nil
}

// GenerateImageForFragment 生成图片（为 FragmentGenerationService 提供简化接口）。
// 走 AIGenerationService.GenerateImage：配额、扣费记录与用户归因一致；国内默认火山配图，海外且 Gemini 已注册时用 Gemini。
// aiTask.Input 可为 JSON：{"prompt":"...","aspectRatio":"16:9"}，aspectRatio 缺省为 16:9。
func (s *AIService) GenerateImageForFragment(ctx context.Context, aiTask *domain.AITask) (string, int, error) {
	if aiTask == nil {
		return "", 0, fmt.Errorf("ai task is nil")
	}
	if s.aiGen == nil {
		return "", 0, fmt.Errorf("AI generation service not configured")
	}

	prompt, aspectIn, refImgs, seed, options, guidanceScale, err := parseFragmentImageInput(aiTask)
	if err != nil {
		return "", 0, err
	}
	ar := domain.NormalizeFragmentAspectRatio(aspectIn)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}

	region := s.fragmentTextUserRegion(ctx, aiTask.UserID)
	_, preferred := ResolvePanelGenerationAIProviders(region, s.defaultImageProvider, s.aiGen)
	provider := strings.TrimSpace(aiTask.Provider)
	if provider != "" {
		provider = CoalesceRegisteredImageProvider(s.genAPI, provider)
	} else {
		provider = CoalesceRegisteredImageProvider(s.genAPI, preferred)
	}
	if s.genAPI != nil && s.genAPI.GetImageProvider(provider) == nil {
		return "", 0, fmt.Errorf("image provider %q is not registered", provider)
	}

	relatedID := strings.TrimSpace(aiTask.RelatedEntityID)
	if relatedID == "" {
		relatedID = aiTask.ID
	}
	relatedType := strings.TrimSpace(aiTask.RelatedEntityType)
	if relatedType == "" {
		relatedType = "fragment_generation"
	}

	imgReq := &GenerateImageRequest{
		UserID:            strings.TrimSpace(aiTask.UserID),
		Prompt:            prompt,
		NegativePrompt:    fragmentImageNegativePrompt(),
		Provider:          provider,
		Quality:           "standard",
		OutputCount:       1,
		RelatedEntityID:   relatedID,
		RelatedEntityType: relatedType,
		Seed:              seed,
		Options:           options,
		GuidanceScale:     guidanceScale,
		Metadata: map[string]interface{}{
			"source":      "fragment_generation_image",
			"aspectRatio": ar,
			"seed":        seed,
		},
	}
	switch provider {
	case "huoshan":
		imgReq.Size = domain.FragmentImagePixelSizeForAspectRatio(ar)
	default:
		imgReq.AspectRatio = ar
	}
	if len(refImgs) > 0 {
		imgReq.ReferenceImages = refImgs
	}

	imgOut, err := s.aiGen.GenerateImage(ctx, imgReq)
	if err != nil {
		return "", 0, err
	}
	if imgOut == nil || len(imgOut.ImageURLs) == 0 {
		tok := 0
		if imgOut != nil {
			tok = imgOut.TokensUsed
		}
		return "", tok, fmt.Errorf("no images generated")
	}
	return imgOut.ImageURLs[0], imgOut.TokensUsed, nil
}
