package service

import (
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestExpandScenesFallbackKeepsRequestedImageCount(t *testing.T) {
	scenes := buildFallbackFragmentExpandedScenes(4, "少年在瀑布边发现一块旧铭牌", "fantasy", "mysterious", "9:16")
	if len(scenes) != 4 {
		t.Fatalf("expected 4 fallback scenes, got %d", len(scenes))
	}
	for i, scene := range scenes {
		if scene.Index != i {
			t.Fatalf("scene %d index mismatch: %d", i, scene.Index)
		}
		if strings.TrimSpace(scene.ImagePrompt) == "" {
			t.Fatalf("scene %d has empty image prompt", i)
		}
	}
}

func TestEnsureFragmentExpandedSceneCountPadsShortModelOutput(t *testing.T) {
	scenes := []fragmentExpandedScene{
		{Index: 0, SceneDesc: "第一格", ImagePrompt: "first image prompt"},
	}
	got := ensureFragmentExpandedSceneCount(scenes, 4, "少年在瀑布边发现一块旧铭牌", "fantasy", "mysterious", "9:16")
	if len(got) != 4 {
		t.Fatalf("expected 4 scenes after padding, got %d", len(got))
	}
	if got[0].SceneDesc != "第一格" {
		t.Fatalf("expected first scene to be preserved, got %q", got[0].SceneDesc)
	}
	for i, scene := range got {
		if scene.Index != i {
			t.Fatalf("scene %d index mismatch: %d", i, scene.Index)
		}
		if strings.TrimSpace(scene.ImagePrompt) == "" {
			t.Fatalf("scene %d has empty image prompt", i)
		}
	}
}

func TestParseSceneExpansionTrimsOverlongModelOutput(t *testing.T) {
	raw := `{"scenes":[
		{"sceneDesc":"一","imagePrompt":"prompt one"},
		{"sceneDesc":"二","imagePrompt":"prompt two"},
		{"sceneDesc":"三","imagePrompt":"prompt three"}
	]}`
	scenes, err := parseSceneExpansion(raw, 2)
	if err != nil {
		t.Fatalf("parse scene expansion: %v", err)
	}
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}
	for i, scene := range scenes {
		if scene.Index != i {
			t.Fatalf("scene %d index mismatch: %d", i, scene.Index)
		}
	}
}

func TestParseSceneExpansionExtractsJSONFromWrappedText(t *testing.T) {
	raw := "好的，下面是 JSON：\n```json\n{\"scenes\":[{\"sceneDesc\":\"一\",\"imagePrompt\":\"prompt one\"}]}\n```\n完成。"
	scenes, err := parseSceneExpansion(raw, 1)
	if err != nil {
		t.Fatalf("parse wrapped scene expansion: %v", err)
	}
	if len(scenes) != 1 || scenes[0].SceneDesc != "一" {
		t.Fatalf("unexpected scenes: %#v", scenes)
	}
}

func TestParseSceneExpansionAcceptsTopLevelArray(t *testing.T) {
	raw := `[{"sceneDesc":"一","imagePrompt":"prompt one"},{"sceneDesc":"二","imagePrompt":"prompt two"}]`
	scenes, err := parseSceneExpansion(raw, 2)
	if err != nil {
		t.Fatalf("parse top-level array: %v", err)
	}
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}
}

func TestEnsureFragmentScenePlanCountRepairsIncompleteScenes(t *testing.T) {
	plans := []domain.FragmentScenePlan{
		{Index: 9, SceneDesc: "第一格", ImagePrompt: ""},
	}
	got := ensureFragmentScenePlanCount(plans, 3, "少年在瀑布边发现一块旧铭牌", "fantasy", "mysterious", "9:16")
	if len(got) != 3 {
		t.Fatalf("expected 3 plans, got %d", len(got))
	}
	for i, plan := range got {
		if plan.Index != i {
			t.Fatalf("plan %d index mismatch: %d", i, plan.Index)
		}
		if strings.TrimSpace(plan.SceneDesc) == "" || strings.TrimSpace(plan.ImagePrompt) == "" {
			t.Fatalf("plan %d was not repaired: %#v", i, plan)
		}
	}
}

