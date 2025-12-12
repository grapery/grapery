package http

import (
	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// GetUserStats 获取用户统计数据
// GET /api/users/:id/stats
func (h *Handler) GetUserStats(c *gin.Context) {
	userID := c.Param("id")

	stats, err := h.svc.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		NotFound(c, "user not found")
		return
	}

	Success(c, stats)
}

// GetStoryStats 获取故事统计数据
// GET /api/stories/:id/stats
func (h *Handler) GetStoryStats(c *gin.Context) {
	storyID := c.Param("id")

	stats, err := h.svc.GetStoryStats(c.Request.Context(), storyID)
	if err != nil {
		NotFound(c, "story not found")
		return
	}

	Success(c, stats)
}

// GetDashboardStats 获取仪表盘统计数据
// GET /api/dashboard/stats
func (h *Handler) GetDashboardStats(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	stats, err := h.svc.GetDashboardStats(c.Request.Context(), userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, stats)
}
