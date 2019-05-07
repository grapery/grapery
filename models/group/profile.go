package group

import "time"

// GroupProfile ...
type GroupProfile struct {
}

type GroupMemberShip struct {
	GroupID        int64 `json:"group_id,omitempty"`
	MemberShipType int   `json:"member_ship_type,omitempty"`
	// join or leave or member role change
	IsProved  bool  `json:"is_proved,omitempty"`
	UserID    int64 `json:"user_id,omitempty"`
	GroupRole int   `json:"group_role,omitempty"`
	// leader ,follower,coordinater

	CreateAt time.Time `json:"create_at,omitempty"`
	UpdateAt time.Time `json:"update_at,omitempty"`
	Deleted  bool      `json:"deleted,omitempty"`
}
