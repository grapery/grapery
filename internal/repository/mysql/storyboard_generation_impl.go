package mysql

import (
	"context"
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ========== Content Generation (Step 1) ==========

func (r *Repository) CreateContentGeneration(ctx context.Context, gen *domain.StoryboardContentGeneration) error {
	model := StoryboardContentGenerationToModel(gen)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *Repository) GetContentGeneration(ctx context.Context, id string) (*domain.StoryboardContentGeneration, error) {
	var model StoryboardContentGeneration
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToStoryboardContentGeneration(&model), nil
}

func (r *Repository) GetContentGenerationByStoryboard(ctx context.Context, storyboardID string) (*domain.StoryboardContentGeneration, error) {
	var model StoryboardContentGeneration
	if err := r.db.WithContext(ctx).Where("storyboard_id = ?", storyboardID).Order("created_at DESC").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToStoryboardContentGeneration(&model), nil
}

func (r *Repository) UpdateContentGeneration(ctx context.Context, gen *domain.StoryboardContentGeneration) error {
	model := StoryboardContentGenerationToModel(gen)
	return r.db.WithContext(ctx).Save(model).Error
}

// ========== Scene Generation (Step 2) ==========

func (r *Repository) CreateSceneGeneration(ctx context.Context, gen *domain.StoryboardSceneGeneration) error {
	model := StoryboardSceneGenerationToModel(gen)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *Repository) GetSceneGeneration(ctx context.Context, id string) (*domain.StoryboardSceneGeneration, error) {
	var model StoryboardSceneGeneration
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToStoryboardSceneGeneration(&model), nil
}

func (r *Repository) ListSceneGenerations(ctx context.Context, storyboardID string) ([]*domain.StoryboardSceneGeneration, error) {
	var models []StoryboardSceneGeneration
	if err := r.db.WithContext(ctx).Where("storyboard_id = ?", storyboardID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.StoryboardSceneGeneration, len(models))
	for i, m := range models {
		result[i] = ModelToStoryboardSceneGeneration(&m)
	}
	return result, nil
}

func (r *Repository) UpdateSceneGeneration(ctx context.Context, gen *domain.StoryboardSceneGeneration) error {
	model := StoryboardSceneGenerationToModel(gen)
	return r.db.WithContext(ctx).Save(model).Error
}

// ========== Image Generation (Step 3) ==========

func (r *Repository) CreateImageGeneration(ctx context.Context, gen *domain.StoryboardImageGeneration) error {
	model := StoryboardImageGenerationToModel(gen)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *Repository) GetImageGeneration(ctx context.Context, id string) (*domain.StoryboardImageGeneration, error) {
	var model StoryboardImageGeneration
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToStoryboardImageGeneration(&model), nil
}

func (r *Repository) ListImageGenerations(ctx context.Context, storyboardID string) ([]*domain.StoryboardImageGeneration, error) {
	var models []StoryboardImageGeneration
	if err := r.db.WithContext(ctx).Where("storyboard_id = ?", storyboardID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.StoryboardImageGeneration, len(models))
	for i, m := range models {
		result[i] = ModelToStoryboardImageGeneration(&m)
	}
	return result, nil
}

func (r *Repository) UpdateImageGeneration(ctx context.Context, gen *domain.StoryboardImageGeneration) error {
	model := StoryboardImageGenerationToModel(gen)
	return r.db.WithContext(ctx).Save(model).Error
}

// ========== Video Generation (Step 4) ==========

func (r *Repository) CreateVideoGeneration(ctx context.Context, gen *domain.StoryboardVideoGeneration) error {
	model := StoryboardVideoGenerationToModel(gen)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *Repository) GetVideoGeneration(ctx context.Context, id string) (*domain.StoryboardVideoGeneration, error) {
	var model StoryboardVideoGeneration
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToStoryboardVideoGeneration(&model), nil
}

func (r *Repository) ListVideoGenerations(ctx context.Context, storyboardID string) ([]*domain.StoryboardVideoGeneration, error) {
	var models []StoryboardVideoGeneration
	if err := r.db.WithContext(ctx).Where("storyboard_id = ?", storyboardID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.StoryboardVideoGeneration, len(models))
	for i, m := range models {
		result[i] = ModelToStoryboardVideoGeneration(&m)
	}
	return result, nil
}

func (r *Repository) UpdateVideoGeneration(ctx context.Context, gen *domain.StoryboardVideoGeneration) error {
	model := StoryboardVideoGenerationToModel(gen)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *Repository) ListPendingVideoGenerations(ctx context.Context) ([]*domain.StoryboardVideoGeneration, error) {
	var models []StoryboardVideoGeneration
	// Find all video generations that are processing and have a provider task ID
	err := r.db.WithContext(ctx).
		Where("status = ? AND provider_task_id != '' AND provider_task_id IS NOT NULL", domain.GenerationStatusProcessing).
		Order("created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domain.StoryboardVideoGeneration, len(models))
	for i, m := range models {
		result[i] = ModelToStoryboardVideoGeneration(&m)
	}
	return result, nil
}

// ========== Token Aggregation ==========

func (r *Repository) GetStoryboardTotalTokens(ctx context.Context, storyboardID string) (int, error) {
	var total int

	// Sum tokens from content generation
	var contentTokens int
	r.db.WithContext(ctx).Model(&StoryboardContentGeneration{}).
		Where("storyboard_id = ?", storyboardID).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&contentTokens)
	total += contentTokens

	// Sum tokens from scene generations
	var sceneTokens int
	r.db.WithContext(ctx).Model(&StoryboardSceneGeneration{}).
		Where("storyboard_id = ?", storyboardID).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&sceneTokens)
	total += sceneTokens

	// Sum tokens from image generations
	var imageTokens int
	r.db.WithContext(ctx).Model(&StoryboardImageGeneration{}).
		Where("storyboard_id = ?", storyboardID).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&imageTokens)
	total += imageTokens

	// Sum tokens from video generations
	var videoTokens int
	r.db.WithContext(ctx).Model(&StoryboardVideoGeneration{}).
		Where("storyboard_id = ?", storyboardID).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&videoTokens)
	total += videoTokens

	return total, nil
}

func (r *Repository) UpdateStoryboardTokens(ctx context.Context, storyboardID string, tokens int) error {
	return r.db.WithContext(ctx).Model(&Storyboard{}).
		Where("id = ?", storyboardID).
		Update("token_consumption", tokens).Error
}

func (r *Repository) UpdateStoryboardWorkflow(ctx context.Context, storyboardID string, status string, step int) error {
	updates := map[string]interface{}{
		"workflow_status": status,
		"current_step":    step,
	}
	return r.db.WithContext(ctx).Model(&Storyboard{}).
		Where("id = ?", storyboardID).
		Updates(updates).Error
}

