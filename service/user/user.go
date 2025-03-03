package user

import (
	"context"
	"errors"
	"log"
	"strconv"

	"connectrpc.com/connect"

	api "github.com/grapery/common-protoc/gen"
	userService "github.com/grapery/grapery/pkg/user"
	"github.com/grapery/grapery/utils"
)

type UserService struct {
}

func (ts *UserService) UserInit(ctx context.Context, req *connect.Request[api.UserInitRequest]) (*connect.Response[api.UserInitResponse], error) {
	if req.Msg.GetUserId() <= 0 {
		return nil, errors.New("user id is empty")

	}
	info, err := userService.GetUserServer().UserInit(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UserInitResponse]{
		Msg: info,
	}, nil
}

func (ts *UserService) UserWatching(ctx context.Context, req *connect.Request[api.UserWatchingRequest]) (*connect.Response[api.UserWatchingResponse], error) {
	if req.Msg.GetUserId() <= 0 {
		return nil, errors.New("user id is empty")
	}
	info, err := userService.GetUserServer().UserWatching(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UserWatchingResponse]{
		Msg: info,
	}, nil
}
func (ts *UserService) UserGroup(ctx context.Context, req *connect.Request[api.UserGroupRequest]) (*connect.Response[api.UserGroupResponse], error) {
	info, err := userService.GetUserServer().GetUserGroup(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UserGroupResponse]{
		Msg: info,
	}, nil
}
func (ts *UserService) UserFollowingGroup(ctx context.Context, req *connect.Request[api.UserFollowingGroupRequest]) (*connect.Response[api.UserFollowingGroupResponse], error) {
	info, err := userService.GetUserServer().GetUserFollowingGroup(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UserFollowingGroupResponse]{
		Msg: info,
	}, nil
}
func (ts *UserService) UserUpdate(ctx context.Context, req *connect.Request[api.UserUpdateRequest]) (*connect.Response[api.UserUpdateResponse], error) {
	info, err := userService.GetUserServer().UpdateUser(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UserUpdateResponse]{
		Msg: info,
	}, nil
}
func (ts *UserService) FetchActives(ctx context.Context, req *connect.Request[api.FetchActivesRequest]) (*connect.Response[api.FetchActivesResponse], error) {
	info, err := userService.GetUserServer().FetchActives(ctx, req.Msg)
	if err != nil {
		log.Println("FetchActives error", err)
		return nil, err
	}
	return connect.NewResponse(info), nil
}
func (ts *UserService) SearchUser(ctx context.Context, req *connect.Request[api.SearchUserRequest]) (*connect.Response[api.SearchUserResponse], error) {
	info, err := userService.GetUserServer().SearchUser(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.SearchUserResponse]{
		Msg: info,
	}, nil
}

func (ts *UserService) UpdateUserAvator(ctx context.Context, req *connect.Request[api.UpdateUserAvatorRequest]) (*connect.Response[api.UpdateUserAvatorResponse], error) {
	info, err := userService.GetUserServer().UpdateAvator(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(info), nil
}
func (ts *UserService) UserInfo(ctx context.Context, req *connect.Request[api.UserInfoRequest]) (*connect.Response[api.UserInfoResponse], error) {
	uidTemp := req.Header().Get(utils.UserIdKey)
	if len(uidTemp) == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
	}
	uid, _ := strconv.Atoi(uidTemp)
	if req.Msg.GetUserId() == 0 {
		req.Msg.UserId = int64(uid)
	}
	info, err := userService.GetUserServer().GetUserInfo(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UserInfoResponse]{
		Msg: info,
	}, nil
}

func (ts *UserService) GetUserProfile(ctx context.Context, req *connect.Request[api.GetUserProfileRequest]) (*connect.Response[api.GetUserProfileResponse], error) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		log.Println("get user id from context failed: ", err.Error())
		return nil, err
	}
	log.Println("user id: ", userID)
	info, err := userService.GetUserServer().GetUserProfile(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetUserProfileResponse]{
		Msg: info,
	}, nil
}

func (ts *UserService) UpdateUserProfile(ctx context.Context, req *connect.Request[api.UpdateUserProfileRequest]) (*connect.Response[api.UpdateUserProfileResponse], error) {
	_, err := userService.GetUserServer().UpdateUserProfile(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.UpdateUserProfileResponse]{
		Msg: &api.UpdateUserProfileResponse{
			Code:    0,
			Message: "success",
		},
	}, nil
}
