package service

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// DashboardStoryboards returns storyboards from stories the user created OR follows.
func (s *Service) DashboardStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.DashboardStoryboards(ctx, userID, limit, offset)
}

// DashboardGroupStoryboards returns storyboards from groups the user joined.
func (s *Service) DashboardGroupStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.DashboardGroupStoryboards(ctx, userID, limit, offset)
}

// DashboardCharacterStoryboards returns storyboards that followed characters participate in.
func (s *Service) DashboardCharacterStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.DashboardCharacterStoryboards(ctx, userID, limit, offset)
}

// TrendingStoryboards returns published storyboards from trending stories.
func (s *Service) TrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.TrendingStoryboards(ctx, userID, limit, offset)
}
