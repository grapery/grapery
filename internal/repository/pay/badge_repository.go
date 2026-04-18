package pay

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// BadgeRepository 徽章仓库
type BadgeRepository struct {
	db *gorm.DB
}

// NewBadgeRepository 创建徽章仓库
func NewBadgeRepository() *BadgeRepository {
	db := DataBase()
	if db == nil {
		return nil
	}

	// 注意：表的迁移现在统一由 migrations 包管理
	// 迁移步骤在 pay/migrations_register.go 中注册
	// 预定义徽章的初始化也在 migrations 包中统一处理

	return &BadgeRepository{db: db}
}

// GetAllBadges 获取所有徽章定义
func (r *BadgeRepository) GetAllBadges(ctx context.Context) ([]*Badge, error) {
	var badges []*Badge
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("display_order ASC").
		Find(&badges).Error
	return badges, err
}

// GetBadgeByCode 根据代码获取徽章
func (r *BadgeRepository) GetBadgeByCode(ctx context.Context, code string) (*Badge, error) {
	var badge Badge
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&badge).Error
	if err != nil {
		return nil, err
	}
	return &badge, nil
}

// GetBadgeByID 根据ID获取徽章
func (r *BadgeRepository) GetBadgeByID(ctx context.Context, id uint) (*Badge, error) {
	var badge Badge
	err := r.db.WithContext(ctx).First(&badge, id).Error
	if err != nil {
		return nil, err
	}
	return &badge, nil
}

// GetBadgesByCategory 根据类别获取徽章
func (r *BadgeRepository) GetBadgesByCategory(ctx context.Context, category BadgeCategory) ([]*Badge, error) {
	var badges []*Badge
	err := r.db.WithContext(ctx).
		Where("category = ? AND is_active = ?", category, true).
		Order("display_order ASC").
		Find(&badges).Error
	return badges, err
}

// GetUserBadges 获取用户已获得的徽章
func (r *BadgeRepository) GetUserBadges(ctx context.Context, userID string) ([]*UserBadge, error) {
	var userBadges []*UserBadge
	err := r.db.WithContext(ctx).
		Preload("Badge").
		Where("user_id = ?", userID).
		Order("earned_at DESC").
		Find(&userBadges).Error
	return userBadges, err
}

// GetUserPinnedBadges 获取用户置顶的徽章
func (r *BadgeRepository) GetUserPinnedBadges(ctx context.Context, userID string) ([]*UserBadge, error) {
	var userBadges []*UserBadge
	err := r.db.WithContext(ctx).
		Preload("Badge").
		Where("user_id = ? AND is_pinned = ?", userID, true).
		Order("earned_at DESC").
		Find(&userBadges).Error
	return userBadges, err
}

// GetUserNewBadges 获取用户未查看的新徽章
func (r *BadgeRepository) GetUserNewBadges(ctx context.Context, userID string) ([]*UserBadge, error) {
	var userBadges []*UserBadge
	err := r.db.WithContext(ctx).
		Preload("Badge").
		Where("user_id = ? AND is_new = ?", userID, true).
		Order("earned_at DESC").
		Find(&userBadges).Error
	return userBadges, err
}

// HasUserBadge 检查用户是否已拥有某徽章
func (r *BadgeRepository) HasUserBadge(ctx context.Context, userID string, badgeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&UserBadge{}).
		Where("user_id = ? AND badge_id = ?", userID, badgeID).
		Count(&count).Error
	return count > 0, err
}

// AwardBadge 授予用户徽章
func (r *BadgeRepository) AwardBadge(ctx context.Context, userID string, badgeID uint) (*UserBadge, error) {
	// 检查是否已拥有
	has, err := r.HasUserBadge(ctx, userID, badgeID)
	if err != nil {
		return nil, err
	}
	if has {
		return nil, nil // 已拥有，不重复授予
	}

	userBadge := &UserBadge{
		UserID:   userID,
		BadgeID:  badgeID,
		EarnedAt: time.Now().Unix(),
		IsNew:    true,
		IsPinned: false,
	}

	err = r.db.WithContext(ctx).Create(userBadge).Error
	if err != nil {
		return nil, err
	}

	// 加载徽章信息
	err = r.db.WithContext(ctx).Preload("Badge").First(userBadge, userBadge.ID).Error
	if err != nil {
		return userBadge, nil
	}

	return userBadge, nil
}

// MarkBadgeAsViewed 标记徽章为已查看
func (r *BadgeRepository) MarkBadgeAsViewed(ctx context.Context, userID string, badgeIDs []uint) error {
	return r.db.WithContext(ctx).
		Model(&UserBadge{}).
		Where("user_id = ? AND badge_id IN ?", userID, badgeIDs).
		Update("is_new", false).Error
}

