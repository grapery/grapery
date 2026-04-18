package genapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	geminiprovider "github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
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
	case OperationKeyframeToVideo:
		return p.keyframeToVideo(ctx, req)
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
	if len(statusResp.Error) > 0 {
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
		Status:       NormalizeStatus(statusResp.State),
		VideoURL:     videoURL,
		ThumbnailURL: thumbnailURL,
		Metadata:     metadata,
		Message:      NormalizeStatus(statusResp.State),
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

	// Add Usage for completed video tasks
	if result.Status == string(StatusCompleted) && videoURL != "" {
		result.Usage = &Usage{
			VideoCount: 1,
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

	switch req.Operation {
	case OperationTextToImage:
		return p.textToImage(ctx, req)
	case OperationImageToImage:
		return p.imageToImage(ctx, req)
	default:
		// Default to text-to-image for unknown operations
		return p.textToImage(ctx, req)
	}
}

func (p *geminiProvider) textToImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for gemini text to image")
	}

	model := strings.TrimSpace(req.Model)

	// Check if the model is an Imagen model (uses different API)
	if isImagenModel(model) {
		// Use Imagen API with GenerateImagesConfig
		config := p.buildImageConfig(req)
		resp, err := p.client.GenerateImages(ctx, model, prompt, config)
		if err != nil {
			return nil, fmt.Errorf("AI image generation failed: %w", err)
		}
		return buildGeminiImageResponse(p.name, req, resp), nil
	}

	// Use conversational image generation API for Gemini models
	opts := p.buildConversationalImageOptions(req)
	images, resp, err := p.client.GenerateConversationalImageFromText(ctx, model, prompt, opts)
	if err != nil {
		return nil, err
	}

	return buildGeminiConversationalImageResponse(p.name, req, images, resp), nil
}

// isImagenModel checks if the model name indicates an Imagen model
func isImagenModel(model string) bool {
	model = strings.ToLower(model)
	return strings.HasPrefix(model, "imagen") || strings.Contains(model, "imagen")
}

func (p *geminiProvider) imageToImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for gemini image to image")
	}

	model := strings.TrimSpace(req.Model)
	opts := p.buildConversationalImageOptions(req)

	// Multi-image reference: use ComposeImages when ReferenceImagesData has one or more images
	if len(req.ReferenceImagesData) >= 1 {
		assets := make([]geminiprovider.ImageAsset, 0, len(req.ReferenceImagesData))
		for _, a := range req.ReferenceImagesData {
			if len(a.Data) == 0 {
				continue
			}
			mime := strings.TrimSpace(a.MIMEType)
			if mime == "" {
				mime = "image/png"
			}
			assets = append(assets, geminiprovider.ImageAsset{Data: a.Data, MimeType: mime})
		}
		if len(assets) == 0 {
			return nil, fmt.Errorf("reference images data is required for multi-image compose")
		}
		images, resp, err := p.client.ComposeImages(ctx, model, assets, prompt, opts)
		if err != nil {
			return nil, err
		}
		return buildGeminiConversationalImageResponse(p.name, req, images, resp), nil
	}

	// Single image edit (text + image -> image)
	var imageData []byte
	var mimeType string
	if len(req.ImageData) > 0 {
		imageData = req.ImageData
		mimeType = strings.TrimSpace(req.ImageMIMEType)
		if mimeType == "" {
			mimeType = "image/png"
		}
	} else {
		imageURL := firstNonEmpty(req.ReferenceImageURL)
		if imageURL == "" && len(req.ReferenceImages) > 0 {
			imageURL = strings.TrimSpace(req.ReferenceImages[0])
		}
		if imageURL != "" {
			return nil, fmt.Errorf("gemini image-to-image requires ImageData bytes; URL-based images not supported directly - please fetch the image first")
		}
		return nil, fmt.Errorf("image data is required for gemini image to image")
	}

	images, resp, err := p.client.EditImageWithPrompt(ctx, model, imageData, mimeType, prompt, opts)
	if err != nil {
		return nil, err
	}

	return buildGeminiConversationalImageResponse(p.name, req, images, resp), nil
}

func (p *geminiProvider) buildConversationalImageOptions(req *GenerateRequest) *geminiprovider.ImageGenerationOptions {
	opts := &geminiprovider.ImageGenerationOptions{}

	if trimmed := strings.TrimSpace(req.AspectRatio); trimmed != "" {
		opts.AspectRatio = trimmed
	}
	if trimmed := strings.TrimSpace(req.Size); trimmed != "" {
		opts.ImageSize = trimmed // e.g. "512px", "1K", "2K", "4K"
	}

	return opts
}

