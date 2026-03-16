package repository

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/gorm"
)

type FragmentRepository struct {
	db *gorm.DB
}

type FragmentEngagement struct {
	Likes    int
	Comments int
	IsLiked  bool
}

func NewFragmentRepository(db *gorm.DB) *FragmentRepository {
	return &FragmentRepository{db: db}
}

// Create creates a new fragment and increments user's fragments count
func (r *FragmentRepository) Create(ctx context.Context, fragment *domain.Fragment) error {
	dbFragment := mysql.DomainToFragmentDB(fragment)
	if dbFragment == nil {
		return domain.ErrInvalidInput
	}

	userID := fragment.UserID
	if userID == "" {
		userID = fragment.CreatorID
	}

	// Start a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the fragment
		if err := tx.Create(dbFragment).Error; err != nil {
			return err
		}

		// Copy ID back to domain fragment
		fragment.ID = dbFragment.ID
		fragment.CreatedAt = dbFragment.CreatedAt
		fragment.UpdatedAt = dbFragment.UpdatedAt

		// Increment user's fragments_count
		if err := tx.Model(&domain.User{}).
			Where("id = ?", userID).
			UpdateColumn("fragments_count", gorm.Expr("fragments_count + 1")).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// GetByID retrieves a fragment by ID
func (r *FragmentRepository) GetByID(ctx context.Context, id string) (*domain.Fragment, error) {
	var fragment mysql.FragmentDB
	err := r.db.WithContext(ctx).Preload("Creator").Where("id = ?", id).First(&fragment).Error
	if err != nil {
		return nil, err
	}
	return mysql.FragmentDBToDomain(&fragment), nil
}

// List retrieves fragments with pagination
func (r *FragmentRepository) List(ctx context.Context, limit, offset int, visibility string) ([]*domain.Fragment, int64, error) {
	var dbFragments []*mysql.FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&mysql.FragmentDB{})

	if visibility != "" {
		query = query.Where("visibility = ?", visibility)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

func fragmentDBListToDomain(list []*mysql.FragmentDB) []*domain.Fragment {
	if list == nil {
		return nil
	}
	result := make([]*domain.Fragment, len(list))
	for i, f := range list {
		result[i] = mysql.FragmentDBToDomain(f)
	}
	return result
}

// ListByCreatorID retrieves fragments by creator ID
func (r *FragmentRepository) ListByCreatorID(ctx context.Context, creatorID string, limit, offset int) ([]*domain.Fragment, int64, error) {
	var dbFragments []*mysql.FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).Where("creator_id = ?", creatorID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

// ListByTopic retrieves fragments by topic with optional converted filter
// convertedOnly: nil = all, true = only converted, false = only not converted
func (r *FragmentRepository) ListByTopic(ctx context.Context, topic string, limit, offset int, convertedOnly *bool) ([]*domain.Fragment, int64, error) {
	var dbFragments []*mysql.FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("topic = ? AND visibility = ?", topic, domain.FragmentVisibilityPublic)

	if convertedOnly != nil {
		if *convertedOnly {
			query = query.Where("is_converted = ?", true)
		} else {
			query = query.Where("is_converted = ?", false)
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

// ListFollowing retrieves fragments from followed users
func (r *FragmentRepository) ListFollowing(ctx context.Context, userID string, limit, offset int) ([]*domain.Fragment, int64, error) {
	var dbFragments []*mysql.FragmentDB
	var total int64

	// Get list of followed user IDs (user_follows has no status column; uses soft delete via deleted_at)
	var followedUserIDs []string
	err := r.db.WithContext(ctx).Table("user_follows").
		Where("follower_id = ? AND deleted_at IS NULL", userID).
		Pluck("followee_id", &followedUserIDs).Error
	if err != nil {
		return nil, 0, err
	}

	if len(followedUserIDs) == 0 {
		return []*domain.Fragment{}, 0, nil
	}

	query := r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("creator_id IN ? AND (visibility = ? OR visibility = ? OR visibility = ?)",
			followedUserIDs,
			domain.FragmentVisibilityPublic,
			domain.FragmentVisibilityFollowers,
			domain.FragmentVisibilityFollowersLegacy)

	err = query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbFragments).Error

	if err != nil {
		return nil, 0, err
	}

	return fragmentDBListToDomain(dbFragments), total, nil
}

// Update updates a fragment
func (r *FragmentRepository) Update(ctx context.Context, fragment *domain.Fragment) error {
	dbFragment := mysql.DomainToFragmentDB(fragment)
	if dbFragment == nil {
		return domain.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Save(dbFragment).Error
}

// Delete deletes a fragment and decrements user's fragments count
func (r *FragmentRepository) Delete(ctx context.Context, id string) error {
	// First get the fragment to know the creator ID
	var fragment mysql.FragmentDB
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&fragment).Error
	if err != nil {
		return err
	}

	userID := fragment.UserID

	// Start a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete the fragment
		if err := tx.Where("id = ?", id).Delete(&mysql.FragmentDB{}).Error; err != nil {
			return err
		}

		// Decrement user's fragments_count
		if err := tx.Model(&domain.User{}).
			Where("id = ?", userID).
			UpdateColumn("fragments_count", gorm.Expr("fragments_count - 1")).
			Error; err != nil {
			return err
		}

		return nil
	})
}

// IncrementLikes increments the likes count
func (r *FragmentRepository) IncrementLikes(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("likes + 1")).
		Error
}

// DecrementLikes decrements the likes count
func (r *FragmentRepository) DecrementLikes(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("likes - 1")).
		Error
}

// IncrementComments increments the comments count
func (r *FragmentRepository) IncrementComments(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", id).
		UpdateColumn("comments", gorm.Expr("comments + 1")).
		Error
}

// IncrementViews increments the view count for a fragment
func (r *FragmentRepository) IncrementViews(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&mysql.FragmentDB{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).
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

// GetEngagementStats retrieves likes/comments counts and current user's like status.
// Comments are aggregated from both legacy comments table and fragment_comments table.
func (r *FragmentRepository) GetEngagementStats(ctx context.Context, fragmentID, userID string) (FragmentEngagement, error) {
	statsMap, err := r.BatchGetEngagementStats(ctx, []string{fragmentID}, userID)
	if err != nil {
		return FragmentEngagement{}, err
	}
	if stats, ok := statsMap[fragmentID]; ok {
		return stats, nil
	}
	return FragmentEngagement{}, nil
}

// BatchGetEngagementStats retrieves likes/comments counts and current user's like status for multiple fragments.
func (r *FragmentRepository) BatchGetEngagementStats(ctx context.Context, fragmentIDs []string, userID string) (map[string]FragmentEngagement, error) {
	result := make(map[string]FragmentEngagement, len(fragmentIDs))
	if len(fragmentIDs) == 0 {
		return result, nil
	}

	for _, id := range fragmentIDs {
		result[id] = FragmentEngagement{}
	}

	type groupedCount struct {
		FragmentID string
		Count      int64
	}

	var likeCounts []groupedCount
	if err := r.db.WithContext(ctx).
		Model(&domain.FragmentLike{}).
		Select("fragment_id as fragment_id, count(*) as count").
		Where("fragment_id IN ?", fragmentIDs).
		Group("fragment_id").
		Find(&likeCounts).Error; err != nil {
		return nil, err
	}
	for _, row := range likeCounts {
		stats := result[row.FragmentID]
		stats.Likes = int(row.Count)
		result[row.FragmentID] = stats
	}

	var commentCounts []groupedCount
	if err := r.db.WithContext(ctx).
		Table("comments").
		Select("target_id as fragment_id, count(*) as count").
		Where("target_type = ? AND target_id IN ?", "fragment", fragmentIDs).
		Group("target_id").
		Find(&commentCounts).Error; err != nil {
		return nil, err
	}
	for _, row := range commentCounts {
		stats := result[row.FragmentID]
		stats.Comments += int(row.Count)
		result[row.FragmentID] = stats
	}

	var fragmentCommentCounts []groupedCount
	if err := r.db.WithContext(ctx).
		Model(&domain.FragmentComment{}).
		Select("fragment_id as fragment_id, count(*) as count").
		Where("fragment_id IN ?", fragmentIDs).
		Group("fragment_id").
		Find(&fragmentCommentCounts).Error; err != nil {
		return nil, err
	}
	for _, row := range fragmentCommentCounts {
		stats := result[row.FragmentID]
		stats.Comments += int(row.Count)
		result[row.FragmentID] = stats
	}

	if userID != "" {
		var likedRows []struct {
			FragmentID string
		}
		if err := r.db.WithContext(ctx).
			Model(&domain.FragmentLike{}).
			Select("fragment_id").
			Where("user_id = ? AND fragment_id IN ?", userID, fragmentIDs).
			Find(&likedRows).Error; err != nil {
			return nil, err
		}
		for _, row := range likedRows {
			stats := result[row.FragmentID]
			stats.IsLiked = true
			result[row.FragmentID] = stats
		}
	}

	return result, nil
}
