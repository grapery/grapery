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

func GetFollowingUser(ctx *utils.Context) {
	req := &api.UserFollowingRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().UserFollowing(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetFollowerUser(ctx *utils.Context) {
	req := &api.UserFollowerRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().UserFollower(ctx.Ctx, req)
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

func UpdateUser(ctx *utils.Context) {
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

func DeleteUser(ctx *utils.Context) {
	ctx.Err = nil
	ctx.Resp = nil
	return
}

func FollowUser(ctx *utils.Context) {
	req := &api.StartFollowUserRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().StartFollowUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func UnFollowUser(ctx *utils.Context) {
	req := &api.StopFollowUserRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := user.GetUserServer().StopFollowUser(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}
