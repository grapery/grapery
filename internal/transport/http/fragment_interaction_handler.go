package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

type FragmentInteractionHandler struct {
	interactionRepo *repository.FragmentInteractionRepository
	fragmentRepo    *repository.FragmentRepository
	svc             *service.Service
	logger          *zap.Logger
}

func NewFragmentInteractionHandler(
	interactionRepo *repository.FragmentInteractionRepository,
	fragmentRepo *repository.FragmentRepository,
	svc *service.Service,
	logger *zap.Logger,
) *FragmentInteractionHandler {
	return &FragmentInteractionHandler{
		interactionRepo: interactionRepo,
		fragmentRepo:    fragmentRepo,
		svc:             svc,
		logger:          logger,
	}
}

// ============= 点赞相关 API =============

// LikeFragment 点赞碎片
// POST /api/fragments/:id/like
func (h *FragmentInteractionHandler) LikeFragment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fragmentID := c.Param("id")
	if fragmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment id is required"})
		return
	}

	// 检查碎片是否存在
	fragment, err := h.fragmentRepo.GetByID(c.Request.Context(), fragmentID)
	if err != nil || fragment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fragment not found"})
		return
	}

	// 检查是否已经点赞
	isLiked, err := h.interactionRepo.IsLiked(c.Request.Context(), fragmentID, userID)
	if err != nil {
		h.logger.Error("Failed to check like status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check like status"})
		return
	}

	if isLiked {
		// 幂等：已点赞时直接返回当前状态，避免前端因状态轻微不同步产生 409
		stats, _ := h.interactionRepo.GetFragmentStats(c.Request.Context(), fragmentID, userID)
		c.JSON(http.StatusOK, gin.H{
			"message": "already liked",
			"stats":   stats,
		})
		return
	}

	// 创建点赞记录
	like := &domain.FragmentLike{
		ID:         uuid.New().String(),
		FragmentID: fragmentID,
		UserID:     userID,
	}

	if err := h.interactionRepo.CreateLike(c.Request.Context(), like); err != nil {
		h.logger.Error("Failed to create like", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like fragment"})
		return
	}

	// 获取更新后的统计信息
	stats, _ := h.interactionRepo.GetFragmentStats(c.Request.Context(), fragmentID, userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "liked",
		"stats":   stats,
	})
}

// UnlikeFragment 取消点赞
// DELETE /api/fragments/:id/like
func (h *FragmentInteractionHandler) UnlikeFragment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fragmentID := c.Param("id")
	if fragmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment id is required"})
		return
	}

	// 删除点赞记录
	if err := h.interactionRepo.DeleteLike(c.Request.Context(), fragmentID, userID); err != nil {
		h.logger.Error("Failed to delete like", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlike fragment"})
		return
	}

	// 获取更新后的统计信息
	stats, _ := h.interactionRepo.GetFragmentStats(c.Request.Context(), fragmentID, userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "unliked",
		"stats":   stats,
	})
}

// GetFragmentLikes 获取碎片的点赞列表
// GET /api/fragments/:id/likes?page=1&limit=20
func (h *FragmentInteractionHandler) GetFragmentLikes(c *gin.Context) {
	fragmentID := c.Param("id")
	if fragmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment id is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	likes, total, err := h.interactionRepo.GetFragmentLikes(c.Request.Context(), fragmentID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get fragment likes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get likes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"likes": likes,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// ============= 评论相关 API =============

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	Content  string  `json:"content" binding:"required,min=1,max=1000"`
	ParentID *string `json:"parentId,omitempty"` // 如果提供，则是回复评论
}

// CreateComment 创建评论（统一 comments 表，与 POST /api/v1/comments 行为一致）
// POST /api/fragments/:id/comments
func (h *FragmentInteractionHandler) CreateComment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fragmentID := c.Param("id")
	if fragmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment id is required"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := &domain.Comment{
		UserID:     userID,
		Content:    req.Content,
		TargetType: "fragment",
		TargetID:   fragmentID,
		ParentID:   "",
	}
	if req.ParentID != nil && *req.ParentID != "" {
		comment.ParentID = *req.ParentID
	}

	if err := h.svc.CreateComment(c.Request.Context(), comment); err != nil {
		h.logger.Error("Failed to create comment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回完整作者信息（与 v1 CreateComment 一致）
	full, err := h.svc.GetComment(c.Request.Context(), comment.ID)
	if err == nil && full != nil {
		comment = full
	}

	c.Status(http.StatusCreated)
	Success(c, comment)
}

// GetFragmentComments 获取碎片的评论列表（统一 /api/v1/comments?targetType=fragment）
// GET /api/fragments/:id/comments?offset&limit&page&sort
func (h *FragmentInteractionHandler) GetFragmentComments(c *gin.Context) {
	userID := c.GetString("userID")
	fragmentID := c.Param("id")
	if fragmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment id is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := 0
	if o := c.Query("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	} else {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}
		if limit <= 0 {
			limit = 20
		}
		offset = (page - 1) * limit
	}
	sort := c.DefaultQuery("sort", "")

	comments, total, err := h.svc.ListComments(c.Request.Context(), "fragment", fragmentID, limit, offset, userID, sort)
	if err != nil {
		h.logger.Error("Failed to get fragment comments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get comments"})
		return
	}

	Success(c, gin.H{
		"comments": comments,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"sort":     sort,
	})
}

