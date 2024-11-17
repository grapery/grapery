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
const (
	PlatformAzure       = 1
	PlatformGoogle      = 2
	PlatformGroq        = 3
	PlatformAliyun      = 4
	PlatformOpenAI      = 5
	PlatformMetricOrder = 6
	PlatformZhipu       = 7
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
