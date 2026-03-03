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

	// Load storyboards with relations
	var rows []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Where("id IN ?", ids).
		Order("updated_at DESC").
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("public trending storyboards load: %w", err)
	}

	result := make([]*domain.Storyboard, 0, len(rows))
	for _, sb := range rows {
		domainSb, err := r.storyboardToDomain(ctx, sb)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, &domainSb)
	}
	return result, total, nil
}
