package doubao

import (
	"context"
	"fmt"
	"time"

	"github.com/grapery/grapery/pkg/cloud"
)

// DoubaoAdapter implements the cloud.CloudService interface for Doubao services
type DoubaoAdapter struct {
	imageClient *SeedreamClient
	videoClient *VideoClient
	config      *cloud.DoubaoConfig
}

// NewDoubaoService creates a new Doubao cloud service
func NewDoubaoService(ctx context.Context, config *cloud.DoubaoConfig) (cloud.CloudService, error) {
	if config == nil {
		return nil, fmt.Errorf("doubao config is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("doubao api key is required")
	}

	fmt.Printf("Creating DoubaoAdapter with APIKey: %s\n", config.APIKey)
	fmt.Printf("About to create imageClient\n")
	imageClient := NewSeedreamClient(config.APIKey)
	fmt.Printf("About to create videoClient\n")
	videoClient := NewVideoClient(config.APIKey)
	fmt.Printf("About to create adapter\n")
	adapter := &DoubaoAdapter{
		config:      config,
		imageClient: imageClient,
		videoClient: videoClient,
	}

	return adapter, nil
}

// GenerateMedia implements the CloudService interface
func (d *DoubaoAdapter) GenerateMedia(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	if req == nil {
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     fmt.Errorf("request is required"),
			Provider:  cloud.ProviderDoubao,
			Timestamp: time.Now(),
		}, fmt.Errorf("request is required")
	}

	switch req.MediaType {
	case cloud.MediaTypeImage:
		return d.generateImage(ctx, req)
	case cloud.MediaTypeVideo:
		return d.generateVideo(ctx, req)
	case cloud.MediaTypeText:
		return d.generateText(ctx, req)
	default:
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     fmt.Errorf("unsupported media type: %s", req.MediaType),
			Provider:  cloud.ProviderDoubao,
			Timestamp: time.Now(),
		}, fmt.Errorf("unsupported media type: %s", req.MediaType)
	}
}

// generateImage handles image generation
func (d *DoubaoAdapter) generateImage(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// Determine generation mode based on options
	generationMode := "text_to_image" // default
	if req.Options != nil {
		if mode, ok := req.Options["generation_mode"].(string); ok {
			generationMode = mode
		}
	}

	var response *ImageGenerationResponse
	var err error

	switch generationMode {
	case "image_to_image":
		response, err = d.generateImageToImage(ctx, req)
	case "multi_image_to_image":
		response, err = d.generateMultiImageToImage(ctx, req)
	default:
		response, err = d.generateTextToImage(ctx, req)
	}

	if err != nil {
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     err,
			Provider:  cloud.ProviderDoubao,
			Timestamp: time.Now(),
		}, err
	}

	// Convert response to unified format
	unifiedResp := &cloud.UnifiedMediaResponse{
		Success:   response.Error == nil,
		Provider:  cloud.ProviderDoubao,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	if response.Error != nil {
		unifiedResp.Error = response.Error
		return unifiedResp, response.Error
	}

	// Extract data from response
	if len(response.Data) > 0 {
		// For URL responses, we'll store the URL in metadata
		if response.Data[0].URL != "" {
			unifiedResp.URL = response.Data[0].URL
			unifiedResp.Metadata["url"] = response.Data[0].URL
			unifiedResp.Metadata["revised_prompt"] = response.Data[0].RevisedPrompt
		}
		if response.Data[0].B64JSON != "" {
			// TODO: Convert base64 to bytes
			unifiedResp.Metadata["base64_data"] = response.Data[0].B64JSON
		}
	}

	// Add usage information
	unifiedResp.Metadata["usage"] = response.Usage
	unifiedResp.Metadata["model"] = response.Model
	unifiedResp.Metadata["created"] = response.Created
	unifiedResp.Metadata["generation_mode"] = generationMode

	return unifiedResp, nil
}

// generateTextToImage handles text-to-image generation
func (d *DoubaoAdapter) generateTextToImage(ctx context.Context, req *cloud.UnifiedMediaRequest) (*ImageGenerationResponse, error) {
	// Build options for image generation
	var options []func(*TextToImageRequest)

	if req.Options != nil {
		if model, ok := req.Options["model"].(string); ok {
			options = append(options, WithModel(model))
		}
		if negativePrompt, ok := req.Options["negative_prompt"].(string); ok {
			options = append(options, WithNegativePrompt(negativePrompt))
		}
		if size, ok := req.Options["size"].(string); ok {
			options = append(options, WithSize(size))
		}
		if quality, ok := req.Options["quality"].(string); ok {
			options = append(options, WithQuality(quality))
		}
		if style, ok := req.Options["style"].(string); ok {
			options = append(options, WithStyle(style))
		}
		if watermark, ok := req.Options["watermark"].(bool); ok {
			options = append(options, WithWatermark(watermark))
		}
		if seed, ok := req.Options["seed"].(int64); ok {
			options = append(options, WithSeed(seed))
		}
		if count, ok := req.Options["count"].(int); ok {
			options = append(options, WithImageCount(count))
		}
	}

	return d.imageClient.SimpleGenerateImage(ctx, req.Prompt, options...)
}

