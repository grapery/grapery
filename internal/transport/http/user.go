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
	viewerID := GetUserID(c)
	user, err := h.svc.UserProfile(c.Request.Context(), userID, viewerID)
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
	viewerID := GetUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	stories, err := h.svc.GetUserStories(c.Request.Context(), userID, viewerID, limit, offset)
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
	viewerID := GetUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	storyboards, err := h.svc.GetUserStoryboards(c.Request.Context(), userID, viewerID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	h.attachStoryboardIsLikedMany(c, storyboards)
	domain.RedactStoryboardViewsUnlessCreatorMany(storyboards, viewerID)
	Success(c, gin.H{
		"storyboards": storyboards,
		"count":       len(storyboards),
	})
}

// GetDashboardStoryboards 当前登录用户的故事板列表（含草稿与已发布），供草稿箱等与碎片草稿合并展示。
// GET /api/v1/dashboard/storyboards?limit=20&offset=0
func (h *Handler) GetDashboardStoryboards(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	storyboards, total, err := h.svc.ListDashboardStoryboards(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	h.attachStoryboardIsLikedMany(c, storyboards)
	domain.RedactStoryboardViewsUnlessCreatorMany(storyboards, userID)
	Success(c, gin.H{
		"storyboards": storyboards,
		"total":       total,
	})
}

// REMOVED: GetUserDrafts - not in StoryCreationAppUI design
// REMOVED: GetDraftStoryboards - not in StoryCreationAppUI design
// REMOVED: GetUserActivityList - not in StoryCreationAppUI design

// GetLikedStories 获取用户点赞的故事
// GET /api/users/:id/liked-stories
func (h *Handler) GetLikedStories(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}
	viewerID := GetUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	stories, err := h.svc.GetLikedStories(c.Request.Context(), userID, viewerID, limit, offset)
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
	viewerID := GetUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	storyboards, err := h.svc.GetLikedStoryboards(c.Request.Context(), userID, viewerID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	h.attachStoryboardIsLikedMany(c, storyboards)
	domain.RedactStoryboardViewsUnlessCreatorMany(storyboards, viewerID)
	Success(c, gin.H{
		"storyboards": storyboards,
		"count":       len(storyboards),
	})
}

// REMOVED: GetDraftStoryboards - not in StoryCreationAppUI design
// REMOVED: GetUserActivityList - not in StoryCreationAppUI design

// BlockUser 屏蔽用户
// POST /api/v1/users/:id/block
func (h *Handler) BlockUser(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	blockedID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	if err := h.svc.BlockUser(c.Request.Context(), userID, blockedID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "user blocked successfully"})
}

// UnblockUser 取消屏蔽用户
// DELETE /api/v1/users/:id/block
func (h *Handler) UnblockUser(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	blockedID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	if err := h.svc.UnblockUser(c.Request.Context(), userID, blockedID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "user unblocked successfully"})
}

// ReportUserRequest 举报用户请求
type ReportUserRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// ReportUser 举报用户
// POST /api/v1/users/:id/report
func (h *Handler) ReportUser(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	reportedID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	var req ReportUserRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.svc.ReportUser(c.Request.Context(), userID, reportedID, req.Reason); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "user reported successfully"})
}
