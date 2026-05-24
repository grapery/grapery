package common

import "testing"

func TestMembershipTierRank(t *testing.T) {
	tests := []struct {
		tier string
		want int
	}{
		{"free", 0},
		{"basic", 1},
		{"pro", 1},
		{"premium", 2},
		{"prime", 2},
		{"ultra", 2},
		{"", 0},
	}
	for _, tc := range tests {
		if got := MembershipTierRank(tc.tier); got != tc.want {
			t.Errorf("MembershipTierRank(%q) = %d, want %d", tc.tier, got, tc.want)
		}
	}
}

func TestDetectSubscriptionChangeKind(t *testing.T) {
	tests := []struct {
		old, new string
		want     SubscriptionChangeKind
	}{
		{"", "com.grapery.pro.monthly", ChangeInitial},
		{"com.grapery.pro.monthly", "com.grapery.pro.monthly", ChangeRenewal},
		{"com.grapery.pro.monthly", "com.grapery.prime.monthly", ChangeUpgrade},
		{"com.grapery.prime.monthly", "com.grapery.pro.monthly", ChangeDowngradeScheduled},
		{"com.grapery.pro.monthly", "com.grapery.pro.yearly", ChangeRenewal},
		{"com.grapery.pro.monthly", "", ChangeRenewal},
	}
	for _, tc := range tests {
		if got := DetectSubscriptionChangeKind(tc.old, tc.new); got != tc.want {
			t.Errorf("DetectSubscriptionChangeKind(%q, %q) = %q, want %q", tc.old, tc.new, got, tc.want)
		}
	}
}

func TestNormalizeAppleNotificationAction(t *testing.T) {
	tests := []struct {
		nt, st string
		want   SubscriptionChangeKind
	}{
		{"SUBSCRIBED", "INITIAL_BUY", ChangeInitial},
		{"SUBSCRIBED", "UPGRADE", ChangeUpgrade},
		{"SUBSCRIBED", "DOWNGRADE", ChangeDowngradeScheduled},
		{"DID_RENEW", "", ChangeRenewal},
		{"DID_CHANGE_RENEWAL_STATUS", "AUTO_RENEW_DISABLED", ChangeCancelRenewal},
		{"DID_CHANGE_RENEWAL_STATUS", "AUTO_RENEW_ENABLED", ChangeRenewal},
		{"EXPIRED", "", ChangeExpired},
		{"REFUND", "", ChangeRevoked},
		{"REVOKE", "", ChangeRevoked},
		{"DID_FAIL_TO_RENEW", "", ChangeExpired},
		{"UNKNOWN", "", ""},
	}
	for _, tc := range tests {
		if got := NormalizeAppleNotificationAction(tc.nt, tc.st); got != tc.want {
			t.Errorf("NormalizeAppleNotificationAction(%q, %q) = %q, want %q", tc.nt, tc.st, got, tc.want)
		}
	}
}
