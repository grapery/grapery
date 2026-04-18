package http

import (
	"github.com/gin-gonic/gin"
)

// REMOVED: GetDashboardStats - not in StoryCreationAppUI design

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
