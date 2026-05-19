package mysql

import (
	"context"
	"errors"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

func normStoryboardParentID(parentID string, inBoard map[string]struct{}) string {
	p := strings.TrimSpace(parentID)
	if p == "" || p == domain.StoryboardRootMarker {
		return ""
	}
	if _, ok := inBoard[p]; !ok {
		return ""
	}
	return p
}

// storyboardTerminationPostOrder traverses boards so leaves are deleted before parents.
func storyboardTerminationPostOrder(boards []*domain.Storyboard) []string {
	ids := make(map[string]struct{}, len(boards))
	for _, b := range boards {
		if b == nil {
			continue
		}
		ids[b.ID] = struct{}{}
	}
	kids := make(map[string][]string)
	for _, b := range boards {
		if b == nil || b.ID == "" {
			continue
		}
		p := normStoryboardParentID(b.ParentID, ids)
		kids[p] = append(kids[p], b.ID)
	}
	var order []string
	var dfs func(pkey string)
	dfs = func(pkey string) {
		for _, id := range kids[pkey] {
			dfs(id)
			order = append(order, id)
		}
	}
	dfs("")
	return order
}

func deleteCommentsForStoryPolymorphic(db *gorm.DB, ctx context.Context, storyID string) error {
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return domain.ErrInvalidInput
	}
	var ids []string
	if err := db.WithContext(ctx).Model(&Comment{}).
		Where("target_type = ? AND target_id = ?", "story", storyID).
		Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if err := db.WithContext(ctx).Where("comment_id IN ?", ids).Delete(&CommentLike{}).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Delete(&Comment{}).Error
}

// TerminateOwnedStoryAndDependents deletes a user's story subtree (storyboards, characters, panels, etc.).
func (r *Repository) TerminateOwnedStoryAndDependents(ctx context.Context, storyID string) error {
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return domain.ErrInvalidInput
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sub := &Repository{db: tx, log: r.log, recoCfg: r.recoCfg}

		boards, err := sub.StoryboardsByStory(ctx, storyID, 10000, 0)
		if err != nil {
			return err
		}
		order := storyboardTerminationPostOrder(boards)
		for _, sbID := range order {
			sbID := strings.TrimSpace(sbID)
			if sbID == "" {
				continue
			}
			if err := sub.SoftDeleteStoryboardRelatedData(ctx, sbID); err != nil {
				return err
			}
			if err := sub.DeleteStoryboard(ctx, sbID); err != nil {
				return err
			}
			if err := sub.DecrementStoryStoryboardCount(ctx, storyID); err != nil {
				return err
			}
		}

		chars, err := sub.CharactersByStory(ctx, storyID)
		if err != nil {
			return err
		}
		for _, ch := range chars {
			if ch == nil || ch.ID == "" {
				continue
			}
			_ = sub.DeleteCharacter(ctx, ch.ID)
		}

		contribs, err := sub.GetStoryContributors(ctx, storyID, 500, 0)
		if err != nil {
			return err
		}
		for _, c := range contribs {
			if c == nil || c.UserID == "" {
				continue
			}
			_ = sub.RemoveStoryContributor(ctx, storyID, c.UserID)
		}

		tags, err := sub.StoryTags(ctx, storyID)
		if err != nil {
			return err
		}
		for _, t := range tags {
			if t == nil || t.ID == "" {
				continue
			}
			_ = sub.RemoveStoryTag(ctx, storyID, t.ID)
		}

		panels, err := sub.PanelsByStory(ctx, storyID)
		if err != nil {
			return err
		}
		for _, p := range panels {
			if p == nil || p.ID == "" {
				continue
			}
			_ = sub.DeletePanel(ctx, p.ID)
		}

		scenes, err := sub.StoryScenes(ctx, storyID, 500, 0)
		if err != nil {
			return err
		}
		for _, sc := range scenes {
			if sc == nil || sc.ID == "" {
				continue
			}
			_ = sub.DeleteStoryScene(ctx, storyID, sc.ID)
		}

		if err := deleteCommentsForStoryPolymorphic(tx, ctx, storyID); err != nil {
			return err
		}

		if err := tx.WithContext(ctx).Unscoped().Where("story_id = ?", storyID).Delete(&StoryPublication{}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Unscoped().Where("story_id = ?", storyID).Delete(&StoryFollow{}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Unscoped().Where("story_id = ?", storyID).Delete(&StoryLike{}).Error; err != nil {
			return err
		}

		if err := sub.DeleteStory(ctx, storyID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil
			}
			return err
		}
		return nil
	})
}
