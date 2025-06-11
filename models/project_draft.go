package models

import (
	_ "time"

	"gorm.io/gorm"

	"context"

	api "github.com/grapery/common-protoc/gen"
)

type ProjectType int

const (
	ComicStory ProjectType = iota + 1
)

/*
项目，或者说事件流；
1.包含多种内容
2.项目里可以开放问题(暂时可以不做)
*/
// Project 项目/事件流
type Project struct {
	IDBase
	Name        string `gorm:"column:name" json:"name,omitempty"`                 // 项目名
	Tilte       string `gorm:"column:tilte" json:"tilte,omitempty"`               // 标题（疑似拼写错误，建议Title）
	ShortDesc   string `gorm:"column:short_desc" json:"short_desc,omitempty"`     // 简短描述
	ProjectType int    `gorm:"column:project_type" json:"project_type,omitempty"` // 项目类型
	CreatorID   int64  `gorm:"column:creator_id" json:"creator_id,omitempty"`     // 创建者ID
	OwnerID     int64  `gorm:"column:owner_id" json:"owner_id,omitempty"`         // 拥有者ID
	GroupID     int64  `gorm:"column:group_id" json:"group_id,omitempty"`         // 群组ID
	ProjectSetting
}

// ProjectSetting 项目扩展设置
type ProjectSetting struct {
	Description string        `gorm:"column:description" json:"description,omitempty"` // 描述
	Avatar      string        `gorm:"column:avatar" json:"avatar,omitempty"`           // 头像
	Visable     api.ScopeType `gorm:"column:visable" json:"visable,omitempty"`         // 可见性
	IsAchieve   bool          `gorm:"column:is_achieve" json:"is_achieve,omitempty"`   // 是否达成
	IsClose     bool          `gorm:"column:is_close" json:"is_close,omitempty"`       // 是否关闭
	IsPrivate   bool          `gorm:"column:is_private" json:"is_private,omitempty"`   // 是否私有
}

func (p Project) TableName() string {
	return "project"
}

