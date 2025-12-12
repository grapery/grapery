package genapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	huoshanprovider "github.com/grapestree/fgrapery/grapery/internal/genai/providers/huoshan"
)

type huoshanProvider struct {
	name   string
	client *huoshanprovider.Client
}

func newHuoshanProvider(cfg *Config) (*huoshanProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("huoshan configuration is required")
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("huoshan api key is required")
	}
	client := huoshanprovider.New(huoshanprovider.Config{
		APIKey:       apiKey,
		BaseURL:      strings.TrimSpace(cfg.BaseURL),
		ImageBaseURL: strings.TrimSpace(cfg.ImageBaseURL),
		Timeout:      cfg.Timeout,
		Workflow:     strings.TrimSpace(cfg.Workflow),
		ImageModel:   strings.TrimSpace(cfg.ImageModel),
	})
	name := strings.TrimSpace(string(cfg.Provider))
	if name == "" {
		name = string(ProviderHuoshan)
	}
	return &huoshanProvider{name: name, client: client}, nil
}

func (p *huoshanProvider) Name() string {
	return p.name
}

func (p *huoshanProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
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
		return nil, fmt.Errorf("huoshan provider does not support operation %s", req.Operation)
	}
}

func (p *huoshanProvider) GenerateImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}
	switch req.Operation {
	case OperationTextToImage:
		return p.textToImage(ctx, req)
	case OperationImageToImage:
		return p.imageToImage(ctx, req)
	default:
		return nil, fmt.Errorf("huoshan provider does not support image operation %s", req.Operation)
	}
}

func (p *huoshanProvider) GetVideoStatus(ctx context.Context, taskID string) (*GenerateResponse, error) {
	statusResp, err := p.client.GetTaskStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}

	metadata := mergeMaps(statusResp.Metadata)
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if statusResp.Usage != nil {
		metadata["usage"] = map[string]interface{}{
			"completion_tokens": statusResp.Usage.CompletionTokens,
		}
	}

	result := &GenerateResponse{
		Provider:     p.name,
		MediaType:    MediaTypeVideo,
		TaskID:       strings.TrimSpace(statusResp.TaskID),
		Status:       NormalizeStatus(statusResp.Status),
		VideoURL:     strings.TrimSpace(statusResp.VideoURL),
		ThumbnailURL: strings.TrimSpace(statusResp.Thumbnail),
		Progress:     statusResp.Progress,
		Metadata:     metadata,
		Message:      NormalizeStatus(statusResp.Status),
		Raw: map[string]interface{}{
			"status_response": statusResp,
		},
	}

	if trimmed := strings.TrimSpace(statusResp.Error); trimmed != "" {
		result.Error = trimmed
		result.Message = trimmed
	}

	if statusResp.Usage != nil {
		tokens := statusResp.Usage.CompletionTokens
		result.Usage = &Usage{
			OutputTokens: tokens,
			TotalTokens:  tokens,
			VideoCount:   1,
		}
	}

	return result, nil
}

func (p *huoshanProvider) DownloadVideo(ctx context.Context, taskID string) ([]byte, error) {
	return p.client.GetTaskAssets(ctx, taskID)
}

func (p *huoshanProvider) textToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for huoshan text to video")
	}
	payload := &huoshanprovider.TextToVideoRequest{
		Model:       strings.TrimSpace(req.Model),
		Prompt:      prompt,
		Workflow:    strings.TrimSpace(stringFromOptions(req.Options, "workflow")),
		Duration:    req.DurationSeconds,
		Resolution:  selectResolution(req),
		CallbackURL: strings.TrimSpace(req.CallbackURL),
		Metadata:    mergeMaps(req.Metadata),
	}
	resp, err := p.client.CreateTextVideo(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHuoshanTaskResponse(p.name, req, resp), nil
}

func (p *huoshanProvider) imageToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	references := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
	if len(references) == 0 {
		return nil, fmt.Errorf("at least one reference image is required for huoshan image to video")
	}
	payload := &huoshanprovider.ImageToVideoRequest{
		Model:           strings.TrimSpace(req.Model),
		Prompt:          strings.TrimSpace(req.Prompt),
		Workflow:        strings.TrimSpace(stringFromOptions(req.Options, "workflow")),
		Duration:        req.DurationSeconds,
		Resolution:      selectResolution(req),
		CallbackURL:     strings.TrimSpace(req.CallbackURL),
		ImageURL:        references[0],
		ReferenceImages: references,
		Metadata:        mergeMaps(req.Metadata),
	}
	resp, err := p.client.CreateImageVideo(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHuoshanTaskResponse(p.name, req, resp), nil
}

