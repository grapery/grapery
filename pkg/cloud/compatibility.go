package cloud

import (
	"context"
	"fmt"
	"io"
	"time"
)

// Legacy interfaces for backward compatibility

// LegacyImageGenerator represents the old image generator interface
type LegacyImageGenerator interface {
	GenerateImage(ctx context.Context, req interface{}) (interface{}, error)
	GenerateImageToFile(ctx context.Context, req interface{}, filename string) error
	GenerateImageToWriter(ctx context.Context, req interface{}, w io.Writer) error
}

// LegacyVideoGenerator represents the old video generator interface
type LegacyVideoGenerator interface {
	GenerateVideo(ctx context.Context, req interface{}) (interface{}, error)
	GenerateVideoToFile(ctx context.Context, req interface{}, filename string) error
	GenerateVideoToWriter(ctx context.Context, req interface{}, w io.Writer) error
}

// CompatibilityWrapper wraps the new unified interface to maintain backward compatibility
type CompatibilityWrapper struct {
	manager *UnifiedCloudManager
}

// NewCompatibilityWrapper creates a new compatibility wrapper
func NewCompatibilityWrapper(manager *UnifiedCloudManager) *CompatibilityWrapper {
	return &CompatibilityWrapper{
		manager: manager,
	}
}

// GenerateImage provides backward compatibility for image generation
func (w *CompatibilityWrapper) GenerateImage(ctx context.Context, req interface{}) (interface{}, error) {
	// Try to convert the request to the new format
	unifiedReq, err := w.convertImageRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert image request: %w", err)
	}

	resp, err := w.manager.GenerateMedia(ctx, unifiedReq)
	if err != nil {
		return nil, err
	}

	return w.convertImageResponse(resp)
}

// GenerateImageToFile provides backward compatibility for image generation to file
func (w *CompatibilityWrapper) GenerateImageToFile(ctx context.Context, req interface{}, filename string) error {
	unifiedReq, err := w.convertImageRequest(req)
	if err != nil {
		return fmt.Errorf("failed to convert image request: %w", err)
	}

	return w.manager.GenerateImageToFile(ctx, unifiedReq.Prompt, filename, unifiedReq.Options)
}

// GenerateImageToWriter provides backward compatibility for image generation to writer
func (w *CompatibilityWrapper) GenerateImageToWriter(ctx context.Context, req interface{}, writer io.Writer) error {
	unifiedReq, err := w.convertImageRequest(req)
	if err != nil {
		return fmt.Errorf("failed to convert image request: %w", err)
	}

	return w.manager.GenerateImageToWriter(ctx, unifiedReq.Prompt, writer, unifiedReq.Options)
}

// GenerateVideo provides backward compatibility for video generation
func (w *CompatibilityWrapper) GenerateVideo(ctx context.Context, req interface{}) (interface{}, error) {
	// Try to convert the request to the new format
	unifiedReq, err := w.convertVideoRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert video request: %w", err)
	}

	resp, err := w.manager.GenerateMedia(ctx, unifiedReq)
	if err != nil {
		return nil, err
	}

	return w.convertVideoResponse(resp)
}

// GenerateVideoToFile provides backward compatibility for video generation to file
func (w *CompatibilityWrapper) GenerateVideoToFile(ctx context.Context, req interface{}, filename string) error {
	unifiedReq, err := w.convertVideoRequest(req)
	if err != nil {
		return fmt.Errorf("failed to convert video request: %w", err)
	}

	return w.manager.GenerateVideoToFile(ctx, unifiedReq.Prompt, filename, unifiedReq.Options)
}

// GenerateVideoToWriter provides backward compatibility for video generation to writer
func (w *CompatibilityWrapper) GenerateVideoToWriter(ctx context.Context, req interface{}, writer io.Writer) error {
	unifiedReq, err := w.convertVideoRequest(req)
	if err != nil {
		return fmt.Errorf("failed to convert video request: %w", err)
	}

	return w.manager.GenerateVideoToWriter(ctx, unifiedReq.Prompt, writer, unifiedReq.Options)
}

// convertImageRequest converts legacy image request to unified format
func (w *CompatibilityWrapper) convertImageRequest(req interface{}) (*GenerateRequest, error) {
	// Handle different types of legacy requests
	switch r := req.(type) {
	case map[string]interface{}:
		prompt, _ := r["prompt"].(string)
		options := make(map[string]interface{})
		
		// Extract known options
		if style, ok := r["style"].(string); ok {
			options["style"] = style
		}
		if quality, ok := r["quality"].(string); ok {
			options["quality"] = quality
		}
		if size, ok := r["size"].(string); ok {
			options["size"] = size
		}
		if aspectRatio, ok := r["aspect_ratio"].(string); ok {
			options["aspect_ratio"] = aspectRatio
		}
		
		return NewImageGenerationRequest(prompt, options), nil
		
	default:
		return nil, fmt.Errorf("unsupported image request type: %T", req)
	}
}

// convertVideoRequest converts legacy video request to unified format
func (w *CompatibilityWrapper) convertVideoRequest(req interface{}) (*GenerateRequest, error) {
	// Handle different types of legacy requests
	switch r := req.(type) {
	case map[string]interface{}:
		prompt, _ := r["prompt"].(string)
		options := make(map[string]interface{})
		
		// Extract known options
		if duration, ok := r["duration"].(int); ok {
			options["duration"] = duration
		}
		if resolution, ok := r["resolution"].(string); ok {
			options["resolution"] = resolution
		}
		if frameRate, ok := r["frame_rate"].(int); ok {
			options["frame_rate"] = frameRate
		}
		if style, ok := r["style"].(string); ok {
			options["style"] = style
		}
		
		return NewVideoGenerationRequest(prompt, options), nil
		
	default:
		return nil, fmt.Errorf("unsupported video request type: %T", req)
	}
}

