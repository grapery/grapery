package user

import (
	"context"

	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
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
	Get(ctx context.Context, uid int64) error
	UpdateAvator(ctx context.Context, uid int64, avator string) error
	Delete(ctx context.Context, uid int64) error
}

type UserService struct {
}

func (user *UserService) Get(ctx context.Context, uid int64) error {
	var u = new(models.User)
	u.ID = uint(uid)
	err := u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return err
	}
	return nil
}

func (u *UserService) Update(ctx context.Context, uid int64) error {
	return nil
}

func (user *UserService) UpdateAvator(ctx context.Context, uid int64, avator string) error {
	var u = new(models.User)
	u.ID = uint(uid)
	u.Avatar = avator
	err := u.UpdateAvatar()
	if err != nil {
		log.Errorf("delete user failed : %s", err.Error())
		return err
	}
	return nil
}

func (user *UserService) Delete(ctx context.Context, uid int64) error {
	var u = new(models.User)
	u.ID = uint(uid)
	err := u.Delete()
	if err != nil {
		log.Errorf("delete user failed : %s", err.Error())
		return err
	}
	return nil
}
