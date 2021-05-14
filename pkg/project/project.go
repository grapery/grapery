package project

import (
	"context"

	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/convert"
)

var projectServer ProjectServer

func init() {
	projectServer = NewProjectService()
}

func GetProjectServer() ProjectServer {
	return projectServer
}

func NewProjectService() *ProjectService {
	return &ProjectService{}
}

type ProjectServer interface {
	GetProject(ctx context.Context, req *api.GetProjectRequest) (resp *api.GetProjectResponse, err error)
	CreateProject(ctx context.Context, req *api.CreateProjectRequest) (resp *api.CreateProjectResponse, err error)
	UpdateProject(ctx context.Context, req *api.UpdateProjectRequest) (resp *api.UpdateProjectResponse, err error)
	DeleteProject(ctx context.Context, req *api.DeleteProjectRequest) (resp *api.DeleteProjectResponse, err error)
	GetProjectProfile(ctx context.Context, req *api.GetProjectProfileRequest) (resp *api.GetProjectProfileResponse, err error)
	UpdateProjectProfile(ctx context.Context, req *api.UpdateProjectProfileRequest) (resp *api.UpdateProjectProfileResponse, err error)
	WatchProject(ctx context.Context, req *api.WatchProjectReqeust) (resp *api.WatchProjectResponse, err error)
	UnWatchProject(ctx context.Context, req *api.UnWatchProjectReqeust) (resp *api.UnWatchProjectResponse, err error)
	SearchProject(ctx context.Context, req *api.SearchProjectRequest) (resp *api.SearchProjectResponse, err error)
}

type ProjectService struct {
}

func (p *ProjectService) GetProject(ctx context.Context, req *api.GetProjectRequest) (resp *api.GetProjectResponse, err error) {
	project := &models.Project{
		GroupID: req.GetGroupId(),
	}
	project.ID = uint(req.GetProjectId())
	err = project.Get()
	if err != nil {
		return nil, err
	}
	return &api.GetProjectResponse{
		Info: convert.ConvertProjectToApiProjectInfo(project),
	}, nil
}
func (p *ProjectService) CreateProject(ctx context.Context, req *api.CreateProjectRequest) (resp *api.CreateProjectResponse, err error) {
	project := &models.Project{
		Name:      req.GetName(),
		GroupID:   req.GetGroupId(),
		IsAchieve: false,
		IsPrivate: false,
		IsClose:   false,
	}
	err = project.Create()
	if err != nil {
		return nil, err
	}
	err = project.Get()
	if err != nil {
		return nil, err
	}
	return &api.CreateProjectResponse{
		Info: convert.ConvertProjectToApiProjectInfo(project),
	}, nil
}
func (p *ProjectService) UpdateProject(ctx context.Context, req *api.UpdateProjectRequest) (resp *api.UpdateProjectResponse, err error) {
	return nil, nil
}
func (p *ProjectService) DeleteProject(ctx context.Context, req *api.DeleteProjectRequest) (resp *api.DeleteProjectResponse, err error) {
	project := &models.Project{
		GroupID: req.GetGroupId(),
	}
	project.ID = uint(req.GetProjectId())
	err = project.Delete()
	if err != nil {
		return nil, err
	}
	return &api.DeleteProjectResponse{}, nil
}
func (p *ProjectService) GetProjectProfile(ctx context.Context, req *api.GetProjectProfileRequest) (resp *api.GetProjectProfileResponse, err error) {
	return nil, nil
}
func (p *ProjectService) UpdateProjectProfile(ctx context.Context, req *api.UpdateProjectProfileRequest) (resp *api.UpdateProjectProfileResponse, err error) {
	return nil, nil
}
func (p *ProjectService) WatchProject(ctx context.Context, req *api.WatchProjectReqeust) (resp *api.WatchProjectResponse, err error) {
	err = models.StartWatchingProject(
		req.GetUserId(),
		req.GetGroupId(),
		req.GetProjectId(),
	)
	if err != nil {
		return nil, err
	}
	return &api.WatchProjectResponse{}, nil
}

func (p *ProjectService) UnWatchProject(ctx context.Context, req *api.UnWatchProjectReqeust) (resp *api.UnWatchProjectResponse, err error) {
	err = models.StopWatchingProject(
		req.GetUserId(),
		req.GetGroupId(),
		req.GetProjectId(),
	)
	if err != nil {
		return nil, err
	}
	return &api.UnWatchProjectResponse{}, nil
}

func (p *ProjectService) ExploreProjects(ctx context.Context, req *api.SearchProjectRequest) (resp *api.SearchProjectResponse, err error) {
	return nil, nil
}

func (p *ProjectService) SearchProject(ctx context.Context, req *api.SearchProjectRequest) (resp *api.SearchProjectResponse, err error) {
	return nil, nil
}
