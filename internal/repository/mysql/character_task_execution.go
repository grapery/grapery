package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) ClaimCharacterTask(ctx context.Context, id, owner string, ttl time.Duration) (*domain.CharacterGenerationTask, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&CharacterGenerationTask{}).
		Where("id = ? AND (status = ? OR (status = ? AND (lease_expires_at IS NULL OR lease_expires_at <= ?)))", id, domain.CharacterGenerationStatusPending, domain.CharacterGenerationStatusProcessing, now).
		Updates(map[string]any{"status": domain.CharacterGenerationStatusProcessing, "lease_owner": owner, "lease_fence": gorm.Expr("lease_fence + 1"), "lease_expires_at": now.Add(ttl), "updated_at": now})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, domain.ErrCharacterTaskLeaseLost
	}
	task, err := r.CharacterGenerationTaskByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.LeaseOwner != owner {
		return nil, domain.ErrCharacterTaskLeaseLost
	}
	return task, nil
}

func (r *Repository) RenewCharacterTask(ctx context.Context, task *domain.CharacterGenerationTask, ttl time.Duration) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&CharacterGenerationTask{}).
		Where("id = ? AND status = ? AND lease_owner = ? AND lease_fence = ? AND lease_expires_at > ?", task.ID, domain.CharacterGenerationStatusProcessing, task.LeaseOwner, task.LeaseFence, now).
		Updates(map[string]any{"lease_expires_at": now.Add(ttl)})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrCharacterTaskLeaseLost
	}
	return nil
}

func (r *Repository) RecoverableCharacterTasks(ctx context.Context, limit int) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&CharacterGenerationTask{}).
		Where("status = ? OR (status = ? AND (lease_expires_at IS NULL OR lease_expires_at <= ?))", domain.CharacterGenerationStatusPending, domain.CharacterGenerationStatusProcessing, time.Now()).
		Order("created_at ASC").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

func (r *Repository) QueueCharacterTaskRetry(ctx context.Context, id, userID string) (bool, error) {
	result := r.db.WithContext(ctx).Model(&CharacterGenerationTask{}).
		Where("id = ? AND user_id = ? AND status IN ?", id, userID, []string{domain.CharacterGenerationStatusFailed, domain.CharacterGenerationStatusPartial}).
		Updates(map[string]any{"status": domain.CharacterGenerationStatusPending, "completed_at": nil, "error_message": "", "current_step": domain.CharacterGenerationStepQueued, "updated_at": time.Now()})
	return result.RowsAffected == 1, result.Error
}

func (r *Repository) withCharacterTaskLease(ctx context.Context, task *domain.CharacterGenerationTask, write func(*Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model CharacterGenerationTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", task.ID).First(&model).Error; err != nil {
			return err
		}
		if model.Status != domain.CharacterGenerationStatusProcessing || model.LeaseOwner != task.LeaseOwner || model.LeaseFence != task.LeaseFence || model.LeaseExpiresAt == nil || !model.LeaseExpiresAt.After(time.Now()) {
			return domain.ErrCharacterTaskLeaseLost
		}
		scoped := *r
		scoped.db = tx
		return write(&scoped)
	})
}

func (r *Repository) CreateCharacterForTask(ctx context.Context, task *domain.CharacterGenerationTask, ch *domain.Character) error {
	return r.withCharacterTaskLease(ctx, task, func(scoped *Repository) error {
		if err := scoped.CreateCharacter(ctx, ch); err != nil {
			return err
		}
		return scoped.db.WithContext(ctx).Model(&CharacterGenerationTask{}).Where("id = ?", task.ID).Update("character_id", ch.ID).Error
	})
}

func (r *Repository) UpdateCharacterForTask(ctx context.Context, task *domain.CharacterGenerationTask, ch *domain.Character) error {
	return r.withCharacterTaskLease(ctx, task, func(scoped *Repository) error { return scoped.UpdateCharacter(ctx, ch) })
}

func characterTaskStageJSON(stages map[string]domain.CharacterGenerationStage) string {
	raw, _ := json.Marshal(stages)
	return string(raw)
}

func (r *Repository) UpdateCharacterVisualAssets(ctx context.Context, lease *domain.CharacterGenerationTask, ch *domain.Character, fields map[string]any) error {
	allowed := map[string]bool{"portrait": true, "portrait_generation_status": true, "avatar": true, "poster": true, "views_json": true, "last_edited_by": true}
	patch := map[string]any{"updated_at": time.Now()}
	for key, value := range fields {
		if !allowed[key] {
			return fmt.Errorf("unsupported character visual field %s", key)
		}
		patch[key] = value
	}
	write := func(scoped *Repository) error {
		var current Character
		if err := scoped.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", ch.ID).First(&current).Error; err != nil {
			return err
		}
		// Fallback avatar/poster fills must preserve a user edit made during generation.
		if current.Avatar != "" {
			delete(patch, "avatar")
		}
		if current.Poster != "" {
			delete(patch, "poster")
		}
		if err := scoped.db.WithContext(ctx).Model(&Character{}).Where("id = ?", ch.ID).Updates(patch).Error; err != nil {
			return err
		}
		updated, err := scoped.CharacterByID(ctx, ch.ID)
		if err != nil {
			return err
		}
		*ch = *updated
		return nil
	}
	if lease != nil {
		return r.withCharacterTaskLease(ctx, lease, write)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { scoped := *r; scoped.db = tx; return write(&scoped) })
}
