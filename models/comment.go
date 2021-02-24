package models

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// Comment ...
type Comment struct {
	IDBase    `json:"id_base,omitempty"`
	CreatorID uint64 `json:"creator_id,omitempty"`
	ItemID    int    `json:"item_id,omitempty"`
	Content   []byte `json:"content,omitempty"`
	Name      string `json:"name,omitempty"`
	Tags      string `json:"tags,omitempty"`
}

func (c Comment) TableNamse() string {
	return "comment"
}

func (c *Comment) Create() error {
	if err := database.Create(c).Error; err != nil {
		log.Errorf("create new active [%s] failed : [%s]", c.Name, err.Error())
		return fmt.Errorf("create new active [%s] failed ", c.Name)
	}
	return nil
}

func (c *Comment) UpdateName() error {
	if err := database.Model(c).Update("name", c.Name).Error; err != nil {
		log.Errorf("update active [%d] failed : [%s]", c.ID, err.Error())
		return fmt.Errorf("update active failed [%s]", err.Error())
	}
	return nil
}

func (c *Comment) UpdateContent() error {
	if err := database.Model(c).Update("content", c.Content).Error; err != nil {
		log.Errorf("update active [%d] failed : [%s]", c.ID, err.Error())
		return fmt.Errorf("update active failed [%s]", err.Error())
	}
	return nil
}

func (a *Comment) GetComment() error {
	if err := database.First(a).Error; err != nil {
		return err
	}
	return nil
}

func GetCommentByCreatorID(creatorId uint64) (*[]Active, error) {
	var ret = new([]Active)
	if err := database.Where("creator_id = ? and delete = 0", creatorId).Find(ret).Error; err != nil {
		log.Errorf("get user [%d] active failed ", creatorId)
		return nil, err
	}

	return ret, nil
}

func GetCommentList(name string) (*[]Active, error) {
	var ret = new([]Active)
	if err := database.Where("name like %?% and delete = 0", name).Find(ret).Error; err != nil {
		log.Errorf("get active like [%s] failed ", name)
		return nil, err
	}
	return ret, nil
}

func GetCommentListByTimeRange(start time.Time, end time.Time) (*[]Active, error) {
	var ret = new([]Active)
	if err := database.Where("created_at < ? and  created_at > ? and delete = 0", end, start).Find(ret).Error; err != nil {
		log.Errorf("get active in range [%s--%s] failed ", start.String(), end.String())
		return nil, err
	}
	return ret, nil
}

func GetCommentListByItem(creatorID uint64, activeType uint) (*[]Active, error) {
	var ret = new([]Active)
	if err := database.Where("creator_id = ? and active_type = ? and delete = 0", creatorID, activeType).Find(ret).Error; err != nil {
		log.Errorf("get user [%d] active type [%d] failed ", creatorID, activeType)
		return nil, err
	}
	return ret, nil
}

func (c *Comment) Delete() error {
	if err := database.Model(c).Update("deleted", 1); err != nil {
		log.Errorf("update active [%d] deleted failed ", c.IDBase.ID)
		return fmt.Errorf("deleted active [%d] failed ", c.IDBase.ID)
	}
	log.Infof("delete active [%d] success", c.IDBase.ID)
	return nil
}
