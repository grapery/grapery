package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// LikeRepositoryImpl implements domain.LikeRepository
type LikeRepositoryImpl struct {
	db *gorm.DB
}

// NewLikeRepository creates a new LikeRepository instance
func NewLikeRepository(db *gorm.DB) domain.LikeRepository {
	return &LikeRepositoryImpl{db: db}
}

// CreateLike creates a like
func (r *LikeRepositoryImpl) CreateLike(ctx context.Context, like *domain.Like) error {
	model := LikeToModel(like)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create like: %w", err)
	}
	return nil
}

// DeleteLike deletes a like by ID
func (r *LikeRepositoryImpl) DeleteLike(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Like{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete like: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("like not found")
	}
	return nil
}

// GetLikeByID gets a like by ID
func (r *LikeRepositoryImpl) GetLikeByID(ctx context.Context, id string) (*domain.Like, error) {
	var model Like
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("like not found")
		}
		return nil, fmt.Errorf("failed to get like: %w", err)
	}
	return ModelToLike(&model), nil
}

// GetLikesByUser gets all likes by a user
func (r *LikeRepositoryImpl) GetLikesByUser(ctx context.Context, userID string, likeableType domain.LikeableType) ([]*domain.Like, error) {
	var models []Like
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if likeableType != "" {
		query = query.Where("likeable_type = ?", string(likeableType))
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get likes by user: %w", err)
	}
	return ModelsToLikes(models), nil
}

// GetLikesByLikeable gets all likes for a specific likeable entity
func (r *LikeRepositoryImpl) GetLikesByLikeable(ctx context.Context, likeableType domain.LikeableType, likeableID string) ([]*domain.Like, error) {
	var models []Like
	if err := r.db.WithContext(ctx).
		Where("likeable_type = ? AND likeable_id = ?", string(likeableType), likeableID).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get likes by likeable: %w", err)
	}
	return ModelsToLikes(models), nil
}

// CheckLikeStatus checks if a user has liked a specific entity
func (r *LikeRepositoryImpl) CheckLikeStatus(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Like{}).
		Where("user_id = ? AND likeable_type = ? AND likeable_id = ?", 
			userID, string(likeableType), likeableID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check like status: %w", err)
	}
	return count > 0, nil
}

// GetLikesCount gets the count of likes for a specific entity
func (r *LikeRepositoryImpl) GetLikesCount(ctx context.Context, likeableType domain.LikeableType, likeableID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Like{}).
		Where("likeable_type = ? AND likeable_id = ?", string(likeableType), likeableID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get likes count: %w", err)
	}
	return int(count), nil
}
