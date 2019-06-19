package models

import (
	_ "database/sql"
	_ "encoding/json"
	_ "time"
)

type User struct {
	Base
	Name     string `json:"name,omitempty"`
	NickName string `json:"nick_name,omitempty"`
	UserType int    `json:"user_type,omitempty"`
	Email    string `json:"email,omitempty"`
	Bio      string `json:"bio,omitempty"`
	Location string `json:"location,omitempty"`

	AvatarURL string `json:"avatar_url,omitempty"`
	URL       string `json:"url,omitempty"`
}

func (u User) TableNamse() string {
	return "user"
}

func (u *User) Create() error {
	if !database.NewRecord(u) {
		database.Create(u)
	}
	return nil
}

func (u *User) Update() error {
	database.Model(u).Update("nick_name", u.NickName)
	return nil
}

func (u *User) Get() error {
	database.First(u)
	return nil
}

func (u *User) Delete() error {
	database.Delete(u)
	return nil
}
