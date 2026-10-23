package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateStoryboardImageAttempt serializes enqueue with edits/publication and
// other image requests. expectedPrevious is nil for an explicit new generation,
// or a CAS token (including empty for no history) for failure recovery.
func (r *Repository) CreateStoryboardImageAttempt(ctx context.Context, gen *domain.StoryboardImageGeneration, owner, expectedRevision string, expectedPrevious *string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var board Storyboard
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&board, "id = ?", gen.StoryboardID).Error; err != nil {
			return err
		}
		if board.UserID != owner || owner == "" || board.WorkflowStatus == domain.WorkflowStatusPublished {
			return domain.ErrForbidden
		}
		var scene StoryboardScene
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND storyboard_id = ?", gen.SceneID, gen.StoryboardID).First(&scene).Error; err != nil {
			return err
		}
		if expectedRevision == "" || r.storyboardImageRevision(&board, scene) != expectedRevision {
			return fmt.Errorf("%w: scene changed before image enqueue", domain.ErrAlreadyExists)
		}
		latest, err := latestStoryboardImageAttempt(tx, gen.StoryboardID, gen.SceneID)
		if err != nil {
			return err
		}
		if expectedPrevious != nil {
			if strings.TrimSpace(scene.Image) != "" {
				return fmt.Errorf("%w: scene already has an accepted image", domain.ErrAlreadyExists)
			}
			id := ""
			if latest != nil {
				id = latest.ID
			}
			if id != *expectedPrevious {
				return fmt.Errorf("%w: image attempt changed; refresh status", domain.ErrAlreadyExists)
			}
		}
		if latest != nil {
			if latest.Status != domain.GenerationStatusFailed && latest.Status != domain.GenerationStatusCompleted {
				return fmt.Errorf("%w: image attempt active or unknown; refresh status", domain.ErrAlreadyExists)
			}
			if strings.Contains(latest.ErrorMessage, "[cancelled]") && latest.GeneratedImageURL == "" {
				return fmt.Errorf("%w: cancelled provider result is unresolved", domain.ErrAlreadyExists)
			}
		}
		attempt := time.Now().UnixNano()
		if latest != nil && latest.PromptDetails != nil && latest.PromptDetails.Execution != nil && attempt <= latest.PromptDetails.Execution.Attempt {
			attempt = latest.PromptDetails.Execution.Attempt + 1
		}
		if gen.PromptDetails == nil {
			gen.PromptDetails = &domain.ImagePromptDetails{}
		}
		gen.PromptDetails.Execution = &domain.StoryboardImageExecution{OwnerID: owner, Revision: r.storyboardImageRevision(&board, scene), Attempt: attempt}
		return tx.Create(StoryboardImageGenerationToModel(gen)).Error
	})
}

func latestStoryboardImageAttempt(tx *gorm.DB, boardID, sceneID string) (*domain.StoryboardImageGeneration, error) {
	var rows []StoryboardImageGeneration
	if err := tx.Where("storyboard_id = ? AND scene_id = ?", boardID, sceneID).Find(&rows).Error; err != nil {
		return nil, err
	}
	var latest *domain.StoryboardImageGeneration
	for i := range rows {
		candidate := ModelToStoryboardImageGeneration(&rows[i])
		if domain.StoryboardImageAttemptAfter(candidate, latest) {
			latest = candidate
		}
	}
	return latest, nil
}

func (r *Repository) storyboardImageRevision(board *Storyboard, scene StoryboardScene) string {
	return domain.StoryboardContentRevision(ModelToStoryboard(board), []domain.StoryboardScene{*r.storyboardSceneToDomain(scene)})
}

