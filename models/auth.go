package models

import (
	_ "github.com/jinzhu/gorm"
)

type Auth struct {
	Base
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Password string `json:"password,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	AuthType string `json:"auth_type,omitempty"`
}

func (a Auth) TableNamse() string {
	return "auth"
}

func (a *Auth) Create() error {
	if !database.NewRecord(a) {
		database.Create(a)
	}
	return nil
}

func (a *Auth) Update() error {
	database.Model(a).Update("password", a.Password)
	return nil
}

func (a *Auth) Get() error {
	database.First(a)
	return nil
}

func (a *Auth) Delete() error {
	database.Delete(a)
	return nil
}
