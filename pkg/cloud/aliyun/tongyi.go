package aliyun

import (
	"context"
	"fmt"
	"io"

	"github.com/tongyi-xingchen/xingchen-sdk-go/xingchen"
)

const (
	TongyiAPiKey  = "lm-OZTeQ88SZTUUIqlbVXbJOQ=="
	TongyiAPiKey2 = "lm-MEh+Q5jsk/S6307TfdPyyw=="
)

type XingchenClient struct {
	apiClient *xingchen.APIClient
	brearer   string
}

func NewXingchenClient(accessKeyID, accessKeySecret string) *XingchenClient {
	newCli := &XingchenClient{}
	configuration := xingchen.NewConfiguration()
	newCli.apiClient = xingchen.NewAPIClient(configuration)
	newCli.brearer = TongyiAPiKey

	return newCli
}

type BotProfile struct {
	Name        string `json:"name,omitempty"`
	BotType     string `json:"bot_type,omitempty"`
	Content     string `json:"content,omitempty"`
	CharacterId string `json:"character_id,omitempty"`
}

type ModelParameters struct {
	Temperature       float64 `json:"temperature,omitempty"`
	TopP              float64 `json:"top_p,omitempty"`
	Seed              int32   `json:"seed,omitempty"`
	IncrementalOutput bool    `json:"incremental_output,omitempty"`
}

type ChatReqParams struct {
	Scenario        string                    `json:"scenario,omitempty"`
	BotProfile      BotProfile                `json:"bot_profile,omitempty"`
	ModelParameters ModelParameters           `json:"model_parameters,omitempty"`
	UserUniqID      string                    `json:"user_uniq_id,omitempty"`
	ChatSamples     []xingchen.ChatSampleItem `json:"chat_samples,omitempty"`
	Messages        []xingchen.Message        `json:"messages,omitempty"`
	Context         *xingchen.ChatContext     `json:"context,omitempty"`
	IsSync          bool                      `json:"is_sync,omitempty"`
	IsSpec          bool                      `json:"is_spec,omitempty"` // 自定义历史
}

type AssistenAsyncResult struct {
	ChatResultDTO chan *xingchen.ChatResultDTO
}

func (x *XingchenClient) AssistenAsync(ctx context.Context, params *ChatReqParams) (*AssistenAsyncResult, error) {
	aCtx := context.WithValue(ctx, xingchen.ContextAccessToken, x.brearer)
	chatReqParam := xingchen.ChatReqParams{
		Scenario: &xingchen.Scenario{
			SafetyPrompt: xingchen.PtrString(params.Scenario),
		},
		BotProfile: xingchen.BotProfile{
			Name:    xingchen.PtrString(params.BotProfile.Name),
			BotType: xingchen.PtrString("assistant"),
			Content: xingchen.PtrString(params.BotProfile.Content),
		},
		ModelParameters: &xingchen.ModelParameters{
			Seed:              xingchen.PtrInt64(*xingchen.NewModelParameters().Seed),
			TopP:              xingchen.PtrFloat64(params.ModelParameters.TopP),
			Temperature:       xingchen.PtrFloat64(params.ModelParameters.Temperature),
			IncrementalOutput: xingchen.PtrBool(params.ModelParameters.IncrementalOutput),
		},
		UserProfile: xingchen.UserProfile{
			UserId: params.UserUniqID,
		},
		ChatSamples: params.ChatSamples,
		Messages:    params.Messages,
	}
	chatReqParam.Streaming = xingchen.PtrBool(true)
	var result = new(AssistenAsyncResult)
	result.ChatResultDTO = make(chan *xingchen.ChatResultDTO, 10)

	go func() {
		chatResultStream, err := x.apiClient.ChatApiSub.Chat(aCtx).ChatReqParams(chatReqParam).StreamExecute()
		if err != nil {
			return
		}
		defer chatResultStream.Close()
		defer close(result.ChatResultDTO)
		for {
			select {
			case <-ctx.Done():
				break
			default:
			}
			resp, err := chatResultStream.Recv()
			if err == io.EOF {
				break
			}
			fmt.Println(*resp.Data.Choices[0].Messages[0].Content)
			result.ChatResultDTO <- resp.Data
		}
	}()
	return result, nil
}

