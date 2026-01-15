package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// GetTrendingStoriesPublic returns top trending stories in the last 24 hours (guest accessible).
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
