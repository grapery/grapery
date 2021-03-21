package group

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/models"
)

var server GroupServer

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
}

type GroupService struct {
}

func (g *GroupService) GetGroup(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error) {
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.GetByID()
	if err != nil {
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(group.CreatorID)
	err = creator.GetById()
	if err != nil {
		return nil, err
	}
	return &api.GetGroupResponse{
		Info: &api.GroupInfo{
			GroupId: uint64(group.ID),
			Name:    group.Name,
			Avatar:  group.Avatar,
			Desc:    group.ShortDesc,
			Creator: &api.UserInfo{
				UserId:   uint64(creator.ID),
				Name:     creator.Name,
				Avatar:   creator.Avatar,
				Email:    creator.Email,
				Location: creator.Location,
				Desc:     creator.ShortDesc,
			},
			Owner: &api.UserInfo{
				UserId:   uint64(creator.ID),
				Name:     creator.Name,
				Avatar:   creator.Avatar,
				Email:    creator.Email,
				Location: creator.Location,
				Desc:     creator.ShortDesc,
			},
		},
	}, nil
}

func (g *GroupService) GetByName(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error) {
	group := &models.Group{}
	group.Name = req.GetName()
	err = group.GetByName()
	if err != nil {
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(group.CreatorID)
	err = creator.GetById()
	if err != nil {
		return nil, err
	}
	return &api.GetGroupResponse{
		Info: &api.GroupInfo{
			GroupId: uint64(group.ID),
			Name:    group.Name,
			Avatar:  group.Avatar,
			Desc:    group.ShortDesc,
			Creator: &api.UserInfo{
				UserId:   uint64(creator.ID),
				Name:     creator.Name,
				Avatar:   creator.Avatar,
				Email:    creator.Email,
				Location: creator.Location,
				Desc:     creator.ShortDesc,
			},
			Owner: &api.UserInfo{
				UserId:   uint64(creator.ID),
				Name:     creator.Name,
				Avatar:   creator.Avatar,
				Email:    creator.Email,
				Location: creator.Location,
				Desc:     creator.ShortDesc,
			},
		},
	}, nil
}

func (g *GroupService) CreateGroup(ctx context.Context, req *api.CreateGroupReqeust) (resp *api.CreateGroupResponse, err error) {
	group := &models.Group{}
	group.Name = req.Name
	group.CreatorID = req.GetUserId()
	err = group.Create()
	if err != nil {
		return nil, err
	}
	return &api.CreateGroupResponse{}, nil
}

func (g *GroupService) DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (resp *api.DeleteGroupResponse, err error) {
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.Create()
	if err != nil {
		return nil, err
	}
	return &api.DeleteGroupResponse{}, nil
}

func (g *GroupService) GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (resp *api.GetGroupActivesResponse, err error) {
	actives, err := models.GetAcviteByGroupID(req.GetGroupId())
	if err != nil {
		return nil, err
	}
	var list = make([]*api.ActiveInfo, len(*actives), len(*actives))
	for idx, _ := range *actives {
		log.Info((*actives)[idx])
		//list[idx]....
	}
	return &api.GetGroupActivesResponse{
		List: list,
	}, nil
}

func (g *GroupService) UpdateGroupInfo(ctx context.Context, req *api.UpdateGroupInfoRequest) (resp *api.UpdateGroupInfoResponse, err error) {
	return nil, nil
}

func (g *GroupService) FetchGroupMembers(ctx context.Context, req *api.FetchGroupMembersRequest) (resp *api.FetchGroupMembersResponse, err error) {
	return nil, nil
}

func (g *GroupService) FetchGroupProjects(ctx context.Context, req *api.FetchGroupProjectsReqeust) (resp *api.FetchGroupProjectsResponse, err error) {
	return nil, nil
}

func (g *GroupService) JoinGroup(ctx context.Context, req *api.JoinGroupRequest) (resp *api.JoinGroupResponse, err error) {
	return nil, nil
}

func (g *GroupService) LeaveGroup(ctx context.Context, req *api.LeaveGroupRequest) (resp *api.LeaveGroupResponse, err error) {
	return nil, nil
}

func (g *GroupService) SearchGroup(ctx context.Context, req *api.SearchGroupReqeust) (resp *api.SearchGroupResponse, err error) {
	return nil, nil
}