func (p *geminiProvider) textToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for gemini text to video")
	}

	config := p.buildVideoConfig(req)
	// When using reference images, API does not allow image or last_frame in source
	source := &genai.GenerateVideosSource{Prompt: prompt}
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
		image = createGeminiImageFromURL(trimmed)
	} else if len(req.ReferenceImages) > 0 && strings.TrimSpace(req.ReferenceImages[0]) != "" {
		image = createGeminiImageFromURL(strings.TrimSpace(req.ReferenceImages[0]))
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

func (p *geminiProvider) keyframeToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for gemini keyframe to video")
	}

	var firstFrame *genai.Image
	if len(req.FirstFrameData) > 0 {
		mime := strings.TrimSpace(req.FirstFrameMIMEType)
		if mime == "" {
			mime = "image/png"
		}
		firstFrame = &genai.Image{
			ImageBytes: append([]byte(nil), req.FirstFrameData...),
			MIMEType:   mime,
		}
	} else if u := firstNonEmpty(req.FirstFrameURL); u != "" {
		firstFrame = createGeminiImageFromURL(u)
	}
	if firstFrame == nil {
		return nil, fmt.Errorf("first frame (FirstFrameData or FirstFrameURL) is required for keyframe to video")
	}

	config := p.buildVideoConfig(req)
	// LastFrame is set in buildVideoConfig when LastFrameData is present
	// When using keyframe, do not set ReferenceImages (API forbids combining)
	config.ReferenceImages = nil

	source := &genai.GenerateVideosSource{
		Prompt: prompt,
		Image:  firstFrame,
	}
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
		config.AspectRatio = trimmed // "16:9" or "9:16"
	}
	if trimmed := strings.TrimSpace(req.Resolution); trimmed != "" {
		config.Resolution = trimmed // "720p", "1080p", "4k"
	}
	hasRefImages := len(req.ReferenceImagesData) > 0
	// LastFrame for keyframe-to-video (first frame is in source.Image).
	// API: when ReferenceImages is set, image/last_frame are not allowed.
	if !hasRefImages && len(req.LastFrameData) > 0 {
		mime := strings.TrimSpace(req.LastFrameMIMEType)
		if mime == "" {
			mime = "image/png"
		}
		config.LastFrame = &genai.Image{
			ImageBytes: append([]byte(nil), req.LastFrameData...),
			MIMEType:   mime,
		}
	}
	// Reference images (up to 3 for Veo 3.1) - mutually exclusive with image/last_frame
	if hasRefImages {
		n := len(req.ReferenceImagesData)
		if n > 3 {
			n = 3
		}
		config.ReferenceImages = make([]*genai.VideoGenerationReferenceImage, 0, n)
		for i := 0; i < n; i++ {
			a := req.ReferenceImagesData[i]
			mime := strings.TrimSpace(a.MIMEType)
			if mime == "" {
				mime = "image/png"
			}
			config.ReferenceImages = append(config.ReferenceImages, &genai.VideoGenerationReferenceImage{
				Image: &genai.Image{
					ImageBytes: append([]byte(nil), a.Data...),
					MIMEType:   mime,
				},
				ReferenceType: genai.VideoGenerationReferenceTypeAsset,
			})
		}
	}

	return config
}

func buildGeminiImageResponse(provider string, req *GenerateRequest, resp *genai.GenerateImagesResponse) *GenerateResponse {
	now := time.Now()
	result := &GenerateResponse{
		Provider:    provider,
		Operation:   req.Operation,
		MediaType:   MediaTypeImage,
		Status:      string(StatusCompleted),
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

func buildGeminiConversationalImageResponse(provider string, req *GenerateRequest, images [][]byte, resp *genai.GenerateContentResponse) *GenerateResponse {
	now := time.Now()
	result := &GenerateResponse{
		Provider:    provider,
		Operation:   req.Operation,
		MediaType:   MediaTypeImage,
		Status:      string(StatusCompleted),
		Metadata:    make(map[string]interface{}),
		Raw:         map[string]interface{}{"content_response": resp},
		StartedAt:   now,
		CompletedAt: now,
	}

	if len(images) > 0 {
		result.Metadata["image_bytes"] = images
		result.Usage = &Usage{
			ImageCount: len(images),
		}
	}

	// Extract usage from response if available
	if resp != nil && resp.UsageMetadata != nil {
		if result.Usage == nil {
			result.Usage = &Usage{}
		}
		result.Usage.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
		result.Usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
		result.Usage.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
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
		Status:      NormalizeStatus(resp.State),
		Metadata:    metadata,
		Raw:         map[string]interface{}{"video_response": resp},
		StartedAt:   now,
		CompletedAt: now,
	}

	return result
}

// createGeminiImageFromURL determines the appropriate image reference type based on URL format.
// GCS URIs (gs://) are passed as GCSURI, HTTP/HTTPS URLs need to be fetched and passed as bytes.
func createGeminiImageFromURL(url string) *genai.Image {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}

	// GCS URI format: gs://bucket-name/path/to/file
	if strings.HasPrefix(url, "gs://") {
		return &genai.Image{
			GCSURI: url,
		}
	}

	// For HTTP/HTTPS URLs, we use FileURI if supported by the SDK
	// Otherwise, the caller should fetch the image and pass bytes
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		// Note: The official SDK may support FileURI for HTTP URLs
		// If not, you'll need to fetch the image bytes first
		return &genai.Image{
			GCSURI: url, // Try as URI first; may need adjustment based on SDK behavior
		}
	}

	// Assume it's a GCS URI if no recognized prefix
	return &genai.Image{
		GCSURI: url,
	}
}
