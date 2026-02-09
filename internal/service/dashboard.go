package service

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// DashboardStoryboards returns storyboards from stories the user created OR follows.
func (s *Service) DashboardStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.DashboardStoryboards(ctx, userID, limit, offset)
}

// DashboardCharacterStoryboards returns storyboards that followed characters participate in.
func (s *Service) DashboardCharacterStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.DashboardCharacterStoryboards(ctx, userID, limit, offset)
}

// TrendingStoryboards returns published storyboards from trending stories.
func (s *Service) TrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.TrendingStoryboards(ctx, userID, limit, offset)
}

// PublicTrendingStoryboards returns published trending storyboards accessible to all users.
// If userID is empty (guest), returns globally trending storyboards.
// If userID is provided (authenticated), returns personalized trending storyboards.
func (s *Service) PublicTrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.GetPublicTrendingStoryboards(ctx, userID, limit, offset)
}
