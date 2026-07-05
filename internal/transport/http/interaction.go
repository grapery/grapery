package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// InteractionHandler 互动处理器
type InteractionHandler struct {
	interactionService service.InteractionService
}

// NewInteractionHandler 创建互动处理器
func NewInteractionHandler(interactionService service.InteractionService) *InteractionHandler {
	return &InteractionHandler{interactionService: interactionService}
}

// RegisterInteractionRoutes 注册互动相关路由
func (h *InteractionHandler) RegisterInteractionRoutes(r *gin.RouterGroup) {
	// 关注相关
	follows := r.Group("/follows")
	{
		follows.POST("", h.Follow)
		follows.DELETE("", h.Unfollow)
		follows.GET("/check", h.CheckFollowStatus)
		follows.GET("/followers/:type/:id", h.GetFollowers)
		follows.GET("/following/:userId", h.GetFollowing)
		follows.GET("/count/:type/:id", h.GetFollowersCount)
		follows.POST("/batch-check", h.BatchCheckFollowStatus)
	}

	// 点赞相关
	likes := r.Group("/likes")
	{
		likes.POST("", h.Like)
		likes.DELETE("", h.Unlike)
		likes.GET("/check", h.CheckLikeStatus)
		likes.GET("/:type/:id", h.GetLikes)
		likes.GET("/count/:type/:id", h.GetLikesCount)
		likes.POST("/batch-check", h.BatchCheckLikeStatus)
	}

	// 收藏/保存相关 (Bookmark - StoryCreationAppUI)
	bookmarks := r.Group("/bookmarks")
	{
		bookmarks.POST("", h.CreateBookmark)
		bookmarks.DELETE("/:id", h.DeleteBookmark)
		bookmarks.GET("/check", h.CheckBookmarkStatus)
		bookmarks.GET("/my", h.GetMyBookmarks)
		bookmarks.GET("/users/:userId", h.GetUserBookmarks)
		bookmarks.GET("/count/:type/:id", h.GetBookmarksCount)
	}
}

// FollowRequest 关注请求
type FollowRequest struct {
	FollowableType string `json:"followableType" binding:"required"` // story, user, group, character
	FollowableID   string `json:"followableId" binding:"required"`
}

// Follow 关注
func (h *InteractionHandler) Follow(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req FollowRequest
	if !BindJSON(c, &req) {
		return
	}

	followableType := domain.FollowableType(req.FollowableType)
	if err := h.interactionService.Follow(c.Request.Context(), userID, followableType, req.FollowableID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "followed successfully"})
}

// Unfollow 取消关注
func (h *InteractionHandler) Unfollow(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req FollowRequest
	if !BindJSON(c, &req) {
		return
	}

	followableType := domain.FollowableType(req.FollowableType)
	if err := h.interactionService.Unfollow(c.Request.Context(), userID, followableType, req.FollowableID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "unfollowed successfully"})
}

// CheckFollowStatusRequest 检查关注状态请求
type CheckFollowStatusRequest struct {
	FollowableType string `form:"type" binding:"required"`
	FollowableID   string `form:"id" binding:"required"`
}

// CheckFollowStatus 检查关注状态
func (h *InteractionHandler) CheckFollowStatus(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req CheckFollowStatusRequest
	if !BindQuery(c, &req) {
		return
	}

	followableType := domain.FollowableType(req.FollowableType)
	isFollowing, err := h.interactionService.CheckFollowStatus(c.Request.Context(), userID, followableType, req.FollowableID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"isFollowing": isFollowing})
}

// GetFollowers 获取关注者列表
func (h *InteractionHandler) GetFollowers(c *gin.Context) {
	followableType := domain.FollowableType(c.Param("type"))
	followableID := c.Param("id")
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	users, total, err := h.interactionService.GetFollowers(c.Request.Context(), followableType, followableID, page, pageSize)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessPaginated(c, users, int64(total), page, pageSize)
}

