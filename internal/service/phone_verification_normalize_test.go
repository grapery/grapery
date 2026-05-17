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
