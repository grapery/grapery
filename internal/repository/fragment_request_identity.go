package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The existing task primary key is the database arbiter across server instances;
// no separate idempotency table or best-effort in-process lock is required.
func FragmentGenerationRequestTaskID(userID, clientMessageID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("fragment-request:"+userID+"\x00"+clientMessageID)).String()
}

// AttachDraftForTask checks the current owner and active generation while the
// draft row is locked. It patches only generation ownership, never a stale copy
// of user-edited text, media, publication metadata or visibility.
func (r *FragmentGenerationRepository) AttachDraftForTask(ctx context.Context, taskID, draftID, userID, aspectRatio string, replaceIndex int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var draft mysql.FragmentDB
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", draftID).First(&draft).Error; err != nil {
			return domain.NewFragmentGenerationError(domain.FragmentGenerationErrorNotFound, "target draft not found", err)
		}
		if draft.UserID != userID || !draft.IsDraft {
			return domain.NewFragmentGenerationError(domain.FragmentGenerationErrorForbidden, "target draft is not writable", nil)
		}
		var urls []string
		_ = json.Unmarshal([]byte(draft.ImageUrls), &urls)
		if replaceIndex > len(urls) || (replaceIndex > 0 && urls[replaceIndex-1] == "") {
			return domain.NewFragmentGenerationError(domain.FragmentGenerationErrorInvalidRequest, "replacement page is unavailable", nil)
		}
		previousID := draft.GenerationTaskID
		if previousID == "" && draft.SourceType == string(domain.FragmentSourceAIGeneration) {
			previousID = draft.SourceID
		}
		if previousID != "" && previousID != taskID {
			var metadata struct {
				BillingStatus string `json:"billingStatus"`
			}
			if json.Unmarshal([]byte(draft.GenerationMetadata), &metadata) == nil && metadata.BillingStatus == "pending" {
				return domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "settle the previous generation before editing this draft", nil)
			}
			var previous mysql.FragmentGenerationTaskDB
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", previousID).First(&previous).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if err == nil && (previous.Status == "pending" || previous.Status == "processing") {
				return domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "the draft already has an active generation task; resume its existing task", nil)
			}
			var result domain.FragmentGenerationResult
			if err == nil && json.Unmarshal([]byte(previous.ResultJSON), &result) == nil && result.GenerationTrace != nil && result.GenerationTrace.BillingStatus == "pending" {
				return domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "settle the previous generation before editing this draft", nil)
			}
		}
		updates := map[string]any{"generation_task_id": taskID, "updated_at": time.Now().UnixMilli()}
		// A page replacement inherits the existing canvas. Overwriting it here
		// would corrupt the very edit baseline loaded by the worker afterwards.
		if replaceIndex == 0 && aspectRatio != "" {
			updates["aspect_ratio"] = aspectRatio
		}
		if draft.SourceType == "" {
			updates["source_type"] = string(domain.FragmentSourceAIGeneration)
		}
		changed := tx.Model(&mysql.FragmentDB{}).Where("id = ? AND creator_id = ? AND is_draft = ?", draftID, userID, true).Updates(updates)
		if changed.Error != nil {
			return changed.Error
		}
		if changed.RowsAffected != 1 {
			return fmt.Errorf("draft generation attachment did not persist")
		}
		return nil
	})
}
