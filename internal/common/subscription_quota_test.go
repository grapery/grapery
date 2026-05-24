package common

import "testing"

func TestNormalizeIAPProductQuotaLimit(t *testing.T) {
	t.Parallel()
	const wantBasic = 100 * CreditToTokenRatio
	const wantPremium = 200 * CreditToTokenRatio

	cases := []struct {
		productID string
		raw       int
		want      int
	}{
		{"com.grapery.pro.monthly", 200, wantBasic},
		{"com.grapery.pro.monthly", 100, wantBasic},
		{"com.grapery.pro.monthly", wantBasic, wantBasic},
		{"com.grapery.prime.monthly", 200, wantPremium},
		{"com.grapery.prime.yearly", wantPremium, wantPremium},
	}
	for _, tc := range cases {
		if got := NormalizeIAPProductQuotaLimit(tc.productID, tc.raw); got != tc.want {
			t.Fatalf("%s quota %d => got %d want %d", tc.productID, tc.raw, got, tc.want)
		}
	}
}

func TestMembershipTierFromIAPProductID(t *testing.T) {
	t.Parallel()
	if got := MembershipTierFromIAPProductID("com.grapery.pro.monthly"); got != "basic" {
		t.Fatalf("pro => %q", got)
	}
	if got := MembershipTierFromIAPProductID("com.grapery.prime.monthly"); got != "premium" {
		t.Fatalf("prime => %q", got)
	}
}
