package group

import (
	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/item"
	"github.com/grapery/grapery/utils"
)

func GetProjectItems(ctx *utils.Context) {
	req := &api.GetProjectItemsRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().GetProjectItems(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetGroupItems(ctx *utils.Context) {
	req := &api.GetGroupItemsRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().GetGroupItems(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetUserItems(ctx *utils.Context) {
	req := &api.GetUserItemsRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().GetUserItems(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetProjectItem(ctx *utils.Context) {
	req := &api.GetItemRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().GetItem(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func UpdateProjectItem(ctx *utils.Context) {
	req := &api.UpdateItemRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().UpdateItem(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func CreateProjectItem(ctx *utils.Context) {
	req := &api.CreateItemRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().CreateItem(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return

}

func DeleteProjectItem(ctx *utils.Context) {
	req := &api.DeleteItemRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().DeleteItem(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func LikeItem(ctx *utils.Context) {
	req := &api.LikeItemRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().LikeItem(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func UnLikeItem(ctx *utils.Context) {
	req := &api.LikeItemRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := item.GetItemServer().UnLikeItem(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}
