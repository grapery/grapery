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


