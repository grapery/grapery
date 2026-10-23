package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"go.uber.org/zap"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

type characterWorkflowIdentityRepo struct {
	domain.Repository
	run *domain.GenerationExecution
}

func (r *characterWorkflowIdentityRepo) GetGenerationExecution(context.Context, string) (*domain.GenerationExecution, error) {
	return r.run, nil
}
func TestCharacterWorkflowRequestValidatesUserKindReleaseAndStory(t *testing.T) {
	req := CharacterGenerationTaskRequest{StoryID: "story", WorkflowRunID: "run", WorkflowReleaseID: "release"}
	repo := &characterWorkflowIdentityRepo{run: &domain.GenerationExecution{UserID: "owner", Kind: "character", WorkflowReleaseID: "release", Input: map[string]any{"storyId": "story"}}}
	svc := &Service{repo: repo}
	if err := svc.validateCharacterWorkflowRequest(context.Background(), "owner", req); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*domain.GenerationExecution){func(r *domain.GenerationExecution) { r.UserID = "other" }, func(r *domain.GenerationExecution) { r.Kind = "fragment" }, func(r *domain.GenerationExecution) { r.WorkflowReleaseID = "other" }, func(r *domain.GenerationExecution) { r.Input = map[string]any{"storyId": "other"} }} {
		copy := *repo.run
		mutate(&copy)
		bad := &Service{repo: &characterWorkflowIdentityRepo{run: &copy}}
		if err := bad.validateCharacterWorkflowRequest(context.Background(), "owner", req); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("forged workflow accepted: %v", err)
		}
	}
	repo.run.Kind = "workflow"
	repo.run.Input["workflowSurface"] = "voyager.character"
	if err := svc.validateCharacterWorkflowRequest(context.Background(), "owner", req); err != nil {
		t.Fatal(err)
	}
	repo.run.Input["workflowSurface"] = "voyager.storyboard"
	if err := svc.validateCharacterWorkflowRequest(context.Background(), "owner", req); err == nil {
		t.Fatal("wrong workflow surface accepted")
	}
}
func TestCharacterCreationRequestHashCoversAllIntentFields(t *testing.T) {
	req := CharacterGenerationTaskRequest{ClientRequestID: "stable", StoryID: "story", SourceType: "manual_form", Name: "雨", Description: "confirmed", ReferenceImage: "https://ref", CreativeControls: &domain.CreativeControls{Tone: "warm"}}
	key, hash, err := characterTaskRequestIdentity("owner", req)
	if err != nil {
		t.Fatal(err)
	}
	sameKey, sameHash, _ := characterTaskRequestIdentity("owner", req)
	if key != sameKey || hash != sameHash {
		t.Fatal("retry changed identity")
	}
	for _, mutate := range []func(*CharacterGenerationTaskRequest){func(r *CharacterGenerationTaskRequest) { r.Appearance = "blue hair" }, func(r *CharacterGenerationTaskRequest) { r.ReferenceImage = "https://other" }, func(r *CharacterGenerationTaskRequest) { r.StoryID = "another-world" }, func(r *CharacterGenerationTaskRequest) { r.ShortTermGoal = "new" }, func(r *CharacterGenerationTaskRequest) { r.CreativeControls = &domain.CreativeControls{Tone: "tense"} }, func(r *CharacterGenerationTaskRequest) {
		r.Suggestion = &domain.FragmentCharacterSuggestion{Name: "other"}
	}} {
		changed := req
		mutate(&changed)
		gotKey, gotHash, err := characterTaskRequestIdentity("owner", changed)
		if err != nil || gotKey != key || gotHash == hash {
			t.Fatal("intent was not bound", gotKey, gotHash, err)
		}
	}
	otherKey, _, _ := characterTaskRequestIdentity("another-owner", req)
	if otherKey == key {
		t.Fatal("request key not user scoped")
	}
	req.ClientRequestID = ""
	req.WorkflowRunID = "agent-child"
	if runKey, _, err := characterTaskRequestIdentity("owner", req); err != nil || runKey == "" {
		t.Fatal("workflow retry has no identity", err)
	}
}
func TestCharacterAttributesUnknownRecoveryAndRepeatedRetryNeverCallsAI(t *testing.T) {
	svc := &Service{} // No AI client/repository; processing/unknown must fail before either is used.
	task := &domain.CharacterGenerationTask{RequestJSON: `{"name":"雨","prompt":"confirmed"}`, Stages: map[string]domain.CharacterGenerationStage{"attributes": {Status: "processing", Attempts: 1}}}
	for i := 0; i < 2; i++ {
		err := svc.executeCharacterGenerationTask(context.Background(), task)
		if err == nil || !strings.Contains(err.Error(), "character_attributes_outcome_unknown") {
			t.Fatal(err)
		}
	}
	if !task.Stages["attributes"].OutcomeUnknown || task.Stages["attributes"].Attempts != 1 {
		t.Fatal(task.Stages)
	}
}

