package http

import (
	"github.com/gin-gonic/gin"
)

// GetMyCreatorAnalytics returns cached analytics for the current user.
// GET /api/v1/me/creator-analytics?range=7d|30d
func (h *Handler) GetMyCreatorAnalytics(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	rangeKey := c.DefaultQuery("range", "7d")
	dto, err := h.svc.GetMyCreatorAnalytics(c.Request.Context(), userID, rangeKey)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, dto)
}
