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
	Get(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error)
	UpdateAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (*api.UpdateUserAvatorResponse, error)
	GetUserGroup(ctx context.Context, req *api.UserGroupRequest) (*api.UserGroupResponse, error)
	GetUserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (*api.UserFollowingGroupResponse, error)
}

type UserService struct {
}

func (user *UserService) Get(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error) {
	var u = new(models.User)
	u.ID = uint(req.GetUserID())
	err := u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	return &api.UserInfoResponse{
		Info: &api.UserInfo{
			UserID:    uint64(u.ID),
			Nickname:  u.Name,
			AvatorUrl: u.Avatar,
			Email:     u.Email,
			Location:  u.Location,
		},
	}, err
}

func (user *UserService) UpdateAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (*api.UpdateUserAvatorResponse, error) {
	var u = new(models.User)
	u.ID = uint(req.GetUserID())
	err := u.UpdateAvatar()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	u.ID = uint(req.GetUserID())
	err = u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	return &api.UpdateUserAvatorResponse{
		Info: &api.UserInfo{
			UserID:    uint64(u.ID),
			Nickname:  u.Name,
			AvatorUrl: u.Avatar,
			Email:     u.Email,
			Location:  u.Location,
		},
	}, err
}
