package telemetry

import (
	"regexp"
	"strings"
)

var (
	uuidSegment = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	numericID   = regexp.MustCompile(`^\d+$`)
)

// NormalizeMetricPath collapses high-cardinality path segments (UUIDs, numeric IDs)
// so HTTP metrics stay bounded when a Gin route pattern is unavailable.
func NormalizeMetricPath(path string) string {
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if uuidSegment.MatchString(part) || numericID.MatchString(part) {
			parts[i] = ":id"
		}
	}
	normalized := strings.Join(parts, "/")
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	return normalized
}
