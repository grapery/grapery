package service

import "testing"

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

func TestStoryboardGenerationStepMessageKey(t *testing.T) {
	if storyboardGenerationStepMessageKey("bible_plan") != "storyboard_generation_planning_bible" {
		t.Fatal("bible_plan mapping")
	}
	if storyboardGenerationStage("image", true, "draft") != "images" {
		t.Fatal("image stage")
	}
}
