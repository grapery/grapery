package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// REMOVED: GetDashboardStoryboards - not in StoryCreationAppUI design
// REMOVED: GetDashboardCharacterStoryboards - not in StoryCreationAppUI design
// REMOVED: GetTrendingStoryboards (authenticated) - not in StoryCreationAppUI design

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
