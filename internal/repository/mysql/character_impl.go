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

// CharacterByID retrieves a character by ID
func (r *Repository) CharacterByID(ctx context.Context, id string) (*domain.Character, error) {
	var character Character
	if err := r.db.WithContext(ctx).Preload("Author").First(&character, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	domainChar := r.characterToDomain(character)
	return &domainChar, nil
}

// ListCharacters retrieves all characters with pagination
func (r *Repository) ListCharacters(ctx context.Context, limit, offset int) ([]*domain.Character, error) {
	var characters []Character
	query := r.db.WithContext(ctx).Preload("Author").Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&characters).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Character, len(characters))
	for i, c := range characters {
		domainChar := r.characterToDomain(c)
		result[i] = &domainChar
	}

	return result, nil
}

// CreateCharacter creates a new character
func (r *Repository) CreateCharacter(ctx context.Context, character *domain.Character) error {
	authorID := character.UserID
	if authorID == "" && character.Author != nil {
		authorID = character.Author.ID
	}

	sourceType := character.SourceType
	if sourceType == "" {
		sourceType = "manual"
	}

	now := time.Now()

	// 设置默认的形象生成状态
	portraitStatus := character.PortraitGenerationStatus
	if portraitStatus == "" {
		if character.NeedsPortrait {
			portraitStatus = string(common.TaskStatusPending)
		} else {
			portraitStatus = "none"
		}
	}

	dbCharacter := Character{
		ID:                       uuid.New().String(),
		StoryID:                  character.StoryID,
		Name:                     character.Name,
		Description:              character.Description,
		Avatar:                   character.Avatar,
		Poster:                   character.Poster,
		Portrait:                 character.Portrait,
		NeedsPortrait:            character.NeedsPortrait,
		ReferenceImage:           character.ReferenceImage,
		PortraitGenerationStatus: portraitStatus,
		UserID:                 authorID,
		Personality:              character.Personality,
		Background:               character.Background,
		ShortTermGoal:            character.ShortTermGoal,
		LongTermGoal:             character.LongTermGoal,
		HandlingStyle:            character.HandlingStyle,
		CognitionRange:           character.CognitionRange,
		AbilityFeatures:          character.AbilityFeatures,
		Appearance:               character.Appearance,
		DressPreference:          character.DressPreference,
		SourceType:               sourceType,
		SourcePrompt:             character.SourcePrompt,
		SourceImage:              character.SourceImage,
		CreatedBy:                character.CreatedBy,
		LastEditedBy:             character.LastEditedBy,
		IsPublic:                 character.IsPublic,
		Traits:                   character.TraitsJSON,
		Skills:                   character.SkillsJSON,
		Followers:                character.Followers,
		Stories:                  character.Stories,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	if err := r.db.WithContext(ctx).Create(&dbCharacter).Error; err != nil {
		return err
	}

	// 更新传入的 character 对象的 ID
	character.ID = dbCharacter.ID
	character.CreatedAt = dbCharacter.CreatedAt.Unix()
	character.UpdatedAt = dbCharacter.UpdatedAt.Unix()
	return nil
}

// UpdateCharacter updates an existing character
func (r *Repository) UpdateCharacter(ctx context.Context, character *domain.Character) error {
	authorID := character.UserID
	if authorID == "" && character.Author != nil {
		authorID = character.Author.ID
	}
	updates := map[string]interface{}{
		"story_id":                   character.StoryID,
		"name":                       character.Name,
		"description":                character.Description,
		"avatar":                     character.Avatar,
		"poster":                     character.Poster,
		"portrait":                   character.Portrait,
		"needs_portrait":             character.NeedsPortrait,
		"reference_image":            character.ReferenceImage,
		"portrait_generation_status": character.PortraitGenerationStatus,
		"user_id":                  authorID,
		"personality":                character.Personality,
		"background":                 character.Background,
		"short_term_goal":            character.ShortTermGoal,
		"long_term_goal":             character.LongTermGoal,
		"handling_style":             character.HandlingStyle,
		"cognition_range":            character.CognitionRange,
		"ability_features":           character.AbilityFeatures,
		"appearance":                 character.Appearance,
		"dress_preference":           character.DressPreference,
		"source_type":                character.SourceType,
		"source_prompt":              character.SourcePrompt,
		"source_image":               character.SourceImage,
		"last_edited_by":             character.LastEditedBy,
		"is_public":                  character.IsPublic,
		"traits":                     character.TraitsJSON,
		"skills":                     character.SkillsJSON,
		"followers":                  character.Followers,
		"stories":                    character.Stories,
		"updated_at":                 time.Now(),
	}

	if err := r.db.WithContext(ctx).Model(&Character{}).
		Where("id = ?", character.ID).
		Updates(updates).Error; err != nil {
		return err
	}

	// 重新加载更新后的角色
	var updated Character
	if err := r.db.WithContext(ctx).Preload("Author").First(&updated, "id = ?", character.ID).Error; err != nil {
		return err
	}

	*character = r.characterToDomain(updated)
	return nil
}

// DeleteCharacter deletes a character (soft delete)
func (r *Repository) DeleteCharacter(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Character{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CharactersByStory retrieves characters for a story
func (r *Repository) CharactersByStory(ctx context.Context, storyID string) ([]*domain.Character, error) {
	var characters []Character
	if err := r.db.WithContext(ctx).
		Preload("Author").
		Where("story_id = ?", storyID).
		Order("created_at DESC").
		Find(&characters).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Character, len(characters))
	for i, c := range characters {
		char := r.characterToDomain(c)
		result[i] = &char
	}
	return result, nil
}

// CharactersByUser retrieves characters by user
func (r *Repository) CharactersByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Character, error) {
	var characters []Character
	query := r.db.WithContext(ctx).Preload("Author").Where("user_id = ?", userID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&characters).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Character, len(characters))
	for i, c := range characters {
		char := r.characterToDomain(c)
		result[i] = &char
	}
	return result, nil
}

