package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// GetDashboardStoryboards returns storyboards from stories the user follows or created.
// GET /api/dashboard/storyboards?limit=20&offset=0
func (h *Handler) GetDashboardStoryboards(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.svc.DashboardStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"storyboards": items,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetDashboardGroupStoryboards returns storyboards from groups the user joined.
// GET /api/dashboard/groups/storyboards?limit=20&offset=0
func (h *Handler) GetDashboardGroupStoryboards(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.svc.DashboardGroupStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"storyboards": items,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetDashboardCharacterStoryboards returns storyboards that followed characters participate in.
// GET /api/dashboard/characters/storyboards?limit=20&offset=0
func (h *Handler) GetDashboardCharacterStoryboards(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.svc.DashboardCharacterStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"storyboards": items,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetTrendingStoryboards returns storyboards from trending stories.
// GET /api/dashboard/trending/storyboards?limit=20&offset=0
func (h *Handler) GetTrendingStoryboards(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.svc.TrendingStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"storyboards": items,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetPublicTrendingStoryboards returns trending storyboards accessible to all users.
// GET /api/public/trending/storyboards?limit=20&offset=0
// Works for both authenticated and non-authenticated users.
// Authenticated users get personalized results with their contributions prioritized.
// Guest users get globally trending storyboards.
func (h *Handler) GetPublicTrendingStoryboards(c *gin.Context) {
	userID := authPkg.GetUserID(c) // May be empty string for guests
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.svc.PublicTrendingStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"storyboards": items,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}