// GetFollowing 获取关注列表
func (h *InteractionHandler) GetFollowing(c *gin.Context) {
	userID := c.Param("userId")
	followableType := domain.FollowableType(c.Query("type"))
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	follows, total, err := h.interactionService.GetFollowing(c.Request.Context(), userID, followableType, page, pageSize)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessPaginated(c, follows, int64(total), page, pageSize)
}

// GetFollowersCount 获取关注者数量
func (h *InteractionHandler) GetFollowersCount(c *gin.Context) {
	followableType := domain.FollowableType(c.Param("type"))
	followableID := c.Param("id")

	count, err := h.interactionService.GetFollowersCount(c.Request.Context(), followableType, followableID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"count": count})
}

// BatchCheckFollowStatusRequest 批量检查关注状态请求
type BatchCheckFollowStatusRequest struct {
	FollowableType string   `json:"followableType" binding:"required"`
	FollowableIDs  []string `json:"followableIds" binding:"required"`
}

// BatchCheckFollowStatus 批量检查关注状态
func (h *InteractionHandler) BatchCheckFollowStatus(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req BatchCheckFollowStatusRequest
	if !BindJSON(c, &req) {
		return
	}

	followableType := domain.FollowableType(req.FollowableType)
	result, err := h.interactionService.BatchCheckFollowStatus(c.Request.Context(), userID, followableType, req.FollowableIDs)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, result)
}

// LikeRequest 点赞请求
type LikeRequest struct {
	LikeableType string `json:"likeableType" binding:"required"` // story, character, storyboard_node (canonical: storyboard_likes), fragment, character_poster
	LikeableID   string `json:"likeableId" binding:"required"`
}

// Like 点赞
func (h *InteractionHandler) Like(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req LikeRequest
	if !BindJSON(c, &req) {
		return
	}

	likeableType := domain.LikeableType(req.LikeableType)
	if err := h.interactionService.Like(c.Request.Context(), userID, likeableType, req.LikeableID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "liked successfully"})
}

// Unlike 取消点赞
func (h *InteractionHandler) Unlike(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req LikeRequest
	if !BindJSON(c, &req) {
		return
	}

	likeableType := domain.LikeableType(req.LikeableType)
	if err := h.interactionService.Unlike(c.Request.Context(), userID, likeableType, req.LikeableID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "unliked successfully"})
}

// CheckLikeStatusRequest 检查点赞状态请求
type CheckLikeStatusRequest struct {
	LikeableType string `form:"type" binding:"required"`
	LikeableID   string `form:"id" binding:"required"`
}

// CheckLikeStatus 检查点赞状态
func (h *InteractionHandler) CheckLikeStatus(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req CheckLikeStatusRequest
	if !BindQuery(c, &req) {
		return
	}

	likeableType := domain.LikeableType(req.LikeableType)
	isLiked, err := h.interactionService.CheckLikeStatus(c.Request.Context(), userID, likeableType, req.LikeableID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"isLiked": isLiked})
}

// GetLikes 获取点赞列表
func (h *InteractionHandler) GetLikes(c *gin.Context) {
	likeableType := domain.LikeableType(c.Param("type"))
	likeableID := c.Param("id")
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	users, total, err := h.interactionService.GetLikes(c.Request.Context(), likeableType, likeableID, page, pageSize)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessPaginated(c, users, int64(total), page, pageSize)
}

// GetLikesCount 获取点赞数量
func (h *InteractionHandler) GetLikesCount(c *gin.Context) {
	likeableType := domain.LikeableType(c.Param("type"))
	likeableID := c.Param("id")

	count, err := h.interactionService.GetLikesCount(c.Request.Context(), likeableType, likeableID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"count": count})
}

// BatchCheckLikeStatusRequest 批量检查点赞状态请求
type BatchCheckLikeStatusRequest struct {
	LikeableType string   `json:"likeableType" binding:"required"`
	LikeableIDs  []string `json:"likeableIds" binding:"required"`
}

