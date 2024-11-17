package group

import (
	// "net/http"
	"context"

	"connectrpc.com/connect"
	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"

	api "github.com/grapery/common-protoc/gen"
	groupService "github.com/grapery/grapery/pkg/group"
	itemService "github.com/grapery/grapery/pkg/item"
)

type GroupService struct {
}

func (g *GroupService) CreateGroup(ctx context.Context, req *connect.Request[api.CreateGroupReqeust]) (*connect.Response[api.CreateGroupResponse], error) {
	ret, err := groupService.GetGroupServer().CreateGroup(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.CreateGroupResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) GetGroup(ctx context.Context, req *connect.Request[api.GetGroupReqeust]) (*connect.Response[api.GetGroupResponse], error) {
	ret, err := groupService.GetGroupServer().GetGroup(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetGroupResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) GetGroupActives(ctx context.Context, req *connect.Request[api.GetGroupActivesRequest]) (*connect.Response[api.GetGroupActivesResponse], error) {
	ret, err := groupService.GetGroupServer().GetGroupActives(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetGroupActivesResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) UpdateGroupInfo(ctx context.Context, req *connect.Request[api.UpdateGroupInfoRequest]) (*connect.Response[api.UpdateGroupInfoResponse], error) {
	ret, err := groupService.GetGroupServer().UpdateGroupInfo(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UpdateGroupInfoResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) DeleteGroup(ctx context.Context, req *connect.Request[api.DeleteGroupRequest]) (*connect.Response[api.DeleteGroupResponse], error) {
	ret, err := groupService.GetGroupServer().DeleteGroup(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.DeleteGroupResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) FetchGroupMembers(ctx context.Context, req *connect.Request[api.FetchGroupMembersRequest]) (*connect.Response[api.FetchGroupMembersResponse], error) {
	ret, err := groupService.GetGroupServer().FetchGroupMembers(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.FetchGroupMembersResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) SearchGroup(ctx context.Context, req *connect.Request[api.SearchGroupReqeust]) (*connect.Response[api.SearchGroupResponse], error) {
	ret, err := groupService.GetGroupServer().SearchGroup(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.SearchGroupResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) FetchGroupProjects(ctx context.Context, req *connect.Request[api.FetchGroupProjectsReqeust]) (*connect.Response[api.FetchGroupProjectsResponse], error) {
	ret, err := groupService.GetGroupServer().FetchGroupProjects(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.FetchGroupProjectsResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) JoinGroup(ctx context.Context, req *connect.Request[api.JoinGroupRequest]) (*connect.Response[api.JoinGroupResponse], error) {
	ret, err := groupService.GetGroupServer().JoinGroup(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.JoinGroupResponse]{
		Msg: ret,
	}, nil
}
func (g *GroupService) LeaveGroup(ctx context.Context, req *connect.Request[api.LeaveGroupRequest]) (*connect.Response[api.LeaveGroupResponse], error) {
	ret, err := groupService.GetGroupServer().LeaveGroup(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.LeaveGroupResponse]{
		Msg: ret,
	}, nil
}

func (g *GroupService) SearchGroupProject(ctx context.Context, req *connect.Request[api.SearchProjectRequest]) (*connect.Response[api.SearchProjectResponse], error) {
	ret, err := groupService.GetGroupServer().QueryGroupProject(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.SearchProjectResponse]{
		Msg: ret,
	}, nil
}

func (g *GroupService) GetGroupItems(ctx context.Context, req *connect.Request[api.GetGroupItemsRequest]) (*connect.Response[api.GetGroupItemsResponse], error) {
	ret, err := itemService.GetItemServer().GetGroupItems(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetGroupItemsResponse]{
		Msg: ret,
	}, nil
}

func (g *GroupService) FetchGroupStorys(ctx context.Context, req *connect.Request[api.FetchGroupStorysReqeust]) (*connect.Response[api.FetchGroupStorysResponse], error) {
	ret, err := groupService.GetGroupServer().FetchGroupStorys(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.FetchGroupStorysResponse]{
		Msg: ret,
	}, nil
}

func (g *GroupService) GetGroupProfile(ctx context.Context, req *connect.Request[api.GetGroupProfileRequest]) (*connect.Response[api.GetGroupProfileResponse], error) {
	ret, err := groupService.GetGroupServer().GetGroupProfile(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetGroupProfileResponse]{
		Msg: ret,
	}, nil
}

func (g *GroupService) UpdateGroupProfile(ctx context.Context, req *connect.Request[api.UpdateGroupProfileRequest]) (*connect.Response[api.UpdateGroupProfileResponse], error) {
	ret, err := groupService.GetGroupServer().UpdateGroupProfile(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UpdateGroupProfileResponse]{
		Msg: ret,
	}, nil
}
