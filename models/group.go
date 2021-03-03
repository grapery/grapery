package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// Group ...
type Group struct {
	IDBase
	Name      string `json:"name,omitempty"`
	ShortDesc string `json:"short_desc,omitempty"`
	Avatar    string `son:"avatar,omitempty"`
	Gtype     string `json:"gtype,omitempty"`
	CreatorID int64  `json:"creator_id,omitempty"`
	IsPrivate bool   `json:"is_private,omitempty"`
}

func (g Group) TableNamse() string {
	return "group"
}

func (g *Group) Create() error {
	database.Begin()
	database.Where("name = ? and  creator_id = ? and deleted = ?", g.Name, g.CreatorID, 0).Find(g)
	var ret *gorm.DB
	if g.IDBase.ID != 0 {
		ret = database.Create(g)
	} else {
		database.Rollback()
		log.Errorf("group [%s] is exist : ", g.IDBase.ID)
		return fmt.Errorf("group [%s] is exist", g.Name)
	}
	if ret.Error != nil {
		database.Rollback()
		log.Errorf("create group [%s] failed [%s] ", g.Name, ret.Error)
		return fmt.Errorf("create group failed")
	}
	database.Commit()
	return nil
}

func (g *Group) UpdateDesc() error {
	if err := database.Model(g).Update("short_desc", g.ShortDesc).Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateGroupType() error {
	if err := database.Model(g).Update("short_desc", g.ShortDesc).Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateAvatar() error {
	if err := database.Model(g).Update("avatar", g.ShortDesc).Error; err != nil {
		log.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) GetByName() error {
	if err := database.Where("name = ? and creator_id = ? and deleted = ?", g.Name, g.CreatorID, 0).Find(g).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) GetByID() error {
	if err := database.Where("id = ? ", g.ID).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) Delete() error {
	if err := database.Model(g).Update("deleted", 1); err != nil {
		log.Errorf("update group [%d] deleted failed ", g.IDBase.ID)
		return fmt.Errorf("deleted group [%d] failed ", g.IDBase.ID)
	}
	return nil
}
