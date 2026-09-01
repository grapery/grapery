package service

import "testing"

func TestNormalizeChinaPhone(t *testing.T) {
	tests := []struct {
		raw    string
		want   string
		wantOk bool
	}{
		{"18589045535", "18589045535", true},
		{"+8618589045535", "18589045535", true},
		{" +86 185 8904 5535 ", "18589045535", true},
		{"13800138000", "13800138000", true},
		{"12100000000", "", false},
		{"1858904553", "", false},
		{"185890455351", "", false},
		{"", "", false},
		{"abcd", "", false},
	}
	for _, tt := range tests {
		got, err := NormalizeChinaPhone(tt.raw)
		if tt.wantOk {
			if err != nil {
				t.Errorf("NormalizeChinaPhone(%q): unexpected err %v", tt.raw, err)
				continue
			}
			if got != tt.want {
				t.Errorf("NormalizeChinaPhone(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		} else {
			if err == nil {
				t.Errorf("NormalizeChinaPhone(%q): want error, got %q", tt.raw, got)
			}
		}
	}
}

func TestDebugPhoneLoginBypassCodeRequiresExplicitDevelopmentConfiguration(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("GRAPERY_DEBUG_PHONE_LOGIN_BYPASS", "1")
	t.Setenv("GRAPERY_DEBUG_PHONE_LOGIN_CODE", "000000")
	if got := debugPhoneLoginBypassCode(); got != "000000" {
		t.Fatalf("debugPhoneLoginBypassCode() = %q, want configured code", got)
	}

	t.Setenv("GIN_MODE", "release")
	if got := debugPhoneLoginBypassCode(); got != "" {
		t.Fatalf("release mode must disable bypass, got %q", got)
	}
}

func TestDebugPhoneLoginBypassCodeRejectsIncompleteConfiguration(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("GRAPERY_DEBUG_PHONE_LOGIN_BYPASS", "")
	t.Setenv("GRAPERY_DEBUG_PHONE_LOGIN_CODE", "000000")
	if got := debugPhoneLoginBypassCode(); got != "" {
		t.Fatalf("missing opt-in flag must disable bypass, got %q", got)
	}

	t.Setenv("GRAPERY_DEBUG_PHONE_LOGIN_BYPASS", "1")
	t.Setenv("GRAPERY_DEBUG_PHONE_LOGIN_CODE", "123")
	if got := debugPhoneLoginBypassCode(); got != "" {
		t.Fatalf("invalid code must disable bypass, got %q", got)
	}
}
