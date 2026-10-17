package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CommitStoryboardRevision locks the parent, compares accepted content, and
// swaps text and scenes in one transaction. Failed inserts roll everything back.
// Non-target rows keep their IDs, media, and generation history.
func (r *Repository) CommitStoryboardRevision(ctx context.Context, proposed *domain.Storyboard, baseline *domain.StoryboardRevisionBaseline) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked Storyboard
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", proposed.ID).First(&locked).Error; err != nil {
			return err
		}
		if locked.UserID != proposed.UserID || locked.WorkflowStatus == domain.WorkflowStatusPublished {
			return domain.ErrForbidden
		}
		scoped := &Repository{db: tx, log: r.log, recoCfg: r.recoCfg, cache: r.cache}
		current := ModelToStoryboard(&locked)
		var lockedScenes []StoryboardScene
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("storyboard_id = ?", proposed.ID).Order("sequence ASC").Find(&lockedScenes).Error; err != nil {
			return err
		}
		rows := make([]*domain.StoryboardScene, 0, len(lockedScenes))
		for _, row := range lockedScenes {
			rows = append(rows, scoped.storyboardSceneToDomain(row))
		}

		scenes := make([]domain.StoryboardScene, 0, len(rows))
		old := map[string]bool{}
		for _, row := range rows {
			if row != nil {
				scenes = append(scenes, *row)
				old[row.ID] = true
			}
		}
		if baseline == nil || domain.StoryboardContentRevision(current, scenes) != baseline.Revision {
			return fmt.Errorf("%w: storyboard revision conflict", domain.ErrAlreadyExists)
		}
		keep := map[string]bool{}
		var insert []*domain.StoryboardScene
		for i := range proposed.StoryboardScenes {
			scene := &proposed.StoryboardScenes[i]
			if scene.ID != "" && old[scene.ID] {
				keep[scene.ID] = true
			} else {
				insert = append(insert, scene)
			}
		}
		var remove []string
		for id := range old {
			if !keep[id] {
				remove = append(remove, id)
			}
		}
		if len(remove) > 0 {
			if err := tx.Where("storyboard_id = ? AND id IN ?", proposed.ID, remove).Delete(&StoryboardScene{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&Storyboard{}).Where("id = ?", proposed.ID).Updates(map[string]any{"content": proposed.Content, "title": proposed.Title, "is_ai_generated": true, "continuation_summary": "", "workflow_status": domain.WorkflowStatusContentReady, "current_step": 2, "scene_count": proposed.SceneCount, "continuation_comic_style": proposed.ContinuationComicStyle, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return scoped.CreateStoryboardScenes(ctx, proposed.ID, insert)
	})
}

func (r *Repository) ListStoryboardRevisionHistory(ctx context.Context, boardID, userID string) ([]domain.StoryboardRevisionHistory, error) {
	var rows []StoryboardGenerationRun
	if err := r.db.WithContext(ctx).Where("storyboard_id = ? AND user_id = ?", boardID, userID).Order("created_at DESC, id DESC").Limit(20).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := []domain.StoryboardRevisionHistory{}
	for _, row := range rows {
		var snapshot struct {
			Baseline *domain.StoryboardRevisionBaseline `json:"acceptedBaseline"`
		}
		if err := json.Unmarshal([]byte(row.ContextSnapshotJSON), &snapshot); err != nil {
			return nil, fmt.Errorf("decode revision baseline: %w", err)
		}
		if snapshot.Baseline != nil {
			out = append(out, domain.StoryboardRevisionHistory{GenerationRunID: row.ID, Status: row.Status, CreatedAt: row.CreatedAt.Unix(), Baseline: *snapshot.Baseline})
		}
	}
	return out, nil
}
