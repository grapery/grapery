package group

import (
	"time"
)

type Group struct {
	GroupName string
	GID       string

	CreatedAt time.Time
	DeletedAt time.Time
	Deleted   bool
}

func NewGroup() *Group {
	return &Group{}
}
