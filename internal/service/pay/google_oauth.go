package pay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// GoogleOAuthConfig Google OAuth2 配置
type GoogleOAuthConfig struct {
	// 生产环境配置
	ClientID string `json:"client_id"` // Google OAuth2 Client ID

	// 其他配置
	TimeoutSeconds int `json:"timeout_seconds,omitempty"` // 请求超时时间（秒），默认30秒
	CacheDuration  int `json:"cache_duration,omitempty"`  // 公钥缓存时间（小时），默认1小时
}

// GoogleIdentityTokenClaims 包含 Google Identity Token 的自定义声明
type GoogleIdentityTokenClaims struct {
	// 标准 JWT 声明
	Subject   string `json:"sub,omitempty"` // 用户唯一标识符
	Issuer    string `json:"iss,omitempty"` // 签发者
	Audience  string `json:"aud,omitempty"` // 受众
	ExpiresAt int64  `json:"exp,omitempty"` // 过期时间
	IssuedAt  int64  `json:"iat,omitempty"` // 签发时间

	// Google 特定声明
	Email         string `json:"email,omitempty"`          // 用户邮箱
	EmailVerified bool   `json:"email_verified,omitempty"` // 邮箱是否已验证
	Name          string `json:"name,omitempty"`           // 用户全名
	GivenName     string `json:"given_name,omitempty"`     // 名
	FamilyName    string `json:"family_name,omitempty"`    // 姓
	Picture       string `json:"picture,omitempty"`        // 用户头像 URL
	Locale        string `json:"locale,omitempty"`         // 用户语言环境
	HostedDomain  string `json:"hd,omitempty"`             // G Suite 域（如果是企业账号）
}

// GoogleSignInVerifier Google Sign-In 验证器
type GoogleSignInVerifier struct {
	config *GoogleOAuthConfig
	cache  *CachedGoogleKeySet
}

// CachedGoogleKeySet 带缓存的 Google 公钥集
type CachedGoogleKeySet struct {
	keySet jwk.Set
	expiry time.Time
	mutex  sync.RWMutex
}

// NewGoogleSignInVerifier 创建新的 Google Sign-In 验证器
func NewGoogleSignInVerifier(config *GoogleOAuthConfig) *GoogleSignInVerifier {
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 30
	}
	if config.CacheDuration == 0 {
		config.CacheDuration = 1
	}

	return &GoogleSignInVerifier{
		config: config,
		cache:  &CachedGoogleKeySet{},
	}
}

