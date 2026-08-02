package hailuo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	videoGenerationEndpoint        = "/v1/video_generation"
	videoStatusQueryEndpoint       = "/v1/query/video_generation"
	fileRetrieveEndpoint           = "/v1/files/retrieve"
	storyboardToVideoEndpoint      = "/api/v1/storyboard_to_video"
	musicGenerationEndpoint        = "/v1/music_generation"
	imageGenerationEndpoint        = "/v1/image_generation"
	downloadEndpointTemplate       = "/api/v1/video_generation/%s/download"
	defaultVideoGenerationDuration = 6
)

// Config configures Minimax Hailuo API calls.
type Config struct {
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	Model      string
	MusicModel string
	ImageModel string
}

// Client provides access to atomic Hailuo endpoints.
type Client struct {
	config     Config
	httpClient *http.Client
}

// New instantiates a Hailuo client.
func New(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: config.Timeout},
	}
}

type APIBaseResponse struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type TaskResponse struct {
	TaskID       string                 `json:"task_id"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	BaseResponse APIBaseResponse        `json:"base_resp"`
}

func (c *Client) post(ctx context.Context, apiPath string, payload interface{}) (*TaskResponse, error) {
	resp := &TaskResponse{}
	if err := c.doRequest(ctx, http.MethodPost, apiPath, payload, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) doRequest(ctx context.Context, method, apiPath string, payload interface{}, out interface{}) error {
	endpoint := c.endpoint(apiPath)
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal hailuo payload: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hailuo api %s failed with status %d: %s", apiPath, resp.StatusCode, string(bodyBytes))
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) endpoint(apiPath string) string {
	base := strings.TrimSuffix(c.config.BaseURL, "/")
	if base == "" {
		base = "https://api.minimax.com"
	}
	if strings.HasPrefix(apiPath, "http") {
		return apiPath
	}
	return base + path.Clean("/"+apiPath)
}

func choose(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
