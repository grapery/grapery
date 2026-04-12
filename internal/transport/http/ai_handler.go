package http

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ============== 故事生成 ==============

// GenerateStory 生成故事内容
// POST /api/ai/generate-story
func (h *Handler) GenerateStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req domain.AIStoryGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// 验证必填字段
	if req.Prompt == "" {
		InvalidParams(c, "prompt is required")
		return
	}

	task, err := h.aiService.GenerateStory(c.Request.Context(), userID, &req)
	if err != nil {
		Error(c, CodeError, "failed to create generation task")
		return
	}

	Success(c, gin.H{
		"taskId":  task.ID,
		"status":  task.Status,
		"message": "Story generation started. Use GET /api/ai/tasks/{taskId} to check status.",
	})
}

// ============== 提示词增强 ==============

// EnhancePrompt 增强提示词
// POST /api/ai/enhance-prompt
func (h *Handler) EnhancePrompt(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req domain.AIPromptEnhanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	if req.OriginalPrompt == "" {
		InvalidParams(c, "originalPrompt is required")
		return
	}

	if req.TargetType == "" {
		req.TargetType = "image" // 默认为图片
	}

	task, err := h.aiService.EnhancePrompt(c.Request.Context(), userID, &req)
	if err != nil {
		Error(c, CodeError, "failed to create enhancement task")
		return
	}

	Success(c, gin.H{
		"taskId":  task.ID,
		"status":  task.Status,
		"message": "Prompt enhancement started. Use GET /api/ai/tasks/{taskId} to check status.",
	})
}

// ============== 图片生成 ==============

// GenerateImage 生成图片
// POST /api/ai/generate-image
func (h *Handler) GenerateImage(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.AIImageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	// 设置默认值
	if req.Size == "" {
		req.Size = "1024x1024"
	}
	if req.Quality == "" {
		req.Quality = "standard"
	}
	if req.N == 0 {
		req.N = 1
	}

	// 验证图片数量限制
	if req.N > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 10 images per request"})
		return
	}

	h.logger.Info("generating image",
		zap.String("userID", userID.(string)),
		zap.String("prompt", req.Prompt),
		zap.Int("count", req.N),
	)

	task, err := h.aiService.GenerateImage(c.Request.Context(), userID.(string), &req)
	if err != nil {
		h.logger.Error("failed to create image generation task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create generation task"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"taskId":  task.ID,
		"status":  task.Status,
		"message": "Image generation started. Use GET /api/ai/tasks/{taskId} to check status.",
	})
}

// ============== 视频生成 ==============

// GenerateVideo 生成视频
// POST /api/ai/generate-video
func (h *Handler) GenerateVideo(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.AIVideoGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	// 设置默认值
	if req.Duration == 0 {
		req.Duration = 5 // 默认5秒
	}
	if req.Resolution == "" {
		req.Resolution = "720p"
	}
	if req.FrameRate == 0 {
		req.FrameRate = 24
	}

	// 验证视频时长限制（假设最长2分钟）
	if req.Duration > 120 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum video duration is 120 seconds"})
		return
	}

	h.logger.Info("generating video",
		zap.String("userID", userID.(string)),
		zap.String("prompt", req.Prompt),
		zap.Int("duration", req.Duration),
	)

	task, err := h.aiService.GenerateVideo(c.Request.Context(), userID.(string), &req)
	if err != nil {
		h.logger.Error("failed to create video generation task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create generation task"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"taskId":  task.ID,
		"status":  task.Status,
		"message": "Video generation started. This may take several minutes. Use GET /api/ai/tasks/{taskId} to check status.",
	})
}

// ============== 任务查询 ==============

// GetTaskStatus 获取任务状态
// GET /api/ai/tasks/:id
func (h *Handler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task ID is required"})
		return
	}

	h.logger.Info("getting task status", zap.String("taskID", taskID))

	task, err := h.aiService.GetTaskStatus(c.Request.Context(), taskID)
	if err != nil {
		h.logger.Error("failed to get task status", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetTaskResult 获取任务结果
// GET /api/ai/tasks/:id/result
func (h *Handler) GetTaskResult(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task ID is required"})
		return
	}

	h.logger.Info("getting task result", zap.String("taskID", taskID))

	task, err := h.aiService.GetTaskResult(c.Request.Context(), taskID)
	if err != nil {
		h.logger.Error("failed to get task result", zap.Error(err))
		if err.Error() == "task not completed yet" {
			c.JSON(http.StatusAccepted, gin.H{
				"error":    "task is still processing",
				"status":   task.Status,
				"progress": task.Progress,
			})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found or failed"})
		return
	}

	// 根据任务类型解析输出结果
	var result interface{}
	switch task.Type {
	case domain.AITaskGenerateStory:
		var storyResult domain.AIStoryGenerationResult
		if err := jsonUnmarshal([]byte(task.Output), &storyResult); err == nil {
			result = storyResult
		}
	case domain.AITaskEnhancePrompt:
		var promptResult domain.AIPromptEnhanceResult
		if err := jsonUnmarshal([]byte(task.Output), &promptResult); err == nil {
			result = promptResult
		}
	case domain.AITaskGenerateImage:
		var imageResult domain.AIImageGenerationResult
		if err := jsonUnmarshal([]byte(task.Output), &imageResult); err == nil {
			result = imageResult
		}
	case domain.AITaskGenerateVideo:
		var videoResult domain.AIVideoGenerationResult
		if err := jsonUnmarshal([]byte(task.Output), &videoResult); err == nil {
			result = videoResult
		}
	}

	if result == nil {
		result = task.Output // 返回原始输出
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId":      task.ID,
		"type":        task.Type,
		"status":      task.Status,
		"result":      result,
		"tokensUsed":  task.TokensUsed,
		"createdAt":   task.CreatedAt,
		"completedAt": task.CompletedAt,
	})
}

// CancelTask 取消任务
// DELETE /api/ai/tasks/:id
func (h *Handler) CancelTask(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task ID is required"})
		return
	}

	h.logger.Info("cancelling task",
		zap.String("userID", userID.(string)),
		zap.String("taskID", taskID),
	)

	err := h.aiService.CancelTask(c.Request.Context(), taskID, userID.(string))
	if err != nil {
		h.logger.Error("failed to cancel task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task cancelled successfully"})
}

// ============== 辅助函数 ==============

// jsonUnmarshal JSON反序列化辅助函数
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
