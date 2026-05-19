package genapi

import (
	"fmt"
	"strconv"
	"strings"
)

func cloneMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		switch typed := v.(type) {
		case map[string]interface{}:
			dst[k] = cloneMap(typed)
		case []string:
			dst[k] = append([]string(nil), typed...)
		case []interface{}:
			cloned := make([]interface{}, len(typed))
			copy(cloned, typed)
			dst[k] = cloned
		default:
			dst[k] = typed
		}
	}
	return dst
}

func mergeMaps(values ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range values {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func normalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// NormalizeStatus converts provider-specific status strings to a standard TaskStatus.
// This ensures consistent status representation across all providers.
func NormalizeStatus(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "pending", "queued", "waiting", "submitted":
		return string(StatusPending)
	case "processing", "running", "in_progress", "inprogress", "started", "working":
		return string(StatusProcessing)
	case "completed", "succeeded", "success", "succeed", "done", "finished", "ready":
		return string(StatusCompleted)
	case "failed", "error", "failure", "errored":
		return string(StatusFailed)
	case "cancelled", "canceled", "aborted", "stopped":
		return string(StatusCancelled)
	default:
		// Return original if no match, preserving provider-specific values
		if raw == "" {
			return string(StatusUnknown)
		}
		return raw
	}
}

// IsTerminalStatus returns true if the status represents a final state (completed, failed, or cancelled).
func IsTerminalStatus(status string) bool {
	normalized := NormalizeStatus(status)
	return normalized == string(StatusCompleted) ||
		normalized == string(StatusFailed) ||
		normalized == string(StatusCancelled)
}

// IsPendingOrProcessing returns true if the task is still in progress.
func IsPendingOrProcessing(status string) bool {
	normalized := NormalizeStatus(status)
	return normalized == string(StatusPending) ||
		normalized == string(StatusProcessing)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func intFromAny(value interface{}) (int, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		if float32(int(v)) != v {
			return 0, false
		}
		return int(v), true
	case float64:
		if float64(int(v)) != v {
			return 0, false
		}
		return int(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case fmt.Stringer:
		parsed, err := strconv.Atoi(strings.TrimSpace(v.String()))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func boolFromOptions(options map[string]interface{}, keys ...string) (bool, bool) {
	if len(options) == 0 || len(keys) == 0 {
		return false, false
	}
	for _, key := range keys {
		value, ok := options[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v, true
		case *bool:
			if v == nil {
				return false, false
			}
			return *v, true
		case string:
			trimmed := strings.ToLower(strings.TrimSpace(v))
			if trimmed == "true" || trimmed == "1" {
				return true, true
			}
			if trimmed == "false" || trimmed == "0" {
				return false, true
			}
		default:
			if parsed, ok := intFromAny(v); ok {
				return parsed != 0, true
			}
		}
	}
	return false, false
}

func collectImages(primary string, extras []string, limit int) []string {
	if limit <= 0 {
		limit = len(extras) + 1
	}
	var result []string
	seen := make(map[string]struct{})
	add := func(raw string) {
		if len(result) >= limit {
			return
		}
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	add(primary)
	for _, img := range extras {
		add(img)
	}
	return result
}

func intFromOptions(options map[string]interface{}, keys ...string) int {
	if len(options) == 0 || len(keys) == 0 {
		return 0
	}
	for _, key := range keys {
		val, ok := options[key]
		if !ok {
			continue
		}
		if parsed, ok := intFromAny(val); ok {
			return parsed
		}
	}
	return 0
}

func stringFromOptions(options map[string]interface{}, keys ...string) string {
	if len(options) == 0 {
		return ""
	}
	for _, key := range keys {
		val, ok := options[key]
		if !ok {
			continue
		}
		switch v := val.(type) {
		case string:
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		case fmt.Stringer:
			if trimmed := strings.TrimSpace(v.String()); trimmed != "" {
				return trimmed
			}
		default:
			if parsed, ok := intFromAny(v); ok {
				return strconv.Itoa(parsed)
			}
		}
	}
	return ""
}
