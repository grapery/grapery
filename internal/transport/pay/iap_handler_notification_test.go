package pay

import (
	"encoding/json"
	"testing"
)

func TestHandleAppleNotificationRequest_payload(t *testing.T) {
	t.Run("camelCase signedPayload", func(t *testing.T) {
		var req HandleAppleNotificationRequest
		if err := json.Unmarshal([]byte(`{"signedPayload":"abc.def.ghi"}`), &req); err != nil {
			t.Fatal(err)
		}
		if got := req.payload(); got != "abc.def.ghi" {
			t.Fatalf("payload() = %q, want abc.def.ghi", got)
		}
	})

	t.Run("snake_case signed_payload fallback", func(t *testing.T) {
		var req HandleAppleNotificationRequest
		if err := json.Unmarshal([]byte(`{"signed_payload":"x.y.z"}`), &req); err != nil {
			t.Fatal(err)
		}
		if got := req.payload(); got != "x.y.z" {
			t.Fatalf("payload() = %q, want x.y.z", got)
		}
	})

	t.Run("prefers camelCase when both present", func(t *testing.T) {
		var req HandleAppleNotificationRequest
		if err := json.Unmarshal([]byte(`{"signedPayload":"first","signed_payload":"second"}`), &req); err != nil {
			t.Fatal(err)
		}
		if got := req.payload(); got != "first" {
			t.Fatalf("payload() = %q, want first", got)
		}
	})
}
