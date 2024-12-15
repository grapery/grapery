package item

import (
	"context"

	api "github.com/grapery/common-protoc/gen"
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

type ItemService struct {
	IsReady bool
}

func (it *ItemService) GetProjectItems(ctx context.Context, req *api.GetProjectItemsRequest) (resp *api.GetProjectItemsResponse, err error) {
	list, err := models.GetStoryItemByProject(ctx,
		int64(req.GetProjectId()),
		int(req.GetOffset()),
		int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	if len(list) != 0 {
		return &api.GetProjectItemsResponse{
			Code: 0,
			Msg:  "success",
			Data: &api.GetProjectItemsResponse_Data{
				GroupId:   req.GetGroupId(),
				ProjectId: req.GetProjectId(),
				UserId:    req.GetUserId(),
				List:      nil,
				PageSize:  0,
				Offset:    req.GetOffset(),
			},
		}, nil
	}
	result := make([]*api.ItemInfo, 0, len(list))
	for idx := range list {
		item := new(api.ItemInfo)
		item.UserId = int64(list[idx].UserID)
		item.Content = nil
		item.ProjectId = int64(list[idx].ProjectID)
		item.Itype = list[idx].ItemType
		item.Title = list[idx].Description
		result = append(result, item)
	}
	return &api.GetProjectItemsResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.GetProjectItemsResponse_Data{
			GroupId:   req.GetGroupId(),
			ProjectId: req.GetProjectId(),
			UserId:    req.GetUserId(),
			List:      result,
			PageSize:  int64(len(result)),
			Offset:    req.GetOffset() + int64(len(result)),
		},
	}, nil
}

func (it *ItemService) GetGroupItems(ctx context.Context, req *api.GetGroupItemsRequest) (resp *api.GetGroupItemsResponse, err error) {
	list, err := models.GetStoryItemByGroup(ctx,
		int64(req.GetGroupId()),
		int(req.GetOffset()),
		int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	if len(list) != 0 {
		return &api.GetGroupItemsResponse{
			Code: 0,
			Msg:  "success",
			Data: &api.GetGroupItemsResponse_Data{
				GroupId:  req.GetGroupId(),
				UserId:   req.GetUserId(),
				List:     nil,
				PageSize: 0,
				Offset:   req.GetOffset(),
			},
		}, nil
	}
	result := make([]*api.ItemInfo, 0, len(list))
	for idx := range list {
		item := new(api.ItemInfo)
		item.UserId = int64(list[idx].UserID)
		item.Content = nil
		item.ProjectId = list[idx].ProjectID
		item.Itype = list[idx].ItemType
		item.Title = list[idx].Description
		result = append(result, item)
	}
	return &api.GetGroupItemsResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.GetGroupItemsResponse_Data{
			GroupId:  req.GetGroupId(),
			UserId:   req.GetUserId(),
			List:     result,
			PageSize: int64(len(result)),
			Offset:   req.GetOffset() + int64(len(result)),
		},
	}, nil
}

func (it *ItemService) GetUserItems(ctx context.Context, req *api.GetUserItemsRequest) (resp *api.GetUserItemsResponse, err error) {
	list, err := models.GetStoryItemByProject(ctx,
		int64(req.GetUserId()),
		int(req.GetOffset()),
		int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &api.GetUserItemsResponse{
			Code: 0,
			Msg:  "success",
			Data: &api.GetUserItemsResponse_Data{
				UserId:   req.GetUserId(),
				List:     nil,
				PageSize: 0,
				Offset:   req.GetOffset(),
			},
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
		Code: 0,
		Msg:  "success",
		Data: &api.GetUserItemsResponse_Data{
			UserId:   req.GetUserId(),
			List:     result,
			PageSize: int64(len(result)),
			Offset:   req.GetOffset() + int64(len(result)),
		},
	}, nil
}

func (it *ItemService) GetItem(ctx context.Context, req *api.GetItemRequest) (resp *api.GetItemResponse, err error) {
	item, err := models.GetStoryItem(ctx, req.GetItemId())
	if err != nil {
		return nil, err
	}
	return &api.GetItemResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.GetItemResponse_Data{
			Info: convert.ConvertItemToInfo(item),
		},
	}, nil
}

func (it *ItemService) UpdateItem(ctx context.Context, req *api.UpdateItemRequest) (resp *api.UpdateItemResponse, err error) {
	item := &models.StoryItem{
		ProjectID:   req.GetProjectId(),
		UserID:      req.GetUserId(),
		Title:       req.GetInfo().Title,
		Description: req.GetInfo().Title,
		ItemType:    api.ItemType_ShortWord,
	}
	err = models.UpdateStoryItemVisable(ctx, int64(req.GetItemId()), api.ScopeType_AllPublic)
	if err != nil {
		return nil, err
	}
	return &api.UpdateItemResponse{
		Code:    0,
		Message: "success",
		Data: &api.UpdateItemResponse_Data{
			Info: convert.ConvertItemToInfo(item),
		},
	}, nil
}

func (it *ItemService) CreateItem(ctx context.Context, req *api.CreateItemRequest) (resp *api.CreateItemResponse, err error) {
	item := &models.StoryItem{
		ProjectID: req.GetProjectId(),
		UserID:    req.GetUserId(),
		Title:     req.GetName(),
		ItemType:  api.ItemType_ShortWord,
	}
	err = models.CreateStoryItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return &api.CreateItemResponse{
		Code:    0,
		Message: "success",
		Data: &api.CreateItemResponse_Data{
			Info: convert.ConvertItemToInfo(item),
		},
	}, nil
}
func (it *ItemService) DeleteItem(ctx context.Context, req *api.DeleteItemRequest) (resp *api.DeleteItemResponse, err error) {
	err = models.DeleteStoryItem(ctx, int64(req.GetItemId()))
	if err != nil {
		return nil, err
	}
	return &api.DeleteItemResponse{}, nil
}

func (it *ItemService) LikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error) {
	err = models.CreateItemLiker(ctx, req.GetProjectId(), req.GetItemId(), req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &api.LikeItemResponse{}, nil
}

func (it *ItemService) UnLikeItem(ctx context.Context, req *api.LikeItemRequest) (resp *api.LikeItemResponse, err error) {
	err = models.DeleteItemLiker(ctx, req.GetProjectId(), req.GetItemId(), req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &api.LikeItemResponse{}, nil
}
