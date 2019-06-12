package models

import (
	"time"
)

type Active struct {
	Base
	CreatedAt time.Time
	DeletedAt time.Time
	Deleted   bool
}

func (a Active) TableNamse() string {
	return "active"
}

func (a *Active) Create() error {
	if !database.NewRecord(a) {
		database.Create(a)
	}
	return nil
}

func (a *Active) Update() error {
	database.Model(a).Update("password", a.Password)
	return nil
}

func (a *Active) Get() error {
	database.First(a)
	return nil
}

func (a *Active) Delete() error {
	database.Delete(a)
	return nil
}
