package http

import (
	"errors"
	"net/http"
	"strconv"
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
	UserInput              string `json:"userInput" binding:"required,min=1,max=2000"`
	ReferenceImageURL      string `json:"referenceImageUrl" binding:"required"`
	Style                  string `json:"style"`
	PanelCount             int    `json:"panelCount"`
	Visibility             string `json:"visibility"`
	Topic                  string `json:"topic"`
	AspectRatio            string `json:"aspectRatio" binding:"omitempty,oneof=1:1 16:9 9:16 3:4 4:3"`
	DialogueMode           string `json:"dialogueMode"`
	ConsistencyLevel       string `json:"consistencyLevel"`
	EnableReferenceAssets  *bool  `json:"enableReferenceAssets"`
	IncludeGenerationTrace bool   `json:"includeGenerationTrace"`
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
		UserInput:              strings.TrimSpace(req.UserInput),
		ReferenceImageURL:      strings.TrimSpace(req.ReferenceImageURL),
		Style:                  strings.TrimSpace(req.Style),
		PanelCount:             req.PanelCount,
		Visibility:             strings.TrimSpace(req.Visibility),
		Topic:                  normalizePanelTopicLabel(req.Topic),
		AspectRatio:            strings.TrimSpace(req.AspectRatio),
		DialogueMode:           strings.TrimSpace(req.DialogueMode),
		ConsistencyLevel:       strings.TrimSpace(req.ConsistencyLevel),
		EnableReferenceAssets:  req.EnableReferenceAssets,
		IncludeGenerationTrace: req.IncludeGenerationTrace,
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

	imageSlots, imageProgress := fragmentPanelImageSnapshot(task)
	resp := gin.H{
		"taskId":          task.ID,
		"status":          task.Status,
		"progress":        task.Progress,
		"currentStep":     task.CurrentStep,
		"stage":           fragmentPanelGenerationStage(task),
		"draftFragmentId": task.DraftFragmentID,
		"createdAt":       task.CreatedAt,
		"startedAt":       task.StartedAt,
		"completedAt":     task.CompletedAt,
		"imageSlots":      imageSlots,
		"imageProgress":   imageProgress,
		"cost":            fragmentPanelGenerationCostSnapshot(task),
		// 客户端生成记录页展示原始提示词与风格等（来自任务表 request_json）
		"request": gin.H{
			"userInput":             task.Request.UserInput,
			"referenceImageUrl":     task.Request.ReferenceImageURL,
			"style":                 task.Request.Style,
			"panelCount":            task.Request.PanelCount,
			"visibility":            task.Request.Visibility,
			"topic":                 task.Request.Topic,
			"consistencyLevel":      task.Request.ConsistencyLevel,
			"enableReferenceAssets": task.Request.EnableReferenceAssets,
		},
	}

	if len(task.Plan) > 0 {
		resp["plan"] = task.Plan
		resp["imagePlan"] = fragmentPanelImagePlan(task)
		resp["chatMessages"] = fragmentPanelChatMessages(task)
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
			resp["storyText"] = task.Result.CombinedContent
		}
		resp["visualBible"] = task.Result.VisualBible
		resp["anchorImages"] = task.Result.AnchorImages
		resp["consistencyIssues"] = task.Result.ConsistencyIssues
		if task.Request.IncludeGenerationTrace {
			resp["visualEvidence"] = task.Result.VisualEvidence
			resp["generationTrace"] = task.Result.GenerationTrace
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

func fragmentPanelGenerationStage(task *domain.FragmentPanelGenerationTask) string {
	step := strings.TrimSpace(task.CurrentStep)
	switch {
	case step == "understanding_reference" || step == "plan_ready":
		return "story"
	case step == "generating_reference_assets":
		return "style"
	case strings.HasPrefix(step, "generating_panel_") || step == "panel_render":
		return "images"
	case step == "checking_consistency":
		return "review"
	case step == "completed" || task.Status == "completed":
		return "completed"
	default:
		if task.Status == "failed" || task.Status == "cancelled" {
			return task.Status
		}
		return "preparing"
	}
}

func fragmentPanelImageSnapshot(task *domain.FragmentPanelGenerationTask) ([]gin.H, gin.H) {
	total := task.Request.PanelCount
	if total < 1 {
		total = 1
	}
	if len(task.Plan) > total {
		total = len(task.Plan)
	}
	var panels []domain.FragmentPanelResultItem
	if task.Result != nil {
		panels = task.Result.Panels
		if len(panels) > total {
			total = len(panels)
		}
	}

	slots := make([]gin.H, 0, total)
	completed := 0
	for i := 0; i < total; i++ {
		title := "第" + strconv.Itoa(i+1) + "页"
		caption := ""
		if i < len(task.Plan) {
			caption = strings.TrimSpace(task.Plan[i].Caption)
			if caption == "" {
				caption = strings.TrimSpace(task.Plan[i].ImagePrompt)
			}
		}
		imageURL := ""
		if i < len(panels) {
			imageURL = strings.TrimSpace(panels[i].ImageURL)
			if caption == "" {
				caption = strings.TrimSpace(panels[i].Caption)
			}
		}
		status := "planned"
		if imageURL != "" {
			status = "completed"
			completed++
		} else if task.Status == "failed" {
			status = "failed"
		} else if strings.HasPrefix(task.CurrentStep, "generating_panel_") && i == completed {
			status = "generating"
		}
		slots = append(slots, gin.H{
			"index":    i + 1,
			"title":    title,
			"caption":  caption,
			"status":   status,
			"imageUrl": imageURL,
		})
	}
	return slots, gin.H{
		"completedCount": completed,
		"totalCount":     total,
	}
}

func fragmentPanelImagePlan(task *domain.FragmentPanelGenerationTask) []gin.H {
	out := make([]gin.H, 0, len(task.Plan))
	for i, panel := range task.Plan {
		caption := strings.TrimSpace(panel.Caption)
		if caption == "" {
			caption = strings.TrimSpace(panel.ImagePrompt)
		}
		out = append(out, gin.H{
			"index":   i + 1,
			"caption": caption,
			"status":  "planned",
		})
	}
	return out
}

func fragmentPanelChatMessages(task *domain.FragmentPanelGenerationTask) []gin.H {
	if len(task.Plan) == 0 {
		return []gin.H{}
	}
	lines := make([]string, 0, len(task.Plan))
	for i, panel := range task.Plan {
		caption := strings.TrimSpace(panel.Caption)
		if caption == "" {
			caption = strings.TrimSpace(panel.ImagePrompt)
		}
		if caption != "" {
			lines = append(lines, "第"+strconv.Itoa(i+1)+"页："+caption)
		}
	}
	if len(lines) == 0 {
		return []gin.H{}
	}
	return []gin.H{{
		"id":   task.ID + ":image_plan",
		"type": "image_plan",
		"text": strings.Join(lines, "\n"),
	}}
}

func fragmentPanelGenerationCostSnapshot(task *domain.FragmentPanelGenerationTask) gin.H {
	count := task.Request.PanelCount
	if count < 1 {
		count = 1
	}
	points := count * 8
	return gin.H{
		"amount": points,
		"unit":   "点数",
		"text":   "本次创作消耗 " + strconv.Itoa(points) + " 点数",
	}
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

// RetryPanelGeneration POST /fragment-panels/generate/:taskId/retry
// 客户端「重试」语义；当前与 Resume 等价（服务端从失败或可恢复的状态重新排队任务），路由独立便于统计与日后分叉。
func (h *FragmentPanelGenerationHandler) RetryPanelGeneration(c *gin.Context) {
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
		g.POST("/:taskId/retry", h.RetryPanelGeneration)
		g.POST("/:taskId/resume", h.ResumePanelGeneration)
		g.GET("/:taskId", h.GetPanelGeneration)
	}
}
