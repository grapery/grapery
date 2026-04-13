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
	if !strings.Contains(out, "Comic / visual style") {
		t.Fatalf("prompt should mention comic style section")
	}
}
