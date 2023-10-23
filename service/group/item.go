package group

import (
	"context"

	api "github.com/grapery/common-protoc/gen"
	itemService "github.com/grapery/grapery/pkg/item"
)

type ItemService struct {
}

func (ts *ItemService) GetUserItems(ctx context.Context, req *api.GetUserItemsRequest) (*api.GetUserItemsResponse, error) {
	info, err := itemService.GetItemServer().GetUserItems(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *ItemService) GetItem(ctx context.Context, req *api.GetItemRequest) (*api.GetItemResponse, error) {
	info, err := itemService.GetItemServer().GetItem(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *ItemService) CreateItem(ctx context.Context, req *api.CreateItemRequest) (*api.CreateItemResponse, error) {
	info, err := itemService.GetItemServer().CreateItem(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *ItemService) UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (*api.UpdateItemResponse, error) {
	info, err := itemService.GetItemServer().UpdateItem(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *ItemService) DeleteItem(ctx context.Context, req *api.DeleteItemRequest) (*api.DeleteItemResponse, error) {
	info, err := itemService.GetItemServer().DeleteItem(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *ItemService) LikeItem(ctx context.Context, req *api.LikeItemRequest) (*api.LikeItemResponse, error) {
	info, err := itemService.GetItemServer().LikeItem(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
