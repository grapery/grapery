package repository

import (
	"context"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func migrateFragmentGenerationTaskTestTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
		CREATE TABLE fragment_generation_tasks (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			status TEXT NOT NULL,
			request_json TEXT NOT NULL,
			result_json TEXT,
			progress INTEGER,
			current_step TEXT,
			error_message TEXT,
			tokens_used INTEGER,
			created_at INTEGER,
			started_at INTEGER,
			completed_at INTEGER,
			updated_at INTEGER
		)
	`).Error; err != nil {
		t.Fatalf("create task table: %v", err)
	}
}

func TestFragmentGenerationRepository_UpsertAndListImageSlots(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&mysql.FragmentGenerationImageSlotDB{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	repo := NewFragmentGenerationRepository(db)
	ctx := context.Background()

	err = repo.UpsertImageSlots(ctx, "task-1", "frag-1", []domain.FragmentGenerationImageSlot{
		{Index: 2, Title: "第2页", Status: "planned"},
		{Index: 1, Title: "第1页", Status: "completed", ImageURL: "https://img.example/1.png", AssetID: "asset-1"},
	})
	if err != nil {
		t.Fatalf("upsert slots: %v", err)
	}
	err = repo.UpsertImageSlots(ctx, "task-1", "frag-1", []domain.FragmentGenerationImageSlot{
		{Index: 2, Title: "第2页", Status: "completed", ImageURL: "https://img.example/2.png", AssetID: "asset-2"},
	})
	if err != nil {
		t.Fatalf("update slot: %v", err)
	}
	err = repo.UpsertImageSlots(ctx, "task-1", "frag-1", []domain.FragmentGenerationImageSlot{
		{Index: 2, Title: "第2页", Status: "generating"},
	})
	if err != nil {
		t.Fatalf("stale update slot: %v", err)
	}

	slots, err := repo.ListImageSlots(ctx, "task-1")
	if err != nil {
		t.Fatalf("list slots: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}
	if slots[0].Index != 1 || slots[0].AssetID != "asset-1" {
		t.Fatalf("unexpected first slot: %#v", slots[0])
	}
	if slots[1].Index != 2 || slots[1].Status != "completed" || slots[1].AssetID != "asset-2" {
		t.Fatalf("unexpected second slot: %#v", slots[1])
	}
	if slots[1].ImageURL != "https://img.example/2.png" {
		t.Fatalf("expected completed image url preserved, got %q", slots[1].ImageURL)
	}
	if slots[0].ID == "" || slots[1].ID == "" || slots[0].ID == slots[1].ID {
		t.Fatalf("expected stable distinct slot ids: %#v", slots)
	}
}

func TestFragmentGenerationRepository_FindIdempotentAndActiveDraftTask(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	migrateFragmentGenerationTaskTestTable(t, db)

	repo := NewFragmentGenerationRepository(db)
	ctx := context.Background()

	tasks := []*domain.FragmentGenerationTask{
		{
			ID:     "task-completed",
			UserID: "user-1",
			Status: string(common.TaskStatusCompleted),
			Request: domain.FragmentGenerationRequest{
				UserInput:             "old",
				TargetDraftFragmentID: "draft-1",
				ClientMessageID:       "msg-old",
			},
			CreatedAt: 1,
		},
		{
			ID:     "task-active",
			UserID: "user-1",
			Status: string(common.TaskStatusProcessing),
			Request: domain.FragmentGenerationRequest{
				UserInput:             "continue",
				TargetDraftFragmentID: "draft-1",
				ClientMessageID:       "msg-1",
			},
			CreatedAt: 2,
		},
	}
	for _, task := range tasks {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("create task %s: %v", task.ID, err)
		}
	}

	byMessage, err := repo.FindByClientMessageID(ctx, "user-1", "msg-1")
	if err != nil {
		t.Fatalf("find by client message id: %v", err)
	}
	if byMessage == nil || byMessage.ID != "task-active" {
		t.Fatalf("expected task-active by client message id, got %#v", byMessage)
	}

	active, err := repo.FindActiveByDraftID(ctx, "user-1", "draft-1")
	if err != nil {
		t.Fatalf("find active by draft: %v", err)
	}
	if active == nil || active.ID != "task-active" {
		t.Fatalf("expected task-active by draft, got %#v", active)
	}

	if err := repo.UpdateStatus(ctx, "task-active", string(common.TaskStatusCompleted), 100, "completed"); err != nil {
		t.Fatalf("complete active task: %v", err)
	}
	active, err = repo.FindActiveByDraftID(ctx, "user-1", "draft-1")
	if err != nil {
		t.Fatalf("find active by draft after complete: %v", err)
	}
	if active != nil {
		t.Fatalf("expected no active task after completion, got %#v", active)
	}
}

func TestFragmentGenerationRepository_TerminalTaskCannotBeOverwritten(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	migrateFragmentGenerationTaskTestTable(t, db)

	repo := NewFragmentGenerationRepository(db)
	ctx := context.Background()
	task := &domain.FragmentGenerationTask{
		ID:        "task-cancel",
		UserID:    "user-1",
		Status:    string(common.TaskStatusProcessing),
		Request:   domain.FragmentGenerationRequest{UserInput: "story"},
		Progress:  40,
		CreatedAt: 1,
	}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := repo.UpdateStatus(ctx, task.ID, string(common.TaskStatusCancelled), 40, "cancelled by user"); err != nil {
		t.Fatalf("cancel task: %v", err)
	}
	if err := repo.UpdateResult(ctx, task.ID, &domain.FragmentGenerationResult{
		Content:   "should not be persisted",
		ImageUrls: []string{"https://img.example/late.png"},
	}); err != nil {
		t.Fatalf("late result update: %v", err)
	}
	if err := repo.UpdateStatus(ctx, task.ID, string(common.TaskStatusCompleted), 100, "completed"); err != nil {
		t.Fatalf("late completed update: %v", err)
	}
	if err := repo.UpdateError(ctx, task.ID, string(common.TaskStatusFailed), "late failure"); err != nil {
		t.Fatalf("late error update: %v", err)
	}

	got, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != string(common.TaskStatusCancelled) {
		t.Fatalf("expected cancelled status to be preserved, got %q", got.Status)
	}
	if got.Result != nil && got.Result.Content != "" {
		t.Fatalf("expected cancelled task result not to be overwritten, got %#v", got.Result)
	}
	if got.ErrorMessage != "" {
		t.Fatalf("expected cancelled task error not to be overwritten, got %q", got.ErrorMessage)
	}
}
