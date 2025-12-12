package genapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	hailuoprovider "github.com/grapestree/fgrapery/grapery/internal/genai/providers/hailuo"
)

type hailuoProvider struct {
	name   string
	client *hailuoprovider.Client
}

func newHailuoProvider(cfg *Config) (*hailuoProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("hailuo configuration is required")
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("hailuo api key is required")
	}
	client := hailuoprovider.New(hailuoprovider.Config{
		APIKey:     apiKey,
		BaseURL:    strings.TrimSpace(cfg.BaseURL),
		Timeout:    cfg.Timeout,
		Model:      strings.TrimSpace(cfg.Model),
		ImageModel: strings.TrimSpace(cfg.ImageModel),
	})
	name := strings.TrimSpace(string(cfg.Provider))
	if name == "" {
		name = string(ProviderHailuo)
	}
	return &hailuoProvider{name: name, client: client}, nil
}

func (p *hailuoProvider) Name() string {
	return p.name
}

func (p *hailuoProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
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
	case OperationStoryboardToVideo:
		return p.storyboardToVideo(ctx, req)
	default:
		return nil, fmt.Errorf("hailuo provider does not support operation %s", req.Operation)
	}
}

func (p *hailuoProvider) GetVideoStatus(ctx context.Context, taskID string) (*GenerateResponse, error) {
	statusResp, err := p.client.GetTaskStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	if trimmed := strings.TrimSpace(statusResp.FileID); trimmed != "" {
		metadata["file_id"] = trimmed
	}
	if statusResp.VideoWidth > 0 {
		metadata["video_width"] = statusResp.VideoWidth
	}
	if statusResp.VideoHeight > 0 {
		metadata["video_height"] = statusResp.VideoHeight
	}
	if statusResp.BaseResponse.StatusCode != 0 {
		metadata["status_code"] = statusResp.BaseResponse.StatusCode
	}
	if trimmed := strings.TrimSpace(statusResp.BaseResponse.StatusMsg); trimmed != "" {
		metadata["status_msg"] = trimmed
	}

	result := &GenerateResponse{
		Provider:  p.name,
		MediaType: MediaTypeVideo,
		TaskID:    strings.TrimSpace(statusResp.TaskID),
		Status:    NormalizeStatus(statusResp.Status),
		Metadata:  metadata,
		Raw: map[string]interface{}{
			"status_response": statusResp,
		},
	}

	if trimmed := strings.TrimSpace(statusResp.BaseResponse.StatusMsg); trimmed != "" {
		result.Message = trimmed
		if statusResp.BaseResponse.StatusCode != 0 || result.Status == string(StatusFailed) {
			result.Error = trimmed
		}
	}
	if statusResp.BaseResponse.StatusCode != 0 {
		result.ErrorCode = fmt.Sprintf("%d", statusResp.BaseResponse.StatusCode)
	}

	// Add Usage for completed video tasks
	if result.Status == string(StatusCompleted) {
		result.Usage = &Usage{
			VideoCount: 1,
		}
		// Add video dimensions if available
		if statusResp.VideoWidth > 0 && statusResp.VideoHeight > 0 {
			result.Usage.Additional = map[string]interface{}{
				"video_width":  statusResp.VideoWidth,
				"video_height": statusResp.VideoHeight,
			}
		}
	}

	return result, nil
}

func (p *hailuoProvider) DownloadVideo(ctx context.Context, taskID string) ([]byte, error) {
	return p.client.Download(ctx, taskID)
}

func (p *hailuoProvider) GenerateImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}
	if req.Operation != OperationTextToImage {
		return nil, fmt.Errorf("hailuo provider only supports text_to_image operation")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for hailuo image generation")
	}
	payload := &hailuoprovider.ImageGenerationRequest{
		Model:          strings.TrimSpace(req.Model),
		Prompt:         prompt,
		AspectRatio:    strings.TrimSpace(req.AspectRatio),
		Width:          req.Width,
		Height:         req.Height,
		ResponseFormat: strings.TrimSpace(req.ResponseFormat),
		Seed:           req.Seed,
		N:              req.OutputCount,
	}
	if value, ok := boolFromOptions(req.Options, "prompt_optimizer"); ok {
		payload.PromptOptimizer = &value
	}
	resp, err := p.client.GenerateImage(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHailuoImageResponse(p.name, req, resp), nil
}

func (p *hailuoProvider) textToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for hailuo text to video")
	}
	payload := &hailuoprovider.TextToVideoRequest{
		Prompt:      prompt,
		Model:       strings.TrimSpace(req.Model),
		Duration:    req.DurationSeconds,
		Resolution:  selectResolution(req),
		CallbackURL: strings.TrimSpace(req.CallbackURL),
		Metadata:    cloneMap(req.Metadata),
		StylePreset: strings.TrimSpace(req.Style),
	}
	if req.Watermark != nil {
		payload.AIGCWatermark = req.Watermark
	}
	if value, ok := boolFromOptions(req.Options, "prompt_optimizer"); ok {
		payload.PromptOptimizer = &value
	}
	if value, ok := boolFromOptions(req.Options, "fast_pretreatment"); ok {
		payload.FastPretreatment = &value
	}
	resp, err := p.client.SubmitTextToVideo(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHailuoTaskResponse(p.name, req, resp), nil
}

