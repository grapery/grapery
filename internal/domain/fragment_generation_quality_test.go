package domain

import "testing"

func TestFragmentPublicationQualityUsesExplicitServerEvidenceAndAllowsLegacy(t *testing.T) {
	for _, raw := range []string{"", `{}`, `{"comicDocument":{"schemaVersion":2,"pages":[]}}`, `{"quality":{"status":"passed"}}`} {
		if err := ValidateFragmentPublicationQuality(raw); err != nil {
			t.Fatalf("compatible document rejected: %s %v", raw, err)
		}
	}
	for _, raw := range []string{`bad json`, `{"quality":{"status":"unreviewed"}}`, `{"quality":{"status":"blocked"}}`, `{"quality":{"status":"passed"},"comicDocument":{"pages":[{"quality":{"status":"blocked"}}]}}`} {
		if err := ValidateFragmentPublicationQuality(raw); err == nil {
			t.Fatalf("unsafe quality accepted: %s", raw)
		}
	}
}

func TestFragmentPublicationRequiresSettledNewBillingAndAllowsLegacy(t *testing.T) {
	for _, raw := range []string{`{"billingStatus":"pending","quality":{"status":"passed"}}`, `{"billingStatus":"unknown"}`} {
		if err := ValidateFragmentPublicationQuality(raw); err == nil {
			t.Fatalf("unsettled fragment published: %s", raw)
		}
	}
	if err := ValidateFragmentPublicationQuality(`{"billingStatus":"settled","quality":{"status":"passed"}}`); err != nil {
		t.Fatal(err)
	}
}
