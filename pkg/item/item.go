package item

import (
	"context"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/convert"
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
	GetProjectItems(ctx context.Context, req *api.GetProjectItemsRequest) (resp *api.GetProjectItemsResponse, err error)
	GetGroupItems(ctx context.Context, req *api.GetGroupItemsRequest) (resp *api.GetGroupItemsResponse, err error)
	GetUserItems(ctx context.Context, req *api.GetUserItemsRequest) (resp *api.GetUserItemsResponse, err error)
	GetItem(ctx context.Context, req *api.GetItemRequest) (resp *api.GetItemResponse, err error)
	UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (resp *api.UpdateItemResponse, err error)
	CreateItem(ctx context.Context, req *api.CreateItemRequest) (resp *api.CreateItemResponse, err error)
	DeleteItem(ctx context.Context, req *api.DeleteItemRequest) (resp *api.DeleteItemResponse, err error)
	LikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error)
	UnLikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error)
}

type ItemService struct{}

func (it *ItemService) GetProjectItems(ctx context.Context, req *api.GetProjectItemsRequest) (resp *api.GetProjectItemsResponse, err error) {
	repo := models.NewRepository(ctx)
	list, err := models.GetItemByProject(repo,
		req.GetProjectId(),
		int(req.GetOffset()),
		int(req.GetNumber()))
	if err != nil {
		return nil, err
	}
	if len(list) != 0 {
		return &api.GetProjectItemsResponse{
			GroupId:   req.GetGroupId(),
			ProjectId: req.GetProjectId(),
			UserId:    req.GetUserId(),
			List:      nil,
			Number:    0,
			Offset:    req.GetOffset(),
		}, nil
	}
	result := make([]*api.ItemInfo, 0, len(list))
	for idx := range list {
		item := new(api.ItemInfo)
		item.UserId = list[idx].UserID
		item.Content = nil
		item.ProjectId = list[idx].ProjectID
		item.Itype = list[idx].ItemType
		item.Title = list[idx].Description
		result = append(result, item)
	}
	return &api.GetProjectItemsResponse{
		GroupId:   req.GetGroupId(),
		ProjectId: req.GetProjectId(),
		UserId:    req.GetUserId(),
		List:      result,
		Number:    uint64(len(result)),
		Offset:    req.GetOffset() + uint64(len(result)),
	}, nil
}

func (it *ItemService) GetGroupItems(ctx context.Context, req *api.GetGroupItemsRequest) (resp *api.GetGroupItemsResponse, err error) {
	repo := models.NewRepository(ctx)
	list, err := models.GetItemByGroup(repo, req.GetGroupId(), int(req.GetOffset()), int(req.GetNumber()))
	if err != nil {
		return nil, err
	}
	if len(list) != 0 {
		return &api.GetGroupItemsResponse{
			GroupId: req.GetGroupId(),
			UserId:  req.GetUserId(),
			List:    nil,
			Number:  0,
			Offset:  req.GetOffset(),
		}, nil
	}
	result := make([]*api.ItemInfo, 0, len(list))
	for idx := range list {
		item := new(api.ItemInfo)
		item.UserId = list[idx].UserID
		item.Content = nil
		item.ProjectId = list[idx].ProjectID
		item.Itype = list[idx].ItemType
		item.Title = list[idx].Description
		result = append(result, item)
	}
	return &api.GetGroupItemsResponse{
		GroupId: req.GetGroupId(),
		UserId:  req.GetUserId(),
		List:    result,
		Number:  uint64(len(result)),
		Offset:  req.GetOffset() + uint64(len(result)),
	}, nil
}

func (it *ItemService) GetUserItems(ctx context.Context, req *api.GetUserItemsRequest) (resp *api.GetUserItemsResponse, err error) {
	repo := models.NewRepository(ctx)
	list, err := models.GetItemByProject(repo, req.GetUserId(), int(req.GetOffset()), int(req.GetNumber()))
	if err != nil {
		return nil, err
	}
	if len(list) != 0 {
		return &api.GetUserItemsResponse{
			UserId: req.GetUserId(),
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
		item.ProjectId = list[idx].ProjectID
		item.Itype = list[idx].ItemType
		item.Title = list[idx].Description
		result = append(result, item)
	}
	return &api.GetUserItemsResponse{
		UserId: req.GetUserId(),
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
		Info: convert.ConvertItemToInfo(item),
	}, nil
}

func (it *ItemService) UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (resp *api.UpdateItemResponse, err error) {
	repo := models.NewRepository(ctx)
	item := &models.Item{
		ProjectID:   req.GetProjectId(),
		UserID:      req.GetUserId(),
		Title:       req.GetInfo().Title,
		Description: req.GetInfo().Title,
		ItemType:    api.ItemType_ShortWord,
	}
	err = models.UpdateItemVisable(repo, req.GetItemId(), api.VisibleType_Public)
	if err != nil {
		return nil, err
	}
	return &api.UpdateItemResponse{
		Info: convert.ConvertItemToInfo(item),
	}, nil
}
func (it *ItemService) CreateItem(ctx context.Context, req *api.CreateItemRequest) (resp *api.CreateItemResponse, err error) {
	repo := models.NewRepository(ctx)
	item := &models.Item{
		ProjectID: req.GetProjectId(),
		UserID:    req.GetUserId(),
		Title:     req.GetName(),
		ItemType:  api.ItemType_ShortWord,
	}
	err = models.CreateItem(repo, item)
	if err != nil {
		return nil, err
	}
	return &api.CreateItemResponse{
		Info: convert.ConvertItemToInfo(item),
	}, nil
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
	repo := models.NewRepository(ctx)
	err = models.CreateItemLiker(repo, req.GetProjectId(), req.GetItemId(), req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &api.LikeItemResponse{}, nil
}

func (it *ItemService) UnLikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error) {
	repo := models.NewRepository(ctx)
	err = models.DeleteItemLiker(repo, req.GetProjectId(), req.GetItemId(), req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &api.LikeItemResponse{}, nil
}
