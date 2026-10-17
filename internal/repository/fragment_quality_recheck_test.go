package repository

import (
	"context"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFragmentQualityRecheckCommitIsAtomicAndRejectsNewerEdit(t *testing.T) {
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
	task := &domain.FragmentGenerationTask{ID: "task", UserID: "owner", Status: "completed", Result: &domain.FragmentGenerationResult{}}
	if err = repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(`INSERT INTO fragments (id,creator_id,is_draft,generation_task_id,generation_metadata) VALUES (?,?,?,?,?)`, "draft", "owner", 1, "task", "old").Error; err != nil {
		t.Fatal(err)
	}
	result := &domain.FragmentGenerationResult{Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}}
	if err = repo.UpdateRecheckedQuality(ctx, "task", "draft", "owner", "old", "reviewed", result); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, "task")
	if err != nil || got.Result.Quality.Status != domain.FragmentQualityPassed {
		t.Fatal("completed task did not persist review")
	}
	if err = repo.UpdateRecheckedQuality(ctx, "task", "draft", "owner", "old", "stale overwrite", result); err == nil {
		t.Fatal("stale audit overwrote newer metadata")
	}
	if err = db.Exec(`UPDATE fragment_generation_tasks SET status = ? WHERE id = ?`, "failed", "task").Error; err != nil {
		t.Fatal(err)
	}
	if err = repo.UpdateRecheckedQuality(ctx, "task", "draft", "owner", "reviewed", "must rollback", result); err == nil {
		t.Fatal("changed task accepted")
	}
	var metadata string
	if err = db.Table("fragments").Select("generation_metadata").Where("id = ?", "draft").Scan(&metadata).Error; err != nil {
		t.Fatal(err)
	}
	if metadata != "reviewed" {
		t.Fatalf("task failure did not roll back draft metadata: %s", metadata)
	}
	if err = repo.UpdateRecheckedQuality(ctx, "task", "draft", "other-owner", "reviewed", "unauthorized", result); err == nil {
		t.Fatal("other owner changed quality")
	}
}
