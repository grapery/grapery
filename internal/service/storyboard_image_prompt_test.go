package service

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
)

func TestMergeStoryboardSceneDescriptionForImage(t *testing.T) {
	got := mergeStoryboardSceneDescriptionForImage("plot", "detail")
	if !strings.Contains(got, "plot") || !strings.Contains(got, "detail") {
		t.Fatalf("expected both parts: %q", got)
	}
	if mergeStoryboardSceneDescriptionForImage("", "d") != "d" {
		t.Fatal("empty plot")
	}
	if mergeStoryboardSceneDescriptionForImage("p", "") != "p" {
		t.Fatal("empty detail")
	}
}

func TestMergeStoryboardPlannedSceneForImageCarriesVisualContract(t *testing.T) {
	planned := &domain.StoryboardScene{
		ImagePrompt:    "planned rain-station close-up",
		ContinuityNote: "same red coat",
		LayoutIntent:   "single_subject_focus", ShotType: "close_up",
		CompositionPlan: "ticket in foreground", VisualHierarchy: "ticket first",
	}
	out := mergeStoryboardPlannedSceneForImage("她低头看票。", planned)
	for _, want := range []string{"她低头看票", "planned rain-station close-up", "same red coat", "layout_intent=single_subject_focus", "shot_type=close_up"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected planned field %q in merged description: %s", want, out)
		}
	}
}

func TestCompileStoryboardImagePromptFromPlanAvoidsSecondRewrite(t *testing.T) {
	svc := &Service{}
	gen := &domain.StoryboardImageGeneration{
		SceneTitle: "雨夜车站", SceneDescription: "她发现车票日期不对。", ContentLanguage: "zh-Hans",
		PlannedScene: &domain.StoryboardScene{ImagePrompt: "close-up of a wet train ticket", ShotType: "close_up", CompositionPlan: "ticket dominates foreground"},
	}
	if !compileStoryboardImagePromptFromPlan(svc, gen) {
		t.Fatal("expected deterministic compilation from persisted scene plan")
	}
	for _, want := range []string{"close-up of a wet train ticket", "ticket dominates foreground", "她发现车票日期不对"} {
		if !strings.Contains(gen.GeneratedPrompt, want) {
			t.Fatalf("expected %q in compiled prompt: %s", want, gen.GeneratedPrompt)
		}
	}
}

func TestStoryboardReferenceManifestKeepsSemanticOrderAndDeduplicates(t *testing.T) {
	seen := map[string]struct{}{}
	var refs []domain.StoryboardImageReference
	refs = appendStoryboardImageReference(refs, seen, "https://img/previous.png", domain.StoryboardImageReferencePreviousPanel, "scene_0", 6)
	refs = appendStoryboardImageReference(refs, seen, "https://img/hero.png", domain.StoryboardImageReferenceCharacter, "char_hero", 6)
	refs = appendStoryboardImageReference(refs, seen, "https://img/previous.png", domain.StoryboardImageReferenceUser, "", 6)
	refs = appendStoryboardImageReference(refs, seen, "https://img/mood.png", domain.StoryboardImageReferenceUser, "", 6)

	if len(refs) != 3 {
		t.Fatalf("expected duplicate URL to be removed, got %#v", refs)
	}
	wantRoles := []string{
		domain.StoryboardImageReferencePreviousPanel,
		domain.StoryboardImageReferenceCharacter,
		domain.StoryboardImageReferenceUser,
	}
	for i, want := range wantRoles {
		if refs[i].Role != want {
			t.Fatalf("reference %d role = %q, want %q", i, refs[i].Role, want)
		}
	}
	urls := storyboardReferenceURLs(refs)
	if strings.Join(urls, ",") != "https://img/previous.png,https://img/hero.png,https://img/mood.png" {
		t.Fatalf("unexpected provider URL order: %#v", urls)
	}
}

func TestStoryboardVisualScenePromptInputCarriesTypedReferencesAndPlan(t *testing.T) {
	gen := &domain.StoryboardImageGeneration{
		SceneTitle:       "A silent arrival",
		SceneDescription: "The traveler enters the empty station.",
		ContentLanguage:  "en",
		PlannedScene: &domain.StoryboardScene{
			ImagePrompt:     "wide shot, rain-lit platform",
			ContinuityNote:  "same red coat",
			LayoutIntent:    "wide_establishing",
			CompositionPlan: "traveler left, tracks lead right",
			ShotType:        "wide_shot",
			VisualHierarchy: "traveler, rails, station",
		},
		ReferenceManifest: []domain.StoryboardImageReference{
			{URL: "https://img/previous.png", Role: domain.StoryboardImageReferencePreviousPanel},
			{URL: "https://img/hero.png", Role: domain.StoryboardImageReferenceCharacter, Key: "char_hero"},
		},
	}
	input := visualSceneSpecPromptInput(storyboardVisualSceneSpec(gen))
	if input["plannedVisualPrompt"] != "wide shot, rain-lit platform" || input["contentLanguage"] != "en" {
		t.Fatalf("planned scene contract was not preserved: %#v", input)
	}
	refs, ok := input["references"].([]map[string]string)
	if !ok || len(refs) != 2 || refs[0]["role"] != domain.StoryboardImageReferencePreviousPanel || refs[1]["key"] != "char_hero" {
		t.Fatalf("typed references missing from prompt input: %#v", input["references"])
	}
}

