package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	DoubaoAPIKey = "doubao"
)

type DoubaoClient struct {
	DoubaoAPIKey string
}

func NewDoubaoClient() *DoubaoClient {
	return &DoubaoClient{
		DoubaoAPIKey: DoubaoAPIKey,
	}
}

/*
	curl -X POST https://ark.cn-beijing.volces.com/api/v3/images/generations \
	  -H "Content-Type: application/json" \
	  -H "Authorization: Bearer doubao" \
	  -d '{
	    "model": "doubao-seedream-3-0-t2i-250415",
	    "prompt": "一只可爱小猫咪",
	    "response_format": "url",
	    "size": "1024x1024",
	    "guidance_scale": 3,
	    "watermark": true
	}'
*/

type DoubaoGenImageParams struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	ResponseFormat string `json:"response_format"`
	Size           string `json:"size"`
	GuidanceScale  int    `json:"guidance_scale"`
	Watermark      bool   `json:"watermark"`
}

func (d DoubaoGenImageParams) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

/*
{
    "model": "doubao-seedream-3-0-t2i-250415",
    "created": 1750120047,
    "data": [
        {
            "url": "https://ar"
        }
    ],
    "usage": {
        "generated_images": 1
    }
}
*/

type DoubaoGenImageResult struct {
	Model   string `json:"model"`
	Created int    `json:"created"`
	Data    []struct {
		URL string `json:"url"`
	} `json:"data"`
	Usage struct {
		GeneratedImages int `json:"generated_images"`
	} `json:"usage"`
}

func (c *DoubaoClient) GenStoryBoardImage(ctx context.Context, params *GenStoryImagesParams) (*GenStoryImagesResult, error) {
	realParams := new(DoubaoGenImageParams)
	realParams.Prompt = params.Content
	realParams.Watermark = true
	realParams.Model = "doubao-seedream-3-0-t2i-250415"
	realParams.Size = "1024x1024"
	realParams.GuidanceScale = 3
	realParams.ResponseFormat = "url"
	// 1. 序列化请求参数为 JSON
	body, err := json.Marshal(realParams)
	if err != nil {
		return nil, err
	}

	// 2. 构造 HTTP 请求
	url := "https://ark.cn-beijing.volces.com/api/v3/images/generations"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 3. 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.DoubaoAPIKey)

	// 4. 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 5. 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 6. 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// 7. 解析响应体为 DoubaoGenImageResult
	var result DoubaoGenImageResult
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}
	ret := new(GenStoryImagesResult)
	if len(result.Data) > 0 {
		ret.ImageUrls = make([]string, 0)
		ret.ImageUrls = append(ret.ImageUrls, result.Data[0].URL)
	}
	return ret, nil
}

