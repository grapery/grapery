package models

import (
	_ "time"
)

/*
项目，或者说事件流；
1.包含多种内容
2.项目里可以开放问题(暂时可以不做)
*/
type Project struct {
	IDBase
	Name        string `json:"name,omitempty"`
	Tilte       string `json:"tilte,omitempty"`
	ShortDesc   string `json:"short_desc,omitempty"`
	ProjectType int    `json:"project_type,omitempty"`
	CreatorID   uint64 `json:"creator_id,omitempty"`
	OwnerID     uint64 `json:"owner_id,omitempty"`
	GroupID     uint64 `json:"group_id,omitempty"`
	IsAchieve   bool   `json:"is_achieve,omitempty"`
	IsClose     bool   `json:"is_close,omitempty"`
	IsPrivate   bool   `json:"is_private,omitempty"`
}

func (p Project) TableNamse() string {
	return "project"
}

func (p *Project) Create() error {
	if !database.NewRecord(p) {
		database.Create(p)
	}
	return nil
}

func (p *Project) Update() error {
	database.Model(p).Update("short_desc", p.ShortDesc)
	return nil
}

func (p *Project) Get() error {
	database.First(p)
	return nil
}

func (p *Project) Delete() error {
	database.Delete(p)
	return nil
}
