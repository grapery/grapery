package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

type storyboardRevisionCommitter interface {
	CommitStoryboardRevision(context.Context, *domain.Storyboard, *domain.StoryboardRevisionBaseline) error
}

func (s *Service) captureStoryboardRevision(ctx context.Context, board *domain.Storyboard, targets []string, expected string) (*domain.StoryboardRevisionBaseline, error) {
	rows, err := s.repo.StoryboardScenes(ctx, board.ID)
	if err != nil {
		return nil, fmt.Errorf("load revision baseline: %w", err)
	}
	scenes := make([]domain.StoryboardScene, 0, len(rows))
	available := map[string]bool{}
	for _, row := range rows {
		if row != nil {
			scenes = append(scenes, *row)
			available[row.ID] = true
		}
	}
	sort.SliceStable(scenes, func(i, j int) bool {
		if scenes[i].Sequence == scenes[j].Sequence {
			return scenes[i].ID < scenes[j].ID
		}
		return scenes[i].Sequence < scenes[j].Sequence
	})
	revision := domain.StoryboardContentRevision(board, scenes)
	if expected != "" && expected != revision {
		return nil, fmt.Errorf("%w: storyboard revision conflict; reload the draft", domain.ErrAlreadyExists)
	}
	seen := map[string]bool{}
	clean := []string{}
	for _, id := range targets {
		id = strings.TrimSpace(id)
		if !available[id] {
			return nil, fmt.Errorf("%w: target scene does not belong to draft", domain.ErrInvalidInput)
		}
		if !seen[id] {
			clean = append(clean, id)
			seen[id] = true
		}
	}
	return &domain.StoryboardRevisionBaseline{Revision: revision, Title: board.Title, Content: board.Content, RawInput: board.RawInput, Scenes: scenes, TargetSceneIDs: clean}, nil
}

// mergeStoryboardRevision keeps every non-target row byte-for-byte, including
// stable ID and accepted media. Model output cannot change another page by accident.
func mergeStoryboardRevision(generated []domain.StoryboardScene, baseline *domain.StoryboardRevisionBaseline) ([]domain.StoryboardScene, error) {
	if baseline == nil || len(baseline.TargetSceneIDs) == 0 {
		return generated, nil
	}
	if len(generated) != len(baseline.Scenes) {
		return nil, fmt.Errorf("%w: local edit must preserve scene count", domain.ErrInvalidInput)
	}
	targets := map[string]bool{}
	for _, id := range baseline.TargetSceneIDs {
		targets[id] = true
	}
	out := append([]domain.StoryboardScene(nil), baseline.Scenes...)
	for i, scene := range baseline.Scenes {
		if targets[scene.ID] {
			out[i] = generated[i]
			out[i].ID = ""
			out[i].Sequence = scene.Sequence
		}
	}
	return out, nil
}

func (s *Service) commitGeneratedStoryboard(ctx context.Context, board *domain.Storyboard) error {
	if board.RevisionBaseline == nil {
		return fmt.Errorf("missing storyboard revision baseline")
	}
	merged, err := mergeStoryboardRevision(board.StoryboardScenes, board.RevisionBaseline)
	if err != nil {
		return err
	}
	board.StoryboardScenes = merged
	committer, ok := s.repo.(storyboardRevisionCommitter)
	if !ok {
		return fmt.Errorf("atomic storyboard revision persistence is not configured")
	}
	if err := committer.CommitStoryboardRevision(ctx, board, board.RevisionBaseline); err != nil {
		return err
	}
	board.ContentRevision = domain.StoryboardContentRevision(board, board.StoryboardScenes)
	s.invalidateStoryboardDetailAndListCaches(ctx, board.ID, board.StoryID)
	return nil
}

func (s *Service) ListStoryboardRevisionHistory(ctx context.Context, userID, boardID string) ([]domain.StoryboardRevisionHistory, error) {
	board, err := s.repo.StoryboardByID(ctx, boardID)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, domain.ErrNotFound
	}
	if userID == "" || board.UserID != userID {
		return nil, domain.ErrForbidden
	}
	reader, ok := s.repo.(interface {
		ListStoryboardRevisionHistory(context.Context, string, string) ([]domain.StoryboardRevisionHistory, error)
	})
	if !ok {
		return nil, fmt.Errorf("storyboard revision history is not configured")
	}
	return reader.ListStoryboardRevisionHistory(ctx, boardID, userID)
}
