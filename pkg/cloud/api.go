package cloud

import (
	"context"
	"fmt"
	"io"
	"time"
)

type AgentType string

const (
	AgentTypeAzure   AgentType = "azure"
	AgentTypeLocal   AgentType = "local"
	AgentTypeTencent AgentType = "tencent"
	AgentTypeOpenAI  AgentType = "openai"
	AgentTypeGroq    AgentType = "groq"
	AgentTypeZhipu   AgentType = "zhipu"
)

// CloudProvider represents the cloud service provider
type CloudProvider string

const (
	ProviderGoogle CloudProvider = "google"
	ProviderDoubao CloudProvider = "doubao"
	ProviderCoze   CloudProvider = "coze"
)

// MediaType represents the type of media to generate
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
	MediaTypeText  MediaType = "text"
)

// UnifiedMediaRequest represents a unified media generation request
type UnifiedMediaRequest struct {
	Prompt    string                 `json:"prompt"`
	Provider  CloudProvider          `json:"provider"`
	MediaType MediaType              `json:"media_type"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// UnifiedMediaResponse represents a unified media generation response
type UnifiedMediaResponse struct {
	Success   bool                   `json:"success"`
	Data      []byte                 `json:"data,omitempty"`
	URL       string                 `json:"url,omitempty"`
	MimeType  string                 `json:"mime_type,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     error                  `json:"error,omitempty"`
	Provider  CloudProvider          `json:"provider"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ProviderInfo contains information about a cloud provider
type ProviderInfo struct {
	Name         CloudProvider          `json:"name"`
	DisplayName  string                 `json:"display_name"`
	Capabilities []MediaType            `json:"capabilities"`
	Models       map[string]ModelInfo   `json:"models"`
	Features     map[string]interface{} `json:"features"`
	Status       string                 `json:"status"`
}

// ModelInfo contains information about a specific model
type ModelInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         MediaType              `json:"type"`
	Capabilities []string               `json:"capabilities"`
	Parameters   map[string]interface{} `json:"parameters"`
	Limits       map[string]interface{} `json:"limits"`
}

// CloudService defines the interface for cloud services
type CloudService interface {
	// GenerateMedia generates media content
	GenerateMedia(ctx context.Context, req *UnifiedMediaRequest) (*UnifiedMediaResponse, error)

	// GetProviderInfo returns provider information
	GetProviderInfo() ProviderInfo

	// HealthCheck checks if the service is healthy
	HealthCheck(ctx context.Context) error

	// GetCapabilities returns supported media types
	GetCapabilities() []MediaType

	// GetModels returns available models
	GetModels() []ModelInfo
}

// GenerateRequest represents a generation request with provider selection
type GenerateRequest struct {
	Prompt    string                 `json:"prompt"`
	MediaType MediaType              `json:"media_type"`
	Provider  CloudProvider          `json:"provider,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
	Model     string                 `json:"model,omitempty"`
}

// CloudConfig contains configuration for cloud services
type CloudConfig struct {
	Google *GoogleConfig `json:"google,omitempty"`
	Doubao *DoubaoConfig `json:"doubao,omitempty"`
	Coze   *CozeConfig   `json:"coze,omitempty"`

	// Default provider to use when none is specified
	DefaultProvider CloudProvider `json:"default_provider"`

	// Routing rules for automatic provider selection
	RoutingRules []RoutingRule `json:"routing_rules,omitempty"`

	// Global settings
	Timeout time.Duration `json:"timeout"`
	Retry   int           `json:"retry"`
}

// GoogleConfig contains Google-specific configuration
type GoogleConfig struct {
	APIKey  string            `json:"api_key"`
	Enabled bool              `json:"enabled"`
	Models  map[string]string `json:"models,omitempty"`
	Region  string            `json:"region,omitempty"`
}

// DoubaoConfig contains Doubao-specific configuration
type DoubaoConfig struct {
	APIKey  string            `json:"api_key"`
	Enabled bool              `json:"enabled"`
	Models  map[string]string `json:"models,omitempty"`
	Region  string            `json:"region,omitempty"`
}

// CozeConfig contains Coze-specific configuration
type CozeConfig struct {
	APIKey  string            `json:"api_key"`
	Enabled bool              `json:"enabled"`
	Models  map[string]string `json:"models,omitempty"`
	Region  string            `json:"region,omitempty"`
}

// RoutingRule defines rules for automatic provider selection
type RoutingRule struct {
	Name      string        `json:"name"`
	MediaType MediaType     `json:"media_type"`
	Provider  CloudProvider `json:"provider"`
	Condition string        `json:"condition,omitempty"`
	Priority  int           `json:"priority"`
}

// UnifiedCloudManager manages multiple cloud services
type UnifiedCloudManager struct {
	services map[CloudProvider]CloudService
	config   *CloudConfig
	ctx      context.Context
}

