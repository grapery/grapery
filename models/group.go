package models

import "time"

// Group ...
type Group struct {
	Base
	GroupName      string `json:"group_name,omitempty"`
	GroupTitle     string `json:"group_title,omitempty"`
	GroupShortDesc string `json:"group_short_desc,omitempty"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	GroupType      string `json:"group_type,omitempty"`
	Members        int    `json:"members,omitempty"`
	CreatorID      int64  `json:"creator_id,omitempty"`
	IsPrivate      bool   `json:"is_private,omitempty"`
}

func (g Group) TableNamse() string {
	return "group"
}

func (g *Group) Create() error {
	if !database.NewRecord(a) {
		database.Create(a)
	}
	return nil
}

func (g *Group) Update() error {
	database.Model(a).Update("password", a.Password)
	return nil
}

func (g *Group) Get() error {
	database.First(a)
	return nil
}

func (g *Group) Delete() error {
	database.Delete(a)
	return nil
}
