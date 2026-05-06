package genapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	qwenprovider "github.com/grapestree/fgrapery/grapery/internal/genai/providers/qwen"
)

type qwenProvider struct {
	name   string
	client *qwenprovider.Client
	config *Config
}

func newQwenProvider(cfg *Config) (*qwenProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("qwen configuration is required")
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("qwen api key is required")
	}

	client := qwenprovider.New(qwenprovider.Config{
		APIKey:       apiKey,
		BaseURL:      strings.TrimSpace(cfg.BaseURL),
		Timeout:      cfg.Timeout,
		DefaultModel: strings.TrimSpace(cfg.Model),
	})

	name := strings.TrimSpace(string(cfg.Provider))
	if name == "" {
		name = string(ProviderQwen)
	}

	return &qwenProvider{
		name:   name,
		client: client,
		config: cfg,
	}, nil
}

func (p *qwenProvider) Name() string {
	return p.name
}

func (p *qwenProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
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
	case OperationVideoEdit:
		return p.videoEdit(ctx, req)
	default:
		return nil, fmt.Errorf("qwen provider does not support video operation %s", req.Operation)
	}
}

func (p *qwenProvider) GenerateImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}
	switch req.Operation {
	case OperationTextToImage:
		return p.textToImage(ctx, req)
	case OperationImageToImage:
		return p.imageToImage(ctx, req)
	default:
		return nil, fmt.Errorf("qwen provider does not support image operation %s", req.Operation)
	}
}

func (p *qwenProvider) GetVideoStatus(ctx context.Context, taskID string) (*GenerateResponse, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}

	// DashScope video-synthesis (HappyHorse, Wan I2V, etc.) returns nested output on GET /tasks/{id}.
	syn, synErr := p.client.GetImageToVideoTask(ctx, taskID)
	if synErr == nil && syn != nil && syn.Output != nil {
		if tid := strings.TrimSpace(syn.Output.TaskID); tid != "" || strings.TrimSpace(syn.Output.TaskStatus) != "" {
			return buildQwenVideoPollFromSynthesis(p.name, syn), nil
		}
	}

	statusResp, err := p.client.GetTaskStatus(ctx, taskID)
	if err != nil {
		if synErr != nil {
			return nil, fmt.Errorf("qwen video status: dashscope task: %v; legacy: %w", synErr, err)
		}
		return nil, err
	}

	metadata := make(map[string]interface{})
	if statusResp.Usage != nil {
		metadata["usage"] = statusResp.Usage
	}
	if statusResp.Progress > 0 {
		metadata["progress"] = statusResp.Progress
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
		Message:      strings.TrimSpace(statusResp.Status),
		Raw: map[string]interface{}{
			"status_response": statusResp,
		},
	}

	if trimmed := strings.TrimSpace(statusResp.Error); trimmed != "" {
		result.Error = trimmed
		result.Message = trimmed
	}

	if result.Status == string(StatusCompleted) && result.VideoURL != "" {
		result.Usage = &Usage{
			VideoCount: 1,
		}
	}

	return result, nil
}

func (p *qwenProvider) DownloadVideo(ctx context.Context, taskID string) ([]byte, error) {
	return p.client.DownloadRendered(ctx, taskID)
}

func (p *qwenProvider) GetImageStatus(ctx context.Context, taskID string) (*GenerateResponse, error) {
	statusResp, err := p.client.GetTextToImageTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	result := &GenerateResponse{
		Provider:  p.name,
		MediaType: MediaTypeImage,
		Metadata:  make(map[string]interface{}),
		Raw: map[string]interface{}{
			"status_response": statusResp,
		},
	}

	if statusResp.RequestID != "" {
		result.Metadata["request_id"] = statusResp.RequestID
	}

	if statusResp.Code != "" {
		result.ErrorCode = statusResp.Code
		result.Error = statusResp.Message
		result.Status = string(StatusFailed)
	}

	if statusResp.Output != nil {
		result.TaskID = strings.TrimSpace(statusResp.Output.TaskID)
		result.Status = NormalizeStatus(statusResp.Output.TaskStatus)
		result.Message = strings.TrimSpace(statusResp.Output.Message)

		if statusResp.Output.Code != "" {
			result.ErrorCode = statusResp.Output.Code
			result.Error = statusResp.Output.Message
		}

		for _, r := range statusResp.Output.Results {
			if trimmed := strings.TrimSpace(r.URL); trimmed != "" {
				result.ImageURLs = append(result.ImageURLs, trimmed)
			}
		}

		if statusResp.Output.TaskMetrics != nil {
			result.Metadata["task_metrics"] = statusResp.Output.TaskMetrics
			result.Usage = &Usage{
				ImageCount: statusResp.Output.TaskMetrics.Succeeded,
			}
		}
	}

	if statusResp.Usage != nil {
		if result.Usage == nil {
			result.Usage = &Usage{}
		}
		result.Usage.ImageCount = statusResp.Usage.ImageCount
	}

	return result, nil
}

