package common

import "testing"

func TestNormalizeGoogleNotificationAction(t *testing.T) {
	cases := map[string]SubscriptionChangeKind{
		"SUBSCRIPTION_PURCHASED": ChangeInitial,
		"SUBSCRIPTION_RENEWED":   ChangeRenewal,
		"SUBSCRIPTION_REVOKED":   ChangeRevoked,
		"SUBSCRIPTION_EXPIRED":   ChangeExpired,
		"SUBSCRIPTION_CANCELED":  ChangeCancelRenewal,
		"UNKNOWN":                "",
	}
	for in, want := range cases {
		if got := NormalizeGoogleNotificationAction(in); got != want {
			t.Fatalf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestDetectSubscriptionChangeKind(t *testing.T) {
	if got := DetectSubscriptionChangeKind("", "com.app.basic.monthly"); got != ChangeInitial {
		t.Fatalf("initial: got %q", got)
	}
	if got := DetectSubscriptionChangeKind("com.app.basic.monthly", "com.app.basic.monthly"); got != ChangeRenewal {
		t.Fatalf("renewal: got %q", got)
	}
}
