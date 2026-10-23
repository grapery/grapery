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
)

func (r *FragmentGenerationRepository) ListRecoverable(ctx context.Context, limit int) ([]string, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	// A process can stop between task insertion and placeholder attachment.
	// Such rows are not executable; retire abandoned setup without any model
	// call rather than leaving them to block a target draft indefinitely.
	if err := r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("status = ? AND execution_version >= 1 AND ready_at IS NULL AND COALESCE(lease_fence, 0) = 0 AND created_at <= ?", "pending", time.Now().Add(-5*time.Minute).Unix()).
		Updates(map[string]any{"status": "failed", "current_step": "preparation_incomplete", "error_message": "recovery_preparation_incomplete: request setup did not complete; retry from the saved draft", "completed_at": time.Now().Unix(), "updated_at": time.Now().Unix()}).Error; err != nil {
		return nil, err
	}
	var ids []string
	err := r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("status IN ? AND (lease_expires_at IS NULL OR lease_expires_at <= ?)", []string{"pending", "processing"}, time.Now().UnixMilli()).
		Where("COALESCE(execution_version, 0) = 0 OR ready_at IS NOT NULL").
		Order("created_at ASC, id ASC").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

// Claim never resets the persisted stage, result, checkpoints or image slots.
// Legacy processing rows have a NULL expiry and can be claimed once immediately.
func (r *FragmentGenerationRepository) Claim(ctx context.Context, id, owner string, ttl time.Duration) (*domain.FragmentGenerationTask, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(owner) == "" || len(owner) > 64 || ttl < time.Millisecond {
		return nil, fmt.Errorf("invalid fragment task lease")
	}
	var task *domain.FragmentGenerationTask
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		claimed := tx.Model(&mysql.FragmentGenerationTaskDB{}).
			Where("id = ? AND status IN ? AND (lease_expires_at IS NULL OR lease_expires_at <= ?)", id, []string{"pending", "processing"}, now.UnixMilli()).
			Where("COALESCE(execution_version, 0) = 0 OR ready_at IS NOT NULL").
			Updates(map[string]any{"status": "processing", "lease_owner": owner, "lease_fence": gorm.Expr("COALESCE(lease_fence, 0) + 1"), "lease_expires_at": now.Add(ttl).UnixMilli(), "started_at": gorm.Expr("COALESCE(started_at, ?)", now.Unix()), "updated_at": now.Unix()})
		if claimed.Error != nil {
			return claimed.Error
		}
		if claimed.RowsAffected != 1 {
			return domain.ErrFragmentTaskLeaseLost
		}
		var err error
		task, err = NewFragmentGenerationRepository(tx).GetByID(ctx, id)
		return err
	})
	return task, err
}

func (r *FragmentGenerationRepository) MarkReady(ctx context.Context, id string) error {
	changed := r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("id = ? AND status = ? AND execution_version >= 1 AND COALESCE(lease_fence, 0) = 0 AND ready_at IS NULL", id, "pending").
		Updates(map[string]any{"ready_at": time.Now().UnixMilli(), "updated_at": time.Now().Unix()})
	if changed.Error != nil {
		return changed.Error
	}
	if changed.RowsAffected != 1 {
		return domain.ErrFragmentTaskLeaseLost
	}
	return nil
}

func (r *FragmentGenerationRepository) Renew(ctx context.Context, task *domain.FragmentGenerationTask, ttl time.Duration) error {
	if task == nil || task.LeaseOwner == "" || task.LeaseFence < 1 || ttl < time.Millisecond {
		return domain.ErrFragmentTaskLeaseLost
	}
	now := time.Now()
	changed := r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("id = ? AND status = ? AND lease_owner = ? AND lease_fence = ? AND lease_expires_at > ?", task.ID, "processing", task.LeaseOwner, task.LeaseFence, now.UnixMilli()).
		Update("lease_expires_at", now.Add(ttl).UnixMilli())
	if changed.Error != nil {
		return changed.Error
	}
	if changed.RowsAffected != 1 {
		return fragmentTaskWriteResult(domain.WithFragmentTaskLease(ctx, task), task.ID, changed)
	}
	return nil
}

func (r *FragmentGenerationRepository) CheckLease(ctx context.Context, id string) error {
	var row mysql.FragmentGenerationTaskDB
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return err
	}
	return mysql.ValidateFragmentTaskLease(ctx, &row)
}

