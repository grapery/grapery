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
