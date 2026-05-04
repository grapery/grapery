package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

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

// ListPublicNonDraftFragments lists public non-draft fragments, newest first.
func (r *Repository) ListPublicNonDraftFragments(ctx context.Context, limit, offset int) ([]*domain.Fragment, int64, error) {
	var fragments []*FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&FragmentDB{}).
		Where("visibility = ? AND COALESCE(is_draft, 0) = ?", domain.FragmentVisibilityPublic, 0)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Creator").
		Order("created_at DESC").
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

// ListPublicFragmentsByTopic lists public non-draft fragments with exact topic match.
func (r *Repository) ListPublicFragmentsByTopic(ctx context.Context, topic string, limit, offset int) ([]*domain.Fragment, int64, error) {
	var fragments []*FragmentDB
	var total int64

	query := r.db.WithContext(ctx).Model(&FragmentDB{}).
		Where("topic = ? AND visibility = ? AND COALESCE(is_draft, 0) = ?", topic, domain.FragmentVisibilityPublic, 0)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Creator").
		Order("created_at DESC").
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

// ListTopPublicFragmentTopicLabels implements domain.Repository.
func (r *Repository) ListTopPublicFragmentTopicLabels(ctx context.Context, minCount, limit int) ([]string, error) {
	if minCount < 1 {
		minCount = 2
	}
	if limit <= 0 {
		limit = 8
	}
	var rows []struct {
		Topic string `gorm:"column:topic"`
	}
	err := r.db.WithContext(ctx).Raw(`
SELECT topic FROM fragments
WHERE visibility = ? AND COALESCE(is_draft, 0) = ? AND TRIM(topic) <> ''
GROUP BY topic
HAVING COUNT(*) >= ?
ORDER BY COUNT(*) DESC, MAX(created_at) DESC
LIMIT ?
`, domain.FragmentVisibilityPublic, 0, minCount, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Topic != "" {
			out = append(out, row.Topic)
		}
	}
	return out, nil
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

// FragmentDBToDomain converts FragmentDB to domain.Fragment (exported for use by repository package)
func FragmentDBToDomain(f *FragmentDB) *domain.Fragment {
	if f == nil {
		return nil
	}
	d := fragmentDBToDomainInternal(*f)
	return &d
}

// fragmentToDomain 将数据库 FragmentDB 转换为 domain.Fragment
func (r *Repository) fragmentToDomain(f FragmentDB) domain.Fragment {
	return fragmentDBToDomainInternal(f)
}

func fragmentDBToDomainInternal(f FragmentDB) domain.Fragment {
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
		Topic:      f.Topic,
		Caption:    f.Caption,
		Style:      cloneTrimmedStringPtr(f.Style),
		Status:     string(common.StatusActive), // 默认状态
		EngagementStats: common.EngagementStats{
			Likes:    f.Likes,
			Comments: f.Comments,
			Shares:   f.Shares,
			Views:    f.Views,
		},
		ConvertedToStoryID: f.ConvertedToStoryID,
		IsConverted:        f.IsConverted,
		IsDraft:            f.IsDraft,
		GenerationTaskID:   f.GenerationTaskID,
		GenerationMetadata: f.GenerationMetadata,
		// 兼容字段
		CreatorID: f.UserID,
		ImageUrls: f.ImageUrls,
	}

	// 如果有 Creator 信息，填充 Author
	if f.Creator.ID != "" {
		if author := ModelToUser(&f.Creator); author != nil {
			fragment.Author = author
		}
	}

	return fragment
}

// DomainToFragmentDB converts domain.Fragment to FragmentDB (exported for use by repository package)
func DomainToFragmentDB(f *domain.Fragment) *FragmentDB {
	if f == nil {
		return nil
	}
	return domainToFragmentDBInternal(f)
}

// fragmentToModel 将 domain.Fragment 转换为数据库 FragmentDB
func (r *Repository) fragmentToModel(f *domain.Fragment) *FragmentDB {
	return domainToFragmentDBInternal(f)
}

func domainToFragmentDBInternal(f *domain.Fragment) *FragmentDB {
	// 持久化优先使用 MediaURLs（内存中的切片是权威来源）。若先更新了 MediaURLs 却未同步 JSON 字符串，
	// 旧逻辑会写回陈旧的 image_urls 列，导致列表/详情里少图（例如多格生成已 append 第 4 张但字符串仍为 3 条）。
	imageUrlsJSON := "[]"
	if len(f.MediaURLs) > 0 {
		if data, err := json.Marshal(f.MediaURLs); err == nil {
			imageUrlsJSON = string(data)
		}
	} else if f.ImageUrls != "" {
		imageUrlsJSON = f.ImageUrls
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
		Topic:              f.Topic,
		Caption:            f.Caption,
		Style:              f.Style,
		ConvertedToStoryID: f.ConvertedToStoryID,
		IsConverted:        f.IsConverted,
		IsDraft:            f.IsDraft,
		GenerationTaskID:   f.GenerationTaskID,
		GenerationMetadata: f.GenerationMetadata,
		Likes:              f.Likes,
		Comments:           f.Comments,
		Shares:             f.Shares,
		Views:              f.Views,
		CreatedAt:          f.CreatedAt,
		UpdatedAt:          f.UpdatedAt,
	}
}

func cloneTrimmedStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	out := s
	return &out
}
