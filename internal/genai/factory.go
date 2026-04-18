package genapi

import (
	"fmt"
)

// VideoGeneratorFactory constructs provider-specific adapters on demand.
type VideoGeneratorFactory struct{}

// NewVideoGeneratorFactory builds a factory instance.
func NewVideoGeneratorFactory() *VideoGeneratorFactory {
	return &VideoGeneratorFactory{}
}

// CreateClient instantiates a provider adapter configured for video generation.
func (f *VideoGeneratorFactory) CreateClient(providerName string, cfg *Config) (VideoGenerator, error) {
	normalized := ProviderKind(normalizeProviderName(providerName))
	switch normalized {
	case ProviderHailuo:
		adapter, err := newHailuoProvider(cfg)
		if err != nil {
			return nil, err
		}
		return adapter, nil
	case ProviderHuoshan:
		adapter, err := newHuoshanProvider(cfg)
		if err != nil {
			return nil, err
		}
		return adapter, nil
	case ProviderGemini:
		adapter, err := newGeminiProvider(cfg)
		if err != nil {
			return nil, err
		}
		return adapter, nil
	case ProviderQwen:
		adapter, err := newQwenProvider(cfg)
		if err != nil {
			return nil, err
		}
		return adapter, nil
	case ProviderKling:
		adapter, err := newKlingProvider(cfg)
		if err != nil {
			return nil, err
		}
		return adapter, nil
	default:
		return nil, fmt.Errorf("unsupported video provider %s", providerName)
	}
}

// NewProviderFromConfig constructs a provider adapter for the given configuration.
func NewProviderFromConfig(cfg *Config) (Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("provider config is required")
	}
	switch ProviderKind(normalizeProviderName(string(cfg.Provider))) {
	case ProviderHailuo:
		return newHailuoProvider(cfg)
	case ProviderHuoshan:
		return newHuoshanProvider(cfg)
	case ProviderGemini:
		return newGeminiProvider(cfg)
	case ProviderQwen:
		return newQwenProvider(cfg)
	case ProviderKling:
		return newKlingProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider %s", cfg.Provider)
	}
}

// RegisterProviderConfig constructs a provider from configuration and registers it on the GenAPI.
func (g *GenAPI) RegisterProviderConfig(cfg *Config) (Provider, error) {
	provider, err := NewProviderFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	g.RegisterProvider(provider)
	return provider, nil
}