// VerifyToken 验证 Google Identity Token
func (v *GoogleSignInVerifier) VerifyToken(tokenString string) (*GoogleIdentityTokenClaims, error) {
	startTime := time.Now()

	// 1. 获取 Google 的公钥集（带缓存）
	fmt.Printf("开始获取 Google 公钥集，Client ID: %s\n", v.config.ClientID)
	keySet, err := v.getGooglePublicKeys()
	if err != nil {
		// 记录 OAuth 登录错误
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordOAuthLoginError("google", "network")
		}
		return nil, fmt.Errorf("failed to get Google public keys: %w", err)
	}
	fmt.Printf("成功获取 Google 公钥集，包含 %d 个密钥\n", keySet.Len())

	// 2. 解析并验证 JWT token
	// Google 的 issuer 可能是 "https://accounts.google.com" 或 "accounts.google.com"
	token, err := jwt.ParseString(tokenString,
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithAudience(v.config.ClientID),
	)
	if err != nil {
		// 记录 OAuth 登录失败
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordOAuthLogin("google", "failed", time.Since(startTime))
			metrics.RecordOAuthLoginError("google", "invalid_token")
		}
		return nil, fmt.Errorf("failed to parse/validate token: %w", err)
	}

	// 验证 issuer（手动验证，因为 Google 有两种格式）
	issuer := token.Issuer()
	if issuer != "https://accounts.google.com" && issuer != "accounts.google.com" {
		// 记录 OAuth 登录失败
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordOAuthLogin("google", "failed", time.Since(startTime))
			metrics.RecordOAuthLoginError("google", "invalid_token")
		}
		return nil, fmt.Errorf("invalid issuer: %s", issuer)
	}

	// 3. 提取声明
	claims := &GoogleIdentityTokenClaims{}

	// 提取标准 JWT 声明
	if sub := token.Subject(); sub != "" {
		claims.Subject = sub
	}
	if iss := token.Issuer(); iss != "" {
		claims.Issuer = iss
	}
	if aud := token.Audience(); len(aud) > 0 {
		claims.Audience = aud[0]
	}
	if exp := token.Expiration(); !exp.IsZero() {
		claims.ExpiresAt = exp.Unix()
	}
	if iat := token.IssuedAt(); !iat.IsZero() {
		claims.IssuedAt = iat.Unix()
	}

	// 手动提取自定义声明
	rawClaims := token.PrivateClaims()
	if email, ok := rawClaims["email"].(string); ok {
		claims.Email = email
	}
	if emailVerified, ok := rawClaims["email_verified"].(bool); ok {
		claims.EmailVerified = emailVerified
	}
	if name, ok := rawClaims["name"].(string); ok {
		claims.Name = name
	}
	if givenName, ok := rawClaims["given_name"].(string); ok {
		claims.GivenName = givenName
	}
	if familyName, ok := rawClaims["family_name"].(string); ok {
		claims.FamilyName = familyName
	}
	if picture, ok := rawClaims["picture"].(string); ok {
		claims.Picture = picture
	}
	if locale, ok := rawClaims["locale"].(string); ok {
		claims.Locale = locale
	}
	if hd, ok := rawClaims["hd"].(string); ok {
		claims.HostedDomain = hd
	}

	// 记录 OAuth 登录成功
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.RecordOAuthLogin("google", "success", time.Since(startTime))
	}

	return claims, nil
}

// GetUserIdentifier 从 token 中提取用户标识符（sub 声明）
func (v *GoogleSignInVerifier) GetUserIdentifier(tokenString string) (string, error) {
	// 解析 token 而不验证（仅用于提取 sub）
	token, err := jwt.ParseString(tokenString, jwt.WithVerify(false))
	if err != nil {
		return "", err
	}

	sub := token.Subject()
	if sub != "" {
		return sub, nil
	}

	return "", fmt.Errorf("subject not found in token")
}

// getGooglePublicKeys 获取 Google 的公钥集（带缓存）
func (v *GoogleSignInVerifier) getGooglePublicKeys() (jwk.Set, error) {
	return v.cache.Get(v.config.CacheDuration)
}

// Get 获取缓存的公钥集，如果缓存过期则重新获取
func (c *CachedGoogleKeySet) Get(cacheDurationHours int) (jwk.Set, error) {
	c.mutex.RLock()
	cacheDuration := time.Duration(cacheDurationHours) * time.Hour
	if time.Now().Before(c.expiry) {
		keySet := c.keySet
		c.mutex.RUnlock()
		return keySet, nil
	}
	c.mutex.RUnlock()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 双重检查
	if time.Now().Before(c.expiry) {
		return c.keySet, nil
	}

	// Google 的 JWK URL
	const googleJWKURL = "https://www.googleapis.com/oauth2/v3/certs"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keySet, err := jwk.Fetch(ctx, googleJWKURL)
	if err != nil {
		return nil, err
	}

	c.keySet = keySet
	c.expiry = time.Now().Add(cacheDuration)
	return keySet, nil
}

// IsValid 检查验证器配置是否有效
func (v *GoogleSignInVerifier) IsValid() bool {
	return v.config != nil && v.config.ClientID != ""
}

// GetClientID 获取当前配置的 Client ID
func (v *GoogleSignInVerifier) GetClientID() string {
	return v.config.ClientID
}
