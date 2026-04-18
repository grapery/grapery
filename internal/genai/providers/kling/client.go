package kling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config configures the Kling HTTP client.
type Config struct {
	AccessKey string
	SecretKey string
	BaseURL   string
	Timeout   time.Duration
}

// Client calls Kling Open API (api-singapore.klingai.com).
type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       *tokenAuth
}

// New creates a Kling API client.
func New(cfg Config) (*Client, error) {
	auth, err := newTokenAuth(strings.TrimSpace(cfg.AccessKey), strings.TrimSpace(cfg.SecretKey))
	if err != nil {
		return nil, err
	}
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}
	base = strings.TrimRight(base, "/")

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	return &Client{
		baseURL: base,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		auth: auth,
	}, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any) (*APIEnvelope, error) {
	token, err := c.auth.bearerToken()
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var env APIEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("kling http %d: %s", resp.StatusCode, string(raw))
		}
		return nil, fmt.Errorf("kling decode envelope: %w body=%s", err, truncateForErr(raw))
	}

	if env.Code != 0 {
		return nil, fmt.Errorf("kling api code=%d message=%s request_id=%s", env.Code, env.Message, env.RequestID)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("kling http %d: %s", resp.StatusCode, string(raw))
	}

	return &env, nil
}

func truncateForErr(b []byte) string {
	const max = 512
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

func decodeData[T any](env *APIEnvelope) (*T, error) {
	if env == nil || len(env.Data) == 0 {
		return nil, fmt.Errorf("kling empty response data")
	}
	var out T
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return nil, fmt.Errorf("kling decode data: %w", err)
	}
	return &out, nil
}

// --- Video ---

// CreateText2Video POST /v1/videos/text2video
func (c *Client) CreateText2Video(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/videos/text2video", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// CreateImage2Video POST /v1/videos/image2video
func (c *Client) CreateImage2Video(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/videos/image2video", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// CreateMultiImage2Video POST /v1/videos/multi-image2video
func (c *Client) CreateMultiImage2Video(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/videos/multi-image2video", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// CreateVideoExtend POST /v1/videos/video-extend
func (c *Client) CreateVideoExtend(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/videos/video-extend", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// CreateOmniVideo POST /v1/videos/omni-video
func (c *Client) CreateOmniVideo(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/videos/omni-video", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// --- Image ---

// CreateImageGeneration POST /v1/images/generations
func (c *Client) CreateImageGeneration(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/images/generations", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// CreateImageExpand POST /v1/images/editing/expand
func (c *Client) CreateImageExpand(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/images/editing/expand", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// CreateOmniImage POST /v1/images/omni-image
func (c *Client) CreateOmniImage(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/images/omni-image", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// CreateMultiImage2Image POST /v1/images/multi-image2image
func (c *Client) CreateMultiImage2Image(ctx context.Context, payload map[string]interface{}) (*CreateTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodPost, "/v1/images/multi-image2image", payload)
	if err != nil {
		return nil, err
	}
	return decodeData[CreateTaskData](env)
}

// QueryTask GET path must include full resource path + task id.
func (c *Client) QueryTask(ctx context.Context, path string) (*QueryTaskData, error) {
	env, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[QueryTaskData](env)
}

// DownloadURL fetches binary from a completed generation URL (e.g. video mp4).
func (c *Client) DownloadURL(ctx context.Context, rawURL string) ([]byte, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("kling: empty download url")
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return nil, fmt.Errorf("kling: download URL must be a valid https URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("kling download http %d: %s", resp.StatusCode, string(slurp))
	}
	return io.ReadAll(resp.Body)
}
