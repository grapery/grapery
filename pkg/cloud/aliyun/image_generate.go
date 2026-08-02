package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WanxiangClient represents a client for the Wanxiang I2V service
type WanxiangClient struct {
	APIKey   string
	Endpoint string
}

// CreateTaskRequest represents the request parameters for creating an I2V task
type CreateTaskRequest struct {
	Model      string `json:"model"`
	Input      Input  `json:"input"`
	Parameters Params `json:"parameters,omitempty"`
}

// Input represents the input parameters for the I2V task
type Input struct {
	Prompt   string `json:"prompt,omitempty"`
	ImgURL   string `json:"img_url"`
	VideoURL string `json:"video_url"`
}

// Params represents optional parameters for the I2V task
type Params struct {
	Duration       *int    `json:"duration,omitempty"`
	PromptExtend   *bool   `json:"prompt_extend,omitempty"`
	Seed           *int    `json:"seed,omitempty"`
	Style          *string `json:"style,omitempty"`
	VideoFPS       *int    `json:"video_fps,omitempty"`       // 生成视频的帧率，默认为15，范围区间为[15, 25]。
	AnimateEmotion *bool   `json:"animate_emotion,omitempty"` // 是否开启动作情感，默认为false。
}

// TaskResponse represents the response from the I2V service
type TaskResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID        string `json:"task_id"`
		TaskStatus    string `json:"task_status"`
		SubmitTime    string `json:"submit_time"`
		ScheduledTime string `json:"scheduled_time"`
		EndTime       string `json:"end_time"`
		VideoURL      string `json:"video_url"`
	} `json:"output"`
	Usage struct {
		VideoDuration int    `json:"video_duration"`
		VideoRatio    string `json:"video_ratio"`
		VideoCount    int    `json:"video_count"`
	} `json:"usage"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type TaskMetrics struct {
	Total     int `json:"TOTAL"`
	Succeeded int `json:"SUCCEEDED"`
	Failed    int `json:"FAILED"`
}

// QueryTaskResponse represents the response for task status query
type QueryTaskResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID         string        `json:"task_id"`
		TaskStatus     string        `json:"task_status"`
		SubmitTime     string        `json:"submit_time"`
		ScheduledTime  string        `json:"scheduled_time"`
		EndTime        string        `json:"end_time"`
		VideoURL       string        `json:"video_url"`
		TaskMetrics    TaskMetrics   `json:"task_metrics"`
		OutputVideoURL string        `json:"output_video_url"`
		Results        []ImageResult `json:"results"`
		Code           string        `json:"code"`
		Message        string        `json:"message"`
	} `json:"output"`
	Usage struct {
		VideoDuration int    `json:"video_duration"`
		VideoRatio    string `json:"video_ratio"`
		VideoCount    int    `json:"video_count"`
	} `json:"usage"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type ImageResult struct {
	OrigPrompt   string
	ActualPrompt string
	Url          string
	Code         string
	Message      string
}

type QueryTaskResponseError struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		Code       string `json:"code"`
		Message    string `json:"message"`
	} `json:"output"`
}

// TaskStatus represents the possible states of a task
const (
	TaskStatusPending   = "PENDING"
	TaskStatusRunning   = "RUNNING"
	TaskStatusSuspended = "SUSPENDED"
	TaskStatusSucceeded = "SUCCEEDED"
	TaskStatusFailed    = "FAILED"
	TaskStatusUnknown   = "UNKNOWN"
)

// NewWanxiangClient creates a new Wanxiang client
func NewWanxiangClient(apiKey string) *WanxiangClient {
	return &WanxiangClient{
		APIKey:   apiKey,
		Endpoint: "https://dashscope.aliyuncs.com",
	}
}

const (
	ImageToVideoModel = "wanx2.1-i2v-turbo"
)

