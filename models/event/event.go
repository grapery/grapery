package event

import "time"

type Event struct {
	ID int64

	CreateAt time.Time `json:"create_at,omitempty"`
	UpdateAt time.Time `json:"update_at,omitempty"`
	Deleted  bool      `json:"deleted,omitempty"`
}