func TestStoryboardComicTextLimitsFollowContentLanguage(t *testing.T) {
	english := normalizeStoryboardComicTextsForLanguage([]domain.StoryboardComicText{{Type: "dialogue", Text: strings.Repeat("a", 40)}}, "en")
	chinese := normalizeStoryboardComicTextsForLanguage([]domain.StoryboardComicText{{Type: "dialogue", Text: strings.Repeat("字", 20)}}, "zh-Hans")
	if len([]rune(english[0].Text)) != 32 {
		t.Fatalf("English comic text should allow 32 characters, got %q", english[0].Text)
	}
	if len([]rune(chinese[0].Text)) != 12 {
		t.Fatalf("CJK comic text should allow 12 characters, got %q", chinese[0].Text)
	}
}

func TestStoryboardFinalPromptPrefersExplicitLanguageOverTextGuess(t *testing.T) {
	var svc Service
	details := &domain.ImagePromptDetails{ComicTexts: []domain.StoryboardComicText{{Type: "dialogue", Text: "出発"}}}
	out := svc.combineImagePrompt(details, "東京駅", "少年出発", "ja")
	if !strings.Contains(out, "supplied Japanese text") {
		t.Fatalf("explicit Japanese language should override Han-only text inference: %s", out)
	}
}

func TestWordlessStoryboardPromptForbidsEmptyLetteringContainers(t *testing.T) {
	var svc Service
	out := svc.combineImagePrompt(&domain.ImagePromptDetails{Composition: "quiet wide shot"}, "Silent", "Empty station")
	for _, want := range []string{"wordless", "Do not draw speech balloons", "caption boxes", "pseudo-readable text"} {
		if !strings.Contains(out, want) {
			t.Fatalf("wordless prompt missing %q: %s", want, out)
		}
	}
}

func TestPrependStoryboardImageNarrativeBlock(t *testing.T) {
	nb := storyboardSceneNarrativeBlock("T", "Desc line")
	out := prependStoryboardImageNarrativeBlock(nb, "Art style: x")
	if !strings.HasPrefix(out, "【分镜叙事") {
		t.Fatalf("expected narrative prefix: %q", out)
	}
	if !strings.Contains(out, "Art style: x") {
		t.Fatalf("expected beauty tail: %q", out)
	}
}

func TestTruncateStoryboardImagePromptPreservingNarrative(t *testing.T) {
	nb := storyboardSceneNarrativeBlock("T", strings.Repeat("汉", 30))
	beauty := strings.Repeat("B", 500)
	full := nb + "\n\n【画面与镜头 / Visual direction】\n" + beauty
	out := truncateStoryboardImagePromptPreservingNarrative(nb, full, 120)
	if !strings.HasPrefix(out, "【分镜叙事") {
		t.Fatalf("narrative should be preserved at start: %q", truncateForTest(out, 80))
	}
	if utf8.RuneCountInString(out) > 120 {
		t.Fatalf("expected max 120 runes, got %d", utf8.RuneCountInString(out))
	}
}