// CreateTask creates a new image-to-video task
func (c *WanxiangClient) CreateTask(ctx context.Context, imgURL, prompt string, params *Params) (*TaskResponse, error) {
	req := CreateTaskRequest{
		Model: ImageToVideoModel,
		Input: Input{
			Prompt: prompt,
			ImgURL: imgURL,
		},
	}
	if params != nil {
		req.Parameters = *params
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v1/services/aigc/video-generation/video-synthesis",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add required headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &taskResp, nil
}

// QueryTask queries the status of an existing task
func (c *WanxiangClient) QueryTask(ctx context.Context, taskID string) (*QueryTaskResponse, error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s", c.Endpoint, taskID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add required headers
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var taskResp QueryTaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &taskResp, nil
}

const (
	ImageToTextModelTurbo = "wanx2.1-t2i-turbo"
	ImageToTextModelPlus  = "wanx2.1-t2i-plus"
	ImageToTextModelPro   = "wanx2.0-t2i-turbo"
)

// TextToImageRequest represents the request parameters for text-to-image generation
type TextToImageRequest struct {
	Model      string           `json:"model"`
	Input      TextToImageInput `json:"input"`
	Parameters T2IParams        `json:"parameters,omitempty"`
}

// TextToImageInput represents the input parameters for text-to-image generation
type TextToImageInput struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
}

// T2IParams represents optional parameters for text-to-image generation
type T2IParams struct {
	Size         string `json:"size,omitempty"`          // format: "width*height", e.g. "1024*1024"
	N            *int   `json:"n,omitempty"`             // 1-4, default 4
	Seed         *int   `json:"seed,omitempty"`          // 0-2147483647
	PromptExtend *bool  `json:"prompt_extend,omitempty"` // default true
	Watermark    *bool  `json:"watermark,omitempty"`     // default false
}

// CreateTextToImageTask creates a new text-to-image generation task
func (c *WanxiangClient) CreateTextToImageTask(ctx context.Context, prompt, negativePrompt string, params *T2IParams) (*TaskResponse, error) {
	req := TextToImageRequest{
		Model: ImageToTextModelTurbo,
		Input: TextToImageInput{
			Prompt:         prompt,
			NegativePrompt: negativePrompt,
		},
	}
	if params != nil {
		req.Parameters = *params
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v1/services/aigc/text2image/image-synthesis",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add required headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &taskResp, nil
}

// Helper function to create integer pointer
func IntPtr(i int) *int {
	return &i
}

// Helper function to create bool pointer
func BoolPtr(b bool) *bool {
	return &b
}

// WaitForTask waits for a task to complete with timeout
func (c *WanxiangClient) WaitForTask(ctx context.Context, taskID string, timeout time.Duration) (*QueryTaskResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task completion")
		case <-ticker.C:
			resp, err := c.QueryTask(ctx, taskID)
			if err != nil {
				return nil, err
			}

			switch resp.Output.TaskStatus {
			case TaskStatusSucceeded:
				return resp, nil
			case TaskStatusFailed:
				return nil, fmt.Errorf("task failed: %s", resp.Message)
			case TaskStatusPending, TaskStatusRunning:
				continue
			default:
				return nil, fmt.Errorf("unexpected task status: %s", resp.Output.TaskStatus)
			}
		}
	}
}

