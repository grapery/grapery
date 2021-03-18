package models

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type Tags struct {
	IDBase
	UserId    uint64 `json:"user_id,omitempty"`
	ItemID    uint64 `json:"item_id,omitempty"`
	ProjectID uint64 `json:"project_id,omitempty"`
	GroupID   uint64 `json:"group_id,omitempty"`
	Title     string `json:"title,omitempty"`
	Desc      string `json:"desc,omitempty"`
	DisAble   bool   `json:"disable,omitempty"`
}

func (t Tags) TableName() string {
	return "tags"
}

func (a *Tags) Create() error {
	if err := database.Create(a).Error; err != nil {
		log.Errorf("create new tag [%d] failed : [%s]", a.ID, err.Error())
		return fmt.Errorf("create new tag [%d] failed: %s", a.ID, err.Error())
	}
	return nil
}

func (a *Tags) Get() error {
	if err := database.First(a).Error; err != nil {
		return err
	}
	return nil
}

func GetTagsByTitleInProject(projectID uint64, title string) ([]*Tags, error) {
	var ret = new([]*Tags)
	if err := database.Where("project_id = ? and delete = 0 and title like %?%", projectID, title).
		Scan(ret).Error; err != nil {
		log.Errorf("get user [%d] tag failed ", projectID)
		return nil, err
	}

	return *ret, nil
}

func GetTagsByTitleInGroup(groupID uint64, title string) ([]*Tags, error) {
	var ret = new([]*Tags)
	if err := database.Where("group_id = ? and delete = 0 and title like %?%", groupID, title).
		Scan(ret).Error; err != nil {
		log.Errorf("get user [%d] tag failed ", groupID)
		return nil, err
	}

	return *ret, nil
}

func GetTagsByGroup(groupID uint64) ([]*Tags, error) {
	var ret = new([]*Tags)
	if err := database.Where("group_id = ? and delete = 0", groupID).
		Scan(ret).Error; err != nil {
		log.Errorf("get user [%d] tag failed ", groupID)
		return nil, err
	}

	return *ret, nil
}

func GetTagsByProject(projectID uint64) ([]*Tags, error) {
	var ret = new([]*Tags)
	if err := database.Where("project_id = ? and delete = 0", projectID).
		Scan(ret).Error; err != nil {
		log.Errorf("get user [%d] tag failed ", projectID)
		return nil, err
	}

	return *ret, nil
}

func (a *Tags) Delete() error {
	if err := database.Model(a).Update("deleted", 1); err != nil {
		log.Errorf("update tag [%d] deleted failed ", a.ID)
		return fmt.Errorf("deleted tag [%d] failed ", a.ID)
	}
	log.Infof("delete active [%d] success", a.ID)
	return nil
}
