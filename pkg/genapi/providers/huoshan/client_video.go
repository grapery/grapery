package huoshan

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// TextToVideoRequest is the payload for text-driven video workflows.
type TextToVideoRequest struct {
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	Workflow    string                 `json:"workflow"`
	Duration    int                    `json:"duration,omitempty"`
	Resolution  string                 `json:"resolution,omitempty"`
	CallbackURL string                 `json:"callback_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ImageToVideoRequest handles image-conditioned workflows (including multi-reference and lite mode).
type ImageToVideoRequest struct {
	Model           string                 `json:"model"`
	Prompt          string                 `json:"prompt,omitempty"`
	Workflow        string                 `json:"workflow"`
	Duration        int                    `json:"duration,omitempty"`
	Resolution      string                 `json:"resolution,omitempty"`
	CallbackURL     string                 `json:"callback_url,omitempty"`
	ImageURL        string                 `json:"image_url"`
	ReferenceImages []string               `json:"reference_images,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// KeyframeRequest drives first/last frame workflows.
type KeyframeRequest struct {
	Model       string                 `json:"model"`
	FirstFrame  string                 `json:"first_frame"`
	LastFrame   string                 `json:"last_frame"`
	Prompt      string                 `json:"prompt,omitempty"`
	Workflow    string                 `json:"workflow"`
	Duration    int                    `json:"duration,omitempty"`
	CallbackURL string                 `json:"callback_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TaskResponse represents task creation output.
type TaskResponse struct {
	TaskID   string                 `json:"task_id"`
	Status   string                 `json:"status"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VideoUsage reports consumption for video generation tasks.
type VideoUsage struct {
	CompletionTokens int `json:"completion_tokens"`
}

// StatusResponse exposes progress info.
type StatusResponse struct {
	TaskID    string                 `json:"task_id"`
	Status    string                 `json:"status"`
	VideoURL  string                 `json:"video_url"`
	Thumbnail string                 `json:"thumbnail"`
	Progress  int                    `json:"progress"`
	Error     string                 `json:"error_message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Usage     *VideoUsage            `json:"usage,omitempty"`
}

// ListVideoTasksRequest describes filters for listing content generation tasks.
type ListVideoTasksRequest struct {
	PageNum  int
	PageSize int
	Status   string
	Model    string
	TaskIDs  []string
}

func (c *Client) CreateTextVideo(ctx context.Context, payload *TextToVideoRequest) (*TaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	modelName := choose(payload.Model, payload.Workflow, c.config.Workflow, defaultVideoModel)
	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("model is required")
	}

	prompt := buildVideoPrompt(payload.Prompt, payload.Duration, payload.Resolution)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	items := []*arkmodel.CreateContentGenerationContentItem{
		{
			Type: arkmodel.ContentGenerationContentItemTypeText,
			Text: ptr(prompt),
		},
	}

	req := arkmodel.CreateContentGenerationTaskRequest{
		Model:           strings.TrimSpace(modelName),
		Content:         items,
		ReturnLastFrame: ptr(true),
	}
	if cb := strings.TrimSpace(payload.CallbackURL); cb != "" {
		req.CallbackUrl = ptr(cb)
	}

	resp, err := c.arkClient.CreateContentGenerationTask(ctx, req)
	if err != nil {
		return nil, err
	}

	metadata := mergeMetadata(payload.Metadata, map[string]interface{}{
		"model": req.Model,
		"mode":  "text_to_video",
	})

	return &TaskResponse{
		TaskID:   resp.ID,
		Status:   "processing",
		Metadata: metadata,
	}, nil
}

/*
参考图生视频功能的文本提示词，可以用自然语言指定多张图片的组合。但若想有更好的指令遵循效果，推荐使用“[图1]xxx，[图2]xxx”的方式来指定图片。
示例1：戴着眼镜穿着蓝色T恤的男生和柯基小狗，坐在草坪上，3D卡通风格
示例2：[图1]戴着眼镜穿着蓝色T恤的男生和[图2]的柯基小狗，坐在[图3]的草坪上，3D卡通风格
*/

func (c *Client) CreateImageVideo(ctx context.Context, payload *ImageToVideoRequest) (*TaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	modelName := choose(payload.Model, payload.Workflow, c.config.Workflow, defaultVideoModel)
	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("model is required")
	}

	referenceImages := collectReferenceImages(payload)
	if len(referenceImages) == 0 {
		return nil, fmt.Errorf("at least one reference image is required")
	}

	prompt := buildVideoPrompt(payload.Prompt, payload.Duration, payload.Resolution)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	items := []*arkmodel.CreateContentGenerationContentItem{
		{
			Type: arkmodel.ContentGenerationContentItemTypeText,
			Text: ptr(prompt),
		},
	}

	role := ptr("reference_image")
	for _, imageURL := range referenceImages {
		items = append(items, &arkmodel.CreateContentGenerationContentItem{
			Type: arkmodel.ContentGenerationContentItemTypeImage,
			ImageURL: &arkmodel.ImageURL{
				URL: imageURL,
			},
			Role: role,
		})
	}

	req := arkmodel.CreateContentGenerationTaskRequest{
		Model:           strings.TrimSpace(modelName),
		Content:         items,
		ReturnLastFrame: ptr(true),
	}

	if cb := strings.TrimSpace(payload.CallbackURL); cb != "" {
		req.CallbackUrl = ptr(cb)
	}

	resp, err := c.arkClient.CreateContentGenerationTask(ctx, req)
	if err != nil {
		return nil, err
	}

	metadata := mergeMetadata(payload.Metadata, map[string]interface{}{
		"model":  req.Model,
		"mode":   "image_to_video",
		"images": referenceImages,
		"prompt": prompt,
	})

	return &TaskResponse{
		TaskID:   resp.ID,
		Status:   "processing",
		Metadata: metadata,
	}, nil
}

func (c *Client) CreateKeyframeVideo(ctx context.Context, payload *KeyframeRequest) (*TaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	first := strings.TrimSpace(payload.FirstFrame)
	last := strings.TrimSpace(payload.LastFrame)
	if first == "" || last == "" {
		return nil, fmt.Errorf("first_frame and last_frame are required")
	}

	modelName := choose(payload.Model, payload.Workflow, c.config.Workflow, defaultVideoModel)
	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("model is required")
	}

	prompt := buildVideoPrompt(payload.Prompt, payload.Duration, "")
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	items := []*arkmodel.CreateContentGenerationContentItem{
		{
			Type: arkmodel.ContentGenerationContentItemTypeText,
			Text: ptr(prompt),
		},
		{
			Type:     arkmodel.ContentGenerationContentItemTypeImage,
			ImageURL: &arkmodel.ImageURL{URL: first},
			Role:     ptr("first_frame"),
		},
		{
			Type:     arkmodel.ContentGenerationContentItemTypeImage,
			ImageURL: &arkmodel.ImageURL{URL: last},
			Role:     ptr("last_frame"),
		},
	}

	req := arkmodel.CreateContentGenerationTaskRequest{
		Model:           strings.TrimSpace(modelName),
		Content:         items,
		ReturnLastFrame: ptr(true),
	}
	if cb := strings.TrimSpace(payload.CallbackURL); cb != "" {
		req.CallbackUrl = ptr(cb)
	}

	resp, err := c.arkClient.CreateContentGenerationTask(ctx, req)
	if err != nil {
		return nil, err
	}

	metadata := mergeMetadata(payload.Metadata, map[string]interface{}{
		"model":       req.Model,
		"mode":        "keyframe_to_video",
		"first_frame": first,
		"last_frame":  last,
	})

	return &TaskResponse{
		TaskID:   resp.ID,
		Status:   "processing",
		Metadata: metadata,
	}, nil
}

func (c *Client) GetTaskStatus(ctx context.Context, taskID string) (*StatusResponse, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	resp, err := c.arkClient.GetContentGenerationTask(ctx, arkmodel.GetContentGenerationTaskRequest{ID: taskID})
	if err != nil {
		return nil, err
	}

	status := strings.TrimSpace(resp.Status)
	if status == "" {
		status = "processing"
	}

	metadata := map[string]interface{}{
		"model": resp.Model,
		"seed":  resp.Seed,
	}
	if resp.RevisedPrompt != nil {
		metadata["revised_prompt"] = strings.TrimSpace(*resp.RevisedPrompt)
	}

	if resp.Error != nil {
		metadata["error"] = resp.Error
	}
	metadata["usage"] = resp.Usage

	var usage *VideoUsage
	if resp.Usage != (arkmodel.Usage{}) {
		usage = &VideoUsage{CompletionTokens: resp.Usage.CompletionTokens}
	}

	return &StatusResponse{
		TaskID:    resp.ID,
		Status:    status,
		VideoURL:  resp.Content.VideoURL,
		Thumbnail: resp.Content.LastFrameURL,
		Progress:  0,
		Error: func() string {
			if resp.Error == nil {
				return ""
			}
			return strings.TrimSpace(resp.Error.Message)
		}(),
		Metadata: metadata,
		Usage:    usage,
	}, nil
}

func (c *Client) GetTaskAssets(ctx context.Context, taskID string) ([]byte, error) {
	status, err := c.GetTaskStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}
	videoURL := strings.TrimSpace(status.VideoURL)
	if videoURL == "" {
		return nil, fmt.Errorf("video_url is empty for task %s", taskID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download video asset failed with status %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) GetVideoTask(ctx context.Context, taskID string) (*arkmodel.GetContentGenerationTaskResponse, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	resp, err := c.arkClient.GetContentGenerationTask(ctx, arkmodel.GetContentGenerationTaskRequest{ID: taskID})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListVideoTasks(ctx context.Context, req *ListVideoTasksRequest) (*arkmodel.ListContentGenerationTasksResponse, error) {
	var request arkmodel.ListContentGenerationTasksRequest
	if req != nil {
		if req.PageNum > 0 {
			request.PageNum = ptr(req.PageNum)
		}
		if req.PageSize > 0 {
			request.PageSize = ptr(req.PageSize)
		}
		if req.Status != "" || req.Model != "" || len(req.TaskIDs) > 0 {
			filter := &arkmodel.ListContentGenerationTasksFilter{}
			if strings.TrimSpace(req.Status) != "" {
				filter.Status = ptr(strings.TrimSpace(req.Status))
			}
			if strings.TrimSpace(req.Model) != "" {
				filter.Model = ptr(strings.TrimSpace(req.Model))
			}
			for _, id := range req.TaskIDs {
				trimmed := strings.TrimSpace(id)
				if trimmed == "" {
					continue
				}
				filter.TaskIDs = append(filter.TaskIDs, ptr(trimmed))
			}
			request.Filter = filter
		}
	}

	resp, err := c.arkClient.ListContentGenerationTasks(ctx, request)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) DeleteVideoTask(ctx context.Context, taskID string) error {
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	return c.arkClient.DeleteContentGenerationTask(ctx, arkmodel.DeleteContentGenerationTaskRequest{ID: taskID})
}

func buildVideoPrompt(basePrompt string, duration int, resolution string) string {
	var segments []string
	if prompt := strings.TrimSpace(basePrompt); prompt != "" {
		segments = append(segments, prompt)
	}
	if duration > 0 {
		segments = append(segments, fmt.Sprintf("--dur %d", duration))
	}
	if res := strings.TrimSpace(resolution); res != "" {
		if strings.HasPrefix(res, "--") {
			segments = append(segments, res)
		} else if strings.Contains(res, ":") || strings.Contains(strings.ToLower(res), "x") {
			segments = append(segments, fmt.Sprintf("--ratio %s", res))
		} else {
			segments = append(segments, res)
		}
	}
	return strings.Join(segments, " ")
}

func collectReferenceImages(payload *ImageToVideoRequest) []string {
	var images []string
	if trimmed := strings.TrimSpace(payload.ImageURL); trimmed != "" {
		images = append(images, trimmed)
	}
	for _, img := range payload.ReferenceImages {
		if trimmed := strings.TrimSpace(img); trimmed != "" {
			images = append(images, trimmed)
		}
	}
	return images
}

func mergeMetadata(values ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, value := range values {
		for k, v := range value {
			result[k] = v
		}
	}
	return result
}