func (p *qwenProvider) textToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for qwen text to video")
	}

	model := resolveQwenVideoModel(p, req)
	if qwenprovider.IsHappyHorseModel(model) {
		if pipe := qwenprovider.HappyHorsePipeline(model); pipe != "t2v" {
			return nil, fmt.Errorf("model %q is not a HappyHorse text-to-video model; use %s", model, qwenprovider.ModelHappyHorseT2V)
		}
		model = qwenprovider.ResolveHappyHorseModelID(model, "t2v")
		resolution := qwenprovider.NormalizeHappyHorseResolution(strings.TrimSpace(req.Resolution))
		if strings.TrimSpace(req.Resolution) == "" && strings.TrimSpace(req.Quality) != "" {
			resolution = qwenprovider.NormalizeHappyHorseResolution(strings.TrimSpace(req.Quality))
		}
		ratio := strings.TrimSpace(req.AspectRatio)
		if ratio == "" {
			ratio = "16:9"
		}
		dur := qwenprovider.ClampHappyHorseDuration(req.DurationSeconds, 5)
		payload := &qwenprovider.HappyHorseT2VPayload{
			Model: model,
			Input: qwenprovider.HappyHorseT2VInput{Prompt: prompt},
			Parameters: &qwenprovider.HappyHorseT2VParams{
				Resolution: resolution,
				Ratio:      ratio,
				Duration:   dur,
				Watermark:  req.Watermark,
			},
		}
		if req.Seed != 0 {
			s := int(req.Seed)
			payload.Parameters.Seed = &s
		}
		resp, err := p.client.StartHappyHorseVideoSynthesis(ctx, payload)
		if err != nil {
			return nil, err
		}
		return buildQwenVideoSynthesisStartResponse(p.name, req, resp), nil
	}

	payload := &qwenprovider.TextToVideoRequest{
		Prompt:      prompt,
		Model:       model,
		AspectRatio: strings.TrimSpace(req.AspectRatio),
		Duration:    req.DurationSeconds,
		Quality:     strings.TrimSpace(req.Quality),
		Style:       strings.TrimSpace(req.Style),
		CallbackURL: strings.TrimSpace(req.CallbackURL),
		Metadata:    cloneMap(req.Metadata),
	}

	resp, err := p.client.StartTextToVideo(ctx, payload)
	if err != nil {
		return nil, err
	}

	return buildQwenVideoResponse(p.name, req, resp), nil
}

