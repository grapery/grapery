package mysql

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ========== Fragment Repository 实现 ==========

// FragmentByID retrieves a fragment by ID
func (r *Repository) FragmentByID(ctx context.Context, id string) (*domain.Fragment, error) {
	var fragment FragmentDB
	if err := r.db.WithContext(ctx).Preload("Creator").First(&fragment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	domainFragment := r.fragmentToDomain(fragment)
	return &domainFragment, nil
}

// ListFragments retrieves fragments with pagination
func (r *Repository) ListFragments(ctx context.Context, limit, offset int, visibility string) ([]*domain.Fragment, int64, error) {
	var fragments []*FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&FragmentDB{})

	if visibility != "" {
		query = query.Where("visibility = ?", visibility)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&fragments).Error

	if err != nil {
		return nil, 0, err
	}

	result := make([]*domain.Fragment, len(fragments))
	for i, f := range fragments {
		domainFragment := r.fragmentToDomain(*f)
		result[i] = &domainFragment
	}

	return result, total, nil
}

// CreateFragment creates a new fragment
func (r *Repository) CreateFragment(ctx context.Context, fragment *domain.Fragment) error {
	dbFragment := r.fragmentToModel(fragment)
	if err := r.db.WithContext(ctx).Create(dbFragment).Error; err != nil {
		return err
	}
	return nil
}

// UpdateFragment updates a fragment
func (r *Repository) UpdateFragment(ctx context.Context, fragment *domain.Fragment) error {
	dbFragment := r.fragmentToModel(fragment)
	return r.db.WithContext(ctx).Save(dbFragment).Error
}

// DeleteFragment deletes a fragment
func (r *Repository) DeleteFragment(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&FragmentDB{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ========== 转换函数 ==========

// fragmentToDomain 将数据库 FragmentDB 转换为 domain.Fragment
func (r *Repository) fragmentToDomain(f FragmentDB) domain.Fragment {
	// 解析 ImageUrls JSON
	var mediaURLs []string
	if f.ImageUrls != "" {
		_ = json.Unmarshal([]byte(f.ImageUrls), &mediaURLs)
	}

	fragment := domain.Fragment{
		BaseModel: common.BaseModel{
			ID:        f.ID,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		},
		UserID:     f.UserID, // authorId in JSON
		Content:    f.Content,
		MediaURLs:  mediaURLs,
		Visibility: f.Visibility,
		SourceType: f.SourceType,
		SourceID:   f.SourceID,
		Status:     string(common.StatusActive), // 默认状态
		EngagementStats: common.EngagementStats{
			Likes:    f.Likes,
			Comments: f.Comments,
			Shares:   f.Shares,
			Views:    f.Views,
		},
		ConvertedToStoryID: f.ConvertedToStoryID,
		IsConverted:        f.IsConverted,
		// 兼容字段
		CreatorID: f.UserID,
		ImageUrls: f.ImageUrls,
	}

	// 如果有 Creator 信息，填充 Author
	if f.Creator.ID != "" {
		author := r.userToDomain(f.Creator)
		fragment.Author = &author
	}

	return fragment
}

// fragmentToModel 将 domain.Fragment 转换为数据库 FragmentDB
func (r *Repository) fragmentToModel(f *domain.Fragment) *FragmentDB {
	// 将 MediaURLs 转换为 JSON 字符串
	imageUrlsJSON := "[]"
	if len(f.MediaURLs) > 0 {
		if data, err := json.Marshal(f.MediaURLs); err == nil {
			imageUrlsJSON = string(data)
		}
	}

	// 使用主字段
	creatorID := f.UserID
	if creatorID == "" {
		// 向后兼容：如果 UserID 为空，尝试使用 CreatorID
		creatorID = f.CreatorID
	}

	return &FragmentDB{
		ID:                 f.ID,
		UserID:             creatorID,
		Content:            f.Content,
		ImageUrls:          imageUrlsJSON,
		Visibility:         f.Visibility,
		SourceType:         f.SourceType,
		SourceID:           f.SourceID,
		ConvertedToStoryID: f.ConvertedToStoryID,
		IsConverted:        f.IsConverted,
		Likes:              f.Likes,
		Comments:           f.Comments,
		Shares:             f.Shares,
		Views:              f.Views,
		CreatedAt:          f.CreatedAt,
		UpdatedAt:          f.UpdatedAt,
	}
}