// convertImageResponse converts unified response to legacy format
func (w *CompatibilityWrapper) convertImageResponse(resp *UnifiedMediaResponse) (interface{}, error) {
	legacyResp := map[string]interface{}{
		"success":    resp.Success,
		"provider":   resp.Provider,
		"timestamp":  resp.Timestamp,
		"mime_type":  resp.MimeType,
		"metadata":   resp.Metadata,
	}
	
	if resp.Error != nil {
		legacyResp["error"] = resp.Error.Error()
	}
	
	if len(resp.Data) > 0 {
		legacyResp["data"] = resp.Data
	}
	
	if resp.URL != "" {
		legacyResp["url"] = resp.URL
	}
	
	return legacyResp, nil
}

// convertVideoResponse converts unified response to legacy format
func (w *CompatibilityWrapper) convertVideoResponse(resp *UnifiedMediaResponse) (interface{}, error) {
	legacyResp := map[string]interface{}{
		"success":    resp.Success,
		"provider":   resp.Provider,
		"timestamp":  resp.Timestamp,
		"mime_type":  resp.MimeType,
		"metadata":   resp.Metadata,
	}
	
	if resp.Error != nil {
		legacyResp["error"] = resp.Error.Error()
	}
	
	if len(resp.Data) > 0 {
		legacyResp["data"] = resp.Data
	}
	
	if resp.URL != "" {
		legacyResp["url"] = resp.URL
	}
	
	return legacyResp, nil
}

// Global compatibility wrapper instance
var globalCompatibilityWrapper *CompatibilityWrapper

// InitializeCompatibility initializes the global compatibility wrapper
func InitializeCompatibility(manager *UnifiedCloudManager) {
	globalCompatibilityWrapper = NewCompatibilityWrapper(manager)
}

// GetCompatibilityWrapper returns the global compatibility wrapper
func GetCompatibilityWrapper() *CompatibilityWrapper {
	return globalCompatibilityWrapper
}

// Legacy convenience functions for backward compatibility

// GenerateImageLegacy provides backward compatibility for image generation
func GenerateImageLegacy(ctx context.Context, prompt string, options map[string]interface{}) (interface{}, error) {
	if globalCompatibilityWrapper == nil {
		return nil, fmt.Errorf("compatibility wrapper not initialized")
	}
	
	req := map[string]interface{}{
		"prompt": prompt,
	}
	
	// Add options
	for k, v := range options {
		req[k] = v
	}
	
	return globalCompatibilityWrapper.GenerateImage(ctx, req)
}

// GenerateVideoLegacy provides backward compatibility for video generation
func GenerateVideoLegacy(ctx context.Context, prompt string, options map[string]interface{}) (interface{}, error) {
	if globalCompatibilityWrapper == nil {
		return nil, fmt.Errorf("compatibility wrapper not initialized")
	}
	
	req := map[string]interface{}{
		"prompt": prompt,
	}
	
	// Add options
	for k, v := range options {
		req[k] = v
	}
	
	return globalCompatibilityWrapper.GenerateVideo(ctx, req)
}

// Provider-specific legacy functions

// GenerateGoogleImageLegacy generates an image using Google (legacy compatibility)
func GenerateGoogleImageLegacy(ctx context.Context, prompt string, options map[string]interface{}) (interface{}, error) {
	if globalCompatibilityWrapper == nil {
		return nil, fmt.Errorf("compatibility wrapper not initialized")
	}
	
	req := map[string]interface{}{
		"prompt": prompt,
	}
	
	// Add options
	for k, v := range options {
		req[k] = v
	}
	
	// Create a request that specifies Google provider
	unifiedReq := &GenerateRequest{
		Prompt:   prompt,
		Provider: ProviderGoogle,
		Options:  options,
	}
	
	resp, err := globalCompatibilityWrapper.manager.GenerateMedia(ctx, unifiedReq)
	if err != nil {
		return nil, err
	}
	
	return globalCompatibilityWrapper.convertImageResponse(resp)
}

// GenerateDoubaoVideoLegacy generates a video using Doubao (legacy compatibility)
func GenerateDoubaoVideoLegacy(ctx context.Context, prompt string, options map[string]interface{}) (interface{}, error) {
	if globalCompatibilityWrapper == nil {
		return nil, fmt.Errorf("compatibility wrapper not initialized")
	}
	
	req := map[string]interface{}{
		"prompt": prompt,
	}
	
	// Add options
	for k, v := range options {
		req[k] = v
	}
	
	// Create a request that specifies Doubao provider
	unifiedReq := &GenerateRequest{
		Prompt:   prompt,
		Provider: ProviderDoubao,
		Options:  options,
	}
	
	resp, err := globalCompatibilityWrapper.manager.GenerateMedia(ctx, unifiedReq)
	if err != nil {
		return nil, err
	}
	
	return globalCompatibilityWrapper.convertVideoResponse(resp)
}

// Helper function to create a legacy-style configuration
func CreateLegacyConfig(googleAPIKey, doubaoAPIKey, cozeAPIKey string, defaultProvider CloudProvider) *CloudConfig {
	return &CloudConfig{
		Google: &GoogleConfig{
			APIKey:  googleAPIKey,
			Enabled: googleAPIKey != "",
		},
		Doubao: &DoubaoConfig{
			APIKey:  doubaoAPIKey,
			Enabled: doubaoAPIKey != "",
		},
		Coze: &CozeConfig{
			APIKey:  cozeAPIKey,
			Enabled: cozeAPIKey != "",
		},
		DefaultProvider: defaultProvider,
		Timeout:        30 * time.Second,
		Retry:          3,
	}
}