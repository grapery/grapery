package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"testing"
)

func publicationSnapshotTest(t *testing.T) (*FragmentGenerationRepository, *FragmentPublicationSnapshot) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { sql.Close() })
	migrateFragmentGenerationTaskTestTable(t, db)
	if err := db.Exec(`CREATE TABLE fragments(id TEXT PRIMARY KEY,creator_id TEXT,is_draft INTEGER,generation_task_id TEXT,generation_metadata TEXT,image_urls TEXT,content TEXT,updated_at INTEGER)`).Error; err != nil {
		t.Fatal(err)
	}
	trace := &domain.FragmentGenerationTrace{BillingStatus: "settled", Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}, ComicDocument: &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{{BackgroundImageURL: "background"}}}}
	metadata, _ := json.Marshal(trace)
	repo := NewFragmentGenerationRepository(db)
	task := &domain.FragmentGenerationTask{ID: "task", UserID: "owner", Status: "completed", Request: domain.FragmentGenerationRequest{BillingVersion: 1}, Result: &domain.FragmentGenerationResult{DraftFragmentID: "draft", GenerationTrace: trace, ComicDocument: trace.ComicDocument}}
	if err := repo.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO fragments VALUES(?,?,?,?,?,?,?,?)`, "draft", "owner", 1, "task", string(metadata), `["background"]`, "story", 1).Error; err != nil {
		t.Fatal(err)
	}
	snapshot, err := repo.PublicationSnapshot(context.Background(), "task", "owner")
	if err != nil {
		t.Fatal(err)
	}
	return repo, snapshot
}
func TestPreparedPublicationCASOnlyOneConcurrentAcceptance(t *testing.T) {
	repo, snapshot := publicationSnapshotTest(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result := &domain.FragmentGenerationResult{DraftFragmentID: "draft", Content: fmt.Sprint(i)}
			err := repo.CommitPreparedPublication(context.Background(), snapshot, fmt.Sprintf("reviewed-%d", i), result, []string{fmt.Sprintf("flattened-%d", i)})
			if err == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if accepted != 1 {
		t.Fatalf("accepted %d competing audits", accepted)
	}
	if _, err := repo.PublicationSnapshot(context.Background(), "task", "other"); err == nil {
		t.Fatal("other owner read source")
	}
}
func TestPreparedPublicationStaleTaskAndDraftRollback(t *testing.T) {
	for _, change := range []string{"content = 'new-story'", "generation_task_id = 'new-task'", "is_draft = 0", "image_urls = '[\"new-media\"]'"} {
		t.Run(change, func(t *testing.T) {
			repo, snapshot := publicationSnapshotTest(t)
			if err := repo.db.Exec("UPDATE fragments SET " + change + " WHERE id='draft'").Error; err != nil {
				t.Fatal(err)
			}
			if err := repo.CommitPreparedPublication(context.Background(), snapshot, "must-not-save", &domain.FragmentGenerationResult{Content: "stale"}, []string{"stale-media"}); err == nil {
				t.Fatal("stale acceptance committed")
			}
			row, err := repo.GetByID(context.Background(), "task")
			if err != nil {
				t.Fatal(err)
			}
			if row.Result.Content == "stale" {
				t.Fatal("task write not rolled back")
			}
		})
	}
	repo, snapshot := publicationSnapshotTest(t)
	if err := repo.db.Exec("UPDATE fragment_generation_tasks SET status='failed'").Error; err != nil {
		t.Fatal(err)
	}
	if repo.CommitPreparedPublication(context.Background(), snapshot, "bad", snapshot.Task.Result, []string{"new"}) == nil {
		t.Fatal("failed task accepted")
	}
}
func TestPreparedPublicationUnavailableAuditKeepsOriginalMedia(t *testing.T) {
	repo, snapshot := publicationSnapshotTest(t)
	result := snapshot.Task.Result
	result.Quality = &domain.FragmentGenerationQuality{Status: domain.FragmentQualityUnreviewed}
	if err := repo.CommitPreparedPublication(context.Background(), snapshot, "unreviewed", result, nil); err != nil {
		t.Fatal(err)
	}
	var media string
	repo.db.Table("fragments").Select("image_urls").Where("id='draft'").Scan(&media)
	if media != snapshot.MediaJSON {
		t.Fatal("unreviewed proposed images replaced source")
	}
}
