package group

import (
	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/pkg/project"
	"github.com/grapery/grapery/utils"
)

func SearchProject(ctx *utils.Context) {
	req := &api.SearchProjectRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().SearchProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func StarProject(ctx *utils.Context) {
	req := &api.StarProjectRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().StarProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func UnStarProject(ctx *utils.Context) {
	req := &api.UnStarProjectRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().UnStarProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func WatchProject(ctx *utils.Context) {
	req := &api.WatchProjectReqeust{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().WatchProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func UnWatchProject(ctx *utils.Context) {
	req := &api.UnWatchProjectReqeust{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().UnWatchProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetProject(ctx *utils.Context) {
	req := &api.GetProjectRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().GetProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func CreateProject(ctx *utils.Context) {
	req := &api.CreateProjectRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().CreateProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func UpdateProject(ctx *utils.Context) {
	req := &api.UpdateProjectRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().UpdateProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func DeleteProject(ctx *utils.Context) {
	req := &api.DeleteProjectRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().DeleteProject(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func GetProjectProfile(ctx *utils.Context) {
	req := &api.GetProjectProfileRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().GetProjectProfile(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}

func UpdateProjectProfile(ctx *utils.Context) {
	req := &api.UpdateProjectProfileRequest{}
	err := ctx.GinC.ShouldBindJSON(req)
	if err != nil {
		ctx.Err = err
		return
	}
	info, err := project.GetProjectServer().UpdateProjectProfile(ctx.Ctx, req)
	if err != nil {
		ctx.Err = err
		return
	}
	ctx.Err = nil
	ctx.Resp = info
	return
}
