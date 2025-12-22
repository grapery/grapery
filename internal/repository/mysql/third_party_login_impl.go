package mysql

import (
	"context"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ThirdPartyLoginModel 第三方登录数据库模型
type ThirdPartyLoginModel struct {
	ID               string `gorm:"primaryKey;column:id;type:varchar(36)"`
	UserID           string `gorm:"column:user_id;type:varchar(36);index"`
	Provider         string `gorm:"column:provider;type:varchar(32);index"`
	ProviderUserID   string `gorm:"column:provider_user_id;type:varchar(255);index"`
	ProviderEmail    string `gorm:"column:provider_email;type:varchar(255);index"`
	ProviderUserName string `gorm:"column:provider_user_name;type:varchar(255)"`
	ProviderUserInfo string `gorm:"column:provider_user_info;type:text"`
	AccessToken      string `gorm:"column:access_token;type:text"`
	RefreshToken     string `gorm:"column:refresh_token;type:text"`
	TokenExpireTime  *int64 `gorm:"column:token_expire_time"`
	Status           int    `gorm:"column:status;default:1"`
	CreatedAt        int64  `gorm:"column:created_at"`
	UpdatedAt        int64  `gorm:"column:updated_at"`
	DeletedAt        *int64 `gorm:"column:deleted_at;index"`
}

// TableName 指定表名
func (ThirdPartyLoginModel) TableName() string {
	return "third_party_logins"
}

// toModel 将 domain 对象转换为数据库模型
func thirdPartyLoginToModel(login *domain.ThirdPartyLogin) *ThirdPartyLoginModel {
	return &ThirdPartyLoginModel{
		ID:               login.ID,
		UserID:           login.UserID,
		Provider:         string(login.Provider),
		ProviderUserID:   login.ProviderUserID,
		ProviderEmail:    login.ProviderEmail,
		ProviderUserName: login.ProviderUserName,
		ProviderUserInfo: login.ProviderUserInfo,
		AccessToken:      login.AccessToken,
		RefreshToken:     login.RefreshToken,
		TokenExpireTime:  login.TokenExpireTime,
		Status:           int(login.Status),
		CreatedAt:        login.CreatedAt,
		UpdatedAt:        login.UpdatedAt,
		DeletedAt:        login.DeletedAt,
	}
}

// toDomain 将数据库模型转换为 domain 对象
func (m *ThirdPartyLoginModel) toDomain() *domain.ThirdPartyLogin {
	return &domain.ThirdPartyLogin{
		ID:               m.ID,
		UserID:           m.UserID,
		Provider:         domain.ThirdPartyProvider(m.Provider),
		ProviderUserID:   m.ProviderUserID,
		ProviderEmail:    m.ProviderEmail,
		ProviderUserName: m.ProviderUserName,
		ProviderUserInfo: m.ProviderUserInfo,
		AccessToken:      m.AccessToken,
		RefreshToken:     m.RefreshToken,
		TokenExpireTime:  m.TokenExpireTime,
		Status:           domain.ThirdPartyLoginStatus(m.Status),
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		DeletedAt:        m.DeletedAt,
	}
}

// CreateThirdPartyLogin 创建第三方登录记录
func (r *Repository) CreateThirdPartyLogin(ctx context.Context, login *domain.ThirdPartyLogin) error {
	now := time.Now().Unix()
	if login.CreatedAt == 0 {
		login.CreatedAt = now
	}
	if login.UpdatedAt == 0 {
		login.UpdatedAt = now
	}
	if login.Status == 0 {
		login.Status = domain.ThirdPartyLoginStatusNormal
	}

	model := thirdPartyLoginToModel(login)
	return r.db.WithContext(ctx).Create(model).Error
}

// GetThirdPartyLogin 根据 ID 获取第三方登录记录
func (r *Repository) GetThirdPartyLogin(ctx context.Context, id string) (*domain.ThirdPartyLogin, error) {
	var model ThirdPartyLoginModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return model.toDomain(), nil
}

// GetThirdPartyLoginByProviderUserID 根据 Provider 和 ProviderUserID 获取记录
// 这是主要的查找方式，用于确定用户是否已经绑定过该第三方账号
func (r *Repository) GetThirdPartyLoginByProviderUserID(ctx context.Context, provider domain.ThirdPartyProvider, providerUserID string) (*domain.ThirdPartyLogin, error) {
	var model ThirdPartyLoginModel
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ? AND deleted_at IS NULL", string(provider), providerUserID).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return model.toDomain(), nil
}

// GetThirdPartyLoginByEmail 根据 Provider 和 Email 获取记录
// 用于通过 email 关联不同设备上的登录
func (r *Repository) GetThirdPartyLoginByEmail(ctx context.Context, provider domain.ThirdPartyProvider, email string) (*domain.ThirdPartyLogin, error) {
	var model ThirdPartyLoginModel
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_email = ? AND deleted_at IS NULL", string(provider), email).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return model.toDomain(), nil
}

// GetThirdPartyLoginsByUserID 获取用户的所有第三方登录记录
func (r *Repository) GetThirdPartyLoginsByUserID(ctx context.Context, userID string) ([]*domain.ThirdPartyLogin, error) {
	var models []ThirdPartyLoginModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ThirdPartyLogin, len(models))
	for i, m := range models {
		result[i] = m.toDomain()
	}
	return result, nil
}

// UpdateThirdPartyLogin 更新第三方登录记录
func (r *Repository) UpdateThirdPartyLogin(ctx context.Context, login *domain.ThirdPartyLogin) error {
	login.UpdatedAt = time.Now().Unix()
	model := thirdPartyLoginToModel(login)
	return r.db.WithContext(ctx).
		Model(&ThirdPartyLoginModel{}).
		Where("id = ? AND deleted_at IS NULL", login.ID).
		Updates(model).Error
}

// DeleteThirdPartyLogin 软删除第三方登录记录
func (r *Repository) DeleteThirdPartyLogin(ctx context.Context, id string) error {
	now := time.Now().Unix()
	return r.db.WithContext(ctx).
		Model(&ThirdPartyLoginModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now).Error
}

// GetUserByThirdPartyEmail 通过任意第三方登录的 email 查找关联的用户
// 场景：用户在设备A上用 Google 登录，在设备B上用 Apple 登录，两者使用相同 email
// 返回关联的用户（如果存在）
func (r *Repository) GetUserByThirdPartyEmail(ctx context.Context, email string) (*domain.User, error) {
	// 先在第三方登录表中查找
	var thirdPartyLogin ThirdPartyLoginModel
	err := r.db.WithContext(ctx).
		Where("provider_email = ? AND deleted_at IS NULL AND status = ?", email, domain.ThirdPartyLoginStatusNormal).
		Order("created_at ASC"). // 取最早的绑定记录
		First(&thirdPartyLogin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 没有第三方登录记录，尝试直接在用户表中查找
			return r.UserByEmail(ctx, email)
		}
		return nil, err
	}

	// 找到了第三方登录记录，获取关联的用户
	return r.UserByID(ctx, thirdPartyLogin.UserID)
}
