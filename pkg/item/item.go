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
	UnLikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error)
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
		item.Content = nil
		item.GroupId = list[idx].GroupID
		item.ProjectId = list[idx].ProjectID
		item.Itype = list[idx].ItemType
		item.Title = list[idx].Description
		result = append(result, item)
	}
	return &api.GetItemsResponse{
		List:   result,
		Number: uint64(len(result)),
		Offset: req.GetOffset() + uint64(len(result)),
	}, nil
}

func (it *ItemService) GetItem(ctx context.Context, req *api.GetItemRequest) (resp *api.GetItemResponse, err error) {
	repo := models.NewRepository(ctx)
	item, err := models.GetItem(repo, req.GetItemId())
	if err != nil {
		return nil, err
	}
	return &api.GetItemResponse{
		Info: ConvertItemToInfo(item),
	}, nil
}

func (it *ItemService) UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (resp *api.UpdateItemResponse, err error) {
	return nil, nil
}
func (it *ItemService) CreateItem(ctx context.Context, req *api.CreateItemRequest) (resp *api.CreateItemResponse, err error) {
	return nil, nil
}
func (it *ItemService) DeleteItem(ctx context.Context, req *api.DeleteItemRequest) (resp *api.DeleteItemResponse, err error) {
	repo := models.NewRepository(ctx)
	err = models.DeleteItem(repo, req.GetItemId())
	if err != nil {
		return nil, err
	}
	return &api.DeleteItemResponse{}, nil
}
func (it *ItemService) LikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error) {
	return nil, nil
}

func (it *ItemService) UnLikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error) {
	return nil, nil
}

func ConvertItemToInfo(item *models.Item) *api.ItemInfo {
	info := new(api.ItemInfo)
	info.UserId = item.UserID
	info.Content = nil
	info.GroupId = item.GroupID
	info.ProjectId = item.ProjectID
	info.Itype = item.ItemType
	info.Title = item.Description
	return info
}

func ConvertInfoToItem(info *api.ItemInfo) *models.Item {
	item := new(models.Item)
	item.UserID = info.UserId
	item.Description = info.Title
	item.GroupID = info.GroupId
	item.ProjectID = info.ProjectId
	item.ItemType = info.Itype
	return item
}
