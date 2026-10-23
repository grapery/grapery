package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func fragmentLeaseRepository(t *testing.T) *FragmentGenerationRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = connection.Close() })
	migrateFragmentGenerationTaskTestTable(t, db)
	if err := db.AutoMigrate(&mysql.FragmentGenerationImageSlotDB{}); err != nil {
		t.Fatal(err)
	}
	return NewFragmentGenerationRepository(db)
}

func TestFragmentLeaseConcurrentClaimAndLegacyRecovery(t *testing.T) {
	repo := fragmentLeaseRepository(t)
	ctx := context.Background()
	for _, task := range []*domain.FragmentGenerationTask{
		{ID: "legacy", UserID: "owner", Status: "processing", Progress: 62, CurrentStep: "generating_images", CheckpointJSON: `{"version":1,"stages":{"plan":{"state":"completed"}}}`, Result: &domain.FragmentGenerationResult{Content: "saved story"}, CreatedAt: time.Now().Add(-time.Minute).Unix()},
		{ID: "setup", UserID: "owner", Status: "pending", CreatedAt: time.Now().Unix()},
		{ID: "old-pending", UserID: "owner", Status: "pending", CreatedAt: time.Now().Add(-time.Minute).Unix()},
		{ID: "terminal", UserID: "owner", Status: "completed", CreatedAt: 1},
	} {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.db.Model(&mysql.FragmentGenerationTaskDB{}).Where("id IN ?", []string{"legacy", "old-pending"}).Update("execution_version", 0).Error; err != nil {
		t.Fatal(err)
	}
	ids, err := repo.ListRecoverable(ctx, 100)
	if err != nil || len(ids) != 2 {
		t.Fatalf("legacy recovery/grace: %v %v", ids, err)
	}
	var claims atomic.Int32
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			task, err := repo.Claim(ctx, "legacy", fmt.Sprintf("worker-%d", i), time.Minute)
			if err == nil {
				claims.Add(1)
				if task.Progress != 62 || task.CurrentStep != "generating_images" || task.CheckpointJSON == "" || task.Result.Content != "saved story" {
					t.Errorf("claim discarded recovery source: %#v", task)
				}
			} else if !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
				t.Errorf("claim: %v", err)
			}
		}(i)
	}
	workers.Wait()
	if claims.Load() != 1 {
		t.Fatalf("claimed %d times", claims.Load())
	}
	if _, err = repo.Claim(ctx, "terminal", "worker", time.Minute); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("terminal task claimed", err)
	}
	if _, err = repo.Claim(ctx, "setup", "too-early", time.Minute); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("unprepared task claimed", err)
	}
	if err = repo.MarkReady(ctx, "setup"); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Claim(ctx, "setup", "immediate", time.Minute); err != nil {
		t.Fatal(err)
	}
}

func TestFragmentExpiredLeaseFencesEveryTaskWrite(t *testing.T) {
	repo := fragmentLeaseRepository(t)
	ctx := context.Background()
	if err := repo.Create(ctx, &domain.FragmentGenerationTask{ID: "task", UserID: "owner", Status: "pending", CreatedAt: 1}); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkReady(ctx, "task"); err != nil {
		t.Fatal(err)
	}
	old, err := repo.Claim(ctx, "task", "old", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	oldCtx := domain.WithFragmentTaskLease(ctx, old)
	if err = repo.SaveCheckpoint(oldCtx, "task", `{"version":1}`); err != nil {
		t.Fatal(err)
	}
	if err = repo.db.Model(&mysql.FragmentGenerationTaskDB{}).Where("id = ?", "task").Update("lease_expires_at", time.Now().Add(-time.Second).UnixMilli()).Error; err != nil {
		t.Fatal(err)
	}
	if err = repo.Renew(ctx, old, time.Minute); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("expired lease resurrected", err)
	}
	fresh, err := repo.Claim(ctx, "task", "fresh", time.Minute)
	if err != nil || fresh.LeaseFence != old.LeaseFence+1 {
		t.Fatalf("takeover: %#v %v", fresh, err)
	}
	staleWrites := []func() error{
		func() error { return repo.CheckLease(oldCtx, "task") },
		func() error { return repo.Renew(ctx, old, time.Minute) },
		func() error { return repo.UpdateStatus(oldCtx, "task", "processing", 99, "stale") },
		func() error { return repo.UpdateStatus(oldCtx, "task", "cancelled", 99, "stale") },
		func() error { return repo.UpdateError(oldCtx, "task", "failed", "stale") },
		func() error {
			return repo.UpdateResult(oldCtx, "task", &domain.FragmentGenerationResult{Content: "stale"})
		},
		func() error { return repo.SaveCheckpoint(oldCtx, "task", `{"stale":true}`) },
		func() error {
			return repo.UpsertImageSlots(oldCtx, "task", "draft", []domain.FragmentGenerationImageSlot{{Index: 1, Status: "completed", ImageURL: "stale"}})
		},
		func() error {
			return repo.UpdateDraftForTask(oldCtx, "task", &domain.Fragment{IsDraft: true, GenerationTaskID: "task"})
		},
		func() error {
			return repo.CreateDraftForTask(oldCtx, "task", &domain.Fragment{IsDraft: true, GenerationTaskID: "task"})
		},
		func() error { return repo.Update(oldCtx, old) },
		func() error { return repo.Delete(oldCtx, "task") },
	}
	for i, write := range staleWrites {
		if err = write(); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
			t.Fatalf("stale write %d: %v", i, err)
		}
	}
	freshCtx := domain.WithFragmentTaskLease(ctx, fresh)
	if err = repo.Renew(ctx, fresh, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = repo.UpdateStatus(freshCtx, "task", "processing", 70, "rendering"); err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveCheckpoint(freshCtx, "task", `{"version":1,"fresh":true}`); err != nil {
		t.Fatal(err)
	}
	if err = repo.UpsertImageSlots(freshCtx, "task", "draft", []domain.FragmentGenerationImageSlot{{Index: 1, Status: "completed", ImageURL: "fresh"}}); err != nil {
		t.Fatal(err)
	}
	// Unbound legacy callers must not overwrite a task after its first claim.
	_ = repo.UpdateResult(ctx, "task", &domain.FragmentGenerationResult{Content: "unbound"})
	got, err := repo.GetByID(ctx, "task")
	if err != nil || got.CheckpointJSON != `{"version":1,"fresh":true}` || got.Progress != 70 || got.Result != nil {
		t.Fatalf("stale mutation leaked: %#v %v", got, err)
	}
	slots, err := repo.ListImageSlots(ctx, "task")
	if err != nil || len(slots) != 1 || slots[0].ImageURL != "fresh" {
		t.Fatalf("stale slots: %v %v", slots, err)
	}
	if err = repo.UpdateStatus(ctx, "task", "cancelled", 70, "cancelled"); err != nil {
		t.Fatal(err)
	}
	if err = repo.CheckLease(freshCtx, "task"); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("cancelled worker can write", err)
	}
	if _, err = repo.Claim(ctx, "task", "third", time.Minute); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("cancelled task reclaimed", err)
	}
}

