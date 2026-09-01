package service

import (
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestEnsureFragmentVisualBibleFallbackBuildsUsableAnchors(t *testing.T) {
	result := &fragmentElementExtractionResult{Elements: fragmentStoryElements{
		Weather:    "雨夜，门廊地面有积水反光",
		Location:   "老公寓狭窄门廊",
		Characters: []string{"刚下班回家的年轻女性，湿透的深蓝风衣，短黑发"},
		Objects:    []string{"泛黄旧信封，暗红封蜡已经开裂"},
	}}
	ensureFragmentVisualBibleFallback(domain.FragmentGenerationRequest{Style: "anime", Mood: "mysterious"}, result)

	if result.VisualBible == nil || result.VisualBible.StyleBible == nil {
		t.Fatal("expected a visual bible and style bible fallback")
	}
	if len(result.VisualBible.Characters) != 1 || len(result.VisualBible.Props) != 1 || len(result.VisualBible.Locations) != 1 {
		t.Fatalf("expected character, prop and location anchors, got %#v", result.VisualBible)
	}
	if !strings.Contains(result.VisualBible.StyleBible.Palette, "blue-gray") {
		t.Fatalf("expected rainy mysterious palette, got %q", result.VisualBible.StyleBible.Palette)
	}
}

func TestStrengthenFragmentScenePlansPreventsRepeatedObjectCloseups(t *testing.T) {
	bible := &domain.FragmentVisualBible{
		StyleBible: &domain.FragmentVisualStyleBible{ArtStyle: "cinematic anime illustration"},
		Characters: []domain.FragmentVisualCharacter{{Key: "char_main", Name: "归家人", ImmutableTraits: []string{"short black hair", "dark blue raincoat"}}},
		Props:      []domain.FragmentVisualProp{{Key: "prop_letter", Name: "旧信", ImmutableTraits: []string{"yellowed envelope", "cracked dark-red wax seal"}}},
		Locations:  []domain.FragmentVisualLocation{{Key: "loc_door", Name: "门廊", ImmutableTraits: []string{"old apartment corridor", "rain reflections"}}},
	}
	plans := []domain.FragmentScenePlan{
		{Index: 0, SceneDesc: "她在门口发现旧信。", ImagePrompt: "manga style, hand with key, close up"},
		{Index: 1, SceneDesc: "她看见封蜡已经开裂。", ImagePrompt: "manga style, hand with envelope, close up"},
		{Index: 2, SceneDesc: "她读到早已遗忘的名字。", ImagePrompt: "manga style, hand with phone, close up"},
		{Index: 3, SceneDesc: "她站在雨声里没有开门。", ImagePrompt: "manga style, hand and door, close up"},
	}
	got := strengthenFragmentScenePlans(plans, bible, fragmentStoryElements{Weather: "rainy night", Location: "old apartment corridor"}, "story", "9:16")

	for i, scene := range got {
		for _, want := range []string{"char_main", "prop_letter", "loc_door"} {
			if !containsFragmentTestString(strings.Join(scene.ReferenceKeys, ","), want) {
				t.Fatalf("scene %d missing reference %s: %#v", i, want, scene.ReferenceKeys)
			}
		}
		if !strings.Contains(scene.ImagePrompt, "no disembodied hand") || !strings.Contains(scene.ImagePrompt, "Narrative must-show") {
			t.Fatalf("scene %d did not receive quality controls: %s", i, scene.ImagePrompt)
		}
	}
	if !strings.Contains(got[0].ImagePrompt, "medium-wide establishing") || !strings.Contains(got[2].ImagePrompt, "face is the focal point") || !strings.Contains(got[3].ImagePrompt, "payoff shot") {
		t.Fatalf("expected a diverse establishing/reaction/payoff sequence: %#v", got)
	}
}

func TestFragmentScenePromptIncludesCompleteStyleBible(t *testing.T) {
	bible := &domain.FragmentVisualBible{StyleBible: &domain.FragmentVisualStyleBible{
		ArtStyle: "cinematic anime illustration", LineQuality: "precise ink", Palette: "blue gray and amber", LightingMood: "rainy night practical light",
	}}
	prompt := buildFragmentSceneImagePromptCore(bible, domain.FragmentScenePlan{ImagePrompt: "a woman returns home"})
	for _, want := range []string{"cinematic anime illustration", "precise ink", "blue gray and amber", "rainy night practical light"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing style control %q: %s", want, prompt)
		}
	}
	if strings.Contains(prompt, "Keep immutable traits") {
		t.Fatalf("prompt must not claim immutable traits when no entity is bound: %s", prompt)
	}
}
