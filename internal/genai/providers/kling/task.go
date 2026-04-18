package kling

import (
	"fmt"
	"strings"
)

// FormatTaskID builds a composite ID for routing status queries.
func FormatTaskID(kind, providerTaskID string) string {
	providerTaskID = strings.TrimSpace(providerTaskID)
	kind = strings.TrimSpace(kind)
	if providerTaskID == "" || kind == "" {
		return ""
	}
	return taskIDPrefix + kind + ":" + providerTaskID
}

// ParseTaskID splits a composite task ID into kind and provider task id.
func ParseTaskID(composite string) (kind string, rawID string, err error) {
	composite = strings.TrimSpace(composite)
	parts := strings.SplitN(composite, ":", 3)
	if len(parts) == 3 && parts[0] == "kling" {
		return parts[1], parts[2], nil
	}
	return "", "", fmt.Errorf("kling: invalid composite task id (expected kling:<kind>:<id>)")
}

// QueryPath returns the GET path for a task kind and raw provider task id.
func QueryPath(kind, rawID string) (string, error) {
	rawID = strings.TrimSpace(rawID)
	if rawID == "" {
		return "", fmt.Errorf("kling: empty task id")
	}
	switch kind {
	case TaskText2Video:
		return "/v1/videos/text2video/" + rawID, nil
	case TaskImage2Video:
		return "/v1/videos/image2video/" + rawID, nil
	case TaskMultiImage2Video:
		return "/v1/videos/multi-image2video/" + rawID, nil
	case TaskVideoExtend:
		return "/v1/videos/video-extend/" + rawID, nil
	case TaskOmniVideo:
		return "/v1/videos/omni-video/" + rawID, nil
	case TaskImageGen:
		return "/v1/images/generations/" + rawID, nil
	case TaskImageExpand:
		return "/v1/images/editing/expand/" + rawID, nil
	case TaskOmniImage:
		return "/v1/images/omni-image/" + rawID, nil
	case TaskMultiImage2Image:
		return "/v1/images/multi-image2image/" + rawID, nil
	default:
		return "", fmt.Errorf("kling: unknown task kind %q", kind)
	}
}