func (p *qwenProvider) imageToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	imageURL := firstNonEmpty(req.ReferenceImageURL)
	if imageURL == "" && len(req.ReferenceImages) > 0 {
		imageURL = strings.TrimSpace(req.ReferenceImages[0])
	}
	if imageURL == "" {
		return nil, fmt.Errorf("image_url is required for qwen image to video")
	}

	model := resolveQwenVideoModel(p, req)
	if qwenprovider.IsHappyHorseModel(model) {
		pipe := qwenprovider.HappyHorsePipeline(model)
		if pipe == "t2v" {
			return nil, fmt.Errorf("model %q is text-only; use %s for image-to-video", model, qwenprovider.ModelHappyHorseI2V)
		}
		if pipe == "video_edit" {
			return nil, fmt.Errorf("use OperationVideoEdit with %s", qwenprovider.ModelHappyHorseVideoEdit)
		}
		model = qwenprovider.ResolveHappyHorseModelID(model, "i2v")
		resolution := qwenprovider.NormalizeHappyHorseResolution(strings.TrimSpace(req.Resolution))
		if strings.TrimSpace(req.Resolution) == "" && strings.TrimSpace(req.Quality) != "" {
			resolution = qwenprovider.NormalizeHappyHorseResolution(strings.TrimSpace(req.Quality))
		}
		dur := qwenprovider.ClampHappyHorseDuration(req.DurationSeconds, 5)
		payload := &qwenprovider.HappyHorseI2VPayload{
			Model: model,
			Input: qwenprovider.HappyHorseI2VInput{
				Prompt: strings.TrimSpace(req.Prompt),
				Media: []qwenprovider.HappyHorseI2VMedia{
					{Type: "first_frame", URL: imageURL},
				},
			},
			Parameters: &qwenprovider.HappyHorseI2VParams{
				Resolution: resolution,
				Duration:   dur,
				Watermark:  req.Watermark,
			},
		}
		if req.Seed != 0 {
			s := int(req.Seed)
			payload.Parameters.Seed = &s
		}
		resp, err := p.client.StartHappyHorseVideoSynthesis(ctx, payload)
		if err != nil {
			return nil, err
		}
		return buildQwenVideoSynthesisStartResponse(p.name, req, resp), nil
	}

	payload := &qwenprovider.ImageToVideoRequest{
		ImageURL:    imageURL,
		Prompt:      strings.TrimSpace(req.Prompt),
		Model:       model,
		AspectRatio: strings.TrimSpace(req.AspectRatio),
		Duration:    req.DurationSeconds,
		CallbackURL: strings.TrimSpace(req.CallbackURL),
		Metadata:    cloneMap(req.Metadata),
	}

	resp, err := p.client.StartImageToVideo(ctx, payload)
	if err != nil {
		return nil, err
	}

	return buildQwenVideoResponse(p.name, req, resp), nil
}

func (p *qwenProvider) keyframeToVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	model := resolveQwenVideoModel(p, req)
	if qwenprovider.IsHappyHorseModel(model) {
		return nil, fmt.Errorf("HappyHorse models do not support first-last-frame on DashScope API; choose another video provider or non-HappyHorse model")
	}

	first := strings.TrimSpace(req.FirstFrameURL)
	last := strings.TrimSpace(req.LastFrameURL)
	if first == "" || last == "" {
		return nil, fmt.Errorf("first_frame_url and last_frame_url are required for qwen keyframe video")
	}

	payload := &qwenprovider.FirstLastFrameRequest{
		FirstFrameURL: first,
		LastFrameURL:  last,
		Prompt:        strings.TrimSpace(req.Prompt),
		Model:         model,
		Duration:      req.DurationSeconds,
		CallbackURL:   strings.TrimSpace(req.CallbackURL),
		Metadata:      cloneMap(req.Metadata),
	}

	resp, err := p.client.StartFirstLastFrameToVideo(ctx, payload)
	if err != nil {
		return nil, err
	}

	return buildQwenVideoResponse(p.name, req, resp), nil
}

func (p *qwenProvider) videoEdit(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for video_edit")
	}

	model := resolveQwenVideoModel(p, req)
	model = qwenprovider.ResolveHappyHorseModelID(model, "video_edit")
	if !strings.Contains(strings.ToLower(model), "video-edit") {
		return nil, fmt.Errorf("video_edit expects model %s", qwenprovider.ModelHappyHorseVideoEdit)
	}

	refs := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
	if len(refs) == 0 {
		return nil, fmt.Errorf("video_edit requires at least one URL: the source video as the first reference")
	}
	var media []qwenprovider.HappyHorseVideoEditMedia
	media = append(media, qwenprovider.HappyHorseVideoEditMedia{Type: "video", URL: refs[0]})
	for _, u := range refs[1:] {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if len(media) >= 6 {
			break
		}
		media = append(media, qwenprovider.HappyHorseVideoEditMedia{Type: "reference_image", URL: u})
	}

	payload := &qwenprovider.HappyHorseVideoEditPayload{
		Model: model,
		Input: qwenprovider.HappyHorseVideoEditInput{
			Prompt: prompt,
			Media:  media,
		},
		Parameters: &qwenprovider.HappyHorseVideoEditParams{
			Resolution: qwenprovider.NormalizeHappyHorseResolution(strings.TrimSpace(req.Resolution)),
			Watermark:  req.Watermark,
		},
	}
	if req.Seed != 0 {
		s := int(req.Seed)
		payload.Parameters.Seed = &s
	}

	resp, err := p.client.StartHappyHorseVideoSynthesis(ctx, payload)
	if err != nil {
		return nil, err
	}

	return buildQwenVideoSynthesisStartResponse(p.name, req, resp), nil
}

