package models

import (
	"fmt"
	"time"

	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/log"
)

type Active struct {
	IDBase
	UserId     uint64         `json:"user_id,omitempty"`
	ActiveType api.ActiveType `json:"active_type,omitempty"`
	ItemID     uint64         `json:"item_id,omitempty"`
	ProjectID  uint64         `json:"project_id,omitempty"`
}

func (a Active) TableName() string {
	return "active"
}

func (a *Active) Create() error {
	if err := DataBase().Model(Active{}).Create(a).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("create new active [%d] failed : [%s]", a.ID, err.Error()))
		return fmt.Errorf("create new active [%d] failed: %s", a.ID, err.Error())
	}
	return nil
}

func (a *Active) Get() error {
	if err := DataBase().Model(Active{}).First(a).Error; err != nil {
		return err
	}
	return nil
}

func (a *Active) Delete() error {
	if err := DataBase().Model(Active{}).Update("deleted", 1); err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("update active [%d] deleted failed ", a.ID))
		return fmt.Errorf("deleted active [%d] failed ", a.ID)
	}
	log.Log().WithOptions(logFieldModels).Info(fmt.Sprintf("delete active [%d] success", a.ID))
	return nil
}

func GetActiveByUserID(userID uint64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and delete = 0", userID).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] active failed ", userID))
		return nil, err
	}

	return ret, nil
}

func GetActiveListByTimeRange(start time.Time, end time.Time) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("created_at < ? and  created_at > ? and delete = 0", end, start).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).
			Error(fmt.Sprintf("get active in range [%s--%s] failed ", start.String(), end.String()))
		return nil, err
	}
	return ret, nil
}

func GetActiveListByActiveType(creatorID uint64, activeType uint) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().
		Model(Active{}).
		Where("creator_id = ? and active_type = ? and delete = 0", creatorID, activeType).
		Scan(ret).
		Error; err != nil {
		log.Log().WithOptions(logFieldModels).
			Error(fmt.Sprintf("get user [%d] active type [%d] failed ", creatorID, activeType))
		return nil, err
	}
	return ret, nil
}

func GetActiveByProjectID(projectID uint64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("project_id = ? and delete = 0", projectID).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).
			Error(fmt.Sprintf("get project [%d] active failed ", projectID))
		return nil, err
	}

	return ret, nil
}

func GetActiveByGroupID(groupID uint64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("group_id = ? and delete = 0", groupID).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get group [%d] active failed ", groupID))
		return nil, err
	}

	return ret, nil
}
