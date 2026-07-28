package mysql

import "testing"

func TestIsTokenUsageRestoreSource(t *testing.T) {
	cases := []struct {
		src  string
		want bool
	}{
		{"ai_image_reservation_refund", true},
		{"story_ai_release", true},
		{"AI_IMAGE_REFUND", true},
		{"ai_image_generation", false},
		{"fragment_generation_text", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isTokenUsageRestoreSource(tc.src); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.src, got, tc.want)
		}
	}
}
