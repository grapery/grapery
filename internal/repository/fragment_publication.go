package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FragmentPublicationSnapshot struct {
	Task                            *domain.FragmentGenerationTask
	Draft                           *domain.Fragment
	TaskResultJSON, TaskRequestJSON string
	Metadata, MediaJSON, Content    string
}

func (r *FragmentGenerationRepository) PublicationSnapshot(ctx context.Context, taskID, userID string) (*FragmentPublicationSnapshot, error) {
	var snapshot FragmentPublicationSnapshot
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row mysql.FragmentGenerationTaskDB
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", taskID, userID).First(&row).Error; err != nil {
			return err
		}
		task, err := taskFromDB(&row)
		if err != nil {
			return err
		}
		if task.Status != "completed" || task.Result == nil || task.Result.DraftFragmentID == "" {
			return fmt.Errorf("only completed rendered tasks can prepare publication")
		}
		if task.Request.BillingVersion >= 1 && (task.Result.GenerationTrace == nil || task.Result.GenerationTrace.BillingStatus != "settled") {
			return fmt.Errorf("generation bill is not settled")
		}
		var draft mysql.FragmentDB
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND creator_id = ? AND is_draft = ? AND generation_task_id = ?", task.Result.DraftFragmentID, userID, true, taskID).First(&draft).Error; err != nil {
			return err
		}
		snapshot = FragmentPublicationSnapshot{Task: task, Draft: mysql.FragmentDBToDomain(&draft), TaskResultJSON: row.ResultJSON, TaskRequestJSON: row.RequestJSON, Metadata: draft.GenerationMetadata, MediaJSON: draft.ImageUrls, Content: draft.Content}
		return nil
	})
	return &snapshot, err
}

// CommitPreparedPublication compares both authority records. A late visual audit
// cannot replace newer generated/edit state, another audit, or a published draft.
func (r *FragmentGenerationRepository) CommitPreparedPublication(ctx context.Context, snapshot *FragmentPublicationSnapshot, metadata string, result *domain.FragmentGenerationResult, acceptedURLs []string) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updated := tx.Model(&mysql.FragmentGenerationTaskDB{}).Where("id = ? AND user_id = ? AND status = ? AND request_json = ? AND result_json = ?", snapshot.Task.ID, snapshot.Task.UserID, "completed", snapshot.TaskRequestJSON, snapshot.TaskResultJSON).Updates(map[string]any{"result_json": string(raw), "tokens_used": result.TokensUsed, "updated_at": time.Now().Unix()})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("task changed during publication audit")
		}
		changes := map[string]any{"generation_metadata": metadata, "updated_at": time.Now().UnixMilli()}
		if acceptedURLs != nil {
			media, _ := json.Marshal(acceptedURLs)
			changes["image_urls"] = string(media)
		}
		updated = tx.Model(&mysql.FragmentDB{}).Where("id = ? AND creator_id = ? AND is_draft = ? AND generation_task_id = ? AND generation_metadata = ? AND image_urls = ? AND content = ?", snapshot.Draft.ID, snapshot.Task.UserID, true, snapshot.Task.ID, snapshot.Metadata, snapshot.MediaJSON, snapshot.Content).Updates(changes)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("draft changed during publication audit")
		}
		return nil
	})
}
