package models

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	ActiveTypeShortWord = iota
	ActiveTypeLongWord
	ActiveTypePicture
	ActiveTypeVideo
)

// Active。。。
type Active struct {
	IDBase     `json:"id_base,omitempty"`
	CreatorID  uint64 `json:"creator_id,omitempty"`
	ActiveType int    `json:"active_type,omitempty"`
	Content    []byte `json:"content,omitempty"`
	Name       string `json:"name,omitempty"`
	Tags       string `json:"tags,omitempty"`
}

func (a Active) TableNamse() string {
	return "active"
}

func (a *Active) Create() error {
	if err := database.Create(a).Error; err != nil {
		log.Errorf("create new active [%s] failed : [%s]", a.Name, err.Error())
		return fmt.Errorf("create new active [%s] failed ", a.Name)
	}
	return nil
}

func (a *Active) UpdateName() error {
	if err := database.Model(a).Update("name", a.Name).Error; err != nil {
		log.Errorf("update active [%d] failed : [%s]", a.ID, err.Error())
		return fmt.Errorf("update active failed [%s]", err.Error())
	}
	return nil
}

func (a *Active) UpdateContent() error {
	if err := database.Model(a).Update("content", a.Content).Error; err != nil {
		log.Errorf("update active [%d] failed : [%s]", a.ID, err.Error())
		return fmt.Errorf("update active failed [%s]", err.Error())
	}
	return nil
}

func (a *Active) Get() error {
	if err := database.First(a).Error; err != nil {
		return err
	}
	return nil
}

func GetAcviteByCreatorID(creatorId uint64) (*[]Active, error) {
	var ret = new([]Active)
	if err := database.Where("creator_id = ? and delete = 0", creatorId).Find(ret).Error; err != nil {
		log.Errorf("get user [%d] active failed ", creatorId)
		return nil, err
	}

	return ret, nil
}

func GetActiveList(name string) (*[]Active, error) {
	var ret = new([]Active)
	if err := database.Where("name like %?% and delete = 0", name).Find(ret).Error; err != nil {
		log.Errorf("get active like [%s] failed ", name)
		return nil, err
	}
	return ret, nil
}

func GetActiveListByTimeRange(start time.Time, end time.Time) (*[]Active, error) {
	var ret = new([]Active)
	if err := database.Where("created_at < ? and  created_at > ? and delete = 0", end, start).Find(ret).Error; err != nil {
		log.Errorf("get active in range [%s--%s] failed ", start.String(), end.String())
		return nil, err
	}
	return ret, nil
}

func GetActiveListByActiveType(creatorID uint64, activeType uint) (*[]Active, error) {
	var ret = new([]Active)
	if err := database.Where("creator_id = ? and active_type = ? and delete = 0", creatorID, activeType).Find(ret).Error; err != nil {
		log.Errorf("get user [%d] active type [%d] failed ", creatorID, activeType)
		return nil, err
	}
	return ret, nil
}

func (a *Active) Delete() error {
	if err := database.Model(a).Update("deleted", 1); err != nil {
		log.Errorf("update active [%d] deleted failed ", a.IDBase.ID)
		return fmt.Errorf("deleted active [%d] failed ", a.IDBase.ID)
	}
	log.Infof("delete active [%d] success", a.IDBase.ID)
	return nil
}

func ()  {
	
}
