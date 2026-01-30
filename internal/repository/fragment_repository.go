package repository

import (
	"context"
	"fmt"

	"github.com/grapestree/voyager/grapery/internal/domain"
	"gorm.io/gorm"
)

type FragmentRepository struct {
	db *gorm.DB
}

func NewFragmentRepository(db *gorm.DB) *FragmentRepository {
	return &FragmentRepository{db: db}
}

// Create creates a new fragment
func (r *FragmentRepository) Create(ctx context.Context, fragment *domain.Fragment) error {
	return r.db.WithContext(ctx).Create(fragment).Error
}

// GetByID retrieves a fragment by ID
func (r *FragmentRepository) GetByID(ctx context.Context, id string) (*domain.Fragment, error) {
	var fragment domain.Fragment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&fragment).Error
	if err != nil {
		return nil, err
	}
	return &fragment, nil
}

// List retrieves fragments with pagination
func (r *FragmentRepository) List(ctx context.Context, limit, offset int, visibility string) ([]*domain.Fragment, int64, error) {
	var fragments []*domain.Fragment
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Fragment{})

	if visibility != "" {
		query = query.Where("visibility = ?", visibility)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&fragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragments, total, nil
}

// ListByCreatorID retrieves fragments by creator ID
func (r *FragmentRepository) ListByCreatorID(ctx context.Context, creatorID string, limit, offset int) ([]*domain.Fragment, int64, error) {
	var fragments []*domain.Fragment
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Fragment{}).Where("creator_id = ?", creatorID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&fragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragments, total, nil
}

// ListFollowing retrieves fragments from followed users
func (r *FragmentRepository) ListFollowing(ctx context.Context, userID string, limit, offset int) ([]*domain.Fragment, int64, error) {
	var fragments []*domain.Fragment
	var total int64

	// Get list of followed user IDs
	var followedUserIDs []string
	err := r.db.WithContext(ctx).Model(&domain.UserFollow{}).
		Where("follower_id = ? AND status = ?", userID, "active").
		Pluck("following_id", &followedUserIDs).Error
	if err != nil {
		return nil, 0, err
	}

	if len(followedUserIDs) == 0 {
		return []*domain.Fragment{}, 0, nil
	}

	query := r.db.WithContext(ctx).Model(&domain.Fragment{}).
		Where("creator_id IN ? AND (visibility = ? OR visibility = ?)",
			followedUserIDs,
			domain.FragmentVisibilityPublic,
			domain.FragmentVisibilityFollowers)

	err = query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&fragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragments, total, nil
}

// Update updates a fragment
func (r *FragmentRepository) Update(ctx context.Context, fragment *domain.Fragment) error {
	return r.db.WithContext(ctx).Save(fragment).Error
}

// Delete deletes a fragment
func (r *FragmentRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Fragment{}).Error
}

// IncrementLikes increments the likes count
func (r *FragmentRepository) IncrementLikes(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&domain.Fragment{}).
		Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("likes + 1")).
		Error
}

// DecrementLikes decrements the likes count
func (r *FragmentRepository) DecrementLikes(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&domain.Fragment{}).
		Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("likes - 1")).
		Error
}

// IncrementComments increments the comments count
func (r *FragmentRepository) IncrementComments(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&domain.Fragment{}).
		Where("id = ?", id).
		UpdateColumn("comments", gorm.Expr("comments + 1")).
		Error
}

// CreateLike creates a like record
func (r *FragmentRepository) CreateLike(ctx context.Context, like *domain.FragmentLike) error {
	return r.db.WithContext(ctx).Create(like).Error
}

// DeleteLike deletes a like record
func (r *FragmentRepository) DeleteLike(ctx context.Context, fragmentID, userID string) error {
	return r.db.WithContext(ctx).
		Where("fragment_id = ? AND user_id = ?", fragmentID, userID).
		Delete(&domain.FragmentLike{}).Error
}

// IsLiked checks if a user has liked a fragment
func (r *FragmentRepository) IsLiked(ctx context.Context, fragmentID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.FragmentLike{}).
		Where("fragment_id = ? AND user_id = ?", fragmentID, userID).
		Count(&count).Error
	return count > 0, err
}
