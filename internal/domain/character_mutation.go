package domain

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrCharacterEditConflict = errors.New("character changed while generation was in progress")
var ErrCharacterStageOutcomeUnknown = errors.New("character_attributes_outcome_unknown: automatic retry requires reconciliation")

// CharacterMutationRepository preserves fields outside the user's patch. Task
// backfills fill only fields that are still empty under the same row lock.
type CharacterMutationRepository interface {
	PatchCharacterFields(context.Context, *CharacterGenerationTask, *Character, map[string]any, bool) error
	UpdateCharacterVisualAssetsIfUnchanged(context.Context, *CharacterGenerationTask, *Character, map[string]any, map[string]string) error
}

// CharacterVisualInputSignature excludes generated images and task status. A
// provider response based on an older setting must not replace newer artwork.
func CharacterVisualInputSignature(ch *Character) string {
	raw, _ := json.Marshal([]string{ch.StoryID, ch.Name, ch.ReferenceImage, ch.Description, ch.Personality, ch.Background, ch.Appearance, ch.ShortTermGoal, ch.LongTermGoal, ch.HandlingStyle, ch.CognitionRange, ch.AbilityFeatures, ch.DressPreference})
	return string(raw)
}
