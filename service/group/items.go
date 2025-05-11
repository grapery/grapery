package group

import (
	"context"

	connect "github.com/bufbuild/connect-go"

	api "github.com/grapery/common-protoc/gen"
	itemService "github.com/grapery/grapery/pkg/item"
)

type StoryItemService struct {
}

func (ts *StoryItemService) GetUserItems(ctx context.Context, req *connect.Request[api.GetUserItemsRequest]) (*connect.Response[api.GetUserItemsResponse], error) {
	info, err := itemService.GetItemServer().GetUserItems(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetUserItemsResponse]{
		Msg: info,
	}, nil
}
func (ts *StoryItemService) GetItem(ctx context.Context, req *connect.Request[api.GetItemRequest]) (*connect.Response[api.GetItemResponse], error) {
	info, err := itemService.GetItemServer().GetItem(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetItemResponse]{
		Msg: info,
	}, nil
}
func (ts *StoryItemService) CreateItem(ctx context.Context, req *connect.Request[api.CreateItemRequest]) (*connect.Response[api.CreateItemResponse], error) {
	info, err := itemService.GetItemServer().CreateItem(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.CreateItemResponse]{
		Msg: info,
	}, nil
}
func (ts *StoryItemService) UpdateItem(ctx context.Context, req *connect.Request[api.UpdateItemRequest]) (*connect.Response[api.UpdateItemResponse], error) {
	info, err := itemService.GetItemServer().UpdateItem(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UpdateItemResponse]{
		Msg: info,
	}, nil
}
func (ts *StoryItemService) DeleteItem(ctx context.Context, req *connect.Request[api.DeleteItemRequest]) (*connect.Response[api.DeleteItemResponse], error) {
	info, err := itemService.GetItemServer().DeleteItem(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.DeleteItemResponse]{
		Msg: info,
	}, nil
}
func (ts *StoryItemService) LikeItem(ctx context.Context, req *connect.Request[api.LikeItemRequest]) (*connect.Response[api.LikeItemResponse], error) {
	info, err := itemService.GetItemServer().LikeItem(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.LikeItemResponse]{
		Msg: info,
	}, nil
}
