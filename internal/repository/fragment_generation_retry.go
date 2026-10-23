package repository

import (
	"context"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *FragmentGenerationRepository) QueueRetry(ctx context.Context, taskID, draftID, userID, checkpoint string) error {
	conflict := func(message string) error {
		return domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, message, nil)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task mysql.FragmentGenerationTaskDB
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
			return err
		}
		var draft mysql.FragmentDB
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND creator_id = ?", draftID, userID).First(&draft).Error; err != nil {
			return conflict("original draft not found")
		}
		if !draft.IsDraft || (draft.GenerationTaskID != taskID && !(draft.GenerationTaskID == "" && draft.SourceID == taskID && draft.SourceType == string(domain.FragmentSourceAIGeneration))) {
			return conflict("the draft changed; reopen its latest task")
		}
		if task.Status == "pending" || task.Status == "processing" {
			return nil
		}
		if task.Status != "failed" || strings.HasPrefix(task.ErrorMessage, "billing_unsettled:") {
			return conflict("task is not eligible for generation retry")
		}
		if task.CheckpointJSON != checkpoint {
			return conflict("task recovery state changed; reload before retrying")
		}
		if err := tx.Model(&mysql.FragmentDB{}).Where("id = ?", draftID).Update("generation_task_id", taskID).Error; err != nil {
			return err
		}
		return tx.Model(&mysql.FragmentGenerationTaskDB{}).Where("id = ? AND status = ?", taskID, "failed").Updates(map[string]any{
			"status": "pending", "error_message": "", "completed_at": nil, "current_step": "queued_retry", "execution_version": 1, "ready_at": time.Now().UnixMilli(), "lease_owner": "", "lease_expires_at": nil, "lease_fence": gorm.Expr("COALESCE(lease_fence, 0) + 1"), "updated_at": time.Now().Unix(),
		}).Error
	})
}