func TestCharacterFullyConfirmedAttributesSkipGenerationAndBilling(t *testing.T) {
	var attrs GeneratedCharacterAttributes
	if err := json.Unmarshal([]byte(validCharacterAttributeJSON()), &attrs); err != nil {
		t.Fatal(err)
	}
	req := CharacterGenerationTaskRequest{Name: "雨", Description: attrs.Description, Personality: attrs.Personality, Background: attrs.Background, Appearance: attrs.Appearance, ShortTermGoal: attrs.ShortTermGoal, LongTermGoal: attrs.LongTermGoal, HandlingStyle: attrs.HandlingStyle, CognitionRange: attrs.CognitionRange, AbilityFeatures: attrs.AbilityFeatures, DressPreference: attrs.DressPreference}
	raw, _ := json.Marshal(req)
	repo := &characterPromptTestRepo{character: &domain.Character{BaseModel: common.BaseModel{ID: "character"}, StoryID: "story", Author: &domain.User{BaseModel: common.BaseModel{ID: "owner"}}, Portrait: "https://confirmed"}}
	svc := &Service{repo: repo, logger: zap.NewNop()} // nil AI; no attributes model or billing is available.
	task := &domain.CharacterGenerationTask{ID: "task", StoryID: "story", CharacterID: "character", UserID: "owner", RequestJSON: string(raw)}
	if err := svc.executeCharacterGenerationTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if task.Stages["attributes"].Status != "completed" || task.Stages["attributes"].Attempts != 0 {
		t.Fatal("paid regeneration of fully confirmed fields", task.Stages)
	}
	if _, err := validateCharacterAttributesOutput(task.AttributesJSON); err != nil {
		t.Fatal(err)
	}
	req.DressPreference = ""
	if _, complete := completeConfirmedCharacterAttributes(characterRequestAttributes(req)); complete {
		t.Fatal("empty field incorrectly treated as confirmed")
	}
}

func TestCharacterDirectAgentRunWithNoReleaseRetainsValidatedIdempotency(t *testing.T) {
	req := CharacterGenerationTaskRequest{StoryID: "story", WorkflowRunID: "direct-character-run"}
	repo := &characterWorkflowIdentityRepo{run: &domain.GenerationExecution{UserID: "owner", Kind: "character", Input: map[string]any{"storyId": "story"}}}
	svc := &Service{repo: repo}
	if err := svc.validateCharacterWorkflowRequest(context.Background(), "owner", req); err != nil {
		t.Fatal("direct agent character broken", err)
	}
	key, hash, err := characterTaskRequestIdentity("owner", req)
	if err != nil || key == "" || hash == "" {
		t.Fatal(key, hash, err)
	}
	repo.run.Kind = "workflow"
	repo.run.Input["workflowSurface"] = "voyager.character"
	if err := svc.validateCharacterWorkflowRequest(context.Background(), "owner", req); err == nil {
		t.Fatal("generic workflow accepted without a pinned release")
	}
}

type characterTaskReplayRepo struct {
	domain.Repository
	domain.CharacterTaskExecutionRepository
	saved *domain.CharacterGenerationTask
	calls int
}

func (r *characterTaskReplayRepo) StoryByID(context.Context, string) (*domain.Story, error) {
	return &domain.Story{UserID: "owner"}, nil
}
func (r *characterTaskReplayRepo) CreateCharacterTaskIdempotent(_ context.Context, proposed *domain.CharacterGenerationTask) (*domain.CharacterGenerationTask, bool, error) {
	r.calls++
	if proposed.RequestKey == "" || proposed.IntentHash == "" {
		return nil, false, errors.New("request identity missing")
	}
	return r.saved, false, nil
}
func TestCharacterCreateEntryRejoinsTaskBeforeSchedulingAnyWork(t *testing.T) {
	saved := &domain.CharacterGenerationTask{ID: "already-running", Status: "processing"}
	repo := &characterTaskReplayRepo{saved: saved}
	svc := &Service{repo: repo}
	for i := 0; i < 2; i++ {
		got, err := svc.StartCharacterGenerationTask(context.Background(), "owner", CharacterGenerationTaskRequest{ClientRequestID: "same-ui-action", StoryID: "story", SourceType: "manual_form", Name: "雨", Description: "confirmed"})
		if err != nil || got != saved {
			t.Fatal(got, err)
		}
	}
	// The embedded execution interface is nil, so scheduling/claiming again would
	// fail. Both requests must return the same durable task without any AI worker.
	if repo.calls != 2 {
		t.Fatal(repo.calls)
	}
}

func TestCharacterLegacyManualRequestWithoutKeyUsesFullIntentIdentity(t *testing.T) {
	for _, source := range []string{domain.CharacterGenerationSourceManualForm, domain.CharacterGenerationSourceManualPrompt} {
		req := CharacterGenerationTaskRequest{SourceType: source, StoryID: "world", Name: "雨", Description: "confirmed"}
		key, hash, err := characterTaskRequestIdentity("owner", req)
		retryKey, retryHash, _ := characterTaskRequestIdentity("owner", req)
		if err != nil || key == "" || key != retryKey || hash != retryHash {
			t.Fatal("legacy manual retry lacks stable identity", key, retryKey, err)
		}
		changed := req
		changed.Appearance = "new appearance"
		changedKey, changedHash, _ := characterTaskRequestIdentity("owner", changed)
		if changedKey == key || changedHash == hash {
			t.Fatal("changed legacy intent reused old task identity")
		}
		repo := &characterTaskReplayRepo{saved: &domain.CharacterGenerationTask{ID: "already-created"}}
		svc := &Service{repo: repo}
		req.StoryID = "world"
		for i := 0; i < 2; i++ {
			got, err := svc.StartCharacterGenerationTask(context.Background(), "owner", req)
			if err != nil || got.ID != "already-created" {
				t.Fatal(got, err)
			}
		}
	}
}
