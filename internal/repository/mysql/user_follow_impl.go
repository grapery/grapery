package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

func (r *Repository) FollowUser(ctx context.Context, followerID, followeeID string) error {
	if followerID == followeeID {
		return fmt.Errorf("cannot follow yourself")
	}
	
	var existing UserFollow
	err := r.db.WithContext(ctx).Where("follower_id = ? AND followee_id = ?", followerID, followeeID).First(&existing).Error
	if err == nil {
		return nil // Already following
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing follow: %w", err)
	}

	follow := UserFollow{
		ID:         uuid.New().String(),
		FollowerID: followerID,
		FolloweeID: followeeID,
	}
	
	if err := r.db.WithContext(ctx).Create(&follow).Error; err != nil {
		return fmt.Errorf("failed to create follow: %w", err)
	}

	// Update follower/following counts
	r.db.WithContext(ctx).Model(&User{}).Where("id = ?", followerID).UpdateColumn("following", gorm.Expr("following + ?", 1))
	r.db.WithContext(ctx).Model(&User{}).Where("id = ?", followeeID).UpdateColumn("followers", gorm.Expr("followers + ?", 1))

	return nil
}

func (r *Repository) UnfollowUser(ctx context.Context, followerID, followeeID string) error {
	result := r.db.WithContext(ctx).Where("follower_id = ? AND followee_id = ?", followerID, followeeID).Delete(&UserFollow{})
	if result.Error != nil {
		return fmt.Errorf("failed to unfollow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil // Not following
	}

	// Update counts
	r.db.WithContext(ctx).Model(&User{}).Where("id = ?", followerID).UpdateColumn("following", gorm.Expr("following - ?", 1))
	r.db.WithContext(ctx).Model(&User{}).Where("id = ?", followeeID).UpdateColumn("followers", gorm.Expr("followers - ?", 1))

	return nil
}

func (r *Repository) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&UserFollow{}).Where("follower_id = ? AND followee_id = ?", followerID, followeeID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check follow status: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) Followers(ctx context.Context, userID string, limit, offset int) ([]*domain.User, error) {
	var follows []UserFollow
	query := r.db.WithContext(ctx).Preload("Follower").Where("followee_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&follows).Error; err != nil {
		return nil, fmt.Errorf("failed to get followers: %w", err)
	}

	result := make([]*domain.User, len(follows))
	for i, f := range follows {
		user := r.userToDomain(f.Follower)
		result[i] = &user
	}
	return result, nil
}

func (r *Repository) Following(ctx context.Context, userID string, limit, offset int) ([]*domain.User, error) {
	var follows []UserFollow
	query := r.db.WithContext(ctx).Preload("Followee").Where("follower_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&follows).Error; err != nil {
		return nil, fmt.Errorf("failed to get following: %w", err)
	}

	result := make([]*domain.User, len(follows))
	for i, f := range follows {
		user := r.userToDomain(f.Followee)
		result[i] = &user
	}
	return result, nil
}

// ========== Liked Content ==========

func (r *Repository) LikedStories(ctx context.Context, userID string, limit, offset int) ([]*domain.Story, error) {
	var likes []StoryLike
	query := r.db.WithContext(ctx).
		Preload("Story").
		Preload("Story.Author").
		Where("user_id = ?", userID).
		Order("created_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	
	if err := query.Find(&likes).Error; err != nil {
		return nil, fmt.Errorf("failed to get liked stories: %w", err)
	}
	
	result := make([]*domain.Story, len(likes))
	for i, like := range likes {
		story := r.storyToDomain(like.Story)
		result[i] = &story
	}
	return result, nil
}

func (r *Repository) LikedCharacters(ctx context.Context, userID string, limit, offset int) ([]*domain.Character, error) {
	var follows []CharacterFollow
	query := r.db.WithContext(ctx).
		Preload("Character").
		Preload("Character.Author").
		Where("user_id = ?", userID).
		Order("created_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	
	if err := query.Find(&follows).Error; err != nil {
		return nil, fmt.Errorf("failed to get liked characters: %w", err)
	}
	
	result := make([]*domain.Character, len(follows))
	for i, follow := range follows {
		character := r.characterToDomain(follow.Character)
		result[i] = &character
	}
	return result, nil
}

func (r *Repository) LikedStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, error) {
	var likes []StoryboardLike
	query := r.db.WithContext(ctx).
		Preload("Storyboard").
		Preload("Storyboard.Creator").
		Where("user_id = ?", userID).
		Order("created_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	
	if err := query.Find(&likes).Error; err != nil {
		return nil, fmt.Errorf("failed to get liked storyboards: %w", err)
	}
	
	result := make([]*domain.Storyboard, len(likes))
	for i, like := range likes {
		storyboard, err := r.storyboardToDomain(ctx, like.Storyboard)
		if err != nil {
			return nil, err
		}
		result[i] = &storyboard
	}
	return result, nil
}

