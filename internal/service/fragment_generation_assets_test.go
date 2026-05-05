package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestBuildFragmentGenerationAssets_SceneFinalUsesTaskTokenScope(t *testing.T) {
	trace := &domain.FragmentGenerationTrace{
		Metrics: []domain.FragmentGenerationStepMetric{
			{Name: "scene_images", Tokens: 222, Provider: "gemini", Model: "image-2"},
		},
	}
	result := &domain.FragmentGenerationResult{
		ImageUrls:       []string{"https://img.example/scene.png"},
		TokensUsed:      999,
		ScenePlan:       []domain.FragmentScenePlan{{Index: 0, GeneratedImageURL: "https://img.example/scene.png"}},
		GenerationTrace: trace,
	}
	assets := buildFragmentGenerationAssets("frag-1", "task-1", domain.FragmentGenerationAssetSourceAIGeneration, "4:3", nil, result, nil)
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	got := assets[0]
	if got.Provider != "gemini" || got.Model != "image-2" {
		t.Fatalf("unexpected provider/model: %s/%s", got.Provider, got.Model)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(got.MetadataJSON), &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["tokensScope"] != "task_total" {
		t.Fatalf("expected tokensScope=task_total, got %#v", meta["tokensScope"])
	}
}

func TestBuildFragmentGenerationAssets_PanelUsesPerPanelMetric(t *testing.T) {
	trace := &domain.FragmentGenerationTrace{
		Metrics: []domain.FragmentGenerationStepMetric{
			{Name: "generating_panel_0", Tokens: 34, Provider: "huoshan", Model: "doubao"},
		},
	}
	result := &domain.FragmentGenerationResult{GenerationTrace: trace}
	panels := []domain.FragmentPanelResultItem{{Index: 0, ImageURL: "https://img.example/panel.png"}}
	assets := buildFragmentGenerationAssets("frag-2", "task-2", domain.FragmentGenerationAssetSourcePanelGeneration, "3:4", nil, result, panels)
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	got := assets[0]
	if got.TokensUsed != 34 {
		t.Fatalf("expected per-panel tokens 34, got %d", got.TokensUsed)
	}
	if got.Provider != "huoshan" || got.Model != "doubao" {
		t.Fatalf("unexpected provider/model: %s/%s", got.Provider, got.Model)
	}
}

func TestComicPromptDirectiveIncludesReadabilityLimits(t *testing.T) {
	var b strings.Builder
	writeFragmentComicLayoutDirective(&b, &domain.FragmentVisualBible{StyleBible: &domain.FragmentVisualStyleBible{ArtStyle: "manga"}}, domain.FragmentScenePlan{})
	out := b.String()
	if !strings.Contains(out, "at most 1 narration box") {
		t.Fatalf("missing narration limit: %s", out)
	}
	if !strings.Contains(out, "<=12 characters") {
		t.Fatalf("missing chinese length guard: %s", out)
	}
}

func TestFragmentCharacterSuggestionsFromContent_UsesNeutralName(t *testing.T) {
	s := &Service{}
	suggestions := s.fragmentCharacterSuggestionsFromContent(&domain.Fragment{
		Caption: "这是一段不应该当名字的标题",
		Content: "主角推开门，风吹进来。",
	})
	if len(suggestions) != 1 {
		t.Fatalf("expected one suggestion, got %d", len(suggestions))
	}
	if suggestions[0].Name != "碎片主角" {
		t.Fatalf("expected neutral fallback name, got %q", suggestions[0].Name)
	}
}