func (p *qwenProvider) textToImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for qwen text to image")
	}

	input := &qwenprovider.TextToImageInput{
		Prompt:         prompt,
		NegativePrompt: strings.TrimSpace(req.NegativePrompt),
	}

	params := &qwenprovider.TextToImageParameters{
		Size: strings.TrimSpace(req.Size),
		N:    req.OutputCount,
	}
	if req.Watermark != nil {
		params.Watermark = req.Watermark
	}
	if req.Seed != 0 {
		seed := req.Seed
		params.Seed = &seed
	}
	if value, ok := boolFromOptions(req.Options, "prompt_extend"); ok {
		params.PromptExtend = &value
	}

	payload := &qwenprovider.TextToImageRequest{
		Model:      strings.TrimSpace(req.Model),
		Input:      input,
		Parameters: params,
	}

	resp, err := p.client.StartTextToImage(ctx, payload)
	if err != nil {
		return nil, err
	}

	return buildQwenImageResponse(p.name, req, resp), nil
}

func (p *qwenProvider) imageToImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for qwen image to image")
	}

	references := collectImages(req.ReferenceImageURL, req.ReferenceImages, 2)
	if len(references) == 0 {
		return nil, fmt.Errorf("at least one reference image is required for qwen image to image")
	}

	input := &qwenprovider.ImageToImageInput{
		Prompt:         prompt,
		Images:         references,
		NegativePrompt: strings.TrimSpace(req.NegativePrompt),
	}

	params := &qwenprovider.ImageToImageParameters{
		N: req.OutputCount,
	}
	if req.Watermark != nil {
		params.Watermark = req.Watermark
	}
	if req.Seed != 0 {
		seed := req.Seed
		params.Seed = &seed
	}

	payload := &qwenprovider.ImageToImageRequest{
		Model:      strings.TrimSpace(req.Model),
		Input:      input,
		Parameters: params,
	}

	resp, err := p.client.StartImageToImage(ctx, payload)
	if err != nil {
		return nil, err
	}

	return buildQwenImageResponse(p.name, req, resp), nil
}

func resolveQwenVideoModel(p *qwenProvider, req *GenerateRequest) string {
	if req != nil {
		if m := strings.TrimSpace(req.Model); m != "" {
			return m
		}
	}
	if p != nil && p.config != nil {
		if m := strings.TrimSpace(p.config.VideoModel); m != "" {
			return m
		}
		return strings.TrimSpace(p.config.Model)
	}
	return ""
}

func buildQwenVideoSynthesisStartResponse(provider string, req *GenerateRequest, resp *qwenprovider.VideoSynthesisTaskResponse) *GenerateResponse {
	now := time.Now()
	out := &GenerateResponse{
		Provider:    provider,
		Operation:   req.Operation,
		MediaType:   MediaTypeVideo,
		Status:      string(StatusPending),
		Metadata:    mergeMaps(req.Metadata),
		Raw:         map[string]interface{}{"synthesis_response": resp},
		StartedAt:   now,
		CompletedAt: now,
		Message:     string(StatusPending),
	}
	if resp == nil {
		return out
	}
	if resp.RequestID != "" {
		if out.Metadata == nil {
			out.Metadata = make(map[string]interface{})
		}
		out.Metadata["request_id"] = resp.RequestID
	}
	if resp.Code != "" {
		out.ErrorCode = resp.Code
		out.Error = resp.Message
		out.Status = string(StatusFailed)
		out.Message = resp.Message
	}
	if resp.Output != nil {
		out.TaskID = strings.TrimSpace(resp.Output.TaskID)
		st := NormalizeStatus(resp.Output.TaskStatus)
		out.Status = st
		out.Message = strings.TrimSpace(resp.Output.TaskStatus)
		if resp.Output.Code != "" {
			out.ErrorCode = resp.Output.Code
			out.Error = resp.Output.Message
			out.Status = string(StatusFailed)
		}
	}
	return out
}

