package models

import "time"

type Event struct {
	Base
}

func (e Event) TableNamse() string {
	return "event"
}

func (e *Event) Create() error {
	if !database.NewRecord(a) {
		database.Create(a)
	}
	return nil
}

func (e *Event) Update() error {
	database.Model(a).Update("password", a.Password)
	return nil
}

func (e *Event) Get() error {
	database.First(a)
	return nil
}

func (e *Event) Delete() error {
	database.Delete(a)
	return nil
}
