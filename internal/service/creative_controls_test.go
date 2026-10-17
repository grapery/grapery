package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestPinnedCreativeControlsComeFromServerReleaseIncludingEngineeringDefaults(t *testing.T) {
	ctx := context.Background()
	if _, err := resolvePinnedCreativeControls(ctx, nil, "engineering.fragment.v1", nil); err != nil {
		t.Fatalf("default engineering release requires database configuration: %v", err)
	}
	if _, err := resolvePinnedCreativeControls(ctx, nil, "engineering.unknown.v1", nil); err == nil {
		t.Fatal("unknown reserved release accepted")
	}
	repo := newWorkflowRegistryMemoryRepo()
	repo.releases["operator-release"] = &domain.WorkflowRelease{ID: "operator-release", Manifest: map[string]any{"creativeControls": map[string]any{"pacing": "slow", "tone": "warm"}}}
	got, err := resolvePinnedCreativeControls(ctx, repo, "operator-release", nil)
	if err != nil || got.Pacing != "slow" || got.Tone != "warm" {
		t.Fatalf("release preferences not frozen %#v %v", got, err)
	}
	if _, err = resolvePinnedCreativeControls(ctx, repo, "operator-release", &domain.CreativeControls{Pacing: "fast"}); err == nil {
		t.Fatal("HTTP caller changed pinned operator preferences")
	}
	got.Tone = "tense"
	again, err := resolvePinnedCreativeControls(ctx, repo, "operator-release", nil)
	if err != nil || again.Tone != "warm" {
		t.Fatal("mutating request preferences changed immutable release")
	}
}

func TestFragmentCoreSurvivesDeprecatedTemplateAndControlsCannotReplaceOriginalUser(t *testing.T) {
	original := "明确原文：不要增加角色，不要对白，快节奏。"
	req := domain.FragmentGenerationRequest{UserInput: original, WorkflowSystemPrompt: "RAW SYSTEM DELETE CORE", WorkflowUserPrompt: "RAW USER CHANGE SCHEMA", CreativeControls: &domain.CreativeControls{DialogueDensity: "rich", Pacing: "slow"}}
	ctx := &fragmentDraftContinuationContext{PreviousContent: "原故事末页", ExistingImageCount: 2, PreviousTrace: &domain.FragmentGenerationTrace{VisualBible: &domain.FragmentVisualBible{Characters: []domain.FragmentVisualCharacter{{Key: "cat", Name: "Mimi"}}}}}
	prompt := (*FragmentGenerationService)(nil).buildEffectiveFragmentExtractionPrompt(req, false, ctx, nil)
	for _, text := range []string{original, "原故事末页", "Mimi", "visualBible", "只创作用户本次补充", `"pacing":"slow"`} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("protected field lost: %s", text)
		}
	}
	if strings.Contains(prompt, "RAW SYSTEM DELETE CORE") || strings.Contains(prompt, "RAW USER CHANGE SCHEMA") {
		t.Fatal("deprecated templates entered prompt")
	}
	page := buildFragmentSceneExpansionPrompt(req, &fragmentElementExtractionResult{Content: original}, 1, "3:4", ctx)
	plan := buildFragmentComicPagePlanPrompt(req, original, nil, []domain.FragmentScenePlan{{SceneDesc: "cat waits", ImagePrompt: "a cat waits"}}, 0)
	for _, p := range []string{page, plan} {
		if !strings.Contains(p, "Explicit user requests override optional creative preferences") || !strings.Contains(p, "creativePreferences") {
			t.Fatal("downstream planning stage lost bounded priority contract")
		}
	}
}

