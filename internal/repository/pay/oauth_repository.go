package pay

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// OAuthUser 用户表模型（与 mysql 包的 User 结构对应）
type OAuthUser struct {
	ID              string         `gorm:"primaryKey;size:36"`
	Username        string         `gorm:"uniqueIndex;size:50;not null"`
	Email           string         `gorm:"uniqueIndex;size:100;not null"`
	PasswordHash    string         `gorm:"size:255;not null"`
	DisplayName     string         `gorm:"size:100;not null"`
	Avatar          string         `gorm:"size:500"`
	Bio             string         `gorm:"type:text"`
	Status          string         `gorm:"size:20;default:'active';index"`
	EmailVerified   bool           `gorm:"default:false"`
	LastLoginAt     int64          `gorm:"type:bigint;default:0"`
	StoryboardCount int            `gorm:"default:0"`
	Followers       int            `gorm:"default:0"`
	Following       int            `gorm:"default:0"`
	CreatedAt       int64          `gorm:"type:bigint;autoCreateTime"`
	UpdatedAt       int64          `gorm:"type:bigint;autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (OAuthUser) TableName() string {
	return "users"
}

// OAuthUserSettings 用户设置表模型
type OAuthUserSettings struct {
	ID                 string         `gorm:"primaryKey;size:36"`
	UserID             string         `gorm:"size:36;not null;uniqueIndex"`
	Language           string         `gorm:"size:10;default:'en'"`
	Theme              string         `gorm:"size:10;default:'auto'"`
	EmailNotifications bool           `gorm:"default:true"`
	PushNotifications  bool           `gorm:"default:true"`
	ShowAdultContent   bool           `gorm:"default:false"`
	ProfileVisibility  string         `gorm:"size:20;default:'public'"`
	AllowComments      bool           `gorm:"default:true"`
	AllowMessages      bool           `gorm:"default:true"`
	ShowOnlineStatus   bool           `gorm:"default:true"`
	UpdatedAt          int64          `gorm:"type:bigint;autoUpdateTime"`
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

func (OAuthUserSettings) TableName() string {
	return "user_settings"
}

// OAuthMembership 会员信息表模型
type OAuthMembership struct {
	ID           string         `gorm:"primaryKey;size:36"`
	UserID       string         `gorm:"size:36;not null;uniqueIndex"`
	Tier         string         `gorm:"size:20;default:'free'"`
	Status       string         `gorm:"size:20;default:'active'"`
	StartDate    int64          `gorm:"type:bigint"`
	EndDate      *int64         `gorm:"type:bigint"`
	AutoRenew    bool           `gorm:"default:false"`
	TokenQuota   int            `gorm:"default:10000"`
	TokenUsed    int            `gorm:"default:0"`
	StorageQuota int64          `gorm:"default:104857600"`
	StorageUsed  int64          `gorm:"default:0"`
	CreatedAt    int64          `gorm:"type:bigint;autoCreateTime"`
	UpdatedAt    int64          `gorm:"type:bigint;autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (OAuthMembership) TableName() string {
	return "memberships"
}

