package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *FragmentGenerationRepository) FinalizeFragmentSettlement(ctx context.Context, taskID, draftID, userID, previousMetadata, metadata string, result *domain.FragmentGenerationResult) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task mysql.FragmentGenerationTaskDB
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		_, worker := domain.FragmentTaskLeaseFromContext(ctx)
		if task.Status == "processing" || worker {
			if err := mysql.ValidateFragmentTaskLease(ctx, &task); err != nil {
				return err
			}
		} else if task.Status != "failed" || !strings.HasPrefix(task.ErrorMessage, "billing_unsettled:") {
			return domain.ErrFragmentTaskLeaseLost
		}
		changed := tx.Model(&mysql.FragmentDB{}).Where("id = ? AND creator_id = ? AND is_draft = ? AND generation_task_id = ? AND generation_metadata = ?", draftID, userID, true, taskID, previousMetadata).Updates(map[string]any{"generation_metadata": metadata, "updated_at": time.Now().UnixMilli()})
		if changed.Error != nil {
			return changed.Error
		}
		if changed.RowsAffected != 1 {
			return fmt.Errorf("draft changed before settlement completed")
		}
		changed = tx.Model(&mysql.FragmentGenerationTaskDB{}).Where("id = ? AND user_id = ? AND (status = ? OR (status = ? AND error_message LIKE ?))", taskID, userID, "processing", "failed", "billing_unsettled:%").Updates(map[string]any{"result_json": string(raw), "tokens_used": result.TokensUsed, "status": "completed", "progress": 100, "current_step": "completed", "error_message": "", "completed_at": time.Now().Unix(), "updated_at": time.Now().Unix()})
		if changed.Error != nil {
			return changed.Error
		}
		if changed.RowsAffected != 1 {
			return fmt.Errorf("task is no longer eligible for settlement")
		}
		return nil
	})
}
