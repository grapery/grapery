package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FragmentPanelGenerationHandler handles multi-panel reference fragment generation APIs.
type FragmentPanelGenerationHandler struct {
	svc    *service.FragmentPanelGenerationService
	logger *zap.Logger
}

// normalizePanelTopicLabel trims whitespace and a single leading “#”; max 200 runes for DB.
func normalizePanelTopicLabel(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) > 200 {
		s = string(r[:200])
	}
	return strings.TrimSpace(s)
}

// NewFragmentPanelGenerationHandler constructs the handler.
func NewFragmentPanelGenerationHandler(svc *service.FragmentPanelGenerationService, logger *zap.Logger) *FragmentPanelGenerationHandler {
	return &FragmentPanelGenerationHandler{svc: svc, logger: logger}
}

// CreatePanelGenerationRequest POST body for /fragment-panels/generate
type CreatePanelGenerationRequest struct {
	UserInput         string `json:"userInput" binding:"required,min=1,max=2000"`
	ReferenceImageURL string `json:"referenceImageUrl" binding:"required"`
	Style             string `json:"style"`
	PanelCount        int    `json:"panelCount"`
	Visibility        string `json:"visibility"`
	Topic             string `json:"topic"`
	AspectRatio       string `json:"aspectRatio" binding:"omitempty,oneof=1:1 16:9 9:16 3:4 4:3"`
	LayoutPreset      string `json:"layoutPreset"`
	GutterStyle       string `json:"gutterStyle"`
	DialogueMode      string `json:"dialogueMode"`
	OutputMode        string `json:"outputMode"`
}

// CreatePanelGeneration POST /fragment-panels/generate
func (h *FragmentPanelGenerationHandler) CreatePanelGeneration(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreatePanelGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domainReq := domain.FragmentPanelGenerationRequest{
		UserInput:         strings.TrimSpace(req.UserInput),
		ReferenceImageURL: strings.TrimSpace(req.ReferenceImageURL),
		Style:             strings.TrimSpace(req.Style),
		PanelCount:        req.PanelCount,
		Visibility:        strings.TrimSpace(req.Visibility),
		Topic:             normalizePanelTopicLabel(req.Topic),
		AspectRatio:       strings.TrimSpace(req.AspectRatio),
		LayoutPreset:      strings.TrimSpace(req.LayoutPreset),
		GutterStyle:       strings.TrimSpace(req.GutterStyle),
		DialogueMode:      strings.TrimSpace(req.DialogueMode),
		OutputMode:        strings.TrimSpace(req.OutputMode),
	}

	task, err := h.svc.StartGeneration(c.Request.Context(), userID, domainReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"taskId":          task.ID,
		"draftFragmentId": task.DraftFragmentID,
		"status":          task.Status,
		"progress":        task.Progress,
		"currentStep":     task.CurrentStep,
	})
}

// GetPanelGeneration GET /fragment-panels/generate/:taskId
func (h *FragmentPanelGenerationHandler) GetPanelGeneration(c *gin.Context) {
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

	task, err := h.svc.GetTask(c.Request.Context(), taskID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		if errors.Is(err, service.ErrFragmentPanelTaskForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load task"})
		return
	}

	resp := gin.H{
		"taskId":          task.ID,
		"status":          task.Status,
		"progress":        task.Progress,
		"currentStep":     task.CurrentStep,
		"draftFragmentId": task.DraftFragmentID,
		"createdAt":       task.CreatedAt,
		"startedAt":       task.StartedAt,
		"completedAt":     task.CompletedAt,
		// 客户端生成记录页展示原始提示词与风格等（来自任务表 request_json）
		"request": gin.H{
			"userInput":         task.Request.UserInput,
			"referenceImageUrl": task.Request.ReferenceImageURL,
			"style":             task.Request.Style,
			"panelCount":        task.Request.PanelCount,
			"visibility":        task.Request.Visibility,
			"topic":             task.Request.Topic,
		},
	}

	if len(task.Plan) > 0 {
		resp["plan"] = task.Plan
	}

	if task.Result != nil && len(task.Result.Panels) > 0 {
		panels := make([]gin.H, 0, len(task.Result.Panels))
		for _, p := range task.Result.Panels {
			panels = append(panels, gin.H{
				"index":    p.Index,
				"imageUrl": p.ImageURL,
				"caption":  p.Caption,
			})
		}
		resp["panels"] = panels
		if task.Result.CombinedContent != "" {
			resp["combinedContent"] = task.Result.CombinedContent
		}
	}

	if task.Metrics != nil {
		steps := make([]gin.H, 0, len(task.Metrics.Steps))
		for _, m := range task.Metrics.Steps {
			steps = append(steps, gin.H{
				"name":       m.Name,
				"tokens":     m.Tokens,
				"durationMs": m.DurationMs,
				"provider":   m.Provider,
				"model":      m.Model,
			})
		}
		resp["metrics"] = gin.H{
			"steps":           steps,
			"totalTokens":     task.Metrics.TotalTokens,
			"totalDurationMs": task.Metrics.TotalDurationMs,
		}
	}

	if task.ErrorMessage != "" {
		resp["error"] = task.ErrorMessage
	}

	c.JSON(http.StatusOK, resp)
}

// ResumePanelGeneration POST /fragment-panels/generate/:taskId/resume
func (h *FragmentPanelGenerationHandler) ResumePanelGeneration(c *gin.Context) {
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

	task, err := h.svc.ResumeGeneration(c.Request.Context(), userID, taskID)
	if err != nil {
		if errors.Is(err, service.ErrPanelGenerationResumeConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrPanelGenerationDraftResetFailed) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrPanelGenerationNotResumable) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		if errors.Is(err, service.ErrFragmentPanelTaskForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"taskId":          task.ID,
		"draftFragmentId": task.DraftFragmentID,
		"status":          task.Status,
		"progress":        task.Progress,
		"currentStep":     task.CurrentStep,
	})
}

// RegisterRoutes registers /generate routes under a parent group (e.g. /api/v1/fragment-panels).
func (h *FragmentPanelGenerationHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	g := router.Group("/generate")
	// main.go 常在父级已挂 AuthMiddleware 时传入 nil；Gin 的 Use(nil) 会在链上注册空 handler，c.Next() 时 panic。
	if authMiddleware != nil {
		g.Use(authMiddleware)
	}
	{
		g.POST("", h.CreatePanelGeneration)
		g.POST("/:taskId/resume", h.ResumePanelGeneration)
		g.GET("/:taskId", h.GetPanelGeneration)
	}
}
