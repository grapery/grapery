package gemini

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/genai"
)

const defaultHTTPTimeout = 120 * time.Second

// Config holds Gemini API configuration.
type Config struct {
	APIKey       string
	BaseURL      string
	Timeout      time.Duration
	DefaultModel string
	Backend      genai.Backend
	Project      string
	Location     string
}

// Client wraps the official Google GenAI SDK client.
type Client struct {
	config     Config
	sdk        *genai.Client
	httpClient *http.Client
}

// New instantiates a client backed by the google.golang.org/genai SDK.
func New(config Config) (*Client, error) {
	if config.Timeout <= 0 {
		config.Timeout = defaultHTTPTimeout
	}

	httpClient := &http.Client{Timeout: config.Timeout}
	clientConfig := &genai.ClientConfig{
		APIKey:     strings.TrimSpace(config.APIKey),
		Backend:    config.Backend,
		Project:    strings.TrimSpace(config.Project),
		Location:   strings.TrimSpace(config.Location),
		HTTPClient: httpClient,
	}

	if base := strings.TrimSpace(config.BaseURL); base != "" || config.Timeout > 0 {
		clientConfig.HTTPOptions = genai.HTTPOptions{}
		if base != "" {
			clientConfig.HTTPOptions.BaseURL = strings.TrimRight(base, "/")
		}
		if config.Timeout > 0 {
			timeout := config.Timeout
			clientConfig.HTTPOptions.Timeout = &timeout
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	sdk, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	return &Client{
		config:     config,
		sdk:        sdk,
		httpClient: httpClient,
	}, nil
}

// SDK exposes the underlying google.golang.org/genai client.
func (c *Client) SDK() *genai.Client {
	return c.sdk
}

func choose(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
