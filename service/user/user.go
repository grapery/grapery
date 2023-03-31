package user

import (
	"github.com/grapery/grapery/api"
	_ "github.com/grapery/grapery/pkg/project"
	"github.com/grapery/grapery/pkg/user"
	"github.com/grapery/grapery/utils"
)

func SearchUser(ctx *utils.Context) {
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

func GetUser(ctx *utils.Context) {
	req := &api.UserInfoRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().GetUserInfo(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetUserProfile(ctx *utils.Context) {
	req := &api.UserInfoRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().GetUserInfo(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetUserGroup(ctx *utils.Context) {
	req := &api.UserGroupRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().GetUserGroup(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return

}

func GetUserActive(ctx *utils.Context) {
	req := &api.FetchUserActivesRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().FetchUserActives(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetWatching(ctx *utils.Context) {
	req := &api.UserWatchingRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().UserWatching(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetFollowingGroup(ctx *utils.Context) {
	req := &api.UserFollowingGroupRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().GetUserFollowingGroup(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetUserSetting(ctx *utils.Context) {
	req := &api.UserUpdateRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().UpdateUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func UpdateUserSetting(ctx *utils.Context) {
	req := &api.UserUpdateRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().UpdateUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

// DeleteUser
func DeleteUser(ctx *utils.Context) {
	ctx.Err = nil
	ctx.Resp = struct{}{}
	return
}

func FollowGroup(ctx *utils.Context) {

	return
}

func UnFollowGroup(ctx *utils.Context) {
	return
}
