package models

type Question struct {
	IDBase
	UserID      uint64 `json:"user_id,omitempty"`
	GroupID     uint64 `json:"group_id,omitempty"`
	ProjectID   uint64 `json:"project_id,omitempty"`
	Tital       string `json:"tital,omitempty"`
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags,omitempty"`
}

func (q Question) TableName() string {
	return "question"
}
