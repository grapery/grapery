package models

import (
	"context"
	"encoding/json"
	"time"
)

// ThirdPartyProvider 第三方提供商
type ThirdPartyProvider string

const (
	ProviderApple    ThirdPartyProvider = "apple"
	ProviderGoogle   ThirdPartyProvider = "google"
	ProviderFacebook ThirdPartyProvider = "facebook"
	ProviderWechat   ThirdPartyProvider = "wechat"
	ProviderAlipay   ThirdPartyProvider = "alipay"
)

// ThirdPartyLoginStatus 状态：1-正常，2-禁用
type ThirdPartyLoginStatus int

const (
	ThirdPartyLoginStatusNormal  ThirdPartyLoginStatus = 1
	ThirdPartyLoginStatusDisable ThirdPartyLoginStatus = 2
)

// ThirdPartyLogin 对应表 `third_party_logins`
type ThirdPartyLogin struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase

	UserID           int64                 `gorm:"column:user_id;index:idx_user_id" json:"user_id"`                                            // 系统用户ID
	Provider         ThirdPartyProvider    `gorm:"column:provider;index:idx_provider;size:50;uniqueIndex:uk_provider_user_id" json:"provider"` // 提供商（与 provider_user_id 组成联合唯一）
	ProviderUserID   string                `gorm:"column:provider_user_id;size:255;uniqueIndex:uk_provider_user_id" json:"provider_user_id"`   // 第三方用户ID（与 provider 组成联合唯一）
	ProviderEmail    string                `gorm:"column:provider_email;size:255;index:idx_provider_email" json:"provider_email"`              // 第三方用户邮箱
	ProviderUserName string                `gorm:"column:provider_user_name;size:255" json:"provider_user_name"`                               // 第三方用户名称
	ProviderUserInfo string                `gorm:"column:provider_user_info;type:json" json:"provider_user_info"`                              // 第三方用户信息 JSON（扩展信息）
	AccessToken      string                `gorm:"column:access_token;size:500" json:"access_token"`                                           // 访问令牌
	RefreshToken     string                `gorm:"column:refresh_token;size:500" json:"refresh_token"`                                         // 刷新令牌
	TokenExpireTime  *time.Time            `gorm:"column:token_expire_time" json:"token_expire_time"`                                          // 令牌过期时间
	Status           ThirdPartyLoginStatus `gorm:"column:status;index:idx_status;default:1" json:"status"`                                     // 状态
}

func (t ThirdPartyLogin) TableName() string { return "third_party_logins" }

// CreateThirdPartyLogin 创建绑定
func CreateThirdPartyLogin(ctx context.Context, l *ThirdPartyLogin) error {
	return DataBase().WithContext(ctx).Create(l).Error
}

