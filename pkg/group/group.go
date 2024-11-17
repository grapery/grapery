package group

import (
	"context"

	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/convert"
)

var (
	server         GroupServer
	logFieldModels = zap.Fields(
		zap.String("module", "pkg"))
)

func init() {
	server = NewGroupService()
}

func GetGroupServer() GroupServer {
	return server
}

func NewGroupService() *GroupService {
	return &GroupService{}
}

// need do some log
type GroupServer interface {
	GetGroup(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error)
	GetByName(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error)
	CreateGroup(ctx context.Context, req *api.CreateGroupReqeust) (resp *api.CreateGroupResponse, err error)
	DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (resp *api.DeleteGroupResponse, err error)
	GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (resp *api.GetGroupActivesResponse, err error)
	UpdateGroupInfo(ctx context.Context, req *api.UpdateGroupInfoRequest) (resp *api.UpdateGroupInfoResponse, err error)
	FetchGroupMembers(ctx context.Context, req *api.FetchGroupMembersRequest) (resp *api.FetchGroupMembersResponse, err error)
	FetchGroupProjects(ctx context.Context, req *api.FetchGroupProjectsReqeust) (resp *api.FetchGroupProjectsResponse, err error)
	JoinGroup(ctx context.Context, req *api.JoinGroupRequest) (resp *api.JoinGroupResponse, err error)
	LeaveGroup(ctx context.Context, req *api.LeaveGroupRequest) (resp *api.LeaveGroupResponse, err error)
	SearchGroup(ctx context.Context, req *api.SearchGroupReqeust) (resp *api.SearchGroupResponse, err error)

	QueryGroupProject(ctx context.Context, req *api.SearchProjectRequest) (*api.SearchProjectResponse, error)
	FetchGroupStorys(ctx context.Context, req *api.FetchGroupStorysReqeust) (*api.FetchGroupStorysResponse, error)

	GetGroupProfile(ctx context.Context, req *api.GetGroupProfileRequest) (*api.GetGroupProfileResponse, error)
	UpdateGroupProfile(ctx context.Context, req *api.UpdateGroupProfileRequest) (*api.UpdateGroupProfileResponse, error)
}

type GroupService struct {
}

func (g *GroupService) GetGroup(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error) {
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.GetByID()
	if err != nil {
		log.Error("get group by id error: ", err.Error())
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(req.GetUserId())
	err = creator.GetById()
	if err != nil {
		log.Error("get user info by id failed:", err.Error())
		return nil, err
	}
	return &api.GetGroupResponse{
		Code:    0,
		Message: "ok",
		Data: &api.GetGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    group.Name,
				Avatar:  group.Avatar,
				Desc:    group.ShortDesc,
				Creator: int64(creator.ID),
				Owner:   int64(creator.ID),
			},
		},
	}, nil
}

func (g *GroupService) GetByName(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error) {
	group := &models.Group{}
	group.Name = req.GetName()
	err = group.GetByName()
	if err != nil {
		log.Error("get group by name error: ", err.Error())
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(group.CreatorID)
	err = creator.GetById()
	if err != nil {
		log.Error("get user info by id failed:", err.Error())
		return nil, err
	}
	return &api.GetGroupResponse{
		Code:    0,
		Message: "ok",
		Data: &api.GetGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    group.Name,
				Avatar:  group.Avatar,
				Desc:    group.ShortDesc,
				Creator: int64(creator.ID),
				Owner:   int64(creator.ID),
			},
		},
	}, nil
}

