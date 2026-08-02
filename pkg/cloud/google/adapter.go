package google

import (
	"context"
	"fmt"
	"time"

	"github.com/grapery/grapery/pkg/cloud"
)

// GoogleAdapter implements the cloud.CloudService interface for Google services
type GoogleAdapter struct {
	imageGenerator *GeminiImageGenerator
	videoGenerator *VeoVideoGenerator
	config        *cloud.GoogleConfig
}

// NewGoogleService creates a new Google cloud service
func NewGoogleService(ctx context.Context, config *cloud.GoogleConfig) (cloud.CloudService, error) {
	if config == nil {
		return nil, fmt.Errorf("google config is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("google api key is required")
	}

	adapter := &GoogleAdapter{
		config: config,
	}

	// Initialize image generator
	imageGen, err := NewGeminiImageGenerator(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create google image generator: %w", err)
	}
	adapter.imageGenerator = imageGen

	// Initialize video generator
	videoGen, err := NewVeoVideoGenerator(config.APIKey, "veo2")
	if err != nil {
		return nil, fmt.Errorf("failed to create google video generator: %w", err)
	}
	adapter.videoGenerator = videoGen

	return adapter, nil
}

// GenerateMedia implements the CloudService interface
func (g *GoogleAdapter) GenerateMedia(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	if req == nil {
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     fmt.Errorf("request is required"),
			Provider:  cloud.ProviderGoogle,
			Timestamp: time.Now(),
		}, fmt.Errorf("request is required")
	}

	switch req.MediaType {
	case cloud.MediaTypeImage:
		return g.generateImage(ctx, req)
	case cloud.MediaTypeVideo:
		return g.generateVideo(ctx, req)
	case cloud.MediaTypeText:
		return g.generateText(ctx, req)
	default:
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     fmt.Errorf("unsupported media type: %s", req.MediaType),
			Provider:  cloud.ProviderGoogle,
			Timestamp: time.Now(),
		}, fmt.Errorf("unsupported media type: %s", req.MediaType)
	}
}

// generateImage handles image generation
func (g *GoogleAdapter) generateImage(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// Determine generation mode based on options
	generationMode := "text_to_image" // default
	if req.Options != nil {
		if mode, ok := req.Options["generation_mode"].(string); ok {
			generationMode = mode
		}
	}

	imageReq := &ImageGenerationRequest{
		Prompt: req.Prompt,
	}

	// Extract options
	if req.Options != nil {
		if style, ok := req.Options["style"].(string); ok {
			imageReq.Style = style
		}
		if quality, ok := req.Options["quality"].(string); ok {
			imageReq.Quality = quality
		}
		if size, ok := req.Options["size"].(string); ok {
			imageReq.Size = size
		}
		if aspectRatio, ok := req.Options["aspect_ratio"].(string); ok {
			imageReq.AspectRatio = aspectRatio
		}
		if negativePrompt, ok := req.Options["negative_prompt"].(string); ok {
			imageReq.NegativePrompt = negativePrompt
		}
	}

	// For image-to-image or multi-image-to-image, add reference images to prompt
	switch generationMode {
	case "image_to_image":
		if req.Options != nil {
			if imageURL, ok := req.Options["reference_image"].(string); ok {
				imageReq.Prompt = fmt.Sprintf("Generate image based on reference: %s. %s", imageURL, req.Prompt)
			}
		}
	case "multi_image_to_image":
		if req.Options != nil {
			if imageURLs, ok := req.Options["reference_images"].([]string); ok {
				imageRefs := ""
				for _, url := range imageURLs {
					imageRefs += fmt.Sprintf("Reference image: %s. ", url)
				}
				imageReq.Prompt = fmt.Sprintf("%s%s", imageRefs, req.Prompt)
			}
		}
	}

	response, err := g.imageGenerator.GenerateImage(ctx, imageReq)
	if err != nil {
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     err,
			Provider:  cloud.ProviderGoogle,
			Timestamp: time.Now(),
		}, err
	}

	unifiedResp := &cloud.UnifiedMediaResponse{
		Success:   response.Success,
		Data:      response.Data,
		MimeType:  response.MimeType,
		Metadata:  response.Metadata,
		Provider:  cloud.ProviderGoogle,
		Timestamp: time.Now(),
	}

	// Add generation mode to metadata
	if unifiedResp.Metadata == nil {
		unifiedResp.Metadata = make(map[string]interface{})
	}
	unifiedResp.Metadata["generation_mode"] = generationMode

	return unifiedResp, nil
}

