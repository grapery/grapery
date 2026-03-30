package mysql

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// REMOVED: DashboardStoryboards - not in StoryCreationAppUI design
// REMOVED: DashboardCharacterStoryboards - not in StoryCreationAppUI design
// REMOVED: TrendingStoryboards (authenticated) - not in StoryCreationAppUI design

// GetPublicTrendingStoryboards returns published trending storyboards accessible to all users.
// Returns ALL published storyboards ordered by popularity metrics (likes, storyboard_count, followers, updated_at).
// If userID is provided (authenticated), user contributions are prioritized in the ranking.
func (r *Repository) GetPublicTrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Build base query for ALL published storyboards (no time or count restrictions)
	base := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Joins("JOIN stories ON stories.id = storyboards.story_id").
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished)

	// If user is authenticated, add personalization with story_contributors
	if userID != "" {
		base = base.Joins(`LEFT JOIN story_contributors sc ON sc.story_id = stories.id AND sc.user_id = ?`, userID)
	}

	// Count total distinct storyboards
	var total int64
	if err := base.
		Select("storyboards.id").
		Distinct().
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("public trending storyboards count: %w", err)
	}

	// Get page of storyboard IDs
	type idRow struct {
		ID string `gorm:"column:id"`
	}
	var idRows []idRow

	// Order by popularity metrics - prioritized for authenticated users
	var orderClause string
	if userID != "" {
		// Authenticated: prioritize user contributions, then by story metrics
		orderClause = `
			MAX(CASE WHEN sc.user_id IS NOT NULL THEN 1 ELSE 0 END) DESC,
			MAX(stories.likes) DESC,
			MAX(stories.storyboard_count) DESC,
			MAX(stories.followers) DESC,
			MAX(storyboards.updated_at) DESC
		`
	} else {
		// Guest: global ranking by popularity (no time restrictions)
		orderClause = `
			MAX(stories.likes) DESC,
			MAX(stories.storyboard_count) DESC,
			MAX(stories.followers) DESC,
			MAX(storyboards.updated_at) DESC
		`
	}

	if err := base.
		Select("storyboards.id").
		Group("storyboards.id").
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Scan(&idRows).Error; err != nil {
		return nil, 0, fmt.Errorf("public trending storyboards ids: %w", err)
	}

	ids := make([]string, 0, len(idRows))
	for _, row := range idRows {
		ids = append(ids, row.ID)
	}

	if len(ids) == 0 {
		return []*domain.Storyboard{}, total, nil
	}

	// Load storyboards with relations (do not re-order by updated_at — preserve ranking from id query above)
	var rows []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Preload("Story.Author").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("public trending storyboards load: %w", err)
	}

	byID := make(map[string]Storyboard, len(rows))
	for _, sb := range rows {
		byID[sb.ID] = sb
	}
	result := make([]*domain.Storyboard, 0, len(idRows))
	for _, id := range ids {
		sb, ok := byID[id]
		if !ok {
			continue
		}
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, &domainSb)
	}
	return result, total, nil
}

// StoryboardFeedRecommended ranks published storyboards for the “for you” tab.
// Currently delegates to engagement-based trending (with contributor boost when userID is set).
// Text/vector similarity can replace or augment this ranking later without changing the HTTP contract.
func (r *Repository) StoryboardFeedRecommended(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	return r.GetPublicTrendingStoryboards(ctx, userID, limit, offset)
}
