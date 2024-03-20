package models

type Licence struct {
	IDBase
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Ref         int64  `json:"ref"`
	Avatar      string `json:"avatar"`
	Creator     int64  `json:"creator"`
	Status      int    `json:"status"`
}

func (licence Licence) TableName() string {
	return "licence"
}