// MarkAllBadgesAsViewed 标记所有徽章为已查看
func (r *BadgeRepository) MarkAllBadgesAsViewed(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&UserBadge{}).
		Where("user_id = ? AND is_new = ?", userID, true).
		Update("is_new", false).Error
}

// PinBadge 置顶徽章
func (r *BadgeRepository) PinBadge(ctx context.Context, userID string, badgeID uint) error {
	return r.db.WithContext(ctx).
		Model(&UserBadge{}).
		Where("user_id = ? AND badge_id = ?", userID, badgeID).
		Update("is_pinned", true).Error
}

// UnpinBadge 取消置顶徽章
func (r *BadgeRepository) UnpinBadge(ctx context.Context, userID string, badgeID uint) error {
	return r.db.WithContext(ctx).
		Model(&UserBadge{}).
		Where("user_id = ? AND badge_id = ?", userID, badgeID).
		Update("is_pinned", false).Error
}

// GetUserBadgeStats 获取用户徽章统计
func (r *BadgeRepository) GetUserBadgeStats(ctx context.Context, userID string) (*UserBadgeStats, error) {
	var stats UserBadgeStats
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&stats).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新的统计记录
			stats = UserBadgeStats{
				UserID:      userID,
				LastUpdated: time.Now().Unix(),
			}
			if err := r.db.WithContext(ctx).Create(&stats).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &stats, nil
}

// UpdateUserBadgeStats 更新用户徽章统计
func (r *BadgeRepository) UpdateUserBadgeStats(ctx context.Context, stats *UserBadgeStats) error {
	stats.LastUpdated = time.Now().Unix()
	return r.db.WithContext(ctx).Save(stats).Error
}

// IncrementStoryCount 增加故事数量
func (r *BadgeRepository) IncrementStoryCount(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&UserBadgeStats{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"story_count":  gorm.Expr("story_count + 1"),
			"last_updated": time.Now().Unix(),
		}).Error
}

// IncrementStoryboardCount 增加故事版数量
func (r *BadgeRepository) IncrementStoryboardCount(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&UserBadgeStats{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"storyboard_count": gorm.Expr("storyboard_count + 1"),
			"last_updated":     time.Now().Unix(),
		}).Error
}

// IncrementStoryLikes 增加故事点赞数
func (r *BadgeRepository) IncrementStoryLikes(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&UserBadgeStats{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"story_likes":  gorm.Expr("story_likes + 1"),
			"total_likes":  gorm.Expr("total_likes + 1"),
			"last_updated": time.Now().Unix(),
		}).Error
}

// IncrementStoryboardLikes 增加故事版点赞数
func (r *BadgeRepository) IncrementStoryboardLikes(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&UserBadgeStats{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"storyboard_likes": gorm.Expr("storyboard_likes + 1"),
			"total_likes":      gorm.Expr("total_likes + 1"),
			"last_updated":     time.Now().Unix(),
		}).Error
}

// UpdateFollowerCount 更新粉丝数量
func (r *BadgeRepository) UpdateFollowerCount(ctx context.Context, userID string, count int) error {
	return r.db.WithContext(ctx).
		Model(&UserBadgeStats{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"follower_count": count,
			"last_updated":   time.Now().Unix(),
		}).Error
}

