package genapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	geminiprovider "github.com/grapery/grapery/pkg/genapi/providers/gemini"
	"google.golang.org/genai"
)

type geminiProvider struct {
	name   string
	client *geminiprovider.Client
	config *Config
}

func newGeminiProvider(cfg *Config) (*geminiProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("gemini configuration is required")
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}

	client, err := geminiprovider.New(geminiprovider.Config{
		APIKey:       apiKey,
		BaseURL:      strings.TrimSpace(cfg.BaseURL),
		Timeout:      cfg.Timeout,
		DefaultModel: strings.TrimSpace(cfg.Model),
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}

	name := strings.TrimSpace(string(cfg.Provider))
	if name == "" {
		name = string(ProviderGemini)
	}

	return &geminiProvider{
		name:   name,
		client: client,
		config: cfg,
	}, nil
}

func (p *geminiProvider) Name() string {
	return p.name
}

func (p *geminiProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}
	switch req.Operation {
	case OperationTextToVideo:
		return p.textToVideo(ctx, req)
	case OperationImageToVideo:
		return p.imageToVideo(ctx, req)
	default:
		return nil, fmt.Errorf("gemini provider does not support operation %s", req.Operation)
	}
}

func (p *geminiProvider) GetVideoStatus(ctx context.Context, taskID string) (*GenerateResponse, error) {
	statusResp, err := p.client.GetVideoStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}

	metadata := cloneMap(statusResp.Metadata)
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if len(statusResp.Assets) > 0 {
		metadata["assets"] = statusResp.Assets
	}
	if statusResp.Error != nil && len(statusResp.Error) > 0 {
		metadata["error"] = statusResp.Error
	}

	videoURL := ""
	thumbnailURL := ""
	for _, asset := range statusResp.Assets {
		uri := strings.TrimSpace(asset.URI)
		if uri == "" {
			continue
		}
		assetType := strings.ToLower(strings.TrimSpace(asset.Type))
		if videoURL == "" && (strings.Contains(assetType, "video") || strings.HasSuffix(strings.ToLower(uri), ".mp4")) {
			videoURL = uri
			continue
		}
		if thumbnailURL == "" && (strings.Contains(assetType, "image") || strings.Contains(assetType, "thumbnail")) {
			thumbnailURL = uri
		}
	}
	if videoURL == "" && len(statusResp.Assets) > 0 {
		videoURL = strings.TrimSpace(statusResp.Assets[0].URI)
	}

	result := &GenerateResponse{
		Provider:     p.name,
		MediaType:    MediaTypeVideo,
		TaskID:       strings.TrimSpace(statusResp.Name),
		Status:       strings.TrimSpace(statusResp.State),
		VideoURL:     videoURL,
		ThumbnailURL: thumbnailURL,
		Metadata:     metadata,
		Message:      strings.TrimSpace(statusResp.State),
		Raw: map[string]interface{}{
			"status_response": statusResp,
		},
	}

	if statusResp.Error != nil {
		if msg, ok := statusResp.Error["message"].(string); ok {
			if trimmed := strings.TrimSpace(msg); trimmed != "" {
				result.Error = trimmed
				result.Message = trimmed
			}
		}
		if code, ok := statusResp.Error["code"]; ok {
			if str := strings.TrimSpace(fmt.Sprint(code)); str != "" {
				result.ErrorCode = str
			}
		}
	}

	return result, nil
}

func (p *geminiProvider) DownloadVideo(ctx context.Context, taskID string) ([]byte, error) {
	return p.client.DownloadVideo(ctx, taskID)
}

func (p *geminiProvider) GenerateImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for gemini image generation")
	}

	model := strings.TrimSpace(req.Model)
	config := p.buildImageConfig(req)

	resp, err := p.client.GenerateImages(ctx, model, prompt, config)
	if err != nil {
		return nil, err
	}

	return buildGeminiImageResponse(p.name, req, resp), nil
}

func (p *geminiProvider) textToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for gemini text to video")
	}

	source := &genai.GenerateVideosSource{
		Prompt: prompt,
	}

	config := p.buildVideoConfig(req)
	model := strings.TrimSpace(req.Model)

	resp, err := p.client.GenerateVideo(ctx, model, source, config)
	if err != nil {
		return nil, err
	}

	return buildGeminiVideoResponse(p.name, req, resp), nil
}

