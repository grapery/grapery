package user

import (
	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

type UserGroupService struct {
}

func (ug *UserGroupService) GetGroups(uid int64) ([]*models.Group, error) {
	var err error
	log.Errorf("get user group failed :%s", err)
	return nil, nil
}

func (ug *UserGroupService) JoinGroup(uid, groupID int64) error {
	var err error
	log.Errorf("user join group failed :%s", err)
	return nil
}

func (ug *UserGroupService) LeaveGroup(uid, groupID int64) error {
	var err error
	log.Errorf("user leave group failed :%s", err)
	return nil
}

func (ug *UserGroupService) GetGroupByName(uid int64, name string) ([]*models.Group, error) {
	var err error
	log.Errorf("get user group failed :%s", err)
	return nil, nil
}
