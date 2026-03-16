package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/handler"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// parseIntParam safely parses a string to int
func parseIntParam(s string) (int, error) {
	return strconv.Atoi(s)
}

type FragmentGenerationHandler struct {
	fragmentGenService *service.FragmentGenerationService
	fragmentHandler    *handler.FragmentHandler
	logger             *zap.Logger
}

func NewFragmentGenerationHandler(fragmentGenService *service.FragmentGenerationService, fragmentHandler *handler.FragmentHandler, logger *zap.Logger) *FragmentGenerationHandler {
	return &FragmentGenerationHandler{
		fragmentGenService: fragmentGenService,
		fragmentHandler:    fragmentHandler,
		logger:             logger,
	}
}

// GenerateFragmentRequest 生成碎片故事的请求
type GenerateFragmentRequest struct {
	UserInput  string   `json:"userInput" binding:"required,min=1,max=500"`
	ImageUrls  []string `json:"imageUrls" binding:"max=10"`
	ImageCount int      `json:"imageCount" binding:"min=0,max=10"`
	Style      string   `json:"style" binding:"omitempty,oneof=fantasy realistic anime scifi"`
	Mood       string   `json:"mood" binding:"omitempty,oneof=happy sad mysterious romantic"`
	Length     string   `json:"length" binding:"omitempty,oneof=short medium long"`
	Language   string   `json:"language" binding:"required,oneof=zh-Hans en ja"`
	Visibility string   `json:"visibility" binding:"required,oneof=public followers followers_only private"`
}

// GenerateFragment handles POST /fragments/generate
func (h *FragmentGenerationHandler) GenerateFragment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req GenerateFragmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换为领域模型
	domainReq := domain.FragmentGenerationRequest{
		UserInput:  req.UserInput,
		ImageUrls:  req.ImageUrls,
		ImageCount: req.ImageCount,
		Style:      req.Style,
		Mood:       req.Mood,
		Length:     req.Length,
		Language:   req.Language,
		Visibility: domain.NormalizeFragmentVisibility(req.Visibility),
	}

	// 如果用户没有指定图片数量，默认生成1张
	if domainReq.ImageCount == 0 {
		domainReq.ImageCount = 1
	}

	// 调用服务生成碎片
	task, err := h.fragmentGenService.GenerateFragment(c.Request.Context(), userID, domainReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create generation task"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"taskId":   task.ID,
		"status":   task.Status,
		"progress": task.Progress,
	})
}

// GetGenerationStatus handles GET /fragments/generate/:taskId
func (h *FragmentGenerationHandler) GetGenerationStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id is required"})
		return
	}

	task, err := h.fragmentGenService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	response := gin.H{
		"taskId":      task.ID,
		"status":      task.Status,
		"progress":    task.Progress,
		"currentStep": task.CurrentStep,
		"createdAt":   task.CreatedAt,
	}

	if task.Result != nil {
		response["result"] = gin.H{
			"content":    task.Result.Content,
			"imageUrls":  task.Result.ImageUrls,
			"tokensUsed": task.Result.TokensUsed,
		}
	}

	if task.ErrorMessage != "" {
		response["error"] = task.ErrorMessage
	}

	c.JSON(http.StatusOK, response)
}

// ListGenerationTasks handles GET /fragments/generate
func (h *FragmentGenerationHandler) ListGenerationTasks(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := parseIntParam(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	tasks, total, err := h.fragmentGenService.ListTasks(c.Request.Context(), userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}

	// Convert tasks to response format
	taskResponses := make([]gin.H, len(tasks))
	for i, task := range tasks {
		taskResponses[i] = gin.H{
			"taskId":      task.ID,
			"status":      task.Status,
			"progress":    task.Progress,
			"currentStep": task.CurrentStep,
			"createdAt":   task.CreatedAt,
		}
		if task.Result != nil {
			taskResponses[i]["result"] = gin.H{
				"content":    task.Result.Content,
				"imageUrls":  task.Result.ImageUrls,
				"tokensUsed": task.Result.TokensUsed,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":  taskResponses,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// CancelGeneration handles DELETE /fragments/generate/:taskId
func (h *FragmentGenerationHandler) CancelGeneration(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id is required"})
		return
	}

	if err := h.fragmentGenService.CancelTask(c.Request.Context(), taskID, userID); err != nil {
		if err.Error() == "unauthorized: task does not belong to user" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId": taskID,
		"status": "cancelled",
	})
}

// RegisterRoutes registers the fragment generation routes
func (h *FragmentGenerationHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	fragmentGenGroup := router.Group("/generate")
	fragmentGenGroup.Use(authMiddleware)
	{
		fragmentGenGroup.POST("", h.GenerateFragment)
		fragmentGenGroup.GET("/:taskId", h.GetGenerationStatus)
		fragmentGenGroup.GET("", h.ListGenerationTasks)
		fragmentGenGroup.DELETE("/:taskId", h.CancelGeneration)
	}
}

// GetFragmentStyles handles GET /fragments/styles
func (h *FragmentGenerationHandler) GetFragmentStyles(c *gin.Context) {
	h.fragmentHandler.GetFragmentStyles(c)
}
