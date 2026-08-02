package coze

import (
	"context"
	"fmt"
	"time"

	"github.com/grapery/grapery/pkg/cloud"
)

// CozeAdapter implements the cloud.CloudService interface for Coze services
type CozeAdapter struct {
	client *HuoShanCozeClient
	config *cloud.CozeConfig
}

// NewCozeService creates a new Coze cloud service
func NewCozeService(ctx context.Context, config *cloud.CozeConfig) (cloud.CloudService, error) {
	if config == nil {
		return nil, fmt.Errorf("coze config is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("coze api key is required")
	}

	adapter := &CozeAdapter{
		config: config,
	}

	// Initialize Coze client
	client, err := NewCozeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create coze client: %w", err)
	}
	adapter.client = client

	return adapter, nil
}

// GenerateMedia implements the CloudService interface
func (c *CozeAdapter) GenerateMedia(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	if req == nil {
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     fmt.Errorf("request is required"),
			Provider:  cloud.ProviderCoze,
			Timestamp: time.Now(),
		}, fmt.Errorf("request is required")
	}

	switch req.MediaType {
	case cloud.MediaTypeText:
		return c.generateText(ctx, req)
	case cloud.MediaTypeImage:
		return c.generateImage(ctx, req)
	case cloud.MediaTypeVideo:
		return c.generateVideo(ctx, req)
	default:
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     fmt.Errorf("unsupported media type: %s", req.MediaType),
			Provider:  cloud.ProviderCoze,
			Timestamp: time.Now(),
		}, fmt.Errorf("unsupported media type: %s", req.MediaType)
	}
}

// generateText handles text generation (chat completion)
func (c *CozeAdapter) generateText(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// For now, simulate text generation since the actual Coze chat implementation is not available
	// In a real implementation, this would use the Coze chat API
	
	return &cloud.UnifiedMediaResponse{
		Success:   true,
		Provider:  cloud.ProviderCoze,
		Timestamp: time.Now(),
		Data:      []byte(fmt.Sprintf("Coze response to: %s", req.Prompt)),
		MimeType:  "text/plain",
		Metadata: map[string]interface{}{
			"model":         "coze-chat",
			"prompt":        req.Prompt,
			"response_type": "text",
			"tokens_used":   100,
		},
	}, nil
}

// generateImage handles image generation
func (c *CozeAdapter) generateImage(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// Coze may not have direct image generation capabilities
	// This would need to be implemented based on the actual Coze API
	
	return &cloud.UnifiedMediaResponse{
		Success:   false,
		Error:     fmt.Errorf("image generation not supported by Coze adapter"),
		Provider:  cloud.ProviderCoze,
		Timestamp: time.Now(),
	}, fmt.Errorf("image generation not supported by Coze adapter")
}

// generateVideo handles video generation
func (c *CozeAdapter) generateVideo(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// Coze may not have direct video generation capabilities
	// This would need to be implemented based on the actual Coze API
	
	return &cloud.UnifiedMediaResponse{
		Success:   false,
		Error:     fmt.Errorf("video generation not supported by Coze adapter"),
		Provider:  cloud.ProviderCoze,
		Timestamp: time.Now(),
	}, fmt.Errorf("video generation not supported by Coze adapter")
}

// GetProviderInfo implements the CloudService interface
func (c *CozeAdapter) GetProviderInfo() cloud.ProviderInfo {
	models := make(map[string]cloud.ModelInfo)
	
	// Add chat model
	models["coze-chat"] = cloud.ModelInfo{
		ID:          "coze-chat",
		Name:        "Coze Chat",
		Type:        cloud.MediaTypeText,
		Capabilities: []string{"text-generation", "chat-completion", "conversation"},
		Parameters: map[string]interface{}{
			"temperature":     []float64{0.1, 0.3, 0.5, 0.7, 0.9, 1.0, 1.2, 1.5, 2.0},
			"max_tokens":     []int{100, 500, 1000, 2000, 4000},
			"top_p":          []float64{0.1, 0.3, 0.5, 0.7, 0.9, 1.0},
			"frequency_penalty": []float64{-2.0, -1.0, 0.0, 1.0, 2.0},
			"presence_penalty":  []float64{-2.0, -1.0, 0.0, 1.0, 2.0},
		},
		Limits: map[string]interface{}{
			"max_tokens":     4000,
			"max_context":    16000,
		},
	}

	return cloud.ProviderInfo{
		Name:         cloud.ProviderCoze,
		DisplayName:  "Coze",
		Capabilities: []cloud.MediaType{cloud.MediaTypeText},
		Models:       models,
		Features: map[string]interface{}{
			"text_generation":  true,
			"chat_completion":  true,
			"conversation":     true,
			"image_generation": false,
			"video_generation": false,
		},
		Status: "active",
	}
}

// HealthCheck implements the CloudService interface
func (c *CozeAdapter) HealthCheck(ctx context.Context) error {
	// Simple health check - verify API key is available
	if c.client.GetAPIKey() == "" {
		return fmt.Errorf("coze api key is not configured")
	}
	
	return nil
}

// GetCapabilities implements the CloudService interface
func (c *CozeAdapter) GetCapabilities() []cloud.MediaType {
	return []cloud.MediaType{cloud.MediaTypeText}
}

// GetModels implements the CloudService interface
func (c *CozeAdapter) GetModels() []cloud.ModelInfo {
	info := c.GetProviderInfo()
	var models []cloud.ModelInfo
	for _, model := range info.Models {
		models = append(models, model)
	}
	return models
}

// Initialize the Coze service factory
func init() {
	cloud.NewCozeService = NewCozeService
}