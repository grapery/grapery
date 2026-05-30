package mysql

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// CreateStory creates a new story
func (r *Repository) CreateStory(ctx context.Context, story *domain.Story) error {
	// Get Author ID from the embedded Author field
	var authorID string
	if story.Author.ID != "" {
		authorID = story.Author.ID
	}

	dbStory := &Story{
		ID:                  story.ID,
		Title:               story.Title,
		Description:         story.Description,
		CoverImage:          story.CoverImage,
		UserID:              authorID,
		SourceFragmentID:    story.SourceFragmentID,
		Likes:               story.Likes,
		Followers:           story.Followers,
		Panels:              story.PanelCount,
		StoryboardCount:     story.StoryboardCount,
		DefaultSceneCount:   story.DefaultSceneCount,
		Genre:               story.Genre,
		Style:               styleConfigToJSON(story.Style),
		Status:              story.Status,
		IsCollaborationOpen: story.IsCollaborationOpen,
		Visibility:          story.Visibility,
		AllowComments:       story.AllowComments,
		UseAI:               story.UseAI,
		AIAssistanceOptions: aiAssistanceOptionsToJSON(story.AIAssistanceOptions),
	}

	if err := r.db.WithContext(ctx).Create(dbStory).Error; err != nil {
		return err
	}
	// allow_comments 带有 default:true，GORM 在插入时会忽略 false 零值，
	// 因此显式回写一次，确保「关闭评论」的创建请求被持久化。
	if !story.AllowComments {
		if err := r.db.WithContext(ctx).Model(&Story{}).
			Where("id = ?", dbStory.ID).
			UpdateColumn("allow_comments", false).Error; err != nil {
			return err
		}
	}
	return nil
}

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
	// Get Author ID from the embedded Author field
	var authorID string
	if story.Author.ID != "" {
		authorID = story.Author.ID
	}

	dbStory := Story{
		ID:                  story.ID,
		Title:               story.Title,
		Description:         story.Description,
		CoverImage:          story.CoverImage,
		UserID:              authorID,
		Genre:               story.Genre,
		Style:               styleConfigToJSON(story.Style),
		Status:              story.Status,
		Likes:               story.Likes,
		Followers:           story.Followers,
		Panels:              story.PanelCount,
		StoryboardCount:     story.StoryboardCount,
		DefaultSceneCount:   story.DefaultSceneCount,
		IsCollaborationOpen: story.IsCollaborationOpen,
		Visibility:          story.Visibility,
		AllowComments:       story.AllowComments,
		UpdatedAt:           time.Now(),
	}

	if err := r.db.WithContext(ctx).Model(&Story{}).Where("id = ?", story.ID).Updates(&dbStory).Error; err != nil {
		return err
	}

	// 布尔字段（默认值/零值）通过结构体 Updates 会被 GORM 跳过，需用 map 显式写入，
	// 以支持「关闭评论 / 关闭协作」等 false 值的更新。
	if err := r.db.WithContext(ctx).Model(&Story{}).Where("id = ?", story.ID).
		Updates(map[string]interface{}{
			"allow_comments":        story.AllowComments,
			"is_collaboration_open": story.IsCollaborationOpen,
		}).Error; err != nil {
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

func isMySQLDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Error 1062") ||
		strings.Contains(s, "Duplicate entry") ||
		strings.Contains(s, "23000")
}

// reviveSoftDeletedStoryLike clears deleted_at and bumps story.likes. Caller must ensure row is soft-deleted.
func (r *Repository) reviveSoftDeletedStoryLike(ctx context.Context, likeID, storyID string) error {
	if err := r.db.WithContext(ctx).Unscoped().Model(&StoryLike{}).
		Where("id = ?", likeID).
		Update("deleted_at", nil).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("likes", gorm.Expr("likes + ?", 1)).Error
}

