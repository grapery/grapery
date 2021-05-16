package teams

import (
	"github.com/grapery/grapery/api"
	_ "github.com/grapery/grapery/pkg/project"
	"github.com/grapery/grapery/pkg/user"
	"github.com/grapery/grapery/utils"
)

func SearchTeamInGroup(ctx *utils.Context) {
	req := &api.SearchUserRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().SearchUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func CreateTeamInGroup(ctx *utils.Context) {
	req := &api.SearchUserRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().SearchUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func JoinTeamInGroup(ctx *utils.Context) {
	req := &api.SearchUserRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().SearchUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func LeaveTeamInGroup(ctx *utils.Context) {
	req := &api.SearchUserRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().SearchUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func DeleteUserFromTeamInGroup(ctx *utils.Context) {
	req := &api.SearchUserRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().SearchUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func SwitchTeamOwnerInGroup(ctx *utils.Context) {
	req := &api.SearchUserRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().SearchUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}
