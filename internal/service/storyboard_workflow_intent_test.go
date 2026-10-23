package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func storyboardIntentTestRun(board *domain.Storyboard, opts StoryboardWorkflowStageOptions) *domain.StoryboardGenerationRun {
	controls := opts.CreativeControls
	request := map[string]any{"requestIntent": newStoryboardWorkflowIntent(board, opts, controls), "clientRequestId": opts.ClientRequestID, "regenerateStructure": opts.RegenerateStructure, "userDirective": opts.UserDirective, "sceneCount": 3, "comicStyle": "ink", "language": "zh-Hans", "creativeControls": controls}
	snapshot := storyboardGenerationContextSnapshot{RawInput: board.RawInput, StoryID: board.StoryID}
	return &domain.StoryboardGenerationRun{ID: "run-A", StoryboardID: board.ID, UserID: board.UserID, RequestJSON: mustJSON(request, "{}"), ContextSnapshotJSON: mustJSON(snapshot, "{}"), AlignmentSnapshotJSON: "{}"}
}
func storyboardIntentTestBoard() *domain.Storyboard {
	return &domain.Storyboard{BaseModel: common.BaseModel{ID: "board"}, UserID: "owner", StoryID: "story", RawInput: "original plot", ParentID: "parent", SceneCount: 3, ContinuationComicStyle: "ink"}
}
func storyboardIntentTestOptions() StoryboardWorkflowStageOptions {
	return StoryboardWorkflowStageOptions{ClientRequestID: "turn-A", RegenerateStructure: true, UserDirective: "change rain", SceneCount: 3, ComicStyle: "ink", Language: "zh-Hans", TargetSceneIDs: []string{"scene-B", "scene-A"}, BaseRevision: "accepted-hash"}
}

func TestStoryboardRevisionKeyBindsEveryCreativeRequestChoice(t *testing.T) {
	board := storyboardIntentTestBoard()
	opts := storyboardIntentTestOptions()
	run := storyboardIntentTestRun(board, opts)
	for _, tc := range []struct {
		name   string
		change func(*StoryboardWorkflowStageOptions, *domain.Storyboard)
	}{
		{"directive", func(o *StoryboardWorkflowStageOptions, _ *domain.Storyboard) { o.UserDirective = "different" }},
		{"target scene set", func(o *StoryboardWorkflowStageOptions, _ *domain.Storyboard) { o.TargetSceneIDs = []string{"scene-C"} }},
		{"base revision", func(o *StoryboardWorkflowStageOptions, _ *domain.Storyboard) { o.BaseRevision = "other-hash" }},
		{"page count", func(o *StoryboardWorkflowStageOptions, _ *domain.Storyboard) { o.SceneCount = 4 }},
		{"style", func(o *StoryboardWorkflowStageOptions, _ *domain.Storyboard) { o.ComicStyle = "oil" }},
		{"language", func(o *StoryboardWorkflowStageOptions, _ *domain.Storyboard) { o.Language = "ja" }},
		{"controls", func(o *StoryboardWorkflowStageOptions, _ *domain.Storyboard) {
			c := (&domain.CreativeControls{Tone: "tense"}).Normalized()
			o.CreativeControls = &c
		}},
		{"raw input", func(_ *StoryboardWorkflowStageOptions, b *domain.Storyboard) { b.RawInput = "other plot" }},
		{"parent", func(_ *StoryboardWorkflowStageOptions, b *domain.Storyboard) { b.ParentID = "other-parent" }},
		{"pipeline", func(_ *StoryboardWorkflowStageOptions, b *domain.Storyboard) { b.UseComicPagePipeline = true }},
		{"selected character", func(_ *StoryboardWorkflowStageOptions, b *domain.Storyboard) {
			b.CharacterRefs = []domain.StoryboardCharacterRef{{CharacterID: "other", Role: "lead"}}
		}},
		{"selected location", func(_ *StoryboardWorkflowStageOptions, b *domain.Storyboard) {
			b.SceneRefs = []domain.StoryboardSceneRef{{StorySceneID: "other-location", IsPrimaryScene: true}}
		}},
		{"revision marker", func(o *StoryboardWorkflowStageOptions, _ *domain.Storyboard) { o.RegenerateStructure = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changedOpts := opts
			changedBoard := *board
			tc.change(&changedOpts, &changedBoard)
			if err := validateStoryboardWorkflowRunIntent(run, &changedBoard, changedOpts, true); !errors.Is(err, domain.ErrAlreadyExists) {
				t.Fatalf("wanted intent conflict, got %v", err)
			}
		})
	}
}
func TestStoryboardRevisionRetryNormalizesTargetsAndKeepsOriginalOmittedBaseline(t *testing.T) {
	board := storyboardIntentTestBoard()
	opts := storyboardIntentTestOptions()
	opts.BaseRevision = ""
	run := storyboardIntentTestRun(board, opts)
	opts.TargetSceneIDs = []string{" scene-A ", "scene-B", "scene-A"}
	if err := validateStoryboardWorkflowRunIntent(run, board, opts, true); err != nil {
		t.Fatal(err)
	}
	opts.BaseRevision = "accepted-hash"
	if err := validateStoryboardWorkflowRunIntent(run, board, opts, true); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatal("a changed baseline must conflict", err)
	}
}
func TestPinnedStoryboardStagesAllowFrozenResumeButRejectSuppliedChanges(t *testing.T) {
	board := storyboardIntentTestBoard()
	opts := storyboardIntentTestOptions()
	run := storyboardIntentTestRun(board, opts)
	// Live draft settings can change after this run; a pinned continuation keeps
	// using its snapshot without asking clients to resubmit the full request.
	board.RawInput = "live text changed"
	board.SceneCount = 7
	if err := validateStoryboardWorkflowRunIntent(run, board, StoryboardWorkflowStageOptions{GenerationRunID: run.ID}, false); err != nil {
		t.Fatal(err)
	}
	for _, changed := range []StoryboardWorkflowStageOptions{{GenerationRunID: run.ID, TargetSceneIDs: []string{"other"}}, {GenerationRunID: run.ID, BaseRevision: "other"}, {GenerationRunID: run.ID, SceneCount: 5}, {GenerationRunID: run.ID, UserDirective: "other"}, {GenerationRunID: run.ID, Language: "ja"}, {GenerationRunID: run.ID, ClientRequestID: "other-key"}} {
		if err := validateStoryboardWorkflowRunIntent(run, board, changed, false); !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("pinned run accepted changed options %+v, %v", changed, err)
		}
	}
}

