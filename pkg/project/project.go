package project

import (
	"context"

	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/log"
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
	GetProjectInfo(ctx context.Context, req *api.GetProjectRequest) (resp *api.GetProjectResponse, err error)
	GetProjectList(ctx context.Context, req *api.GetProjectListRequest) (resp *api.GetProjectListResponse, err error)
	CreateProject(ctx context.Context, req *api.CreateProjectRequest) (resp *api.CreateProjectResponse, err error)
	UpdateProject(ctx context.Context, req *api.UpdateProjectRequest) (resp *api.UpdateProjectResponse, err error)
	DeleteProject(ctx context.Context, req *api.DeleteProjectRequest) (resp *api.DeleteProjectResponse, err error)
	GetProjectProfile(ctx context.Context, req *api.GetProjectProfileRequest) (resp *api.GetProjectProfileResponse, err error)
	UpdateProjectProfile(ctx context.Context, req *api.UpdateProjectProfileRequest) (resp *api.UpdateProjectProfileResponse, err error)
	WatchProject(ctx context.Context, req *api.WatchProjectReqeust) (resp *api.WatchProjectResponse, err error)
	UnWatchProject(ctx context.Context, req *api.UnWatchProjectReqeust) (resp *api.UnWatchProjectResponse, err error)
	SearchGroupProject(ctx context.Context, req *api.SearchProjectRequest) (resp *api.SearchProjectResponse, err error)
	SearchProject(ctx context.Context, req *api.SearchAllProjectRequest) (resp *api.SearchAllProjectResponse, err error)
	ExploreProjects(ctx context.Context, req *api.ExploreProjectsRequest) (resp *api.ExploreProjectsResponse, err error)
	GetProjectMembers(ctx context.Context, req *api.GetProjectMembersRequest) (resp *api.GetProjectMembersResponse, err error)
	GetProjectWatcher(ctx context.Context, req *api.GetProjectWatcherReqeust) (resp *api.GetProjectWatcherResponse, err error)
}

type ProjectService struct {
}