// generateImageToImage handles image-to-image generation
func (d *DoubaoAdapter) generateImageToImage(ctx context.Context, req *cloud.UnifiedMediaRequest) (*ImageGenerationResponse, error) {
	if req.Options == nil {
		return nil, fmt.Errorf("reference image is required for image-to-image generation")
	}

	imageURL, ok := req.Options["reference_image"].(string)
	if !ok || imageURL == "" {
		return nil, fmt.Errorf("valid reference image URL is required")
	}

	// Use SeedDream 4.0 for image-to-image generation
	response, err := d.imageClient.GenerateImageToImage(ctx, req.Prompt, imageURL)
	if err != nil {
		return nil, err
	}

	// Convert SeedDream4Response to ImageGenerationResponse for compatibility
	legacyResp := &ImageGenerationResponse{
		Model:   response.Model,
		Created: response.Created,
		Error:   response.Error,
		Usage: struct {
			GeneratedImages int `json:"generated_images"`
		}{
			GeneratedImages: response.Usage.GeneratedImages,
		},
	}

	if len(response.Data) > 0 {
		legacyResp.Data = make([]struct {
			URL           string `json:"url,omitempty"`
			B64JSON       string `json:"b64_json,omitempty"`
			RevisedPrompt string `json:"revised_prompt,omitempty"`
		}, len(response.Data))

		for i, data := range response.Data {
			legacyResp.Data[i].URL = data.URL
			legacyResp.Data[i].RevisedPrompt = ""
		}
	}

	return legacyResp, nil
}

// generateMultiImageToImage handles multi-image-to-image generation
func (d *DoubaoAdapter) generateMultiImageToImage(ctx context.Context, req *cloud.UnifiedMediaRequest) (*ImageGenerationResponse, error) {
	if req.Options == nil {
		return nil, fmt.Errorf("reference images are required for multi-image-to-image generation")
	}

	imageURLs, ok := req.Options["reference_images"].([]string)
	if !ok || len(imageURLs) == 0 {
		return nil, fmt.Errorf("valid reference image URLs are required")
	}

	// Use SeedDream 4.0 for multi-image-to-image generation
	response, err := d.imageClient.GenerateMultiImageToImage(ctx, req.Prompt, imageURLs)
	if err != nil {
		return nil, err
	}

	// Convert SeedDream4Response to ImageGenerationResponse for compatibility
	legacyResp := &ImageGenerationResponse{
		Model:   response.Model,
		Created: response.Created,
		Error:   response.Error,
		Usage: struct {
			GeneratedImages int `json:"generated_images"`
		}{
			GeneratedImages: response.Usage.GeneratedImages,
		},
	}

	if len(response.Data) > 0 {
		legacyResp.Data = make([]struct {
			URL           string `json:"url,omitempty"`
			B64JSON       string `json:"b64_json,omitempty"`
			RevisedPrompt string `json:"revised_prompt,omitempty"`
		}, len(response.Data))

		for i, data := range response.Data {
			legacyResp.Data[i].URL = data.URL
			legacyResp.Data[i].RevisedPrompt = ""
		}
	}

	return legacyResp, nil
}

// generateVideo handles video generation
func (d *DoubaoAdapter) generateVideo(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// Determine generation mode based on options
	generationMode := "text_to_video" // default
	if req.Options != nil {
		if mode, ok := req.Options["generation_mode"].(string); ok {
			generationMode = mode
		}
	}

	// Determine model based on options
	model := ModelSeedanceProNew // Default model
	if req.Options != nil {
		if m, ok := req.Options["model"].(string); ok {
			model = m
		}
	}

	var taskResp *VideoGenerationResponse
	var err error

	switch generationMode {
	case "image_to_video":
		taskResp, err = d.generateImageToVideo(ctx, req, model)
	case "first_last_frame_video":
		taskResp, err = d.generateFirstLastFrameVideo(ctx, req, model)
	default:
		taskResp, err = d.generateTextToVideo(ctx, req, model)
	}

	if err != nil {
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     err,
			Provider:  cloud.ProviderDoubao,
			Timestamp: time.Now(),
		}, err
	}

	// For video generation, we return the task ID and status
	return &cloud.UnifiedMediaResponse{
		Success:   taskResp.Error == nil,
		Provider:  cloud.ProviderDoubao,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"task_id":         taskResp.ID,
			"model":           model,
			"prompt":          req.Prompt,
			"status":          "pending",
			"created_at":      time.Now().Unix(),
			"generation_mode": generationMode,
		},
	}, nil
}

