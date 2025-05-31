package client

import (
	"context"

	"github.com/grapery/grapery/pkg/cloud/aliyun"
)

type AliyunClient struct {
	Client *aliyun.AliyunClient
}

func NewAliyunClient() *AliyunClient {
	client, _ := aliyun.NewAliyunClient()
	return &AliyunClient{
		Client: client,
	}
}

func (c *AliyunClient) GenStoryInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	return nil, nil
}

func (c *AliyunClient) GenStoryBoardInfo(ctx context.Context, params *StoryInfoParams) (*StoryInfoResult, error) {
	return nil, nil
}

func (c *AliyunClient) GenStoryBoardImages(ctx context.Context, params *GenStoryImagesParams) (*GenStoryImagesResult, error) {
	return nil, nil
}

func (c *AliyunClient) ScaleStoryImages(ctx context.Context, params *ScaleStoryImagesParams) (*ScaleStoryImagesResult, error) {
	return nil, nil
}

func (c *AliyunClient) GenStoryPeopleCharactor(ctx context.Context, params *GenStoryPeopleCharactorParams) (*GenStoryPeopleCharactorResult, error) {
	return nil, nil
}

func (c *AliyunClient) ChatWithRole(ctx context.Context, params *ChatWithRoleParams) (*ChatWithRoleResult, error) {
	return nil, nil
}
