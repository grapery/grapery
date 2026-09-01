package service

import (
	"context"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestFragmentComicPagePromptProducesOneCompletePagePerImageSlot(t *testing.T) {
	page := buildFallbackFragmentComicPagePlan(domain.FragmentScenePlan{
		SceneDesc:   "猫头鹰抵达水之塔并夺回机械核心。",
		ImagePrompt: "black and white fantasy manga adventure at a ruined water tower",
	}, 10, "zh-Hans")
	page.Panels[0].ReferenceKeys = []string{"owl_hero", "water_tower"}
	page.Panels[0].ComicTexts = []domain.FragmentComicText{{Type: "dialogue", Text: "水之塔就在前面", Speaker: "owl_hero", Position: "upper_left"}}

	scene := domain.FragmentScenePlan{
		SceneDesc:     "猫头鹰抵达水之塔并夺回机械核心。",
		ImagePrompt:   "black and white fantasy manga adventure at a ruined water tower",
		ReferenceKeys: []string{"owl_hero", "water_tower"},
		ComicPage:     &page,
	}
	bible := &domain.FragmentVisualBible{
		StyleBible: &domain.FragmentVisualStyleBible{ArtStyle: "dense black-and-white shonen manga linework"},
		Characters: []domain.FragmentVisualCharacter{{Key: "owl_hero", Name: "owl hero", ImmutableTraits: []string{"round body", "thick black glasses", "hoodie and backpack"}}},
		Locations:  []domain.FragmentVisualLocation{{Key: "water_tower", Name: "water tower", ImmutableTraits: []string{"waterfall spires", "wet ruined arches"}}},
	}

	prompt := buildFragmentSceneImagePrompt(bible, scene, "zh-Hans")
	for _, want := range []string{
		"ONE single image that is a complete comic page",
		"exactly 10 visibly separated internal comic panels",
		"black panel dividers and clean white gutters",
		"Internal panel 1/10",
		"thick black glasses",
		"OVERLAY reservation",
		"do NOT draw a balloon",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected comic page prompt to contain %q, got:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "bleed off all four edges") {
		t.Fatalf("comic page prompt must not inherit the legacy single-illustration no-frame directive: %s", prompt)
	}
	if strings.Contains(prompt, "exact_Simplified Chinese_text=\"水之塔就在前面\"") {
		t.Fatalf("overlay dialogue must not be sent to the image model as pixel lettering: %s", prompt)
	}
}

func TestFragmentBatchPromptKeepsEachOutputAsSeparateCompleteComicPage(t *testing.T) {
	first := buildFallbackFragmentComicPagePlan(domain.FragmentScenePlan{SceneDesc: "第一页", ImagePrompt: "page one"}, 8, "zh-Hans")
	second := buildFallbackFragmentComicPagePlan(domain.FragmentScenePlan{SceneDesc: "第二页", ImagePrompt: "page two"}, 6, "zh-Hans")
	scenes := []domain.FragmentScenePlan{
		{SceneDesc: "第一页", ImagePrompt: "page one", ComicPage: &first},
		{SceneDesc: "第二页", ImagePrompt: "page two", ComicPage: &second},
	}

	prompt := buildFragmentScenesBatchHuoshanPrompt(nil, scenes, 2, "zh-Hans")
	for _, want := range []string{
		"exactly 2 separate complete comic-page images",
		"Every output is independently a complete multi-panel comic page",
		"Complete Comic Page Image 1 / 2",
		"Complete Comic Page Image 2 / 2",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected batch comic page prompt to contain %q, got:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "Each output corresponds to one narrative scene") {
		t.Fatalf("new comic page batch must not use the legacy one-scene-one-illustration contract: %s", prompt)
	}
}

func TestNormalizeFragmentComicPagePlanFillsMissingPanelsAndFiltersReferences(t *testing.T) {
	bible := &domain.FragmentVisualBible{
		Characters: []domain.FragmentVisualCharacter{{Key: "hero"}},
		Props:      []domain.FragmentVisualProp{{Key: "core"}},
	}
	plan := domain.FragmentComicPagePlan{
		PanelCount: 8,
		Panels: []domain.FragmentComicPanelPlan{{
			Index:         9,
			SceneDesc:     "主角发现核心",
			ImagePrompt:   "hero discovers a mechanical core",
			ReferenceKeys: []string{"hero", "unknown", "core"},
			ComicTexts:    []domain.FragmentComicText{{Type: "dialogue", Text: "就是它", Speaker: "unknown"}},
		}},
	}

	got := normalizeFragmentComicPagePlan(plan, domain.FragmentScenePlan{SceneDesc: "发现核心", ImagePrompt: "mechanical discovery"}, 8, bible, "zh-Hans")
	if got.PanelCount != 8 || len(got.Panels) != 8 {
		t.Fatalf("expected exactly 8 normalized panels, got count=%d panels=%d", got.PanelCount, len(got.Panels))
	}
	if got.Panels[0].Index != 0 || got.Panels[7].Index != 7 {
		t.Fatalf("expected continuous panel indexes: %#v", got.Panels)
	}
	if strings.Join(got.Panels[0].ReferenceKeys, ",") != "hero,core" {
		t.Fatalf("expected unknown reference removed, got %#v", got.Panels[0].ReferenceKeys)
	}
	if got.Panels[0].ComicTexts[0].Speaker != "" {
		t.Fatalf("expected invalid speaker cleared, got %q", got.Panels[0].ComicTexts[0].Speaker)
	}
}

func TestFragmentComicPagePlannerPromptSeparatesPageCountFromPanelCount(t *testing.T) {
	pages := []domain.FragmentScenePlan{
		{SceneDesc: "第一页建立水之塔", ImagePrompt: "establish the water tower"},
		{SceneDesc: "第二页争夺核心", ImagePrompt: "fight for the core"},
	}
	prompt := buildFragmentComicPagePlanPrompt(domain.FragmentGenerationRequest{Style: "manga", Language: "zh-Hans", AspectRatio: "3:4"}, "完整故事", nil, pages, 0)
	for _, want := range []string{
		"pageCount",
		"minimumPanelCount",
		"maximumPanelCount",
		"第 1/2 张最终图片",
		"选择能够完整且紧凑表达本页的最少格数",
		"示例中的 4 不是默认值",
		"它本身必须是一张完整漫画页",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected page planner prompt to contain %q, got:\n%s", want, prompt)
		}
	}
}

