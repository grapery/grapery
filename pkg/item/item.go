package item

import (
	"context"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/models"
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
	GetItems(ctx context.Context, req *api.GetItemsRequest) (resp *api.GetItemsResponse, err error)
	GetItem(ctx context.Context, req *api.GetItemRequest) (resp *api.GetItemResponse, err error)
	UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (resp *api.UpdateItemResponse, err error)
	CreateItem(ctx context.Context, req *api.CreateItemRequest) (resp *api.CreateItemResponse, err error)
	DeleteItem(ctx context.Context, req *api.DeleteItemRequest) (resp *api.DeleteItemResponse, err error)
	LikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error)
}

type ItemService struct{}

func (it *ItemService) GetItems(ctx context.Context, req *api.GetItemsRequest) (resp *api.GetItemsResponse, err error) {
	repo := models.NewRepository(ctx)
	list, err := models.GetItemByProject(repo, req.GetProjectId(), int(req.GetOffset()), int(req.GetNumber()))
	if err != nil {
		return nil, err
	}
	if len(list) != 0 {
		return &api.GetItemsResponse{
			List:   nil,
			Number: 0,
			Offset: req.GetOffset(),
		}, nil
	}
	result := make([]*api.ItemInfo, 0, len(list))
	for idx := range list {
		item := new(api.ItemInfo)
		item.UserId = list[idx].UserID
		item.Content = list[idx].Title
		result = append(result, item)
	}
	return &api.GetItemsResponse{
		List:   result,
		Number: 0,
		Offset: req.GetOffset(),
	}, nil
}

func (it *ItemService) GetItem(ctx context.Context, req *api.GetItemRequest) (resp *api.GetItemResponse, err error) {
	return nil, nil
}

func (it *ItemService) UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (resp *api.UpdateItemResponse, err error) {
	return nil, nil
}
func (it *ItemService) CreateItem(ctx context.Context, req *api.CreateItemRequest) (resp *api.CreateItemResponse, err error) {
	return nil, nil
}
func (it *ItemService) DeleteItem(ctx context.Context, req *api.DeleteItemRequest) (resp *api.DeleteItemResponse, err error) {
	return nil, nil
}
func (it *ItemService) LikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error) {
	return nil, nil
}
