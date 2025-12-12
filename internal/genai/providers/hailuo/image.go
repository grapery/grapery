package hailuo

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type ImageGenerationRequest struct {
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	AspectRatio     string `json:"aspect_ratio,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
	ResponseFormat  string `json:"response_format,omitempty"`
	Seed            int    `json:"seed,omitempty"`
	N               int    `json:"n,omitempty"`
	PromptOptimizer *bool  `json:"prompt_optimizer,omitempty"`
}

type ImageGenerationResponse struct {
	ID           string                   `json:"id"`
	Data         *ImageGenerationData     `json:"data"`
	Metadata     *ImageGenerationMetadata `json:"metadata"`
	BaseResponse APIBaseResponse          `json:"base_resp"`
}

type ImageGenerationData struct {
	ImageURLs   []string `json:"image_urls"`
	ImageBase64 []string `json:"image_base64"`
}

type ImageGenerationMetadata struct {
	FailedCount  string `json:"failed_count"`
	SuccessCount string `json:"success_count"`
}

func (c *Client) GenerateImage(ctx context.Context, payload *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	payload.Prompt = strings.TrimSpace(payload.Prompt)
	if payload.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	payload.Model = strings.TrimSpace(payload.Model)
	if payload.Model == "" {
		payload.Model = strings.TrimSpace(choose(c.config.ImageModel, "image-01"))
		if payload.Model == "" {
			return nil, fmt.Errorf("model is required")
		}
	}

	payload.AspectRatio = strings.TrimSpace(payload.AspectRatio)
	payload.ResponseFormat = strings.TrimSpace(payload.ResponseFormat)
	if payload.ResponseFormat == "" {
		payload.ResponseFormat = "url"
	}

	if payload.N < 0 {
		return nil, fmt.Errorf("n must be non-negative")
	}
	if payload.N == 0 {
		payload.N = 1
	}

	if (payload.Width > 0 && payload.Height <= 0) || (payload.Height > 0 && payload.Width <= 0) {
		return nil, fmt.Errorf("width and height must be provided together")
	}

	resp := &ImageGenerationResponse{}
	if err := c.doRequest(ctx, http.MethodPost, imageGenerationEndpoint, payload, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
