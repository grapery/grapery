package service

import (
	"context"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"strings"
)

func characterAttributeFields(a GeneratedCharacterAttributes) map[string]any {
	return map[string]any{"description": a.Description, "personality": a.Personality, "background": a.Background, "appearance": a.Appearance, "short_term_goal": a.ShortTermGoal, "long_term_goal": a.LongTermGoal, "handling_style": a.HandlingStyle, "cognition_range": a.CognitionRange, "ability_features": a.AbilityFeatures, "dress_preference": a.DressPreference}
}
func characterDomainAttributes(ch *domain.Character) GeneratedCharacterAttributes {
	return GeneratedCharacterAttributes{Description: ch.Description, Personality: ch.Personality, Background: ch.Background, Appearance: ch.Appearance, ShortTermGoal: ch.ShortTermGoal, LongTermGoal: ch.LongTermGoal, HandlingStyle: ch.HandlingStyle, CognitionRange: ch.CognitionRange, AbilityFeatures: ch.AbilityFeatures, DressPreference: ch.DressPreference}
}
func applyCharacterAttributes(ch *domain.Character, a GeneratedCharacterAttributes) {
	ch.Description, ch.Personality, ch.Background, ch.Appearance = a.Description, a.Personality, a.Background, a.Appearance
	ch.ShortTermGoal, ch.LongTermGoal, ch.HandlingStyle, ch.CognitionRange, ch.AbilityFeatures, ch.DressPreference = a.ShortTermGoal, a.LongTermGoal, a.HandlingStyle, a.CognitionRange, a.AbilityFeatures, a.DressPreference
}
func characterUpdateFields(req UpdateCharacterRequest, userID string) map[string]any {
	fields := map[string]any{"last_edited_by": userID}
	for key, value := range map[string]*string{"name": req.Name, "description": req.Description, "avatar": req.Avatar, "personality": req.Personality, "background": req.Background, "appearance": req.Appearance, "short_term_goal": req.ShortTermGoal, "long_term_goal": req.LongTermGoal, "handling_style": req.HandlingStyle, "cognition_range": req.CognitionRange, "ability_features": req.AbilityFeatures, "dress_preference": req.DressPreference, "source_type": req.SourceType, "source_prompt": req.SourcePrompt, "source_image": req.SourceImage} {
		if value != nil {
			fields[key] = *value
		}
	}
	if req.SourceType != nil {
		fields["source_type"] = strings.ToLower(*req.SourceType)
	}
	if req.IsPublic != nil {
		fields["is_public"] = *req.IsPublic
	}
	return fields
}
func (s *Service) patchCharacterUserFields(ctx context.Context, ch *domain.Character, fields map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if repo, ok := s.repo.(domain.CharacterMutationRepository); ok {
		lease, _ := ctx.Value(characterTaskContextKey{}).(*domain.CharacterGenerationTask)
		return repo.PatchCharacterFields(ctx, lease, ch, fields, false)
	}
	return s.updateCharacterWithExecutionLease(ctx, ch)
}
func (s *Service) saveCharacterVisualFieldsGuarded(ctx context.Context, ch *domain.Character, fields map[string]any, expected map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if repo, ok := s.repo.(domain.CharacterMutationRepository); ok {
		lease, _ := ctx.Value(characterTaskContextKey{}).(*domain.CharacterGenerationTask)
		return repo.UpdateCharacterVisualAssetsIfUnchanged(ctx, lease, ch, fields, expected)
	}
	return s.saveCharacterVisualFields(ctx, ch, fields)
}