func TestReplaceFragmentDraftImageURLPreservesSpecifiedSlot(t *testing.T) {
	base := []string{
		"https://img.example/1.png",
		"https://img.example/2.png",
		"https://img.example/3.png",
		"https://img.example/4.png",
	}
	got := replaceFragmentDraftImageURL(base, []string{" https://img.example/new-3.png "}, 3)
	if len(got) != len(base) {
		t.Fatalf("expected %d urls, got %d: %#v", len(base), len(got), got)
	}
	if got[2] != "https://img.example/new-3.png" {
		t.Fatalf("expected slot 3 replaced, got %#v", got)
	}
	if got[0] != base[0] || got[1] != base[1] || got[3] != base[3] {
		t.Fatalf("expected other slots preserved, got %#v", got)
	}
}

func TestBuildFragmentReplacementImageSlotsMapsTargetIndex(t *testing.T) {
	previous := []string{
		"https://img.example/1.png",
		"https://img.example/2.png",
		"https://img.example/3.png",
		"https://img.example/4.png",
	}
	final := append([]string(nil), previous...)
	final[2] = "https://img.example/new-3.png"
	slots := buildFragmentReplacementImageSlots(
		"task-replace",
		"fragment-1",
		previous,
		final,
		3,
		[]domain.FragmentScenePlan{{SceneDesc: "主角重新发现线索"}},
	)
	if len(slots) != 4 {
		t.Fatalf("expected 4 slots, got %d", len(slots))
	}
	for i, slot := range slots {
		if slot.Index != i+1 {
			t.Fatalf("slot %d index mismatch: %#v", i, slot)
		}
		if slot.Status != "completed" {
			t.Fatalf("slot %d should stay completed: %#v", i, slot)
		}
	}
	if slots[2].ImageURL != "https://img.example/new-3.png" {
		t.Fatalf("expected target slot image url, got %#v", slots[2])
	}
	if slots[2].Caption != "主角重新发现线索" {
		t.Fatalf("expected target slot caption, got %#v", slots[2])
	}
	if slots[0].Caption != "" || slots[1].Caption != "" || slots[3].Caption != "" {
		t.Fatalf("only target slot should receive replacement caption: %#v", slots)
	}
}

