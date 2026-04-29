package mysql

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

func (r *Repository) CreateCharacterGenerationTask(ctx context.Context, task *domain.CharacterGenerationTask) error {
	model := CharacterGenerationTaskToModel(task)
	if model.ID == "" {
		model.ID = uuid.New().String()
	}
	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now
	if model.Status == "" {
		model.Status = domain.CharacterGenerationStatusPending
	}
	if model.CurrentStep == "" {
		model.CurrentStep = domain.CharacterGenerationStepQueued
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	*task = *ModelToCharacterGenerationTask(model)
	return nil
}

func (r *Repository) UpdateCharacterGenerationTask(ctx context.Context, task *domain.CharacterGenerationTask) error {
	model := CharacterGenerationTaskToModel(task)
	updates := map[string]interface{}{
		"character_id":  model.CharacterID,
		"status":        model.Status,
		"progress":      model.Progress,
		"current_step":  model.CurrentStep,
		"request_json":  model.RequestJSON,
		"result_json":   model.ResultJSON,
		"error_message": model.ErrorMessage,
		"updated_at":    time.Now(),
		"completed_at":  model.CompletedAt,
	}
	if err := r.db.WithContext(ctx).Model(&CharacterGenerationTask{}).
		Where("id = ?", model.ID).
		Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func (r *Repository) CharacterGenerationTaskByID(ctx context.Context, id string) (*domain.CharacterGenerationTask, error) {
	var model CharacterGenerationTask
	if err := r.db.WithContext(ctx).
		Preload("Character").
		First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToCharacterGenerationTask(&model), nil
}

func (r *Repository) LatestCharacterGenerationTaskByFragmentKey(ctx context.Context, storyID, fragmentID, fragmentCharacterKey string) (*domain.CharacterGenerationTask, error) {
	var model CharacterGenerationTask
	err := r.db.WithContext(ctx).
		Preload("Character").
		Where("story_id = ? AND source_fragment_id = ? AND source_fragment_character_key = ?", storyID, fragmentID, strings.TrimSpace(fragmentCharacterKey)).
		Order("created_at DESC").
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToCharacterGenerationTask(&model), nil
}

func (r *Repository) ListCharacterGenerationTasks(ctx context.Context, userID, status string, limit, offset int) ([]*domain.CharacterGenerationTask, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	query := r.db.WithContext(ctx).
		Preload("Character").
		Where("user_id = ?", userID)
	if s := strings.TrimSpace(status); s != "" {
		query = query.Where("status = ?", s)
	}
	var models []CharacterGenerationTask
	if err := query.Order("updated_at DESC, created_at DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.CharacterGenerationTask, 0, len(models))
	for i := range models {
		out = append(out, ModelToCharacterGenerationTask(&models[i]))
	}
	return out, nil
}