func (p *ProjectService) GetProjectInfo(ctx context.Context, req *api.GetProjectRequest) (resp *api.GetProjectResponse, err error) {
	project := &models.Project{
		GroupID: int64(req.GetGroupId()),
	}
	project.ID = uint(req.GetProjectId())
	err = project.Get()
	if err != nil {
		log.Log().Error("get project failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("get project success", zap.String("project_name", project.Name))
	return &api.GetProjectResponse{
		Info: convert.ConvertProjectToApiProjectInfo(project),
	}, nil
}

func (p *ProjectService) GetProjectList(ctx context.Context, req *api.GetProjectListRequest) (resp *api.GetProjectListResponse, err error) {
	projects, err := models.GetGroupProjects(int64(req.GetGroupId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get project list failed", zap.Error(err))
		return nil, err
	}
	var tempList = make([]*api.ProjectInfo, 0)
	for _, val := range projects {
		tempList = append(tempList, convert.ConvertProjectToApiProjectInfo(val))
	}
	log.Log().Info("get project list success", zap.Int("project_num", len(tempList)))
	return &api.GetProjectListResponse{
		List:   tempList,
		Offset: req.GetOffset() + int64(len(tempList)),
	}, nil
}

func (p *ProjectService) CreateProject(ctx context.Context, req *api.CreateProjectRequest) (resp *api.CreateProjectResponse, err error) {
	project := &models.Project{
		Name:    req.GetName(),
		GroupID: int64(req.GetGroupId()),
	}
	err = project.Create()
	if err != nil {
		log.Log().Error("create project failed", zap.Error(err))
		return nil, err
	}
	err = project.Get()
	if err != nil {
		log.Log().Error("get project failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("create project success", zap.String("project_name", project.Name))
	return &api.CreateProjectResponse{
		Info: convert.ConvertProjectToApiProjectInfo(project),
	}, nil
}
func (p *ProjectService) UpdateProject(ctx context.Context, req *api.UpdateProjectRequest) (resp *api.UpdateProjectResponse, err error) {
	project := &models.Project{
		GroupID: int64(req.GetGroupId()),
	}
	project.ID = uint(req.GetProjectId())
	err = project.Get()
	if err != nil {
		log.Log().Error("get project failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("update project success", zap.String("project_name", project.Name))
	return nil, nil
}
func (p *ProjectService) DeleteProject(ctx context.Context, req *api.DeleteProjectRequest) (resp *api.DeleteProjectResponse, err error) {
	project := &models.Project{
		GroupID: int64(req.GetGroupId()),
	}
	project.ID = uint(req.GetProjectId())
	err = project.Delete()
	if err != nil {
		log.Log().Error("delete project failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("delete project success", zap.String("project_name", project.Name))
	return &api.DeleteProjectResponse{}, nil
}
func (p *ProjectService) GetProjectProfile(ctx context.Context, req *api.GetProjectProfileRequest) (resp *api.GetProjectProfileResponse, err error) {
	project := &models.Project{
		GroupID: int64(req.GetGroupId()),
	}
	project.ID = uint(req.GetProjectId())
	err = project.Get()
	if err != nil {
		log.Log().Error("get project failed", zap.Error(err))
		return nil, err
	}
	resp = new(api.GetProjectProfileResponse)
	resp.GroupId = req.GetGroupId()
	resp.ProjectId = req.GetProjectId()
	resp.UserId = req.UserId
	resp.Info = &api.ProjectProfileInfo{
		ProjectId:   req.GetProjectId(),
		GroupId:     int32(req.GetGroupId()),
		Description: project.Description,
		Avatar:      project.Avatar,
		ScopeType:   project.Visable,
		IsAchieve:   project.IsAchieve,
		IsClose:     project.IsClose,
	}
	log.Log().Info("get project profile success", zap.String("project_name", project.Name))
	return resp, nil
}
func (p *ProjectService) UpdateProjectProfile(ctx context.Context, req *api.UpdateProjectProfileRequest) (resp *api.UpdateProjectProfileResponse, err error) {
	return nil, nil
}
func (p *ProjectService) WatchProject(ctx context.Context, req *api.WatchProjectReqeust) (resp *api.WatchProjectResponse, err error) {
	err = models.StartWatchingProject(
		int64(req.GetUserId()),
		int64(req.GetGroupId()),
		int64(req.GetProjectId()),
	)
	if err != nil {
		return nil, err
	}
	return &api.WatchProjectResponse{}, nil
}

func (p *ProjectService) UnWatchProject(ctx context.Context, req *api.UnWatchProjectReqeust) (resp *api.UnWatchProjectResponse, err error) {
	err = models.StopWatchingProject(
		int64(req.GetUserId()),
		int64(req.GetGroupId()),
		int64(req.GetProjectId()),
	)
	if err != nil {
		return nil, err
	}
	return &api.UnWatchProjectResponse{}, nil
}

func (p *ProjectService) ExploreProjects(ctx context.Context, req *api.ExploreProjectsRequest) (resp *api.ExploreProjectsResponse, err error) {
	var list []*models.Project
	if req.GetGroupId() != 0 {
		list, err = models.GetGroupProjects(int64(req.GetGroupId()), int(req.GetOffset()), int(req.GetPageSize()))

	} else {
		list, err = models.GetAllProjects(int(req.GetOffset()), int(req.GetPageSize()))
	}
	if err != nil {
		return nil, err
	}
	infoList := make([]*api.ProjectInfo, 0, len(list))
	for _, val := range list {
		infoList = append(infoList, convert.ConvertProjectToApiProjectInfo(val))
	}
	return &api.ExploreProjectsResponse{
		List:     infoList,
		PageSize: int64(len(infoList)),
		Offset:   int64(len(infoList)),
	}, nil
}

func (p *ProjectService) SearchGroupProject(ctx context.Context, req *api.SearchProjectRequest) (resp *api.SearchProjectResponse, err error) {
	list, err := models.GetGroupProjects(int64(req.GetGroupId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	infoList := make([]*api.ProjectInfo, 0, len(list))
	for _, val := range list {
		infoList = append(infoList, convert.ConvertProjectToApiProjectInfo(val))
	}
	return &api.SearchProjectResponse{
		GroupId:  req.GetGroupId(),
		List:     infoList,
		PageSize: int64(len(infoList)),
		Offset:   int64(len(infoList)),
	}, nil
}

func (p *ProjectService) SearchProject(ctx context.Context, req *api.SearchAllProjectRequest) (resp *api.SearchAllProjectResponse, err error) {
	list, err := models.GetAllProjects(int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	infoList := make([]*api.ProjectInfo, 0, len(list))
	for _, val := range list {
		infoList = append(infoList, convert.ConvertProjectToApiProjectInfo(val))
	}
	return &api.SearchAllProjectResponse{
		List:     infoList,
		PageSize: int64(len(infoList)),
		Offset:   int64(len(infoList)),
	}, nil
}

func (p *ProjectService) GetProjectMembers(ctx context.Context, req *api.GetProjectMembersRequest) (resp *api.GetProjectMembersResponse, err error) {
	return nil, nil
}
func (p *ProjectService) GetProjectWatcher(ctx context.Context, req *api.GetProjectWatcherReqeust) (resp *api.GetProjectWatcherResponse, err error) {
	return nil, nil
}
