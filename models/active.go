package models

import (
	_ "time"
)

type Active struct {
	Base
	Name string
	Tags string
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
	database.Model(a).Update("name", a.Name)
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
