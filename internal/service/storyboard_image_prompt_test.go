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

func TestMergePlannedStoryboardComicTextsIntoDetails(t *testing.T) {
	details := &domain.ImagePromptDetails{
		ComicTexts: []domain.StoryboardComicText{
			{Type: "thought", Text: ""},
		},
	}
	planned := &domain.StoryboardScene{
		ComicTexts: []domain.StoryboardComicText{
			{Type: "thought", Text: "从场景合并"},
		},
	}
	mergePlannedStoryboardComicTextsIntoDetails(details, planned)
	if details.ComicTexts[0].Text != "从场景合并" {
		t.Fatalf("got %q", details.ComicTexts[0].Text)
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
