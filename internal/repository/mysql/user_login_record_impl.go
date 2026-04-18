package mysql

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// CreateUserLoginRecord 创建用户登录记录
func (r *Repository) CreateUserLoginRecord(ctx context.Context, record *domain.UserLoginRecord) error {
	dbRecord := r.userLoginRecordFromDomain(record)
	return r.db.WithContext(ctx).Create(dbRecord).Error
}

// GetUserLoginRecords 获取用户的登录记录列表
func (r *Repository) GetUserLoginRecords(ctx context.Context, userID string, limit, offset int) ([]*domain.UserLoginRecord, error) {
	var records []UserLoginRecord
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("login_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.UserLoginRecord, len(records))
	for i, record := range records {
		result[i] = r.userLoginRecordToDomain(&record)
	}
	return result, nil
}

// GetLatestUserLoginRecord 获取用户最新的登录记录
func (r *Repository) GetLatestUserLoginRecord(ctx context.Context, userID string) (*domain.UserLoginRecord, error) {
	var record UserLoginRecord
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("login_at DESC").
		First(&record).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.userLoginRecordToDomain(&record), nil
}

// userLoginRecordToDomain 转换数据库模型到领域模型
func (r *Repository) userLoginRecordToDomain(m *UserLoginRecord) *domain.UserLoginRecord {
	return &domain.UserLoginRecord{
		ID:        m.ID,
		UserID:    m.UserID,
		IPAddress: m.IPAddress,
		Location:  m.Location,
		Device:    m.Device,
		OS:        m.OS,
		Browser:   m.Browser,
		UserAgent: m.UserAgent,
		LoginAt:   m.LoginAt,
		CreatedAt: m.CreatedAt,
	}
}

// userLoginRecordFromDomain 转换领域模型到数据库模型
func (r *Repository) userLoginRecordFromDomain(d *domain.UserLoginRecord) *UserLoginRecord {
	return &UserLoginRecord{
		ID:        d.ID,
		UserID:    d.UserID,
		IPAddress: d.IPAddress,
		Location:  d.Location,
		Device:    d.Device,
		OS:        d.OS,
		Browser:   d.Browser,
		UserAgent: d.UserAgent,
		LoginAt:   d.LoginAt,
	}
}
