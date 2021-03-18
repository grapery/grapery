package group

import (
	"context"

	"github.com/grapery/grapery/api"
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
	SearchGroup(ctx context.Context, req *api.SearchGroupReqeust) (resp *api.SearchGroupResponse, err error)
}

type GroupService struct {
}

func (g *GroupService) GetGroup(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error) {
	return nil, nil
}

func (g *GroupService) GetByName(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error) {
	return nil, nil
}

func (g *GroupService) CreateGroup(ctx context.Context, req *api.CreateGroupReqeust) (resp *api.CreateGroupResponse, err error) {
	return nil, nil
}

func (g *GroupService) DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (resp *api.DeleteGroupResponse, err error) {
	return nil, nil
}

func (g *GroupService) GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (resp *api.GetGroupActivesResponse, err error) {
	return nil, nil
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

func (g *GroupService) SearchGroup(ctx context.Context, req *api.SearchGroupReqeust) (resp *api.SearchGroupResponse, err error) {
	return nil, nil
}
