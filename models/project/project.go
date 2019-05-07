package project

import "time"

const (
	ProjectTypePicture = iota
	ProjectTypeText
	ProjectTypeEvent
	ProjectTypeRecord
)

type Project struct {
	ProjectID   int64  `json:"project_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Tilte       string `json:"tilte,omitempty"`
	Description string `json:"description,omitempty"`
	ShortDesc   string `json:"short_desc,omitempty"`
	ProjectType int    `json:"project_type,omitempty"`
	// 图片，文字,事件．．．．
	IsPrivate bool `json:"is_private,omitempty"`

	CreateAt time.Time `json:"create_at,omitempty"`
	UpdateAt time.Time `json:"update_at,omitempty"`
	Deleted  bool      `json:"deleted,omitempty"`
}
