package models

type LikeItem struct {
	IDBase
	UserID    uint64
	GroupID   uint64
	ProjectID uint64
	ItemID    uint64
}

func (l LikeItem) TableName() string {
	return "like_item"
}