func (p *huoshanProvider) keyframeToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	first := strings.TrimSpace(req.FirstFrameURL)
	last := strings.TrimSpace(req.LastFrameURL)
	if first == "" || last == "" {
		return nil, fmt.Errorf("first_frame_url and last_frame_url are required for huoshan keyframe video")
	}
	payload := &huoshanprovider.KeyframeRequest{
		Model:       strings.TrimSpace(req.Model),
		FirstFrame:  first,
		LastFrame:   last,
		Prompt:      strings.TrimSpace(req.Prompt),
		Workflow:    strings.TrimSpace(stringFromOptions(req.Options, "workflow")),
		Duration:    req.DurationSeconds,
		CallbackURL: strings.TrimSpace(req.CallbackURL),
		Metadata:    mergeMaps(req.Metadata),
	}
	resp, err := p.client.CreateKeyframeVideo(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHuoshanTaskResponse(p.name, req, resp), nil
}

func (p *huoshanProvider) textToImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for huoshan text to image")
	}
	payload := &huoshanprovider.ImageGenerationRequest{
		Model:                            strings.TrimSpace(req.Model),
		Prompt:                           prompt,
		Size:                             strings.TrimSpace(req.Size),
		Seed:                             int64(req.Seed),
		Stream:                           false,
		GuidanceScale:                    req.GuidanceScale,
		ResponseFormat:                   strings.TrimSpace(req.ResponseFormat),
		Watermark:                        req.Watermark,
		SequentialImageGeneration:        strings.TrimSpace(stringFromOptions(req.Options, "sequential_image_generation")),
		SequentialImageGenerationOptions: toMapOption(req.Options, "sequential_image_generation_options"),
		OptimizePromptOptions:            toMapOption(req.Options, "optimize_prompt_options"),
	}
	resp, err := p.client.GenerateImage(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHuoshanImageResponse(p.name, req, resp), nil
}

func (p *huoshanProvider) imageToImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for huoshan image to image")
	}
	references := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
	if len(references) == 0 {
		return nil, fmt.Errorf("at least one reference image is required for huoshan image to image")
	}
	payload := &huoshanprovider.ImageGenerationRequest{
		Model:                            strings.TrimSpace(req.Model),
		Prompt:                           prompt,
		Image:                            references,
		Size:                             strings.TrimSpace(req.Size),
		Seed:                             int64(req.Seed),
		GuidanceScale:                    req.GuidanceScale,
		ResponseFormat:                   strings.TrimSpace(req.ResponseFormat),
		Watermark:                        req.Watermark,
		SequentialImageGeneration:        strings.TrimSpace(stringFromOptions(req.Options, "sequential_image_generation")),
		SequentialImageGenerationOptions: toMapOption(req.Options, "sequential_image_generation_options"),
		OptimizePromptOptions:            toMapOption(req.Options, "optimize_prompt_options"),
	}
	resp, err := p.client.GenerateImageWithReference(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHuoshanImageResponse(p.name, req, resp), nil
}

func buildHuoshanTaskResponse(provider string, req *GenerateRequest, resp *huoshanprovider.TaskResponse) *GenerateResponse {
	now := time.Now()
	metadata := mergeMaps(req.Metadata, resp.Metadata)
	return &GenerateResponse{
		Provider:    provider,
		Operation:   req.Operation,
		MediaType:   MediaTypeVideo,
		TaskID:      strings.TrimSpace(resp.TaskID),
		Status:      NormalizeStatus(resp.Status),
		Metadata:    metadata,
		Raw:         map[string]interface{}{"task_response": resp},
		StartedAt:   now,
		CompletedAt: now,
	}
}

func buildHuoshanImageResponse(provider string, req *GenerateRequest, resp *huoshanprovider.ImageGenerationResponse) *GenerateResponse {
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
		if resp.Usage != nil {
			result.Usage = &Usage{
				OutputTokens: resp.Usage.OutputTokens,
				TotalTokens:  resp.Usage.TotalTokens,
				ImageCount:   resp.Usage.GeneratedImages,
			}
		}
		if resp.Error != nil {
			result.Error = strings.TrimSpace(resp.Error.Message)
			result.ErrorCode = strings.TrimSpace(resp.Error.Code)
		}
		if resp.Created != 0 {
			result.Metadata["created"] = resp.Created
		}
		if len(resp.Data) > 0 {
			result.ImageURLs = extractHuoshanImageURLs(resp.Data)
			base64List := extractHuoshanImageBase64(resp.Data)
			if len(base64List) > 0 {
				result.Metadata["image_base64"] = base64List
			}
		}
	}
	return result
}

func extractHuoshanImageURLs(data []huoshanprovider.ImageGenerationData) []string {
	var urls []string
	for _, item := range data {
		if trimmed := strings.TrimSpace(item.URL); trimmed != "" {
			urls = append(urls, trimmed)
		}
	}
	return urls
}

func extractHuoshanImageBase64(data []huoshanprovider.ImageGenerationData) []string {
	var images []string
	for _, item := range data {
		if trimmed := strings.TrimSpace(item.B64JSON); trimmed != "" {
			images = append(images, trimmed)
		}
	}
	return images
}

func toMapOption(options map[string]interface{}, key string) map[string]interface{} {
	if len(options) == 0 {
		return nil
	}
	if value, ok := options[key]; ok {
		if m, ok := value.(map[string]interface{}); ok {
			return cloneMap(m)
		}
	}
	return nil
}
