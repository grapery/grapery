package item

import (
	"context"

	api "github.com/grapery/common-protoc/gen"
)

var itemServer ItemServer

func init() {
	itemServer = NewItemService()
}

func GetItemServer() ItemServer {
	return itemServer
}

func NewItemService() *ItemService {
	return &ItemService{}
}

type ItemServer interface {
	GetGroupItems(ctx context.Context, req *api.GetGroupItemsRequest) (resp *api.GetGroupItemsResponse, err error)
	GetUserItems(ctx context.Context, req *api.GetUserItemsRequest) (resp *api.GetUserItemsResponse, err error)
	GetItem(ctx context.Context, req *api.GetItemRequest) (resp *api.GetItemResponse, err error)
	UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (resp *api.UpdateItemResponse, err error)
	CreateItem(ctx context.Context, req *api.CreateItemRequest) (resp *api.CreateItemResponse, err error)
	DeleteItem(ctx context.Context, req *api.DeleteItemRequest) (resp *api.DeleteItemResponse, err error)
	LikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error)
	UnLikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error)
}

type ItemService struct {
	IsReady bool
}

func (it *ItemService) GetGroupItems(ctx context.Context, req *api.GetGroupItemsRequest) (resp *api.GetGroupItemsResponse, err error) {
	return &api.GetGroupItemsResponse{
		Code: 0,
		Msg:  "success",
		Data: nil,
	}, nil
}

func (it *ItemService) GetUserItems(ctx context.Context, req *api.GetUserItemsRequest) (resp *api.GetUserItemsResponse, err error) {
	return &api.GetUserItemsResponse{
		Code: 0,
		Msg:  "success",
		Data: nil,
	}, nil
}

func (it *ItemService) GetItem(ctx context.Context, req *api.GetItemRequest) (resp *api.GetItemResponse, err error) {
	return &api.GetItemResponse{
		Code: 0,
		Msg:  "success",
		Data: nil,
	}, nil
}

func (it *ItemService) UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (resp *api.UpdateItemResponse, err error) {
	return &api.UpdateItemResponse{
		Code: 0,
		Data: nil,
	}, nil
}

func (it *ItemService) CreateItem(ctx context.Context, req *api.CreateItemRequest) (resp *api.CreateItemResponse, err error) {
	return &api.CreateItemResponse{
		Code: 0,
		Data: nil,
	}, nil
}
func (it *ItemService) DeleteItem(ctx context.Context, req *api.DeleteItemRequest) (resp *api.DeleteItemResponse, err error) {
	return &api.DeleteItemResponse{
		Code: 0,
		Data: nil,
	}, nil
}

func (it *ItemService) LikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error) {
	return &api.LikeItemResponse{
		Code: 0,
		Data: nil,
	}, nil
}

func (it *ItemService) UnLikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error) {
	return &api.LikeItemResponse{
		Code: 0,
		Data: nil,
	}, nil
}