func TestFragmentComicPageFallbackPanelCountTracksNarrativeDensity(t *testing.T) {
	compact := fragmentComicPageFallbackPanelCount(domain.FragmentScenePlan{SceneDesc: "主角停下脚步。"})
	dense := fragmentComicPageFallbackPanelCount(domain.FragmentScenePlan{SceneDesc: "主角走进路口，红灯开始倒数；他发现鞋边污渍，弯腰擦拭，随后抬头发现整座城市都没有影子。"})
	if compact != 2 {
		t.Fatalf("compact beat should use two panels, got %d", compact)
	}
	if dense <= compact {
		t.Fatalf("denser beat should receive more panels, compact=%d dense=%d", compact, dense)
	}
}

func TestValidateFragmentComicPagePlanAcceptsPlannerSelectedCount(t *testing.T) {
	panels := make([]domain.FragmentComicPanelPlan, 3)
	for i := range panels {
		panels[i].NewInformation = "new beat"
		panels[i].DramaticIntent = "advance story"
		panels[i].SilentIntent = "intentional silent acting"
	}
	plan := &domain.FragmentComicPagePlan{PanelCount: 3, Panels: panels}
	if err := validateFragmentComicPagePlanCount(plan); err != nil {
		t.Fatalf("valid adaptive panel count rejected: %v", err)
	}
	plan.PanelCount = 8
	if err := validateFragmentComicPagePlanCount(plan); err == nil {
		t.Fatal("mismatched planner count should be rejected")
	}
}

