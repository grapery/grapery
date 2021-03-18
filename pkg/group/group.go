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

type GroupServer interface {
	GetGroup(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error)
	GetByName(ctx context.Context, req *api.GetGroupReqeust) (resp *api.GetGroupResponse, err error)
	CreateGroup(ctx context.Context, req *api.CreateGroupReqeust) (resp *api.CreateGroupResponse, err error)
	DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (resp *api.DeleteGroupResponse, err error)
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