func TestFragmentLeaseContextIdentityAndTransactionRollback(t *testing.T) {
	repo := fragmentLeaseRepository(t)
	ctx := context.Background()
	for _, id := range []string{"a", "b"} {
		if err := repo.Create(ctx, &domain.FragmentGenerationTask{ID: id, UserID: "owner", Status: "pending"}); err != nil {
			t.Fatal(err)
		}
		if err := repo.MarkReady(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	task, err := repo.Claim(ctx, "a", "worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	leased := domain.WithFragmentTaskLease(ctx, task)
	task.LeaseOwner = "mutated"
	if err = repo.CheckLease(leased, "a"); err != nil {
		t.Fatal("context lease aliases mutable task", err)
	}
	if err = repo.SaveCheckpoint(leased, "b", `{}`); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("cross-task lease accepted", err)
	}
	for _, raw := range []string{`null`, `[]`, `{bad}`} {
		if repo.SaveCheckpoint(leased, "a", raw) == nil {
			t.Fatal("invalid checkpoint accepted", raw)
		}
	}
	if err = repo.db.Exec(`CREATE TABLE side_effects (id TEXT PRIMARY KEY)`).Error; err != nil {
		t.Fatal(err)
	}
	writeError := errors.New("rollback")
	err = repo.WithTaskLeaseTransaction(leased, "a", func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO side_effects(id) VALUES('atomic')`).Error; err != nil {
			return err
		}
		return writeError
	})
	if !errors.Is(err, writeError) {
		t.Fatal(err)
	}
	var count int64
	if err = repo.db.Table("side_effects").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("task side effect did not rollback", count, err)
	}
}

func TestFragmentPreparationReadinessCannotRaceRecovery(t *testing.T) {
	repo := fragmentLeaseRepository(t)
	ctx := context.Background()
	for _, id := range []string{"preparing", "abandoned"} {
		if err := repo.Create(ctx, &domain.FragmentGenerationTask{ID: id, UserID: "owner", Status: "pending", CreatedAt: time.Now().Unix()}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.db.Model(&mysql.FragmentGenerationTaskDB{}).Where("id = ?", "abandoned").Update("created_at", time.Now().Add(-6*time.Minute).Unix()).Error; err != nil {
		t.Fatal(err)
	}
	ids, err := repo.ListRecoverable(ctx, 100)
	if err != nil || len(ids) != 0 {
		t.Fatalf("preparation rows dispatched: %v %v", ids, err)
	}
	stopped, err := repo.GetByID(ctx, "abandoned")
	if err != nil || stopped.Status != "failed" || stopped.CurrentStep != "preparation_incomplete" {
		t.Fatalf("abandoned setup not retired: %#v %v", stopped, err)
	}
	if err = repo.MarkReady(ctx, "abandoned"); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("late request resurrected retired setup", err)
	}
	if _, err = repo.Claim(ctx, "preparing", "scanner", time.Minute); !errors.Is(err, domain.ErrFragmentTaskLeaseLost) {
		t.Fatal("incomplete setup claimed", err)
	}
	if err = repo.MarkReady(ctx, "preparing"); err != nil {
		t.Fatal(err)
	}
	ids, err = repo.ListRecoverable(ctx, 100)
	if err != nil || len(ids) != 1 || ids[0] != "preparing" {
		t.Fatalf("prepared task not recoverable: %v %v", ids, err)
	}
}
