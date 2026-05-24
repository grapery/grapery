package pay

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseAppleNotification(t *testing.T) {
	t.Parallel()
	payload := map[string]interface{}{
		"notificationType": "DID_RENEW",
		"subtype":          "BILLING_RECOVERY",
		"notificationUUID": "abc-123",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	jws := "aaa." + base64.RawURLEncoding.EncodeToString(raw) + ".bbb"

	got, err := ParseAppleNotification(jws)
	if err != nil {
		t.Fatalf("ParseAppleNotification: %v", err)
	}
	if got.NotificationType != "DID_RENEW" {
		t.Fatalf("type=%q", got.NotificationType)
	}
	if got.Subtype != "BILLING_RECOVERY" {
		t.Fatalf("subtype=%q", got.Subtype)
	}
	if got.NotificationUUID != "abc-123" {
		t.Fatalf("uuid=%q", got.NotificationUUID)
	}
}

func TestParseAppleNotificationRejectsInvalid(t *testing.T) {
	t.Parallel()
	if _, err := ParseAppleNotification("not-a-jws"); err == nil {
		t.Fatal("expected error for invalid jws")
	}
}
