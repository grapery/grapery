package user

import (
	"context"

	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

var userActiveServer UserActiveServer

func init() {
	userActiveServer = NewUserActiveService()
}

func GetUserActiveServer() UserActiveServer {
	return userActiveServer
}

func NewUserActiveService() *UserActiveService {
	return &UserActiveService{}
}

type UserActiveServer interface {
	GetUserActiveByGroupAndAvtiveType(ctx context.Context, uid uint64, offset int, number int, groupID int64, activeType int) ([]*models.Active, error)
	GetUserAllActive(ctx context.Context, uid uint64, offset int, number int, groupID int64, activeType int) ([]*models.Active, error)
	CreateNewActive(uid uint64, groupID uint64, activeType int) error
	UpdateActive(ctx context.Context, uid uint64, groupID uint64, activeID uint64) error
	DeleteActive(ctx context.Context, uid uint64, groupID uint64, activeID uint64) error
}

// active like a drop or a cell
type UserActiveService struct {
}

func (usc *UserActiveService) GetUserActiveByGroupAndAvtiveType(ctx context.Context, uid uint64, offset int, number int, groupID int64, activeType int) ([]*models.Active, error) {
	var err error
	log.Errorf("get user active failed : %s", err)
	return nil, nil
}

func (usc *UserActiveService) GetUserAllActive(ctx context.Context, uid uint64, offset int, number int, groupID int64, activeType int) ([]*models.Active, error) {
	var err error
	log.Errorf("get user all active failed : %s", err)
	return nil, nil
}

func (usc *UserActiveService) CreateNewActive(ctx context.Context, uid uint64, groupID uint64, activeType int) error {
	var err error
	log.Errorf("get user all active failed : %s", err)
	return nil
}

func (usc *UserActiveService) UpdateActive(ctx context.Context, uid uint64, groupID uint64, activeID uint64) error {
	var err error
	log.Errorf("get user all active failed : %s", err)
	return nil
}

func (usc *UserActiveService) DeleteActive(ctx context.Context, uid uint64, groupID uint64, activeID uint64) error {
	var err error
	log.Errorf("get user all active failed : %s", err)
	return nil
}