func (p *Project) Create() error {
	err := DataBase().Model(p).
		Create(p).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateDesc() error {
	err := DataBase().Model(p).
		Update("short_desc", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateTitle() error {
	err := DataBase().Model(p).
		Update("title", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateAchieve() error {
	err := DataBase().Model(p).
		Update("is_achieve", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateIsClose() error {
	err := DataBase().Model(p).
		Update("is_close", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateIsPrivate() error {
	err := DataBase().Model(p).
		Update("is_private", p.ShortDesc).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) Get() error {
	err := DataBase().First(p).
		Where("id = ? and deleted = ?", p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) GetProfile() error {
	err := DataBase().First(p).
		Select(
			"description",
			"avatar",
			"watching_count",
			"involved_count",
			"visable",
			"is_achieve",
			"is_close",
			"is_private").
		Where("id = ? and deleted = ?", p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) UpdateProfile() error {
	err := DataBase().Model(p).
		Updates(map[string]interface{}{
			"description": p.Description,
			"avatar":      p.Avatar,
			"visable":     p.Visable,
			"is_achieve":  p.IsAchieve,
			"is_close":    p.IsClose,
			"is_private":  p.IsPrivate,
		}).
		Where("id = ? and deleted = ?", p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) IncreaseWatcher() error {
	err := DataBase().Model(p).
		UpdateColumn("watching_count", gorm.Expr("watching_count + ?", 1)).
		Where("id = ? and deleted = ?", p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) DecreaseWatcher() error {
	err := DataBase().Model(p).
		UpdateColumn("watching_count", gorm.Expr("watching_count - ?", 1)).
		Where("id = ? and deleted = ?", p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func (p *Project) Delete() error {
	err := DataBase().Model(p).Update("deleted", p.Deleted).
		Where("group_id = ? and id = ? and deleted = ?", p.GroupID, p.ID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func GetProjectListByName(name string, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).Where("name like %?% and deleted = ?", name, 0).
		Offset(offset).Limit(number).Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// func GetProjectListByTag(tags string, offset, number int) (list []*Project, err error) {
// 	list = make([]*Project, 0)
// 	err = DataBase().Model(&Project{}).Where("name like %?%", tags).Offset(offset).Limit(number).Scan(&list).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return list, nil
// }

func GetProjectListByCreator(creatorID int, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).
		Where("creator_id = ? and deleted = ?", creatorID, 0).
		Offset(offset).
		Limit(number).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetProjectListByOwner(ownerID int, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).
		Where("owner_id = ? and deleted = ?", ownerID, 0).
		Offset(offset).
		Limit(number).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetGroupProjectListByName(groupID int, name string, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).
		Where("group_id = ? and name = ? and deleted = ?", groupID, name, 0).
		Offset(offset).
		Limit(number).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetGroupProjects(groupID int64, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).
		Where("group_id = ?  and deleted = ?", groupID, 0).
		Offset(offset).
		Limit(number).
		Scan(&list).
		Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetAllProjects(offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).
		Where("deleted = ?", 0).
		Offset(offset).
		Limit(number).
		Order("update_at").
		Scan(&list).
		Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// func GetGroupProjectListByTag(groupID int, tags string, offset, number int) (list []*Project, err error) {
// 	list = make([]*Project, 0)
// 	err = DataBase().Model(&Project{}).Where("group_id = ? and tags = ? and deleted = ?", groupID, tags, 0).
// 		Offset(offset).Limit(number).Scan(&list).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return list, nil
// }

func GeGroupProjectListByCreator(groupID int, creatorID int, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).
		Where("group_id = ? and creator_id = ? and deleted = ?", groupID, creatorID, 0).
		Offset(offset).
		Limit(number).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetGroupProjectListByOwner(groupID int, ownerID int, offset, number int) (list []*Project, err error) {
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).
		Where("group_id = ? and owner_id = ? and deleted = ?", groupID, ownerID, 0).
		Offset(offset).
		Limit(number).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

type ProjectWatcher struct {
	IDBase
	GroupID   int64 `json:"group_id,omitempty"`
	ProjectID int64 `json:"project_id,omitempty"`
	UserID    int64 `json:"user_id,omitempty"`
}

func (p ProjectWatcher) TableName() string {
	return "project_watcher"
}

func GetUserWatchingProjects(userId int64, number, offset int) (list []*Project, err error) {
	plist := make([]*ProjectWatcher, 0)
	err = DataBase().Model(&ProjectWatcher{}).
		Where("user_id = ? and deleted = ?", userId, 0).
		Offset(offset).
		Limit(number).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	var pidList = make([]int64, len(plist))
	for _, val := range plist {
		pidList = append(pidList, val.ProjectID)
	}
	list = make([]*Project, 0)
	err = DataBase().Model(&Project{}).
		Where("project_id in (?) and deleted = ?", pidList, userId, 0).
		Offset(offset).
		Limit(number).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func StartWatchingProject(userID, groupID, projectId int64) error {
	p := &ProjectWatcher{
		UserID:    userID,
		GroupID:   groupID,
		ProjectID: projectId,
	}
	err := DataBase().Model(p).Create(p).Error
	if err != nil {
		return err
	}
	err = DataBase().Model(Project{}).
		UpdateColumn("watching_count", gorm.Expr("watching_count + ?", 1)).
		Where("id = ?", projectId).
		Where("group_id", groupID).Error
	if err != nil {
		return err
	}
	return nil
}

func StopWatchingProject(userID, groupID, projectId int64) error {
	p := &ProjectWatcher{
		UserID:    userID,
		GroupID:   groupID,
		ProjectID: projectId,
	}
	err := DataBase().Model(p).Delete(p).Error
	if err != nil {
		return err
	}
	err = DataBase().Model(Project{}).
		UpdateColumn("watching_count", gorm.Expr("watching_count - ?", 1)).
		Where("id = ?", projectId).
		Where("group_id", groupID).Error
	if err != nil {
		return err
	}
	return nil
}

type ProjectProfile struct {
	IDBase
	ProjectID int64 `json:"project_id,omitempty"`
}

// 新增：分页获取Project列表
func GetProjectList(ctx context.Context, offset, limit int) ([]*Project, error) {
	var projects []*Project
	err := DataBase().Model(&Project{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&projects).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return projects, nil
}

// 新增：通过Name唯一查询
func GetProjectByNameUnique(ctx context.Context, name string) (*Project, error) {
	project := &Project{}
	err := DataBase().Model(project).
		WithContext(ctx).
		Where("name = ?", name).
		First(project).Error
	if err != nil {
		return nil, err
	}
	return project, nil
}
