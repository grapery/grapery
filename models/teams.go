package models

// team才可以实时聊天，但是team中的人员不可以互相加好友
// team属于一个group,team就类似于协程池一样的东西，一起协作做一件事情
type Team struct {
	IDBase
	UserId  uint64 `json:"user_id,omitempty"`
	GroupID uint64 `json:"group_id,omitempty"`
	Title   string `json:"title,omitempty"`
	Desc    string `json:"desc,omitempty"`
	DisAble bool   `json:"disable,omitempty"`
	Expired int64  `json:"expired,omitempty"`
}

func (t Team) TableName() string {
	return "Team"
}
