package service

import (
	"context"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

type storyboardCharacterContractRepository struct {
	domain.Repository
	character *domain.Character
}

func (r *storyboardCharacterContractRepository) CharacterByID(context.Context, string) (*domain.Character, error) {
	return r.character, nil
}
func (r *storyboardCharacterContractRepository) CharactersByStory(context.Context, string) ([]*domain.Character, error) {
	return []*domain.Character{r.character}, nil
}
func (r *storyboardCharacterContractRepository) StoryScenes(context.Context, string, int, int) ([]*domain.StoryScene, error) {
	return nil, nil
}

func TestStoryboardSnapshotCarriesCompleteCharacterMotivation(t *testing.T) {
	ch := &domain.Character{BaseModel: common.BaseModel{ID: "character"}, StoryID: "story", Name: "Lin", Description: "detective", Personality: "cautious", Background: "lost her mentor", ShortTermGoal: "find the stolen key", LongTermGoal: "clear her sister", HandlingStyle: "questions before fighting", CognitionRange: "does not know the killer", AbilityFeatures: "ordinary human without magic", Appearance: "black bob haircut", DressPreference: "worn yellow raincoat"}
	svc := &Service{repo: &storyboardCharacterContractRepository{character: ch}, logger: zap.NewNop()}
	board := &domain.Storyboard{StoryID: "story", IsStandalone: true, CharacterRefs: []domain.StoryboardCharacterRef{{CharacterID: ch.ID}}}
	story := &domain.Story{BaseModel: common.BaseModel{ID: "story"}}
	snapshot := svc.buildStoryboardGenerationContextSnapshot(context.Background(), board, story, false)
	if len(snapshot.Characters) != 1 {
		t.Fatal("selected character missing")
	}
	prompt := buildStoryboardBiblePlanUserPrompt(story, board, snapshot, 3)
	for _, value := range []string{ch.Description, ch.Personality, ch.Background, ch.ShortTermGoal, ch.LongTermGoal, ch.HandlingStyle, ch.CognitionRange, ch.AbilityFeatures, ch.Appearance, ch.DressPreference} {
		if !strings.Contains(prompt, value) {
			t.Fatalf("character contract lost %q", value)
		}
	}
}
