package models

import (
	"fmt"
	"time"

	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"

	api "github.com/grapery/grapery/api"
)

var logFieldActive = zap.Fields(zap.String("module", "Active"))

/* Active
加载用户自己的活动记录
加载一个project的用户活动记录
加载用户的开放project的活动记录
加载一个group内的public project的活动记录
*/
type Active struct {
	IDBase
	UserId     uint64         `json:"user_id,omitempty"`
	ActiveType api.ActiveType `json:"active_type,omitempty"`
	ItemID     uint64         `json:"item_id,omitempty"`
	ProjectID  uint64         `json:"project_id,omitempty"`
	GroupID    uint64         `json:"group_id,omitempty"`
}

func (a Active) TableName() string {
	return "active"
}

func (a *Active) Create() error {
	if err := DataBase().Model(Active{}).Create(a).Error; err != nil {
		log.Log().WithOptions(logFieldActive).Error(fmt.Sprintf("create new active [%d] failed : [%s]", a.ID, err.Error()))
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
		log.Log().WithOptions(logFieldActive).Error(fmt.Sprintf("update active [%d] deleted failed ", a.ID))
		return fmt.Errorf("deleted active [%d] failed ", a.ID)
	}
	log.Log().WithOptions(logFieldActive).Info(fmt.Sprintf("delete active [%d] success", a.ID))
	return nil
}

func GetAcviteByUserID(userID uint64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and delete = 0", userID).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldActive).Error(fmt.Sprintf("get user [%d] active failed ", userID))
		return nil, err
	}

	return ret, nil
}

func GetActiveListByTimeRange(start time.Time, end time.Time) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("created_at < ? and  created_at > ? and delete = 0", end, start).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldActive).
			Error(fmt.Sprintf("get active in range [%s--%s] failed ", start.String(), end.String()))
		return nil, err
	}
	return ret, nil
}

func GetActiveListByActiveType(creatorID uint64, activeType uint) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("creator_id = ? and active_type = ? and delete = 0", creatorID, activeType).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldActive).
			Error(fmt.Sprintf("get user [%d] active type [%d] failed ", creatorID, activeType))
		return nil, err
	}
	return ret, nil
}

func GetAcviteByProjectID(projectID uint64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("project_id = ? and delete = 0", projectID).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldActive).
			Error(fmt.Sprintf("get project [%d] active failed ", projectID))
		return nil, err
	}

	return ret, nil
}

func GetAcviteByGroupID(groupID uint64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("group_id = ? and delete = 0", groupID).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldActive).Error(fmt.Sprintf("get group [%d] active failed ", groupID))
		return nil, err
	}

	return ret, nil
}
