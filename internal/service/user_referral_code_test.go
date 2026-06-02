package service

import (
	"strings"
	"testing"
)

func TestGenerateUserReferralCode_nonEmptyUniquePerUser(t *testing.T) {
	a := GenerateUserReferralCode("550e8400-e29b-41d4-a716-446655440000")
	b := GenerateUserReferralCode("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	if a == "" || b == "" {
		t.Fatal("expected non-empty referral codes")
	}
	if a == b {
		t.Fatalf("expected distinct codes, got %q and %q", a, b)
	}
	if !strings.HasPrefix(a, "WZ") {
		t.Fatalf("expected WZ prefix, got %q", a)
	}
	if len(a) > 20 {
		t.Fatalf("referral code exceeds column size: %q", a)
	}
}
