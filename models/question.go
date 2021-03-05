package models

type Question struct {
	IDBase
	UserID    uint64
	GroupID   uint64
	ProjectID uint64
}

func (q Question) TableName() string {
	return "question"
}
