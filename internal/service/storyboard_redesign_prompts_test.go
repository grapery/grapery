package service

import (
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestValidateStoryboardBiblePlanRequiresComicFields(t *testing.T) {
	plan := &domain.StoryboardBiblePlan{
		StoryboardBible: domain.StoryboardVisualBible{
			StyleBible: &domain.StoryboardVisualStyleBible{ArtStyle: "cinematic inked manga"},
			Characters: []domain.StoryboardVisualCharacter{{Key: "char_1"}},
		},
		Beats: []domain.StoryboardBeat{
			{
				Index:         0,
				Purpose:       "establish",
				Summary:       "角色进入场景",
				ReferenceKeys: []string{"char_1"},
			},
		},
	}
	err := validateStoryboardBiblePlan(plan, 1)
	if err == nil || !strings.Contains(err.Error(), "comicFunction or layoutHint") {
		t.Fatalf("expected comic fields validation error, got: %v", err)
	}

	plan.Beats[0].ComicFunction = "establish"
	plan.Beats[0].LayoutHint = "wide_establishing"
	if err := validateStoryboardBiblePlan(plan, 1); err != nil {
		t.Fatalf("expected valid plan, got error: %v", err)
	}
}

func TestValidateStoryboardScenePlanRequiresComicLayoutMetadata(t *testing.T) {
	bible := &domain.StoryboardBiblePlan{
		StoryboardBible: domain.StoryboardVisualBible{
			StyleBible: &domain.StoryboardVisualStyleBible{ArtStyle: "inked dramatic comic"},
			Characters: []domain.StoryboardVisualCharacter{{Key: "char_1"}},
		},
		Beats: []domain.StoryboardBeat{{Index: 0, Purpose: "p", Summary: "s", ComicFunction: "dialogue", LayoutHint: "split_screen_two_beat"}},
	}
	scenePlan := &domain.StoryboardScenePlan{
		Content: "content",
		Scenes: []domain.StoryboardScenePlanItem{
			{
				Sequence:      0,
				Title:         "scene",
				Description:   "desc",
				ImagePrompt:   "english prompt",
				ReferenceKeys: []string{"char_1"},
			},
		},
	}
	err := validateStoryboardScenePlan(scenePlan, bible, 1)
	if err == nil || !strings.Contains(err.Error(), "comic layout metadata") {
		t.Fatalf("expected comic layout metadata error, got: %v", err)
	}

	scenePlan.Scenes[0].LayoutIntent = "comic_single_panel"
	scenePlan.Scenes[0].CompositionPlan = "主角居中，右上预留气泡区。"
	scenePlan.Scenes[0].ShotType = "medium_shot"
	scenePlan.Scenes[0].VisualHierarchy = "主角第一，背景第二。"
	scenePlan.Scenes[0].PanelShape = "full"
	if err := validateStoryboardScenePlan(scenePlan, bible, 1); err != nil {
		t.Fatalf("expected valid scene plan, got: %v", err)
	}
}

func TestStoryboardRedesignPromptsMentionEmotionalComicBeats(t *testing.T) {
	system := buildStoryboardBiblePlanSystemPrompt()
	if !strings.Contains(system, "turning points") || !strings.Contains(system, "shock beats") {
		t.Fatalf("bible plan system prompt should require emotional comic beats, got: %s", system)
	}

	user := buildStoryboardBiblePlanUserPrompt(
		&domain.Story{Title: "T", Genre: "adventure"},
		&domain.Storyboard{RawInput: "少年在营地发现秘密。"},
		storyboardGenerationContextSnapshot{},
		5,
	)
	for _, want := range []string{"turning_point", "shock", "anticipation", "celebration", "speech_balloon_safe_space"} {
		if !strings.Contains(user, want) {
			t.Fatalf("bible plan user prompt missing %q", want)
		}
	}

	sceneSystem := buildStoryboardSceneWriterSystemPrompt()
	if !strings.Contains(sceneSystem, "SFX/interjections") || !strings.Contains(sceneSystem, "inner_monologue") {
		t.Fatalf("scene writer system prompt missing comic text emphasis: %s", sceneSystem)
	}
}

