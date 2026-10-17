package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestFragmentStrictPageBeatsRepairWithoutInventingFallback(t *testing.T) {
	calls := 0
	got, tokens, err := requestFragmentSceneExpansion(context.Background(), "original user contract", 2, func(prompt string, budget int32) (string, int, error) {
		calls++
		if budget <= 1000 {
			t.Fatal("multi-page output still uses legacy budget")
		}
		if calls == 1 {
			return `{"scenes":[{"index":0,"sceneDesc":"first","imagePrompt":"one"}]}`, 3, nil
		}
		if !strings.Contains(prompt, "SCHEMA REPAIR") || !strings.Contains(prompt, "original user contract") {
			t.Fatal("repair lost context")
		}
		return `{"scenes":[{"index":0,"sceneDesc":"arrive at door","imagePrompt":"hero arrives"},{"index":1,"sceneDesc":"door opens","imagePrompt":"door opens"}]}`, 4, nil
	})
	if err != nil || calls != 2 || len(got) != 2 || tokens != 7 {
		t.Fatalf("unexpected repair %v %d %d %d", err, calls, len(got), tokens)
	}
	calls = 0
	got, _, err = requestFragmentSceneExpansion(context.Background(), "story", 8, func(string, int32) (string, int, error) { calls++; return "truncated", 1, nil })
	if err == nil || len(got) != 0 || calls != 2 {
		t.Fatal("invalid output must not become paid generic panels")
	}
}

