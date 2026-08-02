package doubao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// VideoClient represents a client for the Doubao Seedance video generation service
type VideoClient struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

// NewVideoClient creates a new video generation client
func NewVideoClient(apiKey string) *VideoClient {
	if apiKey == "" {
		apiKey = os.Getenv("DOUBAO_API_KEY")
	}
	if apiKey == "" {
		panic("DOUBAO_API_KEY is not set")
	}
	return &VideoClient{
		APIKey:   apiKey,
		Endpoint: "https://ark.cn-beijing.volces.com",
		HTTPClient: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

// Video models
const (
	// Doubao Seedance Pro models
	ModelSeedanceProNew    = "doubao-seedance-pronew"
	ModelSeedancePro250528 = "doubao-seedance-1-0-pro-250528"

	// Doubao Seedance Lite models
	ModelSeedanceLiteT2V = "doubao-seedance-1-0-lite-t2v-250428"
	ModelSeedanceLiteI2V = "doubao-seedance-1-0-lite-i2v-250428"

	// Wan2.1 models
	ModelWan21T2V   = "wan2-1-14b-t2v-250417"
	ModelWan21I2V   = "wan2-1-14b-i2v-250417"
	ModelWan21FLF2V = "wan2-1-14b-flf2v-250417"
)

// Constants
const (
	Resolution720p      = "720p"
	Resolution1080p     = "1080p"
	Ratio16x9           = "16:9"
	Ratio9x16           = "9:16"
	RatioAdaptive       = "adaptive"
	StatusPending       = "pending"
	StatusSucceeded     = "succeeded"
	StatusFailed        = "failed"
	ContentTypeText     = "text"
	ContentTypeImageURL = "image_url"
	RoleFirstFrame      = "first_frame"
	RoleLastFrame       = "last_frame"
)

// VideoGenerationRequest represents the request to create a video generation task
type VideoGenerationRequest struct {
	Model       string        `json:"model"`
	Content     []ContentItem `json:"content"`
	CallbackURL string        `json:"callback_url,omitempty"`
}

// ContentItem represents a content item in the video generation request
type ContentItem struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
	Role     string    `json:"role,omitempty"`
}

// ImageURL represents an image URL in the content
type ImageURL struct {
	URL string `json:"url"`
}

// VideoGenerationResponse represents the response from creating a video generation task
type VideoGenerationResponse struct {
	ID    string    `json:"id"`
	Error *APIError `json:"error,omitempty"`
}

// VideoTask represents a video generation task
type VideoTask struct {
	ID              string       `json:"id"`
	Model           string       `json:"model"`
	Status          string       `json:"status"`
	Content         *TaskContent `json:"content,omitempty"`
	Seed            *int64       `json:"seed,omitempty"`
	Resolution      string       `json:"resolution,omitempty"`
	Duration        int          `json:"duration,omitempty"`
	Ratio           string       `json:"ratio,omitempty"`
	FramesPerSecond int          `json:"framespersecond,omitempty"`
	Usage           *TaskUsage   `json:"usage,omitempty"`
	CreatedAt       int64        `json:"created_at"`
	UpdatedAt       int64        `json:"updated_at"`
	Error           *APIError    `json:"error,omitempty"`
}

// TaskContent represents the content of a completed video task
type TaskContent struct {
	VideoURL string `json:"video_url"`
}

// TaskUsage represents the usage statistics for a task
type TaskUsage struct {
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// VideoTaskListResponse represents the response from listing video tasks
type VideoTaskListResponse struct {
	Total int         `json:"total"`
	Items []VideoTask `json:"items"`
	Error *APIError   `json:"error,omitempty"`
}

// TaskListOptions represents options for listing tasks
type TaskListOptions struct {
	PageSize int    `json:"page_size,omitempty"`
	Status   string `json:"status,omitempty"`
	PageNum  int    `json:"page_num,omitempty"`
}

// CreateVideoGenerationTask creates a new video generation task
func (c *VideoClient) CreateVideoGenerationTask(ctx context.Context, req *VideoGenerationRequest) (*VideoGenerationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint+"/api/v3/contents/generations/tasks", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result VideoGenerationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if result.Error != nil {
			return &result, result.Error
		}
		return &result, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return &result, nil
}

// GetVideoTask retrieves a specific video generation task
func (c *VideoClient) GetVideoTask(ctx context.Context, taskID string) (*VideoTask, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.Endpoint+"/api/v3/contents/generations/tasks/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result VideoTask
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if result.Error != nil {
			return &result, result.Error
		}
		return &result, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return &result, nil
}

// ListVideoTasks retrieves a list of video generation tasks
func (c *VideoClient) ListVideoTasks(ctx context.Context, options *TaskListOptions) (*VideoTaskListResponse, error) {
	u, err := url.Parse(c.Endpoint + "/api/v3/contents/generations/tasks")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	query := u.Query()
	if options != nil {
		if options.PageSize > 0 {
			query.Set("page_size", strconv.Itoa(options.PageSize))
		}
		if options.Status != "" {
			query.Set("filter.status", options.Status)
		}
		if options.PageNum > 0 {
			query.Set("page_num", strconv.Itoa(options.PageNum))
		}
	}
	u.RawQuery = query.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result VideoTaskListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if result.Error != nil {
			return &result, result.Error
		}
		return &result, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return &result, nil
}

// DeleteVideoTask cancels or deletes a video generation task
func (c *VideoClient) DeleteVideoTask(ctx context.Context, taskID string) error {
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", c.Endpoint+"/api/v3/contents/generations/tasks/"+taskID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Helper methods for creating requests

// NewTextToVideoRequest creates a text-to-video generation request
func NewTextToVideoRequest(model, prompt string) *VideoGenerationRequest {
	return &VideoGenerationRequest{
		Model: model,
		Content: []ContentItem{
			{
				Type: ContentTypeText,
				Text: prompt,
			},
		},
	}
}

// NewImageToVideoRequest creates an image-to-video generation request
func NewImageToVideoRequest(model, prompt, imageURL string) *VideoGenerationRequest {
	return &VideoGenerationRequest{
		Model: model,
		Content: []ContentItem{
			{
				Type: ContentTypeText,
				Text: prompt,
			},
			{
				Type: ContentTypeImageURL,
				ImageURL: &ImageURL{
					URL: imageURL,
				},
			},
		},
	}
}

// NewFirstLastFrameVideoRequest creates a first-last frame video generation request
func NewFirstLastFrameVideoRequest(model, prompt, firstFrameURL, lastFrameURL string) *VideoGenerationRequest {
	return &VideoGenerationRequest{
		Model: model,
		Content: []ContentItem{
			{
				Type: ContentTypeText,
				Text: prompt,
			},
			{
				Type: ContentTypeImageURL,
				ImageURL: &ImageURL{
					URL: firstFrameURL,
				},
				Role: RoleFirstFrame,
			},
			{
				Type: ContentTypeImageURL,
				ImageURL: &ImageURL{
					URL: lastFrameURL,
				},
				Role: RoleLastFrame,
			},
		},
	}
}

// WithCallbackURL adds a callback URL to the request
func (r *VideoGenerationRequest) WithCallbackURL(callbackURL string) *VideoGenerationRequest {
	r.CallbackURL = callbackURL
	return r
}

// IsCompleted checks if the task is completed (succeeded or failed)
func (t *VideoTask) IsCompleted() bool {
	return t.Status == StatusSucceeded || t.Status == StatusFailed
}

// IsSucceeded checks if the task succeeded
func (t *VideoTask) IsSucceeded() bool {
	return t.Status == StatusSucceeded
}

// GetVideoURL returns the video URL if the task is completed successfully
func (t *VideoTask) GetVideoURL() string {
	if t.Content != nil && t.IsSucceeded() {
		return t.Content.VideoURL
	}
	return ""
}
