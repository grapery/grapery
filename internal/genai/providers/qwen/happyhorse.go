package qwen

import (
	"strings"
)

// Alibaba Cloud Model Studio HappyHorse models (异步 video-synthesis).
// References: https://www.alibabacloud.com/help/en/model-studio/happyhorse-text-to-video-api-reference
// Image-to-video: https://www.alibabacloud.com/help/en/model-studio/happyhorse-image-to-video-api-reference
// Video edit (ZH): https://www.alibabacloud.com/help/zh/model-studio/happyhorse-video-edit-api-reference
const (
	ModelHappyHorseT2V       = "happyhorse-1.0-t2v"
	ModelHappyHorseI2V       = "happyhorse-1.0-i2v"
	ModelHappyHorseVideoEdit = "happyhorse-1.0-video-edit"
)

// IsHappyHorseModel reports whether the model name refers to HappyHorse video-synthesis APIs.
func IsHappyHorseModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(m, "happyhorse")
}

// HappyHorsePipeline classifies which HappyHorse sub-API applies for the configured model ID.
func HappyHorsePipeline(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(m, "video-edit"):
		return "video_edit"
	case strings.Contains(m, "i2v"):
		return "i2v"
	case strings.Contains(m, "t2v"):
		return "t2v"
	default:
		return "t2v"
	}
}

// NormalizeHappyHorseResolution maps common resolution strings to API values (720P / 1080P).
func NormalizeHappyHorseResolution(s string) string {
	v := strings.ToUpper(strings.TrimSpace(s))
	switch v {
	case "", "1080", "1080P", "FHD":
		return "1080P"
	case "720", "720P", "HD":
		return "720P"
	default:
		if strings.Contains(v, "720") {
			return "720P"
		}
		return "1080P"
	}
}

// ClampHappyHorseDuration restricts duration to the model-supported range [3, 15]; 0 yields defaultSecs.
func ClampHappyHorseDuration(secs int, defaultSecs int) int {
	if defaultSecs <= 0 {
		defaultSecs = 5
	}
	if secs <= 0 {
		return defaultSecs
	}
	if secs < 3 {
		return 3
	}
	if secs > 15 {
		return 15
	}
	return secs
}

// ResolveHappyHorseModelID fills in a full model id when the caller passes a generic placeholder.
func ResolveHappyHorseModelID(model string, pipeline string) string {
	m := strings.TrimSpace(model)
	if m != "" && !strings.EqualFold(m, "happyhorse") {
		return m
	}
	switch pipeline {
	case "i2v":
		return ModelHappyHorseI2V
	case "video_edit", "edit":
		return ModelHappyHorseVideoEdit
	default:
		return ModelHappyHorseT2V
	}
}

// HappyHorseT2VPayload is the request body for happyhorse-1.0-t2v.
type HappyHorseT2VPayload struct {
	Model      string               `json:"model"`
	Input      HappyHorseT2VInput   `json:"input"`
	Parameters *HappyHorseT2VParams `json:"parameters,omitempty"`
}

type HappyHorseT2VInput struct {
	Prompt string `json:"prompt"`
}

type HappyHorseT2VParams struct {
	Resolution string `json:"resolution,omitempty"`
	Ratio      string `json:"ratio,omitempty"`
	Duration   int    `json:"duration,omitempty"`
	Watermark  *bool  `json:"watermark,omitempty"`
	Seed       *int   `json:"seed,omitempty"`
}

// HappyHorseI2VPayload is the request body for happyhorse-1.0-i2v (first-frame + optional prompt).
type HappyHorseI2VPayload struct {
	Model      string               `json:"model"`
	Input      HappyHorseI2VInput   `json:"input"`
	Parameters *HappyHorseI2VParams `json:"parameters,omitempty"`
}

type HappyHorseI2VMedia struct {
	Type string `json:"type"` // first_frame
	URL  string `json:"url"`
}

type HappyHorseI2VInput struct {
	Prompt string               `json:"prompt,omitempty"`
	Media  []HappyHorseI2VMedia `json:"media"`
}

type HappyHorseI2VParams struct {
	Resolution string `json:"resolution,omitempty"`
	Duration   int    `json:"duration,omitempty"`
	Watermark  *bool  `json:"watermark,omitempty"`
	Seed       *int   `json:"seed,omitempty"`
}

// HappyHorseVideoEditPayload is the request body for happyhorse-1.0-video-edit.
type HappyHorseVideoEditPayload struct {
	Model      string                     `json:"model"`
	Input      HappyHorseVideoEditInput   `json:"input"`
	Parameters *HappyHorseVideoEditParams `json:"parameters,omitempty"`
}

type HappyHorseVideoEditMedia struct {
	Type string `json:"type"` // video | reference_image
	URL  string `json:"url"`
}

type HappyHorseVideoEditInput struct {
	Prompt string                     `json:"prompt"`
	Media  []HappyHorseVideoEditMedia `json:"media"`
}

type HappyHorseVideoEditParams struct {
	Resolution string `json:"resolution,omitempty"`
	Watermark  *bool  `json:"watermark,omitempty"`
	Seed       *int   `json:"seed,omitempty"`
}