func TestFragmentStrictPageBeatsRejectCountIndicesEmptyAndDuplicates(t *testing.T) {
	for _, raw := range []string{
		`{"scenes":[{"index":1,"sceneDesc":"one","imagePrompt":"one"}]}`,
		`{"scenes":[{"index":0,"sceneDesc":"","imagePrompt":"one"}]}`,
		`{"scenes":[{"index":0,"sceneDesc":"one","imagePrompt":""}]}`,
	} {
		if _, err := parseStrictFragmentSceneExpansion(raw, 1); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if _, err := parseStrictFragmentSceneExpansion(`{"scenes":[{"index":0,"sceneDesc":"same","imagePrompt":"one"},{"index":1,"sceneDesc":"same","imagePrompt":"two"}]}`, 2); err == nil {
		t.Fatal("duplicate beats accepted")
	}
}

func TestFragmentComicPlanRepairsSchemaAndFailsClosedAfterTwoInvalidPlans(t *testing.T) {
	plan := domain.FragmentComicPagePlan{PanelCount: 2, Panels: []domain.FragmentComicPanelPlan{
		{Index: 0, SceneDesc: "arrives", ImagePrompt: "hero arrives", NewInformation: "arrival", DramaticIntent: "setup", SilentIntent: "pause"},
		{Index: 1, SceneDesc: "discovers", ImagePrompt: "hero discovers", NewInformation: "clue", DramaticIntent: "discovery", SilentIntent: "surprise"},
	}}
	raw, _ := json.Marshal(plan)
	calls := 0
	got, tokens, err := requestFragmentComicPagePlan(context.Background(), "user contract", nil, func(prompt, step string) (string, int, error) {
		calls++
		if calls == 1 {
			return `{}`, 2, nil
		}
		if step != "fragment_comic_page_revision" {
			t.Fatal("repair not tagged")
		}
		return string(raw), 5, nil
	})
	if err != nil || got.PlanningStatus != "revised" || tokens != 7 || calls != 2 {
		t.Fatalf("unexpected result %#v %v", got, err)
	}
	got, _, err = requestFragmentComicPagePlan(context.Background(), "story", nil, func(string, string) (string, int, error) { return "", 0, errors.New("provider unavailable") })
	if err == nil || got != nil {
		t.Fatal("unavailable planner produced a fallback")
	}
}

func TestFragmentVisualEditReusesPanelNarrativeAndDoesNotMutateAuthority(t *testing.T) {
	plan := buildFallbackFragmentComicPagePlan(domain.FragmentScenePlan{SceneDesc: "he opens the letter", ImagePrompt: "red scarf boy opens letter"}, 2, "en")
	plan.Panels[0].ComicTexts = []domain.FragmentComicText{{ID: "letter-text", Text: "Come home", RenderMode: "overlay"}}
	original := &domain.FragmentComicDocument{VisualBible: &domain.FragmentVisualBible{Characters: []domain.FragmentVisualCharacter{{Key: "boy", Name: "Sam"}}}, StoryState: domain.FragmentStoryState{LastPageResult: "he reaches home"}, Pages: []domain.FragmentComicPageDocument{
		{PageIndex: 0, PageIntent: "arrives", BackgroundImageURL: "first.jpg"},
		{PageIndex: 1, PageIntent: "opens letter", Plan: plan, BackgroundImageURL: "second.jpg"},
		{PageIndex: 2, PageIntent: "goes home", BackgroundImageURL: "third.jpg"},
	}}
	snapshot, _ := json.Marshal(original)
	edit := &fragmentDraftEditContext{Content: "original complete story", Trace: &domain.FragmentGenerationTrace{VisualBible: original.VisualBible}, Document: original, TargetIndex: 1, ImageURLs: []string{"first.jpg", "second.jpg", "third.jpg"}}
	scenes, ok := fragmentExistingPageEditPlan(edit, "make it rainy")
	if !ok || len(scenes) != 1 || len(scenes[0].ComicPage.Panels) != 2 {
		t.Fatal("structured edit did not reuse selected page")
	}
	if scenes[0].ComicPage.Panels[0].ComicTexts[0].Text != "Come home" || !strings.Contains(scenes[0].ComicPage.Panels[0].ImagePrompt, "make it rainy") {
		t.Fatal("edit lost text or visual instruction")
	}
	incoming := &domain.FragmentComicDocument{VisualBible: &domain.FragmentVisualBible{Characters: []domain.FragmentVisualCharacter{{Key: "wrong"}}}, StoryState: domain.FragmentStoryState{LastPageResult: "wrong"}, Pages: []domain.FragmentComicPageDocument{{Plan: *scenes[0].ComicPage, PageIntent: "opens letter", BackgroundImageURL: "edited.jpg"}}}
	merged := applyFragmentComicDocumentEdit(original, incoming, 0, 2)
	if !reflect.DeepEqual(merged.VisualBible, original.VisualBible) || !reflect.DeepEqual(merged.StoryState, original.StoryState) || !reflect.DeepEqual(merged.Pages[0], original.Pages[0]) || !reflect.DeepEqual(merged.Pages[2], original.Pages[2]) {
		t.Fatal("local edit rewrote global authority or unrelated pages")
	}
	after, _ := json.Marshal(original)
	if string(snapshot) != string(after) {
		t.Fatal("edit mutated persisted authority by alias")
	}
	req := fragmentRequestWithEditContext(domain.FragmentGenerationRequest{UserInput: "rain"}, edit)
	for _, want := range []string{"original complete story", "targetAndNeighborPages", "lockedStoryState", "he reaches home", "opens letter"} {
		if !strings.Contains(req.UserInput, want) {
			t.Fatalf("lost edit context %s", want)
		}
	}
}

func TestFragmentWorkflowOverrideCannotRemoveContinuationContext(t *testing.T) {
	req := domain.FragmentGenerationRequest{WorkflowUserPrompt: "CUSTOM EDITORIAL TEMPLATE"}
	ctx := &fragmentDraftContinuationContext{PreviousContent: "old authoritative story", ExistingImageCount: 3, PreviousTrace: &domain.FragmentGenerationTrace{VisualBible: &domain.FragmentVisualBible{Characters: []domain.FragmentVisualCharacter{{Key: "hero", Name: "Nora"}}}}}
	prompt := (*FragmentGenerationService)(nil).buildEffectiveFragmentExtractionPrompt(req, false, ctx, nil)
	if strings.Contains(prompt, "CUSTOM EDITORIAL TEMPLATE") {
		t.Fatal("deprecated raw template entered production prompt")
	}
	for _, want := range []string{"old authoritative story", "Nora", "只创作用户本次补充"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("template override removed %s", want)
		}
	}
}

func TestFragmentIdentityReferenceAssetsCannotBeCrowdedOutByUploads(t *testing.T) {
	scene := domain.FragmentScenePlan{ReferenceKeys: []string{"prop", "hero"}}
	assets := []domain.FragmentReferenceAsset{{Key: "prop", Kind: "prop", ImageURL: "prop.jpg"}, {Key: "hero", Kind: "character", ImageURL: "hero.jpg"}}
	urls := mergeFragmentSceneReferenceAssets([]string{"inspiration1", "inspiration2", "inspiration3"}, scene, assets, 2)
	if !reflect.DeepEqual(urls, []string{"hero.jpg", "prop.jpg"}) {
		t.Fatalf("identity assets crowded out: %v", urls)
	}
}

