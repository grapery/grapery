package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"github.com/grapestree/fgrapery/grapery/internal/queue"
	"google.golang.org/genai"
)

// Worker 任务处理器
type Worker struct {
	queue        *queue.TaskQueue
	repo         domain.Repository
	genAPI       *genapi.GenAPI
	geminiClient *gemini.Client
	logger       *zap.Logger
	stopChan     chan struct{}
}

// NewWorker 创建Worker
func NewWorker(
	queue *queue.TaskQueue,
	repo domain.Repository,
	genAPI *genapi.GenAPI,
	geminiClient *gemini.Client,
	logger *zap.Logger,
) *Worker {
	return &Worker{
		queue:        queue,
		repo:         repo,
		genAPI:       genAPI,
		geminiClient: geminiClient,
		logger:       logger,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动Worker
func (w *Worker) Start(ctx context.Context, workerType string, concurrency int) {
	w.logger.Info("starting workers",
		zap.String("type", workerType),
		zap.Int("concurrency", concurrency),
	)

	for i := 0; i < concurrency; i++ {
		go w.runWorker(ctx, workerType, i)
	}
}

// Stop 停止Worker
func (w *Worker) Stop() {
	close(w.stopChan)
}

// runWorker 运行单个Worker
func (w *Worker) runWorker(ctx context.Context, workerType string, workerID int) {
	w.logger.Info("worker started",
		zap.String("type", workerType),
		zap.Int("workerId", workerID),
	)

	for {
		select {
		case <-w.stopChan:
			w.logger.Info("worker stopped",
				zap.String("type", workerType),
				zap.Int("workerId", workerID),
			)
			return
		case <-ctx.Done():
			w.logger.Info("worker context cancelled",
				zap.String("type", workerType),
				zap.Int("workerId", workerID),
			)
			return
		default:
			w.processTask(ctx, workerType, workerID)
		}
	}
}

// processTask 处理任务
func (w *Worker) processTask(ctx context.Context, workerType string, workerID int) {
	var message *queue.TaskMessage
	var err error

	// 从队列中取出任务（阻塞5秒）
	switch workerType {
	case "ai":
		message, err = w.queue.DequeueAITask(ctx, 5*time.Second)
	case "render":
		message, err = w.queue.DequeueRenderTask(ctx, 5*time.Second)
	default:
		w.logger.Error("unknown worker type", zap.String("type", workerType))
		time.Sleep(5 * time.Second)
		return
	}

	if err != nil {
		w.logger.Error("failed to dequeue task",
			zap.String("type", workerType),
			zap.Error(err),
		)
		return
	}

	// 没有任务
	if message == nil {
		return
	}

	w.logger.Info("processing task",
		zap.String("type", workerType),
		zap.Int("workerId", workerID),
		zap.String("taskId", message.TaskID),
		zap.Int("retry", message.Retry),
	)

	// 处理任务
	var processErr error
	switch workerType {
	case "ai":
		processErr = w.processAITask(ctx, message)
	case "render":
		processErr = w.processRenderTask(ctx, message)
	}

	// 标记任务完成或失败
	if processErr != nil {
		w.logger.Error("task failed",
			zap.String("taskId", message.TaskID),
			zap.Error(processErr),
		)

		// 标记失败（会自动重试）
		switch workerType {
		case "ai":
			w.queue.FailAITask(ctx, message.TaskID, message, processErr)
		case "render":
			w.queue.FailRenderTask(ctx, message.TaskID, message, processErr)
		}
	} else {
		w.logger.Info("task completed successfully",
			zap.String("taskId", message.TaskID),
		)

		// 从处理中集合移除
		switch workerType {
		case "ai":
			w.queue.CompleteAITask(ctx, message.TaskID)
		case "render":
			w.queue.CompleteRenderTask(ctx, message.TaskID)
		}
	}
}

// ========== AI任务处理 ==========

// processAITask 处理AI任务
func (w *Worker) processAITask(ctx context.Context, message *queue.TaskMessage) error {
	// 从数据库获取任务详情
	task, err := w.repo.GetAITask(ctx, message.TaskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	// 更新任务状态为处理中
	startTime := time.Now().Unix()
	task.Status = domain.AITaskStatusProcessing
	task.StartedAt = &startTime
	task.Progress = 10
	if err := w.repo.UpdateAITask(ctx, task); err != nil {
		w.logger.Error("failed to update task status", zap.Error(err))
	}

	// 根据任务类型调用相应的处理方法
	switch task.Type {
	case domain.AITaskGenerateStory:
		return w.processStoryGeneration(ctx, task)
	case domain.AITaskEnhancePrompt:
		return w.processPromptEnhancement(ctx, task)
	case domain.AITaskGenerateImage:
		return w.processImageGeneration(ctx, task)
	case domain.AITaskGenerateVideo:
		return w.processVideoGeneration(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}

// processStoryGeneration 处理故事生成
func (w *Worker) processStoryGeneration(ctx context.Context, task *domain.AITask) error {
	w.logger.Info("processing story generation task",
		zap.String("taskId", task.ID))

	// 解析输入
	var req domain.AIStoryGenerationRequest
	if err := json.Unmarshal([]byte(task.Input), &req); err != nil {
		w.logger.Error("failed to unmarshal input",
			zap.String("taskId", task.ID),
			zap.Error(err))
		return fmt.Errorf("unmarshal input: %w", err)
	}

	// 构建提示词
	prompt := buildStoryPrompt(&req)

	// 配置生成参数
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}
	tempFloat32 := float32(temperature)
	maxTokens := int32(2000)

	genConfig := &genai.GenerateContentConfig{
		Temperature:     &tempFloat32,
		MaxOutputTokens: maxTokens,
	}

	// 更新进度
	if err := w.repo.UpdateAITaskProgress(ctx, task.ID, 30); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	w.logger.Info("calling Gemini for story generation",
		zap.String("taskId", task.ID),
		zap.Float64("temperature", temperature),
		zap.Int32("maxTokens", maxTokens))

	// 调用 Gemini 生成文本
	text, geminiResp, err := w.geminiClient.GenerateText(ctx, "", prompt, genConfig)
	if err != nil {
		w.logger.Error("story generation failed",
			zap.String("taskId", task.ID),
			zap.Error(err))
		// 更新失败状态
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = err.Error()
		w.repo.UpdateAITask(ctx, task)
		return fmt.Errorf("ai generate: %w", err)
	}

	if err := w.repo.UpdateAITaskProgress(ctx, task.ID, 80); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	// 计算 token 使用量
	tokensUsed := 0
	if geminiResp != nil && geminiResp.UsageMetadata != nil {
		tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
	}

	w.logger.Info("story generation completed",
		zap.String("taskId", task.ID),
		zap.Int("tokensUsed", tokensUsed),
		zap.Int("textLength", len(text)))

	// 解析结果
	result := parseStoryResult(text, &req)
	result.TokensUsed = tokensUsed

	// 保存结果
	outputJSON, _ := json.Marshal(result)
	task.Output = string(outputJSON)
	task.TokensUsed = tokensUsed
	task.Status = domain.AITaskStatusCompleted
	task.Progress = 100
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime

	return w.repo.UpdateAITask(ctx, task)
}

// processPromptEnhancement 处理提示词增强
func (w *Worker) processPromptEnhancement(ctx context.Context, task *domain.AITask) error {
	w.logger.Info("processing prompt enhancement task",
		zap.String("taskId", task.ID))

	var req domain.AIPromptEnhanceRequest
	if err := json.Unmarshal([]byte(task.Input), &req); err != nil {
		w.logger.Error("failed to unmarshal input",
			zap.String("taskId", task.ID),
			zap.Error(err))
		return fmt.Errorf("unmarshal input: %w", err)
	}

	prompt := buildEnhancePrompt(&req)

	temperature := float32(0.5)
	maxTokens := int32(500)
	genConfig := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: maxTokens,
	}

	if err := w.repo.UpdateAITaskProgress(ctx, task.ID, 30); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	w.logger.Info("calling Gemini for prompt enhancement",
		zap.String("taskId", task.ID),
		zap.String("originalPrompt", req.OriginalPrompt))

	// 调用 Gemini 生成文本
	text, geminiResp, err := w.geminiClient.GenerateText(ctx, "", prompt, genConfig)
	if err != nil {
		w.logger.Error("prompt enhancement failed",
			zap.String("taskId", task.ID),
			zap.Error(err))
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = err.Error()
		w.repo.UpdateAITask(ctx, task)
		return fmt.Errorf("ai generate: %w", err)
	}

	// 计算 token 使用量
	tokensUsed := 0
	if geminiResp != nil && geminiResp.UsageMetadata != nil {
		tokensUsed = int(geminiResp.UsageMetadata.TotalTokenCount)
	}

	w.logger.Info("prompt enhancement completed",
		zap.String("taskId", task.ID),
		zap.Int("tokensUsed", tokensUsed))

	result := &domain.AIPromptEnhanceResult{
		EnhancedPrompt: extractEnhancedPrompt(text),
		Improvements:   extractImprovements(text),
		TokensUsed:     tokensUsed,
	}

	outputJSON, _ := json.Marshal(result)
	task.Output = string(outputJSON)
	task.TokensUsed = tokensUsed
	task.Status = domain.AITaskStatusCompleted
	task.Progress = 100
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime

	return w.repo.UpdateAITask(ctx, task)
}

// processImageGeneration 处理图片生成
func (w *Worker) processImageGeneration(ctx context.Context, task *domain.AITask) error {
	w.logger.Info("processing image generation task",
		zap.String("taskId", task.ID),
		zap.String("provider", task.Provider))

	var req domain.AIImageGenerationRequest
	if err := json.Unmarshal([]byte(task.Input), &req); err != nil {
		w.logger.Error("failed to unmarshal input",
			zap.String("taskId", task.ID),
			zap.Error(err))
		return fmt.Errorf("unmarshal input: %w", err)
	}

	// 构建图片生成请求
	genReq := &genapi.GenerateRequest{
		Operation:   genapi.OperationTextToImage,
		Prompt:      req.Prompt,
		Size:        req.Size,
		Quality:     req.Quality,
		Style:       req.Style,
		OutputCount: req.N,
		Model:       task.Model,
		Metadata: map[string]interface{}{
			"taskId": task.ID,
			"userId": task.UserID,
		},
	}

	// 设置默认值
	if genReq.OutputCount == 0 {
		genReq.OutputCount = 1
	}

	// 更新进度
	if err := w.repo.UpdateAITaskProgress(ctx, task.ID, 30); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	// 使用指定的提供商或默认提供商
	providerName := task.Provider
	if providerName == "" {
		providerName = "gemini" // 默认使用 gemini
	}

	w.logger.Info("calling image generation API",
		zap.String("taskId", task.ID),
		zap.String("provider", providerName),
		zap.String("prompt", req.Prompt))

	// 调用 GenAPI 生成图片
	resp, err := w.genAPI.GenerateImage(ctx, providerName, genReq)
	if err != nil {
		w.logger.Error("image generation failed",
			zap.String("taskId", task.ID),
			zap.String("provider", providerName),
			zap.Error(err))
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = err.Error()
		w.repo.UpdateAITask(ctx, task)
		return fmt.Errorf("ai generate: %w", err)
	}

	// 检查响应错误
	if resp.Error != "" {
		w.logger.Error("image generation returned error",
			zap.String("taskId", task.ID),
			zap.String("error", resp.Error),
			zap.String("errorCode", resp.ErrorCode))
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = resp.Error
		w.repo.UpdateAITask(ctx, task)
		return fmt.Errorf("ai generate error: %s", resp.Error)
	}

	// 更新进度
	if err := w.repo.UpdateAITaskProgress(ctx, task.ID, 90); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	// 计算 token 使用量
	tokensUsed := 0
	if resp.Usage != nil {
		tokensUsed = resp.Usage.TotalTokens
	}

	w.logger.Info("image generation completed",
		zap.String("taskId", task.ID),
		zap.String("provider", resp.Provider),
		zap.Int("imageCount", len(resp.ImageURLs)),
		zap.Int("tokensUsed", tokensUsed),
		zap.Duration("duration", resp.Duration()))

	// 上传图片到OSS并替换URL
	ossImageURLs := make([]string, 0, len(resp.ImageURLs))
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil && len(resp.ImageURLs) > 0 {
		for i, imageURL := range resp.ImageURLs {
			// 生成OSS object key
			objectKey := fmt.Sprintf("ai-tasks/%s/images/%d.jpg", task.ID, i)

			// 上传到OSS
			ossURL, err := ossClient.UploadFileFromURL(objectKey, imageURL)
			if err != nil {
				w.logger.Warn("failed to upload image to OSS, using original URL",
					zap.String("taskId", task.ID),
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
				w.logger.Debug("image uploaded to OSS",
					zap.String("taskId", task.ID),
					zap.Int("index", i),
					zap.String("ossURL", ossURL))
			}
		}
	} else {
		// OSS客户端不可用，使用原始URL
		if ossClient == nil {
			w.logger.Warn("OSS client not available, using original image URLs",
				zap.String("taskId", task.ID))
		}
		ossImageURLs = resp.ImageURLs
	}

	result := &domain.AIImageGenerationResult{
		URLs:       ossImageURLs,
		TokensUsed: tokensUsed,
	}

	outputJSON, _ := json.Marshal(result)
	task.Output = string(outputJSON)
	task.TokensUsed = tokensUsed
	task.Status = domain.AITaskStatusCompleted
	task.Progress = 100
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime

	return w.repo.UpdateAITask(ctx, task)
}

// processVideoGeneration 处理视频生成
func (w *Worker) processVideoGeneration(ctx context.Context, task *domain.AITask) error {
	w.logger.Info("processing video generation task",
		zap.String("taskId", task.ID),
		zap.String("provider", task.Provider))

	var req domain.AIVideoGenerationRequest
	if err := json.Unmarshal([]byte(task.Input), &req); err != nil {
		w.logger.Error("failed to unmarshal input",
			zap.String("taskId", task.ID),
			zap.Error(err))
		return fmt.Errorf("unmarshal input: %w", err)
	}

	// 确定操作类型和参考图片
	operation := genapi.OperationTextToVideo
	var referenceImageURL string
	if req.StartFrame != "" {
		operation = genapi.OperationImageToVideo
		referenceImageURL = req.StartFrame
	} else if len(req.SceneImages) > 0 {
		operation = genapi.OperationImageToVideo
		referenceImageURL = req.SceneImages[0]
	}

	// 构建视频生成请求
	genReq := &genapi.GenerateRequest{
		Operation:         operation,
		Prompt:            req.Prompt,
		DurationSeconds:   req.Duration,
		Resolution:        req.Resolution,
		Style:             req.Style,
		ReferenceImageURL: referenceImageURL,
		FirstFrameURL:     req.StartFrame,
		LastFrameURL:      req.EndFrame,
		ReferenceImages:   req.SceneImages,
		Model:             task.Model,
		Metadata: map[string]interface{}{
			"taskId": task.ID,
			"userId": task.UserID,
		},
	}

	// 设置默认值
	if genReq.DurationSeconds == 0 {
		genReq.DurationSeconds = 6 // 默认6秒
	}
	if genReq.Resolution == "" {
		genReq.Resolution = "1080p"
	}

	// 更新进度
	if err := w.repo.UpdateAITaskProgress(ctx, task.ID, 20); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	// 使用指定的提供商或默认提供商
	providerName := task.Provider
	if providerName == "" {
		providerName = "hailuo" // 默认使用 hailuo 视频生成
	}

	w.logger.Info("calling video generation API",
		zap.String("taskId", task.ID),
		zap.String("provider", providerName),
		zap.String("operation", string(operation)),
		zap.String("prompt", req.Prompt),
		zap.Int("duration", req.Duration))

	// 调用 GenAPI 生成视频
	resp, err := w.genAPI.GenerateVideo(ctx, providerName, genReq)
	if err != nil {
		w.logger.Error("video generation failed",
			zap.String("taskId", task.ID),
			zap.String("provider", providerName),
			zap.Error(err))
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = err.Error()
		w.repo.UpdateAITask(ctx, task)
		return fmt.Errorf("ai generate: %w", err)
	}

	// 检查响应错误
	if resp.Error != "" {
		w.logger.Error("video generation returned error",
			zap.String("taskId", task.ID),
			zap.String("error", resp.Error),
			zap.String("errorCode", resp.ErrorCode))
		task.Status = domain.AITaskStatusFailed
		task.ErrorMessage = resp.Error
		w.repo.UpdateAITask(ctx, task)
		return fmt.Errorf("ai generate error: %s", resp.Error)
	}

	// 如果任务还在处理中（异步任务），轮询直到完成
	if resp.Status == "processing" || resp.Status == "pending" {
		w.logger.Info("video generation is async, starting polling",
			zap.String("taskId", task.ID),
			zap.String("providerTaskId", resp.TaskID),
			zap.String("provider", providerName),
			zap.String("status", resp.Status))

		// 轮询获取任务状态
		pollResp, pollErr := w.pollVideoTaskStatus(ctx, task, providerName, resp.TaskID)
		if pollErr != nil {
			w.logger.Error("video task polling failed",
				zap.String("taskId", task.ID),
				zap.String("providerTaskId", resp.TaskID),
				zap.Error(pollErr))
			task.Status = domain.AITaskStatusFailed
			task.ErrorMessage = pollErr.Error()
			w.repo.UpdateAITask(ctx, task)
			return pollErr
		}

		// 使用轮询结果
		resp = pollResp
	}

	// 更新进度
	if err := w.repo.UpdateAITaskProgress(ctx, task.ID, 90); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	// 计算 token 使用量
	tokensUsed := 0
	if resp.Usage != nil {
		tokensUsed = resp.Usage.TotalTokens
		if resp.Usage.DurationSeconds > 0 {
			tokensUsed += resp.Usage.DurationSeconds // 视频时长也可能算入用量
		}
	}

	// 上传视频到OSS并替换URL（包括轮询完成后的情况）
	ossVideoURL := resp.VideoURL
	ossThumbnailURL := resp.ThumbnailURL
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil {
		if resp.VideoURL != "" {
			// 生成OSS object key
			objectKey := fmt.Sprintf("ai-tasks/%s/video.mp4", task.ID)

			// 上传视频到OSS
			uploadedURL, err := ossClient.UploadFileFromURL(objectKey, resp.VideoURL)
			if err != nil {
				w.logger.Warn("failed to upload video to OSS, using original URL",
					zap.String("taskId", task.ID),
					zap.String("originalURL", resp.VideoURL),
					zap.Error(err))
				// 上传失败时使用原始URL
			} else {
				// 清理URL：移除查询参数，确保HTTPS
				ossVideoURL = strings.Split(uploadedURL, "?")[0]
				ossVideoURL = strings.ReplaceAll(ossVideoURL, "http://", "https://")
				w.logger.Debug("video uploaded to OSS",
					zap.String("taskId", task.ID),
					zap.String("ossURL", ossVideoURL))
			}
		}

		// 如果有缩略图，也上传到OSS
		if resp.ThumbnailURL != "" {
			thumbnailKey := fmt.Sprintf("ai-tasks/%s/thumbnail.jpg", task.ID)
			uploadedThumbnailURL, err := ossClient.UploadFileFromURL(thumbnailKey, resp.ThumbnailURL)
			if err != nil {
				w.logger.Warn("failed to upload thumbnail to OSS, using original URL",
					zap.String("taskId", task.ID),
					zap.String("originalURL", resp.ThumbnailURL),
					zap.Error(err))
				// 上传失败时使用原始URL
			} else {
				// 清理URL：移除查询参数，确保HTTPS
				ossThumbnailURL = strings.Split(uploadedThumbnailURL, "?")[0]
				ossThumbnailURL = strings.ReplaceAll(ossThumbnailURL, "http://", "https://")
				w.logger.Debug("thumbnail uploaded to OSS",
					zap.String("taskId", task.ID),
					zap.String("ossURL", ossThumbnailURL))
			}
		}
	} else {
		w.logger.Warn("OSS client not available, using original video URLs",
			zap.String("taskId", task.ID))
	}

	w.logger.Info("video generation completed",
		zap.String("taskId", task.ID),
		zap.String("provider", resp.Provider),
		zap.String("videoURL", ossVideoURL),
		zap.Int("tokensUsed", tokensUsed),
		zap.Duration("duration", resp.Duration()))

	// 构建生成结果（使用OSS URL）
	result := &domain.AIVideoGenerationResult{
		VideoURL:     ossVideoURL,
		ThumbnailURL: ossThumbnailURL,
		Duration:     req.Duration,
		TokensUsed:   tokensUsed,
	}

	outputJSON, _ := json.Marshal(result)
	task.Output = string(outputJSON)
	task.TokensUsed = tokensUsed
	task.Status = domain.AITaskStatusCompleted
	task.Progress = 100
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime

	return w.repo.UpdateAITask(ctx, task)
}

// ========== 渲染任务处理 ==========

// processRenderTask 处理渲染任务
func (w *Worker) processRenderTask(ctx context.Context, message *queue.TaskMessage) error {
	w.logger.Info("processing render task",
		zap.String("taskId", message.TaskID))

	// 从数据库获取任务详情
	task, err := w.repo.GetRenderTask(ctx, message.TaskID)
	if err != nil {
		w.logger.Error("failed to get render task",
			zap.String("taskId", message.TaskID),
			zap.Error(err))
		return fmt.Errorf("get task: %w", err)
	}

	// 更新状态为处理中
	startTime := time.Now().Unix()
	task.Status = domain.RenderTaskStatusProcessing
	task.StartedAt = &startTime
	task.Progress = 10
	if err := w.repo.UpdateRenderTask(ctx, task); err != nil {
		w.logger.Error("failed to update task status",
			zap.String("taskId", task.ID),
			zap.Error(err))
	}

	w.logger.Info("starting render process",
		zap.String("taskId", task.ID),
		zap.String("type", string(task.Type)),
		zap.String("storyId", task.StoryID))

	// 获取故事的所有分镜
	storyboards, err := w.repo.StoryboardsByStory(ctx, task.StoryID, 100, 0)
	if err != nil {
		w.logger.Error("failed to get storyboards",
			zap.String("taskId", task.ID),
			zap.String("storyId", task.StoryID),
			zap.Error(err))
		task.Status = domain.RenderTaskStatusFailed
		task.ErrorMessage = "failed to get storyboards: " + err.Error()
		w.repo.UpdateRenderTask(ctx, task)
		return fmt.Errorf("get storyboards: %w", err)
	}

	if len(storyboards) == 0 {
		w.logger.Warn("no storyboards found for story",
			zap.String("taskId", task.ID),
			zap.String("storyId", task.StoryID))
		task.Status = domain.RenderTaskStatusFailed
		task.ErrorMessage = "no storyboards found for this story"
		w.repo.UpdateRenderTask(ctx, task)
		return fmt.Errorf("no storyboards found")
	}

	w.logger.Info("found storyboards for rendering",
		zap.String("taskId", task.ID),
		zap.Int("count", len(storyboards)))

	// 解析渲染配置
	var config domain.RenderConfig
	if task.Config != "" {
		if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
			w.logger.Warn("failed to parse render config, using defaults",
				zap.String("taskId", task.ID),
				zap.Error(err))
		}
	}

	// 设置默认配置
	if config.Resolution == "" {
		config.Resolution = "1080p"
	}
	if config.Quality == "" {
		config.Quality = "high"
	}

	// 根据任务类型执行不同的渲染流程
	var renderErr error
	switch task.Type {
	case domain.RenderTaskTypeVideo:
		renderErr = w.renderVideo(ctx, task, storyboards, &config)
	case domain.RenderTaskTypeImageSet:
		renderErr = w.renderImageSet(ctx, task, storyboards, &config)
	case domain.RenderTaskTypeAnimation:
		renderErr = w.renderAnimation(ctx, task, storyboards, &config)
	default:
		renderErr = fmt.Errorf("unknown render type: %s", task.Type)
	}

	if renderErr != nil {
		w.logger.Error("render failed",
			zap.String("taskId", task.ID),
			zap.String("type", string(task.Type)),
			zap.Error(renderErr))
		task.Status = domain.RenderTaskStatusFailed
		task.ErrorMessage = renderErr.Error()
		w.repo.UpdateRenderTask(ctx, task)
		return renderErr
	}

	// 生成输出
	task.OutputURL = fmt.Sprintf("/uploads/renders/%s.%s", task.ID, getFileExtension(task.Type))
	task.ThumbnailURL = fmt.Sprintf("/uploads/renders/%s_thumb.jpg", task.ID)
	task.FileSize = 10485760 // 10MB

	if task.Type == domain.RenderTaskTypeVideo {
		task.Duration = 120
		task.Resolution = "1080p"
	}

	task.Status = domain.RenderTaskStatusCompleted
	task.Progress = 100
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime

	w.logger.Info("render task completed",
		zap.String("taskId", task.ID),
		zap.String("outputURL", task.OutputURL),
		zap.Int64("fileSize", task.FileSize))

	return w.repo.UpdateRenderTask(ctx, task)
}

// ========== 渲染辅助方法 ==========

// renderVideo 渲染视频
func (w *Worker) renderVideo(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	w.logger.Info("rendering video",
		zap.String("taskId", task.ID),
		zap.Int("storyboardCount", len(storyboards)),
		zap.String("resolution", config.Resolution))

	totalSteps := len(storyboards) + 1 // 每个分镜 + 最终合成
	currentStep := 0
	baseProgress := 20

	var generatedVideos []string

	// 为每个分镜生成视频片段
	for i, sb := range storyboards {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		currentStep++
		progress := baseProgress + (currentStep * 60 / totalSteps)
		if err := w.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			w.logger.Warn("failed to update progress", zap.Error(err))
		}

		w.logger.Debug("generating video for storyboard",
			zap.String("taskId", task.ID),
			zap.String("storyboardId", sb.ID),
			zap.Int("index", i+1),
			zap.Int("total", len(storyboards)))

		// 构建视频生成请求
		prompt := sb.Content
		if sb.Title != "" {
			prompt = sb.Title + ": " + prompt
		}

		genReq := &genapi.GenerateRequest{
			Operation:       genapi.OperationTextToVideo,
			Prompt:          prompt,
			DurationSeconds: 5, // 默认每个片段5秒
			Resolution:      config.Resolution,
			Model:           config.Format,
			Metadata: map[string]interface{}{
				"taskId":       task.ID,
				"storyboardId": sb.ID,
				"index":        i,
			},
		}

		// 如果分镜有图片，使用图生视频
		imageURL := getStoryboardImageURL(sb)
		if imageURL != "" {
			genReq.Operation = genapi.OperationImageToVideo
			genReq.ReferenceImageURL = imageURL
		}

		// 调用视频生成
		resp, err := w.genAPI.GenerateVideo(ctx, "hailuo", genReq)
		if err != nil {
			w.logger.Error("failed to generate video for storyboard",
				zap.String("taskId", task.ID),
				zap.String("storyboardId", sb.ID),
				zap.Error(err))
			// 继续处理其他分镜，记录错误但不中断
			continue
		}

		// 如果是异步任务，等待完成
		if resp.Status == "processing" || resp.Status == "pending" {
			aiTask := &domain.AITask{ID: task.ID}
			resp, err = w.pollVideoTaskStatus(ctx, aiTask, "hailuo", resp.TaskID)
			if err != nil {
				w.logger.Error("failed to poll video status",
					zap.String("taskId", task.ID),
					zap.String("storyboardId", sb.ID),
					zap.Error(err))
				continue
			}
		}

		if resp.VideoURL != "" {
			generatedVideos = append(generatedVideos, resp.VideoURL)
			w.logger.Info("video segment generated",
				zap.String("taskId", task.ID),
				zap.String("storyboardId", sb.ID),
				zap.String("videoURL", resp.VideoURL))
		}
	}

	if len(generatedVideos) == 0 {
		return fmt.Errorf("no video segments were generated")
	}

	// 更新进度到90%
	if err := w.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	// 设置输出信息
	// 在实际实现中，这里应该调用视频合成服务将多个片段合成为一个视频
	// 目前直接使用第一个视频作为输出
	task.OutputURL = generatedVideos[0]
	if len(generatedVideos) > 1 {
		// 保存所有视频片段URL到config中
		segmentsJSON, _ := json.Marshal(map[string]interface{}{
			"segments": generatedVideos,
			"count":    len(generatedVideos),
		})
		task.Config = string(segmentsJSON)
	}

	// 估算文件大小和时长
	task.Duration = len(generatedVideos) * 5                       // 每个片段5秒
	task.FileSize = int64(len(generatedVideos)) * 10 * 1024 * 1024 // 估算每个片段10MB
	task.Resolution = config.Resolution

	w.logger.Info("video rendering completed",
		zap.String("taskId", task.ID),
		zap.Int("segmentCount", len(generatedVideos)),
		zap.String("outputURL", task.OutputURL))

	return nil
}

// renderImageSet 渲染图片集
func (w *Worker) renderImageSet(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	w.logger.Info("rendering image set",
		zap.String("taskId", task.ID),
		zap.Int("storyboardCount", len(storyboards)))

	totalSteps := len(storyboards)
	currentStep := 0
	baseProgress := 20

	var generatedImages []string

	// 为每个分镜生成图片
	for i, sb := range storyboards {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		currentStep++
		progress := baseProgress + (currentStep * 70 / totalSteps)
		if err := w.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			w.logger.Warn("failed to update progress", zap.Error(err))
		}

		// 如果分镜已有图片，直接使用
		imageURL := getStoryboardImageURL(sb)
		if imageURL != "" {
			generatedImages = append(generatedImages, imageURL)
			continue
		}

		w.logger.Debug("generating image for storyboard",
			zap.String("taskId", task.ID),
			zap.String("storyboardId", sb.ID),
			zap.Int("index", i+1))

		// 构建图片生成请求
		prompt := sb.Content
		if sb.Title != "" {
			prompt = sb.Title + ": " + prompt
		}

		genReq := &genapi.GenerateRequest{
			Operation:   genapi.OperationTextToImage,
			Prompt:      prompt,
			Size:        "1024x1024",
			Quality:     config.Quality,
			OutputCount: 1,
			Metadata: map[string]interface{}{
				"taskId":       task.ID,
				"storyboardId": sb.ID,
				"index":        i,
			},
		}

		// 调用图片生成
		resp, err := w.genAPI.GenerateImage(ctx, "gemini", genReq)
		if err != nil {
			w.logger.Error("failed to generate image for storyboard",
				zap.String("taskId", task.ID),
				zap.String("storyboardId", sb.ID),
				zap.Error(err))
			continue
		}

		if len(resp.ImageURLs) > 0 {
			generatedImages = append(generatedImages, resp.ImageURLs[0])
			w.logger.Info("image generated",
				zap.String("taskId", task.ID),
				zap.String("storyboardId", sb.ID),
				zap.String("imageURL", resp.ImageURLs[0]))
		}
	}

	if len(generatedImages) == 0 {
		return fmt.Errorf("no images were generated")
	}

	// 更新进度到90%
	if err := w.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	// 设置输出信息
	// 在实际实现中，这里应该将所有图片打包成zip
	imagesJSON, _ := json.Marshal(map[string]interface{}{
		"images": generatedImages,
		"count":  len(generatedImages),
	})
	task.Config = string(imagesJSON)
	task.OutputURL = fmt.Sprintf("/uploads/renders/%s_images.zip", task.ID)
	task.FileSize = int64(len(generatedImages)) * 2 * 1024 * 1024 // 估算每张图2MB

	w.logger.Info("image set rendering completed",
		zap.String("taskId", task.ID),
		zap.Int("imageCount", len(generatedImages)))

	return nil
}

// renderAnimation 渲染动画
func (w *Worker) renderAnimation(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	w.logger.Info("rendering animation",
		zap.String("taskId", task.ID),
		zap.Int("storyboardCount", len(storyboards)))

	totalSteps := len(storyboards) + 1
	currentStep := 0
	baseProgress := 20

	var generatedImages []string

	// 先生成所有图片
	for i, sb := range storyboards {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		currentStep++
		progress := baseProgress + (currentStep * 50 / totalSteps)
		if err := w.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			w.logger.Warn("failed to update progress", zap.Error(err))
		}

		// 如果分镜已有图片，直接使用
		imageURL := getStoryboardImageURL(sb)
		if imageURL != "" {
			generatedImages = append(generatedImages, imageURL)
			continue
		}

		w.logger.Debug("generating image for animation frame",
			zap.String("taskId", task.ID),
			zap.String("storyboardId", sb.ID),
			zap.Int("index", i+1))

		prompt := sb.Content
		if sb.Title != "" {
			prompt = sb.Title + ": " + prompt
		}

		genReq := &genapi.GenerateRequest{
			Operation:   genapi.OperationTextToImage,
			Prompt:      prompt,
			Size:        "512x512", // 动画使用较小尺寸
			Quality:     "standard",
			OutputCount: 1,
			Metadata: map[string]interface{}{
				"taskId":       task.ID,
				"storyboardId": sb.ID,
				"index":        i,
			},
		}

		resp, err := w.genAPI.GenerateImage(ctx, "gemini", genReq)
		if err != nil {
			w.logger.Error("failed to generate animation frame",
				zap.String("taskId", task.ID),
				zap.String("storyboardId", sb.ID),
				zap.Error(err))
			continue
		}

		if len(resp.ImageURLs) > 0 {
			generatedImages = append(generatedImages, resp.ImageURLs[0])
		}
	}

	if len(generatedImages) == 0 {
		return fmt.Errorf("no animation frames were generated")
	}

	// 更新进度到80%（合成动画）
	if err := w.repo.UpdateRenderTaskProgress(ctx, task.ID, 80); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	// 在实际实现中，这里应该调用GIF合成服务
	// 目前保存帧信息供后续处理
	framesJSON, _ := json.Marshal(map[string]interface{}{
		"frames":    generatedImages,
		"count":     len(generatedImages),
		"frameRate": config.FrameRate,
	})
	task.Config = string(framesJSON)
	task.OutputURL = fmt.Sprintf("/uploads/renders/%s.gif", task.ID)
	task.Duration = len(generatedImages)                     // 每帧1秒
	task.FileSize = int64(len(generatedImages)) * 500 * 1024 // 估算每帧500KB

	// 更新进度到90%
	if err := w.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		w.logger.Warn("failed to update progress", zap.Error(err))
	}

	w.logger.Info("animation rendering completed",
		zap.String("taskId", task.ID),
		zap.Int("frameCount", len(generatedImages)))

	return nil
}

