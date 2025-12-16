package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// CreateCharacterPoster 创建角色海报
func (r *Repository) CreateCharacterPoster(ctx context.Context, poster *domain.CharacterPoster) error {
	status := string(poster.Status)
	if status == "" {
		status = string(domain.PosterStatusDraft)
	}

	dbPoster := &CharacterPoster{
		ID:                    uuid.New().String(),
		CharacterID:           poster.CharacterID,
		AuthorID:              poster.Author.ID,
		Type:                  poster.Type,
		Title:                 poster.Title,
		Image:                 poster.Image,
		Video:                 poster.Video,
		Thumbnail:             poster.Thumbnail,
		Duration:              poster.Duration,
		Prompt:                poster.Prompt,
		Status:                status,
		ReferenceStoryEnabled: poster.ReferenceStoryEnabled,
		PosterConceptJSON:     poster.PosterConceptJSON,
		FinalImagePrompt:      poster.FinalImagePrompt,
		ErrorMessage:          poster.ErrorMessage,
		ConceptGenerationID:   poster.ConceptGenerationID,
		ImageGenerationID:     poster.ImageGenerationID,
		Likes:                 0,
		Shares:                0,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if dbPoster.Type == "" {
		dbPoster.Type = "image"
	}

	if err := r.db.WithContext(ctx).Create(dbPoster).Error; err != nil {
		return err
	}

	poster.ID = dbPoster.ID
	poster.Status = domain.PosterStatus(dbPoster.Status)
	poster.CreatedAt = dbPoster.CreatedAt.Unix()
	poster.UpdatedAt = dbPoster.UpdatedAt.Unix()
	return nil
}

// CharacterPosterByID 根据 ID 获取海报
func (r *Repository) CharacterPosterByID(ctx context.Context, id string) (*domain.CharacterPoster, error) {
	var poster CharacterPoster
	if err := r.db.WithContext(ctx).
		Preload("Author").
		Preload("Character").
		First(&poster, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return r.characterPosterToDomain(&poster), nil
}

// CharacterPostersByCharacterID 获取角色的所有海报
func (r *Repository) CharacterPostersByCharacterID(ctx context.Context, characterID string, limit, offset int) ([]*domain.CharacterPoster, error) {
	var posters []CharacterPoster
	if err := r.db.WithContext(ctx).
		Preload("Author").
		Where("character_id = ?", characterID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posters).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.CharacterPoster, len(posters))
	for i, p := range posters {
		result[i] = r.characterPosterToDomain(&p)
	}
	return result, nil
}

// UpdateCharacterPoster 更新海报
func (r *Repository) UpdateCharacterPoster(ctx context.Context, poster *domain.CharacterPoster) error {
	updates := map[string]interface{}{
		"title":                   poster.Title,
		"image":                   poster.Image,
		"video":                   poster.Video,
		"thumbnail":               poster.Thumbnail,
		"duration":                poster.Duration,
		"prompt":                  poster.Prompt,
		"status":                  string(poster.Status),
		"reference_story_enabled": poster.ReferenceStoryEnabled,
		"poster_concept_json":     poster.PosterConceptJSON,
		"final_image_prompt":      poster.FinalImagePrompt,
		"error_message":           poster.ErrorMessage,
		"concept_generation_id":   poster.ConceptGenerationID,
		"image_generation_id":     poster.ImageGenerationID,
		"likes":                   poster.Likes,
		"shares":                  poster.Shares,
		"updated_at":              time.Now(),
	}

	return r.db.WithContext(ctx).
		Model(&CharacterPoster{}).
		Where("id = ?", poster.ID).
		Updates(updates).Error
}

// DeleteCharacterPoster 删除海报
func (r *Repository) DeleteCharacterPoster(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&CharacterPoster{}, "id = ?", id).Error
}

// IncrementPosterLikes 增加海报点赞数
func (r *Repository) IncrementPosterLikes(ctx context.Context, posterID string) error {
	return r.db.WithContext(ctx).
		Model(&CharacterPoster{}).
		Where("id = ?", posterID).
		UpdateColumn("likes", gorm.Expr("likes + ?", 1)).Error
}

// IncrementPosterShares 增加海报分享数
func (r *Repository) IncrementPosterShares(ctx context.Context, posterID string) error {
	return r.db.WithContext(ctx).
		Model(&CharacterPoster{}).
		Where("id = ?", posterID).
		UpdateColumn("shares", gorm.Expr("shares + ?", 1)).Error
}

// characterPosterToDomain 转换海报到 domain
func (r *Repository) characterPosterToDomain(poster *CharacterPoster) *domain.CharacterPoster {
	result := &domain.CharacterPoster{
		ID:                    poster.ID,
		CharacterID:           poster.CharacterID,
		Type:                  poster.Type,
		Title:                 poster.Title,
		Image:                 poster.Image,
		Video:                 poster.Video,
		Thumbnail:             poster.Thumbnail,
		Duration:              poster.Duration,
		Prompt:                poster.Prompt,
		Status:                domain.PosterStatus(poster.Status),
		ReferenceStoryEnabled: poster.ReferenceStoryEnabled,
		PosterConceptJSON:     poster.PosterConceptJSON,
		FinalImagePrompt:      poster.FinalImagePrompt,
		ErrorMessage:          poster.ErrorMessage,
		ConceptGenerationID:   poster.ConceptGenerationID,
		ImageGenerationID:     poster.ImageGenerationID,
		Likes:                 poster.Likes,
		Shares:                poster.Shares,
		CreatedAt:             poster.CreatedAt.Unix(),
		UpdatedAt:             poster.UpdatedAt.Unix(),
	}

	if poster.Author.ID != "" {
		result.Author = r.userToDomainPtr(&poster.Author)
	}

	if poster.Character.ID != "" {
		domainChar := r.characterToDomain(poster.Character)
		result.Character = &domainChar
	}

	return result
}
