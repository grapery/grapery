package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// FollowRepositoryImpl implements domain.FollowRepository
type FollowRepositoryImpl struct {
	db *gorm.DB
}

// NewFollowRepository creates a new FollowRepository instance
func NewFollowRepository(db *gorm.DB) domain.FollowRepository {
	return &FollowRepositoryImpl{db: db}
}

// CreateFollow creates a follow relationship
func (r *FollowRepositoryImpl) CreateFollow(ctx context.Context, follow *domain.Follow) error {
	model := FollowToModel(follow)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create follow: %w", err)
	}
	return nil
}

// DeleteFollow deletes a follow relationship by ID
func (r *FollowRepositoryImpl) DeleteFollow(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Follow{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete follow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("follow not found")
	}
	return nil
}

// GetFollowByID gets a follow relationship by ID
func (r *FollowRepositoryImpl) GetFollowByID(ctx context.Context, id string) (*domain.Follow, error) {
	var model Follow
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("follow not found")
		}
		return nil, fmt.Errorf("failed to get follow: %w", err)
	}
	return ModelToFollow(&model), nil
}

// GetFollowsByFollower gets all follows by a follower
func (r *FollowRepositoryImpl) GetFollowsByFollower(ctx context.Context, userID string, followableType domain.FollowableType) ([]*domain.Follow, error) {
	var models []Follow
	query := r.db.WithContext(ctx).Where("follower_id = ?", userID)
	if followableType != "" {
		query = query.Where("followable_type = ?", string(followableType))
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get follows by follower: %w", err)
	}
	return ModelsToFollows(models), nil
}

// GetFollowsByFollowable gets all follows for a specific followable entity
func (r *FollowRepositoryImpl) GetFollowsByFollowable(ctx context.Context, followableType domain.FollowableType, followableID string) ([]*domain.Follow, error) {
	var models []Follow
	if err := r.db.WithContext(ctx).
		Where("followable_type = ? AND followable_id = ?", string(followableType), followableID).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get follows by followable: %w", err)
	}
	return ModelsToFollows(models), nil
}

// CheckFollowStatus checks if a user is following a specific entity
func (r *FollowRepositoryImpl) CheckFollowStatus(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Follow{}).
		Where("follower_id = ? AND followable_type = ? AND followable_id = ?",
			followerID, string(followableType), followableID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check follow status: %w", err)
	}
	return count > 0, nil
}

// GetFollowersCount gets the count of followers for a specific entity
func (r *FollowRepositoryImpl) GetFollowersCount(ctx context.Context, followableType domain.FollowableType, followableID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Follow{}).
		Where("followable_type = ? AND followable_id = ?", string(followableType), followableID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get followers count: %w", err)
	}
	return int(count), nil
}
