package auth

import "time"

type Auth struct {
	ID       int64  `json:"id,omitempty"`
	UserID   int64  `json:"user_id,omitempty"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Password string `json:"password,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	AuthType string `json:"auth_type,omitempty"`

	CreateAt time.Time `json:"create_at,omitempty"`
	UpdateAt time.Time `json:"update_at,omitempty"`
	Deleted  bool      `json:"deleted,omitempty"`
}