func (p *geminiProvider) imageToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	var image *genai.Image

	if len(req.ImageData) > 0 {
		mimeType := strings.TrimSpace(req.ImageMIMEType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		image = &genai.Image{
			ImageBytes: append([]byte(nil), req.ImageData...),
			MIMEType:   mimeType,
		}
	} else if trimmed := strings.TrimSpace(req.ReferenceImageURL); trimmed != "" {
		image = &genai.Image{
			GCSURI: trimmed,
		}
	} else if len(req.ReferenceImages) > 0 && strings.TrimSpace(req.ReferenceImages[0]) != "" {
		image = &genai.Image{
			GCSURI: strings.TrimSpace(req.ReferenceImages[0]),
		}
	}

	if image == nil {
		return nil, fmt.Errorf("image is required for gemini image to video")
	}

	source := &genai.GenerateVideosSource{
		Prompt: strings.TrimSpace(req.Prompt),
		Image:  image,
	}

	config := p.buildVideoConfig(req)
	model := strings.TrimSpace(req.Model)

	resp, err := p.client.GenerateVideo(ctx, model, source, config)
	if err != nil {
		return nil, err
	}

	return buildGeminiVideoResponse(p.name, req, resp), nil
}

func (p *geminiProvider) buildImageConfig(req *GenerateRequest) *genai.GenerateImagesConfig {
	config := &genai.GenerateImagesConfig{}

	if trimmed := strings.TrimSpace(req.NegativePrompt); trimmed != "" {
		config.NegativePrompt = trimmed
	}

	if req.OutputCount > 0 {
		config.NumberOfImages = int32(req.OutputCount)
	}

	if trimmed := strings.TrimSpace(req.AspectRatio); trimmed != "" {
		config.AspectRatio = trimmed
	}

	if req.GuidanceScale > 0 {
		value := float32(req.GuidanceScale)
		config.GuidanceScale = &value
	}

	if req.Seed != 0 {
		value := int32(req.Seed)
		config.Seed = &value
	}

	if trimmed := strings.TrimSpace(req.ResponseFormat); trimmed != "" {
		config.OutputMIMEType = trimmed
	}

	if req.Watermark != nil {
		config.AddWatermark = *req.Watermark
	}

	return config
}

func (p *geminiProvider) buildVideoConfig(req *GenerateRequest) *genai.GenerateVideosConfig {
	config := &genai.GenerateVideosConfig{}

	if trimmed := strings.TrimSpace(req.AspectRatio); trimmed != "" {
		config.AspectRatio = trimmed
	}

	return config
}

func buildGeminiImageResponse(provider string, req *GenerateRequest, resp *genai.GenerateImagesResponse) *GenerateResponse {
	now := time.Now()
	result := &GenerateResponse{
		Provider:    provider,
		Operation:   req.Operation,
		MediaType:   MediaTypeImage,
		Status:      "completed",
		Metadata:    make(map[string]interface{}),
		Raw:         map[string]interface{}{"image_response": resp},
		StartedAt:   now,
		CompletedAt: now,
	}

	if resp != nil {
		for _, generated := range resp.GeneratedImages {
			if generated == nil || generated.Image == nil {
				continue
			}
			if gcsURI := strings.TrimSpace(generated.Image.GCSURI); gcsURI != "" {
				result.ImageURLs = append(result.ImageURLs, gcsURI)
			}
			if len(generated.Image.ImageBytes) > 0 {
				if result.Metadata["image_bytes"] == nil {
					result.Metadata["image_bytes"] = [][]byte{}
				}
				result.Metadata["image_bytes"] = append(result.Metadata["image_bytes"].([][]byte), generated.Image.ImageBytes)
			}
		}

		if len(resp.GeneratedImages) > 0 {
			result.Usage = &Usage{
				ImageCount: len(resp.GeneratedImages),
			}
		}
	}

	return result
}

func buildGeminiVideoResponse(provider string, req *GenerateRequest, resp *geminiprovider.GenerateVideoResponse) *GenerateResponse {
	now := time.Now()
	metadata := make(map[string]interface{})
	if resp.Metadata != nil {
		metadata = cloneMap(resp.Metadata)
	}

	result := &GenerateResponse{
		Provider:    provider,
		Operation:   req.Operation,
		MediaType:   MediaTypeVideo,
		TaskID:      strings.TrimSpace(resp.Name),
		Status:      strings.TrimSpace(resp.State),
		Metadata:    metadata,
		Raw:         map[string]interface{}{"video_response": resp},
		StartedAt:   now,
		CompletedAt: now,
	}

	return result
}
