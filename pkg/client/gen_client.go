package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grapery/grapery/pkg/cloud/zhipu"
	zhipuapi "github.com/grapery/grapery/pkg/cloud/zhipu/zhipu"
)

const (
	RateLimiter = 10
)

const (
	ApiCodeOK    = 0
	ApiErrorCode = 1
)

// platform
// azure
// google
// groq
// aliyun
// openai
// metric_order
// zhipu
// keling
// tencent
// step
const (
	PlatformAzure       = 1
	PlatformGoogle      = 2
	PlatformGroq        = 3
	PlatformAliyun      = 4
	PlatformOpenAI      = 5
	PlatformMetricOrder = 6
	PlatformZhipu       = 7
	PlatformKeling      = 8
	PlatformTencent     = 9
	PlatformStep        = 10
)

type StoryClient struct {
	Client *zhipu.ZhipuAPI
}

func NewStoryClient(platform int) *StoryClient {
	return &StoryClient{
		Client: zhipu.NewZhipuAPI(),
	}
}

type StoryInfoParams struct {
	Content string `json:"content"`
}

type StoryInfoResult struct {
	Content string `json:"content"`
}

func (c *StoryClient) GenStoryInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	ret := &StoryInfoResult{}
	chatService := c.Client.ChatCompletion("glm-4-flash")
	completeMessage := ""
	chatService.AddMessage(zhipuapi.ChatCompletionMessage{
		Role:    "user",
		Content: params.Content,
	}).SetStreamHandler(func(chunk zhipuapi.ChatCompletionResponse) error {
		if len(chunk.Choices[0].Delta.Content) > 0 {
			completeMessage = completeMessage + chunk.Choices[0].Delta.Content
		}
		return nil
	})
	res, err := chatService.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("gen story info failed,code: %s, err: %v",
			zhipuapi.GetAPIErrorCode(err), err)
	}
	resData, _ := json.Marshal(res)
	fmt.Println("chat respnse: ", string(resData))
	ret.Content = completeMessage
	return ret, nil
}

func (c *StoryClient) GenStoryBoardInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	ret := &StoryInfoResult{}
	chatService := c.Client.ChatCompletion("glm-4-flash")
	completeMessage := ""
	chatService.AddMessage(zhipuapi.ChatCompletionMessage{
		Role:    "user",
		Content: params.Content,
	}).SetStreamHandler(func(chunk zhipuapi.ChatCompletionResponse) error {
		if len(chunk.Choices[0].Delta.Content) > 0 {
			completeMessage = completeMessage + chunk.Choices[0].Delta.Content
		}
		return nil
	})
	res, err := chatService.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("gen story info failed,code: %s, err: %v",
			zhipuapi.GetAPIErrorCode(err), err)
	}
	resData, _ := json.Marshal(res)
	fmt.Println("chat respnse: ", string(resData))
	ret.Content = completeMessage
	return ret, nil
}

// 支持的图像尺寸包括1024x1024、768x1344、864x1152、1344x768、1152x864、1440x720以及720x1440，默认的图像尺寸为1024x1024。
const (
	Size1024x1024 = "1024x1024"
	Size768x1344  = "768x1344"
	Size864x1152  = "864x1152"
	Size1344x768  = "1344x768"
	Size1152x864  = "1152x864"
	Size1440x720  = "1440x720"
	Size720x1440  = "720x1440"
)

type GenStoryImagesParams struct {
	Content  string `json:"content"`
	RefImage string `json:"ref_image"`
	Size     int    `json:"size"`
	UserId   string `json:"user_id"`
}

type GenStoryImagesResult struct {
	ImageUrls []string `json:"image_urls"`
}

func (c *StoryClient) GenStoryBoardImages(ctx context.Context, params *GenStoryImagesParams) (*GenStoryImagesResult, error) {
	service := c.Client.ImageGeneration("CogView-3-Plus").
		SetPrompt(params.Content).
		SetUserID("grapestree")
	resp, err := service.Do(ctx)
	if err != nil {
		return nil, err
	}
	println(resp.Created)
	result := &GenStoryImagesResult{
		ImageUrls: []string{},
	}
	for _, val := range resp.Data {
		result.ImageUrls = append(result.ImageUrls, val.URL)
	}
	return result, nil
}

