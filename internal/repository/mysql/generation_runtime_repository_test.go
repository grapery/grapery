package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newGenerationRuntimeTestRepository(t *testing.T) *GenerationRuntimeRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&GenerationExecutionDB{}, &GenerationEventDB{}, &GenerationCheckpointDB{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewGenerationRuntimeRepository(db)
}

func TestGenerationRuntimeRepository_SaveExecutionIncrementsSequenceAndEvents(t *testing.T) {
	repo := newGenerationRuntimeTestRepository(t)
	ctx := context.Background()
	run := &domain.GenerationExecution{ID: "run-1", UserID: "user-1", Kind: "fragment", Status: "pending", ClientRequestID: "request-1", Input: map[string]any{"prompt": "hello"}}
	event, err := repo.SaveGenerationExecution(ctx, run, "run.created")
	if err != nil {
		t.Fatalf("save create: %v", err)
	}
	if run.Sequence != 1 || event.Sequence != 1 {
		t.Fatalf("first sequence = %d/%d, want 1", run.Sequence, event.Sequence)
	}
	run.Status = "running"
	if _, err := repo.SaveGenerationExecution(ctx, run, "run.updated"); err != nil {
		t.Fatalf("save update: %v", err)
	}
	if run.Sequence != 2 {
		t.Fatalf("second sequence = %d, want 2", run.Sequence)
	}
	events, err := repo.ListGenerationEvents(ctx, run.ID, 1, 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Sequence != 2 {
		t.Fatalf("events after 1 = %#v", events)
	}
	loaded, err := repo.GetGenerationExecution(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if loaded.Status != "running" || loaded.Input["prompt"] != "hello" {
		t.Fatalf("loaded run = %#v", loaded)
	}
}

func TestGenerationRuntimeRepository_CheckpointRoundTrip(t *testing.T) {
	repo := newGenerationRuntimeTestRepository(t)
	ctx := context.Background()
	want := []byte{0, 1, 2, 255}
	err := repo.SaveGenerationCheckpoint(ctx, &domain.GenerationCheckpoint{ID: "cp-1", State: want, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	got, err := repo.GetGenerationCheckpoint(ctx, "cp-1")
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if string(got.State) != string(want) {
		t.Fatalf("state = %v, want %v", got.State, want)
	}
}

func TestGenerationRuntimeRepository_FindsLatestExecutionByContent(t *testing.T) {
	repo := newGenerationRuntimeTestRepository(t)
	ctx := context.Background()
	for _, run := range []*domain.GenerationExecution{
		{ID: "run-old", UserID: "user-1", Kind: "fragment", Status: "failed", ContentIDs: map[string]any{"fragmentId": "fragment-1"}, CreatedAt: time.Now().Add(-time.Minute)},
		{ID: "run-latest", UserID: "user-1", Kind: "fragment", Status: "running", ContentIDs: map[string]any{"fragmentId": "fragment-1"}, CreatedAt: time.Now()},
		{ID: "run-other-user", UserID: "user-2", Kind: "fragment", Status: "running", ContentIDs: map[string]any{"fragmentId": "fragment-1"}, CreatedAt: time.Now().Add(time.Minute)},
	} {
		if _, err := repo.SaveGenerationExecution(ctx, run, "run.created"); err != nil {
			t.Fatalf("save %s: %v", run.ID, err)
		}
	}
	got, err := repo.FindLatestGenerationExecution(ctx, "user-1", "fragment", "fragment-1")
	if err != nil {
		t.Fatalf("find latest: %v", err)
	}
	if got.ID != "run-latest" {
		t.Fatalf("latest id = %q, want run-latest", got.ID)
	}
}

func TestGenerationRuntimeRepository_ReusesCanonicalRunForClientRequest(t *testing.T) {
	repo := newGenerationRuntimeTestRepository(t)
	ctx := context.Background()
	first := &domain.GenerationExecution{ID: "run-first", UserID: "user-1", Kind: "fragment", Status: "pending", ClientRequestID: "same-request"}
	if _, err := repo.SaveGenerationExecution(ctx, first, "run.created"); err != nil {
		t.Fatalf("save first: %v", err)
	}
	retry := &domain.GenerationExecution{ID: "run-retry", UserID: "user-1", Kind: "fragment", Status: "pending", ClientRequestID: "same-request"}
	event, err := repo.SaveGenerationExecution(ctx, retry, "run.created")
	if err != nil {
		t.Fatalf("save retry: %v", err)
	}
	if event != nil {
		t.Fatalf("retry created event: %#v", event)
	}
	if retry.ID != first.ID || retry.Sequence != first.Sequence {
		t.Fatalf("retry = %#v, want canonical %#v", retry, first)
	}
}

func TestGenerationRuntimeRepository_TerminalStatusCannotBeOverwritten(t *testing.T) {
	repo := newGenerationRuntimeTestRepository(t)
	ctx := context.Background()
	run := &domain.GenerationExecution{ID: "run-terminal", UserID: "user-1", Kind: "storyboard", Status: "cancelled"}
	if _, err := repo.SaveGenerationExecution(ctx, run, "run.cancelled"); err != nil {
		t.Fatalf("save terminal run: %v", err)
	}
	sequence := run.Sequence
	run.Status = "running"
	event, err := repo.SaveGenerationExecution(ctx, run, "run.updated")
	if err != nil {
		t.Fatalf("save stale update: %v", err)
	}
	if event != nil {
		t.Fatalf("stale update created event: %#v", event)
	}
	if run.Status != "cancelled" || run.Sequence != sequence {
		t.Fatalf("terminal run overwritten: %#v", run)
	}
}