func TestFragmentQualityAuditFailureIsNotPassAndPageBlocksSurviveOtherEdits(t *testing.T) {
	for _, raw := range []string{`{}`, `{"issues":null}`, `{"issues":[{"severity":"unknown","detail":"bad"}]}`} {
		if _, err := parseFragmentConsistencyIssues(raw); err == nil {
			t.Fatalf("invalid audit passed: %s", raw)
		}
	}
	issues, err := parseFragmentConsistencyIssues(`{"issues":[]}`)
	if err != nil || len(issues) != 0 {
		t.Fatal("explicit clean audit invalid")
	}
	for _, tc := range []struct {
		reason, provider  string
		expected, audited int
		want              string
	}{
		{"", "huoshan", 2, 2, domain.FragmentQualityPassed},
		{"parse_failed", "huoshan", 2, 2, domain.FragmentQualityUnreviewed},
		{"", "huoshan", 2, 1, domain.FragmentQualityUnreviewed},
		{"consistency_off", "", 2, 0, domain.FragmentQualityUnreviewed},
	} {
		if got := fragmentGenerationQuality(nil, tc.reason, tc.provider, tc.expected, tc.audited); got.Status != tc.want {
			t.Fatalf("unexpected quality %#v", got)
		}
	}
	blocked := fragmentGenerationQuality([]domain.FragmentConsistencyIssue{{Severity: "high"}}, "", "huoshan", 2, 2)
	passed := &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}
	old := &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{{PageIndex: 0, Quality: blocked}, {PageIndex: 1, Quality: blocked}}}
	current := &domain.FragmentComicDocument{Quality: passed, Pages: []domain.FragmentComicPageDocument{{Quality: passed}}}
	got := applyFragmentComicDocumentEdit(old, current, 0, 1)
	if got.Quality.Status != domain.FragmentQualityBlocked {
		t.Fatal("editing page 1 cleared page 2 block")
	}
	got = applyFragmentComicDocumentEdit(got, current, 0, 2)
	if got.Quality.Status != domain.FragmentQualityPassed {
		t.Fatal("repairing all blocked pages did not clear block")
	}
}

func TestFragmentReplacementTraceKeepsOtherPagesAcrossLegacyIndexConventions(t *testing.T) {
	old := &domain.FragmentGenerationTrace{Scenes: []domain.FragmentScenePlan{
		{Index: 0, SceneDesc: "one", GeneratedImageURL: "one.jpg"},
		{Index: 1, SceneDesc: "two", GeneratedImageURL: "two.jpg"},
		{Index: 2, SceneDesc: "three", GeneratedImageURL: "three.jpg"},
	}}
	got := mergeFragmentReplacementTraceScenes(old, []domain.FragmentScenePlan{{SceneDesc: "edited two", GeneratedImageURL: "new.jpg"}}, 2, []string{"one.jpg", "two.jpg", "three.jpg"})
	if len(got) != 3 || got[1].SceneDesc != "edited two" || got[2].SceneDesc != "three" || old.Scenes[1].SceneDesc != "two" {
		t.Fatal("replacement targeted mixed historical scene index instead of original image identity")
	}
}

func TestFragmentMigratingLegacyPagesDoesNotInventFailedQuality(t *testing.T) {
	page := buildFallbackFragmentComicPagePlan(domain.FragmentScenePlan{SceneDesc: "legacy"}, 2, "en")
	trace := &domain.FragmentGenerationTrace{Scenes: []domain.FragmentScenePlan{{ComicPage: &page, GeneratedImageURL: "old.jpg"}}}
	doc := fragmentComicDocumentFromExistingDraft(domain.FragmentGenerationRequest{}, "old", "story", "3:4", []string{"old.jpg"}, trace)
	if doc.Quality != nil || doc.Pages[0].Quality != nil {
		t.Fatal("legacy migration retroactively blocked a previously usable page")
	}
}

func TestFragmentAuditInvalidTargetsCannotBecomeCleanAudit(t *testing.T) {
	if err := validateFragmentAuditTargets([]domain.FragmentConsistencyIssue{{SceneIndex: 9, Severity: "high"}}, []string{"new.jpg"}, 1); err == nil {
		t.Fatal("unknown page accepted")
	}
	if err := validateFragmentAuditTargets([]domain.FragmentConsistencyIssue{{ImageURL: "baseline.jpg", Severity: "high"}}, []string{"new.jpg"}, 1); err == nil {
		t.Fatal("baseline issue accepted as generated page")
	}
}

