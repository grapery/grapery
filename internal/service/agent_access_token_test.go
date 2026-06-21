package service

import (
	"strings"
	"testing"
	"time"
)

func TestAgentAccessTokenSigner_IssueAndVerify(t *testing.T) {
	signer := NewAgentAccessTokenSigner("test-signing-key", 5*time.Minute)
	if !signer.IsConfigured() {
		t.Fatal("signer should be configured")
	}

	res, err := signer.Issue(AgentAccessTokenRequest{
		UserID:    "user-1",
		Agent:     "fragment-panel",
		Operation: "generate",
		Kind:      "fragment_panel",
		QuotaMode: "budget",
		MaxTokens: 1000,
	})
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}
	if res.AgentAccessToken == "" || res.RequestID == "" || res.JTI == "" {
		t.Fatal("expected non-empty token/requestId/jti")
	}
	if !strings.Contains(res.AgentAccessToken, ".") {
		t.Fatalf("token must be payload.signature, got %q", res.AgentAccessToken)
	}

	claims, err := signer.Verify(res.AgentAccessToken)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if claims.UserID != "user-1" || claims.Agent != "fragment-panel" || claims.Operation != "generate" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.Issuer != AgentTokenIssuer || claims.Audience != AgentTokenAudience {
		t.Fatalf("unexpected iss/aud: %+v", claims)
	}
}

func TestAgentAccessTokenSigner_RejectsTampered(t *testing.T) {
	signer := NewAgentAccessTokenSigner("k1", time.Minute)
	res, err := signer.Issue(AgentAccessTokenRequest{UserID: "u", Agent: "fragment"})
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}
	tampered := res.AgentAccessToken[:len(res.AgentAccessToken)-2] + "xx"
	if _, err := signer.Verify(tampered); err == nil {
		t.Fatal("expected verify to fail for tampered token")
	}

	// Different key must reject.
	other := NewAgentAccessTokenSigner("k2", time.Minute)
	if _, err := other.Verify(res.AgentAccessToken); err == nil {
		t.Fatal("expected verify to fail with different key")
	}
}

func TestAgentAccessTokenSigner_RejectsExpired(t *testing.T) {
	signer := NewAgentAccessTokenSigner("k", -time.Second) // TTL<=0 -> coerced to 5m
	// Force expiry by signing claims with past exp directly.
	past := time.Now().Add(-time.Hour).Unix()
	claims := AgentAccessClaims{
		Version:   agentTokenVersion,
		Issuer:    AgentTokenIssuer,
		Audience:  AgentTokenAudience,
		UserID:    "u",
		Agent:     "fragment",
		Operation: "chat",
		JTI:       "jti_x",
		IssuedAt:  past,
		ExpiresAt: past,
	}
	token, err := signAgentClaims(signer.secret, claims)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if _, err := signer.Verify(token); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestAgentAccessTokenSigner_NotConfigured(t *testing.T) {
	signer := NewAgentAccessTokenSigner("  ", time.Minute)
	if signer.IsConfigured() {
		t.Fatal("blank key should not be configured")
	}
	if _, err := signer.Issue(AgentAccessTokenRequest{UserID: "u", Agent: "fragment"}); err == nil {
		t.Fatal("expected issue to fail when not configured")
	}
}
