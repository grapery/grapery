package service

import (
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestResolveStoryboardBiblePromptUsesPinnedWorkflowTemplate(t *testing.T) {
	story := &domain.Story{Title: "Voyager", Genre: "adventure"}
	storyboard := &domain.Storyboard{
		RawInput: "cross the storm",
		PromptSnapshots: map[string]domain.PromptTemplateVersion{
			storyboardWorkflowPromptNodeID: {
				ID: "ptv_bible_1", Key: "storyboard.bible", Type: "chat", Checksum: "checksum",
				SystemTemplate: "system scenes={{.sceneCount}}",
				UserTemplate:   "{{.legacyUserPrompt}}\nAlignment={{.alignmentPrompt}}",
				ModelConfig:    map[string]any{"model": "gemini-2.5-pro", "temperature": 0.6, "maxTokens": 6000},
			},
		},
	}
	resolved := resolveStoryboardBiblePrompt(story, storyboard, storyboardGenerationContextSnapshot{ContextText: "ocean"}, "keep hero identity", 4)
	if !resolved.Applied || resolved.TemplateVersion != "ptv_bible_1" {
		t.Fatalf("workflow prompt was not applied: %+v", resolved)
	}
	if resolved.SystemPrompt != "system scenes=4" || !strings.Contains(resolved.UserPrompt, "Alignment=keep hero identity") {
		t.Fatalf("unexpected rendered prompt: %+v", resolved)
	}
	if resolved.Model != "gemini-2.5-pro" || resolved.Temperature != 0.6 || resolved.MaxTokens != 6000 {
		t.Fatalf("model config was not applied: %+v", resolved)
	}
}

func TestResolveStoryboardBiblePromptFallsBackOnTemplateError(t *testing.T) {
	story := &domain.Story{Title: "Voyager"}
	storyboard := &domain.Storyboard{
		RawInput: "cross the storm",
		PromptSnapshots: map[string]domain.PromptTemplateVersion{
			storyboardWorkflowPromptNodeID: {
				ID: "ptv_bad", Key: "storyboard.bible", Type: "chat", Checksum: "checksum",
				SystemTemplate: "{{.unknownVariable}}",
			},
		},
	}
	resolved := resolveStoryboardBiblePrompt(story, storyboard, storyboardGenerationContextSnapshot{}, "", 3)
	if resolved.Applied || resolved.TemplateVersion != storyboardPromptTemplateVersion || resolved.FallbackReason == "" {
		t.Fatalf("invalid workflow prompt did not fall back: %+v", resolved)
	}
	if !strings.Contains(resolved.SystemPrompt, "storyboard") || !strings.Contains(resolved.UserPrompt, "exactly 3 beats") {
		t.Fatal("legacy prompts were not preserved during fallback")
	}
}

func TestResolveStoryboardScenePromptUsesSceneSlot(t *testing.T) {
	story := &domain.Story{Title: "Voyager"}
	storyboard := &domain.Storyboard{
		RawInput: "cross the storm",
		PromptSnapshots: map[string]domain.PromptTemplateVersion{
			storyboardWorkflowPromptNodeID + ":" + storyboardBiblePromptSlot: {
				ID: "ptv_bible", Key: "storyboard.bible", Type: "chat", Checksum: "bible",
				SystemTemplate: "bible only",
			},
			storyboardWorkflowPromptNodeID + ":" + storyboardScenePromptSlot: {
				ID: "ptv_scene", Key: "storyboard.scene", Type: "chat", Checksum: "scene",
				SystemTemplate: "scene system {{.sceneCount}}",
				UserTemplate:   "plan={{.biblePlanJSON}}",
			},
		},
	}
	resolved := resolveStoryboardScenePrompt(story, storyboard, storyboardGenerationContextSnapshot{}, &domain.StoryboardBiblePlan{}, "", 5)
	if !resolved.Applied || resolved.TemplateVersion != "ptv_scene" || resolved.SystemPrompt != "scene system 5" {
		t.Fatalf("scene slot was not applied: %+v", resolved)
	}
	if !strings.HasPrefix(resolved.UserPrompt, "plan={") {
		t.Fatalf("scene plan variables were not rendered: %s", resolved.UserPrompt)
	}
}

func TestResolveStoryboardJSONRepairPromptUsesRepairSlot(t *testing.T) {
	storyboard := &domain.Storyboard{PromptSnapshots: map[string]domain.PromptTemplateVersion{
		storyboardWorkflowPromptNodeID + ":" + storyboardJSONRepairPromptSlot: {
			ID: "ptv_repair", Key: "storyboard.repair", Type: "text", Checksum: "repair",
			SystemTemplate: "repair {{.step}}", UserTemplate: "{{.failureDetail}} :: {{.brokenOutput}}",
		},
	}}
	resolved := resolveStoryboardJSONRepairPrompt(storyboard, "{broken", "parse error", "scene_plan", "repair_scene")
	if !resolved.Applied || resolved.TemplateVersion != "ptv_repair" || resolved.SystemPrompt != "repair scene_plan" {
		t.Fatalf("repair slot was not applied: %+v", resolved)
	}
	if resolved.UserPrompt != "parse error :: {broken" {
		t.Fatalf("repair variables were not rendered: %s", resolved.UserPrompt)
	}
}
