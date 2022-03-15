package models

type Disscuss struct {
	IDBase
	Creator   uint64 `json:"creator,omitempty"`
	ProjectID uint64 `json:"project_id,omitempty"`
	GroupID   uint64 `json:"group_id,omitempty"`
	Name      string
	Desc      string
}
