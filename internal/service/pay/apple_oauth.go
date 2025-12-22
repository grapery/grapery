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

// AppleOAuthConfig Apple OAuth2 配置
type AppleOAuthConfig struct {
	// 生产环境配置
	BundleID string `json:"bundle_id"` // iOS App 的 Bundle Identifier，例如：com.yourapp.bundleid

	// 其他配置
	TimeoutSeconds int `json:"timeout_seconds,omitempty"` // 请求超时时间（秒），默认30秒
	CacheDuration  int `json:"cache_duration,omitempty"`  // 公钥缓存时间（小时），默认1小时
}

// AppleIdentityTokenClaims 包含 Apple Identity Token 的自定义声明
type AppleIdentityTokenClaims struct {
	// 标准 JWT 声明
	Subject   string `json:"sub,omitempty"` // 用户唯一标识符
	Issuer    string `json:"iss,omitempty"` // 签发者
	Audience  string `json:"aud,omitempty"` // 受众
	ExpiresAt int64  `json:"exp,omitempty"` // 过期时间
	IssuedAt  int64  `json:"iat,omitempty"` // 签发时间
	NotBefore int64  `json:"nbf,omitempty"` // 生效时间

	// Apple 特定声明
	Email          string `json:"email,omitempty"`            // 用户邮箱（仅在首次登录时提供）
	EmailVerified  bool   `json:"email_verified,omitempty"`   // 邮箱是否已验证
	IsPrivateEmail bool   `json:"is_private_email,omitempty"` // 是否为私有邮箱（Apple 中转邮箱）
	AuthTime       int64  `json:"auth_time,omitempty"`        // 认证时间
	FullName       string `json:"full_name,omitempty"`        // 用户全名（仅在首次登录时提供）
}

// AppleSignInVerifier Apple Sign-In 验证器
type AppleSignInVerifier struct {
	config *AppleOAuthConfig
	cache  *CachedAppleKeySet
}

// CachedAppleKeySet 带缓存的 Apple 公钥集
type CachedAppleKeySet struct {
	keySet jwk.Set
	expiry time.Time
	mutex  sync.RWMutex
}

// NewAppleSignInVerifier 创建新的 Apple Sign-In 验证器
func NewAppleSignInVerifier(config *AppleOAuthConfig) *AppleSignInVerifier {
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 30
	}
	if config.CacheDuration == 0 {
		config.CacheDuration = 1
	}

	return &AppleSignInVerifier{
		config: config,
		cache:  &CachedAppleKeySet{},
	}
}

// VerifyToken 验证 Apple Identity Token
func (v *AppleSignInVerifier) VerifyToken(tokenString string) (*AppleIdentityTokenClaims, error) {
	startTime := time.Now()

	// 1. 获取 Apple 的公钥集（带缓存）
	fmt.Printf("开始获取 Apple 公钥集，Bundle ID: %s\n", v.config.BundleID)
	keySet, err := v.getApplePublicKeys()
	if err != nil {
		// 记录 OAuth 登录错误
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordOAuthLoginError("apple", "network")
		}
		return nil, fmt.Errorf("failed to get Apple public keys: %w", err)
	}
	fmt.Printf("成功获取 Apple 公钥集，包含 %d 个密钥\n", keySet.Len())

	// 2. 解析并验证 JWT token
	token, err := jwt.ParseString(tokenString,
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithIssuer("https://appleid.apple.com"),
		jwt.WithAudience(v.config.BundleID),
	)
	if err != nil {
		// 记录 OAuth 登录失败
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordOAuthLogin("apple", "failed", time.Since(startTime))
			metrics.RecordOAuthLoginError("apple", "invalid_token")
		}
		return nil, fmt.Errorf("failed to parse/validate token: %w", err)
	}

	// 3. 提取声明
	claims := &AppleIdentityTokenClaims{}

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
	if nbf := token.NotBefore(); !nbf.IsZero() {
		claims.NotBefore = nbf.Unix()
	}

	// 手动提取自定义声明
	rawClaims := token.PrivateClaims()
	if email, ok := rawClaims["email"].(string); ok {
		claims.Email = email
	}
	if emailVerified, ok := rawClaims["email_verified"].(bool); ok {
		claims.EmailVerified = emailVerified
	}
	if isPrivateEmail, ok := rawClaims["is_private_email"].(bool); ok {
		claims.IsPrivateEmail = isPrivateEmail
	}
	if authTime, ok := rawClaims["auth_time"].(float64); ok {
		claims.AuthTime = int64(authTime)
	}
	if fullName, ok := rawClaims["full_name"].(string); ok {
		claims.FullName = fullName
	}

	// 记录 OAuth 登录成功
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.RecordOAuthLogin("apple", "success", time.Since(startTime))
	}

	return claims, nil
}

// GetUserIdentifier 从 token 中提取用户标识符（sub 声明）
func (v *AppleSignInVerifier) GetUserIdentifier(tokenString string) (string, error) {
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

// getApplePublicKeys 获取 Apple 的公钥集（带缓存）
func (v *AppleSignInVerifier) getApplePublicKeys() (jwk.Set, error) {
	return v.cache.Get(v.config.CacheDuration)
}

// Get 获取缓存的公钥集，如果缓存过期则重新获取
func (c *CachedAppleKeySet) Get(cacheDurationHours int) (jwk.Set, error) {
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

	const appleJWKURL = "https://appleid.apple.com/auth/keys"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keySet, err := jwk.Fetch(ctx, appleJWKURL)
	if err != nil {
		return nil, err
	}

	c.keySet = keySet
	c.expiry = time.Now().Add(cacheDuration)
	return keySet, nil
}

// IsValid 检查验证器配置是否有效
func (v *AppleSignInVerifier) IsValid() bool {
	return v.config != nil && v.config.BundleID != ""
}

// GetBundleID 获取当前配置的 Bundle ID
func (v *AppleSignInVerifier) GetBundleID() string {
	return v.config.BundleID
}
