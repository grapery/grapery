package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestFragmentRequestLookupNeverForgetsOldClientKey(t *testing.T) {
	repo := fragmentLeaseRepository(t)
	ctx := context.Background()
	for i := 0; i < 105; i++ {
		if err := repo.Create(ctx, &domain.FragmentGenerationTask{ID: fmt.Sprintf("task-%03d", i), UserID: "owner", Status: "completed", CreatedAt: int64(i), Request: domain.FragmentGenerationRequest{ClientMessageID: fmt.Sprintf("message-%d", i)}}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.FindByClientMessageID(ctx, "owner", "message-0")
	if err != nil || got == nil || got.ID != "task-000" {
		t.Fatalf("old retry identity lost: %#v %v", got, err)
	}
	if FragmentGenerationRequestTaskID("owner", "key") == FragmentGenerationRequestTaskID("other", "key") {
		t.Fatal("owner-scoped retry key collision")
	}
}

func TestFragmentDraftAttachmentBlocksInitialGenerationAndPreservesOwnerEdits(t *testing.T) {
	repo := fragmentLeaseRepository(t)
	ctx := context.Background()
	if err := repo.db.Exec(`CREATE TABLE fragments(id TEXT PRIMARY KEY,creator_id TEXT,is_draft INTEGER,generation_task_id TEXT,source_id TEXT,source_type TEXT,image_urls TEXT,generation_metadata TEXT,aspect_ratio TEXT,updated_at INTEGER,content TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, &domain.FragmentGenerationTask{ID: "original", UserID: "owner", Status: "processing"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.db.Exec(`INSERT INTO fragments VALUES('draft','owner',1,'','original','ai_fragment_generation','["old-image"]','saved metadata','16:9',1,'owner current text')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.AttachDraftForTask(ctx, "new", "draft", "owner", "3:4", 0); err == nil {
		t.Fatal("new task overtook initial generation SourceID")
	}
	if err := repo.db.Exec(`UPDATE fragment_generation_tasks SET status='completed' WHERE id='original'`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.AttachDraftForTask(ctx, "new", "draft", "other", "3:4", 0); err == nil {
		t.Fatal("another owner acquired draft")
	}
	if err := repo.AttachDraftForTask(ctx, "new", "draft", "owner", "3:4", 2); err == nil {
		t.Fatal("invalid replacement index accepted")
	}
	if err := repo.AttachDraftForTask(ctx, "new", "draft", "owner", "3:4", 1); err != nil {
		t.Fatal(err)
	}
	var got struct{ Content, GenerationMetadata, GenerationTaskID, AspectRatio string }
	if err := repo.db.Table("fragments").Where("id='draft'").Scan(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Content != "owner current text" || got.GenerationMetadata != "saved metadata" || got.GenerationTaskID != "new" || got.AspectRatio != "16:9" {
		t.Fatalf("attachment clobbered saved draft: %#v", got)
	}
}

func TestFragmentRetryCASPreservesOriginalTaskAndRejectsStaleDraft(t *testing.T) {
	repo := fragmentLeaseRepository(t)
	ctx := context.Background()
	if err := repo.db.Exec(`CREATE TABLE fragments(id TEXT PRIMARY KEY,creator_id TEXT,is_draft INTEGER,generation_task_id TEXT,source_id TEXT,source_type TEXT,updated_at INTEGER)`).Error; err != nil {
		t.Fatal(err)
	}
	cp := `{"version":1,"stages":{"elements":{"state":"completed","payload":{"Content":"saved"}},"images":{"state":"inflight"}}}`
	task := &domain.FragmentGenerationTask{ID: "failed", UserID: "owner", Status: "failed", CheckpointJSON: cp, Result: &domain.FragmentGenerationResult{Content: "saved", ImageUrls: []string{"completed-page"}}}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := repo.db.Exec(`INSERT INTO fragments VALUES('draft','owner',1,'failed','failed','ai_fragment_generation',0)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.QueueRetry(ctx, "failed", "draft", "owner", "different checkpoint"); err == nil {
		t.Fatal("stale checkpoint retry accepted")
	}
	if err := repo.QueueRetry(ctx, "failed", "draft", "owner", cp); err != nil {
		t.Fatal(err)
	}
	if err := repo.QueueRetry(ctx, "failed", "draft", "owner", cp); err != nil {
		t.Fatal("duplicate retry should join active original", err)
	}
	got, err := repo.GetByID(ctx, "failed")
	if err != nil || got.Status != "pending" || got.CheckpointJSON != cp || got.Result.ImageUrls[0] != "completed-page" || got.LeaseFence != 1 {
		t.Fatalf("retry discarded source or queued twice: %#v %v", got, err)
	}
	if err := repo.db.Exec(`UPDATE fragments SET generation_task_id='newer' WHERE id='draft'`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.QueueRetry(ctx, "failed", "draft", "owner", cp); err == nil {
		t.Fatal("stale active task joined newer draft")
	}
}
