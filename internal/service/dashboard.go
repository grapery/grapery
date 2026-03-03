package service

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// REMOVED: DashboardStoryboards - not in StoryCreationAppUI design
// REMOVED: DashboardCharacterStoryboards - not in StoryCreationAppUI design
// REMOVED: TrendingStoryboards (authenticated) - not in StoryCreationAppUI design

// PublicTrendingStoryboards returns published trending storyboards accessible to all users.
// If userID is empty (guest), returns globally trending storyboards.
// If userID is provided (authenticated), returns personalized trending storyboards.
func (s *Service) PublicTrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return s.repo.GetPublicTrendingStoryboards(ctx, userID, limit, offset)
}