type AssistentSyncResult struct {
	ChatResultDTO *xingchen.ChatResultDTO
}

func (x *XingchenClient) AssistentAsync(ctx context.Context, params *ChatReqParams) (*AssistentSyncResult, error) {
	aCtx := context.WithValue(ctx, xingchen.ContextAccessToken, x.brearer)
	chatReqParam := xingchen.ChatReqParams{
		Scenario: &xingchen.Scenario{
			SafetyPrompt: xingchen.PtrString(params.Scenario),
		},
		BotProfile: xingchen.BotProfile{
			Name:    xingchen.PtrString(params.BotProfile.Name),
			BotType: xingchen.PtrString("assistant"),
			Content: xingchen.PtrString(params.BotProfile.Content),
		},
		ModelParameters: &xingchen.ModelParameters{
			Seed:              xingchen.PtrInt64(*xingchen.NewModelParameters().Seed),
			TopP:              xingchen.PtrFloat64(params.ModelParameters.TopP),
			Temperature:       xingchen.PtrFloat64(params.ModelParameters.Temperature),
			IncrementalOutput: xingchen.PtrBool(params.ModelParameters.IncrementalOutput),
		},
		UserProfile: xingchen.UserProfile{
			UserId: params.UserUniqID,
		},
		ChatSamples: params.ChatSamples,
		Messages:    params.Messages,
	}
	var result = new(AssistentSyncResult)
	chatReqParam.Streaming = xingchen.PtrBool(false)
	chatResp, httpRes, err := x.apiClient.ChatApiSub.Chat(aCtx).ChatReqParams(chatReqParam).Execute()
	if err != nil || httpRes == nil {
		return nil, err
	}
	fmt.Println(*chatResp.Data.Choices[0].Messages[0].Content)
	result.ChatResultDTO = chatResp.Data
	return result, nil
}

type NotConstRoleResult struct {
	ChatResultDTO chan *xingchen.ChatResultDTO
}

func (x *XingchenClient) NotConstRoleAsync(ctx context.Context, params *ChatReqParams) (*NotConstRoleResult, error) {
	aCtx := context.WithValue(ctx, xingchen.ContextAccessToken, x.brearer)
	chatReqParam := xingchen.ChatReqParams{
		BotProfile: xingchen.BotProfile{
			Name:    xingchen.PtrString(params.BotProfile.Name),
			Content: xingchen.PtrString(params.BotProfile.Content),
		},
		ModelParameters: &xingchen.ModelParameters{
			Seed:              xingchen.PtrInt64(*xingchen.NewModelParameters().Seed),
			TopP:              xingchen.PtrFloat64(params.ModelParameters.TopP),
			Temperature:       xingchen.PtrFloat64(params.ModelParameters.Temperature),
			IncrementalOutput: xingchen.PtrBool(params.ModelParameters.IncrementalOutput),
		},
		UserProfile: xingchen.UserProfile{
			UserId: params.UserUniqID,
		},
		ChatSamples: params.ChatSamples,
		Messages:    params.Messages,
	}
	chatReqParam.Streaming = xingchen.PtrBool(true)
	var result = new(NotConstRoleResult)
	result.ChatResultDTO = make(chan *xingchen.ChatResultDTO, 10)

	go func() {
		chatResultStream, err := x.apiClient.ChatApiSub.Chat(aCtx).ChatReqParams(chatReqParam).StreamExecute()
		if err != nil {
			return
		}
		defer chatResultStream.Close()
		defer close(result.ChatResultDTO)
		for {
			select {
			case <-ctx.Done():
				break
			default:
			}
			resp, err := chatResultStream.Recv()
			if err == io.EOF {
				break
			}
			fmt.Println(*resp.Data.Choices[0].Messages[0].Content)
			result.ChatResultDTO <- resp.Data
		}
	}()
	return result, nil
}

