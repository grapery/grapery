package http

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// attachStoryViewerState sets isLiked / isFollowing for the authenticated viewer (optional; guests unchanged).
func (h *Handler) attachStoryViewerState(c *gin.Context, story *domain.Story) {
	if story == nil {
		return
	}
	uid := GetUserID(c)
	if uid == "" {
		return
	}
	h.svc.AttachStoryViewerState(c.Request.Context(), uid, story)
}

// GetTrendingStoriesPublic returns top trending stories (guest accessible).
// Trending is determined by: followers > likes > updated_at.
// No time range restriction - includes all published stories.
// GET /api/public/stories/trending?limit=20
func (h *Handler) GetTrendingStoriesPublic(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	stories, err := h.svc.GetTrendingStories24h(c.Request.Context(), limit)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"stories": stories,
		"total":   len(stories),
		"limit":   limit,
		"offset":  0,
	})
}

// CreateStory 创建故事
func (h *Handler) CreateStory(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req service.CreateStoryRequest
	if !BindJSON(c, &req) {
		return
	}

	story, err := h.svc.CreateStory(c.Request.Context(), userID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	h.attachStoryViewerState(c, story)
	Success(c, story)
}

// GetStory 获取故事详情
func (h *Handler) GetStory(c *gin.Context) {
	storyID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	story, err := h.svc.GetStory(c.Request.Context(), storyID)
	if err != nil {
		HandleError(c, err)
		return
	}

	h.attachStoryViewerState(c, story)
	Success(c, story)
}

// ListStories 获取故事列表
func (h *Handler) ListStories(c *gin.Context) {
	var req service.StoryListRequest
	if !BindQuery(c, &req) {
		return
	}

	stories, total, err := h.svc.ListStories(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"stories": stories,
		"total":   total,
		"limit":   req.Limit,
		"offset":  req.Offset,
	})
}

// UpdateStory 更新故事
func (h *Handler) UpdateStory(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	var req service.UpdateStoryRequest
	if !BindJSON(c, &req) {
		return
	}

	story, err := h.svc.UpdateStory(c.Request.Context(), userID, storyID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	h.attachStoryViewerState(c, story)
	Success(c, story)
}

// DeleteStory 删除故事
func (h *Handler) DeleteStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	err := h.svc.DeleteStory(c.Request.Context(), userID, storyID)
	if err != nil {
		if err.Error() == "unauthorized" {
			Forbidden(c, "you can only delete your own stories")
			return
		}
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "story deleted successfully"})
}

// LikeStory 点赞故事
func (h *Handler) LikeStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	err := h.svc.LikeStory(c.Request.Context(), userID, storyID)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "story liked successfully"})
}

// UnlikeStory 取消点赞故事
func (h *Handler) UnlikeStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	err := h.svc.UnlikeStory(c.Request.Context(), userID, storyID)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "story unliked successfully"})
}

// FollowStory 关注故事
func (h *Handler) FollowStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	err := h.svc.FollowStory(c.Request.Context(), userID, storyID)
	if err != nil {
		if errors.Is(err, service.ErrAuthUserNotFound) {
			Unauthorized(c, "session invalid, please sign in again")
			return
		}
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "story followed successfully"})
}

// UnfollowStory 取消关注故事
func (h *Handler) UnfollowStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	err := h.svc.UnfollowStory(c.Request.Context(), userID, storyID)
	if err != nil {
		if errors.Is(err, service.ErrAuthUserNotFound) {
			Unauthorized(c, "session invalid, please sign in again")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "story unfollowed successfully"})
}

// ========== 故事渲染功能 ==========

// RenderStory AI渲染故事（同步模式）
// 丰富故事描述和生成背景图片
// POST /api/stories/:id/render
func (h *Handler) RenderStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	var req service.RenderStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	result, err := h.svc.RenderStory(c.Request.Context(), userID, storyID, req)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "unauthorized: not story owner" {
			Unauthorized(c, "only story owner can render")
			return
		}
		if err.Error() == "no render options selected" {
			InvalidParams(c, "please select at least one render option")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"story":               result.Story,
		"enrichedDescription": result.EnrichedDescription,
		"backgroundUrl":       result.BackgroundURL,
		"coverUrl":            result.CoverURL,
		"tokensUsed":          result.TokensUsed,
	})
}

