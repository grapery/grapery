package user

import (
	"context"

	"github.com/grapery/grapery/api"
	_ "github.com/grapery/grapery/pkg/project"
)

type UserService struct {
}

func (ts *UserService) UserWatching(ctx context.Context, req *api.UserWatchingRequest) (*api.UserWatchingResponse, error) {
	return nil, nil
}
func (ts *UserService) UserGroup(ctx context.Context, req *api.UserGroupRequest) (*api.UserGroupResponse, error) {
	return nil, nil
}
func (ts *UserService) UserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (*api.UserFollowingGroupResponse, error) {
	return nil, nil
}
func (ts *UserService) UserUpdate(ctx context.Context, req *api.UserUpdateRequest) (*api.UserUpdateResponse, error) {
	return nil, nil
}
func (ts *UserService) FetchUserActives(ctx context.Context, req *api.FetchUserActivesRequest) (*api.FetchUserActivesResponse, error) {
	return nil, nil
}
func (ts *UserService) SearchUser(ctx context.Context, req *api.SearchUserRequest) (*api.SearchUserResponse, error) {
	return nil, nil
}

func (ts *UserService) UpdateUserAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (*api.UpdateUserAvatorResponse, error) {
	return nil, nil
}
func (ts *UserService) UserInfo(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error) {
	return nil, nil
}
