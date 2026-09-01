package auth

import (
	"testing"
	"time"
)

func TestTokenPurposeSeparation(t *testing.T) {
	SetJWTSecret("test-secret-at-least-long-enough")

	access, err := GenerateToken("user-1", "alice", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := GenerateRefreshTokenForDevice("user-1", "device-1")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ParseToken(access); err != nil {
		t.Fatalf("access token rejected as access token: %v", err)
	}
	if _, err := ParseRefreshToken(refresh); err != nil {
		t.Fatalf("refresh token rejected as refresh token: %v", err)
	}
	if _, err := ParseRefreshToken(access); err == nil {
		t.Fatal("access token must not be accepted as a refresh token")
	}
	if _, err := ParseToken(refresh); err == nil {
		t.Fatal("refresh token must not be accepted as an access token")
	}
}

func TestDeviceBoundRefreshToken(t *testing.T) {
	SetJWTSecret("test-secret-at-least-long-enough")

	token, err := GenerateRefreshTokenForDevice("user-1", "ios-installation-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseRefreshToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.DeviceID != "ios-installation-1" {
		t.Fatalf("device ID = %q, want ios-installation-1", claims.DeviceID)
	}
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining < 89*24*time.Hour || remaining > 91*24*time.Hour {
		t.Fatalf("refresh lifetime = %v, want about 90 days", remaining)
	}
}
