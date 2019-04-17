package user

import (
	_ "database/sql"
	"encoding/json"
	"time"
)

type User struct {
	UserID   int64
	Name     string
	NickName string
	UserType int
	Email    string
	Bio      string
	Location string

	//
	AvatarURL string
	URL       string
	CreateAt  time.Time
	UpdateAt  time.Time
	Deleted   bool
}

func NewUser() *User {
	return &User{}
}

func (u *User) String() string {
	data, _ := json.Marshal(u)
	return string(data)
}

func GetUser(UID int64) *User {
	return nil
}
