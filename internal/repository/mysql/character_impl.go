package mysql

import (
	"context"
	"encoding/json"
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
		UserID:                   authorID,
		Personality:              character.Personality,
		Background:               character.Background,
		ShortTermGoal:            character.ShortTermGoal,
		LongTermGoal:             character.LongTermGoal,
		HandlingStyle:            character.HandlingStyle,
		CognitionRange:           character.CognitionRange,
		AbilityFeatures:          character.AbilityFeatures,
		Appearance:               character.Appearance,
		DressPreference:          character.DressPreference,
		Role:                     character.Role,
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
	// MySQL JSON 列不接受空字符串；无三视图数据时用 "{}"。
	dbCharacter.ViewsJSON = "{}"
	if character.Views != nil {
		if b, err := json.Marshal(character.Views); err == nil && len(strings.TrimSpace(string(b))) > 0 {
			dbCharacter.ViewsJSON = string(b)
		}
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
		"author_id":                  authorID, // DB column is author_id (see models.Character.UserID)
		"personality":                character.Personality,
		"background":                 character.Background,
		"short_term_goal":            character.ShortTermGoal,
		"long_term_goal":             character.LongTermGoal,
		"handling_style":             character.HandlingStyle,
		"cognition_range":            character.CognitionRange,
		"ability_features":           character.AbilityFeatures,
		"appearance":                 character.Appearance,
		"dress_preference":           character.DressPreference,
		"role":                       character.Role,
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
	if character.Views != nil {
		if b, err := json.Marshal(character.Views); err == nil {
			updates["views_json"] = string(b)
		}
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
	query := r.db.WithContext(ctx).Preload("Author").Where("author_id = ?", userID).Order("created_at DESC")

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

// CountFollowersOfCharacter counts character_follows rows for the character.
func (r *Repository) CountFollowersOfCharacter(ctx context.Context, characterID string) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&CharacterFollow{}).Where("character_id = ?", characterID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// ListCharacterFollowRecordsByCharacter lists followers (newest first) as domain.Follow.
func (r *Repository) ListCharacterFollowRecordsByCharacter(ctx context.Context, characterID string, limit, offset int) ([]*domain.Follow, error) {
	var rows []CharacterFollow
	q := r.db.WithContext(ctx).Where("character_id = ?", characterID).Order("created_at DESC")
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
			FollowableType:       domain.FollowableTypeCharacter,
			FollowableID:         characterID,
			NotificationsEnabled: true,
			CreatedAt:            row.CreatedAt.Unix(),
		}
	}
	return out, nil
}

// CountCharactersFollowedByUser counts characters this user follows.
func (r *Repository) CountCharactersFollowedByUser(ctx context.Context, userID string) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&CharacterFollow{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// ListCharacterFollowRecordsByUser lists followed characters (newest first) as domain.Follow.
func (r *Repository) ListCharacterFollowRecordsByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Follow, error) {
	var rows []CharacterFollow
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
			FollowableType:       domain.FollowableTypeCharacter,
			FollowableID:         row.CharacterID,
			NotificationsEnabled: true,
			CreatedAt:            row.CreatedAt.Unix(),
		}
	}
	return out, nil
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

// ========== Character View operations (Three-view generation) ==========

// CharacterView model for GORM
type CharacterView struct {
	common.BaseModel
	CharacterID   string `gorm:"index"`
	ViewType      string `gorm:"size:20"` // front, side, back
	ImageURL      string `gorm:"size:500"`
	IsAIGenerated bool
	Prompt        string `gorm:"size:2000"`
	Status        string `gorm:"size:20"` // pending, generating, completed, failed
	ErrorMessage  string `gorm:"size:500"`
}

// TableName specifies the table name for CharacterView
func (CharacterView) TableName() string {
	return "character_views"
}

// CreateCharacterView creates a new character view
func (r *Repository) CreateCharacterView(ctx context.Context, view *domain.CharacterView) error {
	cv := r.domainToCharacterView(*view)
	return r.db.WithContext(ctx).Create(&cv).Error
}

// CharacterViewByID retrieves a character view by ID
func (r *Repository) CharacterViewByID(ctx context.Context, id string) (*domain.CharacterView, error) {
	var cv CharacterView
	if err := r.db.WithContext(ctx).First(&cv, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	domainView := r.characterViewToDomain(cv)
	return &domainView, nil
}

// GetCharacterViewsByCharacterID retrieves all views for a character
func (r *Repository) GetCharacterViewsByCharacterID(ctx context.Context, characterID string) ([]domain.CharacterView, error) {
	var views []CharacterView
	if err := r.db.WithContext(ctx).
		Where("character_id = ?", characterID).
		Order("created_at DESC").
		Find(&views).Error; err != nil {
		return nil, err
	}

	result := make([]domain.CharacterView, len(views))
	for i, v := range views {
		result[i] = r.characterViewToDomain(v)
	}
	return result, nil
}

// UpdateCharacterViewStatus updates the status and image URL of a character view
func (r *Repository) UpdateCharacterViewStatus(ctx context.Context, viewID string, status domain.CharacterViewStatus, imageURL string) error {
	updates := map[string]interface{}{
		"status":     string(status),
		"updated_at": time.Now().Unix(),
	}
	if imageURL != "" {
		updates["image_url"] = imageURL
	}
	return r.db.WithContext(ctx).
		Model(&CharacterView{}).
		Where("id = ?", viewID).
		Updates(updates).Error
}

// DeleteCharacterView deletes a character view
func (r *Repository) DeleteCharacterView(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&CharacterView{}, "id = ?", id).Error
}

// domainToCharacterView converts domain.CharacterView to CharacterView model
func (r *Repository) domainToCharacterView(v domain.CharacterView) CharacterView {
	return CharacterView{
		BaseModel: common.BaseModel{
			ID:        v.ID,
			CreatedAt: v.CreatedAt,
			UpdatedAt: v.UpdatedAt,
		},
		CharacterID:   v.CharacterID,
		ViewType:      string(v.ViewType),
		ImageURL:      v.ImageURL,
		IsAIGenerated: v.IsAIGenerated,
		Prompt:        v.Prompt,
		Status:        string(v.Status),
		ErrorMessage:  v.ErrorMessage,
	}
}

// characterViewToDomain converts CharacterView model to domain.CharacterView
func (r *Repository) characterViewToDomain(cv CharacterView) domain.CharacterView {
	return domain.CharacterView{
		BaseModel:     cv.BaseModel,
		CharacterID:   cv.CharacterID,
		ViewType:      domain.CharacterViewType(cv.ViewType),
		ImageURL:      cv.ImageURL,
		IsAIGenerated: cv.IsAIGenerated,
		Prompt:        cv.Prompt,
		Status:        domain.CharacterViewStatus(cv.Status),
		ErrorMessage:  cv.ErrorMessage,
	}
}