// ========== 视频任务轮询 ==========

// pollVideoTaskStatus 轮询视频生成任务状态直到完成
func (w *Worker) pollVideoTaskStatus(ctx context.Context, task *domain.AITask, providerName, providerTaskID string) (*genapi.GenerateResponse, error) {
	const (
		maxPollAttempts = 120             // 最大轮询次数
		pollInterval    = 5 * time.Second // 轮询间隔
		progressPerPoll = 1               // 每次轮询增加的进度
	)

	w.logger.Info("starting video task polling",
		zap.String("taskId", task.ID),
		zap.String("providerTaskId", providerTaskID),
		zap.String("provider", providerName))

	currentProgress := 50 // 从50%开始

	for attempt := 0; attempt < maxPollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		// 调用 GenAPI 获取任务状态
		statusResp, err := w.genAPI.GetVideoStatus(ctx, providerName, providerTaskID)
		if err != nil {
			w.logger.Warn("failed to get video task status",
				zap.String("taskId", task.ID),
				zap.String("providerTaskId", providerTaskID),
				zap.Int("attempt", attempt+1),
				zap.Error(err))
			// 继续轮询，不立即失败
			continue
		}

		w.logger.Debug("video task status",
			zap.String("taskId", task.ID),
			zap.String("providerTaskId", providerTaskID),
			zap.String("status", statusResp.Status),
			zap.Int("attempt", attempt+1))

		// 更新进度
		if currentProgress < 90 {
			currentProgress += progressPerPoll
			if currentProgress > 90 {
				currentProgress = 90
			}
			if err := w.repo.UpdateAITaskProgress(ctx, task.ID, currentProgress); err != nil {
				w.logger.Warn("failed to update progress", zap.Error(err))
			}
		}

		// 检查任务状态
		switch statusResp.Status {
		case "completed", "success":
			w.logger.Info("video task completed",
				zap.String("taskId", task.ID),
				zap.String("providerTaskId", providerTaskID),
				zap.String("videoURL", statusResp.VideoURL))
			return statusResp, nil

		case "failed", "error":
			errMsg := statusResp.Error
			if errMsg == "" {
				errMsg = "video generation failed"
			}
			return nil, fmt.Errorf("video generation failed: %s", errMsg)

		case "cancelled":
			return nil, fmt.Errorf("video generation was cancelled")

		case "processing", "pending", "queued":
			// 继续轮询
			continue

		default:
			w.logger.Warn("unknown video task status",
				zap.String("taskId", task.ID),
				zap.String("status", statusResp.Status))
			continue
		}
	}

	return nil, fmt.Errorf("video generation timed out after %d attempts", maxPollAttempts)
}

