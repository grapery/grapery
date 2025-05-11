package group

import (
	"context"
	"errors"

	connect "github.com/bufbuild/connect-go"

	api "github.com/grapery/common-protoc/gen"
	itemService "github.com/grapery/grapery/pkg/item"
	projectService "github.com/grapery/grapery/pkg/project"
)

type ProjectService struct {
	Filter string
}

func (ps *ProjectService) GetProjectInfo(ctx context.Context, req *connect.Request[api.GetProjectRequest]) (*connect.Response[api.GetProjectResponse], error) {
	info, err := projectService.GetProjectServer().GetProjectInfo(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetProjectResponse]{
		Msg: info,
	}, nil
}

func (ps *ProjectService) GetProjectList(ctx context.Context, req *connect.Request[api.GetProjectListRequest]) (*connect.Response[api.GetProjectListResponse], error) {
	info, err := projectService.GetProjectServer().GetProjectList(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetProjectListResponse]{
		Msg: info,
	}, nil
}

func (ps *ProjectService) CreateProject(ctx context.Context, req *connect.Request[api.CreateProjectRequest]) (*connect.Response[api.CreateProjectResponse], error) {
	info, err := projectService.GetProjectServer().CreateProject(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.CreateProjectResponse]{
		Msg: info,
	}, nil
}
func (ps *ProjectService) UpdateProject(ctx context.Context, req *connect.Request[api.UpdateProjectRequest]) (*connect.Response[api.UpdateProjectResponse], error) {
	info, err := projectService.GetProjectServer().UpdateProject(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UpdateProjectResponse]{
		Msg: info,
	}, nil
}
func (ps *ProjectService) DeleteProject(ctx context.Context, req *connect.Request[api.DeleteProjectRequest]) (*connect.Response[api.DeleteProjectResponse], error) {
	info, err := projectService.GetProjectServer().DeleteProject(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.DeleteProjectResponse]{
		Msg: info,
	}, nil
}
func (ps *ProjectService) GetProjectProfile(ctx context.Context, req *connect.Request[api.GetProjectProfileRequest]) (*connect.Response[api.GetProjectProfileResponse], error) {
	info, err := projectService.GetProjectServer().GetProjectProfile(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetProjectProfileResponse]{
		Msg: info,
	}, nil
}
func (ps *ProjectService) UpdateProjectProfile(ctx context.Context, req *connect.Request[api.UpdateProjectProfileRequest]) (*connect.Response[api.UpdateProjectProfileResponse], error) {
	info, err := projectService.GetProjectServer().UpdateProjectProfile(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UpdateProjectProfileResponse]{
		Msg: info,
	}, nil
}
func (ps *ProjectService) WatchProject(ctx context.Context, req *connect.Request[api.WatchProjectRequest]) (*connect.Response[api.WatchProjectResponse], error) {
	info, err := projectService.GetProjectServer().WatchProject(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.WatchProjectResponse]{
		Msg: info,
	}, nil
}
func (ps *ProjectService) UnWatchProject(ctx context.Context, req *connect.Request[api.UnWatchProjectRequest]) (*connect.Response[api.UnWatchProjectResponse], error) {
	if req.Msg.GetGroupId() <= 0 {
		return nil, errors.New("group id is empty")
	}
	if req.Msg.GetProjectId() <= 0 {
		return nil, errors.New("project id is empty")
	}
	if req.Msg.GetUserId() <= 0 {
		return nil, errors.New("user id is empty")
	}
	info, err := projectService.GetProjectServer().UnWatchProject(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UnWatchProjectResponse]{
		Msg: info,
	}, nil
}

func (ps *ProjectService) SearchProject(ctx context.Context, req *connect.Request[api.SearchAllProjectRequest]) (*connect.Response[api.SearchAllProjectResponse], error) {
	info, err := projectService.GetProjectServer().SearchProject(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.SearchAllProjectResponse]{
		Msg: info,
	}, nil
}
func (ps *ProjectService) ExploreProject(ctx context.Context, req *connect.Request[api.ExploreProjectsRequest]) (*connect.Response[api.ExploreProjectsResponse], error) {

	return &connect.Response[api.ExploreProjectsResponse]{
		Msg: nil,
	}, nil
}
func (ps *ProjectService) GetProjectItems(ctx context.Context, req *connect.Request[api.GetProjectItemsRequest]) (*connect.Response[api.GetProjectItemsResponse], error) {
	info, err := itemService.GetItemServer().GetProjectItems(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetProjectItemsResponse]{
		Msg: info,
	}, nil
}

func (ps *ProjectService) GetProjectMembers(ctx context.Context, req *connect.Request[api.GetProjectMembersRequest]) (*connect.Response[api.GetProjectMembersResponse], error) {
	info, err := projectService.GetProjectServer().GetProjectMembers(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetProjectMembersResponse]{
		Msg: info,
	}, nil
}

func (ps *ProjectService) GetProjectWatcher(ctx context.Context, req *connect.Request[api.GetProjectWatcherRequest]) (*connect.Response[api.GetProjectWatcherResponse], error) {
	info, err := projectService.GetProjectServer().GetProjectWatcher(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetProjectWatcherResponse]{
		Msg: info,
	}, nil
}