// GetCommentReplies 获取评论的回复列表（与 GET /api/v1/comments/:id/replies 一致，统一 comments 表）
// GET /api/fragments/comments/:id/replies?offset&limit&page
func (h *FragmentInteractionHandler) GetCommentReplies(c *gin.Context) {
	parentID := c.Param("id")
	if parentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment id is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := 0
	if o := c.Query("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	} else {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}
		if limit <= 0 {
			limit = 20
		}
		offset = (page - 1) * limit
	}
	userID := c.GetString("userID")

	replies, total, err := h.svc.GetCommentReplies(c.Request.Context(), parentID, limit, offset, userID)
	if err != nil {
		h.logger.Error("Failed to get comment replies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get replies"})
		return
	}

	Success(c, gin.H{
		"replies": replies,
		"total":   total,
		"count":   len(replies),
		"limit":   limit,
		"offset":  offset,
	})
}

// UpdateCommentRequest 更新评论请求
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=1000"`
}

// UpdateComment 更新评论
// PUT /api/fragments/comments/:id
func (h *FragmentInteractionHandler) UpdateComment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	commentID := c.Param("id")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment id is required"})
		return
	}

	existing, err := h.svc.GetComment(c.Request.Context(), commentID)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}
	if existing.TargetType != "fragment" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a fragment comment"})
		return
	}

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := &domain.Comment{
		BaseModel: common.BaseModel{ID: commentID},
		Content:   req.Content,
	}
	if err := h.svc.UpdateComment(c.Request.Context(), comment, userID); err != nil {
		h.logger.Error("Failed to update comment", zap.Error(err))
		if err.Error() == "permission denied: not the author" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, _ := h.svc.GetComment(c.Request.Context(), commentID)
	if updated != nil {
		Success(c, updated)
		return
	}
	Success(c, gin.H{"message": "comment updated successfully"})
}

// DeleteComment 删除评论
// DELETE /api/fragments/comments/:id
func (h *FragmentInteractionHandler) DeleteComment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	commentID := c.Param("id")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment id is required"})
		return
	}

	comment, err := h.svc.GetComment(c.Request.Context(), commentID)
	if err != nil || comment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}
	if comment.TargetType != "fragment" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a fragment comment"})
		return
	}

	if err := h.svc.DeleteComment(c.Request.Context(), commentID, userID); err != nil {
		h.logger.Error("Failed to delete comment", zap.Error(err))
		if err.Error() == "permission denied: not the author" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	Success(c, gin.H{"message": "comment deleted successfully"})
}

// ============= 分享相关 API =============

// ShareFragmentRequest 分享请求
type ShareFragmentRequest struct {
	Platform string `json:"platform" binding:"required,oneof=wechat twitter local"` // wechat, twitter, local
}

// ShareFragment 记录分享行为
// POST /api/fragments/:id/share
func (h *FragmentInteractionHandler) ShareFragment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fragmentID := c.Param("id")
	if fragmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment id is required"})
		return
	}

	var req ShareFragmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查碎片是否存在
	fragment, err := h.fragmentRepo.GetByID(c.Request.Context(), fragmentID)
	if err != nil || fragment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fragment not found"})
		return
	}

	// 创建分享记录
	share := &domain.FragmentShare{
		ID:         uuid.New().String(),
		FragmentID: fragmentID,
		UserID:     userID,
		Platform:   req.Platform,
	}

	if err := h.interactionRepo.CreateShare(c.Request.Context(), share); err != nil {
		h.logger.Error("Failed to create share record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record share"})
		return
	}

	// 获取更新后的统计信息
	stats, _ := h.interactionRepo.GetFragmentStats(c.Request.Context(), fragmentID, userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "share recorded",
		"stats":   stats,
	})
}

// ============= 统计相关 API =============

// GetFragmentStats 获取碎片统计信息
// GET /api/fragments/:id/stats
func (h *FragmentInteractionHandler) GetFragmentStats(c *gin.Context) {
	userID := c.GetString("userID")
	fragmentID := c.Param("id")
	if fragmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment id is required"})
		return
	}

	stats, err := h.interactionRepo.GetFragmentStats(c.Request.Context(), fragmentID, userID)
	if err != nil {
		h.logger.Error("Failed to get fragment stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// RegisterRoutes 注册所有路由
func (h *FragmentInteractionHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// 点赞相关
	router.POST("/:id/like", authMiddleware, h.LikeFragment)
	router.DELETE("/:id/like", authMiddleware, h.UnlikeFragment)
	router.GET("/:id/likes", h.GetFragmentLikes) // 获取点赞列表不需要认证

	// 评论相关
	router.POST("/:id/comments", authMiddleware, h.CreateComment)
	router.GET("/:id/comments", h.GetFragmentComments) // 获取评论列表不需要认证
	router.GET("/comments/:id/replies", h.GetCommentReplies)
	router.PUT("/comments/:id", authMiddleware, h.UpdateComment)
	router.DELETE("/comments/:id", authMiddleware, h.DeleteComment)

	// 分享相关
	router.POST("/:id/share", authMiddleware, h.ShareFragment)

	// 统计相关
	router.GET("/:id/stats", h.GetFragmentStats) // 获取统计信息不需要认证
}
