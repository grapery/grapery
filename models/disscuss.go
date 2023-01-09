package models

type Disscuss struct {
	IDBase
	Creator   uint64 `json:"creator,omitempty"`
	ProjectID uint64 `json:"project_id,omitempty"`
	GroupID   uint64 `json:"group_id,omitempty"`
	Title     string
	Status    int
	Desc      string
}

func (d Disscuss) TableName() string {
	return "disscuss"
}

func GetDisscussById(did int) (*Disscuss, error) {
	return nil, nil
}

func GetDisscussByCreator(creator string, pageSize, pageNum int) ([]*Disscuss, error) {
	result := make([]*Disscuss, 0)
	err := DataBase().Model(Disscuss{}).
		Where("creator = ?", creator).
		Find(&result).Offset(int(pageNum-1) * pageSize).Limit(pageSize).
		Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetDisscussByProjectID(projectID int64, pageSize, pageNum int) ([]*Disscuss, error) {
	result := make([]*Disscuss, 0)
	err := DataBase().Model(Disscuss{}).
		Where("project_id = ?", projectID).
		Find(&result).Offset(int(projectID-1) * pageSize).Limit(pageSize).
		Error
	if err != nil {
		return nil, err
	}
	return result, nil
}
