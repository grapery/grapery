package pay

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseGoogleNotification_subscriptionRenewed(t *testing.T) {
	inner := map[string]interface{}{
		"version":         "1.0",
		"packageName":     "com.example.app",
		"eventTimeMillis": "1710000000000",
		"subscriptionNotification": map[string]interface{}{
			"version":          "1.0",
			"notificationType": 2,
			"purchaseToken":    "token-xyz",
			"subscriptionId":   "premium_monthly",
		},
	}
	raw, _ := json.Marshal(inner)
	got, err := ParseGoogleNotification(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.NotificationType != "SUBSCRIPTION_RENEWED" {
		t.Fatalf("type=%q", got.NotificationType)
	}
	if got.SubscriptionNotification.PurchaseToken != "token-xyz" {
		t.Fatalf("token=%q", got.SubscriptionNotification.PurchaseToken)
	}
	if got.SubscriptionNotification.SubscriptionID != "premium_monthly" {
		t.Fatalf("sku=%q", got.SubscriptionNotification.SubscriptionID)
	}
}

func TestParseGoogleNotification_pubsubEnvelope(t *testing.T) {
	inner := `{"version":"1.0","packageName":"com.example","eventTimeMillis":"1","subscriptionNotification":{"notificationType":12,"purchaseToken":"pt","subscriptionId":"sku"}}`
	envelope, _ := json.Marshal(map[string]interface{}{
		"message": map[string]interface{}{
			"data": base64.StdEncoding.EncodeToString([]byte(inner)),
		},
	})
	got, err := ParseGoogleNotification(string(envelope))
	if err != nil {
		t.Fatal(err)
	}
	if got.NotificationType != "SUBSCRIPTION_REVOKED" {
		t.Fatalf("type=%q", got.NotificationType)
	}
}

func TestParseGoogleNotification_oneTimeCanceled(t *testing.T) {
	inner := map[string]interface{}{
		"version":         "1.0",
		"eventTimeMillis": "2",
		"oneTimeProductNotification": map[string]interface{}{
			"notificationType": 2,
			"purchaseToken":    "ot-token",
			"sku":              "credits_100",
		},
	}
	raw, _ := json.Marshal(inner)
	got, err := ParseGoogleNotification(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.NotificationType != "ONE_TIME_PRODUCT_CANCELED" {
		t.Fatalf("type=%q", got.NotificationType)
	}
	if got.OneTimeProductNotification.SKU != "credits_100" {
		t.Fatalf("sku=%q", got.OneTimeProductNotification.SKU)
	}
}
