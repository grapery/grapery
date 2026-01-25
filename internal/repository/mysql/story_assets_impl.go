package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// CreateStoryScene stores a story-scoped scene asset.
func (r *Repository) CreateStoryScene(ctx context.Context, scene *domain.StoryScene) error {
	dbModel := &StoryScene{
		ID:           uuid.NewString(),
		StoryID:      scene.StoryID,
		Title:        scene.Title,
		Description:  scene.Description,
		Image:        scene.Image,
		Location:     scene.Location,
		TimeOfDay:    scene.TimeOfDay,
		SourceType:   scene.SourceType,
		SourcePrompt: scene.SourcePrompt,
		SourceImage:  scene.SourceImage,
		CreatedBy:    scene.CreatedBy,
		LastEditedBy: scene.LastEditedBy,
		IsPublic:     scene.IsPublic,
	}

	// Exclude Tags field from insert until database column exists
	if err := r.db.WithContext(ctx).
		Select("id", "story_id", "title", "description", "image", "location", "time_of_day",
			"source_type", "source_prompt", "source_image", "created_by", "last_edited_by",
			"is_public", "created_at", "updated_at", "deleted_at").
		Create(dbModel).Error; err != nil {
		return fmt.Errorf("create story scene: %w", err)
	}
	scene.ID = dbModel.ID
	scene.CreatedAt = dbModel.CreatedAt.Unix()
	scene.UpdatedAt = dbModel.UpdatedAt.Unix()
	return nil
}

// UpdateStoryScene updates an existing story scene.
func (r *Repository) UpdateStoryScene(ctx context.Context, scene *domain.StoryScene) error {
	update := map[string]any{
		"title":          scene.Title,
		"description":    scene.Description,
		"image":          scene.Image,
		"location":       scene.Location,
		"time_of_day":    scene.TimeOfDay,
		"source_type":    scene.SourceType,
		"source_prompt":  scene.SourcePrompt,
		"source_image":   scene.SourceImage,
		"last_edited_by": scene.LastEditedBy,
		"is_public":      scene.IsPublic,
	}
	if err := r.db.WithContext(ctx).
		Model(&StoryScene{}).
		Where("id = ? AND story_id = ?", scene.ID, scene.StoryID).
		Updates(update).Error; err != nil {
		return fmt.Errorf("update story scene: %w", err)
	}
	return nil
}

// DeleteStoryScene soft deletes a scene.
func (r *Repository) DeleteStoryScene(ctx context.Context, storyID, id string) error {
	if err := r.db.WithContext(ctx).
		Where("id = ? AND story_id = ?", id, storyID).
		Delete(&StoryScene{}).Error; err != nil {
		return fmt.Errorf("delete story scene: %w", err)
	}
	return nil
}

// StorySceneByID fetches a scene by id.
func (r *Repository) StorySceneByID(ctx context.Context, storyID, id string) (*domain.StoryScene, error) {
	var model StoryScene
	if err := r.db.WithContext(ctx).
		Where("id = ? AND story_id = ?", id, storyID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get story scene: %w", err)
	}
	return modelToStoryScene(&model), nil
}

// StoryScenes lists scenes of a story.
func (r *Repository) StoryScenes(ctx context.Context, storyID string, limit, offset int) ([]*domain.StoryScene, error) {
	var models []StoryScene
	q := r.db.WithContext(ctx).
		Where("story_id = ?", storyID).
		Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list story scenes: %w", err)
	}
	result := make([]*domain.StoryScene, len(models))
	for i := range models {
		result[i] = modelToStoryScene(&models[i])
	}
	return result, nil
}

// AttachCharactersToStoryboard replaces character links for a storyboard.
func (r *Repository) AttachCharactersToStoryboard(ctx context.Context, storyboardID string, refs []domain.StoryboardCharacterRef) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("storyboard_id = ?", storyboardID).Delete(&StoryboardCharacterLink{}).Error; err != nil {
			return err
		}
		if len(refs) == 0 {
			return nil
		}
		links := make([]StoryboardCharacterLink, len(refs))
		for i, ref := range refs {
			var charID *string
			if ref.CharacterID != "" {
				charID = &ref.CharacterID
			}
			links[i] = StoryboardCharacterLink{
				ID:           uuid.NewString(),
				StoryboardID: storyboardID,
				CharacterID:  charID, // Optional: nil for minor/background characters
				Role:         ref.Role,
				Ordering:     ref.Order,
				Notes:        ref.Notes,
			}
		}
		return tx.Create(&links).Error
	})
}

// AttachScenesToStoryboard replaces scene links for a storyboard.
func (r *Repository) AttachScenesToStoryboard(ctx context.Context, storyboardID string, refs []domain.StoryboardSceneRef) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("storyboard_id = ?", storyboardID).Delete(&StoryboardSceneLink{}).Error; err != nil {
			return err
		}
		if len(refs) == 0 {
			return nil
		}
		links := make([]StoryboardSceneLink, len(refs))
		for i, ref := range refs {
			links[i] = StoryboardSceneLink{
				ID:             uuid.NewString(),
				StoryboardID:   storyboardID,
				StorySceneID:   ref.StorySceneID,
				Sequence:       ref.Sequence,
				IsPrimaryScene: ref.IsPrimaryScene,
			}
		}
		return tx.Create(&links).Error
	})
}

// storyboardCharacterLinks loads character associations.
func (r *Repository) storyboardCharacterLinks(ctx context.Context, storyboardID string) ([]domain.StoryboardCharacterRef, error) {
	var links []StoryboardCharacterLink
	if err := r.db.WithContext(ctx).
		Preload("Character").
		Where("storyboard_id = ?", storyboardID).
		Order("ordering ASC").
		Find(&links).Error; err != nil {
		return nil, err
	}
	result := make([]domain.StoryboardCharacterRef, len(links))
	for i := range links {
		result[i] = modelToStoryboardCharacterRef(&links[i])
	}
	return result, nil
}

// storyboardSceneLinks loads scene associations.
func (r *Repository) storyboardSceneLinks(ctx context.Context, storyboardID string) ([]domain.StoryboardSceneRef, error) {
	var links []StoryboardSceneLink
	if err := r.db.WithContext(ctx).
		Preload("StoryScene").
		Where("storyboard_id = ?", storyboardID).
		Order("sequence ASC").
		Find(&links).Error; err != nil {
		return nil, err
	}
	result := make([]domain.StoryboardSceneRef, len(links))
	for i := range links {
		result[i] = modelToStoryboardSceneRef(&links[i])
	}
	return result, nil
}