// generateVideo handles video generation
func (g *GoogleAdapter) generateVideo(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// Determine generation mode based on options
	generationMode := "text_to_video" // default
	if req.Options != nil {
		if mode, ok := req.Options["generation_mode"].(string); ok {
			generationMode = mode
		}
	}

	videoReq := &VideoGenerationRequest{
		Prompt: req.Prompt,
	}

	// Extract options
	if req.Options != nil {
		if duration, ok := req.Options["duration"].(int); ok {
			videoReq.Duration = duration
		}
		if resolution, ok := req.Options["resolution"].(string); ok {
			videoReq.Resolution = resolution
		}
		if frameRate, ok := req.Options["frame_rate"].(int); ok {
			videoReq.FrameRate = frameRate
		}
		if style, ok := req.Options["style"].(string); ok {
			videoReq.Style = style
		}
		if negativePrompt, ok := req.Options["negative_prompt"].(string); ok {
			videoReq.NegativePrompt = negativePrompt
		}
	}

	// For image-to-video or first-last-frame-video, add reference images to prompt
	switch generationMode {
	case "image_to_video":
		if req.Options != nil {
			if imageURL, ok := req.Options["reference_image"].(string); ok {
				videoReq.Prompt = fmt.Sprintf("Generate video based on reference image: %s. %s", imageURL, req.Prompt)
			}
		}
	case "first_last_frame_video":
		if req.Options != nil {
			if firstFrameURL, ok := req.Options["first_frame"].(string); ok {
				if lastFrameURL, ok := req.Options["last_frame"].(string); ok {
					videoReq.Prompt = fmt.Sprintf("Generate video starting with first frame: %s and ending with last frame: %s. %s", firstFrameURL, lastFrameURL, req.Prompt)
				}
			}
		}
	}

	response, err := g.videoGenerator.GenerateVideo(ctx, videoReq)
	if err != nil {
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     err,
			Provider:  cloud.ProviderGoogle,
			Timestamp: time.Now(),
		}, err
	}

	unifiedResp := &cloud.UnifiedMediaResponse{
		Success:   response.Success,
		Data:      response.Data,
		MimeType:  response.MimeType,
		Metadata:  response.Metadata,
		Provider:  cloud.ProviderGoogle,
		Timestamp: time.Now(),
	}

	// Add generation mode to metadata
	if unifiedResp.Metadata == nil {
		unifiedResp.Metadata = make(map[string]interface{})
	}
	unifiedResp.Metadata["generation_mode"] = generationMode

	return unifiedResp, nil
}

// generateText handles text generation (chat completion)
func (g *GoogleAdapter) generateText(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// For now, return an error as text generation is not implemented
	return &cloud.UnifiedMediaResponse{
		Success:   false,
		Error:     fmt.Errorf("text generation not supported by Google adapter"),
		Provider:  cloud.ProviderGoogle,
		Timestamp: time.Now(),
	}, fmt.Errorf("text generation not supported by Google adapter")
}

// GetProviderInfo implements the CloudService interface
func (g *GoogleAdapter) GetProviderInfo() cloud.ProviderInfo {
	models := make(map[string]cloud.ModelInfo)
	
	// Add image generation models
	models["gemini-image"] = cloud.ModelInfo{
		ID:          "gemini-image",
		Name:        "Gemini Image Generator",
		Type:        cloud.MediaTypeImage,
		Capabilities: []string{"text-to-image"},
		Parameters: map[string]interface{}{
			"style":           []string{"realistic", "artistic", "cartoon"},
			"quality":         []string{"standard", "high"},
			"size":            []string{"1024x1024", "512x512", "256x256"},
			"aspect_ratio":    []string{"1:1", "16:9", "4:3", "3:2", "9:16"},
		},
		Limits: map[string]interface{}{
			"max_images":     1,
			"max_resolution": "1024x1024",
		},
	}
	
	// Add video generation models
	models["veo-video"] = cloud.ModelInfo{
		ID:          "veo-video",
		Name:        "Veo Video Generator",
		Type:        cloud.MediaTypeVideo,
		Capabilities: []string{"text-to-video"},
		Parameters: map[string]interface{}{
			"duration":       []int{5, 10, 15, 20, 25, 30},
			"resolution":     []string{"1280x720", "1920x1080", "2560x1440", "3840x2160"},
			"frame_rate":     []int{24, 25, 30, 50, 60},
			"style":          []string{"realistic", "artistic"},
		},
		Limits: map[string]interface{}{
			"max_duration":   60,
			"min_duration":   1,
			"max_resolution": "3840x2160",
		},
	}

	return cloud.ProviderInfo{
		Name:         cloud.ProviderGoogle,
		DisplayName:  "Google Gemini",
		Capabilities: []cloud.MediaType{cloud.MediaTypeImage, cloud.MediaTypeVideo},
		Models:       models,
		Features: map[string]interface{}{
			"image_generation": true,
			"video_generation": true,
			"text_generation":  false,
			"async_generation": false,
		},
		Status: "active",
	}
}

// HealthCheck implements the CloudService interface
func (g *GoogleAdapter) HealthCheck(ctx context.Context) error {
	// Simple health check - try to generate a small test image
	testReq := &ImageGenerationRequest{
		Prompt: "test",
		Size:   "256x256",
	}
	
	_, err := g.imageGenerator.GenerateImage(ctx, testReq)
	if err != nil {
		return fmt.Errorf("google service health check failed: %w", err)
	}
	
	return nil
}

// GetCapabilities implements the CloudService interface
func (g *GoogleAdapter) GetCapabilities() []cloud.MediaType {
	return []cloud.MediaType{cloud.MediaTypeImage, cloud.MediaTypeVideo}
}

// GetModels implements the CloudService interface
func (g *GoogleAdapter) GetModels() []cloud.ModelInfo {
	info := g.GetProviderInfo()
	var models []cloud.ModelInfo
	for _, model := range info.Models {
		models = append(models, model)
	}
	return models
}

// Initialize the Google service factory
func init() {
	cloud.NewGoogleService = NewGoogleService
}