// NewUnifiedCloudManager creates a new unified cloud manager
func NewUnifiedCloudManager(ctx context.Context, config *CloudConfig) (*UnifiedCloudManager, error) {
	manager := &UnifiedCloudManager{
		services: make(map[CloudProvider]CloudService),
		config:   config,
		ctx:      ctx,
	}

	// Initialize services based on configuration
	if config.Google != nil && config.Google.Enabled {
		if service, err := NewGoogleService(ctx, config.Google); err == nil {
			manager.services[ProviderGoogle] = service
		}
	}

	if config.Doubao != nil && config.Doubao.Enabled {
		if service, err := NewDoubaoService(ctx, config.Doubao); err == nil {
			manager.services[ProviderDoubao] = service
		}
	}

	if config.Coze != nil && config.Coze.Enabled {
		if service, err := NewCozeService(ctx, config.Coze); err == nil {
			manager.services[ProviderCoze] = service
		}
	}

	return manager, nil
}

// GenerateMedia generates media using the specified or automatically selected provider
func (m *UnifiedCloudManager) GenerateMedia(ctx context.Context, req *GenerateRequest) (*UnifiedMediaResponse, error) {
	provider := req.Provider
	if provider == "" {
		provider = m.selectProvider(req.MediaType)
	}

	service, exists := m.services[provider]
	if !exists {
		return &UnifiedMediaResponse{
			Success:   false,
			Error:     fmt.Errorf("provider %s not available", provider),
			Provider:  provider,
			Timestamp: time.Now(),
		}, fmt.Errorf("provider %s not available", provider)
	}

	unifiedReq := &UnifiedMediaRequest{
		Prompt:    req.Prompt,
		Provider:  provider,
		MediaType: req.MediaType,
		Options:   req.Options,
	}

	return service.GenerateMedia(ctx, unifiedReq)
}

// selectProvider selects the best provider based on routing rules
func (m *UnifiedCloudManager) selectProvider(mediaType MediaType) CloudProvider {
	// First check routing rules
	for _, rule := range m.config.RoutingRules {
		if rule.MediaType == mediaType {
			return rule.Provider
		}
	}

	// Fall back to default provider
	if m.config.DefaultProvider != "" {
		if _, exists := m.services[m.config.DefaultProvider]; exists {
			return m.config.DefaultProvider
		}
	}

	// Fall back to first available provider
	for provider := range m.services {
		return provider
	}

	return ""
}

// GetProviderInfo returns information about all available providers
func (m *UnifiedCloudManager) GetProviderInfo() map[CloudProvider]ProviderInfo {
	info := make(map[CloudProvider]ProviderInfo)

	for provider, service := range m.services {
		info[provider] = service.GetProviderInfo()
	}

	return info
}

// HealthCheck checks the health of all services
func (m *UnifiedCloudManager) HealthCheck(ctx context.Context) map[CloudProvider]error {
	errors := make(map[CloudProvider]error)

	for provider, service := range m.services {
		if err := service.HealthCheck(ctx); err != nil {
			errors[provider] = err
		}
	}

	return errors
}

// GetService returns a specific cloud service
func (m *UnifiedCloudManager) GetService(provider CloudProvider) (CloudService, bool) {
	service, exists := m.services[provider]
	return service, exists
}

// ListProviders returns all available providers
func (m *UnifiedCloudManager) ListProviders() []CloudProvider {
	var providers []CloudProvider
	for provider := range m.services {
		providers = append(providers, provider)
	}
	return providers
}

// Agent is a cloud agent (legacy interface)
type Agent interface {
	GetName() string
	GetType() AgentType
}

// AgentManage manages cloud agents (legacy interface)
type AgentManage struct {
	Agents map[string]Agent
	Ctx    context.Context
}

func (am *AgentManage) AddAgent(agent Agent) {
	if am.Agents == nil {
		am.Agents = make(map[string]Agent)
	}
	am.Agents[agent.GetName()] = agent
}

func (am *AgentManage) GetAgent(name string) Agent {
	return am.Agents[name]
}

func NewAgentManage() *AgentManage {
	return &AgentManage{
		Agents: make(map[string]Agent),
	}
}

// Factory functions for creating services (to be implemented by each provider)
type GoogleServiceFactory func(ctx context.Context, config *GoogleConfig) (CloudService, error)
type DoubaoServiceFactory func(ctx context.Context, config *DoubaoConfig) (CloudService, error)
type CozeServiceFactory func(ctx context.Context, config *CozeConfig) (CloudService, error)

// Service factories (to be set by each provider package)
var (
	NewGoogleService GoogleServiceFactory
	NewDoubaoService DoubaoServiceFactory
	NewCozeService   CozeServiceFactory
)

// Helper functions for creating media generation requests
func NewImageGenerationRequest(prompt string, options map[string]interface{}) *GenerateRequest {
	return &GenerateRequest{
		Prompt:    prompt,
		MediaType: MediaTypeImage,
		Options:   options,
	}
}

func NewVideoGenerationRequest(prompt string, options map[string]interface{}) *GenerateRequest {
	return &GenerateRequest{
		Prompt:    prompt,
		MediaType: MediaTypeVideo,
		Options:   options,
	}
}

