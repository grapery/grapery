package models

type DiscussStatus int

const (
	DiscussStatusClosed DiscussStatus = iota + 1
	DiscussStatusOpen
	DiscussStatusPending
	DiscussStatusArchived
)

type Disscuss struct {
	IDBase
	Creator      int64 `json:"creator,omitempty"`
	StoryID      int64 `json:"story_id,omitempty"`
	GroupID      int64 `json:"group_id,omitempty"`
	Title        string
	Status       DiscussStatus
	Desc         string
	TotalUser    int64 `json:"total_user,omitempty"`
	TotalMessage int64 `json:"total_message,omitempty"`
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
		Offset(int(pageNum-1) * pageSize).
		Limit(pageSize).
		Scan(&result).
		Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetDisscussByStoryID(storyID int64, pageSize, pageNum int) ([]*Disscuss, error) {
	result := make([]*Disscuss, 0)
	err := DataBase().Model(Disscuss{}).
		Where("story_id = ?", storyID).
		Offset(int(pageNum-1) * pageSize).
		Limit(pageSize).
		Scan(&result).
		Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func SearchDisscuss(keyword string, pageSize, pageNum int) ([]*Disscuss, error) {
	result := make([]*Disscuss, 0)
	err := DataBase().Model(Disscuss{}).
		Where("title like ?", "%"+keyword+"%").
		Offset(int(pageNum-1) * pageSize).
		Limit(pageSize).
		Scan(&result).
		Error
	if err != nil {
		return nil, err
	}
	return result, nil
}
