package huoshan

import (
	"net/http"
	"strings"
	"time"

	arkruntime "github.com/volcengine/volcengine-go-sdk/service/arkruntime"
)

const (
	defaultVideoModel = "doubao-seedance-1-0-pro-250528"
	defaultImageModel = "doubao-seedream-4-0-250828"
	defaultArkBaseURL = "https://ark.cn-beijing.volces.com"
	defaultArkAPIPath = "/api/v3"
)

// Config contains Huoshan (ArkRuntime) configuration.
type Config struct {
	APIKey       string
	BaseURL      string
	ImageBaseURL string
	Timeout      time.Duration
	Workflow     string
	ImageModel   string
}

// Client wraps ArkRuntime operations for image and video generation.
type Client struct {
	config     Config
	httpClient *http.Client
	arkClient  *arkruntime.Client
}

// New constructs a Huoshan client with sane defaults.
func New(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 300 * time.Second
	}

	httpClient := &http.Client{Timeout: config.Timeout}
	options := []arkruntime.ConfigOption{arkruntime.WithHTTPClient(httpClient)}
	if config.Timeout > 0 {
		options = append(options, arkruntime.WithTimeout(config.Timeout))
	}
	arkBaseURL := resolveArkBaseURL(config.ImageBaseURL)
	options = append(options, arkruntime.WithBaseUrl(arkBaseURL))

	client := arkruntime.NewClientWithApiKey(config.APIKey, options...)
	return &Client{
		config:     config,
		httpClient: httpClient,
		arkClient:  client,
	}
}

func resolveArkBaseURL(baseURL string) string {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = defaultArkBaseURL
	}
	base = strings.TrimSuffix(base, "/")
	if strings.HasSuffix(base, defaultArkAPIPath) {
		return base
	}
	if strings.Contains(base, "/api/") {
		return base
	}
	return base + defaultArkAPIPath
}

func choose(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func ptr[T any](v T) *T {
	return &v
}