// FinalizeStoryboardImageGeneration checkpoints the provider result first, so a
// scene-write failure is recoverable without calling a paid provider again. The
// accepted status and scene image then commit together, fenced by the immutable
// execution snapshot and latest attempt. Rejected candidates remain inspectable.
func (r *Repository) FinalizeStoryboardImageGeneration(ctx context.Context, gen *domain.StoryboardImageGeneration) (bool, error) {
	if strings.TrimSpace(gen.GeneratedImageURL) == "" {
		return false, domain.ErrInvalidInput
	}
	checkpoint := r.db.WithContext(ctx).Model(&StoryboardImageGeneration{}).Where("id = ? AND storyboard_id = ? AND scene_id = ?", gen.ID, gen.StoryboardID, gen.SceneID).
		Updates(map[string]any{"generated_image_url": gen.GeneratedImageURL, "generated_prompt": gen.GeneratedPrompt, "input_tokens": gen.InputTokens, "output_tokens": gen.OutputTokens, "total_tokens": gen.TotalTokens})
	if checkpoint.Error != nil {
		return false, checkpoint.Error
	}
	if checkpoint.RowsAffected == 0 {
		var count int64
		if err := r.db.WithContext(ctx).Model(&StoryboardImageGeneration{}).
			Where("id = ? AND storyboard_id = ? AND scene_id = ?", gen.ID, gen.StoryboardID, gen.SceneID).
			Count(&count).Error; err != nil {
			return false, err
		}
		if count == 0 {
			return false, domain.ErrNotFound
		}
	}
	accepted := false
	var rejection string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var board Storyboard
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&board, "id = ?", gen.StoryboardID).Error; err != nil {
			return err
		}
		var persisted StoryboardImageGeneration
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&persisted, "id = ?", gen.ID).Error; err != nil {
			return err
		}
		frozen := ModelToStoryboardImageGeneration(&persisted)
		var execution *domain.StoryboardImageExecution
		if frozen.PromptDetails != nil {
			execution = frozen.PromptDetails.Execution
		}
		var scene StoryboardScene
		sceneErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND storyboard_id = ?", gen.SceneID, gen.StoryboardID).First(&scene).Error
		if sceneErr != nil && sceneErr != gorm.ErrRecordNotFound {
			return sceneErr
		}
		latest, err := latestStoryboardImageAttempt(tx, gen.StoryboardID, gen.SceneID)
		if err != nil {
			return err
		}
		switch {
		case execution == nil || execution.OwnerID == "" || execution.Revision == "":
			rejection = "legacy image result requires explicit review"
		case execution.Rejection != "":
			rejection = execution.Rejection
		case board.UserID != execution.OwnerID || board.WorkflowStatus == domain.WorkflowStatusPublished:
			rejection = "storyboard owner or publication changed"
		case sceneErr == gorm.ErrRecordNotFound:
			rejection = "scene no longer exists"
		case latest == nil || latest.ID != gen.ID:
			rejection = "image attempt superseded"
		case strings.Contains(persisted.ErrorMessage, "[cancelled]"):
			rejection = "cancelled image result requires explicit review"
		case scene.Image == gen.GeneratedImageURL && persisted.Status == domain.GenerationStatusCompleted:
			accepted = true
			return nil
		case r.storyboardImageRevision(&board, scene) != execution.Revision:
			rejection = "accepted scene changed"
		}
		if gen.PromptDetails == nil {
			gen.PromptDetails = frozen.PromptDetails
		}
		if gen.PromptDetails == nil {
			gen.PromptDetails = &domain.ImagePromptDetails{}
		}
		gen.PromptDetails.Execution = execution
		now := time.Now().Unix()
		gen.CompletedAt = &now
		if rejection != "" {
			gen.Status, gen.ErrorMessage = domain.GenerationStatusFailed, rejection
			if execution != nil {
				execution.Rejection = rejection
			}
		} else {
			gen.Status, gen.ErrorMessage = domain.GenerationStatusCompleted, ""
			if err := tx.Model(&StoryboardScene{}).Where("id = ? AND storyboard_id = ?", gen.SceneID, gen.StoryboardID).Update("image", gen.GeneratedImageURL).Error; err != nil {
				return err
			}
			accepted = true
		}
		return tx.Save(StoryboardImageGenerationToModel(gen)).Error
	})
	if err != nil {
		return false, err
	}
	if rejection != "" {
		return false, fmt.Errorf("%w: %s", domain.ErrAlreadyExists, rejection)
	}
	return accepted, nil
}