func NewTextGenerationRequest(prompt string, options map[string]interface{}) *GenerateRequest {
	return &GenerateRequest{
		Prompt:    prompt,
		MediaType: MediaTypeText,
		Options:   options,
	}
}

// Image-to-image generation requests
func NewImageToImageGenerationRequest(prompt string, imageURL string, options map[string]interface{}) *GenerateRequest {
	if options == nil {
		options = make(map[string]interface{})
	}
	options["reference_image"] = imageURL
	options["generation_mode"] = "image_to_image"

	return &GenerateRequest{
		Prompt:    prompt,
		MediaType: MediaTypeImage,
		Options:   options,
	}
}

func NewMultiImageToImageGenerationRequest(prompt string, imageURLs []string, options map[string]interface{}) *GenerateRequest {
	if options == nil {
		options = make(map[string]interface{})
	}
	options["reference_images"] = imageURLs
	options["generation_mode"] = "multi_image_to_image"

	return &GenerateRequest{
		Prompt:    prompt,
		MediaType: MediaTypeImage,
		Options:   options,
	}
}

// Image-to-video generation requests
func NewImageToVideoGenerationRequest(prompt string, imageURL string, options map[string]interface{}) *GenerateRequest {
	if options == nil {
		options = make(map[string]interface{})
	}
	options["reference_image"] = imageURL
	options["generation_mode"] = "image_to_video"

	return &GenerateRequest{
		Prompt:    prompt,
		MediaType: MediaTypeVideo,
		Options:   options,
	}
}

func NewFirstLastFrameVideoGenerationRequest(prompt string, firstFrameURL string, lastFrameURL string, options map[string]interface{}) *GenerateRequest {
	if options == nil {
		options = make(map[string]interface{})
	}
	options["first_frame"] = firstFrameURL
	options["last_frame"] = lastFrameURL
	options["generation_mode"] = "first_last_frame_video"

	return &GenerateRequest{
		Prompt:    prompt,
		MediaType: MediaTypeVideo,
		Options:   options,
	}
}

// Media generation convenience methods
func (m *UnifiedCloudManager) GenerateImage(ctx context.Context, prompt string, options map[string]interface{}) (*UnifiedMediaResponse, error) {
	req := NewImageGenerationRequest(prompt, options)
	return m.GenerateMedia(ctx, req)
}

func (m *UnifiedCloudManager) GenerateVideo(ctx context.Context, prompt string, options map[string]interface{}) (*UnifiedMediaResponse, error) {
	req := NewVideoGenerationRequest(prompt, options)
	return m.GenerateMedia(ctx, req)
}

func (m *UnifiedCloudManager) GenerateText(ctx context.Context, prompt string, options map[string]interface{}) (*UnifiedMediaResponse, error) {
	req := NewTextGenerationRequest(prompt, options)
	return m.GenerateMedia(ctx, req)
}

// File generation methods
func (m *UnifiedCloudManager) GenerateImageToFile(ctx context.Context, prompt string, filename string, options map[string]interface{}) error {
	resp, err := m.GenerateImage(ctx, prompt, options)
	if err != nil {
		return err
	}

	if !resp.Success {
		return resp.Error
	}

	if len(resp.Data) == 0 && resp.URL != "" {
		// TODO: Download from URL
		return fmt.Errorf("URL download not implemented")
	}

	return WriteFile(filename, resp.Data)
}

func (m *UnifiedCloudManager) GenerateVideoToFile(ctx context.Context, prompt string, filename string, options map[string]interface{}) error {
	resp, err := m.GenerateVideo(ctx, prompt, options)
	if err != nil {
		return err
	}

	if !resp.Success {
		return resp.Error
	}

	if len(resp.Data) == 0 && resp.URL != "" {
		// TODO: Download from URL
		return fmt.Errorf("URL download not implemented")
	}

	return WriteFile(filename, resp.Data)
}

func (m *UnifiedCloudManager) GenerateImageToWriter(ctx context.Context, prompt string, writer io.Writer, options map[string]interface{}) error {
	resp, err := m.GenerateImage(ctx, prompt, options)
	if err != nil {
		return err
	}

	if !resp.Success {
		return resp.Error
	}

	if len(resp.Data) == 0 && resp.URL != "" {
		// TODO: Download from URL
		return fmt.Errorf("URL download not implemented")
	}

	_, err = writer.Write(resp.Data)
	return err
}

func (m *UnifiedCloudManager) GenerateVideoToWriter(ctx context.Context, prompt string, writer io.Writer, options map[string]interface{}) error {
	resp, err := m.GenerateVideo(ctx, prompt, options)
	if err != nil {
		return err
	}

	if !resp.Success {
		return resp.Error
	}

	if len(resp.Data) == 0 && resp.URL != "" {
		// TODO: Download from URL
		return fmt.Errorf("URL download not implemented")
	}

	_, err = writer.Write(resp.Data)
	return err
}

// WriteFile is a helper function to write data to a file
func WriteFile(filename string, data []byte) error {
	// TODO: Implement file writing
	return nil
}
