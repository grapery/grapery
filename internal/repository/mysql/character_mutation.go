package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var characterAttributeColumns = map[string]bool{"description": true, "personality": true, "background": true, "appearance": true, "short_term_goal": true, "long_term_goal": true, "handling_style": true, "cognition_range": true, "ability_features": true, "dress_preference": true}

func (r *Repository) PatchCharacterFields(ctx context.Context, lease *domain.CharacterGenerationTask, ch *domain.Character, fields map[string]any, fillEmpty bool) error {
	patch := make(map[string]any, len(fields)+1)
	for key, value := range fields {
		if !characterAttributeColumns[key] && (fillEmpty || !map[string]bool{"name": true, "avatar": true, "is_public": true, "source_type": true, "source_prompt": true, "source_image": true, "last_edited_by": true}[key]) {
			return fmt.Errorf("unsupported character edit field %s", key)
		}
		patch[key] = value
	}
	write := func(scoped *Repository) error {
		var current Character
		if err := scoped.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", ch.ID).First(&current).Error; err != nil {
			return err
		}
		if err := validateCharacterMutationOwner(&current, ch, lease); err != nil {
			return err
		}
		if fillEmpty {
			existing := map[string]string{"description": current.Description, "personality": current.Personality, "background": current.Background, "appearance": current.Appearance, "short_term_goal": current.ShortTermGoal, "long_term_goal": current.LongTermGoal, "handling_style": current.HandlingStyle, "cognition_range": current.CognitionRange, "ability_features": current.AbilityFeatures, "dress_preference": current.DressPreference}
			for key, value := range patch {
				text, ok := value.(string)
				if !ok {
					return fmt.Errorf("invalid character attribute %s", key)
				}
				if strings.TrimSpace(existing[key]) != "" || strings.TrimSpace(text) == "" {
					delete(patch, key)
				}
			}
		}
		patch["updated_at"] = time.Now()
		if err := scoped.db.WithContext(ctx).Model(&Character{}).Where("id = ?", ch.ID).Updates(patch).Error; err != nil {
			return err
		}
		updated, err := scoped.CharacterByID(ctx, ch.ID)
		if err != nil {
			return err
		}
		*ch = *updated
		return nil
	}
	return r.writeCharacterMutation(ctx, lease, write)
}

func validateCharacterMutationOwner(current *Character, ch *domain.Character, lease *domain.CharacterGenerationTask) error {
	owner := ch.UserID
	if owner == "" && ch.Author != nil {
		owner = ch.Author.ID
	}
	if owner == "" || current.UserID != owner || current.StoryID != ch.StoryID {
		return domain.ErrForbidden
	}
	if lease != nil && (lease.UserID != owner || lease.StoryID != current.StoryID) {
		return domain.ErrForbidden
	}
	return nil
}

func (r *Repository) writeCharacterMutation(ctx context.Context, lease *domain.CharacterGenerationTask, write func(*Repository) error) error {
	if lease != nil {
		return r.withCharacterTaskLease(ctx, lease, write)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { scoped := *r; scoped.db = tx; return write(&scoped) })
}

func characterViewsSheet(raw string) string {
	var views domain.CharacterThreeViews
	_ = json.Unmarshal([]byte(raw), &views)
	return views.Sheet
}

func (r *Repository) UpdateCharacterVisualAssetsIfUnchanged(ctx context.Context, lease *domain.CharacterGenerationTask, ch *domain.Character, fields map[string]any, expected map[string]string) error {
	return r.writeCharacterMutation(ctx, lease, func(scoped *Repository) error {
		var current Character
		if err := scoped.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", ch.ID).First(&current).Error; err != nil {
			return err
		}
		if err := validateCharacterMutationOwner(&current, ch, lease); err != nil {
			return err
		}
		for key, value := range expected {
			actual := ""
			switch key {
			case "inputs":
				domainCurrent := scoped.characterToDomain(current)
				actual = domain.CharacterVisualInputSignature(&domainCurrent)
			case "portrait":
				actual = current.Portrait
			case "views_sheet":
				actual = characterViewsSheet(current.ViewsJSON)
			default:
				return fmt.Errorf("unsupported character visual guard %s", key)
			}
			if actual != value {
				return domain.ErrCharacterEditConflict
			}
		}
		// Already inside the lease/row transaction: avoid re-locking a task row.
		return scoped.UpdateCharacterVisualAssets(ctx, nil, ch, fields)
	})
}
