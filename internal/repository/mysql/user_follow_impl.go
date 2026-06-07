package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
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

	// 软删行被默认 scope 过滤，但唯一键 (follower_id, followee_id) 仍冲突，应恢复而非 INSERT
	var soft UserFollow
	errSoft := r.db.WithContext(ctx).Unscoped().
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		First(&soft).Error
	if errSoft == nil {
		if soft.DeletedAt.Valid {
			if err := r.db.WithContext(ctx).Unscoped().Model(&UserFollow{}).
				Where("id = ?", soft.ID).
				UpdateColumn("deleted_at", nil).Error; err != nil {
				return fmt.Errorf("failed to restore follow: %w", err)
			}
			r.db.WithContext(ctx).Model(&User{}).Where("id = ?", followerID).UpdateColumn("following", gorm.Expr("following + ?", 1))
			r.db.WithContext(ctx).Model(&User{}).Where("id = ?", followeeID).UpdateColumn("followers", gorm.Expr("followers + ?", 1))
		}
		return nil
	}
	if !errors.Is(errSoft, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check soft-deleted follow: %w", errSoft)
	}

	follow := UserFollow{
		ID:         uuid.New().String(),
		FollowerID: followerID,
		FolloweeID: followeeID,
	}
	if err := r.db.WithContext(ctx).Create(&follow).Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			var dup UserFollow
			if err2 := r.db.WithContext(ctx).Unscoped().
				Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
				First(&dup).Error; err2 != nil {
				return fmt.Errorf("failed to create follow: %w", err)
			}
			if dup.DeletedAt.Valid {
				if err := r.db.WithContext(ctx).Unscoped().Model(&UserFollow{}).
					Where("id = ?", dup.ID).
					UpdateColumn("deleted_at", nil).Error; err != nil {
					return fmt.Errorf("failed to restore follow after duplicate: %w", err)
				}
				r.db.WithContext(ctx).Model(&User{}).Where("id = ?", followerID).UpdateColumn("following", gorm.Expr("following + ?", 1))
				r.db.WithContext(ctx).Model(&User{}).Where("id = ?", followeeID).UpdateColumn("followers", gorm.Expr("followers + ?", 1))
			}
			return nil
		}
		return fmt.Errorf("failed to create follow: %w", err)
	}

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

func (r *Repository) CountFollowersOfUser(ctx context.Context, followeeID string) (int64, error) {
	var c int64
	if err := r.db.WithContext(ctx).Model(&UserFollow{}).Where("followee_id = ?", followeeID).Count(&c).Error; err != nil {
		return 0, fmt.Errorf("failed to count user followers: %w", err)
	}
	return c, nil
}

func (r *Repository) CountFollowingOfUser(ctx context.Context, followerID string) (int64, error) {
	var c int64
	if err := r.db.WithContext(ctx).Model(&UserFollow{}).Where("follower_id = ?", followerID).Count(&c).Error; err != nil {
		return 0, fmt.Errorf("failed to count user following: %w", err)
	}
	return c, nil
}

func (r *Repository) ListUserFollowsByFollower(ctx context.Context, followerID string, limit, offset int) ([]*domain.Follow, error) {
	var rows []UserFollow
	q := r.db.WithContext(ctx).Where("follower_id = ?", followerID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list user follows: %w", err)
	}
	out := make([]*domain.Follow, len(rows))
	for i := range rows {
		row := rows[i]
		out[i] = &domain.Follow{
			ID:                   row.ID,
			FollowerID:           row.FollowerID,
			FollowableType:       domain.FollowableTypeUser,
			FollowableID:         row.FolloweeID,
			NotificationsEnabled: true,
			CreatedAt:            row.CreatedAt.Unix(),
		}
	}
	return out, nil
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

	storyboards := make([]Storyboard, 0, len(likes))
	for _, like := range likes {
		storyboards = append(storyboards, like.Storyboard)
	}
	domainRows, err := r.storyboardsToDomain(ctx, storyboards)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Storyboard, 0, len(domainRows))
	for i := range domainRows {
		copySb := domainRows[i]
		result = append(result, &copySb)
	}
	return result, nil
}

// ========== User Block Operations ==========

func (r *Repository) BlockUser(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return fmt.Errorf("cannot block yourself")
	}

	var existing UserBlock
	err := r.db.WithContext(ctx).Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).First(&existing).Error
	if err == nil {
		return nil // Already blocked
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing block: %w", err)
	}

	block := UserBlock{
		ID:        uuid.New().String(),
		BlockerID: blockerID,
		BlockedID: blockedID,
	}

	if err := r.db.WithContext(ctx).Create(&block).Error; err != nil {
		return fmt.Errorf("failed to create block: %w", err)
	}

	return nil
}

