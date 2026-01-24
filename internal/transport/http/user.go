package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

func (h *Handler) FollowUser(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	followeeID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	if err := h.svc.FollowUser(c.Request.Context(), userID, followeeID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "user followed successfully"})
}

func (h *Handler) UnfollowUser(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	followeeID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	if err := h.svc.UnfollowUser(c.Request.Context(), userID, followeeID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "user unfollowed successfully"})
}

func (h *Handler) GetFollowers(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	currentUserID := GetUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	followers, err := h.svc.GetFollowersWithFollowStatus(c.Request.Context(), userID, currentUserID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"followers": followers, "count": len(followers)})
}

func (h *Handler) GetFollowing(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	currentUserID := GetUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	following, err := h.svc.GetFollowingWithFollowStatus(c.Request.Context(), userID, currentUserID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"following": following, "count": len(following)})
}

func (h *Handler) GetUserProfile(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	user, err := h.svc.UserProfile(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, user)
}

// UpdateUserProfile 更新用户资料
// PUT /api/users/:id
func (h *Handler) UpdateUserProfile(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	targetID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	// 只能更新自己的资料
	if userID != targetID {
		Forbidden(c, "can only update own profile")
		return
	}

	var req struct {
		DisplayName         *string `json:"displayName"`
		Bio                 *string `json:"bio"`
		Avatar              *string `json:"avatar"`
		Background          *string `json:"background"`
		Location            *string `json:"location"`
		Website             *string `json:"website"`
		AIPromptPreferences *string `json:"aiPromptPreferences"`
	}
	if !BindJSON(c, &req) {
		return
	}

	updateReq := &service.UpdateProfileRequest{
		DisplayName:         req.DisplayName,
		Bio:                 req.Bio,
		Avatar:              req.Avatar,
		Background:          req.Background,
		Location:            req.Location,
		Website:             req.Website,
		AIPromptPreferences: req.AIPromptPreferences,
	}

	user, err := h.svc.UpdateUserProfile(c.Request.Context(), userID, updateReq)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, user)
}

// UpdateUserAvatar 更新用户头像
// PUT /api/users/:id/avatar
func (h *Handler) UpdateUserAvatar(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	targetID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	if userID != targetID {
		Forbidden(c, "can only update own avatar")
		return
	}

	var req struct {
		AvatarURL string `json:"avatarUrl" binding:"required"`
	}
	if !BindJSON(c, &req) {
		return
	}

	if err := h.svc.UpdateUserAvatar(c.Request.Context(), userID, req.AvatarURL); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "avatar updated successfully"})
}

// UpdateUserBackground 更新用户背景图
// PUT /api/users/:id/background
func (h *Handler) UpdateUserBackground(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	targetID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	if userID != targetID {
		Forbidden(c, "can only update own background")
		return
	}

	var req struct {
		BackgroundURL string `json:"backgroundUrl" binding:"required"`
	}
	if !BindJSON(c, &req) {
		return
	}

	if err := h.svc.UpdateUserBackground(c.Request.Context(), userID, req.BackgroundURL); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "background updated successfully"})
}

// GetUserStories 获取用户的故事列表
// GET /api/users/:id/stories
func (h *Handler) GetUserStories(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	stories, err := h.svc.GetUserStories(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"stories": stories,
		"count":   len(stories),
	})
}

// GetUserCharacters 获取用户的角色列表
// GET /api/users/:id/characters
func (h *Handler) GetUserCharacters(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	characters, err := h.svc.GetUserCharacters(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"characters": characters,
		"count":      len(characters),
	})
}

// GetUserStoryboards 获取用户的故事板列表
// GET /api/users/:id/storyboards
func (h *Handler) GetUserStoryboards(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	storyboards, err := h.svc.GetUserStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"storyboards": storyboards,
		"count":       len(storyboards),
	})
}

// GetUserDrafts 获取用户的草稿列表
// GET /api/users/:id/drafts
func (h *Handler) GetUserDrafts(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	drafts, err := h.svc.GetUserDrafts(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"drafts": drafts,
		"count":  len(drafts),
	})
}

// GetLikedStories 获取用户点赞的故事
// GET /api/users/:id/liked-stories
func (h *Handler) GetLikedStories(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	stories, err := h.svc.GetLikedStories(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"stories": stories,
		"count":   len(stories),
	})
}

// GetLikedCharacters 获取用户点赞（关注）的角色
// GET /api/users/:id/liked-characters
func (h *Handler) GetLikedCharacters(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	characters, err := h.svc.GetLikedCharacters(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"characters": characters,
		"count":      len(characters),
	})
}

// GetLikedStoryboards 获取用户点赞的故事板
// GET /api/users/:id/liked-storyboards
func (h *Handler) GetLikedStoryboards(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	storyboards, err := h.svc.GetLikedStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"storyboards": storyboards,
		"count":       len(storyboards),
	})
}

// GetDraftStoryboards 获取用户的草稿故事板
// GET /api/users/:id/draft-storyboards
func (h *Handler) GetDraftStoryboards(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	storyboards, err := h.svc.GetDraftStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"storyboards": storyboards,
		"count":       len(storyboards),
	})
}

// GetUserActivityList 获取用户活动列表
// GET /api/users/:id/activities
// Query params:
//   - limit: int (default 50, max 100)
//   - offset: int (default 0)
//   - time_range: string (today, week, month) - default: week
//   - date: string (YYYY-MM-DD) - filter by specific date
func (h *Handler) GetUserActivityList(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	timeRangeStr := c.DefaultQuery("time_range", "week")
	date := c.Query("date")

	// Convert time range string to enum
	var timeRange domain.ActivityTimeRange
	switch timeRangeStr {
	case "today":
		timeRange = domain.TimeRangeToday
	case "week":
		timeRange = domain.TimeRangeWeek
	case "month":
		timeRange = domain.TimeRangeMonth
	default:
		timeRange = domain.TimeRangeWeek
	}

	activities, count, err := h.svc.GetUserActivitiesWithFilter(c.Request.Context(), userID, timeRange, date, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"activities": activities,
		"count":      count,
	})
}

// GetUserActivityHeatmap 获取用户活动热力图
// GET /api/users/:id/activities/heatmap
// Query params:
//   - time_range: string (today, week, month) - default: week
func (h *Handler) GetUserActivityHeatmapByID(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	timeRangeStr := c.DefaultQuery("time_range", "week")

	// Convert time range string to enum
	var timeRange domain.ActivityTimeRange
	switch timeRangeStr {
	case "today":
		timeRange = domain.TimeRangeToday
	case "week":
		timeRange = domain.TimeRangeWeek
	case "month":
		timeRange = domain.TimeRangeMonth
	default:
		timeRange = domain.TimeRangeWeek
	}

	heatmapResponse, err := h.svc.GetUserActivityHeatmap(c.Request.Context(), userID, timeRange)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, heatmapResponse)
}
