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

// 根据图象生成视频
func (c *WanxiangClient) GenerateVideoFromImage(ctx context.Context, imgURL, prompt string, params *T2IParams) (*TaskResponse, error) {
	type input struct {
		Prompt string `json:"prompt"`
		ImgURL string `json:"img_url"`
	}
	type parameters struct {
		Resolution   string `json:"resolution,omitempty"`
		Duration     int    `json:"duration,omitempty"`
		PromptExtend *bool  `json:"prompt_extend,omitempty"`
		Seed         *int   `json:"seed,omitempty"`
	}
	type request struct {
		Model      string     `json:"model"`
		Input      input      `json:"input"`
		Parameters parameters `json:"parameters,omitempty"`
	}
	type response struct {
		RequestID string `json:"request_id"`
		Output    struct {
			TaskID        string `json:"task_id"`
			TaskStatus    string `json:"task_status"`
			VideoURL      string `json:"video_url,omitempty"`
			SubmitTime    string `json:"submit_time,omitempty"`
			ScheduledTime string `json:"scheduled_time,omitempty"`
			EndTime       string `json:"end_time,omitempty"`
			Code          string `json:"code,omitempty"`
			Message       string `json:"message,omitempty"`
		} `json:"output"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}

	// 构造请求体
	model := "wan2.1-i2v-turbo"
	if params != nil && params.Size == "720P" {
		model = "wan2.1-i2v-plus" // plus模型仅支持720P
	}
	in := input{
		Prompt: prompt,
		ImgURL: imgURL,
	}
	p := parameters{}
	if params != nil {
		if params.Size != "" {
			p.Resolution = params.Size // "480P" 或 "720P"
		}
		if params.N != nil {
			p.Duration = *params.N // 用N字段传递时长（秒），如需更细致可扩展
		}
		if params.PromptExtend != nil {
			p.PromptExtend = params.PromptExtend
		}
		if params.Seed != nil {
			p.Seed = params.Seed
		}
	}
	req := request{
		Model:      model,
		Input:      in,
		Parameters: p,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://dashscope-intl.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	client := &http.Client{Timeout: 60 * time.Second}
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

	if res.Output.TaskID == "" {
		return nil, fmt.Errorf("api error: %s", res.Message)
	}

	taskResp := &TaskResponse{
		RequestID: res.RequestID,
	}
	taskResp.Output.TaskID = res.Output.TaskID
	taskResp.Output.TaskStatus = res.Output.TaskStatus
	taskResp.Output.SubmitTime = res.Output.SubmitTime
	taskResp.Output.ScheduledTime = res.Output.ScheduledTime
	taskResp.Output.EndTime = res.Output.EndTime
	taskResp.Output.VideoURL = res.Output.VideoURL
	return taskResp, nil
}

// 视频风格转绘
func (c *WanxiangClient) VideoStyleTransfer(ctx context.Context, videoURL, prompt string, params *T2IParams) (*TaskResponse, error) {
	type input struct {
		Prompt   string `json:"prompt"`
		VideoURL string `json:"video_url"`
	}
	type parameters struct {
		Resolution   string `json:"resolution,omitempty"`
		PromptExtend *bool  `json:"prompt_extend,omitempty"`
		Seed         *int   `json:"seed,omitempty"`
	}
	type request struct {
		Model      string     `json:"model"`
		Input      input      `json:"input"`
		Parameters parameters `json:"parameters,omitempty"`
	}
	type response struct {
		RequestID string `json:"request_id"`
		Output    struct {
			TaskID        string `json:"task_id"`
			TaskStatus    string `json:"task_status"`
			VideoURL      string `json:"video_url,omitempty"`
			SubmitTime    string `json:"submit_time,omitempty"`
			ScheduledTime string `json:"scheduled_time,omitempty"`
			EndTime       string `json:"end_time,omitempty"`
			Code          string `json:"code,omitempty"`
			Message       string `json:"message,omitempty"`
		} `json:"output"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}

	model := "wanx2.1-v2v-turbo" // 假设官方API模型名，实际以文档为准
	in := input{
		Prompt:   prompt,
		VideoURL: videoURL,
	}
	p := parameters{}
	if params != nil {
		if params.Size != "" {
			p.Resolution = params.Size // "480P" 或 "720P"
		}
		if params.PromptExtend != nil {
			p.PromptExtend = params.PromptExtend
		}
		if params.Seed != nil {
			p.Seed = params.Seed
		}
	}
	req := request{
		Model:      model,
		Input:      in,
		Parameters: p,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	client := &http.Client{Timeout: 60 * time.Second}
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

	if res.Output.TaskID == "" {
		return nil, fmt.Errorf("api error: %s", res.Message)
	}

	taskResp := &TaskResponse{
		RequestID: res.RequestID,
	}
	taskResp.Output.TaskID = res.Output.TaskID
	taskResp.Output.TaskStatus = res.Output.TaskStatus
	taskResp.Output.SubmitTime = res.Output.SubmitTime
	taskResp.Output.ScheduledTime = res.Output.ScheduledTime
	taskResp.Output.EndTime = res.Output.EndTime
	taskResp.Output.VideoURL = res.Output.VideoURL
	return taskResp, nil
}
