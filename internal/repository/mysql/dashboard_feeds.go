package mysql

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// DashboardStoryboards returns published storyboards from stories the user created OR follows.
// Only published storyboards are returned regardless of story ownership.
func (r *Repository) DashboardStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	base := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Joins("JOIN stories ON stories.id = storyboards.story_id").
		Joins("LEFT JOIN story_follows ON story_follows.story_id = stories.id AND story_follows.user_id = ?", userID).
		Where("(stories.author_id = ? OR story_follows.user_id IS NOT NULL)", userID).
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished)

	// total (distinct storyboard ids)
	var total int64
	if err := base.
		Select("storyboards.id").
		Distinct().
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard storyboards count: %w", err)
	}

	// page ids (distinct storyboard ids)
	type idRow struct {
		ID string `gorm:"column:id"`
	}
	var idRows []idRow
	if err := base.
		Select("storyboards.id").
		Group("storyboards.id").
		Order("MAX(storyboards.updated_at) DESC").
		Limit(limit).
		Offset(offset).
		Scan(&idRows).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard storyboards ids: %w", err)
	}

	ids := make([]string, 0, len(idRows))
	for _, row := range idRows {
		ids = append(ids, row.ID)
	}

	if len(ids) == 0 {
		return []*domain.Storyboard{}, total, nil
	}

	// load storyboards
	var rows []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Where("id IN ?", ids).
		Order("updated_at DESC").
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard storyboards load: %w", err)
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

// DashboardGroupStoryboards returns published storyboards from stories that belong to groups the user joined.
func (r *Repository) DashboardGroupStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	base := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Joins("JOIN stories ON stories.id = storyboards.story_id").
		Joins("JOIN group_members gm ON gm.group_id = stories.group_id AND gm.user_id = ?", userID).
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished)

	var total int64
	if err := base.Select("storyboards.id").Distinct().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard group storyboards count: %w", err)
	}

	type idRow struct {
		ID string `gorm:"column:id"`
	}
	var idRows []idRow
	if err := base.
		Select("storyboards.id").
		Group("storyboards.id").
		Order("MAX(storyboards.created_at) DESC").
		Limit(limit).
		Offset(offset).
		Scan(&idRows).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard group storyboards ids: %w", err)
	}

	ids := make([]string, 0, len(idRows))
	for _, row := range idRows {
		ids = append(ids, row.ID)
	}

	if len(ids) == 0 {
		return []*domain.Storyboard{}, total, nil
	}

	var rows []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Where("id IN ?", ids).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard group storyboards load: %w", err)
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

// DashboardCharacterStoryboards returns published storyboards that followed characters participate in.
func (r *Repository) DashboardCharacterStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	base := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Joins("JOIN storyboard_character_links scl ON scl.storyboard_id = storyboards.id").
		Joins("JOIN character_follows cf ON cf.character_id = scl.character_id AND cf.user_id = ?", userID).
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished)

	var total int64
	if err := base.Select("storyboards.id").Distinct().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard character storyboards count: %w", err)
	}

	type idRow struct {
		ID string `gorm:"column:id"`
	}
	var idRows []idRow
	if err := base.
		Select("storyboards.id").
		Group("storyboards.id").
		Order("MAX(storyboards.created_at) DESC").
		Limit(limit).
		Offset(offset).
		Scan(&idRows).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard character storyboards ids: %w", err)
	}

	ids := make([]string, 0, len(idRows))
	for _, row := range idRows {
		ids = append(ids, row.ID)
	}

	if len(ids) == 0 {
		return []*domain.Storyboard{}, total, nil
	}

	var rows []Storyboard
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Story").
		Where("id IN ?", ids).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("dashboard character storyboards load: %w", err)
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

// TrendingStoryboards returns published storyboards from trending stories:
// - Stories the user contributed to (via story_contributors)
// - Stories with high likes (top stories by likes)
// - Stories with high storyboard count (top stories by storyboard_count)
// - Stories with high followers (top stories by followers)
func (r *Repository) TrendingStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Build base query with multiple conditions using JOINs instead of IN subqueries
	// This avoids MySQL's limitation with LIMIT in IN subqueries
	base := r.db.WithContext(ctx).
		Model(&Storyboard{}).
		Joins("JOIN stories ON stories.id = storyboards.story_id").
		Joins(`LEFT JOIN (
			SELECT id FROM stories 
			WHERE status = 'published'
			ORDER BY likes DESC 
			LIMIT 100
		) top_likes ON top_likes.id = stories.id`).
		Joins(`LEFT JOIN (
			SELECT id FROM stories 
			WHERE status = 'published'
			ORDER BY storyboard_count DESC 
			LIMIT 100
		) top_storyboards ON top_storyboards.id = stories.id`).
		Joins(`LEFT JOIN (
			SELECT id FROM stories 
			WHERE status = 'published'
			ORDER BY followers DESC 
			LIMIT 100
		) top_followers ON top_followers.id = stories.id`).
		Joins(`LEFT JOIN story_contributors sc ON sc.story_id = stories.id AND sc.user_id = ?`, userID).
		Where("storyboards.workflow_status = ?", domain.WorkflowStatusPublished).
		Where("(sc.user_id IS NOT NULL OR top_likes.id IS NOT NULL OR top_storyboards.id IS NOT NULL OR top_followers.id IS NOT NULL)")

	// Count total distinct storyboards
	var total int64
	if err := base.
		Select("storyboards.id").
		Distinct().
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("trending storyboards count: %w", err)
	}

	// Get page of storyboard IDs, ordered by story metrics
	type idRow struct {
		ID string `gorm:"column:id"`
	}
	var idRows []idRow
	// Order by contribution status (using JOIN), then by story metrics
	orderClause := `
		MAX(CASE WHEN sc.user_id IS NOT NULL THEN 1 ELSE 0 END) DESC,
		MAX(stories.likes) DESC,
		MAX(stories.storyboard_count) DESC,
		MAX(stories.followers) DESC,
		MAX(storyboards.updated_at) DESC
	`
	if err := base.
		Select("storyboards.id").
		Group("storyboards.id").
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Scan(&idRows).Error; err != nil {
		return nil, 0, fmt.Errorf("trending storyboards ids: %w", err)
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
		return nil, 0, fmt.Errorf("trending storyboards load: %w", err)
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