func truncateForTest(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

func TestCombineImagePrompt_LetteringBeforeLongVisualNotes(t *testing.T) {
	var s Service
	pi := 1
	details := &domain.ImagePromptDetails{
		AdditionalNotes: strings.Repeat("note ", 400),
		ComicTexts: []domain.StoryboardComicText{
			{Type: "thought", Text: "会下雨吗", Speaker: "boy", PanelIndex: &pi},
		},
	}
	out := s.combineImagePrompt(details, "T", "D")
	idxLetter := strings.Index(out, "会下雨吗")
	idxVisual := strings.Index(out, "note ")
	if idxLetter < 0 || idxVisual < 0 {
		t.Fatalf("missing parts: letter=%d visual=%d", idxLetter, idxVisual)
	}
	if idxLetter > idxVisual {
		t.Fatalf("lettering should precede long visual notes inside beauty section: letter=%d visual=%d", idxLetter, idxVisual)
	}
}

func TestCombineImagePrompt_SkipsEmptyComicText(t *testing.T) {
	var s Service
	details := &domain.ImagePromptDetails{
		ComicTexts: []domain.StoryboardComicText{
			{Type: "thought", Text: ""},
			{Type: "thought", Text: "有字"},
		},
	}
	out := s.combineImagePrompt(details, "", "")
	if strings.Contains(out, "「」") {
		t.Fatalf("should not emit empty bracket placeholders: %s", truncateForTest(out, 200))
	}
	if !strings.Contains(out, "有字") {
		t.Fatal("expected non-empty comic text in prompt")
	}
}

func TestApplyPlannedStoryboardComicTextsIsAuthoritative(t *testing.T) {
	details := &domain.ImagePromptDetails{ComicTexts: []domain.StoryboardComicText{{Type: "dialogue", Text: "模型擅自添加的对白"}}}
	planned := &domain.StoryboardScene{ComicTexts: []domain.StoryboardComicText{
		{Type: "narration", Text: "这是规划阶段确定且明显超过十二个汉字的旁白"},
		{Type: "narration", Text: "第二条旁白应丢弃"},
	}}
	applyPlannedStoryboardComicTextsToDetails(details, planned)
	if len(details.ComicTexts) != 1 || details.ComicTexts[0].Type != "narration" {
		t.Fatalf("planned lettering should replace normalizer output: %#v", details.ComicTexts)
	}
	if len([]rune(details.ComicTexts[0].Text)) > 12 {
		t.Fatalf("planned lettering should be capped: %q", details.ComicTexts[0].Text)
	}
}

func TestEstimateComicPagePanelCountRangeAndCompactness(t *testing.T) {
	transition := estimateComicPagePanelCount(nil, "夜色笼罩营地。", true, 6)
	if transition != 1 {
		t.Fatalf("short transition should stay one panel, got %d", transition)
	}

	dense := &domain.StoryboardScene{
		Sequence:    5,
		BeatPurpose: "climax reveal",
		Characters:  []string{"A", "B"},
		ComicTexts: []domain.StoryboardComicText{
			{Type: "dialogue", Text: "别动"},
			{Type: "thought", Text: "要来了"},
			{Type: "sfx", Text: "砰！"},
			{Type: "dialogue", Text: "太好了"},
		},
	}
	got := estimateComicPagePanelCount(dense, strings.Repeat("高潮", 220), false, 6)
	if got < 6 || got > comicPageMaxPanelCount {
		t.Fatalf("dense climax should use 6-7 panels, got %d", got)
	}

	if maxed := estimateComicPagePanelCount(dense, strings.Repeat("高潮", 800), false, 6); maxed > 7 {
		t.Fatalf("panel count must cap at 7, got %d", maxed)
	}
}

func TestResolveComicPagePipelineIgnoresClientPanelCount(t *testing.T) {
	got := resolveComicPagePipeline(ComicPagePipelineOptions{
		LayoutPreset:    "strip5_top2_middle_wide_bottom2",
		PanelCount:      5,
		PageAspectRatio: "9:16",
		DialogueMode:    "auto",
	}, nil, "安静的雨夜。", true, 4)
	if got.PanelCount != 1 {
		t.Fatalf("server should auto-select compact transition count, got %d", got.PanelCount)
	}
	if got.LayoutPreset != "single_panel_full_page" {
		t.Fatalf("layout should follow auto count, got %q", got.LayoutPreset)
	}
}

func TestSelectStoryboardImageRefsAndOperation(t *testing.T) {
	gen := &domain.StoryboardImageGeneration{
		IsTransitionScene: false,
		ReferenceImages:   []string{"http://a", "http://b"},
	}
	refs, op, _ := selectStoryboardImageRefsAndOperation(gen, storyboardImageT2INarrativeMinRunes)
	if op != genapi.OperationTextToImage || len(refs) != 0 {
		t.Fatalf("long narrative should force t2i: op=%s len=%d", op, len(refs))
	}
	refs2, op2, primary := selectStoryboardImageRefsAndOperation(gen, 10)
	if op2 != genapi.OperationImageToImage || len(refs2) != 2 || primary != "http://a" {
		t.Fatalf("short narrative should keep i2i: op=%s refs=%v primary=%q", op2, refs2, primary)
	}
	genTrans := &domain.StoryboardImageGeneration{
		IsTransitionScene: true,
		ReferenceImages:   []string{"http://p"},
	}
	refs3, op3, _ := selectStoryboardImageRefsAndOperation(genTrans, 500)
	if op3 != genapi.OperationImageToImage || len(refs3) != 1 {
		t.Fatalf("transition should keep i2i: op=%s len=%d", op3, len(refs3))
	}
}