// 编辑图像
func (c *WanxiangClient) EditImage(ctx context.Context, imgURL, prompt string, params *T2IParams) (*TaskResponse, error) {
	type EditImageInput struct {
		Function     string `json:"function"`
		Prompt       string `json:"prompt"`
		BaseImageURL string `json:"base_image_url"`
		MaskImageURL string `json:"mask_image_url,omitempty"`
	}

	type EditImageRequest struct {
		Model      string         `json:"model"`
		Input      EditImageInput `json:"input"`
		Parameters T2IParams      `json:"parameters,omitempty"`
	}

	// 这里默认用 description_edit（指令编辑），如需支持更多功能可扩展参数
	input := EditImageInput{
		Function:     "description_edit",
		Prompt:       prompt,
		BaseImageURL: imgURL,
	}

	req := EditImageRequest{
		Model: "wanx2.1-imageedit",
		Input: input,
	}
	if params != nil {
		req.Parameters = *params
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v1/services/aigc/image2image/image-synthesis",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &taskResp, nil
}

// 涂鸦
func (c *WanxiangClient) SketchImage(ctx context.Context, imgURL, prompt string, params *T2IParams) (*TaskResponse, error) {
	type SketchImageInput struct {
		Prompt         string `json:"prompt"`
		SketchImageURL string `json:"sketch_image_url"`
	}

	type SketchImageParams struct {
		Style        string `json:"style,omitempty"`
		Size         string `json:"size,omitempty"`
		N            *int   `json:"n,omitempty"`
		SketchWeight *int   `json:"sketch_weight,omitempty"`
	}

	type SketchImageRequest struct {
		Model      string             `json:"model"`
		Input      SketchImageInput   `json:"input"`
		Parameters *SketchImageParams `json:"parameters,omitempty"`
	}

	input := SketchImageInput{
		Prompt:         prompt,
		SketchImageURL: imgURL,
	}

	// 默认参数
	defaultParams := &SketchImageParams{
		Style: "<auto>",
		Size:  "768*768",
		N:     nil,
	}
	if params != nil {
		if params.Size != "" {
			defaultParams.Size = params.Size
		}
		if params.N != nil {
			defaultParams.N = params.N
		}
	}

	req := SketchImageRequest{
		Model:      "wanx-sketch-to-image-lite",
		Input:      input,
		Parameters: defaultParams,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v1/services/aigc/image2image/image-synthesis",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &taskResp, nil
}

// 图像局部重绘
func (c *WanxiangClient) LocalEditImage(ctx context.Context, imgURL, prompt string, params *T2IParams, maskImageURL string) (*TaskResponse, error) {
	type LocalEditImageInput struct {
		Prompt       string `json:"prompt"`
		BaseImageURL string `json:"base_image_url"`
		MaskImageURL string `json:"mask_image_url"`
	}

	type LocalEditImageParams struct {
		Style string `json:"style,omitempty"`
		Size  string `json:"size,omitempty"`
		N     *int   `json:"n,omitempty"`
	}

	type LocalEditImageRequest struct {
		Model      string                `json:"model"`
		Input      LocalEditImageInput   `json:"input"`
		Parameters *LocalEditImageParams `json:"parameters,omitempty"`
	}

	input := LocalEditImageInput{
		Prompt:       prompt,
		BaseImageURL: imgURL,
		MaskImageURL: maskImageURL,
	}

	defaultParams := &LocalEditImageParams{
		Style: "<auto>",
		Size:  "1024*1024",
		N:     nil,
	}
	if params != nil {
		if params.Size != "" {
			defaultParams.Size = params.Size
		}
		if params.N != nil {
			defaultParams.N = params.N
		}
	}

	req := LocalEditImageRequest{
		Model:      "wanx-x-painting",
		Input:      input,
		Parameters: defaultParams,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v1/services/aigc/image2image/image-synthesis",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &taskResp, nil
}

// 人像风格重绘
func (c *WanxiangClient) PortraitStyleTransfer(ctx context.Context, imgURL, prompt string, params *T2IParams) (*TaskResponse, error) {
	type PortraitStyleInput struct {
		ImageURL    string `json:"image_url"`
		StyleIndex  int    `json:"style_index"`
		StyleRefURL string `json:"style_ref_url,omitempty"`
	}

	type PortraitStyleRequest struct {
		Model string             `json:"model"`
		Input PortraitStyleInput `json:"input"`
	}

	// 这里 prompt 作为风格描述，若为自定义风格图则 prompt 传 style_ref_url，否则传 style_index
	input := PortraitStyleInput{
		ImageURL:   imgURL,
		StyleIndex: 0, // 默认复古漫画风格
	}
	if params != nil && params.Seed != nil {
		input.StyleIndex = *params.Seed // 这里复用 Seed 字段传递风格index，实际可根据业务自定义
	}
	if prompt != "" {
		input.StyleRefURL = prompt // 若 prompt 传的是风格参考图URL
		input.StyleIndex = -1
	}

	req := PortraitStyleRequest{
		Model: "wanx-style-repaint-v1",
		Input: input,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v1/services/aigc/image-generation/generation",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &taskResp, nil
}

// 海报生成
func (c *WanxiangClient) PosterGeneration(ctx context.Context, title, subTitle, bodyText, promptZh, promptEn, whRatios, loraName string, loraWeight, ctrlRatio, ctrlStep float64, generateNum int, params *T2IParams) (*TaskResponse, error) {
	type PosterInput struct {
		Title        string  `json:"title"`
		SubTitle     string  `json:"sub_title,omitempty"`
		BodyText     string  `json:"body_text,omitempty"`
		PromptTextZh string  `json:"prompt_text_zh,omitempty"`
		PromptTextEn string  `json:"prompt_text_en,omitempty"`
		WhRatios     string  `json:"wh_ratios,omitempty"`
		LoraName     string  `json:"lora_name,omitempty"`
		LoraWeight   float64 `json:"lora_weight,omitempty"`
		CtrlRatio    float64 `json:"ctrl_ratio,omitempty"`
		CtrlStep     float64 `json:"ctrl_step,omitempty"`
		GenerateMode string  `json:"generate_mode"`
		GenerateNum  int     `json:"generate_num,omitempty"`
	}
	type PosterRequest struct {
		Model      string      `json:"model"`
		Input      PosterInput `json:"input"`
		Parameters struct{}    `json:"parameters"`
	}

	input := PosterInput{
		Title:        title,
		SubTitle:     subTitle,
		BodyText:     bodyText,
		PromptTextZh: promptZh,
		PromptTextEn: promptEn,
		WhRatios:     whRatios,
		LoraName:     loraName,
		LoraWeight:   loraWeight,
		CtrlRatio:    ctrlRatio,
		CtrlStep:     ctrlStep,
		GenerateMode: "generate",
		GenerateNum:  generateNum,
	}

	req := PosterRequest{
		Model: "wanx-poster-generation-v1",
		Input: input,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v1/services/aigc/text2image/image-synthesis",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &taskResp, nil
}

// 人物分隔（人像分割）
func (c *WanxiangClient) PersonSegmentation(ctx context.Context, imgURL string) (*TaskResponse, error) {
	type task struct {
		DataId string `json:"dataId,omitempty"`
		Url    string `json:"url"`
	}
	type request struct {
		Scenes []string `json:"scenes"`
		Tasks  []task   `json:"tasks"`
	}
	type response struct {
		Code      int    `json:"code"`
		Msg       string `json:"msg"`
		RequestId string `json:"requestId"`
		Data      []struct {
			Code   int    `json:"code"`
			Msg    string `json:"msg"`
			DataId string `json:"dataId"`
			TaskId string `json:"taskId"`
			Url    string `json:"url"`
		} `json:"data"`
	}

	req := request{
		Scenes: []string{"sface"},
		Tasks:  []task{{Url: imgURL}},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	endpoint := c.Endpoint
	if endpoint == "https://dashscope.aliyuncs.com" {
		endpoint = "https://green.cn-shanghai.aliyuncs.com" // 内容安全API默认区域
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		endpoint+"/green/image/asyncscan",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res response
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if res.Code != 200 || len(res.Data) == 0 || res.Data[0].Code != 200 {
		return nil, fmt.Errorf("api error: %s", res.Msg)
	}

	// 兼容 TaskResponse 结构，返回 taskId
	taskResp := &TaskResponse{
		RequestID: res.RequestId,
	}
	taskResp.Output.TaskID = res.Data[0].TaskId
	taskResp.Output.TaskStatus = "PENDING"
	taskResp.Output.SubmitTime = time.Now().Format(time.RFC3339)
	taskResp.Output.VideoURL = "" // 无视频
	return taskResp, nil
}