// RenderStoryMedia 渲染故事媒体（异步模式）
// 生成视频、图片集、动画等复杂媒体内容
// POST /api/stories/:id/render-media
func (h *Handler) RenderStoryMedia(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	var req service.MediaRenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	task, err := h.svc.RenderStoryMedia(c.Request.Context(), userID, storyID, req)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "unauthorized: not story owner" {
			Unauthorized(c, "only story owner can render")
			return
		}
		if err.Error() == "story has no panels to render" {
			InvalidParams(c, "story has no panels to render")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"taskId":  task.ID,
		"status":  task.Status,
		"message": "Media render task created successfully. Use GET /api/stories/" + storyID + "/render-status to check progress.",
	})
}

// GetRenderTaskStatus 获取渲染任务状态
// GET /api/stories/:id/render-status
// 支持两种查询方式：
// 1. 通过 taskId 查询参数获取指定任务：?taskId=xxx
// 2. 不提供 taskId 则自动获取该故事的最新渲染任务
func (h *Handler) GetRenderTaskStatus(c *gin.Context) {
	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	taskID := c.Query("taskId")

	var task *domain.RenderTask
	var err error

	if taskID != "" {
		// 方式1：通过指定的 taskId 获取任务状态
		task, err = h.svc.GetRenderTaskStatus(c.Request.Context(), taskID)
	} else {
		// 方式2：获取该故事的最新渲染任务
		task, err = h.svc.GetLatestRenderTaskByStoryID(c.Request.Context(), storyID)
	}

	if err != nil {
		if err.Error() == "task not found" {
			NotFound(c, "render task not found")
			return
		}
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, task)
}

// ========== 故事发布功能 ==========

// PublishStory 发布故事
// POST /api/stories/:id/publish
func (h *Handler) PublishStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	publication, err := h.svc.PublishStory(c.Request.Context(), userID, storyID)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "unauthorized: not story owner" {
			Unauthorized(c, "only story owner can publish")
			return
		}
		if err.Error() == "story is already published" {
			Error(c, CodeError, "story is already published")
			return
		}
		if err.Error() == "cannot publish empty story" {
			Error(c, CodeError, "cannot publish empty story")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"publication": publication,
		"message":     "Story published successfully",
	})
}

// UnpublishStory 取消发布故事
// POST /api/stories/:id/unpublish
func (h *Handler) UnpublishStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	err := h.svc.UnpublishStory(c.Request.Context(), userID, storyID)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "unauthorized: not story owner" {
			Unauthorized(c, "only story owner can unpublish")
			return
		}
		if err.Error() == "story is not published" {
			Error(c, CodeError, "story is not published")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "Story unpublished successfully"})
}

// ========== Story Contributor Handlers ==========

// InviteStoryContributor 邀请故事贡献者
// POST /api/stories/:id/contributors
func (h *Handler) InviteStoryContributor(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	var req service.InviteStoryContributorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	contributor, err := h.svc.InviteStoryContributor(c.Request.Context(), userID, storyID, req)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "user not found" {
			NotFound(c, "user not found")
			return
		}
		if err.Error() == "permission denied: not a contributor" {
			Forbidden(c, "you don't have permission to invite contributors")
			return
		}
		if err.Error() == "user is already a contributor" {
			Error(c, CodeError, "user is already a contributor")
			return
		}
		if err.Error() == "cannot invite the story author" {
			Error(c, CodeError, "cannot invite the story author")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, contributor)
}

