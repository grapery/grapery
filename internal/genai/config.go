package genapi

import "time"

// ProviderKind enumerates the supported provider identifiers.
type ProviderKind string

const (
	ProviderHailuo  ProviderKind = "hailuo"
	ProviderHuoshan ProviderKind = "huoshan"
	ProviderGemini  ProviderKind = "gemini"
	ProviderQwen    ProviderKind = "qwen"
)

// Config encapsulates the minimal credentials and preferences required to bootstrap a provider adapter.
type Config struct {
	Provider     ProviderKind
	APIKey       string
	Secret       string
	BaseURL      string
	ImageBaseURL string
	Timeout      time.Duration
	Model        string
	ImageModel   string
	Workflow     string
	Additional   map[string]interface{}
}

// Clone returns a shallow copy of the config.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	cp := *c
	if c.Additional != nil {
		cp.Additional = cloneMap(c.Additional)
	}
	return &cp
}
