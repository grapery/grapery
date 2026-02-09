package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// GroupShowcase 数据库模型
type GroupShowcase struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"`
	GroupID     string    `gorm:"type:varchar(36);not null;index:idx_group_showcase"`
	ContentID   string    `gorm:"type:varchar(36);not null"`
	ContentType string    `gorm:"type:varchar(20);not null;index:idx_group_showcase_type"` // fragment, story
	AddedBy     string    `gorm:"type:varchar(36);not null"`
	Status      string    `gorm:"type:varchar(20);default:'active';index:idx_group_showcase_status"` // active, removed
	SortOrder   int       `gorm:"default:0;index:idx_group_showcase_sort"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (GroupShowcase) TableName() string {
	return "group_showcases"
}

// AddGroupShowcase 添加展示内容到小组
func (r *Repository) AddGroupShowcase(ctx context.Context, showcase *domain.GroupShowcase) error {
	dbShowcase := GroupShowcase{
		ID:          uuid.New().String(),
		GroupID:     showcase.GroupID,
		ContentID:   showcase.ContentID,
		ContentType: string(showcase.ContentType),
		AddedBy:     showcase.AddedBy,
		Status:      string(showcase.Status),
		SortOrder:   showcase.SortOrder,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&dbShowcase).Error; err != nil {
		return err
	}

	showcase.ID = dbShowcase.ID
	showcase.CreatedAt = dbShowcase.CreatedAt.Unix()
	showcase.UpdatedAt = dbShowcase.UpdatedAt.Unix()
	return nil
}

// RemoveGroupShowcase 移除小组展示内容（软删除）
func (r *Repository) RemoveGroupShowcase(ctx context.Context, showcaseID string) error {
	return r.db.WithContext(ctx).Model(&GroupShowcase{}).
		Where("id = ?", showcaseID).
		Update("status", string(domain.GroupShowcaseStatusRemoved)).Error
}

// GetGroupShowcases 获取小组展示列表
func (r *Repository) GetGroupShowcases(ctx context.Context, groupID string, contentType domain.GroupShowcaseRelationType, limit, offset int) ([]*domain.GroupShowcase, int64, error) {
	var showcases []GroupShowcase
	var total int64

	query := r.db.WithContext(ctx).Model(&GroupShowcase{}).
		Where("group_id = ? AND status = ?", groupID, string(domain.GroupShowcaseStatusActive))

	if contentType != "" {
		query = query.Where("content_type = ?", string(contentType))
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表，按排序顺序降序
	if err := query.Order("sort_order DESC, created_at DESC").
		Limit(limit).Offset(offset).
		Find(&showcases).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*domain.GroupShowcase, len(showcases))
	for i, s := range showcases {
		result[i] = r.groupShowcaseToDomain(s)
	}

	return result, total, nil
}

// GetGroupShowcaseByID 获取小组展示详情
func (r *Repository) GetGroupShowcaseByID(ctx context.Context, showcaseID string) (*domain.GroupShowcase, error) {
	var showcase GroupShowcase
	if err := r.db.WithContext(ctx).First(&showcase, "id = ?", showcaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	result := r.groupShowcaseToDomain(showcase)
	return result, nil
}

// UpdateGroupShowcaseOrder 更新小组展示排序
func (r *Repository) UpdateGroupShowcaseOrder(ctx context.Context, showcaseID string, sortOrder int) error {
	return r.db.WithContext(ctx).Model(&GroupShowcase{}).
		Where("id = ?", showcaseID).
		Update("sort_order", sortOrder).Error
}

// groupShowcaseToDomain 转换数据库模型到领域模型
func (r *Repository) groupShowcaseToDomain(s GroupShowcase) *domain.GroupShowcase {
	return &domain.GroupShowcase{
		ID:          s.ID,
		GroupID:     s.GroupID,
		ContentID:   s.ContentID,
		ContentType: domain.GroupShowcaseRelationType(s.ContentType),
		AddedBy:     s.AddedBy,
		Status:      domain.GroupShowcaseStatus(s.Status),
		SortOrder:   s.SortOrder,
		CreatedAt:   s.CreatedAt.Unix(),
		UpdatedAt:   s.UpdatedAt.Unix(),
	}
}
