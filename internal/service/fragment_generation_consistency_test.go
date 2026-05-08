package service

import (
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestFragmentReferenceAssetSelectionUsesRepeatedCoreEntities(t *testing.T) {
	bible := &domain.FragmentVisualBible{
		Characters: []domain.FragmentVisualCharacter{
			{Key: "char_a", RoleImportance: "core", ImmutableTraits: []string{"red coat"}},
			{Key: "char_b", RoleImportance: "supporting", ImmutableTraits: []string{"blue scarf"}},
		},
		Props: []domain.FragmentVisualProp{
			{Key: "prop_key", RoleImportance: "core", ImmutableTraits: []string{"old brass key"}},
		},
		Locations: []domain.FragmentVisualLocation{
			{Key: "loc_station", RoleImportance: "core", ImmutableTraits: []string{"green clock"}},
		},
	}
	scenes := []domain.FragmentScenePlan{
		{Index: 0, ReferenceKeys: []string{"char_a", "prop_key", "loc_station"}},
		{Index: 1, ReferenceKeys: []string{"char_a", "char_b", "prop_key", "loc_station"}},
		{Index: 2, ReferenceKeys: []string{"char_b", "loc_station"}},
	}
	policy := &domain.FragmentConsistencyPolicy{
		Level:                 "strong",
		EnableReferenceAssets: true,
		MaxCharacterAssets:    3,
		MaxPropAssets:         1,
		MaxLocationAssets:     1,
	}

	got := selectFragmentReferenceAssetCandidates(bible, scenes, policy)
	keys := map[string]string{}
	for _, c := range got {
		keys[c.Key] = c.Kind
	}

	if keys["char_a"] != "character" {
		t.Fatalf("expected repeated core character reference, got %#v", got)
	}
	if keys["prop_key"] != "prop" {
		t.Fatalf("expected repeated core prop reference, got %#v", got)
	}
	if keys["loc_station"] != "location" {
		t.Fatalf("expected repeated location reference in strong mode, got %#v", got)
	}
}

func TestFragmentScenePromptBindsEntities(t *testing.T) {
	bible := &domain.FragmentVisualBible{
		Characters: []domain.FragmentVisualCharacter{
			{Key: "char_lina", Name: "Lina", ImmutableTraits: []string{"short black bob", "red raincoat"}, NegativeTraits: []string{"must not wear blue scarf"}},
			{Key: "char_mo", Name: "Mo", ImmutableTraits: []string{"silver hair", "navy uniform"}},
		},
		Props: []domain.FragmentVisualProp{
			{Key: "prop_key", Name: "key", ImmutableTraits: []string{"old brass key with red thread"}, Ownership: "char_lina"},
		},
	}
	scene := domain.FragmentScenePlan{
		ImagePrompt:   "cinematic watercolor. Lina faces Mo inside an empty station.",
		ReferenceKeys: []string{"char_lina", "char_mo", "prop_key"},
		EntityBindings: []domain.FragmentEntityBinding{
			{Key: "char_lina", Kind: "character", Position: "left foreground", Action: "holding prop_key"},
			{Key: "prop_key", Kind: "prop", OwnerKey: "char_lina", Position: "in char_lina right hand"},
		},
	}

	prompt := buildFragmentSceneImagePrompt(bible, scene)
	for _, want := range []string{"char_lina", "red raincoat", "char_mo", "prop_key", "owner=char_lina", "Do not merge or swap"} {
		if !containsFragmentTestString(prompt, want) {
			t.Fatalf("expected prompt to contain %q, got:\n%s", want, prompt)
		}
	}
}

func TestFragmentStoryImageSeedUsesSeriesSeed(t *testing.T) {
	policy := &domain.FragmentConsistencyPolicy{Level: "standard", SeriesSeed: 12345}
	if got := fragmentStoryImageSeed(policy); got != 12345 {
		t.Fatalf("expected story image seed equal to SeriesSeed, got %d", got)
	}
	// Product path always enforces consistency; seed still follows SeriesSeed when set.
	still := &domain.FragmentConsistencyPolicy{Level: "off", SeriesSeed: 99999}
	if got := fragmentStoryImageSeed(still); got != 99999 {
		t.Fatalf("expected SeriesSeed when set regardless of level label, got %d", got)
	}
	if got := fragmentStoryImageSeed(&domain.FragmentConsistencyPolicy{Level: "standard"}); got != 1 {
		t.Fatalf("expected fallback 1 when SeriesSeed unset, got %d", got)
	}
	if got := fragmentStoryImageSeed(nil); got != 0 {
		t.Fatalf("expected 0 when policy nil, got %d", got)
	}
}

func TestParseFragmentImageInputIncludesProviderOptions(t *testing.T) {
	task := &domain.AITask{Input: `{"prompt":"draw","aspectRatio":"16:9","seed":42,"guidanceScale":7.5,"options":{"consistency_group_id":"task_1"},"referenceImages":["https://example.com/a.png"]}`}
	prompt, aspect, refs, seed, options, guidance, err := parseFragmentImageInput(task)
	if err != nil {
		t.Fatalf("parseFragmentImageInput returned error: %v", err)
	}
	if prompt != "draw" || aspect != "16:9" || seed != 42 || guidance != 7.5 {
		t.Fatalf("unexpected parsed basics: prompt=%q aspect=%q seed=%d guidance=%f", prompt, aspect, seed, guidance)
	}
	if len(refs) != 1 {
		t.Fatalf("expected one reference image, got %#v", refs)
	}
	if options["consistency_group_id"] != "task_1" {
		t.Fatalf("expected provider options to survive parse, got %#v", options)
	}
}

func TestParseFragmentVisualEvidence(t *testing.T) {
	raw := `{"evidence":[{"summary":"girl in red coat","entities":[{"key":"char_0","kind":"character","traits":["red coat"],"confidence":1.2}],"confidence":-1}]}`
	got, err := parseFragmentVisualEvidence(raw, []string{"https://example.com/ref.png"})
	if err != nil {
		t.Fatalf("parseFragmentVisualEvidence returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one evidence item, got %#v", got)
	}
	if got[0].ImageURL != "https://example.com/ref.png" {
		t.Fatalf("expected fallback image URL, got %q", got[0].ImageURL)
	}
	if got[0].Confidence != 0 || got[0].Entities[0].Confidence != 1 {
		t.Fatalf("expected confidence clamped, got %#v", got[0])
	}
}

func TestReferenceAssetSelectionPrefersVisualEvidenceTraits(t *testing.T) {
	bible := &domain.FragmentVisualBible{
		Characters: []domain.FragmentVisualCharacter{
			{Key: "char_0", RoleImportance: "core", ImmutableTraits: []string{"blue coat"}},
		},
	}
	evidence := []domain.FragmentVisualEvidence{{
		Entities: []domain.FragmentVisualEvidenceEntity{
			{Key: "char_0", Kind: "character", Traits: []string{"red coat", "round glasses"}},
		},
	}}
	scenes := []domain.FragmentScenePlan{
		{Index: 0, ReferenceKeys: []string{"char_0"}},
		{Index: 1, ReferenceKeys: []string{"char_0"}},
	}
	policy := &domain.FragmentConsistencyPolicy{Level: "standard", EnableReferenceAssets: true, MaxCharacterAssets: 1}
	got := selectFragmentReferenceAssetCandidatesWithEvidence(bible, evidence, scenes, policy)
	if len(got) != 1 {
		t.Fatalf("expected one candidate, got %#v", got)
	}
	for _, want := range []string{"red coat", "round glasses", "blue coat"} {
		found := false
		for _, trait := range got[0].Traits {
			if trait == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected trait %q in %#v", want, got[0].Traits)
		}
	}
}

func TestSelectFragmentAuditImageURLsByPolicy(t *testing.T) {
	urls := []string{"u0", "u1", "u2", "u3"}
	scenes := []domain.FragmentScenePlan{
		{Index: 0},
		{Index: 1, ReferenceKeys: []string{"char_0", "prop_0"}},
		{Index: 2},
		{Index: 3},
	}
	standard, skipped := selectFragmentAuditImageURLs(urls, scenes, &domain.FragmentConsistencyPolicy{Level: "standard"})
	if skipped != "" || len(standard) != 3 {
		t.Fatalf("expected standard to audit first/last/complex, got %#v skipped=%q", standard, skipped)
	}
	strong, _ := selectFragmentAuditImageURLs(urls, scenes, &domain.FragmentConsistencyPolicy{Level: "strong"})
	if len(strong) != len(urls) {
		t.Fatalf("expected strong to audit all images, got %#v", strong)
	}
	off, skipped := selectFragmentAuditImageURLs(urls, scenes, &domain.FragmentConsistencyPolicy{Level: "off"})
	if len(off) != 0 || skipped != "consistency_off" {
		t.Fatalf("expected off to skip audit, got %#v skipped=%q", off, skipped)
	}
}

func TestParseFragmentPanelPlanIncludesAILayoutFields(t *testing.T) {
	raw := `{"visualBible":{"styleBible":{"artStyle":"cinematic watercolor"},"characters":[],"props":[],"locations":[]},"panels":[{"index":0,"image_prompt":"cinematic watercolor scene with a lonely station platform, layered depth, warm light, quiet mood, detailed materials, strong leading lines, textured brushwork, atmospheric particles","caption":"她站在站台边。","reference_keys":[],"layout_intent":"wide_establishing","composition_plan":"主角在左下三分之一，站台线条引向右上远处。","shot_type":"wide_shot","visual_hierarchy":"主角轮廓优先，站台透视其次，天空氛围最后"}]}`
	plan, _, err := parseFragmentPanelPlanJSON(raw, 1)
	if err != nil {
		t.Fatalf("parseFragmentPanelPlanJSON returned error: %v", err)
	}
	if len(plan) != 1 {
		t.Fatalf("expected one panel, got %#v", plan)
	}
	if plan[0].LayoutIntent != "wide_establishing" || plan[0].ShotType != "wide_shot" {
		t.Fatalf("expected layout fields to parse, got %#v", plan[0])
	}
	if !containsFragmentTestString(plan[0].CompositionPlan, "左下三分之一") {
		t.Fatalf("expected composition plan to survive, got %#v", plan[0])
	}
}

func TestPanelFinalImagePromptIncludesAILayoutDirective(t *testing.T) {
	item := domain.FragmentPanelPlanItem{
		ImagePrompt:     "cinematic watercolor. A girl waits on an empty station platform.",
		LayoutIntent:    "diagonal_motion",
		CompositionPlan: "主角位于右下角，铁轨形成对角线引导到远方灯光。",
		ShotType:        "low_angle",
		VisualHierarchy: "主角动作第一，铁轨引导线第二，远方灯光第三。",
	}
	prompt := buildPanelFinalImagePrompt(item, "fantasy", domain.FragmentAspect16x9, 0, 3)
	for _, want := range []string{"layout_intent=diagonal_motion", "shot_type=low_angle", "composition_plan=", "Layout directive", "distinct regions"} {
		if !containsFragmentTestString(prompt, want) {
			t.Fatalf("expected prompt to contain %q, got:\n%s", want, prompt)
		}
	}
}

func TestPanelPlanPromptNoLongerMentionsFixedStripLayout(t *testing.T) {
	prompt := buildFragmentPanelPlanUserPrompt("女孩在雨夜车站等待", "fantasy", 5, fragmentPanelPlanLayoutAddon(domain.FragmentPanelGenerationRequest{}))
	for _, banned := range []string{"上二", "中通栏", "下二", "strip5_top2_middle_wide_bottom2"} {
		if containsFragmentTestString(prompt, banned) {
			t.Fatalf("expected prompt not to contain fixed strip layout %q", banned)
		}
	}
	for _, want := range []string{"layout_intent", "composition_plan", "shot_type", "visual_hierarchy", "自动布局决策", "多个区域"} {
		if !containsFragmentTestString(prompt, want) {
			t.Fatalf("expected prompt to contain %q", want)
		}
	}
	for _, want := range []string{"文本阶段的前置漫画规划", "不允许“先不规划，后续生图再决定漫画元素”"} {
		if !containsFragmentTestString(prompt, want) {
			t.Fatalf("expected prompt to contain strengthened planning rule %q", want)
		}
	}
}

func TestPanelPlanPromptUsesStructuredInputSections(t *testing.T) {
	prompt := buildFragmentPanelPlanUserPrompt("女孩在雨夜车站等待", "fantasy", 4, "")
	for _, want := range []string{"# PromptDSL", "prompt_dsl_v1", "# Role", "## Task", "## Inputs", "## Global Visual Config", "## Paneling / Camera / Action / Comic Elements Rules"} {
		if !containsFragmentTestString(prompt, want) {
			t.Fatalf("expected structured section %q, got:\n%s", want, prompt)
		}
	}
}

func containsFragmentTestString(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexFragmentTestString(s, sub) >= 0)
}

func indexFragmentTestString(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