func (g *GroupService) CreateGroup(ctx context.Context, req *api.CreateGroupReqeust) (resp *api.CreateGroupResponse, err error) {
	group := &models.Group{}
	group.Name = req.Name
	group.CreatorID = req.GetUserId()
	group.OwnerID = req.GetUserId()
	group.Members = 1
	desc := req.GetDescription()
	if desc == "" {
		desc = "这是一个神秘的组织"
	}
	group.ShortDesc = desc
	group.Description = desc
	if req.GetAvatar() != "" {
		group.Avatar = req.GetAvatar()
	} else {
		group.Avatar = utils.DefaultGroupAvatorUrl
	}
	err = group.Create()
	if err != nil {
		log.Info("create group error: ", err.Error())
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(group.CreatorID)
	err = creator.GetById()
	if err != nil {
		log.Info("get user info by id failed:", err.Error())
		return nil, err
	}
	log.Info("create group success: ", group.ID, group.Name, group.CreatorID)
	err = models.CreateGroupProfile(ctx,
		int64(group.ID),
		desc,
		0, false, 1)
	if err != nil {
		return nil, err
	}
	return &api.CreateGroupResponse{
		Code:    0,
		Message: "ok",
		Data: &api.CreateGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    group.Name,
				Avatar:  group.Avatar,
				Desc:    group.ShortDesc,
				Creator: int64(creator.ID),
				Owner:   int64(creator.ID),
			},
		},
	}, nil
}

func (g *GroupService) DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (resp *api.DeleteGroupResponse, err error) {
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.Delete()
	if err != nil {
		return nil, err
	}
	return &api.DeleteGroupResponse{
		Code:    0,
		Message: "ok",
		Data:    &api.DeleteGroupResponse_Data{},
	}, nil
}

func (g *GroupService) GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (resp *api.GetGroupActivesResponse, err error) {
	actives, err := models.GetActiveByGroupID(req.GetGroupId())
	if err != nil {
		return nil, err
	}
	var list = make([]*api.ActiveInfo, len(*actives), len(*actives))
	for idx, _ := range *actives {
		log.Info((*actives)[idx])
		//list[idx]....
	}
	return &api.GetGroupActivesResponse{
		Code:    0,
		Message: "ok",
		Data: &api.GetGroupActivesResponse_Data{
			List: list,
		},
	}, nil
}

func (g *GroupService) UpdateGroupInfo(ctx context.Context, req *api.UpdateGroupInfoRequest) (resp *api.UpdateGroupInfoResponse, err error) {
	group := new(models.Group)
	group.ID = uint(req.GetGroupId())
	err = group.GetByID()
	if err != nil {
		return &api.UpdateGroupInfoResponse{}, err
	}
	if req.GetInfo().GetAvatar() != "" {
		group.Avatar = req.GetInfo().GetAvatar()
	}
	if req.GetInfo().GetDesc() != "" {
		group.ShortDesc = req.GetInfo().GetDesc()
	}
	if req.GetInfo().GetName() != "" {
		group.Name = req.GetInfo().GetName()
	}
	if req.GetInfo().Status != 0 {
		group.Status = int64(req.GetInfo().Status)
	}

	err = group.UpdateAll()
	if err != nil {
		return nil, err
	}
	return &api.UpdateGroupInfoResponse{
		Code:    0,
		Message: "ok",
		Data:    &api.UpdateGroupInfoResponse_Data{},
	}, nil
}

