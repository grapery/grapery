package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// CreateStyleConfig creates a new style configuration
func (r *Repository) CreateStyleConfig(ctx context.Context, styleConfig *domain.StyleConfig) error {
	dbStyleConfig := StyleConfigToModel(styleConfig)
	if dbStyleConfig.ID == "" {
		dbStyleConfig.ID = uuid.New().String()
	}

	if err := r.db.WithContext(ctx).Create(dbStyleConfig).Error; err != nil {
		return fmt.Errorf("failed to create style config: %w", err)
	}

	// Update the domain object with generated values
	styleConfig.ID = dbStyleConfig.ID
	styleConfig.CreatedAt = dbStyleConfig.CreatedAt
	styleConfig.UpdatedAt = dbStyleConfig.UpdatedAt

	return nil
}

// GetStyleConfigByID retrieves a style configuration by ID
func (r *Repository) GetStyleConfigByID(ctx context.Context, id string) (*domain.StyleConfig, error) {
	var dbStyleConfig StyleConfig
	if err := r.db.WithContext(ctx).First(&dbStyleConfig, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("style config not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get style config: %w", err)
	}

	return ModelToStyleConfig(&dbStyleConfig), nil
}

// ListStyleConfigs retrieves all style configurations with optional pagination
// If groupID is provided, returns group-specific styles first, then public styles
func (r *Repository) ListStyleConfigs(ctx context.Context, groupID string, limit, offset int) ([]*domain.StyleConfig, int64, error) {
	var dbStyleConfigs []StyleConfig
	var total int64

	// Build base query - include public styles (no group/user) and group-specific styles
	baseQuery := r.db.WithContext(ctx).Model(&StyleConfig{})
	if groupID != "" {
		// Include public styles (group_id is empty) OR group-specific styles
		baseQuery = baseQuery.Where("(group_id = '' OR group_id IS NULL) OR group_id = ?", groupID)
	} else {
		// Only public styles when no groupID specified
		baseQuery = baseQuery.Where("group_id = '' OR group_id IS NULL")
	}

	// Get total count
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count style configs: %w", err)
	}

	// Get paginated results with group styles first
	query := r.db.WithContext(ctx)
	if groupID != "" {
		query = query.Where("(group_id = '' OR group_id IS NULL) OR group_id = ?", groupID)
		// Order: group-specific styles first, then by created_at
		query = query.Order(gorm.Expr("CASE WHEN group_id = ? THEN 0 ELSE 1 END, created_at ASC", groupID))
	} else {
		query = query.Where("group_id = '' OR group_id IS NULL")
		query = query.Order("created_at ASC")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&dbStyleConfigs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list style configs: %w", err)
	}

	styleConfigs := make([]*domain.StyleConfig, len(dbStyleConfigs))
	for i, dbStyleConfig := range dbStyleConfigs {
		styleConfigs[i] = ModelToStyleConfig(&dbStyleConfig)
	}

	return styleConfigs, total, nil
}

// SearchStyleConfigs searches style configurations by style name or description
// If groupID is provided, group-specific styles are prioritized in the results
func (r *Repository) SearchStyleConfigs(ctx context.Context, keyword, groupID string, limit, offset int) ([]*domain.StyleConfig, int64, error) {
	var dbStyleConfigs []StyleConfig
	var total int64

	searchPattern := "%" + keyword + "%"

	// Build base query with search and group filtering
	baseQuery := r.db.WithContext(ctx).Model(&StyleConfig{}).
		Where("style LIKE ? OR description LIKE ?", searchPattern, searchPattern)

	if groupID != "" {
		// Include public styles OR group-specific styles
		baseQuery = baseQuery.Where("(group_id = '' OR group_id IS NULL) OR group_id = ?", groupID)
	} else {
		// Only public styles
		baseQuery = baseQuery.Where("group_id = '' OR group_id IS NULL")
	}

	// Get total count for filtered results
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count filtered style configs: %w", err)
	}

	// Get paginated filtered results with group styles first
	query := r.db.WithContext(ctx).
		Where("style LIKE ? OR description LIKE ?", searchPattern, searchPattern)

	if groupID != "" {
		query = query.Where("(group_id = '' OR group_id IS NULL) OR group_id = ?", groupID)
		// Order: group-specific styles first, then by created_at
		query = query.Order(gorm.Expr("CASE WHEN group_id = ? THEN 0 ELSE 1 END, created_at ASC", groupID))
	} else {
		query = query.Where("group_id = '' OR group_id IS NULL")
		query = query.Order("created_at ASC")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&dbStyleConfigs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search style configs: %w", err)
	}

	styleConfigs := make([]*domain.StyleConfig, len(dbStyleConfigs))
	for i, dbStyleConfig := range dbStyleConfigs {
		styleConfigs[i] = ModelToStyleConfig(&dbStyleConfig)
	}

	return styleConfigs, total, nil
}

// UpdateStyleConfig updates an existing style configuration
func (r *Repository) UpdateStyleConfig(ctx context.Context, styleConfig *domain.StyleConfig) error {
	dbStyleConfig := StyleConfigToModel(styleConfig)

	result := r.db.WithContext(ctx).Model(&StyleConfig{}).
		Where("id = ?", styleConfig.ID).
		Updates(map[string]interface{}{
			"style":            dbStyleConfig.Style,
			"description":      dbStyleConfig.Description,
			"sample_image_url": dbStyleConfig.SampleImageURL,
			"updated_at":       dbStyleConfig.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update style config: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("style config not found: %s", styleConfig.ID)
	}

	return nil
}

// DeleteStyleConfig deletes a style configuration by ID
func (r *Repository) DeleteStyleConfig(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&StyleConfig{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete style config: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("style config not found: %s", id)
	}

	return nil
}

// GetStyleConfigByStyle retrieves a style configuration by style name
func (r *Repository) GetStyleConfigByStyle(ctx context.Context, styleName string) (*domain.StyleConfig, error) {
	var dbStyleConfig StyleConfig
	if err := r.db.WithContext(ctx).First(&dbStyleConfig, "style = ?", styleName).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("style config not found for style: %s", styleName)
		}
		return nil, fmt.Errorf("failed to get style config by style: %w", err)
	}

	return ModelToStyleConfig(&dbStyleConfig), nil
}

// BatchCreateStyleConfigs creates multiple style configurations in batch
func (r *Repository) BatchCreateStyleConfigs(ctx context.Context, styleConfigs []*domain.StyleConfig) error {
	if len(styleConfigs) == 0 {
		return nil
	}

	dbStyleConfigs := make([]StyleConfig, len(styleConfigs))
	for i, styleConfig := range styleConfigs {
		dbStyleConfig := StyleConfigToModel(styleConfig)
		if dbStyleConfig.ID == "" {
			dbStyleConfig.ID = uuid.New().String()
		}
		dbStyleConfigs[i] = *dbStyleConfig
	}

	if err := r.db.WithContext(ctx).CreateInBatches(dbStyleConfigs, 100).Error; err != nil {
		return fmt.Errorf("failed to batch create style configs: %w", err)
	}

	// Update domain objects with generated IDs
	for i, dbStyleConfig := range dbStyleConfigs {
		styleConfigs[i].ID = dbStyleConfig.ID
		styleConfigs[i].CreatedAt = dbStyleConfig.CreatedAt
		styleConfigs[i].UpdatedAt = dbStyleConfig.UpdatedAt
	}

	return nil
}