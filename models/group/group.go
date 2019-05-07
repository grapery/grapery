package group

import "time"

// Group ...
type Group struct {
	GroupID        int64     `json:"group_id,omitempty"`
	GroupName      string    `json:"group_name,omitempty"`
	GroupTitle     string    `json:"group_title,omitempty"`
	GroupShortDesc string    `json:"group_short_desc,omitempty"`
	AvatarURL      string    `json:"avatar_url,omitempty"`
	GroupType      string    `json:"group_type,omitempty"`
	Members        int       `json:"members,omitempty"`
	CreatorID      int64     `json:"creator_id,omitempty"`
	IsPrivate      bool      `json:"is_private,omitempty"`
	CreateAt       time.Time `json:"create_at,omitempty"`
	UpdateAt       time.Time `json:"update_at,omitempty"`
	Deleted        bool      `json:"deleted,omitempty"`
}
