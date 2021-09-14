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

func (t *Team) CreateTeam() error {
	return nil
}

func (t *Team) UpdateTeam() error {
	return nil
}

func (t *Team) DeleteTeam() error {
	return nil
}

func (t *Team) GetTeam() error {
	return nil
}

func GetTeamsByName(name string) ([]*Team, error) {
	return nil, nil
}

func GetTeamsByCreator(userId uint64) ([]*Team, error) {
	return nil, nil
}

type TeamMemeber struct {
}

func (t TeamMemeber) TableName() string {
	return "Team"
}

func (t *TeamMemeber) AddTeamMember() error {
	return nil
}

func (t *TeamMemeber) DeleteTeamMember() error {
	return nil
}

func (t *TeamMemeber) GetTeamMembers() error {
	return nil
}
