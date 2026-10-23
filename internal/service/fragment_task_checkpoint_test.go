package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"go.uber.org/zap"
)

func TestFragmentCheckpointRestartSkipsCompletedModelStages(t *testing.T) {
	persisted := ""
	calls := 0
	save := func(raw string) error { persisted = raw; return nil }
	cp, _ := newFragmentTaskCheckpoint("", save)
	ctx := context.WithValue(context.Background(), fragmentTaskCheckpointKey{}, cp)
	for _, stage := range []string{"elements", "scenes", "page_plan:0", "references", "images", "audit"} {
		_, err := fragmentCheckpointStage(ctx, stage, false, func() (fragmentCheckpointPagePlan, error) {
			calls++
			return fragmentCheckpointPagePlan{Tokens: 17, Plans: []domain.FragmentScenePlan{{Index: 1, SceneDesc: "frozen"}}}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	restored, err := newFragmentTaskCheckpoint(persisted, save)
	if err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(context.Background(), fragmentTaskCheckpointKey{}, restored)
	for _, stage := range []string{"elements", "scenes", "page_plan:0", "references", "images", "audit"} {
		value, err := fragmentCheckpointStage(ctx, stage, false, func() (fragmentCheckpointPagePlan, error) {
			t.Fatal("completed paid stage replayed")
			return fragmentCheckpointPagePlan{}, nil
		})
		if err != nil || value.Tokens != 17 || value.Plans[0].SceneDesc != "frozen" {
			t.Fatal(value, err)
		}
	}
	if calls != 6 {
		t.Fatal(calls)
	}
}
func TestFragmentCheckpointUnknownProviderOutcomeNeverResends(t *testing.T) {
	for _, stage := range []string{"elements", "scenes", "page_plan:1", "audit", "recheck"} {
		persisted := ""
		cp, _ := newFragmentTaskCheckpoint("", func(raw string) error { persisted = raw; return nil })
		ctx := context.WithValue(context.Background(), fragmentTaskCheckpointKey{}, cp)
		_, _ = fragmentCheckpointStage(ctx, stage, false, func() (int, error) { return 0, errors.New("simulated process loss after request") })
		restored, _ := newFragmentTaskCheckpoint(persisted, func(string) error { return nil })
		ctx = context.WithValue(context.Background(), fragmentTaskCheckpointKey{}, restored)
		_, err := fragmentCheckpointStage(ctx, stage, false, func() (int, error) { t.Fatal("ambiguous provider request resent"); return 0, nil })
		if err == nil || !strings.Contains(err.Error(), "recovery_unknown_provider_outcome") {
			t.Fatal(err)
		}
	}
}
func TestFragmentCheckpointSaveFailureStopsExternalCalls(t *testing.T) {
	cp, _ := newFragmentTaskCheckpoint("", func(string) error { return errors.New("lost lease") })
	ctx := context.WithValue(context.Background(), fragmentTaskCheckpointKey{}, cp)
	_, err := fragmentCheckpointStage(ctx, "elements", false, func() (int, error) { t.Fatal("called provider without durable inflight"); return 1, nil })
	if err == nil {
		t.Fatal("ignored save failure")
	}
}
func TestFragmentCheckpointImageUsesStableOperationAndCompletedOutput(t *testing.T) {
	id := fragmentImageOperationID("task", "panel:1", `{"prompt":"rain"}`)
	if id != fragmentImageOperationID("task", "panel:1", `{"prompt":"rain"}`) || id == fragmentImageOperationID("task", "panel:2", `{"prompt":"rain"}`) || id == fragmentImageOperationID("task", "panel:1", `{"prompt":"repair"}`) {
		t.Fatal("image operation identities collide/drift")
	}
	raw := `{"version":1,"stages":{"image_intent:` + id + `":{"state":"completed","payload":{"id":"` + id + `","provider":"original-provider","input":"frozen"}},"image_output:` + id + `":{"state":"completed","payload":{"URL":"https://stored/image","Tokens":42}}}}`
	cp, err := newFragmentTaskCheckpoint(raw, func(string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), fragmentTaskCheckpointKey{}, cp)
	svc := &FragmentGenerationService{}
	url, tokens, err := svc.checkpointFragmentImage(ctx, &domain.AITask{RelatedEntityID: "task", Input: `{"prompt":"rain"}`}, "panel:1", func(*domain.AITask) (string, int, error) {
		t.Fatal("stored output called provider or charged again")
		return "", 0, nil
	})
	if err != nil || url != "https://stored/image" || tokens != 42 {
		t.Fatal(url, tokens, err)
	}
}
func TestFragmentLegacyExecutedTaskFailsWithoutChangingExistingOutputs(t *testing.T) {
	repo, _ := fragmentWorkerTestRepo(t)
	ctx := context.Background()
	task := &domain.FragmentGenerationTask{ID: "legacy-output", UserID: "author", Status: "processing", Progress: 70, CurrentStep: "generating_images", Result: &domain.FragmentGenerationResult{Content: "saved story", ImageUrls: []string{"https://stored/image"}}}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	svc := &FragmentGenerationService{fragmentGenRepo: repo, logger: zap.NewNop()}
	svc.processFragmentGeneration(ctx, task.ID) // no AI service or fragment repo: either use would panic.
	current, err := repo.GetByID(ctx, task.ID)
	if err != nil || current.Status != "failed" || !strings.Contains(current.ErrorMessage, "recovery_checkpoint_missing") || current.Result.Content != "saved story" || current.Result.ImageUrls[0] != "https://stored/image" {
		t.Fatal(current, err)
	}
}
func TestFragmentPreparedCommitRecoveryDoesNotAppendTwice(t *testing.T) {
	repo, db := fragmentWorkerTestRepo(t)
	if err := db.Exec(`CREATE TABLE fragments (
 id TEXT PRIMARY KEY,creator_id TEXT,content TEXT,image_urls TEXT,visibility TEXT,
 source_type TEXT,source_id TEXT,topic TEXT,caption TEXT,style TEXT,converted_to_story_id TEXT,
 is_converted NUMERIC,is_draft NUMERIC,generation_task_id TEXT,generation_metadata TEXT,
 image_aspect_ratio TEXT,likes INTEGER,comments INTEGER,shares INTEGER,views INTEGER,
 created_at INTEGER,updated_at INTEGER)`).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	task := &domain.FragmentGenerationTask{ID: "append", UserID: "author", Status: "processing"}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	draft := &domain.Fragment{BaseModel: common.BaseModel{ID: "draft"}, UserID: "author", CreatorID: "author", IsDraft: true, GenerationTaskID: task.ID, Content: "first\nsecond", MediaURLs: []string{"https://first", "https://second"}, GenerationMetadata: "pending"}
	if err := db.Create(mysql.DomainToFragmentDB(draft)).Error; err != nil {
		t.Fatal(err)
	}
	prepared := fragmentCheckpointCommit{Draft: draft, Result: &domain.FragmentGenerationResult{DraftFragmentID: draft.ID, Content: draft.Content, ImageUrls: draft.MediaURLs, GenerationTrace: &domain.FragmentGenerationTrace{BillingStatus: "pending"}}}
	prepared.Draft.GenerationMetadata = marshalFragmentGenerationMetadata(prepared.Result.GenerationTrace)
	cp, _ := newFragmentTaskCheckpoint("", func(raw string) error { return repo.SaveCheckpoint(ctx, task.ID, raw) })
	checkpointCtx := context.WithValue(ctx, fragmentTaskCheckpointKey{}, cp)
	if _, err := fragmentCheckpointStage(checkpointCtx, "prepared_commit", true, func() (*fragmentCheckpointCommit, error) { return &prepared, nil }); err != nil {
		t.Fatal(err)
	}
	// Simulate crash after successful draft write but before task result/settlement.
	if err := repo.UpdateDraftForTask(ctx, task.ID, prepared.Draft); err != nil {
		t.Fatal(err)
	}
	svc := &FragmentGenerationService{fragmentGenRepo: repo, logger: zap.NewNop()}
	svc.processFragmentGeneration(ctx, task.ID) // no AI service: recovery must only commit+settle.
	current, err := repo.GetByID(ctx, task.ID)
	if err != nil || current.Status != "completed" || current.Result.GenerationTrace.BillingStatus != "settled" {
		t.Fatal(current, err)
	}
	svc.processFragmentGeneration(ctx, task.ID) // terminal replay is also a no-op.

	var stored mysql.FragmentDB
	if err := db.Where("id = ?", draft.ID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Content != "first\nsecond" {
		t.Fatal("duplicate append", stored.Content)
	}
	var urls []string
	if err := json.Unmarshal([]byte(stored.ImageUrls), &urls); err != nil || len(urls) != 2 {
		t.Fatal(stored.ImageUrls, err)
	}
}

func TestFragmentProcessResumesFrozenExtractionThenFailsUnknownNextStage(t *testing.T) {
	repo, _ := fragmentWorkerTestRepo(t)
	ctx := context.Background()
	task := &domain.FragmentGenerationTask{ID: "resume-text", UserID: "author", Status: "processing", Request: domain.FragmentGenerationRequest{ImageCount: 1}, Progress: 28, CurrentStep: "expanding_scenes"}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	cp, _ := newFragmentTaskCheckpoint("", func(raw string) error { return repo.SaveCheckpoint(ctx, task.ID, raw) })
	cp.Stages["baseline"] = fragmentTaskCheckpointStage{State: "completed", Payload: json.RawMessage(`{}`)}
	cp.Stages["elements"] = fragmentTaskCheckpointStage{State: "completed", Payload: json.RawMessage(`{"content":"preserved extraction","TokensUsed":19}`)}
	cp.Stages["scenes"] = fragmentTaskCheckpointStage{State: "inflight"}
	if err := cp.persist(); err != nil {
		t.Fatal(err)
	}
	svc := &FragmentGenerationService{fragmentGenRepo: repo, logger: zap.NewNop()}
	svc.processFragmentGeneration(ctx, task.ID) // nil AI/repo proves cached extraction, no model replay.
	current, err := repo.GetByID(ctx, task.ID)
	if err != nil || current.Status != "failed" || !strings.Contains(current.ErrorMessage, "recovery_unknown_provider_outcome: scenes") {
		t.Fatal(current, err)
	}
	if !strings.Contains(current.CheckpointJSON, "preserved extraction") {
		t.Fatal("lost checkpoint", current.CheckpointJSON)
	}
}