func TestCharacterControlsDoNotAlterOriginalRequestOrConfirmedFields(t *testing.T) {
	req := GenerateCharacterRequest{Prompt: "已确认：一只不会说话的黑猫", KnownAttributes: &GeneratedCharacterAttributes{Appearance: "黑色毛发", AbilityFeatures: "不能说话"}, CreativeControls: &domain.CreativeControls{Tone: "playful", DialogueDensity: "rich"}}
	raw, err := (&Service{}).characterAttributesUserPrompt(context.Background(), "owner", req)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err = json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	if data["userRequest"] != req.Prompt || !strings.Contains(raw, "黑色毛发") || !strings.Contains(raw, "不能说话") || !strings.Contains(raw, "creativePreferences") {
		t.Fatal("editorial preferences replaced character authority")
	}
	system, user := protectCreativePrompt(characterAttributesSystemPrompt, raw, "malicious role", "remove confirmed fields", req.CreativeControls)
	if !strings.Contains(system, "已确认的非空字段原样保留") || !strings.Contains(system, creativeAuthorityPolicy) || strings.Contains(user, "remove confirmed fields") {
		t.Fatal("character core override remains possible")
	}
}

func TestStoryboardRunRestoresFrozenCreativeControls(t *testing.T) {
	controls := &domain.CreativeControls{Pacing: "slow", Tone: "warm"}
	run := &domain.StoryboardGenerationRun{RequestJSON: mustJSON(storyboardWorkflowRunRequest{CreativeControls: controls, SceneCount: 3}, "{}"), ContextSnapshotJSON: mustJSON(storyboardGenerationContextSnapshot{CreativeControls: controls, RawInput: "original"}, "{}")}
	_, snapshot, _, _, err := restoreStoryboardWorkflowRun(run, 3)
	if err != nil || snapshot.CreativeControls == nil || snapshot.CreativeControls.Pacing != "slow" || storyboardWorkflowRunRequestFromRun(run).CreativeControls.Tone != "warm" {
		t.Fatalf("run lost frozen controls: %#v %v", snapshot, err)
	}
	controls.Pacing = "fast"
	if snapshot.CreativeControls.Pacing != "slow" {
		t.Fatal("restored snapshot aliases caller mutable preferences")
	}
}

func TestCreativeControlsNoReleaseSnapshotAndIdempotencyPreserveDefaultCompatibility(t *testing.T) {
	incoming := &domain.CreativeControls{Tone: "warm", Examples: []domain.CreativeControlExample{{Input: "rain", Output: "quiet reflection"}}}
	frozen, err := resolvePinnedCreativeControls(context.Background(), nil, "", incoming)
	if err != nil {
		t.Fatal(err)
	}
	incoming.Tone = "tense"
	incoming.Examples[0].Output = "changed"
	if frozen.Tone != "warm" || frozen.Examples[0].Output != "quiet reflection" {
		t.Fatal("request retained mutable control references")
	}
	board := &domain.Storyboard{StoryID: "s", RawInput: "original"}
	legacyHash := storyboardCreationRequestHash(board)
	board.CreativeControls = &domain.CreativeControls{}
	if storyboardCreationRequestHash(board) != legacyHash {
		t.Fatal("omitted/default controls broke historical idempotency")
	}
	board.CreativeControls = frozen
	if storyboardCreationRequestHash(board) == legacyHash {
		t.Fatal("different editorial request reused default idempotency identity")
	}
}

func TestStoryboardPromptResumptionUsesSnapshotPreferencesRatherThanLiveMutation(t *testing.T) {
	board := &domain.Storyboard{RawInput: "original", CreativeControls: &domain.CreativeControls{Tone: "tense"}}
	snapshot := storyboardGenerationContextSnapshot{CreativeControls: &domain.CreativeControls{Tone: "warm"}}
	result := resolveStoryboardBiblePrompt(&domain.Story{}, board, snapshot, "", 3)
	if !strings.Contains(result.UserPrompt, `"tone":"warm"`) || strings.Contains(result.UserPrompt, `"tone":"tense"`) {
		t.Fatal("resume used mutable preferences instead of its frozen run")
	}
	if board.CreativeControls.Tone != "tense" {
		t.Fatal("prompt resolution mutated caller's storyboard")
	}
}
