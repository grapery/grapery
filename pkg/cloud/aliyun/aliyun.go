package aliyun

import "context"

var (
	APiKey    = ""
	SecretKey = ""
)

type AliyunClient struct {
}

func NewAliyunClient() *AliyunClient {
	return &AliyunClient{}
}

type AliChatParams struct {
	Prompt string `json:"prompt"`
	UserId string `json:"userId"`
}

type AliChatResponse struct {
	Response string `json:"response"`
}

func (a *AliyunClient) Chat(ctx context.Context, params *AliChatParams) (*AliChatResponse, error) {
	return nil, nil
}

type GenImagesParams struct {
	Prompt string `json:"prompt"`
	UserId string `json:"userId"`
}
type AliyunGenImageDetail struct {
	Data []byte
	Num  int
	ID   string
}

type GenImagesResponse struct {
	Images []AliyunGenImageDetail
	Prompt string
	Error  error
	ErrMsg string
}

func (a *AliyunClient) GenImages(ctx context.Context, params *GenImagesParams) (*GenImagesResponse, error) {
	return nil, nil
}

type AliyunGenVideoParams struct {
	Prompt string `json:"prompt"`
	UserId string `json:"userId"`
}

type AliyunGenVideoResponse struct {
	Video string `json:"video"`
}

func (a *AliyunClient) GenVideo(ctx context.Context, params *AliyunGenVideoParams) (*AliyunGenVideoResponse, error) {
	return nil, nil
}
