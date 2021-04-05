package models

import (
	_ "time"

	api "github.com/grapery/grapery/api"
)

/*
项目，或者说事件流；
1.包含多种内容
2.项目里可以开放问题(暂时可以不做)
*/
type Project struct {
	IDBase
	Name          string          `json:"name,omitempty"`
	Tilte         string          `json:"tilte,omitempty"`
	ShortDesc     string          `json:"short_desc,omitempty"`
	ProjectType   int             `json:"project_type,omitempty"`
	CreatorID     uint64          `json:"creator_id,omitempty"`
	OwnerID       uint64          `json:"owner_id,omitempty"`
	GroupID       uint64          `json:"group_id,omitempty"`
	Description   string          `json:"description,omitempty"`
	Avatar        string          `json:"avatar,omitempty"`
	WatchingCount uint64          `json:"watching_count,omitempty"`
	InvolvedCount uint64          `json:"involved_count,omitempty"`
	Visable       api.VisibleType `json:"visable,omitempty"`
	IsAchieve     bool            `json:"is_achieve,omitempty"`
	IsClose       bool            `json:"is_close,omitempty"`
	IsPrivate     bool            `json:"is_private,omitempty"`
}

func (p Project) TableName() string {
	return "project"
}

func (p *Project) Create() error {
	err := database.Model(p).Create(p).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateDesc() error {
	err := database.Model(p).Update("short_desc", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateTitle() error {
	err := database.Model(p).Update("title", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateAchieve() error {
	err := database.Model(p).Update("is_achieve", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateIsClose() error {
	err := database.Model(p).Update("is_close", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateIsPrivate() error {
	err := database.Model(p).Update("is_private", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) Get() error {
	err := database.First(p).Where("id = ? and deleted = ?", p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) Delete() error {
	err := database.Model(p).Update("deleted", p.Deleted).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func GetProjectListByName(name string, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = database.Model(&Project{}).Where("name like %?% and deleted = ?", name, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// func GetProjectListByTag(tags string, offset, number int) (list []*Project, err error) {
// 	list = make([]*Project, 0)
// 	err = database.Model(&Project{}).Where("name like %?%", tags).Offset(offset).Limit(number).Scan(&list).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return list, nil
// }

func GetProjectListByCreator(creatorID int, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = database.Model(&Project{}).Where("creator_id = ? and deleted = ?", creatorID, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetProjectListByOwner(ownerID int, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = database.Model(&Project{}).Where("owner_id = ? and deleted = ?", ownerID, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetGroupProjectListByName(groupID int, name string, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = database.Model(&Project{}).Where("group_id = ? and name = ? and deleted = ?", groupID, name, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// func GetGroupProjectListByTag(groupID int, tags string, offset, number int) (list []*Project, err error) {
// 	list = make([]*Project, 0)
// 	err = database.Model(&Project{}).Where("group_id = ? and tags = ? and deleted = ?", groupID, tags, 0).
// 		Offset(offset).Limit(number).Scan(&list).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return list, nil
// }

func GeGrouptProjectListByCreator(groupID int, creatorID int, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = database.Model(&Project{}).Where("group_id = ? and creator_id = ? and deleted = ?", groupID, creatorID, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetGroupProjectListByOwner(groupID int, ownerID int, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = database.Model(&Project{}).Where("group_id = ? and owner_id = ? and deleted = ?", groupID, ownerID, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

type ProjectWatcher struct {
	IDBase
	GroupID   uint64 `json:"group_id,omitempty"`
	ProjectID uint64 `json:"project_id,omitempty"`
	UserID    uint64 `json:"user_id,omitempty"`
}

func (p ProjectWatcher) TableName() string {
	return "project_watcher"
}

func GetUserWatchingProjects(userId int64, number, offset int) (list []*Project, err error) {
	plist := make([]*ProjectWatcher, 0)
	err = database.Model(&ProjectWatcher{}).Where("user_id = ? and deleted = ?", userId, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	var pidList = make([]uint64, len(plist))
	for _, val := range plist {
		pidList = append(pidList, val.ProjectID)
	}
	list = make([]*Project, 0)
	err = database.Model(&Project{}).Where("project_id in (?) and deleted = ?", pidList, userId, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func StartWatchingProject(userID, groupID, projectId uint64) error {
	p := &ProjectWatcher{
		UserID:    userID,
		GroupID:   groupID,
		ProjectID: projectId,
	}
	err := database.Model(p).Create(p).Error
	if err != nil {
		return err
	}
	err = database.Model(Project{}).Update("watching_count = ?", "watching_count + 1").
		Where("id = ?", projectId).Where("group_id", groupID).Error
	if err != nil {
		return err
	}
	return nil
}

func StopWatchingProject(userID, groupID, projectId uint64) error {
	p := &ProjectWatcher{
		UserID:    userID,
		GroupID:   groupID,
		ProjectID: projectId,
	}
	err := database.Model(p).Delete(p).Error
	if err != nil {
		return err
	}
	err = database.Model(Project{}).Update("watching_count = ?", "watching_count - 1").
		Where("id = ?", projectId).Where("group_id", groupID).Error
	if err != nil {
		return err
	}
	return nil
}
