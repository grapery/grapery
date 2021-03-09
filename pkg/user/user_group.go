package user

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/models"
)

var userGroupServer UserGroupServer

func init() {
	userGroupServer = NewUserGroupService()
}

func GetUserGroupServer() UserGroupServer {
	return userGroupServer
}

func NewUserGroupService() *UserGroupService {
	return &UserGroupService{}
}

type UserGroupServer interface {
	GetGroups(ctx context.Context, uid int64) ([]*models.Group, error)
	JoinGroup(ctx context.Context, uid, groupID int64) error
	LeaveGroup(ctx context.Context, uid, groupID int64) error
	GetGroupByName(ctx context.Context, uid int64, name string) ([]*models.Group, error)
}

type UserGroupService struct {
}

func (ug *UserGroupService) GetGroups(ctx context.Context, uid int64) ([]*models.Group, error) {
	var err error
	log.Errorf("get user group failed :%s", err)
	return nil, nil
}

func (ug *UserGroupService) JoinGroup(ctx context.Context, uid, groupID int64) error {
	var err error
	log.Errorf("user join group failed :%s", err)
	return nil
}

func (ug *UserGroupService) LeaveGroup(ctx context.Context, uid, groupID int64) error {
	var err error
	log.Errorf("user leave group failed :%s", err)
	return nil
}

func (ug *UserGroupService) GetGroupByName(ctx context.Context, uid int64, name string) ([]*models.Group, error) {
	var err error
	log.Errorf("get user group failed :%s", err)
	return nil, nil
}
