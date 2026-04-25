package repository

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// FragmentInteractionRepository 碎片交互Repository（点赞、评论、分享）
type FragmentInteractionRepository struct {
	db *gorm.DB
}

func NewFragmentInteractionRepository(db *gorm.DB) *FragmentInteractionRepository {
	return &FragmentInteractionRepository{db: db}
}

// ============= 点赞相关 =============

// CreateLike 创建点赞
func (r *FragmentInteractionRepository) CreateLike(ctx context.Context, like *domain.FragmentLike) error {
	return r.db.WithContext(ctx).Create(like).Error
}

// DeleteLike 删除点赞
func (r *FragmentInteractionRepository) DeleteLike(ctx context.Context, fragmentID, userID string) error {
	return r.db.WithContext(ctx).
		Where("fragment_id = ? AND user_id = ?", fragmentID, userID).
		Delete(&domain.FragmentLike{}).Error
}

// GetLike 获取点赞记录
func (r *FragmentInteractionRepository) GetLike(ctx context.Context, fragmentID, userID string) (*domain.FragmentLike, error) {
	var like domain.FragmentLike
	err := r.db.WithContext(ctx).
		Where("fragment_id = ? AND user_id = ?", fragmentID, userID).
		First(&like).Error
	if err != nil {
		return nil, err
	}
	return &like, nil
}

// IsLiked 检查用户是否已点赞
func (r *FragmentInteractionRepository) IsLiked(ctx context.Context, fragmentID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.FragmentLike{}).
		Where("fragment_id = ? AND user_id = ?", fragmentID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetLikesCount 获取点赞数
func (r *FragmentInteractionRepository) GetLikesCount(ctx context.Context, fragmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.FragmentLike{}).
		Where("fragment_id = ?", fragmentID).
		Count(&count).Error
	return count, err
}

// GetFragmentLikes 获取碎片的点赞列表（分页）
func (r *FragmentInteractionRepository) GetFragmentLikes(ctx context.Context, fragmentID string, limit, offset int) ([]*domain.FragmentLike, int64, error) {
	var likes []*domain.FragmentLike
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.FragmentLike{}).Where("fragment_id = ?", fragmentID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&likes).Error

	if err != nil {
		return nil, 0, err
	}

	return likes, total, nil
}

// ============= 评论相关 =============

