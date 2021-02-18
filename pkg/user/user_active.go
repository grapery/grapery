package user

import (
	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

// active like a drop or a cell
type UserActiveService struct {
}

func (usc *UserActiveService) GetUserActiveByGroupAndAvtiveType(uid uint64, offset int, number int, groupID int64, activeType int) ([]*models.Active, error) {
	var err error
	log.Errorf("get user active failed : %s", err)
	return nil, nil
}

func (usc *UserActiveService) GetUserAllActive(uid uint64, offset int, number int, groupID int64, activeType int) ([]*models.Active, error) {
	var err error
	log.Errorf("get user all active failed : %s", err)
	return nil, nil
}

func (usc *UserActiveService) CreateNewActive(uid uint64, groupID uint64, activeType int) error {
	var err error
	log.Errorf("get user all active failed : %s", err)
	return nil
}

func (usc *UserActiveService) UpdateActive(uid uint64, groupID uint64, activeID uint64) error {
	var err error
	log.Errorf("get user all active failed : %s", err)
	return nil
}

func (usc *UserActiveService) DeleteActive(uid uint64, groupID uint64, activeID uint64) error {
	var err error
	log.Errorf("get user all active failed : %s", err)
	return nil
}
