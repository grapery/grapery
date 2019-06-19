package models

import (
	_ "time"
)

type Event struct {
	Base
	Etype    int    `json:"etype,omitempty"`
	Describe string `json:"describe,omitempty"`
}

func (e Event) TableNamse() string {
	return "event"
}

func (e *Event) Create() error {
	if !database.NewRecord(e) {
		database.Create(e)
	}
	return nil
}

func (e *Event) Update() error {
	database.Model(e).Update("etype", e.Etype)
	return nil
}

func (e *Event) Get() error {
	database.First(e)
	return nil
}

func (e *Event) Delete() error {
	database.Delete(e)
	return nil
}
