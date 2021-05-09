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
	UserWatching(ctx context.Context, req *api.UserWatchingRequest) (*api.UserWatchingResponse, error)
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
	u.Avatar = req.GetAvatar()
	err := u.UpdateAvatar()
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
	list, err := models.GetUserGroups(int(req.GetUserId()), 0, 10)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &api.UserGroupResponse{}, nil
	}
	var groups = make([]*api.GroupInfo, len(list), len(list))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err = u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	info := &api.UserInfo{
		UserId:   uint64(u.ID),
		Name:     u.Name,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Location: u.Location,
	}
	for idx, _ := range list {
		groups[idx] = &api.GroupInfo{}
		groups[idx].Avatar = list[idx].Avatar
		groups[idx].Name = list[idx].Name
		groups[idx].GroupId = uint64(list[idx].ID)
		groups[idx].Desc = list[idx].ShortDesc
		groups[idx].Owner = info
		groups[idx].Creator = info
	}
	return &api.UserGroupResponse{
		List: groups,
	}, nil
}
func (user *UserService) GetUserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (
	*api.UserFollowingGroupResponse, error) {
	list, err := models.GetUserJoinedGroups(int(req.GetUserId()), 0, 10)
	if err != nil {
		return nil, err
	}
	var groups = make([]*api.GroupInfo, 0, len(list))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err = u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	info := &api.UserInfo{
		UserId:   uint64(u.ID),
		Name:     u.Name,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Location: u.Location,
	}
	for idx, _ := range list {
		groups[idx] = &api.GroupInfo{}
		groups[idx].Avatar = list[idx].Avatar
		groups[idx].Name = list[idx].Name
		groups[idx].GroupId = uint64(list[idx].ID)
		groups[idx].Desc = list[idx].ShortDesc
		groups[idx].Owner = info
		groups[idx].Creator = info
	}
	return &api.UserFollowingGroupResponse{
		List: groups,
	}, nil
}

func (user *UserService) UpdateUser(ctx context.Context, req *api.UserUpdateRequest) (
	*api.UserUpdateResponse, error) {
	u := &models.User{
		Avatar:    req.GetAvatar(),
		Name:      req.GetNickname(),
		ShortDesc: req.GetDesc(),
	}
	err := u.UpdateAvatar()
	if err != nil {
		return nil, err
	}
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

// 组织内搜索指定用户
func (user *UserService) SearchUser(ctx context.Context, req *api.SearchUserRequest) (
	*api.SearchUserResponse, error) {
	return nil, nil
}

func (user *UserService) UserWatching(ctx context.Context, req *api.UserWatchingRequest) (
	*api.UserWatchingResponse, error) {
	list, err := models.GetUserWatchingProjects(int64(req.GetUserId()), 0, 10)
	if err != nil {
		return nil, err
	}
	var projects = make([]*api.ProjectInfo, 0, len(list))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err = u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	info := &api.UserInfo{
		UserId:   uint64(u.ID),
		Name:     u.Name,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Location: u.Location,
	}
	for idx, _ := range list {
		projects[idx] = &api.ProjectInfo{}
		projects[idx].Avatar = list[idx].Avatar
		projects[idx].Name = list[idx].Name
		projects[idx].ProjectId = uint64(list[idx].ID)
		projects[idx].Owner = info
		projects[idx].Creator = info
	}
	return &api.UserWatchingResponse{
		List: projects,
	}, nil
}