// RemoveStoryContributor 移除故事贡献者
// DELETE /api/stories/:id/contributors/:userId
func (h *Handler) RemoveStoryContributor(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	contributorID := c.Param("userId")
	if contributorID == "" {
		InvalidParams(c, "user id is required")
		return
	}

	err := h.svc.RemoveStoryContributor(c.Request.Context(), userID, storyID, contributorID)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "contributor not found" {
			NotFound(c, "contributor not found")
			return
		}
		if err.Error() == "permission denied: only author can remove contributors" {
			Forbidden(c, "only story author can remove contributors")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "Contributor removed successfully"})
}

// GetStoryContributors 获取故事贡献者列表
// GET /api/stories/:id/contributors
func (h *Handler) GetStoryContributors(c *gin.Context) {
	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	contributors, err := h.svc.GetStoryContributors(c.Request.Context(), storyID, limit, offset)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"contributors": contributors,
		"count":        len(contributors),
	})
}

// ========== Story Scene Handlers ==========

// CreateStoryScene 创建故事场景
// POST /api/stories/:id/scenes
func (h *Handler) CreateStoryScene(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	var req service.CreateStorySceneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	req.StoryID = storyID

	scene, err := h.svc.CreateStoryScene(c.Request.Context(), userID, req)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "permission denied: insufficient rights" {
			Forbidden(c, "you don't have permission to create scenes for this story")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, scene)
}

// ListStoryScenes 获取故事场景列表
// GET /api/stories/:id/scenes
func (h *Handler) ListStoryScenes(c *gin.Context) {
	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	scenes, err := h.svc.ListStoryScenes(c.Request.Context(), storyID, limit, offset)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"scenes": scenes,
		"count":  len(scenes),
	})
}

