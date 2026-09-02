package http

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"gorm.io/gorm"
)

type WorkflowRegistryHandler struct {
	registry *service.WorkflowRegistryService
	apiKey   string
}

func NewWorkflowRegistryHandler(registry *service.WorkflowRegistryService, apiKey string) *WorkflowRegistryHandler {
	return &WorkflowRegistryHandler{registry: registry, apiKey: strings.TrimSpace(apiKey)}
}

func (h *WorkflowRegistryHandler) RegisterInternalRoutes(r *gin.Engine) {
	g := r.Group("/api/v1/agent-policy")
	g.Use(h.internalAPIKeyMiddleware())
	{
		g.PUT("/prompt-versions/:id", h.publishPromptVersion)
		g.GET("/prompt-versions/:id", h.getPromptVersion)
		g.PUT("/workflow-releases/:id", h.publishRelease)
		g.GET("/workflow-releases/:id", h.getRelease)
		g.POST("/workflow-releases/:id/pause-bindings", h.pauseReleaseBindings)
		g.POST("/workflow-releases/:id/rebind", h.rebindRelease)
		g.PUT("/workflow-bindings/:id", h.saveBinding)
		g.GET("/workflow-catalog", h.catalog)
		g.POST("/workflow-resolve", h.resolve)
		g.GET("/workflow-stats", h.stats)
	}
}

func (h *WorkflowRegistryHandler) rebindRelease(c *gin.Context) {
	var body struct {
		Surface     string `json:"surface" binding:"required"`
		Action      string `json:"action" binding:"required"`
		WorkflowKey string `json:"workflowKey" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	count, err := h.registry.RebindWorkflowBindings(c.Request.Context(), body.Surface, body.Action, body.WorkflowKey, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, gin.H{"releaseId": c.Param("id"), "updatedBindings": count})
}

func (h *WorkflowRegistryHandler) pauseReleaseBindings(c *gin.Context) {
	count, err := h.registry.PauseReleaseBindings(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, gin.H{"releaseId": c.Param("id"), "disabledBindings": count})
}

func (h *WorkflowRegistryHandler) stats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	items, err := h.registry.ReleaseStats(c.Request.Context(), days)
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, gin.H{"items": items, "days": days})
}

func (h *WorkflowRegistryHandler) RegisterPublicRoutes(g *gin.RouterGroup) {
	g.GET("/workflows/catalog", h.catalog)
	g.POST("/workflows/resolve", h.resolve)
}

func (h *WorkflowRegistryHandler) publishPromptVersion(c *gin.Context) {
	var prompt domain.PromptTemplateVersion
	if err := c.ShouldBindJSON(&prompt); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	if prompt.ID == "" {
		prompt.ID = c.Param("id")
	}
	if prompt.ID != c.Param("id") {
		InvalidParams(c, "prompt id does not match path")
		return
	}
	saved, err := h.registry.PublishPromptVersion(c.Request.Context(), &prompt)
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, saved)
}

func (h *WorkflowRegistryHandler) getPromptVersion(c *gin.Context) {
	prompt, err := h.registry.GetPromptVersion(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, prompt)
}

func (h *WorkflowRegistryHandler) publishRelease(c *gin.Context) {
	var release domain.WorkflowRelease
	if err := c.ShouldBindJSON(&release); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	if release.ID == "" {
		release.ID = c.Param("id")
	}
	if release.ID != c.Param("id") {
		InvalidParams(c, "workflow release id does not match path")
		return
	}
	saved, err := h.registry.PublishRelease(c.Request.Context(), &release)
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, saved)
}

func (h *WorkflowRegistryHandler) getRelease(c *gin.Context) {
	release, err := h.registry.GetRelease(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, release)
}

func (h *WorkflowRegistryHandler) saveBinding(c *gin.Context) {
	var binding domain.WorkflowBinding
	if err := c.ShouldBindJSON(&binding); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	if binding.ID == "" {
		binding.ID = c.Param("id")
	}
	if binding.ID != c.Param("id") {
		InvalidParams(c, "workflow binding id does not match path")
		return
	}
	saved, err := h.registry.SaveBinding(c.Request.Context(), &binding)
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, saved)
}

func (h *WorkflowRegistryHandler) catalog(c *gin.Context) {
	entries, err := h.registry.Catalog(c.Request.Context(), c.Query("surface"), c.Query("action"), c.Query("tenantId"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, gin.H{"items": entries})
}

func (h *WorkflowRegistryHandler) resolve(c *gin.Context) {
	var body struct {
		Surface  string         `json:"surface" binding:"required"`
		Action   string         `json:"action" binding:"required"`
		TenantID string         `json:"tenantId"`
		Input    map[string]any `json:"input,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	resolution, err := h.registry.ResolveForInput(c.Request.Context(), body.Surface, body.Action, body.TenantID, body.Input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	Success(c, resolution)
}

func (h *WorkflowRegistryHandler) internalAPIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.apiKey == "" {
			InternalError(c, "workflow registry internal API is not configured")
			return
		}
		key := strings.TrimSpace(c.GetHeader("X-Internal-Api-Key"))
		if key == "" {
			key = strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		}
		if key != h.apiKey {
			Unauthorized(c, "invalid internal api key")
			return
		}
		c.Next()
	}
}

func (h *WorkflowRegistryHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		NotFound(c, "workflow resource not found")
	case errors.Is(err, domain.ErrWorkflowReleaseImmutable), errors.Is(err, domain.ErrPromptVersionImmutable):
		Error(c, CodeConflict, err.Error())
	default:
		InvalidParams(c, err.Error())
	}
}
