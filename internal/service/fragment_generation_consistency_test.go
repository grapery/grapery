package service

import (
	"strings"
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

// Feed 里图片贴边全屏展示，模型自绘的外框会被读成 UI 缺陷，所以出血约束必须出现在最终提示词里。
func TestFragmentImagePromptsForbidOuterFrame(t *testing.T) {
	comicScene := domain.FragmentScenePlan{
		ImagePrompt: "manga panel. A dog watches the sunset over dunes.",
		ComicTexts:  []domain.FragmentComicText{{Type: "narration", Text: "風還在吹着"}},
	}
	scenePrompt := buildFragmentSceneImagePrompt(nil, comicScene)
	panelPrompt := buildPanelFinalImagePrompt(domain.FragmentPanelPlanItem{
		ImagePrompt:     "comic panel. A dog watches the sunset over dunes.",
		LayoutIntent:    "comic_single_panel",
		CompositionPlan: "single continuous scene, subject lower right",
	}, "manga", "4:3", 0, 3)
	batchPrompt := buildFragmentScenesBatchHuoshanPrompt(nil, []domain.FragmentScenePlan{comicScene}, 1)

	for name, prompt := range map[string]string{"scene": scenePrompt, "panel": panelPrompt, "batch": batchPrompt} {
		if !containsFragmentTestString(prompt, "bleed off all four edges") {
			t.Fatalf("%s prompt missing full-bleed canvas directive, got:\n%s", name, prompt)
		}
		for _, banned := range []string{"bold ink panel borders", "bold black panel borders"} {
			if containsFragmentTestString(prompt, banned) {
				t.Fatalf("%s prompt still asks for %q, got:\n%s", name, banned, prompt)
			}
		}
	}

	// 组图路径把出血规则提到开头统一声明，逐场景不再重复。
	if strings.Count(batchPrompt, "bleed off all four edges") != 1 {
		t.Fatalf("batch prompt should declare the canvas rule exactly once, got:\n%s", batchPrompt)
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

func TestSelectFragmentAuditImageURLsAuditsEveryComicPage(t *testing.T) {
	urls := []string{"u0", "u1", "u2", "u3", "u4"}
	comicPage := buildFallbackFragmentComicPagePlan(domain.FragmentScenePlan{SceneDesc: "page", ImagePrompt: "page"}, 8, "zh-Hans")
	scenes := make([]domain.FragmentScenePlan, len(urls))
	for i := range scenes {
		scenes[i] = domain.FragmentScenePlan{Index: i, ComicPage: &comicPage}
	}

	got, skipped := selectFragmentAuditImageURLs(urls, scenes, &domain.FragmentConsistencyPolicy{Level: "standard"})
	if skipped != "" || len(got) != len(urls) {
		t.Fatalf("expected every complete comic page to be audited, got %#v skipped=%q", got, skipped)
	}
}

func TestFragmentComicPageRepairTargetsAndResultIndexes(t *testing.T) {
	issues := []domain.FragmentConsistencyIssue{
		{SceneIndex: 0, Severity: "high", ImageURL: "u2", Detail: "角色外观漂移"},
		{SceneIndex: 1, Severity: "medium", ImageURL: "u1", Detail: "轻微偏差"},
		{SceneIndex: 2, Severity: "HIGH", ImageURL: "u2", Detail: "分格数量错误"},
	}
	indexes := fragmentHighSeverityComicPageIndexes(issues, []string{"u0", "u1", "u2"}, 3)
	if len(indexes) != 1 || indexes[0] != 2 {
		t.Fatalf("expected exact image URL to identify one failed page, got %#v", indexes)
	}

	appended := remapFragmentConsistencyIssuesForResult([]domain.FragmentConsistencyIssue{{SceneIndex: 1}}, 5, 0)
	if appended[0].SceneIndex != 6 {
		t.Fatalf("expected appended issue to use full-gallery index, got %#v", appended)
	}
	replaced := remapFragmentConsistencyIssuesForResult([]domain.FragmentConsistencyIssue{{SceneIndex: 0}}, 0, 4)
	if replaced[0].SceneIndex != 3 {
		t.Fatalf("expected replacement issue to use target gallery index, got %#v", replaced)
	}
}

func TestFragmentComicPageRepairDirectiveOnlyIncludesTargetPageIssues(t *testing.T) {
	issues := []domain.FragmentConsistencyIssue{
		{SceneIndex: 0, Severity: "high", ImageURL: "u0", Detail: "target failure"},
		{SceneIndex: 1, Severity: "high", ImageURL: "u1", Detail: "other failure"},
	}
	directive := fragmentComicPageRepairDirective(0, "u0", issues)
	if !strings.Contains(directive, "target failure") || strings.Contains(directive, "other failure") {
		t.Fatalf("repair directive must contain only the target page failure: %s", directive)
	}
}

func TestNormalizeFragmentConsistencyIssuesRejectsBaselineAndInvalidTargets(t *testing.T) {
	issues := []domain.FragmentConsistencyIssue{
		{SceneIndex: 0, Severity: "HIGH", ImageURL: "new-1", Detail: "  valid issue  "},
		{SceneIndex: 0, Severity: "high", ImageURL: "baseline", Detail: "must be ignored"},
		{SceneIndex: 8, Severity: "high", Detail: "invalid index"},
		{SceneIndex: 1, Severity: "unexpected", Detail: "normalize severity"},
	}

	got := normalizeFragmentConsistencyIssuesForGeneratedImages(issues, []string{"new-0", "new-1"}, 2)
	if len(got) != 2 {
		t.Fatalf("expected only generated-image issues, got %#v", got)
	}
	if got[0].SceneIndex != 1 || got[0].Severity != "high" || got[0].Detail != "valid issue" {
		t.Fatalf("expected URL mapping and normalized high issue, got %#v", got[0])
	}
	if got[1].SceneIndex != 1 || got[1].ImageURL != "new-1" || got[1].Severity != "medium" {
		t.Fatalf("expected valid index fallback and normalized severity, got %#v", got[1])
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

func TestParseFragmentPanelPlanToleratesMalformedVisualBible(t *testing.T) {
	// visualBible 内字段类型漂移时不得拖垮整块 JSON；panels 仍可解析。
	raw := `{"visualBible":{"characters":[{"key":"a","immutableTraits":"should-have-been-array"}],"props":[],"locations":[]},"panels":[{"index":0,"image_prompt":"a detailed campsite scene at dusk","caption":"露营夜话","reference_keys":"user_ref_main"}]}`
	plan, vb, err := parseFragmentPanelPlanJSON(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vb != nil {
		t.Fatalf("expected visualBible discarded when invalid, got %#v", vb)
	}
	if len(plan) != 1 {
		t.Fatalf("expected 1 panel, got %d", len(plan))
	}
	if len(plan[0].ReferenceKeys) != 1 || plan[0].ReferenceKeys[0] != "user_ref_main" {
		t.Fatalf("reference_keys string coerced to slice: %#v", plan[0].ReferenceKeys)
	}
}

func TestParseFragmentPanelPlanCamelCasePanelsAndSparseComicText(t *testing.T) {
	raw := `[{"index":0,"imagePrompt":"wide shot of tents under stars","caption":"生火","comicTexts":{"type":"dialogue","text":"好冷呀","speaker":"主角"}}]`
	plan, _, err := parseFragmentPanelPlanJSON(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan) != 1 || plan[0].ImagePrompt == "" {
		t.Fatalf("camelCase imagePrompt not applied: %#v", plan)
	}
	if len(plan[0].ComicTexts) != 1 || strings.TrimSpace(plan[0].ComicTexts[0].Text) == "" {
		t.Fatalf("sparse comicTexts object expected: %#v", plan[0].ComicTexts)
	}
}

func TestParseFragmentPanelPlanNumericKeyedPanels(t *testing.T) {
	raw := `{"panels":{"0":{"image_prompt":"p0","caption":"c0"},"1":{"image_prompt":"p1","caption":"c1"}}}`
	plan, _, err := parseFragmentPanelPlanJSON(raw, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan) != 2 {
		t.Fatalf("expected 2 panels, got %d", len(plan))
	}
	if strings.TrimSpace(plan[0].ImagePrompt) != "p0" || strings.TrimSpace(plan[1].ImagePrompt) != "p1" {
		t.Fatalf("unexpected order/content: %#v", plan)
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
	for _, want := range []string{"visual_scene_v2", "# Role", "## Task", "## Inputs", "## Global Visual Config", "## Paneling / Camera / Action / Comic Elements Rules"} {
		if !containsFragmentTestString(prompt, want) {
			t.Fatalf("expected structured section %q, got:\n%s", want, prompt)
		}
	}
	for _, banned := range []string{"# PromptDSL", "prompt_dsl_v1", "```json"} {
		if containsFragmentTestString(prompt, banned) {
			t.Fatalf("model-facing prompt should omit DSL metadata %q", banned)
		}
	}
}

func TestFragmentScenePromptPreservesLocalizedNarrativeAndForbidsInventedText(t *testing.T) {
	scene := domain.FragmentScenePlan{SceneDesc: "少女は濡れた切符を見つめる。", ImagePrompt: "close-up of a girl holding a rain-soaked ticket"}
	prompt := buildFragmentSceneImagePrompt(nil, scene, "ja")
	for _, want := range []string{"少女は濡れた切符", "Japanese", "Visual execution (English)", "this scene is wordless"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected %q in localized final image prompt: %s", want, prompt)
		}
	}
	if strings.Contains(prompt, "Include at most") {
		t.Fatalf("wordless scene must not invite invented lettering: %s", prompt)
	}
}

func TestCompactPanelPlanPromptAvoidsQualityByWordCount(t *testing.T) {
	prompt := buildFragmentPanelPlanUserPrompt("女孩在雨夜车站等待", "fantasy", 5, "")
	for _, banned := range []string{"至少 52", "至少 70", "minWordsEach", "八层齐全"} {
		if containsFragmentTestString(prompt, banned) {
			t.Fatalf("compact production prompt must not contain %q", banned)
		}
	}
	if n := strings.Count(prompt, "【叙事规划契约】"); n != 1 {
		t.Fatalf("expected narrative contract once, got %d", n)
	}
}

func TestConstrainFragmentScenePlansToBible(t *testing.T) {
	bible := &domain.FragmentVisualBible{
		Characters: []domain.FragmentVisualCharacter{{Key: "char_main"}},
		Props:      []domain.FragmentVisualProp{{Key: "prop_key"}},
	}
	plans := []domain.FragmentScenePlan{{
		ReferenceKeys: []string{"char_main", "unknown", "prop_key"},
		ComicTexts: []domain.FragmentComicText{
			{Type: "dialogue", Text: "这是一句明显超过十二个汉字的对白内容", Speaker: "unknown"},
			{Type: "dialogue", Text: "第二句", Speaker: "char_main"},
			{Type: "dialogue", Text: "第三句应被丢弃", Speaker: "char_main"},
			{Type: "narration", Text: "旁白一"},
			{Type: "narration", Text: "旁白二应被丢弃"},
		},
	}}

	got := constrainFragmentScenePlansToBible(plans, bible)
	if len(got[0].ReferenceKeys) != 2 {
		t.Fatalf("expected unknown reference key removed: %#v", got[0].ReferenceKeys)
	}
	if len(got[0].ComicTexts) != 3 {
		t.Fatalf("expected comic type caps, got %#v", got[0].ComicTexts)
	}
	if got[0].ComicTexts[0].Speaker != "" {
		t.Fatalf("expected inactive speaker cleared, got %q", got[0].ComicTexts[0].Speaker)
	}
	if len([]rune(got[0].ComicTexts[0].Text)) > 12 {
		t.Fatalf("expected comic text capped at 12 runes, got %q", got[0].ComicTexts[0].Text)
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