// UpdateStoryScene 更新故事场景
// PUT /api/stories/:id/scenes/:sceneId
func (h *Handler) UpdateStoryScene(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	sceneID := c.Param("sceneId")
	if sceneID == "" {
		InvalidParams(c, "scene id is required")
		return
	}

	var req service.UpdateStorySceneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	scene, err := h.svc.UpdateStoryScene(c.Request.Context(), userID, storyID, sceneID, req)
	if err != nil {
		if err.Error() == "scene not found" {
			NotFound(c, "scene not found")
			return
		}
		if err.Error() == "permission denied: insufficient rights" {
			Forbidden(c, "you don't have permission to update this scene")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, scene)
}

// DeleteStoryScene 删除故事场景
// DELETE /api/stories/:id/scenes/:sceneId
func (h *Handler) DeleteStoryScene(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	sceneID := c.Param("sceneId")
	if sceneID == "" {
		InvalidParams(c, "scene id is required")
		return
	}

	err := h.svc.DeleteStoryScene(c.Request.Context(), userID, storyID, sceneID)
	if err != nil {
		if err.Error() == "permission denied: insufficient rights" {
			Forbidden(c, "you don't have permission to delete this scene")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "scene deleted successfully"})
}

// UploadSceneImage 接收场景图片OSS URL
// POST /api/stories/:id/scenes/register-image
func (h *Handler) UploadSceneImage(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	var req struct {
		ImageURL string `json:"imageUrl" binding:"required,url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// Verify story ownership
	if err := h.svc.EnsureStoryOwnership(c.Request.Context(), storyID, userID); err != nil {
		Forbidden(c, err.Error())
		return
	}

	// Return the OSS URL to frontend
	Success(c, gin.H{
		"success": true,
		"url":     req.ImageURL,
	})
}

// GenerateSceneImage AI生成场景图片
// POST /api/stories/:id/scenes/ai-generate-image
func (h *Handler) GenerateSceneImage(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	// Get sceneId and custom prompt from request (JSON body is optional)
	var req struct {
		SceneID *string `json:"sceneId"`
		Prompt  *string `json:"prompt"`
	}
	// Try to bind JSON, but don't fail if body is empty
	c.ShouldBindJSON(&req)

	// Also check query parameter for sceneId (for backward compatibility)
	sceneID := ""
	if req.SceneID != nil && *req.SceneID != "" {
		sceneID = *req.SceneID
	} else if querySceneID := c.Query("sceneId"); querySceneID != "" {
		sceneID = querySceneID
	}

	customPrompt := ""
	if req.Prompt != nil && *req.Prompt != "" {
		customPrompt = *req.Prompt
	}

	// Call service method to generate image
	imageURL, filename, err := h.svc.GenerateStorySceneImage(c.Request.Context(), storyID, sceneID, userID, customPrompt)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "scene not found" {
			NotFound(c, "scene not found")
			return
		}
		if err.Error() == "permission denied: insufficient rights" {
			Forbidden(c, "you don't have permission to generate images for this story")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"success":  true,
		"url":      imageURL,
		"filename": filename,
	})
}

// parseInt helper function for parsing integers
func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, domain.ErrNotFound
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}

// ========== Story Panel Handlers ==========

// ListStoryPanels 列出故事面板
// GET /api/v1/stories/:id/panels
func (h *Handler) ListStoryPanels(c *gin.Context) {
	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	panels, total, err := h.svc.ListStoryPanels(c.Request.Context(), storyID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"panels": panels,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateStoryPanel 创建故事面板
// POST /api/v1/stories/:id/panels
func (h *Handler) CreateStoryPanel(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	var req service.CreatePanelRequest
	if !BindJSON(c, &req) {
		return
	}
	req.StoryID = storyID

	panel, err := h.svc.CreateStoryPanel(c.Request.Context(), userID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, panel)
}

// UpdateStoryPanel 更新故事面板
// PUT /api/v1/stories/:id/panels/:panelId
func (h *Handler) UpdateStoryPanel(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	panelID := c.Param("panelId")
	if panelID == "" {
		InvalidParams(c, "panel id is required")
		return
	}

	var req service.UpdatePanelRequest
	if !BindJSON(c, &req) {
		return
	}

	panel, err := h.svc.UpdateStoryPanel(c.Request.Context(), userID, storyID, panelID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, panel)
}

// DeleteStoryPanel 删除故事面板
// DELETE /api/v1/stories/:id/panels/:panelId
func (h *Handler) DeleteStoryPanel(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	panelID := c.Param("panelId")
	if panelID == "" {
		InvalidParams(c, "panel id is required")
		return
	}

	err := h.svc.DeleteStoryPanel(c.Request.Context(), userID, storyID, panelID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "panel deleted successfully"})
}

// ReorderStoryPanels 重排故事面板
// POST /api/v1/stories/:id/panels/reorder
func (h *Handler) ReorderStoryPanels(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	var req service.ReorderPanelsRequest
	if !BindJSON(c, &req) {
		return
	}

	err := h.svc.ReorderStoryPanels(c.Request.Context(), userID, storyID, req.PanelIDs)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "panels reordered successfully"})
}

// ========== Story Comment Handlers (Enhanced) ==========

// ListStoryComments 列出故事评论及回复
// GET /api/v1/stories/:id/comments
func (h *Handler) ListStoryComments(c *gin.Context) {
	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	userID := authPkg.GetUserID(c)

	limit := 20
	offset := 0
	sortBy := c.DefaultQuery("sort", "newest") // newest, hottest

	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	comments, total, err := h.svc.ListStoryComments(c.Request.Context(), storyID, userID, limit, offset, sortBy)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"comments": comments,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// CreateStoryComment 创建故事评论
// POST /api/v1/stories/:id/comments
func (h *Handler) CreateStoryComment(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID := c.Param("id")
	if storyID == "" {
		InvalidParams(c, "story id is required")
		return
	}

	var req service.CreateStoryCommentRequest
	if !BindJSON(c, &req) {
		return
	}
	req.StoryID = storyID

	comment, err := h.svc.CreateStoryComment(c.Request.Context(), userID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, comment)
}

// CreateCommentReply 创建评论回复
// POST /api/v1/comments/:id/replies
func (h *Handler) CreateCommentReply(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	commentID := c.Param("id")
	if commentID == "" {
		InvalidParams(c, "comment id is required")
		return
	}

	var req service.CreateReplyRequest
	if !BindJSON(c, &req) {
		return
	}
	req.CommentID = commentID

	reply, err := h.svc.CreateCommentReply(c.Request.Context(), userID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, reply)
}
