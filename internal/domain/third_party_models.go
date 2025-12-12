package domain

import (
	"encoding/json"
	"time"
)

// ThirdPartyProvider 第三方登录提供商
type ThirdPartyProvider string

const (
	ProviderGoogle   ThirdPartyProvider = "google"
	ProviderApple    ThirdPartyProvider = "apple"
	ProviderFacebook ThirdPartyProvider = "facebook"
	ProviderWechat   ThirdPartyProvider = "wechat"
	ProviderAlipay   ThirdPartyProvider = "alipay"
)

// ThirdPartyLoginStatus 第三方登录状态
type ThirdPartyLoginStatus int

const (
	ThirdPartyLoginStatusNormal  ThirdPartyLoginStatus = 1
	ThirdPartyLoginStatusDisable ThirdPartyLoginStatus = 2
)

// ThirdPartyLogin 第三方登录信息
type ThirdPartyLogin struct {
	ID               string                `json:"id"`
	UserID           string                `json:"userId"`
	Provider         ThirdPartyProvider    `json:"provider"`
	ProviderUserID   string                `json:"providerUserId"`
	ProviderEmail    string                `json:"providerEmail"`
	ProviderUserName string                `json:"providerUserName"`
	ProviderUserInfo string                `json:"providerUserInfo"`
	AccessToken      string                `json:"accessToken"`
	RefreshToken     string                `json:"refreshToken"`
	TokenExpireTime  *int64                `json:"tokenExpireTime"`
	Status           ThirdPartyLoginStatus `json:"status"`
	CreatedAt        int64                 `json:"createdAt"`
	UpdatedAt        int64                 `json:"updatedAt"`
	DeletedAt        *int64                `json:"deletedAt,omitempty"`
}

// GetProviderUserInfoMap 获取第三方用户信息的Map格式
func (t *ThirdPartyLogin) GetProviderUserInfoMap() (map[string]interface{}, error) {
	if t.ProviderUserInfo == "" {
		return map[string]interface{}{}, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(t.ProviderUserInfo), &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// SetProviderUserInfoMap 设置第三方用户信息的Map格式
func (t *ThirdPartyLogin) SetProviderUserInfoMap(m map[string]interface{}) error {
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

// IsTokenExpired 检查访问令牌是否过期
func (t *ThirdPartyLogin) IsTokenExpired() bool {
	if t.TokenExpireTime == nil {
		return false // 如果没有过期时间，认为未过期
	}
	return time.Now().Unix() > *t.TokenExpireTime
}

// ThirdPartyLoginRequest 第三方登录请求
type ThirdPartyLoginRequest struct {
	Provider    string `json:"provider" binding:"required,oneof=google apple"`
	AuthCode    string `json:"authCode" binding:"required"`
	IDToken     string `json:"idToken,omitempty"`     // Apple ID Token
	State       string `json:"state,omitempty"`       // 用于验证请求的完整性
	RedirectURI string `json:"redirectUri,omitempty"` // 重定向URI
}

// ThirdPartyLoginResponse 第三方登录响应
type ThirdPartyLoginResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	IsNewUser    bool   `json:"isNewUser"` // 是否为新用户
}

// ThirdProfile 第三方用户信息
type ThirdProfile struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Verified bool   `json:"verified"`
}

// BindThirdPartyRequest 绑定第三方账号请求
type BindThirdPartyRequest struct {
	Provider    string `json:"provider" binding:"required,oneof=google apple"`
	AuthCode    string `json:"authCode" binding:"required"`
	IDToken     string `json:"idToken,omitempty"`     // Apple ID Token
	RedirectURI string `json:"redirectUri,omitempty"` // 重定向URI
}

// UnbindThirdPartyRequest 解绑第三方账号请求
type UnbindThirdPartyRequest struct {
	Provider string `json:"provider" binding:"required,oneof=google apple"`
}

// ThirdPartyLinkResponse 第三方绑定状态响应
type ThirdPartyLinkResponse struct {
	Provider    string           `json:"provider"`
	IsLinked    bool             `json:"isLinked"`
	LinkInfo    *ThirdPartyLogin `json:"linkInfo,omitempty"`
	LinkedAt    *int64           `json:"linkedAt,omitempty"`
	LastLoginAt *int64           `json:"lastLoginAt,omitempty"`
	Profile     *ThirdProfile    `json:"profile,omitempty"`
}
