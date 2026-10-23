package service

import (
	"context"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func fragmentWorkerTestRepo(t *testing.T) (*repository.FragmentGenerationRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	err = db.Exec(`CREATE TABLE fragment_generation_tasks (
 id TEXT PRIMARY KEY,user_id TEXT,status TEXT,request_json TEXT,result_json TEXT,
 progress INTEGER,current_step TEXT,error_message TEXT,tokens_used INTEGER,
 created_at INTEGER,started_at INTEGER,completed_at INTEGER,updated_at INTEGER,
 lease_owner TEXT NOT NULL DEFAULT '',lease_fence INTEGER NOT NULL DEFAULT 0,
	lease_expires_at INTEGER,checkpoint_json TEXT,execution_version INTEGER DEFAULT 0,ready_at INTEGER)`).Error
	if err != nil {
		t.Fatal(err)
	}
	return repository.NewFragmentGenerationRepository(db), db
}

func TestFragmentWorkerRestartClaimsExpiredTaskAndPreservesCheckpoint(t *testing.T) {
	repo, db := fragmentWorkerTestRepo(t)
	ctx := context.Background()
	task := &domain.FragmentGenerationTask{ID: "restart", UserID: "author", Status: "pending", CreatedAt: time.Now().Unix()}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkReady(ctx, task.ID); err != nil { t.Fatal(err) }
	old, err := repo.Claim(ctx, task.ID, "dead-process", time.Minute)
	if err != nil || old == nil {
		t.Fatalf("claim: %v", err)
	}
	if err := db.Exec("UPDATE fragment_generation_tasks SET checkpoint_json = ?, lease_expires_at = ? WHERE id = ?", `{"version":1,"sentinel":"saved-plan"}`, time.Now().Add(-time.Minute).UnixMilli(), task.ID).Error; err != nil {
		t.Fatal(err)
	}
	service := &FragmentGenerationService{fragmentGenRepo: repo, logger: zap.NewNop()}
	calls := 0
	service.runFragmentTask(ctx, task.ID, func(exec context.Context, id string) {
		calls++
		saved, err := repo.GetByID(exec, id)
		if err != nil {
			t.Fatal(err)
		}
		if saved.CheckpointJSON != `{"version":1,"sentinel":"saved-plan"}` || saved.LeaseFence <= old.LeaseFence {
			t.Fatalf("lost frozen data/fence: %+v", saved)
		}
		// A second server must not execute this live leased task.
		service.runFragmentTask(ctx, id, func(context.Context, string) { t.Error("duplicate executor") })
		if err := repo.UpdateStatus(exec, id, "completed", 100, "completed"); err != nil {
			t.Fatal(err)
		}
	})
	if calls != 1 {
		t.Fatalf("executions=%d", calls)
	}
	ids, err := repo.ListRecoverable(ctx, 10)
	if err != nil || len(ids) != 0 {
		t.Fatalf("terminal task requeued: %v %v", ids, err)
	}
}

func TestFragmentWorkerDoesNotExecuteCancelledTask(t *testing.T) {
	repo, _ := fragmentWorkerTestRepo(t)
	if err := repo.Create(context.Background(), &domain.FragmentGenerationTask{ID: "cancelled", UserID: "author", Status: "cancelled"}); err != nil {
		t.Fatal(err)
	}
	service := &FragmentGenerationService{fragmentGenRepo: repo, logger: zap.NewNop()}
	service.runFragmentTask(context.Background(), "cancelled", func(context.Context, string) { t.Fatal("cancelled task executed") })
}
