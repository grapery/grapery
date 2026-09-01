package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

var (
	// JWT 密钥，必须通过 SetJWTSecret 设置
	// SECURITY: 不提供默认值，必须在启动时通过配置设置
	jwtSecret []byte

	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrSecretNotSet = errors.New("JWT secret not configured")
)

const (
	accessTokenIssuer  = "grapery"
	refreshTokenIssuer = "grapery-refresh"
	accessTokenType    = "access"
	refreshTokenType   = "refresh"
	refreshTokenTTL    = 90 * 24 * time.Hour
)

var jwtLogger = logrus.WithField("module", "jwt")

// Claims JWT 声明
type Claims struct {
	UserID    string `json:"userId"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	TokenType string `json:"tokenType,omitempty"`
	DeviceID  string `json:"deviceId,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID, username, email string) (string, error) {
	// SECURITY: Check if secret is configured
	if len(jwtSecret) == 0 {
		return "", ErrSecretNotSet
	}

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // 24小时过期

	claims := Claims{
		UserID:    userID,
		Username:  username,
		Email:     email,
		TokenType: accessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    accessTokenIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)

	if err != nil {
		jwtLogger.WithFields(logrus.Fields{
			"user_id":  userID,
			"username": username,
			"email":    email,
			"error":    err,
		}).Error("Failed to generate JWT token")
	} else {
		jwtLogger.WithFields(logrus.Fields{
			"user_id":    userID,
			"username":   username,
			"email":      email,
			"expires_at": expiresAt.Unix(),
			"issued_at":  now.Unix(),
		}).Info("JWT token generated successfully")
	}

	return tokenString, err
}

// GenerateRefreshToken keeps compatibility with non-device-aware clients.
func GenerateRefreshToken(userID string) (string, error) {
	return GenerateRefreshTokenForDevice(userID, "")
}

// GenerateRefreshTokenForDevice creates a long-lived, device-bound refresh token.
// The client keeps its device ID in a ThisDeviceOnly Keychain item, so a restored
// backup cannot silently reuse the session on a different iPhone.
func GenerateRefreshTokenForDevice(userID, deviceID string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", ErrSecretNotSet
	}
	now := time.Now()
	expiresAt := now.Add(refreshTokenTTL)

	claims := Claims{
		UserID:    userID,
		TokenType: refreshTokenType,
		DeviceID:  deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    refreshTokenIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken parses access tokens only. Legacy access tokens without tokenType
// remain valid when (and only when) their issuer is the access-token issuer.
func ParseToken(tokenString string) (*Claims, error) {
	return parseTokenForPurpose(tokenString, accessTokenIssuer, accessTokenType)
}

// ParseRefreshToken parses refresh tokens only, preventing an access token from
// being exchanged at /auth/refresh.
func ParseRefreshToken(tokenString string) (*Claims, error) {
	return parseTokenForPurpose(tokenString, refreshTokenIssuer, refreshTokenType)
}

func parseTokenForPurpose(tokenString, expectedIssuer, expectedType string) (*Claims, error) {
	// SECURITY: Check if secret is configured
	if len(jwtSecret) == 0 {
		return nil, ErrSecretNotSet
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return jwtSecret, nil
	}, jwt.WithIssuer(expectedIssuer))

	if err != nil {
		jwtLogger.WithFields(logrus.Fields{
			"error":     err,
			"token_len": len(tokenString),
		}).Warn("Failed to parse JWT token")

		if errors.Is(err, jwt.ErrTokenExpired) {
			jwtLogger.Warn("Token expired")
			return nil, ErrExpiredToken
		}
		jwtLogger.Warn("Invalid token")
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid && claims.UserID != "" &&
		(claims.TokenType == expectedType || claims.TokenType == "") {
		jwtLogger.WithFields(logrus.Fields{
			"user_id":       claims.UserID,
			"username":      claims.Username,
			"email":         claims.Email,
			"expires_at":    claims.ExpiresAt.Time.Unix(),
			"issued_at":     claims.IssuedAt.Time.Unix(),
			"remaining_sec": claims.ExpiresAt.Time.Sub(time.Now()).Seconds(),
		}).Info("JWT token validated successfully")
		return claims, nil
	}

	jwtLogger.Warn("Invalid token claims")
	return nil, ErrInvalidToken
}

// ValidateToken 验证 Token 是否有效
func ValidateToken(tokenString string) bool {
	_, err := ParseToken(tokenString)
	return err == nil
}

// SetJWTSecret 设置 JWT 密钥（用于配置）
func SetJWTSecret(secret string) {
	oldSecret := string(jwtSecret)
	jwtSecret = []byte(secret)
	secretDisplay := secret
	if len(secret) > 10 {
		secretDisplay = secret[:10] + "..."
	}
	jwtLogger.WithFields(logrus.Fields{
		"secret_length":  len(secret),
		"secret_preview": secretDisplay,
		"secret_changed": oldSecret != secret,
		"was_default":    oldSecret == "grapery-secret-key-change-in-production",
	}).Info("JWT secret configured")
}
