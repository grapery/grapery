package pay

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// AppleOAuthConfig Apple OAuth2 配置
type AppleOAuthConfig struct {
	BundleID string `json:"bundle_id"` // iOS App 的 Bundle Identifier

	// Token revocation (optional — required for revoke/token exchange)
	TeamID      string `json:"team_id,omitempty"`      // Apple Developer Team ID
	KeyID       string `json:"key_id,omitempty"`        // Apple Sign-In Key ID
	PrivateKey  string `json:"private_key,omitempty"`   // Apple Sign-In Private Key (PEM)
	ClientID    string `json:"client_id,omitempty"`     // Service ID for web (defaults to BundleID)
	ServiceName string `json:"service_name,omitempty"`  // Service identifier for revoke

	TimeoutSeconds int `json:"timeout_seconds,omitempty"` // 请求超时时间（秒），默认30秒
	CacheDuration  int `json:"cache_duration,omitempty"`  // 公钥缓存时间（小时），默认1小时
}

// CanRevoke 检查是否配置了凭证撤销所需的参数
func (c *AppleOAuthConfig) CanRevoke() bool {
	return c.TeamID != "" && c.KeyID != "" && c.PrivateKey != ""
}

// appleTokenResponse Apple token API 响应
type appleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Error        string `json:"error"`
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
	Nonce          string `json:"nonce,omitempty"`            // request.nonce echo (SHA256 of raw nonce)
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
	logrus.Debugf("Fetching Apple public keys for Bundle ID: %s", v.config.BundleID)
	keySet, err := v.getApplePublicKeys()
	if err != nil {
		// 记录 OAuth 登录错误
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordOAuthLoginError("apple", "network")
		}
		return nil, fmt.Errorf("failed to get Apple public keys: %w", err)
	}
	logrus.Debugf("Fetched Apple public keys, count: %d", keySet.Len())

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
	if nonce, ok := rawClaims["nonce"].(string); ok {
		claims.Nonce = nonce
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

// CanRevoke 检查是否配置了凭证撤销所需的参数
func (v *AppleSignInVerifier) CanRevoke() bool {
	return v.config != nil && v.config.CanRevoke()
}

// GetBundleID 获取当前配置的 Bundle ID
func (v *AppleSignInVerifier) GetBundleID() string {
	return v.config.BundleID
}

// generateAppleClientSecret 生成 Apple client_secret JWT（ES256 签名）
func (v *AppleSignInVerifier) generateAppleClientSecret() (string, error) {
	if !v.config.CanRevoke() {
		return "", fmt.Errorf("apple token revocation not configured: missing team_id, key_id, or private_key")
	}

	privateKey, err := parseECPrivateKeyFromPEM(v.config.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse Apple private key: %w", err)
	}

	clientID := v.config.ClientID
	if clientID == "" {
		clientID = v.config.BundleID
	}

	now := time.Now()
	exp := now.Add(6 * time.Hour) // client_secret 最长 6 个月，这里用 6 小时

	// 手动构建 ES256 JWT
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256","kid":"` + v.config.KeyID + `"}`))

	payloadMap := map[string]interface{}{
		"iss": v.config.TeamID,
		"iat": now.Unix(),
		"exp": exp.Unix(),
		"aud": "https://appleid.apple.com",
		"sub": clientID,
	}
	payloadBytes, _ := json.Marshal(payloadMap)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := header + "." + payload
	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign client_secret: %w", err)
	}

	sigBytes := append(r.Bytes(), s.Bytes()...)
	// 确保 r 和 s 都是 32 字节（P-256）
	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)
	r.FillBytes(rBytes)
	s.FillBytes(sBytes)
	sigBytes = append(rBytes, sBytes...)

	signature := base64.RawURLEncoding.EncodeToString(sigBytes)
	return signingInput + "." + signature, nil
}

// ExchangeAuthorizationCode 用 authorizationCode 换取 Apple refresh token
func (v *AppleSignInVerifier) ExchangeAuthorizationCode(ctx context.Context, code string) (string, error) {
	clientSecret, err := v.generateAppleClientSecret()
	if err != nil {
		return "", err
	}

	clientID := v.config.ClientID
	if clientID == "" {
		clientID = v.config.BundleID
	}

	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}

	return v.doTokenRequest(ctx, data)
}

// RevokeAppleToken 撤销 Apple refresh token 或 access token
func (v *AppleSignInVerifier) RevokeAppleToken(ctx context.Context, token, tokenType string) error {
	clientSecret, err := v.generateAppleClientSecret()
	if err != nil {
		return err
	}

	clientID := v.config.ClientID
	if clientID == "" {
		clientID = v.config.BundleID
	}

	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"token":         {token},
		"token_type_hint": {tokenType},
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(v.config.TimeoutSeconds)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://appleid.apple.com/auth/revoke", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Apple revoke endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logrus.WithField("token_type", tokenType).Info("Apple token revoked successfully")
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(body),
	}).Warn("Apple token revocation returned non-200 status")
	return fmt.Errorf("apple revoke returned status %d: %s", resp.StatusCode, string(body))
}

// doTokenRequest 发送 token API 请求（用于 exchange 和 refresh）
func (v *AppleSignInVerifier) doTokenRequest(ctx context.Context, data url.Values) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(v.config.TimeoutSeconds)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://appleid.apple.com/auth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Apple token endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp appleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("apple token error: %s", tokenResp.Error)
	}

	if tokenResp.RefreshToken == "" {
		return "", fmt.Errorf("apple token response missing refresh_token")
	}

	return tokenResp.RefreshToken, nil
}

// parseECPrivateKeyFromPEM 从 PEM 字符串解析 ECDSA 私钥
func parseECPrivateKeyFromPEM(pemStr string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
	}

	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not ECDSA (got %T)", key)
	}

	return ecKey, nil
}