type DoubaoChatCompletionMessageContent struct {
	Text     string `json:"text"`
	Type     string `json:"type"`
	ImageUrl struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type DoubaoChatCompletionMessage struct {
	Content []DoubaoChatCompletionMessageContent `json:"content"`
	Role    string                               `json:"role"`
}

type DoubaoGenStoryInfoParams struct {
	Model    string                        `json:"model"`
	Messages []DoubaoChatCompletionMessage `json:"messages"`
}

type DoubaoGenStoryInfoResult struct {
	Choices []struct {
		FinishReason string      `json:"finish_reason"`
		Index        int         `json:"index"`
		Logprobs     interface{} `json:"logprobs"`
		Message      struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Role             string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Created     int    `json:"created"`
	ID          string `json:"id"`
	Model       string `json:"model"`
	ServiceTier string `json:"service_tier"`
	Object      string `json:"object"`
	Usage       struct {
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokens        int `json:"prompt_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
}

func (c *DoubaoClient) GenStoryInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	realParams := &DoubaoGenStoryInfoParams{
		Model: "doubao-seed-1-6-flash-250615",
		Messages: []DoubaoChatCompletionMessage{
			{
				Role: "user",
				Content: []DoubaoChatCompletionMessageContent{
					{
						Type: "text",
						Text: params.Content,
					},
				},
			},
		},
	}
	body, err := json.Marshal(realParams)
	if err != nil {
		return nil, err
	}

	url := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.DoubaoAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result DoubaoGenStoryInfoResult
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("doubao return empty choices")
	}

	ret := &StoryInfoResult{
		Content: result.Choices[0].Message.Content,
	}

	return ret, nil
}

func (c *DoubaoClient) GenStoryBoardInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	realParams := &DoubaoGenStoryInfoParams{
		Model: "doubao-seed-1-6-flash-250615",
		Messages: []DoubaoChatCompletionMessage{
			{
				Role: "user",
				Content: []DoubaoChatCompletionMessageContent{
					{
						Type: "text",
						Text: params.Content,
					},
				},
			},
		},
	}
	body, err := json.Marshal(realParams)
	if err != nil {
		return nil, err
	}

	url := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.DoubaoAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result DoubaoGenStoryInfoResult
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("doubao return empty choices")
	}

	ret := &StoryInfoResult{
		Content: result.Choices[0].Message.Content,
	}
	return ret, nil
}

func (c *DoubaoClient) GenStoryRole(ctx context.Context, params *GenStoryCharactorParams) (*GenStoryCharactorResult, error) {
	realParams := &DoubaoGenStoryInfoParams{
		Model: "doubao-seed-1-6-flash-250615",
		Messages: []DoubaoChatCompletionMessage{
			{
				Role: "user",
				Content: []DoubaoChatCompletionMessageContent{
					{
						Type: "text",
						Text: params.Content,
					},
				},
			},
			{
				Role: "system",
				Content: []DoubaoChatCompletionMessageContent{
					{
						Type: "text",
						Text: "你是一个资深作家、小说家，根据输入的角色的简介以及描述，丰富角色的描述",
					},
				},
			},
		},
	}
	body, err := json.Marshal(realParams)
	if err != nil {
		return nil, err
	}

	url := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.DoubaoAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result DoubaoGenStoryInfoResult
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("doubao return empty choices")
	}

	ret := &GenStoryCharactorResult{
		Content: result.Choices[0].Message.Content,
	}
	return ret, nil
}

/*
	{
	    "model": "doubao-seedance-1-0-pro-250528",
	    "content": [
	        {
	            "type": "text",
	            "text": "无人机以极快速度穿越复杂障碍或自然奇观，带来沉浸式飞行体验  --resolution 1080p  --duration 5 --camerafixed false"
	        },
	        {
	            "type": "image_url",
	            "image_url": {
	                "url": "https://ark-project.tos-cn-beijing.volces.com/doc_image/seepro_i2v.png"
	            }
	        }
	    ]
	}
*/
type DoubaoGenStoryboardVideoParams struct {
	Model   string                               `json:"model"`
	Content []DoubaoChatCompletionMessageContent `json:"content"`
}

/*
{"id":"cgt-20250622134237-plkgw"}
*/
type DoubaoGenStoryboardVideoResult struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	VideoUrl string `json:"video_url"`
}

type GenStoryboardVideoParams struct {
	TaskId      string `json:"task_id"`
	Content     string `json:"content"`
	RefImageUrl string `json:"ref_image_url"`
}

type GenStoryboardVideoResult struct {
	TaskId   string `json:"task_id"`
	ID       string `json:"id"`
	Status   string `json:"status"`
	VideoUrl string `json:"video_url"`
}

func (c *DoubaoClient) GenStoryboardVideo(ctx context.Context, params *GenStoryboardVideoParams) (*GenStoryboardVideoResult, error) {
	realParams := &DoubaoGenStoryboardVideoParams{
		Model: "doubao-seedance-1-0-pro-250528",
		Content: []DoubaoChatCompletionMessageContent{
			{
				Type: "text",
				Text: params.Content,
			},
			{
				Type: "image_url",
				ImageUrl: struct {
					URL string `json:"url"`
				}{URL: params.RefImageUrl},
			},
		},
	}
	body, err := json.Marshal(realParams)
	if err != nil {
		return nil, err
	}

	url := "https://ark.cn-beijing.volces.com/api/v3/images/generations"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.DoubaoAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result DoubaoGenStoryboardVideoResult
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return &GenStoryboardVideoResult{
		ID:       result.ID,
		Status:   result.Status,
		VideoUrl: result.VideoUrl,
	}, nil
}

/*
	curl -X GET https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks/cgt-20250622134237-plkgw \
	  -H "Content-Type: application/json" \
	  -H "Authorization: Bearer doubao"
*/

/*
	{
	    "id": "cgt-20250622134237-plkgw",
	    "model": "doubao-seedance-1-0-pro-250528",
	    "status": "succeeded",
	    "content": {
	        "video_url": "https://"
	    },
	    "usage": {
	        "completion_tokens": 246840,
	        "total_tokens": 246840
	    },
	    "created_at": 1750570958,
	    "updated_at": 1750571005
	}
*/
type DoubaoQueryStoryboardVideoTaskStatusResult struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Content struct {
		VideoUrl string `json:"video_url"`
	} `json:"content"`
	Usage struct {
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *DoubaoClient) QueryStoryboardVideoTaskStatus(ctx context.Context, taskId string) (*GenStoryboardVideoResult, error) {
	url := fmt.Sprintf("https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks/%s", taskId)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.DoubaoAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result DoubaoQueryStoryboardVideoTaskStatusResult
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return &GenStoryboardVideoResult{
		TaskId:   taskId,
		ID:       result.ID,
		Status:   result.Status,
		VideoUrl: result.Content.VideoUrl,
	}, nil
}