// LikeStory adds a like to a story.
// StoryLike uses GORM soft delete: a soft-deleted row still occupies idx_user_story, so INSERT can fail with 1062
// while Count/IsStoryLiked see no active row. In that case we revive the row instead of treating it as idempotent no-op.
func (r *Repository) LikeStory(ctx context.Context, userID, storyID string) error {
	var active int64
	if err := r.db.WithContext(ctx).Model(&StoryLike{}).
		Where("user_id = ? AND story_id = ?", userID, storyID).
		Count(&active).Error; err != nil {
		return err
	}
	if active > 0 {
		return domain.ErrAlreadyLiked
	}

	var existing StoryLike
	err := r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND story_id = ?", userID, storyID).
		First(&existing).Error
	if err == nil {
		if !existing.DeletedAt.Valid {
			return domain.ErrAlreadyLiked
		}
		return r.reviveSoftDeletedStoryLike(ctx, existing.ID, storyID)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	like := StoryLike{
		ID:        uuid.New().String(),
		UserID:    userID,
		StoryID:   storyID,
		CreatedAt: time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&like).Error; err != nil {
		if isMySQLDuplicateKeyErr(err) {
			if err := r.db.WithContext(ctx).Unscoped().
				Where("user_id = ? AND story_id = ?", userID, storyID).
				First(&existing).Error; err != nil {
				return domain.ErrAlreadyLiked
			}
			if !existing.DeletedAt.Valid {
				return domain.ErrAlreadyLiked
			}
			return r.reviveSoftDeletedStoryLike(ctx, existing.ID, storyID)
		}
		return err
	}

	return r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("likes", gorm.Expr("likes + ?", 1)).Error
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
		return domain.ErrNotFound
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
		return domain.ErrAlreadyExists
	}

	// 创建关注记录
	follow := StoryFollow{
		ID:        uuid.New().String(),
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
		return domain.ErrNotFound
	}

	// 更新故事的关注数
	if err := r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ?", storyID).
		UpdateColumn("followers", gorm.Expr("GREATEST(followers - ?, 0)", 1)).Error; err != nil {
		return err
	}

	return nil
}

// IsStoryFollowing reports whether userID has an active row in story_follows for storyID.
func (r *Repository) IsStoryFollowing(ctx context.Context, userID, storyID string) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&StoryFollow{}).
		Where("user_id = ? AND story_id = ?", userID, storyID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// CountFollowersOfStory counts active story_follows rows for the story.
func (r *Repository) CountFollowersOfStory(ctx context.Context, storyID string) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&StoryFollow{}).Where("story_id = ?", storyID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// ListStoryFollowRecordsByStory lists followers (newest first) as domain.Follow for the polymorphic /follows API.
func (r *Repository) ListStoryFollowRecordsByStory(ctx context.Context, storyID string, limit, offset int) ([]*domain.Follow, error) {
	var rows []StoryFollow
	q := r.db.WithContext(ctx).Where("story_id = ?", storyID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Follow, len(rows))
	for i := range rows {
		row := rows[i]
		out[i] = &domain.Follow{
			ID:                   row.ID,
			FollowerID:           row.UserID,
			FollowableType:       domain.FollowableTypeStory,
			FollowableID:         storyID,
			NotificationsEnabled: true,
			CreatedAt:            row.CreatedAt.Unix(),
		}
	}
	return out, nil
}

// CountStoriesFollowedByUser counts stories this user follows.
func (r *Repository) CountStoriesFollowedByUser(ctx context.Context, userID string) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&StoryFollow{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// ListStoryFollowRecordsByUser lists followed stories (newest first) as domain.Follow.
func (r *Repository) ListStoryFollowRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Follow, error) {
	var rows []StoryFollow
	q := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Follow, len(rows))
	for i := range rows {
		row := rows[i]
		out[i] = &domain.Follow{
			ID:                   row.ID,
			FollowerID:           userID,
			FollowableType:       domain.FollowableTypeStory,
			FollowableID:         row.StoryID,
			NotificationsEnabled: true,
			CreatedAt:            row.CreatedAt.Unix(),
		}
	}
	return out, nil
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
		BaseModel: common.BaseModel{
			ID:        c.ID,
			CreatedAt: c.JoinedAt.Unix(),
			UpdatedAt: c.JoinedAt.Unix(),
		},
		StoryID:    c.StoryID,
		UserID:     c.UserID,
		Role:       domain.StoryContributorRole(c.Role),
		InvitedBy:  c.InvitedBy,
		JoinedAt:   c.JoinedAt.Unix(),
		BadgeStyle: domain.StoryContributorRole(c.Role), // 使用 role 作为 badge_style
	}

	if c.User.ID != "" {
		user := r.userToDomain(c.User)
		contributor.User = &user
		// 填充扁平化字段供客户端显示
		contributor.Name = user.DisplayName
		if contributor.Name == "" {
			contributor.Name = user.Username
		}
		contributor.Avatar = user.Avatar
	}

	if c.Inviter.ID != "" {
		inviter := r.userToDomain(c.Inviter)
		contributor.Inviter = &inviter
	}

	return contributor
}
