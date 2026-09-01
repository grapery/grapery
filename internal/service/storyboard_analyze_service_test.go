package service

import (
	"context"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestInferStoryboardInputIntent(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "ask_clarification"},
		{"今天天气怎么样", "chat_only"},
		{"今天天气很好，我请假休息了", "new_storyboard"},
		{"换个故事板重新开始", "new_storyboard"},
		{"把格数改成 6", "adjust_options"},
		{"改一下画风更暗一点", "adjust_options"},
		{"主角走进迷雾森林", "new_storyboard"},
		{"修改一下上一幕的结尾", "revise_current"},
	}
	for _, tc := range cases {
		got := inferStoryboardInputIntent(tc.in)
		if got != tc.want {
			t.Fatalf("inferStoryboardInputIntent(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeStoryboardSceneCount(t *testing.T) {
	if normalizeStoryboardSceneCount(1) != 2 {
		t.Fatal("min should clamp to 2")
	}
	if normalizeStoryboardSceneCount(9) != 8 {
		t.Fatal("max should clamp to 8")
	}
	if normalizeStoryboardSceneCount(4) != 4 {
		t.Fatal("4 should stay 4")
	}
}

func TestAnalyzeStoryboardDirectionKeepsForkAndAppliesVisualOptions(t *testing.T) {
	service := &Service{}
	response, err := service.AnalyzeStoryboardDirection(context.Background(), "user-1", domain.StoryboardAnalyzeRequest{
		UserInput:          "从这里继续，做成6格水墨，比例 4:3",
		StoryID:            "story-1",
		ParentStoryboardID: "parent-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.EditPlan.Operation != "adjust_options" {
		t.Fatalf("operation=%q", response.EditPlan.Operation)
	}
	intent := response.GenerationIntent
	if intent.ParentStoryboardID != "parent-1" || intent.SceneCount != 6 || intent.Style != "ink_wash" || intent.AspectRatio != "4:3" {
		t.Fatalf("fork context/options lost: %+v", intent)
	}
}

func TestStoryboardGenerationStepMessageKey(t *testing.T) {
	if storyboardGenerationStepMessageKey("bible_plan") != "storyboard_generation_planning_bible" {
		t.Fatal("bible_plan mapping")
	}
	if storyboardGenerationStage("image", true, "draft") != "images" {
		t.Fatal("image stage")
	}
}

func TestDetectStoryboardFrameworkWarningsRequiresExplicitRetconConfirmation(t *testing.T) {
	warnings := detectStoryboardFrameworkWarnings("让已经牺牲的主角复活，并继续下一幕")
	if len(warnings) != 1 {
		t.Fatalf("warnings=%v", warnings)
	}
	if got := detectStoryboardFrameworkWarnings("让主角在雨夜继续寻找出口"); len(got) != 0 {
		t.Fatalf("ordinary continuation should stay low-friction: %v", got)
	}
}

func TestApplyParentStoryboardFrameworkCarriesEnding(t *testing.T) {
	alignment := storyboardFrameworkAlignment(&domain.Story{
		Title:       "瀑布露营记",
		Genre:       "冒险",
		Description: "朋友们在山谷露营，并共同解决突发危机。",
	})
	applyParentStoryboardFramework(alignment, &domain.Storyboard{
		Title:               "雨夜营地",
		ContinuationSummary: "营地被暴雨冲毁，所有人决定沿河寻找避难处。",
	})
	if alignment.ParentEnding == "" || alignment.ParentStoryboardTitle != "雨夜营地" {
		t.Fatalf("parent framework missing: %+v", alignment)
	}
	if len(alignment.InheritedFacts) < 3 {
		t.Fatalf("expected story and parent facts: %+v", alignment.InheritedFacts)
	}
}
