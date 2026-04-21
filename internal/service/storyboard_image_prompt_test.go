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