// BatchCheckLikeStatus 批量检查点赞状态
func (h *InteractionHandler) BatchCheckLikeStatus(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req BatchCheckLikeStatusRequest
	if !BindJSON(c, &req) {
		return
	}

	likeableType := domain.LikeableType(req.LikeableType)
	result, err := h.interactionService.BatchCheckLikeStatus(c.Request.Context(), userID, likeableType, req.LikeableIDs)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, result)
}

// ========== Bookmark Handlers (StoryCreationAppUI) ==========

// BookmarkRequest 收藏请求
type BookmarkRequest struct {
	BookmarkType   string `json:"bookmarkType" binding:"required"` // story, fragment, storyboard
	BookmarkID     string `json:"bookmarkId" binding:"required"`
	CollectionName string `json:"collectionName,omitempty"`
}

// CreateBookmark 创建收藏
func (h *InteractionHandler) CreateBookmark(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req BookmarkRequest
	if !BindJSON(c, &req) {
		return
	}

	bookmarkType := domain.BookmarkType(req.BookmarkType)
	bookmark, err := h.interactionService.CreateBookmark(c.Request.Context(), userID, bookmarkType, req.BookmarkID, req.CollectionName)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, bookmark)
}

// DeleteBookmark 删除收藏
func (h *InteractionHandler) DeleteBookmark(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	bookmarkID := c.Param("id")
	if err := h.interactionService.DeleteBookmark(c.Request.Context(), userID, bookmarkID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "bookmark removed successfully"})
}

// CheckBookmarkStatusRequest 检查收藏状态请求
type CheckBookmarkStatusRequest struct {
	BookmarkType string `json:"bookmarkType" form:"bookmarkType" binding:"required"`
	BookmarkID   string `json:"bookmarkId" form:"bookmarkId" binding:"required"`
}

// CheckBookmarkStatus 检查收藏状态
func (h *InteractionHandler) CheckBookmarkStatus(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req CheckBookmarkStatusRequest
	if !BindQuery(c, &req) {
		return
	}

	bookmarkType := domain.BookmarkType(req.BookmarkType)
	isBookmarked, err := h.interactionService.CheckBookmarkStatus(c.Request.Context(), userID, bookmarkType, req.BookmarkID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"isBookmarked": isBookmarked})
}

// GetMyBookmarks 获取当前用户的收藏列表
func (h *InteractionHandler) GetMyBookmarks(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	bookmarkType := domain.BookmarkType(c.Query("type")) // Optional filter
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	bookmarks, total, hasMore, err := h.interactionService.GetBookmarksByUserPaged(c.Request.Context(), userID, bookmarkType, page, limit)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"bookmarks": bookmarks,
		"total":     total,
		"page":      page,
		"limit":     limit,
		"hasMore":   hasMore,
	})
}

// GetUserBookmarks 获取指定用户主页可见的收藏列表
func (h *InteractionHandler) GetUserBookmarks(c *gin.Context) {
	viewerID, ok := RequireUserID(c)
	if !ok {
		return
	}
	ownerID := c.Param("userId")
	bookmarkType := domain.BookmarkType(c.Query("type")) // Optional filter
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	bookmarks, total, hasMore, err := h.interactionService.GetUserBookmarksPaged(c.Request.Context(), ownerID, viewerID, bookmarkType, page, limit)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"bookmarks": bookmarks,
		"total":     total,
		"page":      page,
		"limit":     limit,
		"hasMore":   hasMore,
	})
}

// GetBookmarksCount 获取收藏数量
func (h *InteractionHandler) GetBookmarksCount(c *gin.Context) {
	bookmarkType := domain.BookmarkType(c.Param("type"))
	bookmarkID := c.Param("id")

	count, err := h.interactionService.GetBookmarksCount(c.Request.Context(), bookmarkType, bookmarkID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"count": count, "saves": count}) // Both for API compatibility
}
