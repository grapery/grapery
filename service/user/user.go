package user

import (
	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/user"
	"github.com/grapery/grapery/utils"
)

func SearchUser(ctx *utils.Context) {

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

}

func GetWatching(ctx *utils.Context) {

}

func GetFollowingUser(ctx *utils.Context) {

}

func GetFollowerUser(ctx *utils.Context) {

}

func GetFollowingGroup(ctx *utils.Context) {

}

func UpdateUser(ctx *utils.Context) {

}

func DeleteUser(ctx *utils.Context) {

}

func FollowUser(ctx *utils.Context) {

}

func UnFollowUser(ctx *utils.Context) {

}
