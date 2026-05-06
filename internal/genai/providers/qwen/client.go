package qwen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	textToVideoPath      = "/api/v1/services/aigc/video-generation/generation"
	imageToVideoPath     = "/api/v1/services/aigc/video-generation/image-to-video"
	firstLastFramePath   = "/api/v1/services/aigc/video-generation/first-last-frame"
	textToImagePath      = "/api/v1/services/aigc/text2image/image-synthesis"
	imageToImagePath     = "/api/v1/services/aigc/image2image/image-synthesis"
	videoSynthesisPath   = "/api/v1/services/aigc/video-generation/video-synthesis"
	taskStatusPath       = "/api/v1/tasks/%s"
	renderDownloadPath   = "/api/v1/videos/%s"
	defaultDurationValue = 8
)

// Config captures the minimal configuration necessary to call Qwen video APIs.
type Config struct {
	APIKey       string
	BaseURL      string
	Timeout      time.Duration
	DefaultModel string
}

// Client provides atomic Qwen API operations.
type Client struct {
	config     Config
	httpClient *http.Client
}

// New instantiates a Qwen client with sensible defaults.
func New(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: config.Timeout},
	}
}

// TextToVideoRequest matches the text-to-video API payload.
type TextToVideoRequest struct {
	Prompt      string                 `json:"prompt"`
	Model       string                 `json:"model"`
	AspectRatio string                 `json:"aspect_ratio,omitempty"`
	Duration    int                    `json:"duration,omitempty"`
	Quality     string                 `json:"quality,omitempty"`
	Style       string                 `json:"style,omitempty"`
	CallbackURL string                 `json:"callback_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ImageToVideoRequest describes the image-to-video payload.
type ImageToVideoRequest struct {
	ImageURL    string                 `json:"image_url"`
	Prompt      string                 `json:"prompt,omitempty"`
	Model       string                 `json:"model"`
	AspectRatio string                 `json:"aspect_ratio,omitempty"`
	Duration    int                    `json:"duration,omitempty"`
	CallbackURL string                 `json:"callback_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FirstLastFrameRequest configures keyframe conditioned videos.
type FirstLastFrameRequest struct {
	FirstFrameURL string                 `json:"first_frame_url"`
	LastFrameURL  string                 `json:"last_frame_url"`
	Prompt        string                 `json:"prompt,omitempty"`
	Model         string                 `json:"model"`
	Duration      int                    `json:"duration,omitempty"`
	CallbackURL   string                 `json:"callback_url,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// TextToImageRequest configures text-to-image generation tasks.
type TextToImageRequest struct {
	Model      string                 `json:"model"`
	Input      *TextToImageInput      `json:"input"`
	Parameters *TextToImageParameters `json:"parameters,omitempty"`
}

// TextToImageInput captures textual guidance for image synthesis.
type TextToImageInput struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
}

// TextToImageParameters holds optional generation controls.
type TextToImageParameters struct {
	Size         string `json:"size,omitempty"`
	N            int    `json:"n,omitempty"`
	PromptExtend *bool  `json:"prompt_extend,omitempty"`
	Watermark    *bool  `json:"watermark,omitempty"`
	Seed         *int   `json:"seed,omitempty"`
}

// ImageToImageRequest configures multi-image guided generation tasks.
type ImageToImageRequest struct {
	Model      string                  `json:"model"`
	Input      *ImageToImageInput      `json:"input"`
	Parameters *ImageToImageParameters `json:"parameters,omitempty"`
}

// ImageToImageInput enumerates guidance for image-to-image synthesis.
type ImageToImageInput struct {
	Prompt         string   `json:"prompt"`
	Images         []string `json:"images"`
	NegativePrompt string   `json:"negative_prompt,omitempty"`
}

// ImageToImageParameters holds optional generation controls.
type ImageToImageParameters struct {
	N         int   `json:"n,omitempty"`
	Watermark *bool `json:"watermark,omitempty"`
	Seed      *int  `json:"seed,omitempty"`
}

// ImageToVideoSynthesisRequest configures first-frame guided video generation.
type ImageToVideoSynthesisRequest struct {
	Model      string                       `json:"model"`
	Input      *ImageToVideoSynthesisInput  `json:"input"`
	Parameters *ImageToVideoSynthesisParams `json:"parameters,omitempty"`
}

// ImageToVideoSynthesisInput captures video conditioning inputs.
type ImageToVideoSynthesisInput struct {
	Prompt         string `json:"prompt,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	ImageURL       string `json:"img_url"`
	AudioURL       string `json:"audio_url,omitempty"`
	Template       string `json:"template,omitempty"`
}

// ImageToVideoSynthesisParams controls optional video attributes.
type ImageToVideoSynthesisParams struct {
	Resolution   string `json:"resolution,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	PromptExtend *bool  `json:"prompt_extend,omitempty"`
	Watermark    *bool  `json:"watermark,omitempty"`
	Audio        *bool  `json:"audio,omitempty"`
	Seed         *int   `json:"seed,omitempty"`
}

// StartResponse represents the task creation response body.
type StartResponse struct {
	TaskID string                 `json:"task_id"`
	Status string                 `json:"status"`
	Usage  map[string]interface{} `json:"usage,omitempty"`
}

// TextToImageTaskResponse represents asynchronous task metadata.
type TextToImageTaskResponse struct {
	Output    *TextToImageTaskOutput `json:"output,omitempty"`
	Usage     *TextToImageUsage      `json:"usage,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Code      string                 `json:"code,omitempty"`
	Message   string                 `json:"message,omitempty"`
}

// TextToImageTaskOutput captures task progress and results.
type TextToImageTaskOutput struct {
	TaskID        string                  `json:"task_id"`
	TaskStatus    string                  `json:"task_status"`
	SubmitTime    string                  `json:"submit_time,omitempty"`
	ScheduledTime string                  `json:"scheduled_time,omitempty"`
	EndTime       string                  `json:"end_time,omitempty"`
	Results       []TextToImageResult     `json:"results,omitempty"`
	TaskMetrics   *TextToImageTaskMetrics `json:"task_metrics,omitempty"`
	Code          string                  `json:"code,omitempty"`
	Message       string                  `json:"message,omitempty"`
}

// TextToImageResult describes a single generated asset.
type TextToImageResult struct {
	OrigPrompt   string `json:"orig_prompt,omitempty"`
	ActualPrompt string `json:"actual_prompt,omitempty"`
	URL          string `json:"url,omitempty"`
	Code         string `json:"code,omitempty"`
	Message      string `json:"message,omitempty"`
}

// TextToImageTaskMetrics aggregates task execution stats.
type TextToImageTaskMetrics struct {
	Total     int `json:"TOTAL"`
	Succeeded int `json:"SUCCEEDED"`
	Failed    int `json:"FAILED"`
}

// TextToImageUsage reports billing-related counters.
type TextToImageUsage struct {
	ImageCount int `json:"image_count"`
}

// VideoSynthesisTaskResponse represents asynchronous video generation metadata.
type VideoSynthesisTaskResponse struct {
	Output    *VideoSynthesisTaskOutput `json:"output,omitempty"`
	Usage     *VideoSynthesisUsage      `json:"usage,omitempty"`
	RequestID string                    `json:"request_id,omitempty"`
	Code      string                    `json:"code,omitempty"`
	Message   string                    `json:"message,omitempty"`
}

// VideoSynthesisTaskOutput captures generated video information.
type VideoSynthesisTaskOutput struct {
	TaskID        string `json:"task_id"`
	TaskStatus    string `json:"task_status"`
	SubmitTime    string `json:"submit_time,omitempty"`
	ScheduledTime string `json:"scheduled_time,omitempty"`
	EndTime       string `json:"end_time,omitempty"`
	VideoURL      string `json:"video_url,omitempty"`
	OrigPrompt    string `json:"orig_prompt,omitempty"`
	ActualPrompt  string `json:"actual_prompt,omitempty"`
	Code          string `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
}

// VideoSynthesisUsage reports usage statistics for video tasks.
type VideoSynthesisUsage struct {
	Duration   int `json:"duration"`
	VideoCount int `json:"video_count"`
	SR         int `json:"SR"`
}

// StatusResponse describes task progress information.
type StatusResponse struct {
	TaskID    string                 `json:"task_id"`
	Status    string                 `json:"status"`
	VideoURL  string                 `json:"video_url,omitempty"`
	Thumbnail string                 `json:"thumbnail,omitempty"`
	Progress  int                    `json:"progress,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Usage     map[string]interface{} `json:"usage,omitempty"`
}

// StartTextToVideo calls the Qwen text-to-video API.
func (c *Client) StartTextToVideo(ctx context.Context, payload *TextToVideoRequest) (*StartResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Model == "" {
		payload.Model = choose(c.config.DefaultModel, "qwen-video-1")
	}
	if payload.Duration <= 0 {
		payload.Duration = defaultDurationValue
	}
	resp := &StartResponse{}
	if err := c.doRequest(ctx, http.MethodPost, textToVideoPath, payload, resp, nil); err != nil {
		return nil, err
	}
	return resp, nil
}

// StartImageToVideo triggers the image-to-video workflow.
func (c *Client) StartImageToVideo(ctx context.Context, payload *ImageToVideoRequest) (*StartResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if strings.TrimSpace(payload.ImageURL) == "" {
		return nil, fmt.Errorf("image_url is required")
	}
	if payload.Model == "" {
		payload.Model = choose(c.config.DefaultModel, "qwen-video-1")
	}
	if payload.Duration <= 0 {
		payload.Duration = defaultDurationValue
	}
	resp := &StartResponse{}
	if err := c.doRequest(ctx, http.MethodPost, imageToVideoPath, payload, resp, nil); err != nil {
		return nil, err
	}
	return resp, nil
}

// StartFirstLastFrameToVideo starts the keyframe guided generation.
func (c *Client) StartFirstLastFrameToVideo(ctx context.Context, payload *FirstLastFrameRequest) (*StartResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if strings.TrimSpace(payload.FirstFrameURL) == "" || strings.TrimSpace(payload.LastFrameURL) == "" {
		return nil, fmt.Errorf("first_frame_url and last_frame_url are required")
	}
	if payload.Model == "" {
		payload.Model = choose(c.config.DefaultModel, "qwen-video-1")
	}
	if payload.Duration <= 0 {
		payload.Duration = defaultDurationValue
	}
	resp := &StartResponse{}
	if err := c.doRequest(ctx, http.MethodPost, firstLastFramePath, payload, resp, nil); err != nil {
		return nil, err
	}
	return resp, nil
}

// StartTextToImage creates an asynchronous text-to-image generation task.
func (c *Client) StartTextToImage(ctx context.Context, payload *TextToImageRequest) (*TextToImageTaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Input == nil || strings.TrimSpace(payload.Input.Prompt) == "" {
		return nil, fmt.Errorf("input.prompt is required")
	}
	if strings.TrimSpace(payload.Model) == "" {
		payload.Model = choose(c.config.DefaultModel, "wan2.5-t2i-preview")
	}
	resp := &TextToImageTaskResponse{}
	headers := map[string]string{"X-DashScope-Async": "enable"}
	if err := c.doRequest(ctx, http.MethodPost, textToImagePath, payload, resp, headers); err != nil {
		return nil, err
	}
	return resp, nil
}

// StartImageToImage creates a multi-image reference async generation task.
func (c *Client) StartImageToImage(ctx context.Context, payload *ImageToImageRequest) (*TextToImageTaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Input == nil {
		return nil, fmt.Errorf("input is required")
	}
	if strings.TrimSpace(payload.Input.Prompt) == "" {
		return nil, fmt.Errorf("input.prompt is required")
	}
	if len(payload.Input.Images) == 0 {
		return nil, fmt.Errorf("input.images requires at least one image")
	}
	if len(payload.Input.Images) > 2 {
		return nil, fmt.Errorf("input.images supports at most two images")
	}
	for i, img := range payload.Input.Images {
		if strings.TrimSpace(img) == "" {
			return nil, fmt.Errorf("input.images[%d] cannot be empty", i)
		}
	}
	if strings.TrimSpace(payload.Model) == "" {
		payload.Model = choose(c.config.DefaultModel, "wan2.5-i2i-preview")
	}
	resp := &TextToImageTaskResponse{}
	headers := map[string]string{"X-DashScope-Async": "enable"}
	if err := c.doRequest(ctx, http.MethodPost, imageToImagePath, payload, resp, headers); err != nil {
		return nil, err
	}
	return resp, nil
}

// StartImageToVideoAutoAudio generates video with automatic narration (audio enabled).
func (c *Client) StartImageToVideoAutoAudio(ctx context.Context, payload *ImageToVideoSynthesisRequest) (*VideoSynthesisTaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Input == nil {
		return nil, fmt.Errorf("input is required")
	}
	if strings.TrimSpace(payload.Input.AudioURL) != "" {
		return nil, fmt.Errorf("input.audio_url must be empty when using automatic audio")
	}
	if payload.Parameters == nil {
		payload.Parameters = &ImageToVideoSynthesisParams{}
	}
	payload.Parameters.Audio = boolPtr(true)
	return c.startImageToVideo(ctx, payload)
}

// StartImageToVideoWithAudioURL generates video using a caller-provided audio track.
func (c *Client) StartImageToVideoWithAudioURL(ctx context.Context, payload *ImageToVideoSynthesisRequest) (*VideoSynthesisTaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Input == nil {
		return nil, fmt.Errorf("input is required")
	}
	if strings.TrimSpace(payload.Input.AudioURL) == "" {
		return nil, fmt.Errorf("input.audio_url is required")
	}
	if payload.Parameters != nil && payload.Parameters.Audio != nil && !*payload.Parameters.Audio {
		return nil, fmt.Errorf("parameters.audio cannot be false when audio_url is provided")
	}
	return c.startImageToVideo(ctx, payload)
}

// StartImageToVideoSilent generates a mute video without audio.
func (c *Client) StartImageToVideoSilent(ctx context.Context, payload *ImageToVideoSynthesisRequest) (*VideoSynthesisTaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Input == nil {
		return nil, fmt.Errorf("input is required")
	}
	if strings.TrimSpace(payload.Input.AudioURL) != "" {
		return nil, fmt.Errorf("input.audio_url must be empty for silent videos")
	}
	if payload.Parameters == nil {
		payload.Parameters = &ImageToVideoSynthesisParams{}
	}
	payload.Parameters.Audio = boolPtr(false)
	return c.startImageToVideo(ctx, payload)
}

// StartImageToVideoWithTemplate applies a video template effect during generation.
func (c *Client) StartImageToVideoWithTemplate(ctx context.Context, payload *ImageToVideoSynthesisRequest) (*VideoSynthesisTaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Input == nil {
		return nil, fmt.Errorf("input is required")
	}
	if strings.TrimSpace(payload.Input.Template) == "" {
		return nil, fmt.Errorf("input.template is required for template-based synthesis")
	}
	return c.startImageToVideo(ctx, payload)
}

// GetTaskStatus fetches job status information.
func (c *Client) GetTaskStatus(ctx context.Context, taskID string) (*StatusResponse, error) {
	resp := &StatusResponse{}
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf(taskStatusPath, taskID), nil, resp, nil); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetTextToImageTask retrieves task progress and results for text-to-image jobs.
func (c *Client) GetTextToImageTask(ctx context.Context, taskID string) (*TextToImageTaskResponse, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("taskID is required")
	}
	resp := &TextToImageTaskResponse{}
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf(taskStatusPath, taskID), nil, resp, nil); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetImageToVideoTask retrieves task progress for image-to-video synthesis.
func (c *Client) GetImageToVideoTask(ctx context.Context, taskID string) (*VideoSynthesisTaskResponse, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("taskID is required")
	}
	resp := &VideoSynthesisTaskResponse{}
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf(taskStatusPath, taskID), nil, resp, nil); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) startImageToVideo(ctx context.Context, payload *ImageToVideoSynthesisRequest) (*VideoSynthesisTaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Input == nil {
		return nil, fmt.Errorf("input is required")
	}
	payload.Input.ImageURL = strings.TrimSpace(payload.Input.ImageURL)
	if payload.Input.ImageURL == "" {
		return nil, fmt.Errorf("input.img_url is required")
	}
	payload.Input.AudioURL = strings.TrimSpace(payload.Input.AudioURL)
	if payload.Parameters != nil && payload.Parameters.Audio != nil && payload.Input.AudioURL != "" && !*payload.Parameters.Audio {
		return nil, fmt.Errorf("parameters.audio cannot be false when audio_url is provided")
	}
	if strings.TrimSpace(payload.Model) == "" {
		payload.Model = choose(c.config.DefaultModel, "wan2.5-i2v-preview")
	}
	resp := &VideoSynthesisTaskResponse{}
	headers := map[string]string{"X-DashScope-Async": "enable"}
	if err := c.doRequest(ctx, http.MethodPost, videoSynthesisPath, payload, resp, headers); err != nil {
		return nil, err
	}
	return resp, nil
}

