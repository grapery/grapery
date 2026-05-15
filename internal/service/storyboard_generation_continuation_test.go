package service

import (
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestVideoSceneHasInFlightJob(t *testing.T) {
	scene := "scene-1"
	list := []*domain.StoryboardVideoGeneration{
		{SceneID: scene, Status: domain.GenerationStatusCompleted},
	}
	if videoSceneHasInFlightJob(list, scene) {
		t.Fatal("expected no in-flight when only completed")
	}
	list = append(list, &domain.StoryboardVideoGeneration{SceneID: scene, Status: domain.GenerationStatusPending})
	if !videoSceneHasInFlightJob(list, scene) {
		t.Fatal("expected in-flight when pending exists for scene")
	}
	if videoSceneHasInFlightJob(list, "other-scene") {
		t.Fatal("expected no in-flight for different scene")
	}
}

func TestBuildImageGenerationPromptIncludesComicStyle(t *testing.T) {
	s := &Service{}
	gen := &domain.StoryboardImageGeneration{
		SceneTitle:       "T",
		SceneDescription: "D",
		ComicStyle:       "shonen-action",
	}
	out := s.buildImageGenerationPrompt(gen)
	if !strings.Contains(out, "shonen-action") {
		t.Fatalf("prompt should contain comic style slug, got: %s", out)
	}
	if !strings.Contains(out, "Comic Style Continuation") {
		t.Fatalf("prompt should mention comic style section")
	}
	for _, want := range []string{"# PromptDSL", "prompt_dsl_v1", "# Role", "## Task", "## Inputs", "## Global Visual Config", "## Output Contract"} {
		if !strings.Contains(out, want) {
			t.Fatalf("prompt should contain structured section %q, got: %s", want, out)
		}
	}
}

func TestBuildComicPagePromptUsesStructuredSections(t *testing.T) {
	s := &Service{}
	gen := &domain.StoryboardImageGeneration{
		SceneTitle:       "T",
		SceneDescription: "D",
		ComicStyle:       "shonen-action",
	}
	out := s.buildComicPageImageGenerationLLMPrompt(gen, ComicPagePipelineOptions{
		PanelCount:      4,
		LayoutPreset:    "grid_2x2",
		PageAspectRatio: "3:4",
		DialogueMode:    "auto",
	}, nil, 0)
	for _, want := range []string{"# PromptDSL", "prompt_dsl_v1", "# Role", "## Task", "## Inputs", "## Global Visual Config", "## Detailed Instructions"} {
		if !strings.Contains(out, want) {
			t.Fatalf("comic page prompt should contain structured section %q, got: %s", want, out)
		}
	}
}

func TestBuildComicPagePromptPrefersPreplannedComicMetadata(t *testing.T) {
	s := &Service{}
	gen := &domain.StoryboardImageGeneration{
		SceneTitle:       "湖边冲突",
		SceneDescription: "主角与对手在湖边对峙。",
		ComicStyle:       "shonen-action",
	}
	planned := &domain.StoryboardScene{
		LayoutIntent:    "diagonal_motion",
		CompositionPlan: "画面左下主角，右上对手，中央留白放 SFX。",
		ShotType:        "low_angle",
		VisualHierarchy: "主角动作第一，对手反应第二，背景第三。",
		ComicTexts: []domain.StoryboardComicText{
			{Type: "dialogue", Text: "你别过来", Speaker: "char_1", Position: "speech-bubble"},
		},
	}
	out := s.buildComicPageImageGenerationLLMPrompt(gen, ComicPagePipelineOptions{
		PanelCount:      5,
		LayoutPreset:    "strip5_top2_middle_wide_bottom2",
		PageAspectRatio: "9:16",
		DialogueMode:    "auto",
	}, planned, 0)
	for _, want := range []string{
		"Pre-planned comic metadata from text stage (highest priority)",
		"layoutIntent: diagonal_motion",
		"compositionPlan: 画面左下主角",
		"shotType: low_angle",
		"visualHierarchy: 主角动作第一",
		"type=dialogue",
		"text=你别过来",
		"preplannedLayoutIntent",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("comic page prompt should include %q, got: %s", want, out)
		}
	}
}