// generateTextToVideo handles text-to-video generation
func (d *DoubaoAdapter) generateTextToVideo(ctx context.Context, req *cloud.UnifiedMediaRequest, model string) (*VideoGenerationResponse, error) {
	// Create video generation request
	videoReq := NewTextToVideoRequest(model, req.Prompt)

	// Add callback URL if specified
	if req.Options != nil {
		if callbackURL, ok := req.Options["callback_url"].(string); ok {
			videoReq.WithCallbackURL(callbackURL)
		}
	}

	// Create the task
	return d.videoClient.CreateVideoGenerationTask(ctx, videoReq)
}

// generateImageToVideo handles image-to-video generation
func (d *DoubaoAdapter) generateImageToVideo(ctx context.Context, req *cloud.UnifiedMediaRequest, model string) (*VideoGenerationResponse, error) {
	if req.Options == nil {
		return nil, fmt.Errorf("reference image is required for image-to-video generation")
	}

	imageURL, ok := req.Options["reference_image"].(string)
	if !ok || imageURL == "" {
		return nil, fmt.Errorf("valid reference image URL is required")
	}

	// Create video generation request with image
	videoReq := NewImageToVideoRequest(model, req.Prompt, imageURL)

	// Add callback URL if specified
	if req.Options != nil {
		if callbackURL, ok := req.Options["callback_url"].(string); ok {
			videoReq.WithCallbackURL(callbackURL)
		}
	}

	// Create the task
	return d.videoClient.CreateVideoGenerationTask(ctx, videoReq)
}

// generateFirstLastFrameVideo handles first-last-frame video generation
func (d *DoubaoAdapter) generateFirstLastFrameVideo(ctx context.Context, req *cloud.UnifiedMediaRequest, model string) (*VideoGenerationResponse, error) {
	if req.Options == nil {
		return nil, fmt.Errorf("first and last frame images are required for first-last-frame video generation")
	}

	firstFrameURL, ok := req.Options["first_frame"].(string)
	if !ok || firstFrameURL == "" {
		return nil, fmt.Errorf("valid first frame image URL is required")
	}

	lastFrameURL, ok := req.Options["last_frame"].(string)
	if !ok || lastFrameURL == "" {
		return nil, fmt.Errorf("valid last frame image URL is required")
	}

	// Create video generation request with first and last frames
	videoReq := NewFirstLastFrameVideoRequest(model, req.Prompt, firstFrameURL, lastFrameURL)

	// Add callback URL if specified
	if req.Options != nil {
		if callbackURL, ok := req.Options["callback_url"].(string); ok {
			videoReq.WithCallbackURL(callbackURL)
		}
	}

	// Create the task
	return d.videoClient.CreateVideoGenerationTask(ctx, videoReq)
}

