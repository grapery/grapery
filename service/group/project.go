package group

import (
	"context"

	"github.com/grapery/grapery/api"
	itemService "github.com/grapery/grapery/pkg/item"
	projectService "github.com/grapery/grapery/pkg/project"
)

type ProjectService struct {
}

func (ps *ProjectService) GetProjectInfo(ctx context.Context, req *api.GetProjectRequest) (*api.GetProjectResponse, error) {
	info, err := projectService.GetProjectServer().GetProjectInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (ps *ProjectService) GetProjectList(ctx context.Context, req *api.GetProjectListRequest) (*api.GetProjectListResponse, error) {
	info, err := projectService.GetProjectServer().GetProjectList(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (ps *ProjectService) CreateProject(ctx context.Context, req *api.CreateProjectRequest) (*api.CreateProjectResponse, error) {
	info, err := projectService.GetProjectServer().CreateProject(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ps *ProjectService) UpdateProject(ctx context.Context, req *api.UpdateProjectRequest) (*api.UpdateProjectResponse, error) {
	info, err := projectService.GetProjectServer().UpdateProject(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ps *ProjectService) DeleteProject(ctx context.Context, req *api.DeleteProjectRequest) (*api.DeleteProjectResponse, error) {
	info, err := projectService.GetProjectServer().DeleteProject(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ps *ProjectService) GetProjectProfile(ctx context.Context, req *api.GetProjectProfileRequest) (*api.GetProjectProfileResponse, error) {
	info, err := projectService.GetProjectServer().GetProjectProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ps *ProjectService) UpdateProjectProfile(ctx context.Context, req *api.UpdateProjectProfileRequest) (*api.UpdateProjectProfileResponse, error) {
	info, err := projectService.GetProjectServer().UpdateProjectProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ps *ProjectService) WatchProject(ctx context.Context, req *api.WatchProjectReqeust) (*api.WatchProjectResponse, error) {
	info, err := projectService.GetProjectServer().WatchProject(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ps *ProjectService) UnWatchProject(ctx context.Context, req *api.UnWatchProjectReqeust) (*api.UnWatchProjectResponse, error) {
	info, err := projectService.GetProjectServer().UnWatchProject(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (ps *ProjectService) SearchProject(ctx context.Context, req *api.SearchAllProjectRequest) (*api.SearchAllProjectResponse, error) {
	info, err := projectService.GetProjectServer().SearchProject(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ps *ProjectService) ExploreProject(ctx context.Context, req *api.ExploreProjectsRequest) (*api.ExploreProjectsResponse, error) {

	return nil, nil
}
func (ps *ProjectService) GetProjectItems(ctx context.Context, req *api.GetProjectItemsRequest) (*api.GetProjectItemsResponse, error) {
	info, err := itemService.GetItemServer().GetProjectItems(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
