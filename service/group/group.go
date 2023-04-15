package group

import (
	// "net/http"
	"context"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"

	"github.com/grapery/grapery/api"
	groupService "github.com/grapery/grapery/pkg/group"
	itemService "github.com/grapery/grapery/pkg/item"
)

type GroupService struct {
}

func (g *GroupService) CreateGroup(ctx context.Context, req *api.CreateGroupReqeust) (*api.CreateGroupResponse, error) {
	ret, err := groupService.GetGroupServer().CreateGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) GetGroup(ctx context.Context, req *api.GetGroupReqeust) (*api.GetGroupResponse, error) {
	ret, err := groupService.GetGroupServer().GetGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (*api.GetGroupActivesResponse, error) {
	ret, err := groupService.GetGroupServer().GetGroupActives(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) UpdateGroupInfo(ctx context.Context, req *api.UpdateGroupInfoRequest) (*api.UpdateGroupInfoResponse, error) {
	ret, err := groupService.GetGroupServer().UpdateGroupInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (*api.DeleteGroupResponse, error) {
	ret, err := groupService.GetGroupServer().DeleteGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) FetchGroupMembers(ctx context.Context, req *api.FetchGroupMembersRequest) (*api.FetchGroupMembersResponse, error) {
	ret, err := groupService.GetGroupServer().FetchGroupMembers(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) SearchGroup(ctx context.Context, req *api.SearchGroupReqeust) (*api.SearchGroupResponse, error) {
	ret, err := groupService.GetGroupServer().SearchGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) FetchGroupProjects(ctx context.Context, req *api.FetchGroupProjectsReqeust) (*api.FetchGroupProjectsResponse, error) {
	ret, err := groupService.GetGroupServer().FetchGroupProjects(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) JoinGroup(ctx context.Context, req *api.JoinGroupRequest) (*api.JoinGroupResponse, error) {
	ret, err := groupService.GetGroupServer().JoinGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (g *GroupService) LeaveGroup(ctx context.Context, req *api.LeaveGroupRequest) (*api.LeaveGroupResponse, error) {
	ret, err := groupService.GetGroupServer().LeaveGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (g *GroupService) SearchGroupProject(ctx context.Context, req *api.SearchProjectRequest) (*api.SearchProjectResponse, error) {
	ret, err := groupService.GetGroupServer().QueryGroupProject(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (g *GroupService) GetGroupItems(ctx context.Context, req *api.GetGroupItemsRequest) (*api.GetGroupItemsResponse, error) {
	ret, err := itemService.GetItemServer().GetGroupItems(ctx, req)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