// generateText handles text generation (embeddings)
func (d *DoubaoAdapter) generateText(ctx context.Context, req *cloud.UnifiedMediaRequest) (*cloud.UnifiedMediaResponse, error) {
	// Use text embedding as text generation
	texts := []string{req.Prompt}

	response, err := d.imageClient.GenerateTextEmbedding(ctx, texts)
	if err != nil {
		return &cloud.UnifiedMediaResponse{
			Success:   false,
			Error:     err,
			Provider:  cloud.ProviderDoubao,
			Timestamp: time.Now(),
		}, err
	}

	// Convert response to unified format
	unifiedResp := &cloud.UnifiedMediaResponse{
		Success:   response.Error == nil,
		Provider:  cloud.ProviderDoubao,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	if response.Error != nil {
		unifiedResp.Error = response.Error
		return unifiedResp, response.Error
	}

	// Store embedding data
	if len(response.Data) > 0 {
		unifiedResp.Metadata["embedding"] = response.Data[0].Embedding
		unifiedResp.Metadata["embedding_length"] = len(response.Data[0].Embedding)
	}

	// Add usage information
	unifiedResp.Metadata["usage"] = response.Usage
	unifiedResp.Metadata["model"] = response.Model
	unifiedResp.Metadata["created"] = response.Created

	return unifiedResp, nil
}

// GetProviderInfo implements the CloudService interface
func (d *DoubaoAdapter) GetProviderInfo() cloud.ProviderInfo {
	models := make(map[string]cloud.ModelInfo)

	// Add image generation models
	models["seedream-4-0"] = cloud.ModelInfo{
		ID:           "seedream-4-0",
		Name:         "SeedDream 4.0",
		Type:         cloud.MediaTypeImage,
		Capabilities: []string{"text-to-image", "image-to-image", "multi-image-to-image", "sequential-image"},
		Parameters: map[string]interface{}{
			"size":            []string{"2K", "4K", "1024x1024", "1024x1792", "1792x1024"},
			"quality":         []string{"standard", "hd"},
			"style":           []string{"vivid", "natural"},
			"sequential_mode": []string{"disabled", "auto", "manual"},
		},
		Limits: map[string]interface{}{
			"max_images":     4,
			"max_resolution": "4K",
		},
	}

	models["seedream-3-0"] = cloud.ModelInfo{
		ID:           "seedream-3-0",
		Name:         "SeedDream 3.0",
		Type:         cloud.MediaTypeImage,
		Capabilities: []string{"text-to-image"},
		Parameters: map[string]interface{}{
			"size":    []string{"1024x1024", "1024x1792", "1792x1024"},
			"quality": []string{"standard", "hd"},
			"style":   []string{"vivid", "natural"},
		},
		Limits: map[string]interface{}{
			"max_images":     4,
			"max_resolution": "1792x1024",
		},
	}

	// Add video generation models
	models["seedance-pro"] = cloud.ModelInfo{
		ID:           "seedance-pro",
		Name:         "Seedance Pro",
		Type:         cloud.MediaTypeVideo,
		Capabilities: []string{"text-to-video", "image-to-video", "first-last-frame-video"},
		Parameters: map[string]interface{}{
			"resolution": []string{"720p", "1080p"},
			"ratio":      []string{"16:9", "9:16", "adaptive"},
		},
		Limits: map[string]interface{}{
			"max_duration": 60,
			"min_duration": 1,
		},
	}

	models["wan2-1"] = cloud.ModelInfo{
		ID:           "wan2-1",
		Name:         "Wan 2.1",
		Type:         cloud.MediaTypeVideo,
		Capabilities: []string{"text-to-video", "image-to-video", "first-last-frame-video"},
		Parameters: map[string]interface{}{
			"resolution": []string{"720p", "1080p"},
			"ratio":      []string{"16:9", "9:16", "adaptive"},
		},
		Limits: map[string]interface{}{
			"max_duration": 60,
			"min_duration": 1,
		},
	}

	// Add embedding models
	models["embedding-text"] = cloud.ModelInfo{
		ID:           "embedding-text",
		Name:         "Text Embedding",
		Type:         cloud.MediaTypeText,
		Capabilities: []string{"text-embedding"},
		Parameters: map[string]interface{}{
			"encoding_format": []string{"float", "base64"},
		},
		Limits: map[string]interface{}{
			"max_tokens": 8192,
			"batch_size": 100,
		},
	}

	models["embedding-vision"] = cloud.ModelInfo{
		ID:           "embedding-vision",
		Name:         "Multimodal Embedding",
		Type:         cloud.MediaTypeText,
		Capabilities: []string{"text-embedding", "image-embedding", "video-embedding"},
		Parameters: map[string]interface{}{
			"encoding_format": []string{"float", "base64"},
		},
		Limits: map[string]interface{}{
			"max_tokens": 8192,
			"batch_size": 1,
		},
	}

	return cloud.ProviderInfo{
		Name:         cloud.ProviderDoubao,
		DisplayName:  "Doubao",
		Capabilities: []cloud.MediaType{cloud.MediaTypeImage, cloud.MediaTypeVideo, cloud.MediaTypeText},
		Models:       models,
		Features: map[string]interface{}{
			"image_generation":      true,
			"video_generation":      true,
			"text_generation":       true,
			"async_generation":      true,
			"embeddings":            true,
			"multimodal_embeddings": true,
		},
		Status: "active",
	}
}

// HealthCheck implements the CloudService interface
func (d *DoubaoAdapter) HealthCheck(ctx context.Context) error {
	// Simple health check - try to generate a text embedding
	texts := []string{"health check"}

	_, err := d.imageClient.GenerateTextEmbedding(ctx, texts)
	if err != nil {
		return fmt.Errorf("doubao service health check failed: %w", err)
	}

	return nil
}

// GetCapabilities implements the CloudService interface
func (d *DoubaoAdapter) GetCapabilities() []cloud.MediaType {
	return []cloud.MediaType{cloud.MediaTypeImage, cloud.MediaTypeVideo, cloud.MediaTypeText}
}

// GetModels implements the CloudService interface
func (d *DoubaoAdapter) GetModels() []cloud.ModelInfo {
	info := d.GetProviderInfo()
	var models []cloud.ModelInfo
	for _, model := range info.Models {
		models = append(models, model)
	}
	return models
}

// Initialize the Doubao service factory
func init() {
	cloud.NewDoubaoService = NewDoubaoService
}
