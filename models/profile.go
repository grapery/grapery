package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

type UserProfile struct {
	Base
	UserID    int64 `json:"user_id,omitempty"`
	Followers int64 `json:"followers,omitempty"`
	Following int64 `json:"following,omitempty"`
	//
	Emotion   int    `json:"emotion,omitempty"`
	ShortDesc string `json:"short_desc,omitempty"`
	//
}

func (p UserProfile) TableNamse() string {
	return "user_profile"
}

func (p *UserProfile) Create() error {
	if !database.NewRecord(p) {
		database.Create(p)
	}
	return nil
}

func (p *UserProfile) Update() error {
	database.Model(p).Update("emotion", p.Emotion)
	return nil
}

func (p *UserProfile) Get() error {
	database.First(p)
	return nil
}

func (p *UserProfile) Delete() error {
	database.Delete(p)
	return nil
}

type GroupProfile struct {
	IDBase
	CreatorID   uint64
	Name        string
	Description string
	Avatar      string
	Tags        []string
}

func (g GroupProfile) TableNamse() string {
	return "group_profile"
}

func (g *GroupProfile) Create() error {
	if err := database.Where("creatot_id = ? and deleted = ?", g.CreatorID, 0).Find(g).Error; err != nil {
		log.Errorf("create group [%s] profile failed : ", g.Name)
		return err
	}
	var ret *gorm.DB
	if g.IDBase.ID != 0 {
		ret = database.Create(g)
	} else {
		log.Errorf("group [%s] profile is exist : ", g.ID)
		return fmt.Errorf("group [%d] profile is exist : ", g.ID)
	}
	if ret.Error != nil {
		log.Errorf("create group [%s] profile failed : [%s]", g.Name, ret.Error.Error())
		return fmt.Errorf("create group [%s] profile failed : ", g.Name)
	}
	return nil
}

func (g *GroupProfile) Update() error {
	database.Model(g).Update("emotion", g.Name)
	return nil
}

func (g *GroupProfile) Get() error {
	database.First(g)
	return nil
}

func (g *GroupProfile) Delete() error {
	database.Delete(g)
	return nil
}
