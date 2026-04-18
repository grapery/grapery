package kling

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenAuthBearerCached(t *testing.T) {
	a, err := newTokenAuth("access-key-test", "secret-key-test")
	if err != nil {
		t.Fatal(err)
	}
	t1, err := a.bearerToken()
	if err != nil {
		t.Fatal(err)
	}
	t2, err := a.bearerToken()
	if err != nil {
		t.Fatal(err)
	}
	if t1 != t2 {
		t.Fatal("expected cached token")
	}
	parsed, err := jwt.Parse(t1, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret-key-test"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("invalid jwt: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || claims["iss"] != "access-key-test" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}
