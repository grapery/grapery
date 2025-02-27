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

/*
HTTP调用
图生视频模型处理时间较长，为了避免请求超时，HTTP调用仅支持异步获取模型结果。您需要发起两个请求：

创建任务：首先发送一个请求创建任务，该请求会返回任务ID。

根据任务ID查询结果：使用上一步获得的任务ID，查询模型生成的结果。

图生视频耗时较长，turbo模型大约需要3-5分钟，plus模型则需7-10分钟。实际耗时取决于排队任务数量和网络状况，请您在获取结果时耐心等待。

1.创建任务
POST https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis
curl 样例
curl --location 'https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis' \
    -H 'X-DashScope-Async: enable' \
    -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
    -H 'Content-Type: application/json' \
    -d '{
    "model": "wanx2.1-i2v-turbo",
    "input": {
        "prompt": "一只猫在草地上奔跑",
        "img_url": "https://cdn.translate.alibaba.com/r/wanx-demo-1.png"

    },
    "parameters": {
        "prompt_extend": true
    }
}'

请求头（Headers）
Content-Type string （必选）

请求内容类型。此参数必须设置为application/json。

Authorization string（必选）

请求身份认证。接口使用百炼API-Key进行身份认证。示例值：Bearer d1xxx2a。

X-DashScope-Async string （必选）

异步处理配置参数。HTTP请求只支持异步，必须设置为enable。

请求体（Request Body）

请求参数
图生视频

curl --location 'https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis' \
    -H 'X-DashScope-Async: enable' \
    -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
    -H 'Content-Type: application/json' \
    -d '{
    "model": "wanx2.1-i2v-turbo",
    "input": {
        "prompt": "一只猫在草地上奔跑",
        "img_url": "https://cdn.translate.alibaba.com/r/wanx-demo-1.png"

    },
    "parameters": {
        "prompt_extend": true
    }
}'
请求头（Headers）
Content-Type string （必选）

请求内容类型。此参数必须设置为application/json。

Authorization string（必选）

请求身份认证。接口使用百炼API-Key进行身份认证。示例值：Bearer d1xxx2a。

X-DashScope-Async string （必选）

异步处理配置参数。HTTP请求只支持异步，必须设置为enable。

请求体（Request Body）
model string （必选）

模型名称。示例值：wanx2.1-i2v-turbo。
input object （必选）
输入的基本信息，如提示词等。
属性
	prompt string （可选）
		文本提示词。支持中英文，长度不超过800个字符，每个汉字/字母占一个字符，超过部分会自动截断。
		示例值：一只小猫在草地上奔跑。
		提示词的使用技巧请参见文生/图生视频Prompt使用指南。

	img_url string （必选）
		生成视频时所使用的第一帧图像的URL。
		图像限制：
		图像格式：JPEG、JPG、PNG（不支持透明通道）、BMP、WEBP。
		文件大小：不超过10 MB。
		分辨率：360≤图像边长≤2000，单位像素。
parameters object （可选）

视频处理参数。

属性

	duration integer （可选）
		生成视频的时长，默认值为5，单位为秒。
		wanx2.1-i2v-turbo：可选值为3、4或5。
		wanx2.1-i2v-plus：目前仅支持5秒固定时长生成。
	prompt_extend bool （可选）
		是否开启prompt智能改写。开启后使用大模型对输入prompt进行智能改写。对于较短的prompt生成效果提升明显，但会增加耗时。
		true：默认值，开启智能改写。
		false：不开启智能改写。
	seed integer （可选）
		随机数种子，用于控制模型生成内容的随机性。取值范围为[0, 2147483647]。
		如果不提供，则算法自动生成一个随机数作为种子。如果希望生成内容保持相对稳定，可以使用相同的seed参数值。

返回的响应数据
	output object

任务输出信息。

属性
task_id string
	任务ID。
task_status string
	任务状态。
	枚举值
	PENDING：任务排队中
	RUNNING：任务处理中
	SUSPENDED：任务挂起
	SUCCEEDED：任务执行成功
	FAILED：任务执行失败
	UNKNOWN：任务不存在或状态未知
request_id string
	请求唯一标识。可用于请求明细溯源和问题排查。
code string
	请求失败的错误码。请求成功时不会返回此参数，详情请参见错误信息。
message string
	请求失败的详细信息。请求成功时不会返回此参数，详情请参见错误信息。

示例：
成功的返回响应：
{
    "output": {
        "task_status": "PENDING",
        "task_id": "0385dc79-5ff8-4d82-bcb6-xxxxxx"
    },
    "request_id": "4909100c-7b5a-9f92-bfe5-xxxxxx"
}
异常的返回响应:
{
    "code":"InvalidApiKey",
    "message":"Invalid API-key provided.",
    "request_id":"fb53c4ec-1c12-4fc4-a580-xxxxxx"
}
*/

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
	Prompt string `json:"prompt,omitempty"`
	ImgURL string `json:"img_url"`
}

// Params represents optional parameters for the I2V task
type Params struct {
	Duration     *int  `json:"duration,omitempty"`
	PromptExtend *bool `json:"prompt_extend,omitempty"`
	Seed         *int  `json:"seed,omitempty"`
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

// QueryTaskResponse represents the response for task status query
type QueryTaskResponse struct {
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

// CreateTask creates a new image-to-video task
func (c *WanxiangClient) CreateTask(ctx context.Context, imgURL, prompt string, params *Params) (*TaskResponse, error) {
	req := CreateTaskRequest{
		Model: "wanx2.1-i2v-turbo",
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