type storyboardIntentRepository struct {
	domain.Repository
	runs    []*domain.StoryboardGenerationRun
	latest  *domain.StoryboardGenerationRun
	findErr error
	created int
}

func (r *storyboardIntentRepository) FindStoryboardGenerationRunsByClientRequest(_ context.Context, board, user, key string) ([]*domain.StoryboardGenerationRun, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	out := []*domain.StoryboardGenerationRun{}
	for _, run := range r.runs {
		if run.StoryboardID == board && run.UserID == user && storyboardWorkflowRunRequestFromRun(run).ClientRequestID == key {
			out = append(out, run)
		}
	}
	return out, nil
}
func (r *storyboardIntentRepository) GetStoryboardGenerationRun(_ context.Context, id string) (*domain.StoryboardGenerationRun, error) {
	for _, run := range r.runs {
		if run.ID == id {
			return run, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *storyboardIntentRepository) LatestStoryboardGenerationRun(context.Context, string) (*domain.StoryboardGenerationRun, error) {
	return r.latest, nil
}
func (r *storyboardIntentRepository) CreateStoryboardGenerationRun(context.Context, *domain.StoryboardGenerationRun) error {
	r.created++
	return errors.New("unexpected new generation")
}

func TestStoryboardWorkflowInterleavedRequestRetriesReuseHistoryAndConflictBeforeSideEffects(t *testing.T) {
	board := storyboardIntentTestBoard()
	a := storyboardIntentTestOptions()
	runA := storyboardIntentTestRun(board, a)
	b := a
	b.ClientRequestID = "turn-B"
	b.UserDirective = "second turn"
	runB := storyboardIntentTestRun(board, b)
	runB.ID = "run-B"
	repo := &storyboardIntentRepository{runs: []*domain.StoryboardGenerationRun{runA, runB}, latest: runB}
	svc := &Service{repo: repo}
	got, _, _, _, err := svc.ensureStoryboardWorkflowRun(context.Background(), board, &domain.Story{}, a)
	if err != nil || got.ID != runA.ID || repo.created != 0 {
		t.Fatalf("interleaved retry did not find A: got=%+v err=%v created=%d", got, err, repo.created)
	}
	a.TargetSceneIDs = []string{"other"}
	if _, _, _, _, err = svc.ensureStoryboardWorkflowRun(context.Background(), board, &domain.Story{}, a); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatal("history replay should conflict", err)
	}
	a.GenerationRunID = runA.ID
	if _, _, _, _, err = svc.ensureStoryboardWorkflowRun(context.Background(), board, &domain.Story{}, a); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatal("explicit run ID should conflict", err)
	}
	if repo.created != 0 {
		t.Fatal("conflict created work")
	}
	repo.findErr = errors.New("database unavailable")
	a.GenerationRunID = ""
	if _, _, _, _, err = svc.ensureStoryboardWorkflowRun(context.Background(), board, &domain.Story{}, a); err == nil || errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatal("history storage failure was hidden", err)
	}
}
func TestLegacyStoryboardRunOnlyResumesWhenSuppliedFactsAreProvable(t *testing.T) {
	board := storyboardIntentTestBoard()
	run := &domain.StoryboardGenerationRun{RequestJSON: `{"clientRequestId":"legacy","regenerateStructure":true,"userDirective":"rain","sceneCount":3,"language":"ja","targetSceneIds":["b","a"],"baseRevision":"hash"}`}
	if err := validateStoryboardWorkflowRunIntent(run, board, StoryboardWorkflowStageOptions{GenerationRunID: "old"}, false); err != nil {
		t.Fatal(err)
	}
	if err := validateStoryboardWorkflowRunIntent(run, board, StoryboardWorkflowStageOptions{TargetSceneIDs: []string{"a", "b"}, BaseRevision: "hash", Language: "ja"}, false); err != nil {
		t.Fatal(err)
	}
	if err := validateStoryboardWorkflowRunIntent(run, board, StoryboardWorkflowStageOptions{ComicStyle: "ink"}, false); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatal("missing legacy evidence should conflict", err)
	}
	if err := validateStoryboardWorkflowRunIntent(run, board, StoryboardWorkflowStageOptions{ClientRequestID: "legacy", RegenerateStructure: true, UserDirective: "rain"}, true); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatal("unfingerprinted keyed replay must not silently reuse", err)
	}
	// Unsupported/corrupt future intent is not interpreted as a legacy default.
	var request map[string]any
	_ = json.Unmarshal([]byte(run.RequestJSON), &request)
	request["requestIntent"] = map[string]any{"version": 2}
	run.RequestJSON = mustJSON(request, "{}")
	if err := validateStoryboardWorkflowRunIntent(run, board, StoryboardWorkflowStageOptions{}, false); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatal("unknown intent version accepted", err)
	}
}