func buildQwenVideoPollFromSynthesis(provider string, resp *qwenprovider.VideoSynthesisTaskResponse) *GenerateResponse {
	result := &GenerateResponse{
		Provider:  provider,
		MediaType: MediaTypeVideo,
		Metadata:  make(map[string]interface{}),
		Raw: map[string]interface{}{
			"synthesis_task": resp,
		},
	}
	if resp == nil {
		return result
	}
	if resp.RequestID != "" {
		result.Metadata["request_id"] = resp.RequestID
	}
	if resp.Code != "" {
		result.Error = resp.Message
		result.ErrorCode = resp.Code
		result.Status = string(StatusFailed)
	}
	if resp.Output != nil {
		result.TaskID = strings.TrimSpace(resp.Output.TaskID)
		result.Status = NormalizeStatus(resp.Output.TaskStatus)
		result.Message = strings.TrimSpace(resp.Output.TaskStatus)
		result.VideoURL = strings.TrimSpace(resp.Output.VideoURL)
		if resp.Output.Code != "" {
			result.Error = resp.Output.Message
			result.ErrorCode = resp.Output.Code
			result.Status = string(StatusFailed)
		}
	}
	if resp.Usage != nil {
		u := &Usage{VideoCount: resp.Usage.VideoCount}
		if resp.Usage.Duration > 0 {
			u.DurationSeconds = resp.Usage.Duration
		}
		result.Usage = u
		if result.VideoURL != "" && result.Status == string(StatusCompleted) {
			result.Usage.VideoCount = 1
		}
	}
	return result
}

func buildQwenVideoResponse(provider string, req *GenerateRequest, resp *qwenprovider.StartResponse) *GenerateResponse {
	now := time.Now()
	metadata := mergeMaps(req.Metadata)
	if resp.Usage != nil {
		metadata["usage"] = resp.Usage
	}

	return &GenerateResponse{
		Provider:    provider,
		Operation:   req.Operation,
		MediaType:   MediaTypeVideo,
		TaskID:      strings.TrimSpace(resp.TaskID),
		Status:      NormalizeStatus(resp.Status),
		Metadata:    metadata,
		Raw:         map[string]interface{}{"start_response": resp},
		StartedAt:   now,
		CompletedAt: now,
	}
}

func buildQwenImageResponse(provider string, req *GenerateRequest, resp *qwenprovider.TextToImageTaskResponse) *GenerateResponse {
	now := time.Now()
	result := &GenerateResponse{
		Provider:    provider,
		Operation:   req.Operation,
		MediaType:   MediaTypeImage,
		Metadata:    make(map[string]interface{}),
		Raw:         map[string]interface{}{"image_response": resp},
		StartedAt:   now,
		CompletedAt: now,
	}

	if resp != nil {
		if resp.RequestID != "" {
			result.Metadata["request_id"] = resp.RequestID
		}

		if resp.Code != "" {
			result.ErrorCode = resp.Code
			result.Error = resp.Message
		}

		if resp.Output != nil {
			result.TaskID = strings.TrimSpace(resp.Output.TaskID)
			result.Status = NormalizeStatus(resp.Output.TaskStatus)
			result.Message = strings.TrimSpace(resp.Output.Message)

			if resp.Output.Code != "" {
				result.ErrorCode = resp.Output.Code
				result.Error = resp.Output.Message
			}

			for _, r := range resp.Output.Results {
				if trimmed := strings.TrimSpace(r.URL); trimmed != "" {
					result.ImageURLs = append(result.ImageURLs, trimmed)
				}
			}

			if resp.Output.TaskMetrics != nil {
				result.Metadata["task_metrics"] = resp.Output.TaskMetrics
			}
		}

		if resp.Usage != nil {
			result.Usage = &Usage{
				ImageCount: resp.Usage.ImageCount,
			}
		}
	}

	return result
}
