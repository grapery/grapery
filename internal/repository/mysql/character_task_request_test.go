package mysql

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestCharacterCreationRequestConcurrentReplicasReturnOneTask(t *testing.T) {
	first := characterExecutionTestRepo(t)
	// Separate repository instances model API replicas sharing the unique key.
	second := &Repository{db: first.db}
	var wg sync.WaitGroup
	var mu sync.Mutex
	ids := map[string]bool{}
	createdCount := 0
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			repo := first
			if i%2 == 1 {
				repo = second
			}
			task, created, err := repo.CreateCharacterTaskIdempotent(context.Background(), &domain.CharacterGenerationTask{UserID: "owner", StoryID: "world", SourceType: "manual_prompt", RequestKey: "same-key", IntentHash: "full-intent", Status: "pending", RequestJSON: `{"prompt":"saved"}`})
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			ids[task.ID] = true
			if created {
				createdCount++
			}
		}(i)
	}
	wg.Wait()
	if len(ids) != 1 || createdCount != 1 {
		t.Fatalf("ids=%v created=%d", ids, createdCount)
	}
	var count int64
	if err := first.db.Model(&CharacterGenerationTask{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal(count, err)
	}
	_, _, err := second.CreateCharacterTaskIdempotent(context.Background(), &domain.CharacterGenerationTask{UserID: "owner", StoryID: "world", RequestKey: "same-key", IntentHash: "different-reference-or-description", Status: "pending"})
	if !errors.Is(err, domain.ErrCharacterTaskRequestConflict) {
		t.Fatalf("same key silently changed intent: %v", err)
	}
}
func TestCharacterCreationRequestRetainsImmutableIdentityOnUpdates(t *testing.T) {
	repo := characterExecutionTestRepo(t)
	ctx := context.Background()
	task, _, err := repo.CreateCharacterTaskIdempotent(ctx, &domain.CharacterGenerationTask{UserID: "owner", RequestKey: "key", IntentHash: "original", Status: "pending"})
	if err != nil {
		t.Fatal(err)
	}
	task.RequestKey = "attempted-other-key"
	task.IntentHash = "attempted-other-intent"
	if err := repo.UpdateCharacterGenerationTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	saved, err := repo.CharacterGenerationTaskByID(ctx, task.ID)
	if err != nil || saved.RequestKey != "key" || saved.IntentHash != "original" {
		t.Fatal(saved, err)
	}
}

func TestCharacterTaskCanReadRealAgentExecutionRepository(t *testing.T) {
	runtime := newGenerationRuntimeTestRepository(t)
	run := &domain.GenerationExecution{ID: "character-child", UserID: "owner", Kind: "character", Status: "running", WorkflowReleaseID: "release", Input: map[string]any{"storyId": "story"}}
	if _, err := runtime.SaveGenerationExecution(context.Background(), run, "created"); err != nil {
		t.Fatal(err)
	}
	repo := &Repository{db: runtime.db}
	saved, err := repo.GetGenerationExecution(context.Background(), run.ID)
	if err != nil || saved.UserID != "owner" || saved.Kind != "character" || saved.WorkflowReleaseID != "release" || saved.Input["storyId"] != "story" {
		t.Fatal(saved, err)
	}
}

func TestCharacterTaskIdentityMigrationPreservesLegacyRowsAndEnforcesUniqueKey(t *testing.T) {
	repo := characterExecutionTestRepo(t)
	db := repo.db
	if err := db.Exec(`DROP TABLE character_generation_tasks`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE character_generation_tasks(id TEXT PRIMARY KEY,user_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO character_generation_tasks(id,user_id) VALUES('legacy-1','owner'),('legacy-2','owner')`).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := migrateCharacterTaskRequestIdentity(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}
	var rows int64
	if err := db.Table("character_generation_tasks").Where("request_key IS NULL").Count(&rows).Error; err != nil || rows != 2 {
		t.Fatal("legacy rows changed", rows, err)
	}
	if err := db.Exec(`INSERT INTO character_generation_tasks(id,user_id,request_key,intent_hash) VALUES('new-1','owner','stable','intent')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO character_generation_tasks(id,user_id,request_key,intent_hash) VALUES('new-2','owner','stable','intent')`).Error; err == nil {
		t.Fatal("migration did not install DB uniqueness")
	}
}