func (p *hailuoProvider) imageToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	image := firstNonEmpty(req.ReferenceImageURL)
	if image == "" && len(req.ReferenceImages) > 0 {
		image = strings.TrimSpace(req.ReferenceImages[0])
	}
	if image == "" {
		return nil, fmt.Errorf("first frame image is required for hailuo image to video")
	}
	payload := &hailuoprovider.ImageToVideoRequest{
		FirstFrameImage: strings.TrimSpace(image),
		Prompt:          strings.TrimSpace(req.Prompt),
		Model:           strings.TrimSpace(req.Model),
		Duration:        req.DurationSeconds,
		Resolution:      selectResolution(req),
		CallbackURL:     strings.TrimSpace(req.CallbackURL),
		Metadata:        cloneMap(req.Metadata),
	}
	if req.Watermark != nil {
		payload.AIGCWatermark = req.Watermark
	}
	if value, ok := boolFromOptions(req.Options, "prompt_optimizer"); ok {
		payload.PromptOptimizer = &value
	}
	if value, ok := boolFromOptions(req.Options, "fast_pretreatment"); ok {
		payload.FastPretreatment = &value
	}
	resp, err := p.client.SubmitImageToVideo(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHailuoTaskResponse(p.name, req, resp), nil
}

func (p *hailuoProvider) keyframeToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	first := strings.TrimSpace(req.FirstFrameURL)
	last := strings.TrimSpace(req.LastFrameURL)
	if first == "" || last == "" {
		return nil, fmt.Errorf("first_frame_url and last_frame_url are required for hailuo keyframe video")
	}
	payload := &hailuoprovider.FirstLastFrameRequest{
		FirstFrameImage: first,
		LastFrameImage:  last,
		Prompt:          strings.TrimSpace(req.Prompt),
		Model:           strings.TrimSpace(req.Model),
		Duration:        req.DurationSeconds,
		Resolution:      selectResolution(req),
		CallbackURL:     strings.TrimSpace(req.CallbackURL),
		Metadata:        cloneMap(req.Metadata),
	}
	if req.Watermark != nil {
		payload.AIGCWatermark = req.Watermark
	}
	if value, ok := boolFromOptions(req.Options, "prompt_optimizer"); ok {
		payload.PromptOptimizer = &value
	}
	if value, ok := boolFromOptions(req.Options, "fast_pretreatment"); ok {
		payload.FastPretreatment = &value
	}
	resp, err := p.client.SubmitFirstLastFrame(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHailuoTaskResponse(p.name, req, resp), nil
}

func (p *hailuoProvider) storyboardToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if len(req.Storyboard) == 0 {
		return nil, fmt.Errorf("storyboard is required for hailuo storyboard to video")
	}
	payload := &hailuoprovider.StoryboardToVideoRequest{
		Storyboard:  cloneMap(req.Storyboard),
		Model:       strings.TrimSpace(req.Model),
		CallbackURL: strings.TrimSpace(req.CallbackURL),
		Metadata:    cloneMap(req.Metadata),
	}
	resp, err := p.client.SubmitStoryboard(ctx, payload)
	if err != nil {
		return nil, err
	}
	return buildHailuoTaskResponse(p.name, req, resp), nil
}

func buildHailuoTaskResponse(provider string, req *GenerateRequest, resp *hailuoprovider.TaskResponse) *GenerateResponse {
	now := time.Now()
	metadata := mergeMaps(req.Metadata, resp.Metadata)
	base := &GenerateResponse{
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
	return base
}

func buildHailuoImageResponse(provider string, req *GenerateRequest, resp *hailuoprovider.ImageGenerationResponse) *GenerateResponse {
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
		imageCount := 0
		if resp.Data != nil {
			if len(resp.Data.ImageURLs) > 0 {
				for _, url := range resp.Data.ImageURLs {
					if trimmed := strings.TrimSpace(url); trimmed != "" {
						result.ImageURLs = append(result.ImageURLs, trimmed)
						imageCount++
					}
				}
			}
			if len(resp.Data.ImageBase64) > 0 {
				result.Metadata["image_base64"] = append([]string(nil), resp.Data.ImageBase64...)
				if imageCount == 0 {
					imageCount = len(resp.Data.ImageBase64)
				}
			}
		}
		if resp.Metadata != nil {
			result.Metadata["generation_metadata"] = resp.Metadata
			// Extract success/failed counts from metadata
			if resp.Metadata.SuccessCount != "" {
				result.Metadata["success_count"] = resp.Metadata.SuccessCount
			}
			if resp.Metadata.FailedCount != "" {
				result.Metadata["failed_count"] = resp.Metadata.FailedCount
			}
		}
		if resp.BaseResponse.StatusCode != 0 {
			result.Metadata["status_code"] = resp.BaseResponse.StatusCode
			result.Metadata["status_msg"] = resp.BaseResponse.StatusMsg
			result.Error = resp.BaseResponse.StatusMsg
			result.ErrorCode = fmt.Sprintf("%d", resp.BaseResponse.StatusCode)
			result.Status = string(StatusFailed)
		}
		if resp.ID != "" {
			result.TaskID = resp.ID
		}
		// Set Usage
		if imageCount > 0 {
			result.Usage = &Usage{
				ImageCount: imageCount,
			}
		}
	}
	return result
}

func selectResolution(req *GenerateRequest) string {
	if trimmed := strings.TrimSpace(req.Resolution); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(req.AspectRatio)
}
