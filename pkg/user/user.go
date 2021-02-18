package user

import (
	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

type UserService struct {
}

func (user *UserService) Get(uid int64) error {
	var u = new(models.User)
	u.ID = uint(uid)
	err := u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return err
	}
	return nil
}

func (u *UserService) Update(uid int64) error {
	return nil
}

func (user *UserService) UpdateAvator(uid int64, avator string) error {
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

func (user *UserService) Delete(uid int64) error {
	var u = new(models.User)
	u.ID = uint(uid)
	err := u.Delete()
	if err != nil {
		log.Errorf("delete user failed : %s", err.Error())
		return err
	}
	return nil
}
