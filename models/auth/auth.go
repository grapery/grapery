package auth

import "time"

type Auth struct {
	Username  string
	Password  string
	Email     string
	Phone     string
	Area      int
	CreatedAt time.Time
	Deleted   bool
	DeletedAt time.Time
	ThridPart string
}