type NotConstRoleSyncResult struct {
	ChatResultDTO *xingchen.ChatResultDTO
}

func (x *XingchenClient) NotConstRoleSync(ctx context.Context, params *ChatReqParams) (*NotConstRoleSyncResult, error) {
	aCtx := context.WithValue(ctx, xingchen.ContextAccessToken, x.brearer)
	chatReqParam := xingchen.ChatReqParams{
		BotProfile: xingchen.BotProfile{
			Name:    xingchen.PtrString(params.BotProfile.Name),
			Content: xingchen.PtrString(params.BotProfile.Content),
		},
		ModelParameters: &xingchen.ModelParameters{
			Seed:              xingchen.PtrInt64(*xingchen.NewModelParameters().Seed),
			TopP:              xingchen.PtrFloat64(params.ModelParameters.TopP),
			Temperature:       xingchen.PtrFloat64(params.ModelParameters.Temperature),
			IncrementalOutput: xingchen.PtrBool(params.ModelParameters.IncrementalOutput),
		},
		UserProfile: xingchen.UserProfile{
			UserId: params.UserUniqID,
		},
		ChatSamples: params.ChatSamples,
		Messages:    params.Messages,
	}
	var result = new(NotConstRoleSyncResult)
	chatReqParam.Streaming = xingchen.PtrBool(false)
	chatResp, httpRes, err := x.apiClient.ChatApiSub.Chat(aCtx).ChatReqParams(chatReqParam).Execute()
	if err != nil || httpRes == nil {
		return nil, err
	}
	fmt.Println(*chatResp.Data.Choices[0].Messages[0].Content)
	result.ChatResultDTO = chatResp.Data
	return result, nil
}

type ConstRoleWithPlatformHistoryAsyncResult struct {
	ChatResultDTO chan *xingchen.ChatResultDTO
}

func (x *XingchenClient) ConstRoleWithChatHistoryAsync(ctx context.Context, params *ChatReqParams) (*ConstRoleWithPlatformHistoryAsyncResult, error) {
	aCtx := context.WithValue(ctx, xingchen.ContextAccessToken, x.brearer)
	chatReqParam := xingchen.ChatReqParams{
		BotProfile: xingchen.BotProfile{
			Name:        xingchen.PtrString(params.BotProfile.Name),
			CharacterId: xingchen.PtrString(params.BotProfile.CharacterId),
		},
		ModelParameters: &xingchen.ModelParameters{
			Seed:              xingchen.PtrInt64(*xingchen.NewModelParameters().Seed),
			IncrementalOutput: xingchen.PtrBool(params.ModelParameters.IncrementalOutput),
		},
		UserProfile: xingchen.UserProfile{
			UserId: params.UserUniqID,
		},
		Messages: params.Messages,
	}
	if params.IsSpec {
		chatReqParam.Context = params.Context
	}
	chatReqParam.Streaming = xingchen.PtrBool(true)
	var result = new(ConstRoleWithPlatformHistoryAsyncResult)
	result.ChatResultDTO = make(chan *xingchen.ChatResultDTO, 10)

	go func() {
		chatResultStream, err := x.apiClient.ChatApiSub.Chat(aCtx).ChatReqParams(chatReqParam).StreamExecute()
		if err != nil {
			return
		}
		defer chatResultStream.Close()
		defer close(result.ChatResultDTO)
		for {
			select {
			case <-ctx.Done():
				break
			default:
			}
			resp, err := chatResultStream.Recv()
			if err == io.EOF {
				break
			}
			fmt.Println(*resp.Data.Choices[0].Messages[0].Content)
			result.ChatResultDTO <- resp.Data
		}
	}()
	return result, nil
}

type ConstRoleWithPlatformHistorySyncResult struct {
	ChatResultDTO *xingchen.ChatResultDTO
}

