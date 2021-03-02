package group

import (
	"github.com/grapery/grapery/models"
)

var groupServicer GroupServicer

func init() {
	groupServicer = NewGroupService()
}

func GetGroupServicer() GroupServicer {
	return groupServicer
}

func NewGroupService() *GroupService {
	return &GroupService{}
}

type GroupServicer interface {
	Get(groupID uint64) (*models.Group, error)
	GetByName(name string) ([]*models.Group, error)
	CreateGroup(name string, uid int64) error
	DeleteGroup(name string, uid int64) error
}

type GroupService struct {
}

func (g *GroupService) Get(groupID uint64) (*models.Group, error) {
	return nil, nil
}

func (g *GroupService) GetByName(name string) ([]*models.Group, error) {
	return nil, nil
}

func (g *GroupService) CreateGroup(name string, uid int64) error {
	return nil
}

func (g *GroupService) DeleteGroup(name string, uid int64) error {
	return nil
}
