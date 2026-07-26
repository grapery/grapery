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
		ImageSlots:      []domain.FragmentGenerationImageSlot{{ID: "slot-1", Index: 1, Status: "completed", ImageURL: "https://img.example/scene.png"}},
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
	if got.SlotID != "slot-1" {
		t.Fatalf("expected slot id linked, got %q", got.SlotID)
	}
	if got.SlotIndex == nil || *got.SlotIndex != 1 {
		t.Fatalf("expected slot index 1, got %#v", got.SlotIndex)
	}
	linkFragmentGenerationSceneAssetsToSlots(result, assets)
	if result.ImageSlots[0].AssetID == "" {
		t.Fatalf("expected slot asset id to be populated")
	}
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(got.MetadataJSON), &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["tokensScope"] != "task_total" {
		t.Fatalf("expected tokensScope=task_total, got %#v", meta["tokensScope"])
	}
	if meta["slotId"] != "slot-1" {
		t.Fatalf("expected slotId metadata, got %#v", meta)
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

func TestBuildFragmentGenerationAssets_UserReferenceKeepsSlotMetadata(t *testing.T) {
	req := domain.FragmentGenerationRequest{
		ReferenceSlots: []domain.FragmentReferenceSlot{
			{
				Key:        "character_main",
				Label:      "添加角色",
				Kind:       domain.FragmentGenerationAssetEntityCharacter,
				ImageURL:   "https://img.example/character.png",
				HelperText: "可上传主角或重要角色参考图",
			},
		},
	}
	assets := buildFragmentGenerationAssets(
		"frag-3",
		"task-3",
		domain.FragmentGenerationAssetSourceAIGeneration,
		"9:16",
		fragmentUserReferenceAssetsFromRequest(req),
		&domain.FragmentGenerationResult{},
		nil,
	)
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	got := assets[0]
	if got.Kind != domain.FragmentGenerationAssetKindUserReference {
		t.Fatalf("expected user_reference asset, got %s", got.Kind)
	}
	if got.EntityKind != domain.FragmentGenerationAssetEntityCharacter || got.EntityKey != "character_main" {
		t.Fatalf("unexpected entity metadata: %s/%s", got.EntityKind, got.EntityKey)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(got.MetadataJSON), &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["slotLabel"] != "添加角色" || meta["helperText"] == "" {
		t.Fatalf("missing slot metadata: %#v", meta)
	}
}

func TestBuildFragmentGenerationAssets_ReplacementUsesTargetSlotIndex(t *testing.T) {
	result := &domain.FragmentGenerationResult{
		ImageUrls: []string{
			"https://img.example/1.png",
			"https://img.example/2.png",
			"https://img.example/new-3.png",
			"https://img.example/4.png",
		},
		ImageSlots: []domain.FragmentGenerationImageSlot{
			{ID: "slot-1", Index: 1, Status: "completed", ImageURL: "https://img.example/1.png"},
			{ID: "slot-2", Index: 2, Status: "completed", ImageURL: "https://img.example/2.png"},
			{ID: "slot-3", Index: 3, Status: "completed", ImageURL: "https://img.example/new-3.png"},
			{ID: "slot-4", Index: 4, Status: "completed", ImageURL: "https://img.example/4.png"},
		},
		ScenePlan: []domain.FragmentScenePlan{
			{Index: 3, SceneDesc: "重绘第三页", GeneratedImageURL: "https://img.example/new-3.png"},
		},
	}
	assets := buildFragmentGenerationAssets("frag-4", "task-4", domain.FragmentGenerationAssetSourceAIGeneration, "9:16", nil, result, nil)
	if len(assets) != 1 {
		t.Fatalf("expected only the replacement scene asset, got %d", len(assets))
	}
	got := assets[0]
	if got.SlotID != "slot-3" {
		t.Fatalf("expected replacement asset linked to slot-3, got %q", got.SlotID)
	}
	if got.SlotIndex == nil || *got.SlotIndex != 3 {
		t.Fatalf("expected slot index 3, got %#v", got.SlotIndex)
	}
	if got.SceneIndex == nil || *got.SceneIndex != 3 {
		t.Fatalf("expected scene index 3, got %#v", got.SceneIndex)
	}
	if got.URL != "https://img.example/new-3.png" {
		t.Fatalf("expected replacement image url, got %q", got.URL)
	}
}

func TestBuildFragmentGenerationAssets_AppendUsesFinalSlotIndex(t *testing.T) {
	result := &domain.FragmentGenerationResult{
		ImageUrls: []string{
			"https://img.example/1.png",
			"https://img.example/2.png",
			"https://img.example/3.png",
			"https://img.example/4.png",
			"https://img.example/5.png",
		},
		ImageSlots: []domain.FragmentGenerationImageSlot{
			{ID: "slot-1", Index: 1, Status: "completed", ImageURL: "https://img.example/1.png"},
			{ID: "slot-2", Index: 2, Status: "completed", ImageURL: "https://img.example/2.png"},
			{ID: "slot-3", Index: 3, Status: "completed", ImageURL: "https://img.example/3.png"},
			{ID: "slot-4", Index: 4, Status: "completed", ImageURL: "https://img.example/4.png"},
			{ID: "slot-5", Index: 5, Status: "completed", ImageURL: "https://img.example/5.png"},
		},
		ScenePlan: []domain.FragmentScenePlan{
			{Index: 5, SceneDesc: "新增第五页", GeneratedImageURL: "https://img.example/5.png"},
		},
	}
	assets := buildFragmentGenerationAssets("frag-5", "task-5", domain.FragmentGenerationAssetSourceAIGeneration, "9:16", nil, result, nil)
	if len(assets) != 1 {
		t.Fatalf("expected only the appended scene asset, got %d", len(assets))
	}
	got := assets[0]
	if got.SlotID != "slot-5" {
		t.Fatalf("expected appended asset linked to slot-5, got %q", got.SlotID)
	}
	if got.SlotIndex == nil || *got.SlotIndex != 5 {
		t.Fatalf("expected slot index 5, got %#v", got.SlotIndex)
	}
	if got.SceneIndex == nil || *got.SceneIndex != 5 {
		t.Fatalf("expected scene index 5, got %#v", got.SceneIndex)
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
