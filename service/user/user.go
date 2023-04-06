package user

import (
	"context"

	"github.com/grapery/grapery/api"
	userService "github.com/grapery/grapery/pkg/user"
)

type UserService struct {
}

func (ts *UserService) UserWatching(ctx context.Context, req *api.UserWatchingRequest) (*api.UserWatchingResponse, error) {
	info, err := userService.GetUserServer().UserWatching(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *UserService) UserGroup(ctx context.Context, req *api.UserGroupRequest) (*api.UserGroupResponse, error) {
	info, err := userService.GetUserServer().GetUserGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *UserService) UserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (*api.UserFollowingGroupResponse, error) {
	info, err := userService.GetUserServer().GetUserFollowingGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *UserService) UserUpdate(ctx context.Context, req *api.UserUpdateRequest) (*api.UserUpdateResponse, error) {
	info, err := userService.GetUserServer().UpdateUser(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *UserService) FetchUserActives(ctx context.Context, req *api.FetchUserActivesRequest) (*api.FetchUserActivesResponse, error) {
	info, err := userService.GetUserServer().FetchUserActives(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *UserService) SearchUser(ctx context.Context, req *api.SearchUserRequest) (*api.SearchUserResponse, error) {
	info, err := userService.GetUserServer().SearchUser(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (ts *UserService) UpdateUserAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (*api.UpdateUserAvatorResponse, error) {
	info, err := userService.GetUserServer().UpdateAvator(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
func (ts *UserService) UserInfo(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error) {
	info, err := userService.GetUserServer().GetUserInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	return info, nil
}
