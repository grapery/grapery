package telemetry

import "testing"

func TestNormalizeMetricPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/api/stories/123/scenes", "/api/stories/:id/scenes"},
		{"/api/stories/550e8400-e29b-41d4-a716-446655440000", "/api/stories/:id"},
		{"api/users/42", "/api/users/:id"},
		{"", "/"},
	}
	for _, tt := range tests {
		if got := NormalizeMetricPath(tt.in); got != tt.want {
			t.Errorf("NormalizeMetricPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
