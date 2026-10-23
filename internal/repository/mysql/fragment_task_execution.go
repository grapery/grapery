package mysql

import (
	"context"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WithFragmentTaskLeaseTransaction serializes task-owned side effects against
// takeover/cancellation. The task row stays locked until all writes commit.
// Without a lease only a never-claimed legacy task may write.
func WithFragmentTaskLeaseTransaction(ctx context.Context, db *gorm.DB, taskID string, write func(*gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task FragmentGenerationTaskDB
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		if err := ValidateFragmentTaskLease(ctx, &task); err != nil {
			return err
		}
		return write(tx)
	})
}

func ValidateFragmentTaskLease(ctx context.Context, task *FragmentGenerationTaskDB) error {
	if task.Status != "pending" && task.Status != "processing" {
		return domain.ErrFragmentTaskLeaseLost
	}
	lease, claimed := domain.FragmentTaskLeaseFromContext(ctx)
	if !claimed {
		if task.LeaseFence != 0 || task.LeaseOwner != "" {
			return domain.ErrFragmentTaskLeaseLost
		}
		return nil
	}
	if lease.TaskID != task.ID || lease.Owner == "" || lease.Owner != task.LeaseOwner || lease.Fence != task.LeaseFence || task.LeaseExpiresAt == nil || *task.LeaseExpiresAt <= time.Now().UnixMilli() {
		return domain.ErrFragmentTaskLeaseLost
	}
	return nil
}
