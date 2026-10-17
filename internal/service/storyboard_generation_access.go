package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

type storyboardCallerKey struct{}

// WithStoryboardGenerationCaller carries a server-authenticated actor through
// both HTTP and internal jobs. The value must never come from a JSON userId.
func WithStoryboardGenerationCaller(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, storyboardCallerKey{}, strings.TrimSpace(userID))
}

// AuthorizeStoryboardGeneration is shared by individual and batch endpoints.
// Fork permission permits creating a child; it never permits mutating its parent.
func (s *Service) AuthorizeStoryboardGeneration(ctx context.Context, storyboardID string) (*domain.Storyboard, error) {
	board, err := s.AuthorizeStoryboardGenerationRead(ctx, storyboardID)
	if err != nil {
		return nil, err
	}
	if board.WorkflowStatus == domain.WorkflowStatusPublished {
		return nil, fmt.Errorf("%w: published storyboard is immutable; create a new draft", domain.ErrForbidden)
	}
	return board, nil
}

// Generation traces contain private input and prompt snapshots even when the
// published work itself is public. Only its creator can read these traces.
func (s *Service) AuthorizeStoryboardGenerationRead(ctx context.Context, storyboardID string) (*domain.Storyboard, error) {
	caller, _ := ctx.Value(storyboardCallerKey{}).(string)
	if caller == "" {
		return nil, fmt.Errorf("%w: authenticated generation caller required", domain.ErrUnauthorized)
	}
	board, err := s.repo.StoryboardByID(ctx, strings.TrimSpace(storyboardID))
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, domain.ErrNotFound
	}
	if board.UserID != caller {
		return nil, fmt.Errorf("%w: only the storyboard creator may access generation", domain.ErrForbidden)
	}
	return board, nil
}

func (s *Service) authorizeStoryboardSceneGeneration(ctx context.Context, storyboardID, sceneID string) (*domain.Storyboard, *domain.StoryboardScene, error) {
	board, err := s.AuthorizeStoryboardGeneration(ctx, storyboardID)
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(sceneID) == "" {
		return nil, nil, fmt.Errorf("%w: sceneId is required", domain.ErrInvalidInput)
	}
	scenes, err := s.repo.StoryboardScenes(ctx, board.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, scene := range scenes {
		if scene != nil && scene.ID == sceneID {
			return board, scene, nil
		}
	}
	return nil, nil, fmt.Errorf("%w: scene does not belong to storyboard", domain.ErrInvalidInput)
}