func TestFragmentRecheckAuditsOnlyPendingPagesAndPreservesExistingWork(t *testing.T) {
	original := &domain.FragmentComicDocument{Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityUnreviewed}, Pages: []domain.FragmentComicPageDocument{
		{PageIndex: 0, BackgroundImageURL: "passed.jpg", Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}},
		{PageIndex: 1, BackgroundImageURL: "pending.jpg", TextLayers: []domain.FragmentComicText{{ID: "exact-text", Text: "你好"}}, Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityUnreviewed}},
		{PageIndex: 2, BackgroundImageURL: "blocked.jpg", Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityBlocked}},
	}}
	calls := 0
	got, _, tokens, err := recheckFragmentDocument(context.Background(), original, func(page domain.FragmentComicPageDocument) fragmentPageAuditResult {
		calls++
		if page.PageIndex != 1 {
			t.Fatal("rechecked an already passed/blocked page")
		}
		return fragmentPageAuditResult{Provider: "vision", Audited: 1, Expected: 1, Tokens: 12}
	})
	if err != nil || calls != 1 || tokens != 12 || got.Pages[1].Quality.Status != domain.FragmentQualityPassed || got.Quality.Status != domain.FragmentQualityBlocked {
		t.Fatalf("unexpected recheck %v %#v", err, got)
	}
	if !reflect.DeepEqual(original.Pages[1].TextLayers, got.Pages[1].TextLayers) || original.Pages[1].Quality.Status != domain.FragmentQualityUnreviewed {
		t.Fatal("recheck changed text or mutated persisted source")
	}
	got, _, _, err = recheckFragmentDocument(context.Background(), original, func(domain.FragmentComicPageDocument) fragmentPageAuditResult {
		return fragmentPageAuditResult{Skipped: "provider_unavailable", Expected: 1}
	})
	if err != nil || got.Pages[1].Quality.Status != domain.FragmentQualityUnreviewed {
		t.Fatal("temporary audit outage became pass")
	}
}

func TestFragmentSponsoredBudgetPrecheckNeverTouchesUserBalance(t *testing.T) {
	if err := (*FragmentGenerationService)(nil).ensureFragmentRenderBalance(domain.WithSponsoredGeneration(context.Background()), "owner", []domain.FragmentScenePlan{{}}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFragmentSettlementRecoveryRequiresCompletePersistedWork(t *testing.T) {
	task := &domain.FragmentGenerationTask{Status: "failed", ErrorMessage: "billing_unsettled: unavailable", Result: &domain.FragmentGenerationResult{DraftFragmentID: "draft", ExpectedImageCount: 1, ImageSlots: []domain.FragmentGenerationImageSlot{{Index: 1, Status: "completed", ImageURL: "finished.jpg"}}, ComicDocument: &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{{BackgroundImageURL: "finished.jpg"}}}, GenerationTrace: &domain.FragmentGenerationTrace{BillingStatus: "pending", TextBillingTokens: 72}}}
	if !fragmentSettlementRecoverable(task) {
		t.Fatal("complete pending work cannot recover")
	}
	task.Result.ImageSlots[0].Status = "failed"
	if fragmentSettlementRecoverable(task) {
		t.Fatal("partial images could skip regeneration via settlement")
	}
	task.Result.ImageSlots[0].Status = "completed"
	task.ErrorMessage = "image generation failed"
	if fragmentSettlementRecoverable(task) {
		t.Fatal("ordinary failure misclassified as pending bill")
	}
	if got := fragmentTextBillingTokens(10, 20, 30, 40); got != 100 {
		t.Fatalf("text debit includes non-text use: %d", got)
	}
}

func TestFragmentEditsReuseExistingIdentityAssetsInsteadOfBillingDiscardedCopies(t *testing.T) {
	candidates := []fragmentReferenceAssetCandidate{{Key: "hero", Kind: "character"}, {Key: "new-character", Kind: "character"}, {Key: "missing-image", Kind: "prop"}}
	existing := []domain.FragmentReferenceAsset{{Key: "hero", Kind: "character", ImageURL: "authoritative-hero.jpg"}, {Key: "missing-image", Kind: "prop"}}
	got := missingFragmentReferenceCandidates(candidates, existing)
	if len(got) != 2 || got[0].Key != "new-character" || got[1].Key != "missing-image" {
		t.Fatalf("wrong reused identity selection: %#v", got)
	}
}