// ========== 辅助函数 ==========

func buildStoryPrompt(req *domain.AIStoryGenerationRequest) string {
	prompt := "作为一位专业的故事创作者，请根据以下要求创作一个精彩的故事：\n\n"
	prompt += fmt.Sprintf("核心提示: %s\n\n", req.Prompt)
	if req.Context != "" {
		prompt += fmt.Sprintf("背景信息: %s\n\n", req.Context)
	}
	return prompt
}

func buildEnhancePrompt(req *domain.AIPromptEnhanceRequest) string {
	return fmt.Sprintf("作为专业的AI提示词工程师，请优化以下提示词：\n\n%s", req.OriginalPrompt)
}

func parseStoryResult(text string, req *domain.AIStoryGenerationRequest) *domain.AIStoryGenerationResult {
	var result domain.AIStoryGenerationResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		result.Title = "生成的故事"
		result.Content = text
		result.Summary = text[:min(100, len(text))] + "..."
	}
	return &result
}

func extractEnhancedPrompt(text string) string {
	var result struct {
		EnhancedPrompt string `json:"enhancedPrompt"`
	}
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result.EnhancedPrompt
	}
	return text
}

func extractImprovements(text string) string {
	var result struct {
		Improvements string `json:"improvements"`
	}
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result.Improvements
	}
	return ""
}

func getFileExtension(renderType domain.RenderTaskType) string {
	switch renderType {
	case domain.RenderTaskTypeVideo:
		return "mp4"
	case domain.RenderTaskTypeImageSet:
		return "zip"
	case domain.RenderTaskTypeAnimation:
		return "gif"
	default:
		return "mp4"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getStoryboardImageURL 获取分镜的图片URL
func getStoryboardImageURL(sb *domain.Storyboard) string {
	// 优先从StoryboardScenes中获取图片 (AI-generated plot scenes)
	if len(sb.StoryboardScenes) > 0 && sb.StoryboardScenes[0].Image != "" {
		return sb.StoryboardScenes[0].Image
	}
	// 从SceneRefs中获取图片 (story-level static scenes)
	if len(sb.SceneRefs) > 0 && sb.SceneRefs[0].StoryScene != nil && sb.SceneRefs[0].StoryScene.Image != "" {
		return sb.SceneRefs[0].StoryScene.Image
	}
	return ""
}
