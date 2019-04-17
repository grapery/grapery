package user

import "time"

type Profile struct {
	PID       int64
	UserID    int64
	Followers int64
	Following int64

	CreateAt time.Time
	UpdateAt time.Time
	Deleted  bool
}

func NewProfile() *Profile {
	return &Profile{}
}