func (r *Repository) UnblockUser(ctx context.Context, blockerID, blockedID string) error {
	result := r.db.WithContext(ctx).Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).Delete(&UserBlock{})
	if result.Error != nil {
		return fmt.Errorf("failed to unblock: %w", result.Error)
	}
	return nil
}

func (r *Repository) IsBlocked(ctx context.Context, blockerID, blockedID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&UserBlock{}).Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check block status: %w", err)
	}
	return count > 0, nil
}

// ListBlockedUserIDs returns the IDs of users that blockerID has blocked.
func (r *Repository) ListBlockedUserIDs(ctx context.Context, blockerID string) ([]string, error) {
	if blockerID == "" {
		return nil, nil
	}
	var ids []string
	if err := r.db.WithContext(ctx).
		Model(&UserBlock{}).
		Where("blocker_id = ?", blockerID).
		Pluck("blocked_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("failed to list blocked user ids: %w", err)
	}
	return ids, nil
}

// ListBlockedUsers returns blocked users with profile metadata for the blocker's list UI.
func (r *Repository) ListBlockedUsers(ctx context.Context, blockerID string, limit, offset int) ([]*domain.BlockedUser, int64, error) {
	if blockerID == "" {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	base := r.db.WithContext(ctx).
		Table("user_blocks").
		Joins("JOIN users ON users.id = user_blocks.blocked_id AND users.deleted_at IS NULL").
		Where("user_blocks.blocker_id = ? AND user_blocks.deleted_at IS NULL", blockerID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count blocked users: %w", err)
	}

	type row struct {
		ID          string    `gorm:"column:id"`
		DisplayName string    `gorm:"column:display_name"`
		Username    string    `gorm:"column:username"`
		BlockedAt   time.Time `gorm:"column:created_at"`
	}
	var rows []row
	if err := base.
		Select("users.id, users.display_name, users.username, user_blocks.created_at").
		Order("user_blocks.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list blocked users: %w", err)
	}

	out := make([]*domain.BlockedUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.BlockedUser{
			ID:          row.ID,
			DisplayName: row.DisplayName,
			Username:    row.Username,
			BlockedAt:   row.BlockedAt.Unix(),
		})
	}
	return out, total, nil
}

// ========== User Report Operations ==========

func (r *Repository) ReportUser(ctx context.Context, reporterID, reportedID string, reason string) error {
	if reporterID == reportedID {
		return fmt.Errorf("cannot report yourself")
	}

	report := UserReport{
		ID:         uuid.New().String(),
		ReporterID: reporterID,
		ReportedID: reportedID,
		Reason:     reason,
		Status:     "pending",
	}

	if err := r.db.WithContext(ctx).Create(&report).Error; err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	return nil
}

// ReportContent records a UGC report against a specific piece of content.
func (r *Repository) ReportContent(ctx context.Context, reporterID, contentType, contentID, reason string) error {
	report := ContentReport{
		ID:          uuid.New().String(),
		ReporterID:  reporterID,
		ContentType: contentType,
		ContentID:   contentID,
		Reason:      reason,
		Status:      "pending",
	}

	if err := r.db.WithContext(ctx).Create(&report).Error; err != nil {
		return fmt.Errorf("failed to create content report: %w", err)
	}

	return nil
}

// ========== Get Liked Content IDs ==========

func (r *Repository) GetLikedStoryIDs(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	var likes []StoryLike
	query := r.db.WithContext(ctx).
		Select("story_id").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&likes).Error; err != nil {
		return nil, fmt.Errorf("failed to get liked story IDs: %w", err)
	}

	ids := make([]string, len(likes))
	for i, like := range likes {
		ids[i] = like.StoryID
	}
	return ids, nil
}

func (r *Repository) GetLikedCharacterIDs(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	var follows []CharacterFollow
	query := r.db.WithContext(ctx).
		Select("character_id").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&follows).Error; err != nil {
		return nil, fmt.Errorf("failed to get liked character IDs: %w", err)
	}

	ids := make([]string, len(follows))
	for i, follow := range follows {
		ids[i] = follow.CharacterID
	}
	return ids, nil
}

func (r *Repository) GetLikedStoryboardIDs(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	var likes []StoryboardLike
	query := r.db.WithContext(ctx).
		Select("storyboard_id").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&likes).Error; err != nil {
		return nil, fmt.Errorf("failed to get liked storyboard IDs: %w", err)
	}

	ids := make([]string, len(likes))
	for i, like := range likes {
		ids[i] = like.StoryboardID
	}
	return ids, nil
}
