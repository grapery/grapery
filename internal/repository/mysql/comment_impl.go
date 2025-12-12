package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CommentByID retrieves a comment by ID
func (r *Repository) CommentByID(ctx context.Context, id string) (*domain.Comment, error) {
	var comment Comment
	if err := r.db.WithContext(ctx).
		Preload("Author").
		Where("id = ?", id).
		First(&comment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	domainComment := r.commentToDomain(comment)
	return &domainComment, nil
}

// CreateComment creates a new comment
func (r *Repository) CreateComment(ctx context.Context, comment *domain.Comment) error {
	dbComment := Comment{
		ID:         uuid.New().String(),
		AuthorID:   comment.AuthorID,
		Content:    comment.Content,
		TargetType: comment.TargetType,
		TargetID:   comment.TargetID,
		ParentID:   nil, // Will be set below if needed
		RootID:     nil, // Will be set below if needed
		Likes:      0,
		Dislikes:   0,
		ReplyCount: 0,
	}

	// 如果是回复，设置 ParentID 和 RootID
	if comment.ParentID != "" {
		dbComment.ParentID = &comment.ParentID
		
		// 获取父评论
		var parent Comment
		if err := r.db.WithContext(ctx).Where("id = ?", comment.ParentID).First(&parent).Error; err != nil {
			return fmt.Errorf("parent comment not found: %w", err)
		}

		// 如果父评论有 RootID，使用父评论的 RootID，否则使用父评论的 ID 作为 RootID
		if parent.RootID != nil {
			dbComment.RootID = parent.RootID
		} else {
			dbComment.RootID = &comment.ParentID
		}

		// 更新父评论的回复计数
		if err := r.db.WithContext(ctx).
			Model(&Comment{}).
			Where("id = ?", comment.ParentID).
			UpdateColumn("reply_count", gorm.Expr("reply_count + ?", 1)).Error; err != nil {
			r.log.Warn("failed to update parent reply count", zap.Error(err))
		}
	}

	if err := r.db.WithContext(ctx).Create(&dbComment).Error; err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	// 更新目标对象的评论计数
	go r.updateTargetCommentCount(context.Background(), comment.TargetType, comment.TargetID, 1)

	comment.ID = dbComment.ID
	comment.CreatedAt = dbComment.CreatedAt.Unix()
	comment.UpdatedAt = dbComment.UpdatedAt.Unix()
	return nil
}

// UpdateComment updates an existing comment
func (r *Repository) UpdateComment(ctx context.Context, comment *domain.Comment) error {
	updates := map[string]interface{}{
		"content":    comment.Content,
		"updated_at": time.Now().UTC(),
	}

	if err := r.db.WithContext(ctx).
		Model(&Comment{}).
		Where("id = ?", comment.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

// DeleteComment deletes a comment
func (r *Repository) DeleteComment(ctx context.Context, id string) error {
	// 获取评论信息
	var comment Comment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&comment).Error; err != nil {
		return fmt.Errorf("comment not found: %w", err)
	}

	// 软删除
	if err := r.db.WithContext(ctx).Delete(&Comment{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	// 如果有父评论，更新父评论的回复计数
	if comment.ParentID != nil {
		if err := r.db.WithContext(ctx).
			Model(&Comment{}).
			Where("id = ?", *comment.ParentID).
			UpdateColumn("reply_count", gorm.Expr("reply_count - ?", 1)).Error; err != nil {
			r.log.Warn("failed to update parent reply count", zap.Error(err))
		}
	}

	// 更新目标对象的评论计数
	go r.updateTargetCommentCount(context.Background(), comment.TargetType, comment.TargetID, -1)

	return nil
}

// CommentsByTarget retrieves comments for a target
func (r *Repository) CommentsByTarget(ctx context.Context, targetType, targetID string, limit, offset int) ([]*domain.Comment, int64, error) {
	var comments []Comment
	var total int64

	// 获取总数 - 需要单独的 Model 查询
	if err := r.db.WithContext(ctx).
		Model(&Comment{}).
		Where("target_type = ? AND target_id = ? AND parent_id IS NULL", targetType, targetID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count comments: %w", err)
	}

	// 获取分页数据 - 只获取顶层评论（ParentID 为 null）
	query := r.db.WithContext(ctx).
		Preload("Author").
		Where("target_type = ? AND target_id = ? AND parent_id IS NULL", targetType, targetID)

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get comments: %w", err)
	}

	result := make([]*domain.Comment, len(comments))
	for i, c := range comments {
		domainComment := r.commentToDomain(c)
		result[i] = &domainComment
	}

	return result, total, nil
}

// CommentReplies retrieves replies to a comment
func (r *Repository) CommentReplies(ctx context.Context, parentID string, limit, offset int) ([]*domain.Comment, error) {
	var comments []Comment
	query := r.db.WithContext(ctx).
		Preload("Author").
		Where("parent_id = ?", parentID).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}

	result := make([]*domain.Comment, len(comments))
	for i, c := range comments {
		domainComment := r.commentToDomain(c)
		result[i] = &domainComment
	}

	return result, nil
}

// CommentTree retrieves the entire comment tree
func (r *Repository) CommentTree(ctx context.Context, rootID string) ([]*domain.Comment, error) {
	var comments []Comment
	if err := r.db.WithContext(ctx).
		Preload("Author").
		Where("root_id = ? OR id = ?", rootID, rootID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("failed to get comment tree: %w", err)
	}

	result := make([]*domain.Comment, len(comments))
	for i, c := range comments {
		domainComment := r.commentToDomain(c)
		result[i] = &domainComment
	}

	return result, nil
}

// LikeComment likes or dislikes a comment
func (r *Repository) LikeComment(ctx context.Context, userID, commentID string, isLike bool) error {
	// 检查是否已经点赞/踩过
	var existing CommentLike
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		First(&existing).Error

	if err == nil {
		// 已存在，更新
		if existing.IsLike != isLike {
			// 切换点赞/踩
			if err := r.db.WithContext(ctx).
				Model(&CommentLike{}).
				Where("id = ?", existing.ID).
				Update("is_like", isLike).Error; err != nil {
				return fmt.Errorf("failed to update like: %w", err)
			}

			// 更新评论的点赞/踩计数
			if isLike {
				r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", commentID).
					UpdateColumn("likes", gorm.Expr("likes + ?", 1))
				r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", commentID).
					UpdateColumn("dislikes", gorm.Expr("dislikes - ?", 1))
			} else {
				r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", commentID).
					UpdateColumn("likes", gorm.Expr("likes - ?", 1))
				r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", commentID).
					UpdateColumn("dislikes", gorm.Expr("dislikes + ?", 1))
			}
		}
		return nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing like: %w", err)
	}

	// 不存在，创建新的
	like := CommentLike{
		ID:        uuid.New().String(),
		UserID:    userID,
		CommentID: commentID,
		IsLike:    isLike,
	}

	if err := r.db.WithContext(ctx).Create(&like).Error; err != nil {
		return fmt.Errorf("failed to create like: %w", err)
	}

	// 更新评论的点赞/踩计数
	if isLike {
		r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", commentID).
			UpdateColumn("likes", gorm.Expr("likes + ?", 1))
	} else {
		r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", commentID).
			UpdateColumn("dislikes", gorm.Expr("dislikes + ?", 1))
	}

	return nil
}

// UnlikeComment removes a like/dislike
func (r *Repository) UnlikeComment(ctx context.Context, userID, commentID string) error {
	var like CommentLike
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		First(&like).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // 已经没有点赞/踩了
		}
		return fmt.Errorf("failed to find like: %w", err)
	}

	// 删除点赞/踩记录
	if err := r.db.WithContext(ctx).Delete(&like).Error; err != nil {
		return fmt.Errorf("failed to delete like: %w", err)
	}

	// 更新评论的点赞/踩计数
	if like.IsLike {
		r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", commentID).
			UpdateColumn("likes", gorm.Expr("likes - ?", 1))
	} else {
		r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", commentID).
			UpdateColumn("dislikes", gorm.Expr("dislikes - ?", 1))
	}

	return nil
}