// OAuthThirdPartyLogin 第三方登录表模型
type OAuthThirdPartyLogin struct {
	ID               string         `gorm:"primaryKey;size:36"`
	UserID           string         `gorm:"size:36;not null;index"`
	Provider         string         `gorm:"size:32;not null;index:idx_oauth_provider_user_id,unique"`
	ProviderUserID   string         `gorm:"size:255;not null;index:idx_oauth_provider_user_id,unique"`
	ProviderEmail    string         `gorm:"size:255;index"`
	ProviderUserName string         `gorm:"size:255"`
	ProviderUserInfo string         `gorm:"type:text"`
	AccessToken      string         `gorm:"type:text"`
	RefreshToken     string         `gorm:"type:text"`
	TokenExpireTime  *int64         `gorm:"type:bigint"`
	Status           int            `gorm:"default:1"`
	CreatedAt        int64          `gorm:"type:bigint;autoCreateTime"`
	UpdatedAt        int64          `gorm:"type:bigint;autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (OAuthThirdPartyLogin) TableName() string {
	return "third_party_logins"
}

// OAuthRepository 实现 pay.OAuthRepository 接口
type OAuthRepository struct {
	db *gorm.DB
}

// NewOAuthRepository 创建 OAuth Repository
func NewOAuthRepository() *OAuthRepository {
	db := DataBase()
	if db == nil {
		return nil
	}

	// 注意：OAuth 相关表的迁移现在统一由 migrations 包管理
	// OAuthUser, OAuthUserSettings, OAuthMembership, OAuthThirdPartyLogin
	// 这些表实际上映射到 mysql 包的表（users, user_settings, memberships, third_party_logins）
	// 迁移步骤在 mysql/migrations_register.go 中注册

	return &OAuthRepository{db: db}
}

// UserByID 根据 ID 获取用户
func (r *OAuthRepository) UserByID(ctx context.Context, id string) (*domain.User, error) {
	var model OAuthUser
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.userModelToDomain(&model), nil
}

// UserByEmail 根据 Email 获取用户
func (r *OAuthRepository) UserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model OAuthUser
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.userModelToDomain(&model), nil
}

// CreateUser 创建用户
func (r *OAuthRepository) CreateUser(ctx context.Context, user *domain.User) error {
	model := r.userDomainToModel(user)
	return r.db.WithContext(ctx).Create(model).Error
}

// UpdateUser 更新用户
func (r *OAuthRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	model := r.userDomainToModel(user)
	return r.db.WithContext(ctx).Model(&OAuthUser{}).Where("id = ?", user.ID).Updates(model).Error
}

// CreateUserSettings 创建用户设置
func (r *OAuthRepository) CreateUserSettings(ctx context.Context, settings *domain.UserSettings) error {
	model := &OAuthUserSettings{
		ID:                 settings.ID,
		UserID:             settings.UserID,
		Language:           settings.Language,
		Theme:              settings.Theme,
		EmailNotifications: settings.EmailNotifications,
		PushNotifications:  settings.PushNotifications,
		ShowAdultContent:   settings.ShowAdultContent,
		ProfileVisibility:  settings.ProfileVisibility,
		AllowComments:      settings.AllowComments,
		AllowMessages:      settings.AllowMessages,
		ShowOnlineStatus:   settings.ShowOnlineStatus,
		UpdatedAt:          settings.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// CreateMembership 创建会员信息
func (r *OAuthRepository) CreateMembership(ctx context.Context, membership *domain.Membership) error {
	model := &OAuthMembership{
		ID:           membership.ID,
		UserID:       membership.UserID,
		Tier:         membership.Tier,
		Status:       membership.Status,
		StartDate:    membership.StartDate,
		EndDate:      membership.EndDate,
		AutoRenew:    membership.AutoRenew,
		TokenQuota:   membership.TokenQuota,
		TokenUsed:    membership.TokenUsed,
		StorageQuota: membership.StorageQuota,
		StorageUsed:  membership.StorageUsed,
		CreatedAt:    membership.CreatedAt,
		UpdatedAt:    membership.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// CreateThirdPartyLogin 创建第三方登录记录
func (r *OAuthRepository) CreateThirdPartyLogin(ctx context.Context, login *domain.ThirdPartyLogin) error {
	now := time.Now().Unix()
	if login.ID == "" {
		login.ID = uuid.New().String()
	}
	if login.CreatedAt == 0 {
		login.CreatedAt = now
	}
	if login.UpdatedAt == 0 {
		login.UpdatedAt = now
	}
	if login.Status == 0 {
		login.Status = domain.ThirdPartyLoginStatusNormal
	}

	model := &OAuthThirdPartyLogin{
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
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// GetThirdPartyLoginByProviderUserID 根据 Provider 和 ProviderUserID 获取第三方登录记录
func (r *OAuthRepository) GetThirdPartyLoginByProviderUserID(ctx context.Context, provider domain.ThirdPartyProvider, providerUserID string) (*domain.ThirdPartyLogin, error) {
	var model OAuthThirdPartyLogin
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ?", string(provider), providerUserID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.thirdPartyLoginModelToDomain(&model), nil
}

// GetThirdPartyLoginByEmail 根据 Provider 和 Email 获取第三方登录记录
func (r *OAuthRepository) GetThirdPartyLoginByEmail(ctx context.Context, provider domain.ThirdPartyProvider, email string) (*domain.ThirdPartyLogin, error) {
	var model OAuthThirdPartyLogin
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_email = ?", string(provider), email).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.thirdPartyLoginModelToDomain(&model), nil
}

// GetThirdPartyLoginsByUserID 获取用户的所有第三方登录记录
func (r *OAuthRepository) GetThirdPartyLoginsByUserID(ctx context.Context, userID string) ([]*domain.ThirdPartyLogin, error) {
	var models []OAuthThirdPartyLogin
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ThirdPartyLogin, len(models))
	for i := range models {
		result[i] = r.thirdPartyLoginModelToDomain(&models[i])
	}
	return result, nil
}

// UpdateThirdPartyLogin 更新第三方登录记录
func (r *OAuthRepository) UpdateThirdPartyLogin(ctx context.Context, login *domain.ThirdPartyLogin) error {
	login.UpdatedAt = time.Now().Unix()
	model := &OAuthThirdPartyLogin{
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
		UpdatedAt:        login.UpdatedAt,
	}
	return r.db.WithContext(ctx).Model(&OAuthThirdPartyLogin{}).Where("id = ?", login.ID).Updates(model).Error
}

// GetUserByThirdPartyEmail 通过任意第三方登录的 email 查找关联的用户
func (r *OAuthRepository) GetUserByThirdPartyEmail(ctx context.Context, email string) (*domain.User, error) {
	// 先在第三方登录表中查找
	var thirdPartyLogin OAuthThirdPartyLogin
	err := r.db.WithContext(ctx).
		Where("provider_email = ? AND status = ?", email, domain.ThirdPartyLoginStatusNormal).
		Order("created_at ASC"). // 取最早的绑定记录
		First(&thirdPartyLogin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 没有第三方登录记录，尝试直接在用户表中查找
			return r.UserByEmail(ctx, email)
		}
		return nil, err
	}

	// 找到了第三方登录记录，获取关联的用户
	return r.UserByID(ctx, thirdPartyLogin.UserID)
}

// DeleteThirdPartyLogin 删除用户的第三方登录绑定
func (r *OAuthRepository) DeleteThirdPartyLogin(ctx context.Context, userID string, provider domain.ThirdPartyProvider) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, string(provider)).
		Delete(&OAuthThirdPartyLogin{}).Error
}

// Helper functions

func (r *OAuthRepository) userModelToDomain(model *OAuthUser) *domain.User {
	var lastLoginAt *int64
	if model.LastLoginAt > 0 {
		lastLoginAt = &model.LastLoginAt
	}
	return &domain.User{
		ID:              model.ID,
		Username:        model.Username,
		Email:           model.Email,
		PasswordHash:    model.PasswordHash,
		DisplayName:     model.DisplayName,
		Avatar:          model.Avatar,
		Bio:             model.Bio,
		Status:          model.Status,
		EmailVerified:   model.EmailVerified,
		LastLoginAt:     lastLoginAt,
		StoryboardCount: model.StoryboardCount,
		Followers:       model.Followers,
		Following:       model.Following,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func (r *OAuthRepository) userDomainToModel(user *domain.User) *OAuthUser {
	var lastLoginAt int64
	if user.LastLoginAt != nil {
		lastLoginAt = *user.LastLoginAt
	}
	return &OAuthUser{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		PasswordHash:    user.PasswordHash,
		DisplayName:     user.DisplayName,
		Avatar:          user.Avatar,
		Bio:             user.Bio,
		Status:          user.Status,
		EmailVerified:   user.EmailVerified,
		LastLoginAt:     lastLoginAt,
		StoryboardCount: user.StoryboardCount,
		Followers:       user.Followers,
		Following:       user.Following,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}

func (r *OAuthRepository) thirdPartyLoginModelToDomain(model *OAuthThirdPartyLogin) *domain.ThirdPartyLogin {
	var deletedAt *int64
	if model.DeletedAt.Valid {
		ts := model.DeletedAt.Time.Unix()
		deletedAt = &ts
	}
	return &domain.ThirdPartyLogin{
		ID:               model.ID,
		UserID:           model.UserID,
		Provider:         domain.ThirdPartyProvider(model.Provider),
		ProviderUserID:   model.ProviderUserID,
		ProviderEmail:    model.ProviderEmail,
		ProviderUserName: model.ProviderUserName,
		ProviderUserInfo: model.ProviderUserInfo,
		AccessToken:      model.AccessToken,
		RefreshToken:     model.RefreshToken,
		TokenExpireTime:  model.TokenExpireTime,
		Status:           domain.ThirdPartyLoginStatus(model.Status),
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
		DeletedAt:        deletedAt,
	}
}