func TestNormalizeFragmentComicPagePlanValidatesEntityBindings(t *testing.T) {
	bible := &domain.FragmentVisualBible{
		Characters: []domain.FragmentVisualCharacter{{Key: "hero"}, {Key: "guide"}},
		Props:      []domain.FragmentVisualProp{{Key: "core"}},
	}
	plan := domain.FragmentComicPagePlan{
		Panels: []domain.FragmentComicPanelPlan{{
			SceneDesc:   "英雄拿起核心",
			ImagePrompt: "hero holds the core",
			EntityBindings: []domain.FragmentEntityBinding{
				{Key: "core", Kind: "character", OwnerKey: "hero", Position: " center "},
				{Key: "hero", Kind: "prop", OwnerKey: "hero"},
				{Key: "missing", Kind: "character", OwnerKey: "guide"},
			},
		}},
	}

	got := normalizeFragmentComicPagePlan(plan, domain.FragmentScenePlan{SceneDesc: "核心", ImagePrompt: "core"}, 6, bible, "zh-Hans")
	bindings := got.Panels[0].EntityBindings
	if len(bindings) != 2 {
		t.Fatalf("expected invalid entity binding removed, got %#v", bindings)
	}
	if bindings[0].Kind != "prop" || bindings[0].OwnerKey != "hero" || bindings[0].Position != "center" {
		t.Fatalf("expected prop kind and valid owner to be canonicalized, got %#v", bindings[0])
	}
	if bindings[1].Kind != "character" || bindings[1].OwnerKey != "" {
		t.Fatalf("expected self owner to be cleared for character binding, got %#v", bindings[1])
	}
}

func TestFragmentComicPagePlanningUsesExpandedStructuredBudget(t *testing.T) {
	if fragmentComicPagePlanMaxTokens < 4096 {
		t.Fatalf("comic page plans need enough JSON budget for 6-10 panels, got %d", fragmentComicPagePlanMaxTokens)
	}
}

func TestReviewFragmentComicPagePlanRejectsRepeatedBeatsAndFlatPerformance(t *testing.T) {
	plan := &domain.FragmentComicPagePlan{Panels: []domain.FragmentComicPanelPlan{
		{NewInformation: "主角看见门", SceneDesc: "主角站在门前", ShotType: "medium", CameraAngle: "eye", Composition: "center", EntityBindings: []domain.FragmentEntityBinding{{Key: "hero", Kind: "character"}}},
		{NewInformation: "主角看见门", SceneDesc: "主角站在门前", ShotType: "medium", CameraAngle: "eye", Composition: "center"},
	}}
	issues := reviewFragmentComicPagePlan(plan)
	if len(issues) < 3 {
		t.Fatalf("expected narrative, staging, and performance review issues, got %#v", issues)
	}
}

func TestFragmentComicPagePlannerPersistsEachFallbackPageIncrementally(t *testing.T) {
	pages := []domain.FragmentScenePlan{
		{Index: 0, SceneDesc: "第一页", ImagePrompt: "page one"},
		{Index: 1, SceneDesc: "第二页", ImagePrompt: "page two"},
	}
	var callbacks []int
	got, _ := (*FragmentGenerationService)(nil).planFragmentComicPages(
		context.Background(), "user", "task", domain.FragmentGenerationRequest{Language: "zh-Hans"}, "story", nil, pages,
		func(pageIndex int, snapshot []domain.FragmentScenePlan) {
			callbacks = append(callbacks, pageIndex)
			if snapshot[pageIndex].ComicPage == nil {
				t.Fatalf("callback %d must include its planned page", pageIndex)
			}
		},
	)
	if len(callbacks) != 2 || callbacks[0] != 0 || callbacks[1] != 1 {
		t.Fatalf("expected one progress callback per page, got %#v", callbacks)
	}
	for i, page := range got {
		if page.ComicPage == nil || page.ComicPage.PlanningStatus != "fallback" {
			t.Fatalf("page %d should keep an explicit fallback status, got %#v", i, page.ComicPage)
		}
	}
}
