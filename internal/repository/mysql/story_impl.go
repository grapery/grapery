package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// StoryByID retrieves a story by ID
func (r *Repository) StoryByID(ctx context.Context, id string) (*domain.Story, error) {
	var story Story
	if err := r.db.WithContext(ctx).Preload("Author").First(&story, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	domainStory := r.storyToDomain(story)
	return &domainStory, nil
}

// UpdateStory updates an existing story
func (r *Repository) UpdateStory(ctx context.Context, story *domain.Story) error {
	dbStory := Story{
		ID:          story.ID,
		Title:       story.Title,
		Description: story.Description,
		CoverImage:  story.CoverImage,
		AuthorID:    story.Author.ID,
		GroupID:     &story.GroupID,
		Genre:       story.Genre,
		Status:      story.Status,
		Likes:       story.Likes,
		Followers:   story.Followers,
		Panels:      story.Panels,
		UpdatedAt:   time.Now(),
	}

	if story.GroupID == "" {
		dbStory.GroupID = nil
	}

	if err := r.db.WithContext(ctx).Model(&Story{}).Where("id = ?", story.ID).Updates(&dbStory).Error; err != nil {
		return err
	}

	// 重新加载更新后的故事
	var updated Story
	if err := r.db.WithContext(ctx).Preload("Author").First(&updated, "id = ?", story.ID).Error; err != nil {
		return err
	}

	*story = r.storyToDomain(updated)
	return nil
}

// DeleteStory deletes a story (soft delete)
func (r *Repository) DeleteStory(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Story{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// LikeStory adds a like to a story
func (r *Repository) LikeStory(ctx context.Context, userID, storyID string) error {
	// 检查是否已经点赞
	var count int64
	if err := r.db.WithContext(ctx).Model(&StoryLike{}).
		Where("user_id = ? AND story_id = ?", userID, storyID).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("already liked")
	}

	// 创建点赞记录
	like := StoryLike{
		UserID:    userID,
		StoryID:   storyID,
		CreatedAt: time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&like).Error; err != nil {
		return err
	}

	// 更新故事的点赞数
	if err := r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("likes", gorm.Expr("likes + ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

// UnlikeStory removes a like from a story
func (r *Repository) UnlikeStory(ctx context.Context, userID, storyID string) error {
	// 删除点赞记录
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND story_id = ?", userID, storyID).
		Delete(&StoryLike{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("not liked")
	}

	// 更新故事的点赞数
	if err := r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("likes", gorm.Expr("GREATEST(likes - ?, 0)", 1)).Error; err != nil {
		return err
	}

	return nil
}

// FollowStory adds a follow to a story
func (r *Repository) FollowStory(ctx context.Context, userID, storyID string) error {
	// 检查是否已经关注
	var count int64
	if err := r.db.WithContext(ctx).Model(&StoryFollow{}).
		Where("user_id = ? AND story_id = ?", userID, storyID).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("already following")
	}

	// 创建关注记录
	follow := StoryFollow{
		UserID:    userID,
		StoryID:   storyID,
		CreatedAt: time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&follow).Error; err != nil {
		return err
	}

	// 更新故事的关注数
	if err := r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("followers", gorm.Expr("followers + ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

// UnfollowStory removes a follow from a story
func (r *Repository) UnfollowStory(ctx context.Context, userID, storyID string) error {
	// 删除关注记录
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND story_id = ?", userID, storyID).
		Delete(&StoryFollow{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("not following")
	}

	// 更新故事的关注数
	if err := r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("followers", gorm.Expr("GREATEST(followers - ?, 0)", 1)).Error; err != nil {
		return err
	}

	return nil
}

// ========== Story Contributor operations ==========

// AddStoryContributor adds a contributor to a story
func (r *Repository) AddStoryContributor(ctx context.Context, contributor *domain.StoryContributor) error {
	dbContributor := StoryContributor{
		ID:        contributor.ID,
		StoryID:   contributor.StoryID,
		UserID:    contributor.UserID,
		Role:      string(contributor.Role),
		InvitedBy: contributor.InvitedBy,
		JoinedAt:  time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&dbContributor).Error; err != nil {
		return err
	}

	contributor.JoinedAt = dbContributor.JoinedAt.Unix()
	return nil
}

// RemoveStoryContributor removes a contributor from a story
func (r *Repository) RemoveStoryContributor(ctx context.Context, storyID, userID string) error {
	result := r.db.WithContext(ctx).
		Where("story_id = ? AND user_id = ?", storyID, userID).
		Delete(&StoryContributor{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetStoryContributors retrieves all contributors of a story
func (r *Repository) GetStoryContributors(ctx context.Context, storyID string, limit, offset int) ([]*domain.StoryContributor, error) {
	var contributors []StoryContributor

	query := r.db.WithContext(ctx).
		Preload("User").
		Preload("Inviter").
		Where("story_id = ?", storyID).
		Order("joined_at ASC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&contributors).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.StoryContributor, len(contributors))
	for i, c := range contributors {
		result[i] = r.storyContributorToDomain(c)
	}

	return result, nil
}

// GetStoryContributor retrieves a specific contributor
func (r *Repository) GetStoryContributor(ctx context.Context, storyID, userID string) (*domain.StoryContributor, error) {
	var contributor StoryContributor

	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Inviter").
		Where("story_id = ? AND user_id = ?", storyID, userID).
		First(&contributor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return r.storyContributorToDomain(contributor), nil
}

// IsStoryContributor checks if a user is a contributor of a story
func (r *Repository) IsStoryContributor(ctx context.Context, storyID, userID string) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).Model(&StoryContributor{}).
		Where("story_id = ? AND user_id = ?", storyID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateStoryContributorRole updates the role of a contributor
func (r *Repository) UpdateStoryContributorRole(ctx context.Context, storyID, userID string, role domain.StoryContributorRole) error {
	result := r.db.WithContext(ctx).Model(&StoryContributor{}).
		Where("story_id = ? AND user_id = ?", storyID, userID).
		Update("role", string(role))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// storyContributorToDomain converts a database StoryContributor to domain model
func (r *Repository) storyContributorToDomain(c StoryContributor) *domain.StoryContributor {
	contributor := &domain.StoryContributor{
		ID:        c.ID,
		StoryID:   c.StoryID,
		UserID:    c.UserID,
		Role:      domain.StoryContributorRole(c.Role),
		InvitedBy: c.InvitedBy,
		JoinedAt:  c.JoinedAt.Unix(),
	}

	if c.User.ID != "" {
		user := r.userToDomain(c.User)
		contributor.User = &user
	}

	if c.Inviter.ID != "" {
		inviter := r.userToDomain(c.Inviter)
		contributor.Inviter = &inviter
	}

	return contributor
}