func TestPinnedStoryboardCannotEraseFrozenIntentWithExplicitJSONZeroValues(t *testing.T) {
	board := storyboardIntentTestBoard()
	opts := storyboardIntentTestOptions()
	run := storyboardIntentTestRun(board, opts)
	for _, key := range []string{"clientRequestId", "regenerateStructure", "userDirective", "sceneCount", "comicStyle", "language", "targetSceneIds", "baseRevision"} {
		t.Run(key, func(t *testing.T) {
			zero := StoryboardWorkflowStageOptions{GenerationRunID: run.ID, ProvidedFields: map[string]bool{key: true}}
			if err := validateStoryboardWorkflowRunIntent(run, board, zero, false); !errors.Is(err, ErrStoryboardRequestIntentConflict) {
				t.Fatalf("explicit zero erased %s: %v", key, err)
			}
		})
	}
}
func TestStoryboardKeyedRetryKeepsFrozenDefaultChoicesAfterDraftSettingsChange(t *testing.T) {
	board := storyboardIntentTestBoard()
	opts := storyboardIntentTestOptions()
	opts.SceneCount = 0
	opts.ComicStyle = ""
	run := storyboardIntentTestRun(board, opts)
	board.SceneCount = 7
	board.ContinuationComicStyle = "oil"
	if err := validateStoryboardWorkflowRunIntent(run, board, opts, true); err != nil {
		t.Fatal("a retry must not resolve defaults from mutable settings", err)
	}
	explicit := opts
	explicit.SceneCount = 3
	explicit.ComicStyle = "ink"
	if err := validateStoryboardWorkflowRunIntent(run, board, explicit, true); err != nil {
		t.Fatal("explicit frozen effective choices are equivalent", err)
	}
}
