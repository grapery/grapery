package models

type Topic struct {
	IDBase
	UserId  uint64 `json:"user_id,omitempty"`
	GroupID uint64 `json:"group_id,omitempty"`
	Title   string `json:"title,omitempty"`
	Desc    string `json:"desc,omitempty"`
	Disable bool   `json:"disable,omitempty"`
	Expired int64  `json:"expired,omitempty"`
}

func (t Topic) TableName() string {
	return "topic"
}