type ScaleStoryImagesParams struct {
	ImageUrls []string `json:"image_urls"`
	Size      int      `json:"size"`
}

type ScaleStoryImagesResult struct {
	ImageUrls []string `json:"image_urls"`
	TimeCost  int
}

func (c *StoryClient) ScaleStoryImages(ctx context.Context, params *ScaleStoryImagesParams) (*ScaleStoryImagesResult, error) {
	return nil, nil
}

type GenStoryPeopleCharactorParams struct {
	Content string `json:"content"`
}

type GenStoryPeopleCharactorResult struct {
	Content string `json:"content"`
}

func (c *StoryClient) GenStoryPeopleCharactor(ctx context.Context, params *GenStoryPeopleCharactorParams) (*GenStoryPeopleCharactorResult, error) {
	ret := &GenStoryPeopleCharactorResult{}
	chatService := c.Client.ChatCompletion("glm-4-assistant")
	completeMessage := ""
	chatService.AddMessage(zhipuapi.ChatCompletionMessage{
		Role:    "user",
		Content: params.Content,
	}).SetStreamHandler(func(chunk zhipuapi.ChatCompletionResponse) error {
		if len(chunk.Choices[0].Delta.Content) > 0 {
			completeMessage = completeMessage + chunk.Choices[0].Delta.Content
		}
		return nil
	})
	res, err := chatService.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("gen story info failed,code: %s, err: %v",
			zhipuapi.GetAPIErrorCode(err), err)
	}
	resData, _ := json.Marshal(res)
	fmt.Println("chat respnse: ", string(resData))
	ret.Content = completeMessage
	return ret, nil
}

type ChatWithRoleParams struct {
	Role           string `json:"role"`
	MessageContent string `json:"message_content"`
	Background     string `json:"background"`
	SenseDesc      string `json:"sense_desc"`
	RolePositive   string `json:"role_positive"`
	RoleNegative   string `json:"role_negative"`
	RequestId      string `json:"request_id"`
	UserId         string `json:"user_id"`
}

type ChatWithRoleResult struct {
	Content string `json:"content"`
}

func (c *StoryClient) ChatWithRole(ctx context.Context, params *ChatWithRoleParams) (*ChatWithRoleResult, error) {
	ret := &ChatWithRoleResult{}
	chatService := c.Client.ChatCompletion("charglm-4")
	chatService.AddMessage(zhipuapi.ChatCompletionMessage{
		Role:    "user",
		Content: params.MessageContent,
	}).
		SetUserID(params.UserId).
		SetRequestID(params.RequestId).
		SetModel("charglm-4")
	roleContent := fmt.Sprintf("角色描述：%s", params.Background)
	chatService.AddMessage(zhipuapi.ChatCompletionMessage{
		Role:    "system",
		Content: roleContent,
	})
	res, err := chatService.Do(ctx)
	if err != nil {
		return nil, err
	}
	resData, _ := json.Marshal(res)
	fmt.Println("ChatWithRole chat respnse: ", string(resData))
	ret.Content = res.Choices[0].Message.Content
	fmt.Printf("ChatWithRole chat respnse: %+v\n", res.Choices[0].Message)
	return ret, nil
}

type GenStoryRoleInfoParams struct {
	Content string `json:"content"`
}

type GenStoryRoleInfoResult struct {
	Content string `json:"content"`
}

func (c *StoryClient) GenStoryRoleInfo(ctx context.Context, params *GenStoryRoleInfoParams) (*GenStoryRoleInfoResult, error) {
	ret := &GenStoryRoleInfoResult{}
	chatService := c.Client.ChatCompletion("glm-4-flash")
	completeMessage := ""
	chatService.AddMessage(zhipuapi.ChatCompletionMessage{
		Role:    "user",
		Content: params.Content,
	}).SetStreamHandler(func(chunk zhipuapi.ChatCompletionResponse) error {
		if len(chunk.Choices[0].Delta.Content) > 0 {
			completeMessage = completeMessage + chunk.Choices[0].Delta.Content
		}
		return nil
	})
	res, err := chatService.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("gen story role info failed,code: %s, err: %v",
			zhipuapi.GetAPIErrorCode(err), err)
	}
	resData, _ := json.Marshal(res)
	fmt.Println("chat respnse: ", string(resData))
	ret.Content = completeMessage
	return ret, nil
}
