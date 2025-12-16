package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// CreateComment 创建评论
func (h *Handler) CreateComment(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req struct {
		TargetType string  `json:"targetType" binding:"required"` // story, storyboard, character
		TargetID   string  `json:"targetId" binding:"required"`
		Content    string  `json:"content" binding:"required"`
		ParentID   *string `json:"parentId"` // 回复评论时提供
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	comment := &domain.Comment{
		AuthorID:   userID.(string),
		Content:    req.Content,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		ParentID:   "",
	}
	if req.ParentID != nil {
		comment.ParentID = *req.ParentID
	}

	if err := h.svc.CreateComment(c.Request.Context(), comment); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, comment)
}

// GetComment 获取评论详情
func (h *Handler) GetComment(c *gin.Context) {
	id := c.Param("id")

	comment, err := h.svc.GetComment(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			NotFound(c, "comment not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, comment)
}

// UpdateComment 更新评论
func (h *Handler) UpdateComment(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	comment := &domain.Comment{
		ID:      id,
		Content: req.Content,
	}

	if err := h.svc.UpdateComment(c.Request.Context(), comment, userID.(string)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "comment updated successfully"})
}

// DeleteComment 删除评论
func (h *Handler) DeleteComment(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	if err := h.svc.DeleteComment(c.Request.Context(), id, userID.(string)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "comment deleted successfully"})
}

// ListComments 获取目标的评论列表
func (h *Handler) ListComments(c *gin.Context) {
	// Support both camelCase and snake_case for compatibility
	targetType := c.Query("targetType")
	if targetType == "" {
		targetType = c.Query("target_type")
	}

	targetID := c.Query("targetId")
	if targetID == "" {
		targetID = c.Query("target_id")
	}

	if targetType == "" || targetID == "" {
		InvalidParams(c, "targetType and targetId are required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 获取可选的用户ID以填充点赞状态
	userID := ""
	if uid, exists := c.Get("userID"); exists {
		userID = uid.(string)
	}

	comments, total, err := h.svc.ListComments(c.Request.Context(), targetType, targetID, limit, offset, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"comments": comments,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetCommentReplies 获取评论的回复
func (h *Handler) GetCommentReplies(c *gin.Context) {
	parentID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	replies, err := h.svc.GetCommentReplies(c.Request.Context(), parentID, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"replies": replies,
		"count":   len(replies),
		"limit":   limit,
		"offset":  offset,
	})
}

// GetCommentTree 获取完整的评论树
func (h *Handler) GetCommentTree(c *gin.Context) {
	rootID := c.Param("id")

	tree, err := h.svc.GetCommentTree(c.Request.Context(), rootID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"tree":  tree,
		"count": len(tree),
	})
}

// ToggleLikeComment 切换点赞状态 (点赞/取消点赞)
func (h *Handler) ToggleLikeComment(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	result, err := h.svc.ToggleLikeComment(c.Request.Context(), userID.(string), id)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// LikeComment 点赞评论
func (h *Handler) LikeComment(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	if err := h.svc.LikeComment(c.Request.Context(), userID.(string), id); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "comment liked successfully"})
}

// DislikeComment 踩评论
func (h *Handler) DislikeComment(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	if err := h.svc.DislikeComment(c.Request.Context(), userID.(string), id); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "comment disliked successfully"})
}

// UnlikeComment 取消点赞/踩
func (h *Handler) UnlikeComment(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	if err := h.svc.UnlikeComment(c.Request.Context(), userID.(string), id); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "comment unliked successfully"})
}