func fragmentTaskFencedQuery(ctx context.Context, query *gorm.DB, id string) *gorm.DB {
	lease, ok := domain.FragmentTaskLeaseFromContext(ctx)
	if !ok {
		return query.Where("COALESCE(lease_fence, 0) = 0 AND COALESCE(lease_owner, '') = ''")
	}
	if lease.TaskID != id || lease.Owner == "" || lease.Fence < 1 {
		return query.Where("1 = 0")
	}
	return query.Where("lease_owner = ? AND lease_fence = ? AND lease_expires_at > ?", lease.Owner, lease.Fence, time.Now().UnixMilli())
}

func fragmentTaskWriteResult(ctx context.Context, id string, changed *gorm.DB) error {
	if changed.Error != nil {
		return changed.Error
	}
	if lease, ok := domain.FragmentTaskLeaseFromContext(ctx); ok && changed.RowsAffected != 1 {
		if lease.TaskID != id {
			return domain.ErrFragmentTaskLeaseLost
		}
		// MySQL reports changed rows by default. An identical checkpoint/result
		// written in the same second is a successful no-op, not evidence that
		// the worker lost its lease. Revalidate the exact authority after a zero.
		var row mysql.FragmentGenerationTaskDB
		if err := changed.Session(&gorm.Session{NewDB: true}).WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return domain.ErrFragmentTaskLeaseLost
			}
			return err
		}
		return mysql.ValidateFragmentTaskLease(ctx, &row)
	}
	return nil
}

func (r *FragmentGenerationRepository) WithTaskLeaseTransaction(ctx context.Context, id string, write func(*gorm.DB) error) error {
	return mysql.WithFragmentTaskLeaseTransaction(ctx, r.db, id, write)
}

func (r *FragmentGenerationRepository) withTaskLease(ctx context.Context, id string, write func(*FragmentGenerationRepository) error) error {
	return r.WithTaskLeaseTransaction(ctx, id, func(tx *gorm.DB) error { return write(NewFragmentGenerationRepository(tx)) })
}

func (r *FragmentGenerationRepository) SaveCheckpoint(ctx context.Context, id, rawJSON string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &object); err != nil || object == nil {
		return fmt.Errorf("fragment checkpoint must be a JSON object")
	}
	query := r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).Where("id = ? AND status IN ?", id, []string{"pending", "processing"})
	return fragmentTaskWriteResult(ctx, id, fragmentTaskFencedQuery(ctx, query, id).Updates(map[string]any{"checkpoint_json": rawJSON, "updated_at": time.Now().Unix()}))
}

func (r *FragmentGenerationRepository) UpdateDraftForTask(ctx context.Context, id string, fragment *domain.Fragment) error {
	if fragment == nil || !fragment.IsDraft {
		return domain.ErrInvalidInput
	}
	return r.WithTaskLeaseTransaction(ctx, id, func(tx *gorm.DB) error {
		var current mysql.FragmentDB
		if err := tx.Where("id = ?", fragment.ID).First(&current).Error; err != nil {
			return err
		}
		// A new task or publication may have taken ownership while a model was
		// executing; neither may be overwritten by this task's stale draft copy.
		if !current.IsDraft || (current.GenerationTaskID != id && !(current.GenerationTaskID == "" && current.SourceID == id)) || fragment.GenerationTaskID != id {
			return domain.ErrFragmentTaskLeaseLost
		}
		fragment.ExpectedGenerationTaskID = current.GenerationTaskID
		scoped := &FragmentRepository{db: tx}
		return scoped.Update(ctx, fragment)
	})
}

func (r *FragmentGenerationRepository) CreateDraftForTask(ctx context.Context, id string, fragment *domain.Fragment) error {
	if fragment == nil || !fragment.IsDraft || fragment.GenerationTaskID != id {
		return domain.ErrInvalidInput
	}
	return r.WithTaskLeaseTransaction(ctx, id, func(tx *gorm.DB) error {
		var existing mysql.FragmentDB
		err := tx.Where("id = ?", fragment.ID).First(&existing).Error
		if err == nil {
			if existing.IsDraft && existing.GenerationTaskID == id && existing.UserID == fragment.UserID {
				return nil
			}
			return domain.ErrFragmentTaskLeaseLost
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return (&FragmentRepository{db: tx}).CreateWithTx(ctx, tx, fragment)
	})
}
