package huoshan

import (
	"net/http"
	"strings"
	"time"

	arkruntime "github.com/volcengine/volcengine-go-sdk/service/arkruntime"
)

const (
	defaultVideoModel       = "doubao-seedance-1-5-pro-251215"
	defaultImageModelLevel2 = "doubao-seedream-5-0-260128"
	defaultImageModelLevel1 = "doubao-seedream-4-5-251128"
	defaultImageModel       = "doubao-seedream-4-0-250828"
	// DefaultHuoshanImageModelID is used when config and request do not specify an image model (Seedream 5.0).
	DefaultHuoshanImageModelID = defaultImageModelLevel2
	// defaultTextModel: 火山方舟 Chat API 使用 Endpoint ID（如 ep-20241104104259-xxxx），
	// 非模型名。需在控制台创建推理接入点后配置 TextModel/Endpoint ID。
	defaultTextModel  = "doubao-seed-2-0-lite-260215"
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
	VideoModel   string
	TextModel    string
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
	// ImageBaseURL 优先；为空时回退到 BaseURL，便于统一配置
	arkBaseURL := resolveArkBaseURL(choose(config.ImageBaseURL, config.BaseURL))
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
