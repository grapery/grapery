package group

import (
	// "net/http"
	"context"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"

	"github.com/grapery/grapery/api"
)

type GroupService struct {
}

func (ts *GroupService) CreateGroup(ctx context.Context, req *api.CreateGroupReqeust) (*api.CreateGroupResponse, error) {
	return nil, nil
}
func (ts *GroupService) GetGroup(ctx context.Context, req *api.GetGroupReqeust) (*api.GetGroupResponse, error) {
	return nil, nil
}
func (ts *GroupService) GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (*api.GetGroupActivesResponse, error) {
	return nil, nil
}
func (ts *GroupService) UpdateGroupInfo(ctx context.Context, req *api.UpdateGroupInfoRequest) (*api.UpdateGroupInfoResponse, error) {
	return nil, nil
}
func (ts *GroupService) DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (*api.DeleteGroupResponse, error) {
	return nil, nil
}
func (ts *GroupService) FetchGroupMembers(ctx context.Context, req *api.FetchGroupMembersRequest) (*api.FetchGroupMembersResponse, error) {
	return nil, nil
}
func (ts *GroupService) SearchGroup(ctx context.Context, req *api.SearchGroupReqeust) (*api.SearchGroupResponse, error) {
	return nil, nil
}
func (ts *GroupService) FetchGroupProjects(ctx context.Context, req *api.FetchGroupProjectsReqeust) (*api.FetchGroupProjectsResponse, error) {
	return nil, nil
}
func (ts *GroupService) JoinGroup(ctx context.Context, req *api.JoinGroupRequest) (*api.JoinGroupResponse, error) {
	return nil, nil
}
func (ts *GroupService) LeaveGroup(ctx context.Context, req *api.LeaveGroupRequest) (*api.LeaveGroupResponse, error) {
	return nil, nil
}

func (ts *GroupService) SearchGroupProject(ctx context.Context, req *api.SearchProjectRequest) (*api.SearchProjectResponse, error) {
	return nil, nil
}

func (ts *GroupService) GetGroupItems(ctx context.Context, req *api.GetGroupItemsRequest) (*api.GetGroupItemsResponse, error) {
	return nil, nil
}

func (ts *GroupService) GetGroupItemComment(ctx context.Context, req *api.GetUserProjectCommentReq) (*api.GetUserProjectCommentResp, error) {
	return nil, nil
}
