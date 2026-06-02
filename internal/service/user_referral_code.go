package service

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// GenerateUserReferralCode returns a stable, unique-ish referral code for a user ID.
// Must be non-empty before INSERT: users.referral_code has a unique index and MySQL treats
// multiple ” values as duplicates (unlike NULL).
func GenerateUserReferralCode(userID string) string {
	compact := strings.ReplaceAll(strings.TrimSpace(userID), "-", "")
	if len(compact) > 18 {
		compact = compact[:18]
	}
	if compact != "" {
		return "WZ" + strings.ToUpper(compact)
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "WZ" + strings.ToUpper(base64.URLEncoding.EncodeToString(b)[:6])
}
