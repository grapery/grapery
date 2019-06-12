package models

import "time"

const (
	ProjectTypePicture = iota
	ProjectTypeText
	ProjectTypeEvent
	ProjectTypeRecord
)

type Project struct {
	Name        string `json:"name,omitempty"`
	Tilte       string `json:"tilte,omitempty"`
	Description string `json:"description,omitempty"`
	ShortDesc   string `json:"short_desc,omitempty"`
	ProjectType int    `json:"project_type,omitempty"`
	// 图片，文字,事件．．．．
	IsPrivate bool `json:"is_private,omitempty"`
}

func (p Project) TableNamse() string {
	return "project"
}

func (p *Project) Create() error {
	if !database.NewRecord(a) {
		database.Create(a)
	}
	return nil
}

func (p *Project) Update() error {
	database.Model(a).Update("password", a.Password)
	return nil
}

func (p *Project) Get() error {
	database.First(a)
	return nil
}

func (p *Project) Delete() error {
	database.Delete(a)
	return nil
}
