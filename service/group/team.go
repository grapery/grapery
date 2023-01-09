package group

import (
	// "net/http"
	"context"
	"net/http"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/group"
	"github.com/grapery/grapery/utils"
)

func SearchTeamInGroup(ctx *gin.Context) {
	req := &api.GetGroupReqeust{}
	err := ctx.ShouldBindJSON(req)
	var ret = new(utils.Result)
	if err != nil {
		ret.Message = err.Error()
		ret.Code = http.StatusOK
		ret.Data = nil
		ctx.JSON(http.StatusOK, ret)
		return
	}
	info, err := group.GetGroupServer().GetGroup(context.Background(), req)
	if err != nil {
		ret.Message = err.Error()
		ret.Code = http.StatusOK
		ret.Data = nil
		ctx.JSON(http.StatusOK, ret)
		return
	}
	ret.Message = "ok"
	ret.Code = http.StatusOK
	ret.Data = info
	ctx.JSON(http.StatusOK, ret)
	return
}

func GetGroupTeams(ctx *utils.Context) {
	req := &api.GetGroupReqeust{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := group.GetGroupServer().GetGroup(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func CreateGroupTeam(ctx *utils.Context) {
	req := &api.UpdateGroupInfoRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := group.GetGroupServer().UpdateGroupInfo(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func DeleteGroupTeam(ctx *utils.Context) {
	req := &api.DeleteGroupRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := group.GetGroupServer().DeleteGroup(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetTeamMembers(ctx *utils.Context) {
	req := &api.FetchGroupMembersRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := group.GetGroupServer().FetchGroupMembers(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return

}

func JoinGroupTeam(ctx *utils.Context) {
	req := &api.JoinGroupRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := group.GetGroupServer().JoinGroup(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func LeaveGroupTeam(ctx *utils.Context) {
	req := &api.LeaveGroupRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := group.GetGroupServer().LeaveGroup(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}