func TestAppendFragmentDraftImageURLsPreservesOrderAndDuplicates(t *testing.T) {
	base := []string{
		"https://img.example/1.png",
		"https://img.example/2.png",
	}
	next := []string{
		"https://img.example/2.png",
		"https://img.example/3.png",
	}
	got := appendFragmentDraftImageURLs(base, next)
	want := []string{
		"https://img.example/1.png",
		"https://img.example/2.png",
		"https://img.example/2.png",
		"https://img.example/3.png",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d urls, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("url %d mismatch: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestRemapFragmentAppendedScenePlanOffsetsToFinalPageNumbers(t *testing.T) {
	scenes := []domain.FragmentScenePlan{
		{Index: 0, SceneDesc: "新增第一页"},
		{Index: 1, SceneDesc: "新增第二页"},
	}
	got := remapFragmentAppendedScenePlan(scenes, 4)
	if got[0].Index != 5 || got[1].Index != 6 {
		t.Fatalf("expected appended scenes mapped to pages 5 and 6, got %#v", got)
	}
	if scenes[0].Index != 0 {
		t.Fatalf("expected original scenes not mutated, got %#v", scenes)
	}
}

func TestBuildFragmentAppendImageSlotsIncludesExistingAndNewPages(t *testing.T) {
	previous := []string{
		"https://img.example/1.png",
		"https://img.example/2.png",
		"https://img.example/3.png",
		"https://img.example/4.png",
	}
	final := appendFragmentDraftImageURLs(previous, []string{"https://img.example/5.png"})
	slots := buildFragmentAppendImageSlots(
		"task-append",
		"fragment-1",
		previous,
		final,
		[]domain.FragmentScenePlan{{Index: 5, SceneDesc: "敌方士兵开始撤退"}},
	)
	if len(slots) != 5 {
		t.Fatalf("expected 5 slots, got %d", len(slots))
	}
	if slots[4].Index != 5 || slots[4].ImageURL != "https://img.example/5.png" {
		t.Fatalf("expected fifth slot to be appended image, got %#v", slots[4])
	}
	if slots[4].Caption != "敌方士兵开始撤退" {
		t.Fatalf("expected appended slot caption, got %#v", slots[4])
	}
	for i := 0; i < 4; i++ {
		if slots[i].Caption != "" {
			t.Fatalf("old slot %d should not receive new caption: %#v", i, slots[i])
		}
	}
}

func TestBuildFragmentGenerationImageSlotsMarksFailedOneBasedSlot(t *testing.T) {
	slots := buildFragmentGenerationImageSlots(
		2,
		[]domain.FragmentScenePlan{{SceneDesc: "第一格"}, {SceneDesc: "第二格"}},
		nil,
		map[int]string{1: "image failed"},
	)
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}
	if slots[0].Status != "failed" || slots[0].ErrorMessage != "image failed" {
		t.Fatalf("expected first slot failed, got %#v", slots[0])
	}
	if slots[1].Status == "failed" {
		t.Fatalf("second slot should not inherit first failure: %#v", slots[1])
	}
}

func TestBuildFragmentGenerationProgressSlotsOffsetsAppendPartial(t *testing.T) {
	partial := &domain.FragmentGenerationResult{
		DraftFragmentID:    "fragment-1",
		ExpectedImageCount: 5,
		ImageSlots: []domain.FragmentGenerationImageSlot{
			{Index: 1, Status: "completed", ImageURL: "https://img.example/1.png"},
			{Index: 2, Status: "completed", ImageURL: "https://img.example/2.png"},
			{Index: 3, Status: "completed", ImageURL: "https://img.example/3.png"},
			{Index: 4, Status: "completed", ImageURL: "https://img.example/4.png"},
		},
	}
	scenes := []domain.FragmentScenePlan{{Index: 0, SceneDesc: "新增第五页"}}
	slotIndex := fragmentGenerationProgressSlotIndex(partial, len(scenes), 0)
	if slotIndex != 5 {
		t.Fatalf("expected generating slot index 5, got %d", slotIndex)
	}
	slots := buildFragmentGenerationProgressSlots("task-append", partial, scenes, []string{""}, map[int]string{5: "new page failed"})
	if len(slots) != 5 {
		t.Fatalf("expected 5 slots, got %d", len(slots))
	}
	if slots[4].Index != 5 || slots[4].Caption != "新增第五页" {
		t.Fatalf("expected appended slot mapped to page 5, got %#v", slots[4])
	}
	if slots[4].Status != "failed" || slots[4].ErrorMessage != "new page failed" {
		t.Fatalf("expected appended failure on slot 5, got %#v", slots[4])
	}
	if slots[0].Status != "completed" || slots[0].ImageURL == "" {
		t.Fatalf("expected old slot preserved, got %#v", slots[0])
	}
}

func TestFragmentScenePlanDisplayPageIndexUsesFinalSceneIndex(t *testing.T) {
	if got := fragmentScenePlanDisplayPageIndex(0, domain.FragmentScenePlan{Index: 5}); got != 5 {
		t.Fatalf("expected final scene index 5, got %d", got)
	}
	if got := fragmentScenePlanDisplayPageIndex(1, domain.FragmentScenePlan{}); got != 2 {
		t.Fatalf("expected fallback page index 2, got %d", got)
	}
}
