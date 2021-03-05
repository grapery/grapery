package models

import "github.com/grapery/grapery/api"

type ShareItem struct {
	IDBase
	UserID      uint64          `json:"user_id,omitempty"`
	GroupID     uint64          `json:"group_id,omitempty"`
	ProjectID   uint64          `json:"project_id,omitempty"`
	ItemID      uint64          `json:"item_id,omitempty"`
	ToGroupID   uint64          `json:"to_group_id,omitempty"`
	ToProjectID uint64          `json:"to_project_id,omitempty"`
	ToItemID    uint64          `json:"to_item_id,omitempty"`
	Description string          `json:"description,omitempty"`
	Visable     api.VisibleType `json:"visable,omitempty"`
}

func (s ShareItem) TableName() string {
	return "share"
}
