package pay

import (
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestRequiresPhoneVerificationSkipsWhenPhoneVerified(t *testing.T) {
	verified := time.Now().Unix()
	u := &domain.User{
		PendingOAuthPhoneSMS: true,
		PhoneVerifiedAt:      &verified,
	}
	if RequiresPhoneVerification("wechat", u) {
		t.Fatal("expected false when phone already verified")
	}
}

func TestRequiresPhoneVerificationTrueWhenPending(t *testing.T) {
	u := &domain.User{PendingOAuthPhoneSMS: true}
	if !RequiresPhoneVerification("wechat", u) {
		t.Fatal("expected true when SMS pending and phone not verified")
	}
}

func TestWechatPlaceholderEmailStable(t *testing.T) {
	a := wechatPlaceholderEmail("openid-abc")
	b := wechatPlaceholderEmail("openid-abc")
	if a != b {
		t.Fatalf("expected stable email, got %q vs %q", a, b)
	}
	if a == "" {
		t.Fatal("expected non-empty placeholder email")
	}
}
