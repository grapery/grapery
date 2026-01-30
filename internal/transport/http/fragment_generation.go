package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

type FragmentGenerationHandler struct {
	fragmentGenService *service.FragmentGenerationService
	logger             *zap.Logger
}

func NewFragmentGenerationHandler(fragmentGenService *service.FragmentGenerationService, logger *zap.Logger) *FragmentGenerationHandler {
	return &FragmentGenerationHandler{
		fragmentGenService: fragmentGenService,
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
	Visibility string   `json:"visibility" binding:"required,oneof=public followers private"`
}

// GenerateFragment handles POST /fragments/generate
func (h *FragmentGenerationHandler) GenerateFragment(c *gin.Context) {
	userID := c.GetString("user_id")
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
		Visibility: req.Visibility,
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

	// 从 service 获取任务状态（需要添加到 service）
	// task, err := h.fragmentGenService.GetTask(c.Request.Context(), taskID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Get generation status - TODO: implement service method",
		"taskId":  taskID,
	})
}

// ListGenerationTasks handles GET /fragments/generate
func (h *FragmentGenerationHandler) ListGenerationTasks(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	// TODO: 调用 service 获取任务列表
	c.JSON(http.StatusOK, gin.H{
		"message": "List generation tasks - TODO: implement",
		"userId":  userID,
		"page":    page,
		"limit":   limit,
	})
}

// CancelGeneration handles DELETE /fragments/generate/:taskId
func (h *FragmentGenerationHandler) CancelGeneration(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id is required"})
		return
	}

	// TODO: 调用 service 取消任务
	c.JSON(http.StatusOK, gin.H{
		"message": "Cancel generation - TODO: implement",
		"taskId":  taskID,
	})
}

// RegisterRoutes registers the fragment generation routes
func (h *FragmentGenerationHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	fragmentGenGroup := router.Group("/generate")
	fragmentGenGroup.Use(authMiddleware)
	{
		fragmentGenGroup.POST("", h.GenerateFragment)
		fragmentGenGroup.GET(":taskId", h.GetGenerationStatus)
		fragmentGenGroup.GET("", h.ListGenerationTasks)
		fragmentGenGroup.DELETE(":taskId", h.CancelGeneration)
	}
}