// FollowCharacter adds a follow to a character
func (r *Repository) FollowCharacter(ctx context.Context, userID, characterID string) error {
	// 检查是否已经关注
	var count int64
	if err := r.db.WithContext(ctx).Model(&CharacterFollow{}).
		Where("user_id = ? AND character_id = ?", userID, characterID).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return domain.ErrAlreadyExists
	}

	// 创建关注记录
	follow := CharacterFollow{
		ID:          uuid.New().String(),
		UserID:      userID,
		CharacterID: characterID,
		CreatedAt:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&follow).Error; err != nil {
		// Handle MySQL duplicate entry error (Error 1062)
		if strings.Contains(err.Error(), "Error 1062") ||
			strings.Contains(err.Error(), "Duplicate entry") ||
			strings.Contains(err.Error(), "23000") {
			return domain.ErrAlreadyExists
		}
		return err
	}

	// 更新角色的关注数
	if err := r.db.WithContext(ctx).Model(&Character{}).
		Where("id = ?", characterID).
		UpdateColumn("followers", gorm.Expr("followers + ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

// UnfollowCharacter removes a follow from a character
func (r *Repository) UnfollowCharacter(ctx context.Context, userID, characterID string) error {
	// 删除关注记录
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND character_id = ?", userID, characterID).
		Delete(&CharacterFollow{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	// 更新角色的关注数
	if err := r.db.WithContext(ctx).Model(&Character{}).
		Where("id = ?", characterID).
		UpdateColumn("followers", gorm.Expr("GREATEST(followers - ?, 0)", 1)).Error; err != nil {
		return err
	}

	return nil
}

// IsCharacterFollowing checks if a user is following a character
func (r *Repository) IsCharacterFollowing(ctx context.Context, userID, characterID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&CharacterFollow{}).
		Where("user_id = ? AND character_id = ?", userID, characterID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// StoryboardsByCharacter retrieves storyboards that a character participates in
func (r *Repository) StoryboardsByCharacter(ctx context.Context, characterID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	// Get storyboard IDs from the character link table
	var storyboardIDs []string
	if err := r.db.WithContext(ctx).
		Model(&StoryboardCharacterLink{}).
		Where("character_id = ?", characterID).
		Order("created_at DESC").
		Pluck("storyboard_id", &storyboardIDs).Error; err != nil {
		return nil, 0, err
	}

	if len(storyboardIDs) == 0 {
		return []*domain.Storyboard{}, 0, nil
	}

	// Count total
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Where("id IN ?", storyboardIDs).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch storyboards with pagination
	var storyboards []Storyboard
	query := r.db.WithContext(ctx).
		Preload("Story").
		Preload("Creator").
		Where("id IN ?", storyboardIDs).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&storyboards).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*domain.Storyboard, len(storyboards))
	for i := range storyboards {
		sb, err := r.storyboardToDomain(ctx, storyboards[i])
		if err != nil {
			return nil, 0, err
		}
		result[i] = &sb
	}

	return result, total, nil
}
