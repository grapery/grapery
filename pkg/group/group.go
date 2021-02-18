package group

import (
	"github.com/grapery/grapery/models"
)

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
