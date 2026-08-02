package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/grapery/grapery/pkg/cloud/aliyun"
)

var (
	DashScopeAPIKey = os.Getenv("DASHSCOPE_API_KEY")
)

type AliyunStoryClient struct {
	Client          *aliyun.AliyunClient
	DashScopeAPIKey string
}

func NewAliyunClient() *AliyunStoryClient {
	client, _ := aliyun.NewAliyunClient()
	DashScopeAPIKey = os.Getenv("DASHSCOPE_API_KEY")
	if DashScopeAPIKey == "" {
		panic("DASHSCOPE_API_KEY is not set")
	}
	return &AliyunStoryClient{
		Client:          client,
		DashScopeAPIKey: DashScopeAPIKey,
	}
}

type DashScopeTextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DashScopeInput struct {
	Messages []DashScopeTextMessage `json:"messages"`
}

type DashScopeTextParameters struct {
	ResultFormat string `json:"result_format"`
}

type DashScopeTextRequestBody struct {
	Model      string                  `json:"model"`
	Input      DashScopeInput          `json:"input"`
	Parameters DashScopeTextParameters `json:"parameters"`
}

func (d DashScopeTextRequestBody) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type DashScopeTextResponse struct {
	Output struct {
		FinishReason string `json:"finish_reason"`
		Text         string `json:"text"`
		Choices      []struct {
			Message struct {
				Content string `json:"content"`
				Role    string `json:"role"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`

	Usage struct {
		TotalTokens         int `json:"total_tokens"`
		OutputTokens        int `json:"output_tokens"`
		InputTokens         int `json:"input_tokens"`
		PromptTokensDetails struct {
			PromptTokens        int `json:"prompt_tokens"`
			PromptTokensDetails []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"prompt_tokens_details"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

func (d DashScopeTextResponse) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

func (c *AliyunStoryClient) GenStoryInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeTextRequestBody{
		Model: "qwen-plus",
		Input: DashScopeInput{
			Messages: []DashScopeTextMessage{
				{
					Role:    "system",
					Content: params.RoleDesc,
				},
				{
					Role:    "user",
					Content: params.Content,
				},
			},
		},
		Parameters: DashScopeTextParameters{
			ResultFormat: "json_object",
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &DashScopeTextResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		return nil, err
	}
	return &StoryInfoResult{
		Content: ret.Output.Text,
	}, nil
}

func (c *AliyunStoryClient) GenStoryBoardInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeTextRequestBody{
		Model: "qwen-plus",
		Input: DashScopeInput{
			Messages: []DashScopeTextMessage{
				{
					Role:    "system",
					Content: params.RoleDesc,
				},
				{
					Role:    "user",
					Content: params.Content,
				},
			},
		},
		Parameters: DashScopeTextParameters{
			ResultFormat: "json_object",
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &DashScopeTextResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		return nil, err
	}
	return &StoryInfoResult{
		Content: ret.Output.Text,
	}, nil
}

func (c *AliyunStoryClient) GenStoryRoleInfo(ctx context.Context, params *GenStoryCharactorParams) (*GenStoryCharactorResult, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeTextRequestBody{
		Model: "qwen-plus",
		Input: DashScopeInput{
			Messages: []DashScopeTextMessage{
				{
					Role:    "system",
					Content: "你是一个资深作家、小说家，根据输入的角色的简介以及描述，丰富角色的描述",
				},
				{
					Role:    "user",
					Content: params.Content,
				},
			},
		},
		Parameters: DashScopeTextParameters{
			ResultFormat: "json_object",
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Default().Println("Error reading response body:", err)
		return nil, err
	}
	log.Default().Printf("Response body: %s", bodyText)
	ret := &DashScopeTextResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		log.Default().Println("Error unmarshalling response:", err)
		return nil, err
	}
	log.Default().Printf("Generated character info: %s", ret.String())

	return &GenStoryCharactorResult{
		Content: ret.Output.Text,
	}, nil
}

func (c *AliyunStoryClient) ChatWithRole(ctx context.Context, params *ChatWithRoleParams) (*ChatWithRoleResult, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeTextRequestBody{
		Model: "qwen-plus",
		Input: DashScopeInput{
			Messages: []DashScopeTextMessage{
				{
					Role:    "system",
					Content: params.Role,
				},
				{
					Role:    "user",
					Content: params.MessageContent,
				},
			},
		},
		Parameters: DashScopeTextParameters{
			ResultFormat: "json_object",
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Default().Printf("Response body: %s", bodyText)
	ret := &DashScopeTextResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		log.Default().Println("Error unmarshalling response:", err)
		return nil, err
	}
	log.Default().Printf("chat message info: %s", ret.String())
	return &ChatWithRoleResult{
		Content: ret.Output.Text,
	}, nil
}

/*
curl -X POST https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis \
    -H 'X-DashScope-Async: enable' \
    -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
    -H 'Content-Type: application/json' \
    -d '{
    "model": "wanx2.1-t2i-turbo",
    "input": {
        "prompt": "雪地，白色小教堂，极光，冬日场景，柔和的光线。",
        "negative_prompt": "人物"
    },
    "parameters": {
        "size": "1024*1024",
        "n": 1
    }
}'
*/
/*
{
    "model": "wanx2.1-t2i-turbo",
    "input": {
        "prompt": "雪地，白色小教堂，极光，冬日场景，柔和的光线。",
        "negative_prompt": "人物"
    },
    "parameters": {
        "size": "1024*1024",
        "n": 1
    }
*/

type DashScopeImageInput struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt"`
	Function       string `json:"function,omitempty"`       // 可选字段，用于指定图像生成的功能
	BaseImageURL   string `json:"base_image_url,omitempty"` // 可选字段，用于指定基础图像的URL
	MaskImageURL   string `json:"mask_image_url,omitempty"` // 可选字段，用于指定遮罩图像的URL
}

func (d DashScopeImageInput) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

/*
"title":"春节快乐",
"sub_title":"家庭团聚，共享天伦之乐",
"body_text":"春节是中国最重要的传统节日之一，它象征着新的开始和希望",
"prompt_text_zh":"灯笼，小猫，梅花",
"wh_ratios":"竖版",
"lora_name":"童话油画",
"lora_weight":0.8,
"ctrl_ratio":0.7,
"ctrl_step":0.7,
"generate_mode":"generate",
"generate_num":1
*/
type DashScopePosterImageInput struct {
	Title        string  `json:"title"`                 // 海报标题
	SubTitle     string  `json:"sub_title"`             // 海报副标题
	BodyText     string  `json:"body_text"`             // 海报正文文本
	PromptTextZH string  `json:"prompt_text_zh"`        // 中文提示词
	WhRatios     string  `json:"wh_ratios"`             // 宽高比
	LoraName     string  `json:"lora_name,omitempty"`   // 可选字段，用于指定Lora模型名称
	LoraWeight   float64 `json:"lora_weight,omitempty"` // 可选字段，用于指定Lora模型权重
	CtrlRatio    float64 `json:"ctrl_ratio,omitempty"`  // 可选字段，用于控制生成图像的比例
	CtrlStep     float64 `json:"ctrl_step,omitempty"`   // 可选字段，用于控制生成图像的步长
	GenerateMode string  `json:"generate_mode"`         // 生成模式，默认为"generate"
	GenerateNum  int     `json:"generate_num"`          // 生成图像的数量
}

func (d DashScopePosterImageInput) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type DashScopeGenDescImageInput struct {
	Title    []string `json:"title"`     // 海报标题
	SubTitle []string `json:"subtitle"`  // 海报副标题
	Text     []string `json:"text"`      // 海报正文文本
	ImageURL string   `json:"image_url"` // 图像URL
	Underlay int      `json:"underlay"`  // 是否使用底图，1表示使用，0表示不使用
	Logo     string   `json:"logo"`      // 可选字段，用于指定Logo的URL
}

func (d DashScopeGenDescImageInput) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type DashScopeImageInputParams struct {
	Size         string  `json:"size"`
	N            int     `json:"n"`
	PromptExtend bool    `json:"prompt_extend,omitempty"` // 可选字段，用于扩展提示词
	Watermark    bool    `json:"watermark,omitempty"`     // 可选字段，用于添加水印
	IsSketch     bool    `json:"is_sketch,omitempty"`     // 可选字段，用于指定是否为草图
	Temperature  float64 `json:"temperature,omitempty"`   // 可选字段，用于控制生成图像的温度
}

func (d DashScopeImageInputParams) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type DashScopeImageInputRequest struct {
	Model      string                    `json:"model"`
	Input      interface{}               `json:"input"`
	Parameters DashScopeImageInputParams `json:"parameters"`
}

type DashScopeTaskStatus struct {
	TaskStatus string `json:"task_status"` // 任务状态
	TaskID     string `json:"task_id"`     // 任务ID
}

type DashScopeGenStoryImagesResponse struct {
	Output    DashScopeTaskStatus `json:"output"`     // 任务状态
	RequestID string              `json:"request_id"` // 请求ID
	Code      string              `json:"code"`       // 响应码
	Message   string              `json:"message"`    // 响应消息
}

func (d DashScopeGenStoryImagesResponse) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

func (c *AliyunStoryClient) GenStoryBoardImages(ctx context.Context, params *GenStoryImagesParams) (*DashScopeGenStoryImagesResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeImageInputRequest{
		Model: "wanx2.1-t2i-turbo",
		Input: DashScopeImageInput{
			Prompt:         params.Content,
			NegativePrompt: params.NegativePrompt,
		},
		Parameters: DashScopeImageInputParams{
			Size:         "1024*1024",
			N:            1,
			PromptExtend: true, // 默认开启提示词扩展
			Watermark:    true, // 默认添加水印
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	log.Default().Printf("Request body: %s", jsonData)
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Default().Println("Error creating request:", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 设置异步标志
	req.Header.Set("X-DashScope-Async", "enable")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		log.Default().Println("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()
	// 处理异步响应
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		log.Default().Printf("Unexpected status code: %d", resp.StatusCode)
		return nil, errors.New("failed to start image generation task")
	}
	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Default().Println("Error reading response body:", err)
		// 如果读取失败，返回原始错误
		return nil, err
	}
	log.Default().Println("Response body:", string(bodyText))
	ret := &DashScopeGenStoryImagesResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		log.Default().Println("Error unmarshalling response:", err)
		// 如果解析失败，返回原始响应体以便调试
		return nil, err
	}
	log.Default().Printf("Image generation task started successfully, Task: %s", ret.String())
	// 返回任务状态和请求ID

	return ret, nil
}

const (
	// TaskStatusPending 任务排队中
	TaskStatusPending string = "PENDING"
	// TaskStatusRunning 任务处理中
	TaskStatusRunning string = "RUNNING"
	// TaskStatusSucceeded 任务执行成功
	TaskStatusSucceeded string = "SUCCEEDED"
	// TaskStatusFailed 任务执行失败
	TaskStatusFailed string = "FAILED"
	// TaskStatusCanceled 任务取消成功
	TaskStatusCanceled string = "CANCELED"
	// TaskStatusUnknown 任务不存在或状态未知
	TaskStatusUnknown string = "UNKNOWN"
)

type DashScopeTaskStatusResponse struct {
	RequestID string `json:"request_id"` // 请求ID
	Output    struct {
		TaskID        string `json:"task_id"`        // 任务ID
		TaskStatus    string `json:"task_status"`    // 任务状态
		SubmitTime    string `json:"submit_time"`    // 提交时间
		ScheduledTime string `json:"scheduled_time"` // 调度时间
		EndTime       string `json:"end_time"`       // 结束时间
		Results       []struct {
			OrigPrompt   string `json:"orig_prompt"`   // 原始提示词
			ActualPrompt string `json:"actual_prompt"` // 实际提示词
			URL          string `json:"url"`           // 生成的图像URL
		} `json:"results"` // 生成的图像结果
		ResultsUrls []string `json:"results_urls,omitempty"` // 生成的图像URL列表（可选）
		BgUrls      []string `json:"bg_urls,omitempty"`      // 背景图像URL列表（可选）
		Video_url   string   `json:"video_url,omitempty"`    // 生成的视频URL（可选）
		TaskMetrics struct {
			Total     int `json:"TOTAL"`     // 总任务数
			Succeeded int `json:"SUCCEEDED"` // 成功任务数
			Failed    int `json:"FAILED"`    // 失败任务数
		} `json:"task_metrics"` // 任务指标
	} `json:"output"` // 输出结果
	Usage struct {
		ImageCount int `json:"image_count"` // 生成的图像数量
	} `json:"usage"` // 使用情况
}

func (d DashScopeTaskStatusResponse) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

func (c *AliyunStoryClient) GetImageGenerationTaskStatus(ctx context.Context, taskID string) (*DashScopeTaskStatusResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求 URL
	url := "https://dashscope.aliyuncs.com/api/v1/tasks/" + taskID
	// 创建 GET 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &DashScopeTaskStatusResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *AliyunStoryClient) RepaintingStoryBoardImages(ctx context.Context, params *GenStoryImagesParams) (*DashScopeGenStoryImagesResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeImageInputRequest{
		Model: "wanx2.1-t2i-turbo",
		Input: DashScopeImageInput{
			Prompt:         params.Prompt,
			NegativePrompt: params.NegativePrompt,
			Function:       "description_edit_with_mask", // 指定为重绘功能
			BaseImageURL:   params.RefImage,              // 基础图像的URL
			MaskImageURL:   params.MaskImageUrl,          // 掩码图像的URL
		},
		Parameters: DashScopeImageInputParams{
			Size:         "1024*1024",
			N:            1,
			PromptExtend: true, // 默认开启提示词扩展
			Watermark:    true, // 默认添加水印
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 设置异步标志
	req.Header.Set("X-DashScope-Async", "enable")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 处理异步响应
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to start image generation task")
	}
	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Default().Println("Error reading response body:", err)
		// 如果读取失败，返回原始错误
		return nil, err
	}
	ret := &DashScopeGenStoryImagesResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		log.Default().Println("Error unmarshalling response:", err)
		// 如果解析失败，返回原始响应体以便调试
		return nil, err
	}
	log.Default().Printf("Image generation task started successfully, Task ID: %s, Status: %s", ret.Output.TaskID, ret.Output.TaskStatus)
	// 返回任务状态和请求ID
	return ret, nil
}

func (c *AliyunStoryClient) SketchStoryBoardImages(ctx context.Context, params *GenStoryImagesParams) (*DashScopeGenStoryImagesResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeImageInputRequest{
		Model: "wanx2.1-t2i-turbo",
		Input: DashScopeImageInput{
			Prompt:         params.Prompt,
			NegativePrompt: params.NegativePrompt,
			Function:       "doodle",            // 指定为重绘功能
			BaseImageURL:   params.RefImage,     // 基础图像的URL
			MaskImageURL:   params.MaskImageUrl, // 掩码图像的URL
		},
		Parameters: DashScopeImageInputParams{
			Size:         "1024*1024",
			N:            1,
			PromptExtend: true, // 默认开启提示词扩展
			Watermark:    true, // 默认添加水印
			IsSketch:     true, // 指定为草图模式
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 设置异步标志
	req.Header.Set("X-DashScope-Async", "enable")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 处理异步响应
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to start image generation task")
	}
	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Default().Println("Error reading response body:", err)
		// 如果读取失败，返回原始错误
		return nil, err
	}
	ret := &DashScopeGenStoryImagesResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		log.Default().Println("Error unmarshalling response:", err)
		// 如果解析失败，返回原始响应体以便调试
		return nil, err
	}
	log.Default().Printf("Image generation task started successfully, Task ID: %s, Status: %s", ret.Output.TaskID, ret.Output.TaskStatus)
	// 返回任务状态和请求ID
	return ret, nil
}

func (c *AliyunStoryClient) StoryPosterImages(ctx context.Context, params *GenStoryPosterParams) (*DashScopeGenStoryImagesResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeImageInputRequest{
		Model: "wanx-poster-generation-v1",
		Input: DashScopePosterImageInput{
			Title:        params.Title,
			SubTitle:     params.SubTitle,
			BodyText:     params.BodyText,
			PromptTextZH: params.PromptTextZh,
			WhRatios:     params.WhRatios,
			LoraName:     params.LoraName,     // 可选字段，用于指定Lora模型名称
			LoraWeight:   params.LoraWeight,   // 可选字段，用于指定Lora模型权重
			CtrlRatio:    params.CtrlRatio,    // 可选字段，用于控制生成图像的比例
			CtrlStep:     params.CtrlStep,     // 可选字段，用于控制生成图像的步长
			GenerateMode: params.GenerateMode, // 生成模式，默认为"generate"
			GenerateNum:  params.GenerateNum,  // 生成图像的数量
		},
		Parameters: DashScopeImageInputParams{
			Size:         "1024*1024",
			N:            1,
			PromptExtend: true,  // 默认开启提示词扩展
			Watermark:    true,  // 默认添加水印
			IsSketch:     false, // 指定为草图模式
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 设置异步标志
	req.Header.Set("X-DashScope-Async", "enable")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 处理异步响应
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to start image generation task")
	}
	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Default().Println("Error reading response body:", err)
		// 如果读取失败，返回原始错误
		return nil, err
	}
	ret := &DashScopeGenStoryImagesResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		log.Default().Println("Error unmarshalling response:", err)
		// 如果解析失败，返回原始响应体以便调试
		return nil, err
	}
	log.Default().Printf("Image generation task started successfully, Task ID: %s, Status: %s", ret.Output.TaskID, ret.Output.TaskStatus)
	// 返回任务状态和请求ID
	return ret, nil
}

func (c *AliyunStoryClient) DescStoryImages(ctx context.Context, params *DashScopeGenDescImageInput) (*DashScopeGenStoryImagesResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	requestBody := DashScopeImageInputRequest{
		Model: "wanx-ast",
		Input: params,
		Parameters: DashScopeImageInputParams{
			Size:         "1024*1024",
			N:            1,
			PromptExtend: true,  // 默认开启提示词扩展
			Watermark:    true,  // 默认添加水印
			IsSketch:     false, // 指定为草图模式
			Temperature:  0.7,   // 可选字段，用于控制生成图像的温度
		},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 设置异步标志
	req.Header.Set("X-DashScope-Async", "enable")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 处理异步响应
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to start image generation task")
	}
	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Default().Println("Error reading response body:", err)
		// 如果读取失败，返回原始错误
		return nil, err
	}
	ret := &DashScopeGenStoryImagesResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		log.Default().Println("Error unmarshalling response:", err)
		// 如果解析失败，返回原始响应体以便调试
		return nil, err
	}
	log.Default().Printf("Image generation task started successfully, Task ID: %s, Status: %s", ret.Output.TaskID, ret.Output.TaskStatus)
	// 返回任务状态和请求ID
	return ret, nil
}

func (c *AliyunStoryClient) SetDashScopeAPIKey(apiKey string) {
	c.DashScopeAPIKey = apiKey
	log.Default().Printf("Set DashScope API Key: %s", apiKey)
}

func (c *AliyunStoryClient) GetDashScopeAPIKey() string {
	if c.DashScopeAPIKey == "" {
		log.Default().Println("DashScope API Key is not set")
		return ""
	}
	log.Default().Printf("Get DashScope API Key: %s", c.DashScopeAPIKey)
	return c.DashScopeAPIKey
}

type DashScopeVideoInput struct {
	Prompt         string   `json:"prompt"`                    // 视频生成的提示词
	ImageURL       string   `json:"image_url"`                 // 可选字段，用于指定首帧图像的URL
	FirstFrameURL  string   `json:"first_frame_url,omitempty"` // 可选字段，用于指定首帧图像的URL
	LastFrameURL   string   `json:"last_frame_url,omitempty"`  // 可选字段，用于指定尾帧图像的URL
	NegativePrompt string   `json:"negative_prompt,omitempty"` // 可选字段，用于指定负面提示词
	Function       string   `json:"function,omitempty"`        // 可选字段，用于指定视频生成的功能
	RefImagesUrl   []string `json:"ref_images_url,omitempty"`  // 可选字段，用于指定参考图像的URL列表
}

func (d DashScopeVideoInput) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type DashScopeVideoParameters struct {
	Resolution   string   `json:"resolution"`              // 视频分辨率，例如 "720P,480P"
	PromptExtend bool     `json:"prompt_extend,omitempty"` // 可选字段，用于扩展提示词
	Duration     int      `json:"duration"`                // 视频时长，单位为秒
	ObjOrBg      []string `json:"obj_or_bg,omitempty"`     // 可选字段，用于指定视频中的对象或背景
	Watermark    bool     `json:"watermark,omitempty"`     // 可选字段，用于添加水印
}

func (d DashScopeVideoParameters) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

type DashScopeVideoRequestBody struct {
	Model      string                   `json:"model"`
	Input      DashScopeVideoInput      `json:"input"`
	Parameters DashScopeVideoParameters `json:"parameters"`
}

func (d DashScopeVideoRequestBody) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

// 基于首帧的视频生成
func (c *AliyunStoryClient) GenVideoFromFirstFrame(ctx context.Context, params *DashScopeVideoRequestBody) (*DashScopeGenStoryImagesResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求体
	params.Model = "wanx2.1-i2v-turbo" // 使用适合首帧视频生成的模型
	requestBody := params
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &DashScopeGenStoryImagesResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// 首尾帧生视频
func (c *AliyunStoryClient) GenVideoFromFirstAndLastFrame(ctx context.Context, params *DashScopeVideoRequestBody) (*DashScopeGenStoryImagesResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	params.Model = "wanx2.1-kf2v-plus" // 使用适合首尾帧视频生成的模型
	// 构建请求体
	requestBody := params
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &DashScopeGenStoryImagesResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// 多图参考编辑
func (c *AliyunStoryClient) MultiImageGenVideo(ctx context.Context, params *DashScopeVideoRequestBody) (*DashScopeGenStoryImagesResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	params.Model = "wanx2.1-vace-plus" // 使用适合首尾帧视频生成的模型
	// 构建请求体
	requestBody := params
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	// 创建 POST 请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &DashScopeGenStoryImagesResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type DashScopeVideoTaskStatusResponse struct {
	RequestID string `json:"request_id"` // 请求ID
	Output    struct {
		TaskID        string `json:"task_id"`             // 任务ID
		TaskStatus    string `json:"task_status"`         // 任务状态
		SubmitTime    string `json:"submit_time"`         // 提交时间
		ScheduledTime string `json:"scheduled_time"`      // 调度时间
		EndTime       string `json:"end_time"`            // 结束时间
		Video_url     string `json:"video_url,omitempty"` // 生成的视频URL（可选）
		OrigPrompt    string `json:"orig_prompt"`         // 原始提示词
		ActualPrompt  string `json:"actual_prompt"`       // 实际提示词
		TaskMetrics   struct {
			Total     int `json:"TOTAL"`     // 总任务数
			Succeeded int `json:"SUCCEEDED"` // 成功任务数
			Failed    int `json:"FAILED"`    // 失败任务数
		} `json:"task_metrics"` // 任务指标
	} `json:"output"` // 输出结果
	Usage struct {
		VideoDuration int    `json:"video_duration"` // 生成的视频时长，单位为秒
		VideoCount    int    `json:"video_count"`    // 生成的视频数量
		VideoRatio    string `json:"video_ratio"`    // 生成的图像数量

	} `json:"usage"` // 使用情况
}

func (d DashScopeVideoTaskStatusResponse) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

func (c *AliyunStoryClient) GetVideoGenerationTaskStatus(ctx context.Context, taskID string) (*DashScopeTaskStatusResponse, error) {
	// 创建 HTTP 客户端
	client := &http.Client{}
	// 构建请求 URL
	url := "https://dashscope.aliyuncs.com/api/v1/tasks/" + taskID
	// 创建 GET 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.DashScopeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 读取响应体
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := &DashScopeTaskStatusResponse{}
	err = json.Unmarshal(bodyText, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
