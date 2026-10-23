package service

import (
	"context"
	"encoding/json"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"strings"
	"testing"
)

func validCharacterAttributeJSON() string {
	values := map[string]string{}
	for key := range characterAttributeLimits {
		values[key] = "未设定"
	}
	values["description"] = "雨夜寻找失踪家人的青年"
	raw, _ := json.Marshal(values)
	return string(raw)
}
func TestCharacterAttributesRejectEmptyMissingWrongTypeAndOverlong(t *testing.T) {
	good := validCharacterAttributeJSON()
	if _, err := validateCharacterAttributesOutput(good); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{}`, `null`, `{"description":"only"}`, strings.Replace(good, `"appearance":"未设定"`, `"appearance":null`, 1), strings.Replace(good, `"appearance":"未设定"`, `"appearance":""`, 1), strings.Replace(good, `"appearance":"未设定"`, `"appearance":"`+strings.Repeat("字", 1001)+`"`, 1), good + ` {}`} {
		if _, err := validateCharacterAttributesOutput(raw); err == nil {
			t.Fatalf("accepted invalid output %.80s", raw)
		}
	}
}

type characterPromptTestRepo struct {
	domain.Repository
	character *domain.Character
	task      *domain.CharacterGenerationTask
}

func (r *characterPromptTestRepo) StoryByID(context.Context, string) (*domain.Story, error) {
	return &domain.Story{UserID: "owner", Title: "无魔法城市", Description: "没有超自然能力", Genre: "悬疑"}, nil
}
func (r *characterPromptTestRepo) CharactersByStory(context.Context, string) ([]*domain.Character, error) {
	return []*domain.Character{{Name: "李明", LongTermGoal: "找到妹妹"}}, nil
}
func (r *characterPromptTestRepo) UpdateCharacter(_ context.Context, ch *domain.Character) error {
	r.character = ch
	return nil
}
func (r *characterPromptTestRepo) CharacterByID(context.Context, string) (*domain.Character, error) {
	return r.character, nil
}
func (r *characterPromptTestRepo) UpdateCharacterGenerationTask(_ context.Context, task *domain.CharacterGenerationTask) error {
	copy := *task
	r.task = &copy
	return nil
}
func TestCharacterPromptCarriesWorldAndConfirmedGoals(t *testing.T) {
	s := &Service{repo: &characterPromptTestRepo{}}
	prompt, err := s.characterAttributesUserPrompt(context.Background(), "owner", GenerateCharacterRequest{StoryID: "story", Name: "小雨", Prompt: "补空", KnownAttributes: &GeneratedCharacterAttributes{ShortTermGoal: "今日取回证据", CognitionRange: "不知道凶手是谁"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"没有超自然能力", "找到妹妹", "今日取回证据", "不知道凶手是谁"} {
		if !strings.Contains(prompt, value) {
			t.Errorf("missing %q", value)
		}
	}
	if _, err := s.characterAttributesUserPrompt(context.Background(), "other", GenerateCharacterRequest{StoryID: "story"}); err == nil {
		t.Fatal("other owner accepted")
	}
	known := GeneratedCharacterAttributes{Appearance: "用户指定：银色短发", ShortTermGoal: "查证", Description: "确认介绍"}
	mergeGeneratedCharacterAttributes(&known, &GeneratedCharacterAttributes{Appearance: "黑发", ShortTermGoal: "逃跑", Background: "补充背景"})
	if known.Appearance != "用户指定：银色短发" || known.ShortTermGoal != "查证" || known.Background != "补充背景" {
		t.Fatal("confirmed settings overwritten")
	}
}
func TestCharacterVisualReferenceInheritance(t *testing.T) {
	ch := &domain.Character{ReferenceImage: "reference", Portrait: "portrait", Views: &domain.CharacterThreeViews{Sheet: "sheet"}}
	if got := characterVisualReferences(ch, "")[0]; got != "sheet" {
		t.Fatal(got)
	}
	if got := characterVisualReferences(ch, "new-reference")[0]; got != "new-reference" {
		t.Fatal(got)
	}
	ch.Views = nil
	if got := characterVisualReferences(ch, "")[0]; got != "portrait" {
		t.Fatal(got)
	}
	ch.Portrait = ""
	if got := characterVisualReferences(ch, "")[0]; got != "reference" {
		t.Fatal(got)
	}
}
func TestCharacterMissingVisualAssetsBecomePartialAndKeepCompletedAttributes(t *testing.T) {
	repo := &characterPromptTestRepo{character: &domain.Character{BaseModel: common.BaseModel{ID: "character"}, StoryID: "story", Author: &domain.User{BaseModel: common.BaseModel{ID: "owner"}}}}
	s := &Service{repo: repo, logger: zap.NewNop()}
	task := &domain.CharacterGenerationTask{ID: "task", StoryID: "story", UserID: "owner", CharacterID: "character", RequestJSON: `{"name":"雨","description":"已有设定"}`, AttributesJSON: validCharacterAttributeJSON(), Stages: map[string]domain.CharacterGenerationStage{"attributes": {Status: "completed", Attempts: 1}, "character": {Status: "completed", Attempts: 1}}}
	if err := s.executeCharacterGenerationTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.CharacterGenerationStatusPartial || task.Stages["portrait"].Status != "failed" || task.Stages["threeViews"].Status != "blocked" {
		t.Fatalf("incorrect success: %+v", task)
	}
	if task.Stages["attributes"].Attempts != 1 {
		t.Fatal("successful attributes regenerated")
	}
	if err := s.executeCharacterGenerationTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if task.Stages["portrait"].Attempts != 2 || task.Stages["attributes"].Attempts != 1 {
		t.Fatal("retry did not target failed stage")
	}
}

func TestCharacterAttributesRepairIsBoundedAndPreservesOriginalContext(t *testing.T) {
	for _, repairValid := range []bool{true, false} {
		calls := 0
		attrs, err := generateValidatedCharacterAttributes(context.Background(), &GenerateTextRequest{OriginalPrompt: "confirmed-world-and-fields"}, func(_ context.Context, req *GenerateTextRequest) (*GenerateTextResult, error) {
			calls++
			if calls == 1 {
				return &GenerateTextResult{Text: `{}`}, nil
			}
			if !strings.Contains(req.OriginalPrompt, "confirmed-world-and-fields") {
				t.Error("repair dropped canonical context")
			}
			if repairValid {
				return &GenerateTextResult{Text: validCharacterAttributeJSON()}, nil
			}
			return &GenerateTextResult{Text: `{}`}, nil
		})
		if calls != 2 {
			t.Fatalf("unbounded or absent repair: %d", calls)
		}
		if repairValid && (err != nil || attrs == nil) {
			t.Fatal(err)
		}
		if !repairValid && err == nil {
			t.Fatal("accepted empty repaired object")
		}
	}
}