// StartHappyHorseVideoSynthesis creates an async HappyHorse job (T2V / I2V / video-edit) on the DashScope video-synthesis endpoint.
func (c *Client) StartHappyHorseVideoSynthesis(ctx context.Context, payload interface{}) (*VideoSynthesisTaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	resp := &VideoSynthesisTaskResponse{}
	headers := map[string]string{"X-DashScope-Async": "enable"}
	if err := c.doRequest(ctx, http.MethodPost, videoSynthesisPath, payload, resp, headers); err != nil {
		return nil, err
	}
	return resp, nil
}

// DownloadRendered retrieves rendered media for the specified task.
func (c *Client) DownloadRendered(ctx context.Context, taskID string) ([]byte, error) {
	endpoint := c.endpoint(fmt.Sprintf(renderDownloadPath, url.PathEscape(taskID)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) doRequest(ctx context.Context, method, apiPath string, payload interface{}, out interface{}, headers map[string]string) error {
	endpoint := c.endpoint(apiPath)
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qwen api %s failed with status %d: %s", apiPath, resp.StatusCode, string(bodyBytes))
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) endpoint(apiPath string) string {
	base := strings.TrimSuffix(c.config.BaseURL, "/")
	if base == "" {
		base = "https://dashscope.aliyuncs.com"
	}
	if strings.HasPrefix(apiPath, "http") {
		return apiPath
	}
	return base + path.Clean("/"+apiPath)
}

func boolPtr(value bool) *bool {
	return &value
}

func choose(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