// GetThirdPartyLogin 获取
func GetThirdPartyLogin(ctx context.Context, id uint) (*ThirdPartyLogin, error) {
	var l ThirdPartyLogin
	if err := DataBase().WithContext(ctx).Where("id = ? and deleted = ?", id, 0).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// GetThirdPartyLoginByProviderUser 唯一键(provider, provider_user_id)
func GetThirdPartyLoginByProviderUser(ctx context.Context, provider ThirdPartyProvider, providerUserID string) (*ThirdPartyLogin, error) {
	var l ThirdPartyLogin
	if err := DataBase().WithContext(ctx).
		Where("provider = ? and provider_user_id = ? and deleted = ?", provider, providerUserID, 0).
		First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// GetThirdPartyLoginByProviderEmail 通过提供商和邮箱查找第三方登录记录
func GetThirdPartyLoginByProviderEmail(ctx context.Context, provider ThirdPartyProvider, email string) (*ThirdPartyLogin, error) {
	var l ThirdPartyLogin
	if err := DataBase().WithContext(ctx).
		Where("provider = ? and provider_email = ? and deleted = ?", provider, email, 0).
		First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// GetThirdPartyLoginsByEmail 通过邮箱查找所有第三方登录记录
func GetThirdPartyLoginsByEmail(ctx context.Context, email string) ([]*ThirdPartyLogin, error) {
	var list []*ThirdPartyLogin
	if err := DataBase().WithContext(ctx).
		Where("provider_email = ? and deleted = ?", email, 0).
		Order("create_at desc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListThirdPartyLogins 列表（分页）
func ListThirdPartyLogins(ctx context.Context, offset, limit int) ([]*ThirdPartyLogin, error) {
	var list []*ThirdPartyLogin
	if err := DataBase().WithContext(ctx).
		Where("deleted = ?", 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetThirdPartyLoginsByUserID 根据用户ID获取（分页）
func GetThirdPartyLoginsByUserID(ctx context.Context, userID int64, offset, limit int) ([]*ThirdPartyLogin, error) {
	var list []*ThirdPartyLogin
	if err := DataBase().WithContext(ctx).
		Where("user_id = ? and deleted = ?", userID, 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateThirdPartyLogin 更新
func UpdateThirdPartyLogin(ctx context.Context, id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return DataBase().WithContext(ctx).
		Model(&ThirdPartyLogin{}).
		Where("id = ? and deleted = ?", id, 0).
		Updates(updates).Error
}

// UpdateThirdPartyLoginStatus 更新状态
func UpdateThirdPartyLoginStatus(ctx context.Context, id uint, status ThirdPartyLoginStatus) error {
	return DataBase().WithContext(ctx).
		Model(&ThirdPartyLogin{}).
		Where("id = ? and deleted = ?", id, 0).
		Update("status", status).Error
}

// SoftDeleteThirdPartyLogin 软删除
func SoftDeleteThirdPartyLogin(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&ThirdPartyLogin{}).
		Where("id = ?", id).
		Update("deleted", 1).Error
}

// ParseProviderUserInfo 解析 JSON 到 map
func (t *ThirdPartyLogin) ParseProviderUserInfo() (map[string]interface{}, error) {
	if t.ProviderUserInfo == "" {
		return map[string]interface{}{}, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(t.ProviderUserInfo), &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// SetProviderUserInfo 设置 JSON
func (t *ThirdPartyLogin) SetProviderUserInfo(m map[string]interface{}) error {
	if m == nil {
		t.ProviderUserInfo = "{}"
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	t.ProviderUserInfo = string(b)
	return nil
}

// GetUserIDByThirdPartyLogin 通过第三方登录信息获取系统用户ID
func GetUserIDByThirdPartyLogin(ctx context.Context, provider ThirdPartyProvider, providerUserID string) (int64, error) {
	login, err := GetThirdPartyLoginByProviderUser(ctx, provider, providerUserID)
	if err != nil {
		return 0, err
	}
	return login.UserID, nil
}

// CreateOrUpdateThirdPartyLogin 创建或更新第三方登录记录
func CreateOrUpdateThirdPartyLogin(ctx context.Context, provider ThirdPartyProvider, providerUserID, providerEmail, providerUserName string, userID int64, userInfo map[string]interface{}, accessToken, refreshToken string, expireTime *time.Time) (*ThirdPartyLogin, error) {
	// 先尝试查找现有记录
	existingLogin, err := GetThirdPartyLoginByProviderUser(ctx, provider, providerUserID)
	if err != nil && err.Error() != "record not found" {
		return nil, err
	}

	if existingLogin != nil {
		// 更新现有记录
		updates := map[string]interface{}{
			"user_id":            userID,
			"provider_email":     providerEmail,
			"provider_user_name": providerUserName,
			"provider_user_info": "",
			"access_token":       accessToken,
			"refresh_token":      refreshToken,
			"token_expire_time":  expireTime,
			"status":             ThirdPartyLoginStatusNormal,
			"update_at":          time.Now(),
		}

		// 设置用户信息
		if userInfo != nil {
			jsonData, err := json.Marshal(userInfo)
			if err == nil {
				updates["provider_user_info"] = string(jsonData)
			}
		}

		err = UpdateThirdPartyLogin(ctx, existingLogin.ID, updates)
		if err != nil {
			return nil, err
		}

		// 重新获取更新后的记录
		return GetThirdPartyLogin(ctx, existingLogin.ID)
	} else {
		// 创建新记录
		newLogin := &ThirdPartyLogin{
			UserID:           userID,
			Provider:         provider,
			ProviderUserID:   providerUserID,
			ProviderEmail:    providerEmail,
			ProviderUserName: providerUserName,
			AccessToken:      accessToken,
			RefreshToken:     refreshToken,
			TokenExpireTime:  expireTime,
			Status:           ThirdPartyLoginStatusNormal,
		}

		// 设置用户信息
		if userInfo != nil {
			err = newLogin.SetProviderUserInfo(userInfo)
			if err != nil {
				return nil, err
			}
		}

		err = CreateThirdPartyLogin(ctx, newLogin)
		if err != nil {
			return nil, err
		}

		return newLogin, nil
	}
}
