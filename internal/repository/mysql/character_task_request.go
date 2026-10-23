package mysql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) CreateCharacterTaskIdempotent(ctx context.Context, task *domain.CharacterGenerationTask) (*domain.CharacterGenerationTask, bool, error) {
	if task == nil || task.RequestKey == "" || task.IntentHash == "" {
		return nil, false, domain.ErrInvalidInput
	}
	model := CharacterGenerationTaskToModel(task)
	if model.ID == "" {
		model.ID = uuid.NewString()
	}
	model.CreatedAt = time.Now()
	model.UpdatedAt = model.CreatedAt
	var stored *domain.CharacterGenerationTask
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(model)
		if result.Error != nil {
			return result.Error
		}
		var saved CharacterGenerationTask
		// The unique key arbitrates concurrent inserts. A locking read also sees
		// a winner committed after this transaction began under MySQL RR isolation.
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_key = ?", task.RequestKey).First(&saved).Error; err != nil {
			return err
		}
		if saved.UserID != task.UserID || saved.IntentHash != task.IntentHash {
			return domain.ErrCharacterTaskRequestConflict
		}
		if saved.DeletedAt.Valid {
			return domain.ErrCharacterTaskRequestConflict
		}
		stored = ModelToCharacterGenerationTask(&saved)
		created = saved.ID == model.ID
		return nil
	})
	return stored, created, err
}

// Character task requests validate the agent envelope using the same runtime
// storage as the HTTP generation service (without accepting client assertions).
func (r *Repository) GetGenerationExecution(ctx context.Context, id string) (*domain.GenerationExecution, error) {
	return NewGenerationRuntimeRepository(r.db).GetGenerationExecution(ctx, id)
}

func migrateCharacterTaskRequestIdentity(ctx context.Context, db *gorm.DB) error {
	migrator := db.WithContext(ctx).Migrator()
	for _, field := range []string{"RequestKey", "IntentHash"} {
		if !migrator.HasColumn(&CharacterGenerationTask{}, field) {
			if err := migrator.AddColumn(&CharacterGenerationTask{}, field); err != nil {
				return err
			}
		}
	}
	if !migrator.HasIndex(&CharacterGenerationTask{}, "idx_character_task_request_key") {
		return migrator.CreateIndex(&CharacterGenerationTask{}, "idx_character_task_request_key")
	}
	return nil
}
