package user

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/models"
)

var userServer UserServer

func init() {
	userServer = NewUserSerivce()
}

func GetUserServer() UserServer {
	return userServer
}

func NewUserSerivce() *UserService {
	return &UserService{}
}

type UserServer interface {
	GetUserInfo(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error)
	UpdateAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (*api.UpdateUserAvatorResponse, error)
	GetUserGroup(ctx context.Context, req *api.UserGroupRequest) (*api.UserGroupResponse, error)
	GetUserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (*api.UserFollowingGroupResponse, error)
	UpdateUser(ctx context.Context, req *api.UserUpdateRequest) (*api.UserUpdateResponse, error)
	StartFollowUser(ctx context.Context, req *api.StartFollowUserRequest) (*api.StartFollowUserResponse, error)
	StopFollowUser(ctx context.Context, req *api.StopFollowUserRequest) (*api.StopFollowUserResponse, error)
	FetchUserActives(ctx context.Context, req *api.FetchUserActivesRequest) (*api.FetchUserActivesResponse, error)
	UserFollowing(ctx context.Context, req *api.UserFollowingRequest) (*api.UserFollowingResponse, error)
	UserFollower(ctx context.Context, req *api.UserFollowerRequest) (*api.UserFollowerResponse, error)
	SearchUser(ctx context.Context, req *api.SearchUserRequest) (*api.SearchUserResponse, error)
}

type UserService struct {
}

func (user *UserService) GetUserInfo(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error) {
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err := u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	return &api.UserInfoResponse{
		Info: &api.UserInfo{
			UserId:   uint64(u.ID),
			Name:     u.Name,
			Avatar:   u.Avatar,
			Email:    u.Email,
			Location: u.Location,
		},
	}, err
}

func (user *UserService) UpdateAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (
	*api.UpdateUserAvatorResponse, error) {
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err := u.UpdateAvatar()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	u.ID = uint(req.GetUserId())
	err = u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	return &api.UpdateUserAvatorResponse{
		Info: &api.UserInfo{
			UserId:   uint64(u.ID),
			Name:     u.Name,
			Avatar:   u.Avatar,
			Email:    u.Email,
			Location: u.Location,
		},
	}, err
}

func (user *UserService) GetUserGroup(ctx context.Context, req *api.UserGroupRequest) (*api.UserGroupResponse, error) {
	return &api.UserGroupResponse{
		List: nil,
	}, nil
}
func (user *UserService) GetUserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (
	*api.UserFollowingGroupResponse, error) {
	return &api.UserFollowingGroupResponse{
		List: nil,
	}, nil
}

func (user *UserService) UpdateUser(ctx context.Context, req *api.UserUpdateRequest) (
	*api.UserUpdateResponse, error) {
	return &api.UserUpdateResponse{}, nil
}
func (user *UserService) StartFollowUser(ctx context.Context, req *api.StartFollowUserRequest) (
	*api.StartFollowUserResponse, error) {
	return &api.StartFollowUserResponse{}, nil
}
func (user *UserService) StopFollowUser(ctx context.Context, req *api.StopFollowUserRequest) (
	*api.StopFollowUserResponse, error) {
	return &api.StopFollowUserResponse{}, nil
}
func (user *UserService) FetchUserActives(ctx context.Context, req *api.FetchUserActivesRequest) (
	*api.FetchUserActivesResponse, error) {
	return &api.FetchUserActivesResponse{
		List: nil,
	}, nil
}
func (user *UserService) UserFollowing(ctx context.Context, req *api.UserFollowingRequest) (
	*api.UserFollowingResponse, error) {
	return &api.UserFollowingResponse{
		List: nil,
	}, nil
}
func (user *UserService) UserFollower(ctx context.Context, req *api.UserFollowerRequest) (
	*api.UserFollowerResponse, error) {
	return &api.UserFollowerResponse{
		List: nil,
	}, nil
}

func (user *UserService) SearchUser(ctx context.Context, req *api.SearchUserRequest) (
	*api.SearchUserResponse, error) {
	return nil, nil
}
