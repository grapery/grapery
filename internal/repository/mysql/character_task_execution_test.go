package mysql

import (
	"context"
	"errors"
	"fmt"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"testing"
	"time"
)

func characterExecutionTestRepo(t *testing.T) *Repository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	err = db.Exec(`CREATE TABLE character_generation_tasks (id TEXT PRIMARY KEY,user_id TEXT,story_id TEXT,character_id TEXT,source_type TEXT,source_fragment_id TEXT,source_fragment_character_key TEXT,status TEXT,progress INTEGER,current_step TEXT,request_json TEXT,result_json TEXT,error_message TEXT,stages_json TEXT,attributes_json TEXT,lease_owner TEXT,lease_fence INTEGER NOT NULL DEFAULT 0,lease_expires_at DATETIME,created_at DATETIME,updated_at DATETIME,completed_at DATETIME,draft_dismissed_at DATETIME,deleted_at DATETIME)`).Error
	if err != nil {
		t.Fatal(err)
	}
	return &Repository{db: db}
}
func TestCharacterTaskOnlyOneClaimAndRetry(t *testing.T) {
	repo := characterExecutionTestRepo(t)
	ctx := context.Background()
	task := &domain.CharacterGenerationTask{ID: "task", UserID: "user", StoryID: "story", Status: domain.CharacterGenerationStatusPending}
	if err := repo.CreateCharacterGenerationTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	claimed := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := repo.ClaimCharacterTask(ctx, task.ID, fmt.Sprint(i), time.Minute); err == nil {
				mu.Lock()
				claimed++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if claimed != 1 {
		t.Fatalf("got %d claimants", claimed)
	}
	if err := repo.db.Model(&CharacterGenerationTask{}).Where("id = ?", task.ID).Update("status", domain.CharacterGenerationStatusPartial).Error; err != nil {
		t.Fatal(err)
	}
	queued := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.QueueCharacterTaskRetry(ctx, task.ID, "user")
			if err != nil {
				t.Error(err)
			}
			if ok {
				mu.Lock()
				queued++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if queued != 1 {
		t.Fatalf("got %d queued retries", queued)
	}
}
func TestCharacterTaskExpiredLeaseFencesOldWorker(t *testing.T) {
	repo := characterExecutionTestRepo(t)
	ctx := context.Background()
	task := &domain.CharacterGenerationTask{ID: "task", UserID: "user", StoryID: "story", Status: domain.CharacterGenerationStatusPending}
	if err := repo.CreateCharacterGenerationTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	old, err := repo.ClaimCharacterTask(ctx, task.ID, "old", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.db.Model(&CharacterGenerationTask{}).Where("id = ?", task.ID).Update("lease_expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	ids, err := repo.RecoverableCharacterTasks(ctx, 20)
	if err != nil || len(ids) != 1 {
		t.Fatalf("recoverable: %v %v", ids, err)
	}
	fresh, err := repo.ClaimCharacterTask(ctx, task.ID, "new", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.LeaseFence != old.LeaseFence+1 {
		t.Fatal("fence did not advance")
	}
	if err := repo.RenewCharacterTask(ctx, old, time.Minute); !errors.Is(err, domain.ErrCharacterTaskLeaseLost) {
		t.Fatalf("old renewal: %v", err)
	}
	old.ErrorMessage = "stale"
	if err := repo.UpdateCharacterGenerationTask(ctx, old); !errors.Is(err, domain.ErrCharacterTaskLeaseLost) {
		t.Fatalf("old progress: %v", err)
	}
	if err := repo.UpdateCharacterForTask(ctx, old, &domain.Character{}); !errors.Is(err, domain.ErrCharacterTaskLeaseLost) {
		t.Fatalf("old character write: %v", err)
	}
	if err := repo.RenewCharacterTask(ctx, fresh, time.Minute); err != nil {
		t.Fatal(err)
	}
	fresh.Stages = map[string]domain.CharacterGenerationStage{"portrait": {Status: "failed", Attempts: 1, Error: "timeout"}}
	fresh.AttributesJSON = `{"description":"saved"}`
	if err := repo.UpdateCharacterGenerationTask(ctx, fresh); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repo.CharacterGenerationTaskByID(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Stages["portrait"].Error != "timeout" || reloaded.AttributesJSON != fresh.AttributesJSON {
		t.Fatal("stage checkpoint lost")
	}
}
