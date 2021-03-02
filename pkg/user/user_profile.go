package user

import (
	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

var userProfileSerivcer UserProfileSerivcer

func init() {
	userProfileSerivcer = NewUserProfileSerivce()
}

func GetUserProfileServer() UserGroupServer {
	return userGroupServer
}

func NewUserProfileSerivce() *UserProfileSerivce {
	return &UserProfileSerivce{}
}

type UserProfileSerivcer interface {
	CreateProfile(uid int64) error
	GetUserProfile(uid int64) (*models.UserProfile, error)
	UpdateUserProfile(uid int64) error
}

type UserProfileSerivce struct {
}

func (up *UserProfileSerivce) CreateProfile(uid int64) error {
	return nil
}

func (up *UserProfileSerivce) GetUserProfile(uid int64) (*models.UserProfile, error) {
	var err error
	log.Errorf("get user profile failed : %s", err.Error())
	return nil, nil
}

func (up *UserProfileSerivce) UpdateUserProfile(uid int64) error {
	return nil
}