func (g *GroupService) FetchGroupMembers(ctx context.Context, req *api.FetchGroupMembersRequest) (resp *api.FetchGroupMembersResponse, err error) {
	users, err := models.GetGroupMemberInfoList(int(req.GetGroupId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	usersInfo := make([]*api.UserInfo, len(users), len(users))
	for idx := range users {
		usersInfo[idx] = convert.ConvertUserToApiUser(users[idx])
	}

	return &api.FetchGroupMembersResponse{
		Code:    0,
		Message: "ok",
		Data: &api.FetchGroupMembersResponse_Data{
			List:   usersInfo,
			Offset: req.GetOffset() + int64(len(usersInfo)),
			Total:  int64(len(usersInfo)),
		},
	}, nil
}

func (g *GroupService) FetchGroupProjects(ctx context.Context, req *api.FetchGroupProjectsReqeust) (resp *api.FetchGroupProjectsResponse, err error) {
	projects, err := models.GetGroupProjects(int64(req.GetGroupId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	list := make([]*api.ProjectInfo, len(projects), len(projects))
	for idx, val := range projects {
		list[idx] = convert.ConvertProjectToApiProjectInfo(val)
	}
	return &api.FetchGroupProjectsResponse{
		Code:    0,
		Message: "ok",
		Data: &api.FetchGroupProjectsResponse_Data{
			List:     list,
			Offset:   req.GetOffset() + int64(len(list)),
			PageSize: int64(len(list)),
		},
	}, nil
}

func (g *GroupService) JoinGroup(ctx context.Context, req *api.JoinGroupRequest) (resp *api.JoinGroupResponse, err error) {
	groupMember := &models.GroupMember{
		GroupID: req.GetGroupId(),
		UserID:  req.GetUserId(),
	}
	isIn, err := groupMember.IsInGroup()
	if err != nil {
		return nil, err
	}
	if isIn {
		return &api.JoinGroupResponse{}, nil
	}
	err = groupMember.Create()
	if err != nil {
		return nil, err
	}
	err = models.IncGroupProfileMembers(ctx, req.GetGroupId())
	if err != nil {
		return nil, err
	}
	return &api.JoinGroupResponse{}, nil
}

func (g *GroupService) LeaveGroup(ctx context.Context, req *api.LeaveGroupRequest) (resp *api.LeaveGroupResponse, err error) {
	// group 包含 资源（project），处理组（teams）,退出组的话，teams也会同时停止使用
	groupMember := &models.GroupMember{
		GroupID: int64(req.GetGroupId()),
		UserID:  req.GetUserId(),
	}
	isIn, err := groupMember.IsInGroup()
	if err != nil {
		return nil, err
	}
	if !isIn {
		return &api.LeaveGroupResponse{}, nil
	}
	err = groupMember.Delete()
	if err != nil {
		return nil, err
	}
	err = models.DecGroupProfileMembers(ctx, req.GetGroupId())
	if err != nil {
		return nil, err
	}
	return &api.LeaveGroupResponse{}, nil
}

func (g *GroupService) GetGroupProfile(ctx context.Context, req *api.GetGroupProfileRequest) (resp *api.GetGroupProfileResponse, err error) {
	profile := &models.GroupProfile{}
	profile.GroupID = req.GetGroupId()
	profile, err = models.GetGroupProfile(ctx, profile.GroupID)
	if err != nil {
		log.Info("get group profile failed: ", err.Error())
		return nil, err
	}
	if profile == nil {
		log.Info("group profile is nil")
		return &api.GetGroupProfileResponse{
			Code:    0,
			Message: "ok",
			Data: &api.GetGroupProfileResponse_Data{
				Info: &api.GroupProfileInfo{
					GroupId:            req.GetGroupId(),
					Description:        "",
					DefaultProjectList: 0,
					IsVerified:         false,
					GroupFollowerNum:   0,
				},
			},
		}, nil
	}
	return &api.GetGroupProfileResponse{
		Code:    0,
		Message: "ok",
		Data: &api.GetGroupProfileResponse_Data{
			Info: convert.ConvertGroupProfileToApiGroupProfile(profile),
		},
	}, nil
}

func (g *GroupService) UpdateGroupProfile(ctx context.Context, req *api.UpdateGroupProfileRequest) (resp *api.UpdateGroupProfileResponse, err error) {
	profile := req.GetInfo()
	err = models.UpdateGroupProfile(ctx,
		req.GetGroupId(),
		profile.GetDescription(),
		int64(profile.GetDefaultProjectList()),
		profile.GetIsVerified(),
		int64(profile.GetGroupFollowerNum()),
	)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (g *GroupService) SearchGroup(ctx context.Context, req *api.SearchGroupReqeust) (resp *api.SearchGroupResponse, err error) {

	return nil, nil
}

func (g *GroupService) QueryGroupProject(ctx context.Context, req *api.SearchProjectRequest) (*api.SearchProjectResponse, error) {
	return nil, nil
}

func (g *GroupService) FetchGroupStorys(ctx context.Context, req *api.FetchGroupStorysReqeust) (*api.FetchGroupStorysResponse, error) {
	// TODO: 实现获取群组的故事列表
	storys, err := models.GetStoryByGroupID(ctx, req.GetGroupId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		log.Info("get story by group id failed: ", err.Error())
		return nil, err
	}
	list := make([]*api.Story, len(storys), len(storys))
	for idx, val := range storys {
		list[idx] = convert.ConvertStoryToApiStory(val)
	}
	return &api.FetchGroupStorysResponse{
		Code:    0,
		Message: "ok",
		Data: &api.FetchGroupStorysResponse_Data{
			List: list,
		},
	}, nil
}