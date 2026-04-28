package mysql

import (
	"context"
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

func (r *Repository) CreateStoryboardGenerationRun(ctx context.Context, run *domain.StoryboardGenerationRun) error {
	return r.db.WithContext(ctx).Create(storyboardGenerationRunToModel(run)).Error
}

func (r *Repository) UpdateStoryboardGenerationRun(ctx context.Context, run *domain.StoryboardGenerationRun) error {
	return r.db.WithContext(ctx).Save(storyboardGenerationRunToModel(run)).Error
}

func (r *Repository) GetStoryboardGenerationRun(ctx context.Context, runID string) (*domain.StoryboardGenerationRun, error) {
	var model StoryboardGenerationRun
	if err := r.db.WithContext(ctx).Where("id = ?", runID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return modelToStoryboardGenerationRun(&model), nil
}

func (r *Repository) LatestStoryboardGenerationRun(ctx context.Context, storyboardID string) (*domain.StoryboardGenerationRun, error) {
	var model StoryboardGenerationRun
	if err := r.db.WithContext(ctx).Where("storyboard_id = ?", storyboardID).Order("created_at DESC, id DESC").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return modelToStoryboardGenerationRun(&model), nil
}

func (r *Repository) CreateStoryboardGenerationAsset(ctx context.Context, asset *domain.StoryboardGenerationAsset) error {
	return r.db.WithContext(ctx).Create(storyboardGenerationAssetToModel(asset)).Error
}

func (r *Repository) CreateStoryboardGenerationAssets(ctx context.Context, assets []*domain.StoryboardGenerationAsset) error {
	if len(assets) == 0 {
		return nil
	}
	models := make([]StoryboardGenerationAsset, 0, len(assets))
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		models = append(models, *storyboardGenerationAssetToModel(asset))
	}
	if len(models) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&models).Error
}

func (r *Repository) ListStoryboardGenerationAssets(ctx context.Context, runID string) ([]*domain.StoryboardGenerationAsset, error) {
	var models []StoryboardGenerationAsset
	if err := r.db.WithContext(ctx).Where("run_id = ?", runID).Order("created_at ASC, id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.StoryboardGenerationAsset, len(models))
	for i := range models {
		out[i] = modelToStoryboardGenerationAsset(&models[i])
	}
	return out, nil
}

func (r *Repository) CreateAIPromptAuditRecord(ctx context.Context, record *domain.AIPromptAuditRecord) error {
	return r.db.WithContext(ctx).Create(aiPromptAuditRecordToModel(record)).Error
}

func (r *Repository) ListAIPromptAuditRecords(ctx context.Context, runID string) ([]*domain.AIPromptAuditRecord, error) {
	var models []AIPromptAuditRecord
	if err := r.db.WithContext(ctx).Where("run_id = ?", runID).Order("created_at ASC, id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.AIPromptAuditRecord, len(models))
	for i := range models {
		out[i] = modelToAIPromptAuditRecord(&models[i])
	}
	return out, nil
}