func (x *XingchenClient) ConstRoleWithChatHistorySync(ctx context.Context, params *ChatReqParams) (*ConstRoleWithPlatformHistorySyncResult, error) {
	aCtx := context.WithValue(ctx, xingchen.ContextAccessToken, x.brearer)
	chatReqParam := xingchen.ChatReqParams{
		BotProfile: xingchen.BotProfile{
			Name:        xingchen.PtrString(params.BotProfile.Name),
			CharacterId: xingchen.PtrString(params.BotProfile.CharacterId),
		},
		ModelParameters: &xingchen.ModelParameters{
			Seed:              xingchen.PtrInt64(*xingchen.NewModelParameters().Seed),
			IncrementalOutput: xingchen.PtrBool(params.ModelParameters.IncrementalOutput),
		},
		UserProfile: xingchen.UserProfile{
			UserId: params.UserUniqID,
		},
		Messages: params.Messages,
	}
	if params.IsSpec {
		chatReqParam.Context = params.Context
	}
	var result = new(ConstRoleWithPlatformHistorySyncResult)
	chatReqParam.Streaming = xingchen.PtrBool(false)
	chatResp, httpRes, err := x.apiClient.ChatApiSub.Chat(aCtx).ChatReqParams(chatReqParam).Execute()
	if err != nil || httpRes == nil {
		return nil, err
	}
	fmt.Println(*chatResp.Data.Choices[0].Messages[0].Content)
	result.ChatResultDTO = chatResp.Data
	return result, nil
}

type RoleParams struct {
	CharacterId  string
	Name         string
	Type         string
	AvatorName   string
	AvatorUrl    string
	Introduction string
	Traits       string
	OpeningLine  string
	BasicInfo    string
	ChatExample  string
}

func (x *XingchenClient) CreateConstRole(ctx context.Context, params *RoleParams) (interface{}, error) {
	characterCreateDTO := xingchen.CharacterCreateDTO{
		Name: params.Name,
		Type: xingchen.PtrString(params.Type),
		Avatar: &xingchen.FileInfoVO{
			FileUrl:  xingchen.PtrString(params.AvatorUrl),
			Filename: xingchen.PtrString(params.AvatorName),
		},
		Introduction:     params.Introduction,
		Traits:           xingchen.PtrString(params.Traits),
		OpeningLine:      params.OpeningLine,
		BasicInformation: params.BasicInfo,
		ChatExample:      xingchen.PtrString(params.ChatExample),
		PermConfig: &xingchen.CharacterPermissionConfig{
			AllowChat: 1,
			AllowApi:  1,
			IsPublic:  0,
		},
	}
	resp, httpRes, err := x.apiClient.CharacterApiSub.Create(ctx).CharacterCreateDTO(characterCreateDTO).Execute()

	if err != nil || httpRes == nil {
		return nil, err
	}
	return resp.Data.CharacterId, nil
}

func (x *XingchenClient) UpdateConstRole(ctx context.Context, params *RoleParams) (bool, error) {
	characterCreateDTO := xingchen.CharacterUpdateDTO{
		CharacterId: params.CharacterId,
		Name:        params.Name,
		Type:        xingchen.PtrString(params.Type),
		Avatar: &xingchen.FileInfoVO{
			FileUrl:  xingchen.PtrString(params.AvatorUrl),
			Filename: xingchen.PtrString(params.AvatorName),
		},
		Introduction:     params.Introduction,
		Traits:           xingchen.PtrString(params.Traits),
		OpeningLine:      params.OpeningLine,
		BasicInformation: params.BasicInfo,
		ChatExample:      xingchen.PtrString(params.ChatExample),
	}
	resp, httpRes, err := x.apiClient.CharacterApiSub.Update(ctx).CharacterUpdateDTO(characterCreateDTO).Execute()

	if err != nil || httpRes == nil {
		return false, err
	}
	if !resp.GetSuccess() {
		return false, fmt.Errorf(resp.GetErrorMessage())
	}
	return resp.GetData(), nil
}
