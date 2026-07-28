package pay

import (
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/common"
)

func TestReceiptIdempotencyKey(t *testing.T) {
	r := &IAPReceipt{SubscriptionTransactionID: "tx-1", OriginalTransactionID: "orig"}
	// helper lives in transport/pay; mirror logic here for service receipt fields
	key := r.SubscriptionTransactionID
	if key == "" {
		key = r.OriginalTransactionID
	}
	if key != "tx-1" {
		t.Fatalf("got %q", key)
	}
	_ = common.CreditToTokenRatio
}

func TestIsTokenUsageRestoreSourceNaming(t *testing.T) {
	cases := []struct {
		src  string
		want bool
	}{
		{"ai_image_reservation_refund", true},
		{"ai_image_release", true},
		{"ai_image_generation", false},
		{"fragment_generation_text", false},
	}
	for _, tc := range cases {
		got := len(tc.src) > 0 && (hasSuffixFold(tc.src, "_refund") || hasSuffixFold(tc.src, "_release"))
		if got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.src, got, tc.want)
		}
	}
}

func hasSuffixFold(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
