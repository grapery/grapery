package http

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// GetGenerationExecution returns the durable snapshot used to recover a
// disconnected client. Ownership is enforced even though the agent-policy
// variant of this endpoint uses service authentication.
func (h *Handler) GetGenerationExecution(c *gin.Context) {
	run, ok := h.authorizedGenerationExecution(c)
	if !ok {
		return
	}
	Success(c, run)
}

// FindLatestGenerationExecution recovers the durable run for a content draft
// when the client has lost its local run ID (for example after reinstalling).
func (h *Handler) FindLatestGenerationExecution(c *gin.Context) {
	if h.generationRuntime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	kind, contentID := strings.TrimSpace(c.Query("kind")), strings.TrimSpace(c.Query("contentId"))
	if kind == "" || contentID == "" {
		InvalidParams(c, "kind and contentId are required")
		return
	}
	run, err := h.generationRuntime.FindLatestExecution(c.Request.Context(), userID, kind, contentID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, run)
}

func (h *Handler) ListGenerationExecutionEvents(c *gin.Context) {
	run, ok := h.authorizedGenerationExecution(c)
	if !ok {
		return
	}
	after, _ := strconv.ParseInt(c.DefaultQuery("afterSequence", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
	events, err := h.generationRuntime.ListEvents(c.Request.Context(), run.ID, after, limit)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"events": events, "snapshot": run})
}

func (h *Handler) CancelGenerationExecution(c *gin.Context) {
	run, ok := h.authorizedGenerationExecution(c)
	if !ok {
		return
	}
	switch strings.ToLower(run.Status) {
	case "succeeded", "failed", "cancelled":
		Success(c, run)
		return
	}
	run.Status = "cancelled"
	run.Error = "cancelled by client"
	now := time.Now().UTC()
	run.CompletedAt = &now
	saved, err := h.generationRuntime.SaveExecution(c.Request.Context(), run, "run.cancelled")
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, saved)
}

func (h *Handler) authorizedGenerationExecution(c *gin.Context) (*domain.GenerationExecution, bool) {
	if h.generationRuntime == nil {
		InternalError(c, "generation runtime unavailable")
		return nil, false
	}
	userID, ok := RequireUserID(c)
	if !ok {
		return nil, false
	}
	run, err := h.generationRuntime.GetExecution(c.Request.Context(), c.Param("id"))
	if err != nil {
		HandleError(c, err)
		return nil, false
	}
	if run.UserID == "" || run.UserID != userID {
		Forbidden(c, "generation does not belong to current user")
		return nil, false
	}
	return run, true
}
