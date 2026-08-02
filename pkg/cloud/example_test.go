package cloud

import (
	"context"
	"log"
	"testing"
	"time"
)

func TestUnifiedCloudManager(t *testing.T) {
	// Create a test configuration with disabled services to avoid initialization issues
	config := &CloudConfig{
		Google: &GoogleConfig{
			APIKey:  "test-google-api-key",
			Enabled: false, // Disabled for testing
		},
		Doubao: &DoubaoConfig{
			APIKey:  "test-doubao-api-key",
			Enabled: false, // Disabled for testing
		},
		Coze: &CozeConfig{
			APIKey:  "test-coze-api-key",
			Enabled: false, // Disabled for testing
		},
		DefaultProvider: ProviderGoogle,
		RoutingRules: []RoutingRule{
			{
				Name:      "video-rule",
				MediaType: MediaTypeVideo,
				Provider:  ProviderDoubao,
				Priority:  1,
			},
			{
				Name:      "text-rule",
				MediaType: MediaTypeText,
				Provider:  ProviderCoze,
				Priority:  1,
			},
		},
		Timeout: 30 * time.Second,
		Retry:   3,
	}

	ctx := context.Background()
	manager, err := NewUnifiedCloudManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create unified cloud manager: %v", err)
	}

	// Test provider listing
	providers := manager.ListProviders()
	t.Logf("Available providers: %v", providers)

	// Test provider info
	info := manager.GetProviderInfo()
	for provider, providerInfo := range info {
		t.Logf("Provider %s:", provider)
		t.Logf("  Display Name: %s", providerInfo.DisplayName)
		t.Logf("  Capabilities: %v", providerInfo.Capabilities)
		t.Logf("  Models: %d", len(providerInfo.Models))
	}

	// Test image generation (should fail gracefully with no providers available)
	imageResp, err := manager.GenerateImage(ctx, "A beautiful sunset over mountains", map[string]interface{}{
		"style":  "realistic",
		"size":   "1024x1024",
		"quality": "high",
	})
	if err != nil {
		t.Logf("Image generation error (expected with no providers): %v", err)
	} else {
		t.Logf("Image generation response: %+v", imageResp)
	}

	// Test video generation (should fail gracefully with no providers available)
	videoResp, err := manager.GenerateVideo(ctx, "A car driving through a city", map[string]interface{}{
		"duration":   10,
		"resolution": "1080p",
		"model":      "seedance-pro",
	})
	if err != nil {
		t.Logf("Video generation error (expected with no providers): %v", err)
	} else {
		t.Logf("Video generation response: %+v", videoResp)
	}

	// Test text generation (should fail gracefully with no providers available)
	textResp, err := manager.GenerateText(ctx, "Hello, how are you?", map[string]interface{}{
		"temperature": 0.7,
		"max_tokens":  100,
	})
	if err != nil {
		t.Logf("Text generation error (expected with no providers): %v", err)
	} else {
		t.Logf("Text generation response: %+v", textResp)
	}
}

// Example usage of the unified cloud manager
func ExampleUnifiedCloudManager() {
	// Create configuration
	config := &CloudConfig{
		Google: &GoogleConfig{
			APIKey:  "your-google-api-key",
			Enabled: true,
		},
		Doubao: &DoubaoConfig{
			APIKey:  "your-doubao-api-key",
			Enabled: true,
		},
		DefaultProvider: ProviderGoogle,
		Timeout:        30 * time.Second,
		Retry:          3,
	}

	// Create manager
	ctx := context.Background()
	manager, err := NewUnifiedCloudManager(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create cloud manager: %v", err)
	}

	// Generate an image
	imageResp, err := manager.GenerateImage(ctx, "A futuristic cityscape", map[string]interface{}{
		"style":  "realistic",
		"size":   "1024x1024",
		"quality": "high",
	})
	if err != nil {
		log.Printf("Failed to generate image: %v", err)
	} else if imageResp.Success {
		log.Printf("Image generated successfully using %s", imageResp.Provider)
		// Save the image or use the URL
	}

	// Generate a video
	videoResp, err := manager.GenerateVideo(ctx, "A robot walking in a park", map[string]interface{}{
		"duration":   15,
		"resolution": "1080p",
		"model":      "seedance-pro",
	})
	if err != nil {
		log.Printf("Failed to generate video: %v", err)
	} else if videoResp.Success {
		log.Printf("Video generated successfully using %s", videoResp.Provider)
		// Save the video or use the URL
	}

	// Generate text/chat response
	textResp, err := manager.GenerateText(ctx, "What is artificial intelligence?", map[string]interface{}{
		"temperature": 0.7,
		"max_tokens":  200,
	})
	if err != nil {
		log.Printf("Failed to generate text: %v", err)
	} else if textResp.Success {
		log.Printf("Text generated successfully using %s", textResp.Provider)
		log.Printf("Response: %s", string(textResp.Data))
	}
}

// Example of using specific providers
func ExampleUnifiedCloudManager_specificProviderUsage() {
	config := &CloudConfig{
		Google: &GoogleConfig{
			APIKey:  "your-google-api-key",
			Enabled: true,
		},
		Doubao: &DoubaoConfig{
			APIKey:  "your-doubao-api-key",
			Enabled: true,
		},
	}

	ctx := context.Background()
	manager, err := NewUnifiedCloudManager(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create cloud manager: %v", err)
	}

	// Generate image using Google specifically
	googleImageReq := NewImageGenerationRequest("A beautiful landscape", map[string]interface{}{
		"style": "artistic",
	})
	googleImageReq.Provider = ProviderGoogle
	googleImageResp, err := manager.GenerateMedia(ctx, googleImageReq)
	if err != nil {
		log.Printf("Failed to generate image with Google: %v", err)
	} else {
		log.Printf("Google image generation: %v", googleImageResp.Success)
	}

	// Generate video using Doubao specifically
	doubaoVideoReq := NewVideoGenerationRequest("A car driving on a highway", map[string]interface{}{
		"duration": 10,
	})
	doubaoVideoReq.Provider = ProviderDoubao
	doubaoVideoResp, err := manager.GenerateMedia(ctx, doubaoVideoReq)
	if err != nil {
		log.Printf("Failed to generate video with Doubao: %v", err)
	} else {
		log.Printf("Doubao video generation: %v", doubaoVideoResp.Success)
	}
}

// Example of health checking
func ExampleUnifiedCloudManager_healthCheck() {
	config := &CloudConfig{
		Google: &GoogleConfig{
			APIKey:  "your-google-api-key",
			Enabled: true,
		},
		Doubao: &DoubaoConfig{
			APIKey:  "your-doubao-api-key",
			Enabled: true,
		},
	}

	ctx := context.Background()
	manager, err := NewUnifiedCloudManager(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create cloud manager: %v", err)
	}

	// Check health of all services
	errors := manager.HealthCheck(ctx)
	for provider, err := range errors {
		if err != nil {
			log.Printf("Provider %s is unhealthy: %v", provider, err)
		} else {
			log.Printf("Provider %s is healthy", provider)
		}
	}
}