package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Agent Access Token 是 grapery 签发、grapery-agent 校验的短期凭证。
// grapery 仍是 auth/quota 权威方：先校验用户 JWT、活跃状态、会员/额度策略，
// 通过后才签发带 scope 的 token；客户端随后凭该 token 直连 grapery-agent。
//
// 线格式（与 grapery-agent internal/agentauth 保持一致）：
//
//	base64url(payloadJSON) + "." + base64url(HMAC_SHA256(payloadJSON, key))
//
// 校验方对 base64 解码后的原始 payload 字节做 HMAC 校验，再解析 claims，
// 因此双方无需对 JSON 做 canonical 化即可一致。
const (
	AgentTokenIssuer   = "grapery"
	AgentTokenAudience = "grapery-agent"
	agentTokenVersion  = "v1"
)

// AgentAccessClaims 是 Agent Access Token 的载荷。
type AgentAccessClaims struct {
	Version            string `json:"v"`
	Issuer             string `json:"iss"`
	Audience           string `json:"aud"`
	UserID             string `json:"userId"`
	RequestID          string `json:"requestId"`
	SessionID          string `json:"sessionId,omitempty"`
	Agent              string `json:"agent"`     // e.g. fragment-panel / fragment / character / storyboard
	Operation          string `json:"operation"` // chat | generate
	Scope              string `json:"scope,omitempty"` // e.g. agent:fragment-panel:chat
	Kind               string `json:"kind,omitempty"`
	QuotaMode          string `json:"quotaMode,omitempty"` // chat_only | reserve | budget
	QuotaReservationID string `json:"quotaReservationId,omitempty"`
	MaxTokens          int    `json:"maxTokens,omitempty"`
	MaxImages          int    `json:"maxImages,omitempty"`
	JTI                string `json:"jti"`
	IssuedAt           int64  `json:"iat"`
	ExpiresAt          int64  `json:"exp"`
}

// AgentAccessTokenRequest 描述一次 token 签发请求。
type AgentAccessTokenRequest struct {
	UserID             string
	Agent              string
	Operation          string
	Kind               string
	SessionID          string
	QuotaMode          string
	QuotaReservationID string
	Scope              string
	MaxTokens          int
	MaxImages          int
}

// AgentAccessTokenResult 返回给客户端，用于直连 grapery-agent。
type AgentAccessTokenResult struct {
	AgentAccessToken   string `json:"agentAccessToken"`
	TokenType          string `json:"tokenType"`
	ExpiresAt          int64  `json:"expiresAt"`
	ExpiresInSec       int    `json:"expiresInSec"`
	RequestID          string `json:"requestId"`
	JTI                string `json:"jti"`
	Agent              string `json:"agent"`
	Operation          string `json:"operation"`
	QuotaMode          string `json:"quotaMode,omitempty"`
	QuotaReservationID string `json:"quotaReservationId,omitempty"`
}

// AgentAccessTokenSigner 用对称密钥签发/校验 Agent Access Token。
type AgentAccessTokenSigner struct {
	secret []byte
	ttl    time.Duration
}

// NewAgentAccessTokenSigner 创建签发器；ttl<=0 时回落到 5 分钟。
func NewAgentAccessTokenSigner(secret string, ttl time.Duration) *AgentAccessTokenSigner {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &AgentAccessTokenSigner{secret: []byte(strings.TrimSpace(secret)), ttl: ttl}
}

// IsConfigured 仅当密钥已设置时返回 true。
func (s *AgentAccessTokenSigner) IsConfigured() bool {
	return s != nil && len(s.secret) > 0
}

// TTL 返回 token 有效期。
func (s *AgentAccessTokenSigner) TTL() time.Duration {
	if s == nil || s.ttl <= 0 {
		return 5 * time.Minute
	}
	return s.ttl
}

// Issue 生成一枚短期 Agent Access Token。
func (s *AgentAccessTokenSigner) Issue(req AgentAccessTokenRequest) (*AgentAccessTokenResult, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("agent access token signing is not configured")
	}
	if strings.TrimSpace(req.UserID) == "" {
		return nil, fmt.Errorf("userId is required")
	}
	if strings.TrimSpace(req.Agent) == "" {
		return nil, fmt.Errorf("agent is required")
	}
	op := strings.TrimSpace(req.Operation)
	if op == "" {
		op = "chat"
	}
	now := time.Now()
	exp := now.Add(s.ttl)
	requestID := randomID("req_")
	jti := randomID("jti_")
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = BuildScope(req.Agent, op)
	}
	claims := AgentAccessClaims{
		Version:            agentTokenVersion,
		Issuer:             AgentTokenIssuer,
		Audience:           AgentTokenAudience,
		UserID:             strings.TrimSpace(req.UserID),
		RequestID:          requestID,
		SessionID:          strings.TrimSpace(req.SessionID),
		Agent:              strings.TrimSpace(req.Agent),
		Operation:          op,
		Scope:              scope,
		Kind:               strings.TrimSpace(req.Kind),
		QuotaMode:          strings.TrimSpace(req.QuotaMode),
		QuotaReservationID: strings.TrimSpace(req.QuotaReservationID),
		MaxTokens:          req.MaxTokens,
		MaxImages:          req.MaxImages,
		JTI:                jti,
		IssuedAt:           now.Unix(),
		ExpiresAt:          exp.Unix(),
	}
	token, err := signAgentClaims(s.secret, claims)
	if err != nil {
		return nil, err
	}
	return &AgentAccessTokenResult{
		AgentAccessToken:   token,
		TokenType:          "Bearer",
		ExpiresAt:          claims.ExpiresAt,
		ExpiresInSec:       int(s.ttl.Seconds()),
		RequestID:          requestID,
		JTI:                jti,
		Agent:              claims.Agent,
		Operation:          claims.Operation,
		QuotaMode:          claims.QuotaMode,
		QuotaReservationID: claims.QuotaReservationID,
	}, nil
}

// Verify 校验 token 并返回 claims（供测试与可选的 grapery 侧校验复用）。
func (s *AgentAccessTokenSigner) Verify(token string) (*AgentAccessClaims, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("agent access token signing is not configured")
	}
	claims, err := parseAndVerifyAgentClaims(s.secret, token)
	if err != nil {
		return nil, err
	}
	if claims.Audience != AgentTokenAudience {
		return nil, fmt.Errorf("invalid token audience")
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}
	return claims, nil
}

func signAgentClaims(secret []byte, claims AgentAccessClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	sig := mac.Sum(nil)
	enc := base64.RawURLEncoding
	return enc.EncodeToString(payload) + "." + enc.EncodeToString(sig), nil
}

func parseAndVerifyAgentClaims(secret []byte, token string) (*AgentAccessClaims, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed agent access token")
	}
	enc := base64.RawURLEncoding
	payload, err := enc.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid token payload encoding")
	}
	sig, err := enc.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token signature encoding")
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(expected, sig) {
		return nil, fmt.Errorf("invalid token signature")
	}
	var claims AgentAccessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid token claims")
	}
	return &claims, nil
}

func randomID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return prefix + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return prefix + hex.EncodeToString(b)
}