// GetUserBadgeCount 获取用户已获得的徽章数量
func (r *BadgeRepository) GetUserBadgeCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&UserBadge{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// GetUserTotalPoints 获取用户总积分
func (r *BadgeRepository) GetUserTotalPoints(ctx context.Context, userID string) (int, error) {
	var total int
	err := r.db.WithContext(ctx).
		Model(&UserBadge{}).
		Joins("JOIN badges ON user_badges.badge_id = badges.id").
		Where("user_badges.user_id = ?", userID).
		Select("COALESCE(SUM(badges.points), 0)").
		Scan(&total).Error
	return total, err
}

// CheckAndAwardBadges 检查并授予符合条件的徽章
func (r *BadgeRepository) CheckAndAwardBadges(ctx context.Context, userID string) ([]*UserBadge, error) {
	stats, err := r.GetUserBadgeStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	badges, err := r.GetAllBadges(ctx)
	if err != nil {
		return nil, err
	}

	var awardedBadges []*UserBadge

	for _, badge := range badges {
		// 检查是否已拥有
		has, err := r.HasUserBadge(ctx, userID, badge.ID)
		if err != nil || has {
			continue
		}

		// 检查是否满足条件
		var currentValue int
		switch badge.Category {
		case BadgeCategoryStory:
			currentValue = stats.StoryCount
		case BadgeCategoryStoryboard:
			currentValue = stats.StoryboardCount
		case BadgeCategoryLike:
			currentValue = stats.TotalLikes
		case BadgeCategoryFollower:
			currentValue = stats.FollowerCount
		default:
			continue // 特殊徽章需要手动授予
		}

		if currentValue >= badge.Threshold {
			awarded, err := r.AwardBadge(ctx, userID, badge.ID)
			if err == nil && awarded != nil {
				awardedBadges = append(awardedBadges, awarded)
			}
		}
	}

	// 更新用户徽章统计
	if len(awardedBadges) > 0 {
		count, _ := r.GetUserBadgeCount(ctx, userID)
		points, _ := r.GetUserTotalPoints(ctx, userID)
		stats.TotalBadges = int(count)
		stats.TotalPoints = points
		r.UpdateUserBadgeStats(ctx, stats)
	}

	return awardedBadges, nil
}

// GetBadgeProgress 获取徽章进度
func (r *BadgeRepository) GetBadgeProgress(ctx context.Context, userID string) ([]*BadgeProgress, error) {
	stats, err := r.GetUserBadgeStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	badges, err := r.GetAllBadges(ctx)
	if err != nil {
		return nil, err
	}

	var progresses []*BadgeProgress

	for _, badge := range badges {
		has, err := r.HasUserBadge(ctx, userID, badge.ID)
		if err != nil {
			continue
		}

		var currentValue int
		switch badge.Category {
		case BadgeCategoryStory:
			currentValue = stats.StoryCount
		case BadgeCategoryStoryboard:
			currentValue = stats.StoryboardCount
		case BadgeCategoryLike:
			currentValue = stats.TotalLikes
		case BadgeCategoryFollower:
			currentValue = stats.FollowerCount
		default:
			continue // 特殊徽章不显示进度
		}

		progress := float64(currentValue) / float64(badge.Threshold) * 100
		if progress > 100 {
			progress = 100
		}

		progresses = append(progresses, &BadgeProgress{
			Badge:       badge,
			Current:     currentValue,
			Target:      badge.Threshold,
			Progress:    progress,
			IsCompleted: has,
		})
	}

	return progresses, nil
}

// GetUserBadgeProfile 获取用户完整的徽章档案
func (r *BadgeRepository) GetUserBadgeProfile(ctx context.Context, userID string) (*UserBadgeProfile, error) {
	stats, err := r.GetUserBadgeStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	earnedBadges, err := r.GetUserBadges(ctx, userID)
	if err != nil {
		return nil, err
	}

	pinnedBadges, err := r.GetUserPinnedBadges(ctx, userID)
	if err != nil {
		return nil, err
	}

	newBadges, err := r.GetUserNewBadges(ctx, userID)
	if err != nil {
		return nil, err
	}

	progress, err := r.GetBadgeProgress(ctx, userID)
	if err != nil {
		return nil, err
	}

	allBadges, err := r.GetAllBadges(ctx)
	if err != nil {
		return nil, err
	}

	completionRate := float64(0)
	if len(allBadges) > 0 {
		completionRate = float64(len(earnedBadges)) / float64(len(allBadges)) * 100
	}

	return &UserBadgeProfile{
		UserID:         userID,
		Stats:          stats,
		EarnedBadges:   earnedBadges,
		PinnedBadges:   pinnedBadges,
		NewBadges:      newBadges,
		BadgeProgress:  progress,
		TotalBadges:    len(earnedBadges),
		TotalPoints:    stats.TotalPoints,
		CompletionRate: completionRate,
	}, nil
}

// SyncUserStats 同步用户统计数据（从主服务获取真实数据）
func (r *BadgeRepository) SyncUserStats(ctx context.Context, userID string, storyCount, storyboardCount, totalLikes, storyLikes, storyboardLikes, followerCount, followingCount int) error {
	stats, err := r.GetUserBadgeStats(ctx, userID)
	if err != nil {
		return err
	}

	stats.StoryCount = storyCount
	stats.StoryboardCount = storyboardCount
	stats.TotalLikes = totalLikes
	stats.StoryLikes = storyLikes
	stats.StoryboardLikes = storyboardLikes
	stats.FollowerCount = followerCount
	stats.FollowingCount = followingCount
	stats.LastUpdated = time.Now().Unix()

	return r.UpdateUserBadgeStats(ctx, stats)
}
