package models

import "errors"

// team才可以实时聊天，但是team中的人员不可以互相加好友
// team属于一个group,team就类似于协程池一样的东西，一起协作做一件事情
type Team struct {
	IDBase
	UserId   uint64 `json:"user_id,omitempty"`
	GroupID  uint64 `json:"group_id,omitempty"`
	Title    string `json:"title,omitempty"`
	Desc     string `json:"desc,omitempty"`
	DisAble  bool   `json:"disable,omitempty"`
	IsClosed bool   `json:"is_closed"`
	Dismiss  bool   `json:"dismiss,omitempty"`
}

func (t Team) TableName() string {
	return "Team"
}

func (t *Team) CreateTeam() error {
	err := DataBase().Model(t).Create(t).Error
	if err != nil {
		return err
	}
	return nil
}

func (t *Team) UpdateTeam() error {
	err := DataBase().Model(t).
		Update("desc", t.Desc).
		Update("title", t.Title).
		Where("group_id = ? and id = ? and deleted = ?", t.GroupID, t.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (t *Team) DismissTeam() error {
	err := DataBase().Model(t).
		Update("dismiss", t.Dismiss).
		Where("group_id = ? and id = ? and deleted = ?", t.GroupID, t.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (t *Team) DeleteTeam() error {
	err := DataBase().Model(t).
		Update("deleted", 1).
		Where("group_id = ? and id = ? and deleted = ?", t.GroupID, t.ID, 1).Error
	if err != nil {
		return err
	}
	return nil
}

func (t *Team) GetTeam() error {
	err := DataBase().First(t).Where("id = ? and deleted = ?", t.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func GetTeamsByName(title string, groupId int64) (list []*Team, err error) {
	list = make([]*Team, 0)
	err = DataBase().Model(&Team{}).
		Where("name like %?% and group_id = ? and deleted = ?", title, groupId, 0).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetTeamsByCreator(userId int64) (list []*Team, err error) {
	list = make([]*Team, 0)
	err = DataBase().Model(&Team{}).
		Where("user_id = ? and deleted = ?", userId, 0).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetTeamsByCreatorAndGroup(userId int64, groupId int64) (list []*Team, err error) {
	list = make([]*Team, 0)
	err = DataBase().Model(&Team{}).
		Where("user_id = ? and group_id = ? and deleted = ?", userId, groupId, 0).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetTeamsByGroup(groupId int64) (list []*Team, err error) {
	list = make([]*Team, 0)
	err = DataBase().Model(&Team{}).
		Where("and group_id = ? and deleted = ?", groupId, 0).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetTeamsByMultiIds(teamIDs []int64) (list []*Team, err error) {
	list = make([]*Team, 0)
	err = DataBase().Model(&Team{}).
		Where("and id in (?) and deleted = ?", teamIDs, 0).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

type TeamMemeber struct {
	IDBase
	TeamID      uint64 `json:"team_id,omitempty"`
	UserId      uint64 `json:"user_id,omitempty"`
	GroupID     uint64 `json:"group_id,omitempty"`
	NickName    string `json:"nick_name,omitempty"`
	Description string `json:"description,omitempty"`
}

func (t TeamMemeber) TableName() string {
	return "team_member"
}

func (t *TeamMemeber) AddTeamMember() error {
	err := DataBase().Model(t).Create(t).Error
	if err != nil {
		return err
	}
	return nil
}

func (t *TeamMemeber) UpdateTeamMemberInfo() error {
	err := DataBase().Model(t).
		Update("nick_name", t.NickName).
		Update("description", t.Description).
		Where("group_id = ? and team_id = ? and user_id = ? and deleted = ?",
			t.GroupID, t.TeamID, t.UserId, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (t *TeamMemeber) DeleteTeamMember() error {
	err := DataBase().Model(t).
		Update("deleted", 1).
		Where("group_id = ? and team_id = ? and user_id = ? and deleted = ?",
			t.GroupID, t.TeamID, t.UserId, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func GetTeamMembers(groupID int64, teamID int64) (list []*User, err error) {
	tlist := make([]*TeamMemeber, 0)
	err = DataBase().Model(&TeamMemeber{}).
		Where("group_id = ? and team_id = ? and deleted = ?", groupID, teamID, 0).
		Scan(&tlist).Error
	if err != nil {
		return nil, err
	}
	userIds := make([]int64, len(tlist), len(tlist))
	for idx := range tlist {
		userIds = append(userIds, int64(tlist[idx].UserId))
	}
	list, err = GetUsersByIds(userIds)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetUserJoinedTeamIDInGroup(userId, groupId int64) (ids []int64, err error) {
	tlist := make([]*TeamMemeber, 0)
	err = DataBase().Model(&TeamMemeber{}).
		Where("user_id = ? and group_id = ? and deleted = ?", userId, groupId, 0).
		Scan(&tlist).Error
	if err != nil {
		return nil, err
	}
	ids = make([]int64, 0, len(tlist))
	for idx := range tlist {
		ids = append(ids, int64(tlist[idx].ID))
	}
	return ids, nil
}

func GetUserJoinedTeamInGroup(userId, groupId int64) (list []*Team, err error) {
	ids, err := GetUserJoinedTeamIDInGroup(userId, groupId)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errors.New("not joined any team")
	}
	list, err = GetTeamsByMultiIds(ids)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func BatchLeaveTeams(userId int64, ids []int64) error {
	err := DataBase().Model(&TeamMemeber{}).
		Update("deleted", 1).
		Where(" team_id in (?) and user_id = ? and deleted = ?",
			ids, userId, 0).Error
	if err != nil {
		return err
	}
	return nil
}