// IsCommentLiked checks if a user has liked/disliked a comment
func (r *Repository) IsCommentLiked(ctx context.Context, userID, commentID string) (bool, bool, error) {
	var like CommentLike
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		First(&like).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, nil // 没有点赞/踩
		}
		return false, false, fmt.Errorf("failed to check like: %w", err)
	}

	return true, like.IsLike, nil
}

// commentToDomain converts database model to domain model
func (r *Repository) commentToDomain(c Comment) domain.Comment {
	refAuthor := r.userToDomain(c.Author)
	
	// Handle nullable pointer fields
	var parentID, rootID string
	if c.ParentID != nil {
		parentID = *c.ParentID
	}
	if c.RootID != nil {
		rootID = *c.RootID
	}
	
	return domain.Comment{
		ID:         c.ID,
		AuthorID:   c.AuthorID,
		Author:     &refAuthor,
		Content:    c.Content,
		TargetType: c.TargetType,
		TargetID:   c.TargetID,
		ParentID:   parentID,
		RootID:     rootID,
		Likes:      c.Likes,
		Dislikes:   c.Dislikes,
		ReplyCount: c.ReplyCount,
		Replies:    nil, // 不自动加载回复，需要单独查询
		CreatedAt:  c.CreatedAt.Unix(),
		UpdatedAt:  c.UpdatedAt.Unix(),
	}
}

// updateTargetCommentCount 更新目标对象的评论计数
func (r *Repository) updateTargetCommentCount(ctx context.Context, targetType, targetID string, delta int) {
	switch targetType {
	case "story":
		// 更新故事的评论计数（如果有这个字段）
		// r.db.WithContext(ctx).Model(&Story{}).Where("id = ?", targetID).
		// 	UpdateColumn("comments", gorm.Expr("comments + ?", delta))
	case "storyboard":
		r.db.WithContext(ctx).Model(&Storyboard{}).Where("id = ?", targetID).
			UpdateColumn("comments", gorm.Expr("comments + ?", delta))
	case "character":
		// 更新角色的评论计数（如果有这个字段）
	}
}
