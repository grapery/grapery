package repository

import (
	"context"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFragmentSettlementAtomicallyCompletesOriginalTaskAndBlocksStaleDraft(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	migrateFragmentGenerationTaskTestTable(t, db)
	if err = db.Exec(`CREATE TABLE fragments (id TEXT PRIMARY KEY, creator_id TEXT, is_draft INTEGER, generation_task_id TEXT, generation_metadata TEXT, updated_at INTEGER)`).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewFragmentGenerationRepository(db)
	ctx := context.Background()
	task := &domain.FragmentGenerationTask{ID: "task", UserID: "owner", Status: "failed", ErrorMessage: "billing_unsettled: error"}
	if err = repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(`INSERT INTO fragments(id,creator_id,is_draft,generation_task_id,generation_metadata) VALUES(?,?,?,?,?)`, "draft", "owner", 1, "task", "pending metadata").Error; err != nil {
		t.Fatal(err)
	}
	result := &domain.FragmentGenerationResult{DraftFragmentID: "draft", GenerationTrace: &domain.FragmentGenerationTrace{BillingStatus: "settled", TextBillingTokens: 72}}
	if err = repo.FinalizeFragmentSettlement(ctx, "task", "draft", "owner", "pending metadata", "settled metadata", result); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, "task")
	if err != nil || got.Status != "completed" || got.ErrorMessage != "" || got.Result.GenerationTrace.BillingStatus != "settled" {
		t.Fatalf("settlement did not complete task %#v %v", got, err)
	}
	if err = repo.FinalizeFragmentSettlement(ctx, "task", "draft", "owner", "pending metadata", "stale", result); err == nil {
		t.Fatal("stale settlement changed newer draft")
	}
	var metadata string
	_ = db.Table("fragments").Select("generation_metadata").Where("id = ?", "draft").Scan(&metadata).Error
	if metadata != "settled metadata" {
		t.Fatal("settled authority overwritten")
	}
}