// CreateComment 创建评论
func (r *FragmentInteractionRepository) CreateComment(ctx context.Context, comment *domain.FragmentComment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// GetCommentByID 获取评论详情
func (r *FragmentInteractionRepository) GetCommentByID(ctx context.Context, id string) (*domain.FragmentComment, error) {
	var comment domain.FragmentComment
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("id = ?", id).
		First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// UpdateComment 更新评论
func (r *FragmentInteractionRepository) UpdateComment(ctx context.Context, comment *domain.FragmentComment) error {
	return r.db.WithContext(ctx).Save(comment).Error
}

// DeleteComment 删除评论
func (r *FragmentInteractionRepository) DeleteComment(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.FragmentComment{}, "id = ?", id).Error
}

// GetFragmentComments 获取碎片的评论列表（支持分页和回复）
func (r *FragmentInteractionRepository) GetFragmentComments(ctx context.Context, fragmentID, userID string, limit, offset int) ([]*domain.FragmentComment, int64, error) {
	var comments []*domain.FragmentComment
	var total int64

	// 只查询顶级评论（没有父评论的）
	query := r.db.WithContext(ctx).
		Model(&domain.FragmentComment{}).
		Where("fragment_id = ? AND parent_id IS NULL", fragmentID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 查询评论并预加载用户信息
	err = query.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error

	if err != nil {
		return nil, 0, err
	}

	// 为每个评论加载回复数和用户是否点赞信息
	for _, comment := range comments {
		// 加载回复数
		var replyCount int64
		r.db.WithContext(ctx).
			Model(&domain.FragmentComment{}).
			Where("parent_id = ?", comment.ID).
			Count(&replyCount)
		comment.ReplyCount = int(replyCount)

		// TODO: 可以在这里加载每个评论的点赞信息和当前用户是否点赞
	}

	return comments, total, nil
}

// GetCommentReplies 获取评论的回复列表
func (r *FragmentInteractionRepository) GetCommentReplies(ctx context.Context, parentID string, limit, offset int) ([]*domain.FragmentComment, int64, error) {
	var replies []*domain.FragmentComment
	var total int64

	query := r.db.WithContext(ctx).
		Model(&domain.FragmentComment{}).
		Where("parent_id = ?", parentID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("User").
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&replies).Error

	if err != nil {
		return nil, 0, err
	}

	return replies, total, nil
}

// GetCommentsCount 获取评论数（统一 comments 表，含回复；与 CreateComment/updateTargetCommentCount 一致）
func (r *FragmentInteractionRepository) GetCommentsCount(ctx context.Context, fragmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM comments WHERE deleted_at IS NULL AND target_type = ? AND target_id = ?`,
		"fragment", fragmentID,
	).Scan(&count).Error
	return count, err
}

// ============= 分享相关 =============

// CreateShare 创建分享记录
func (r *FragmentInteractionRepository) CreateShare(ctx context.Context, share *domain.FragmentShare) error {
	return r.db.WithContext(ctx).Create(share).Error
}

// GetSharesCount 获取分享数
func (r *FragmentInteractionRepository) GetSharesCount(ctx context.Context, fragmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.FragmentShare{}).
		Where("fragment_id = ?", fragmentID).
		Count(&count).Error
	return count, err
}

// ============= 统计相关 =============

// GetFragmentStats 获取碎片统计信息
func (r *FragmentInteractionRepository) GetFragmentStats(ctx context.Context, fragmentID, userID string) (*domain.FragmentStats, error) {
	stats := &domain.FragmentStats{
		FragmentID: fragmentID,
	}

	// 获取点赞数
	likesCount, err := r.GetLikesCount(ctx, fragmentID)
	if err != nil {
		return nil, err
	}
	stats.LikesCount = int(likesCount)

	// 获取评论数
	commentsCount, err := r.GetCommentsCount(ctx, fragmentID)
	if err != nil {
		return nil, err
	}
	stats.CommentsCount = int(commentsCount)

	// 获取分享数
	sharesCount, err := r.GetSharesCount(ctx, fragmentID)
	if err != nil {
		return nil, err
	}
	stats.SharesCount = int(sharesCount)

	// 检查当前用户是否点赞
	if userID != "" {
		isLiked, err := r.IsLiked(ctx, fragmentID, userID)
		if err != nil {
			return nil, err
		}
		stats.IsLikedByUser = isLiked
	}

	return stats, nil
}

// BatchGetFragmentStats 批量获取多个碎片的统计信息
func (r *FragmentInteractionRepository) BatchGetFragmentStats(ctx context.Context, fragmentIDs []string, userID string) (map[string]*domain.FragmentStats, error) {
	statsMap := make(map[string]*domain.FragmentStats)

	// 批量获取点赞数
	var likesStats []struct {
		FragmentID string
		Count      int64
	}
	err := r.db.WithContext(ctx).
		Model(&domain.FragmentLike{}).
		Select("fragment_id, count(*) as count").
		Where("fragment_id IN ?", fragmentIDs).
		Group("fragment_id").
		Find(&likesStats).Error
	if err != nil {
		return nil, err
	}

	// 批量获取评论数（统一 comments 表）
	var commentsStats []struct {
		FragmentID string `gorm:"column:fragment_id"`
		Count      int64
	}
	if len(fragmentIDs) > 0 {
		err = r.db.WithContext(ctx).Raw(`
			SELECT target_id AS fragment_id, COUNT(*) AS count
			FROM comments
			WHERE deleted_at IS NULL AND target_type = 'fragment' AND target_id IN ?
			GROUP BY target_id
		`, fragmentIDs).Scan(&commentsStats).Error
		if err != nil {
			return nil, err
		}
	}

	// 批量获取分享数
	var sharesStats []struct {
		FragmentID string
		Count      int64
	}
	err = r.db.WithContext(ctx).
		Model(&domain.FragmentShare{}).
		Select("fragment_id, count(*) as count").
		Where("fragment_id IN ?", fragmentIDs).
		Group("fragment_id").
		Find(&sharesStats).Error
	if err != nil {
		return nil, err
	}

	// 获取用户点赞的碎片
	var likedFragmentIDs []string
	if userID != "" {
		err = r.db.WithContext(ctx).
			Model(&domain.FragmentLike{}).
			Where("fragment_id IN ? AND user_id = ?", fragmentIDs, userID).
			Pluck("fragment_id", &likedFragmentIDs).Error
		if err != nil {
			return nil, err
		}
	}

	// 组装统计信息
	likedMap := make(map[string]bool)
	for _, id := range likedFragmentIDs {
		likedMap[id] = true
	}

	for _, fragmentID := range fragmentIDs {
		stats := &domain.FragmentStats{
			FragmentID:    fragmentID,
			LikesCount:    0,
			CommentsCount: 0,
			SharesCount:   0,
			IsLikedByUser: false,
		}

		for _, ls := range likesStats {
			if ls.FragmentID == fragmentID {
				stats.LikesCount = int(ls.Count)
				break
			}
		}

		for _, cs := range commentsStats {
			if cs.FragmentID == fragmentID {
				stats.CommentsCount = int(cs.Count)
				break
			}
		}

		for _, ss := range sharesStats {
			if ss.FragmentID == fragmentID {
				stats.SharesCount = int(ss.Count)
				break
			}
		}

		stats.IsLikedByUser = likedMap[fragmentID]
		statsMap[fragmentID] = stats
	}

	return statsMap, nil
}
