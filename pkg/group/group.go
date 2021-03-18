package group

import (
	"context"

	"github.com/grapery/grapery/models"
)

var server GroupServer

func init() {
	server = NewGroupService()
}

func GetGroupServer() GroupServer {
	return server
}

func NewGroupService() *GroupService {
	return &GroupService{}
}

type GroupServer interface {
	Get(ctx context.Context, groupID uint64) (*models.Group, error)
	GetByName(ctx context.Context, name string) ([]*models.Group, error)
	CreateGroup(ctx context.Context, name string, uid int64) error
	DeleteGroup(ctx context.Context, name string, uid int64) error
}

type GroupService struct {
}

func (g *GroupService) Get(ctx context.Context, groupID uint64) (*models.Group, error) {
	return nil, nil
}

func (g *GroupService) GetByName(ctx context.Context, name string) ([]*models.Group, error) {
	return nil, nil
}

func (g *GroupService) CreateGroup(ctx context.Context, name string, uid int64) error {
	return nil
}

func (g *GroupService) DeleteGroup(ctx context.Context, name string, uid int64) error {
	return nil
}
