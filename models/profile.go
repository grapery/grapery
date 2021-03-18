package models

import (
	"fmt"

	"github.com/grapery/grapery/api"
	log "github.com/sirupsen/logrus"
)

/*
用户的profile文件
*/
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

func (p UserProfile) TableName() string {
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

/*
组织的profile文件
*/
type GroupProfile struct {
	IDBase
	GroupID     uint64          `json:"group_id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Avatar      string          `json:"avatar,omitempty"`
	Visable     api.VisibleType `json:"visable,omitempty"`
}

func (g GroupProfile) TableName() string {
	return "group_profile"
}

func (g *GroupProfile) Create() error {
	if err := database.Where("group_id = ? and deleted = ?", g.GroupID, 0).Find(g).Error; err != nil {
		log.Errorf("create group [%d] profile failed : %s", g.GroupID, err.Error())
		return err
	}
	var err error
	if g.ID != 0 {
		err = database.Create(g).Error
	} else {
		log.Errorf("group [%s] profile is exist : ", g.ID)
		return fmt.Errorf("group [%d] profile is exist : ", g.ID)
	}
	if err != nil {
		log.Errorf("create group [%d] profile failed : [%s]", g.GroupID, err.Error())
		return fmt.Errorf("create group [%d] profile failed : %s", g.GroupID, err.Error())
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

/*
项目中的组织的profile文件
*/
type ProjectFile struct {
	IDBase
	GroupID     uint64          `json:"group_id,omitempty"`
	ProjectID   uint64          `json:"project_id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Avatar      string          `json:"avatar,omitempty"`
	Visable     api.VisibleType `json:"visable,omitempty"`
}

func (p ProjectFile) TableName() string {
	return "project_profile"
}
