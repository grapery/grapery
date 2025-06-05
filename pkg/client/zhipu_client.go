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

type ZhipuStoryClient struct {
	ZhipuClient *zhipu.ZhipuAPI
}

func NewStoryClient(platform int) *ZhipuStoryClient {
	return &ZhipuStoryClient{
		ZhipuClient: zhipu.NewZhipuAPI(),
	}
}

type StoryInfoParams struct {
	RoleDesc string `json:"role_desc"`
	Content  string `json:"content"`
}

type StoryInfoResult struct {
	Content string `json:"content"`
}

func (c *ZhipuStoryClient) GenStoryInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	ret := &StoryInfoResult{}
	chatService := c.ZhipuClient.ChatCompletion("glm-4-flash")
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

func (c *ZhipuStoryClient) GenStoryBoardInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	ret := &StoryInfoResult{}
	chatService := c.ZhipuClient.ChatCompletion("glm-4-flash")
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

type GenStoryPosterParams struct {
	Title        string  `json:"title"`          // 海报标题
	SubTitle     string  `json:"sub_title"`      // 海报副标题
	BodyText     string  `json:"body_text"`      // 海报正文内容
	PromptTextZh string  `json:"prompt_text_zh"` // 提示词
	WhRatios     string  `json:"wh_ratios"`      // 图像宽高比
	LoraName     string  `json:"lora_name"`      // Lora名称
	LoraWeight   float64 `json:"lora_weight"`    // Lora权重
	CtrlRatio    float64 `json:"ctrl_ratio"`     // 控制比率
	CtrlStep     float64 `json:"ctrl_step"`      // 控制步长
	GenerateMode string  `json:"generate_mode"`  // 生成模式
	GenerateNum  int     `json:"generate_num"`   // 生成数量
	UserId       string  `json:"user_id"`        // 用户ID
	RequestId    string  `json:"request_id"`     // 请求ID
}

type GenStoryImagesParams struct {
	Content        string `json:"content"`
	RefImage       string `json:"ref_image"`
	Size           int    `json:"size"`
	UserId         string `json:"user_id"`
	RequestId      string `json:"request_id"`
	Prompt         string `json:"prompt"`          // 额外的提示词
	NegativePrompt string `json:"negative_prompt"` // 负面提示词
	MaskImageUrl   string `json:"mask_image_url"`  // 掩码图像
}

type GenStoryImagesResult struct {
	ImageUrls []string `json:"image_urls"`
}

func (c *ZhipuStoryClient) GenStoryBoardImages(ctx context.Context, params *GenStoryImagesParams) (*GenStoryImagesResult, error) {
	service := c.ZhipuClient.ImageGeneration("CogView-3-Plus").
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

func (c *ZhipuStoryClient) ScaleStoryImages(ctx context.Context, params *ScaleStoryImagesParams) (*ScaleStoryImagesResult, error) {
	return nil, nil
}

type GenStoryCharactorParams struct {
	Content string `json:"content"`
}

type GenStoryCharactorResult struct {
	Content string `json:"content"`
}

func (c *ZhipuStoryClient) GenStoryPeopleCharactor(ctx context.Context, params *GenStoryCharactorParams) (*GenStoryCharactorResult, error) {
	ret := &GenStoryCharactorResult{}
	chatService := c.ZhipuClient.ChatCompletion("glm-4-assistant")
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

func (c *ZhipuStoryClient) ChatWithRole(ctx context.Context, params *ChatWithRoleParams) (*ChatWithRoleResult, error) {
	ret := &ChatWithRoleResult{}
	chatService := c.ZhipuClient.ChatCompletion("charglm-4")
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

func (c *ZhipuStoryClient) GenStoryRoleInfo(ctx context.Context, params *GenStoryRoleInfoParams) (*GenStoryRoleInfoResult, error) {
	ret := &GenStoryRoleInfoResult{}
	chatService := c.ZhipuClient.ChatCompletion("glm-4-flash")
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
