package kling

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenValiditySeconds = 1800 // 30 minutes
	cacheBufferSeconds   = 300  // refresh 5 minutes before expiry
	clockSkewSeconds     = 5
)

// tokenAuth generates and caches HS256 JWTs for Kling API (iss=accessKey, exp, nbf).
type tokenAuth struct {
	accessKey string
	secretKey string

	mu           sync.Mutex
	cachedToken  string
	tokenExpiry  int64 // unix seconds
}

func newTokenAuth(accessKey, secretKey string) (*tokenAuth, error) {
	if accessKey == "" {
		return nil, fmt.Errorf("kling access key is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("kling secret key is required")
	}
	return &tokenAuth{accessKey: accessKey, secretKey: secretKey}, nil
}

func (a *tokenAuth) bearerToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now().Unix()
	if a.cachedToken != "" && a.tokenExpiry > now+int64(cacheBufferSeconds) {
		return a.cachedToken, nil
	}

	claims := jwt.MapClaims{
		"iss": a.accessKey,
		"exp": now + tokenValiditySeconds,
		"nbf": now - clockSkewSeconds,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t.Header["typ"] = "JWT"

	signed, err := t.SignedString([]byte(a.secretKey))
	if err != nil {
		return "", err
	}
	a.cachedToken = signed
	a.tokenExpiry = now + tokenValiditySeconds
	return signed, nil
}
