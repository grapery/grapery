package models

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/errors"
)

type GroupType int

// Group 代表一个用户组/社群
// status: 1-有效, 0-无效
// is_default: 是否为默认组
// visable_type: 可见性
// ...
type Group struct {
	IDBase
	Name        string        `gorm:"column:name;index" json:"name,omitempty"`           // 组名
	ShortDesc   string        `gorm:"column:short_desc" json:"short_desc,omitempty"`     // 简短描述
	Gtype       string        `gorm:"column:gtype" json:"gtype,omitempty"`               // 组类型
	CreatorID   int64         `gorm:"column:creator_id" json:"creator_id,omitempty"`     // 创建者ID
	OwnerID     int64         `gorm:"column:owner_id" json:"owner_id,omitempty"`         // 拥有者ID
	Members     int64         `gorm:"column:members" json:"members,omitempty"`           // 成员数
	VisableType api.ScopeType `gorm:"column:visable_type" json:"visable_type,omitempty"` // 可见性
	Description string        `gorm:"column:description" json:"description,omitempty"`   // 详细描述
	Avatar      string        `gorm:"column:avatar" json:"avatar,omitempty"`             // 头像
	IsDefault   bool          `gorm:"column:is_default" json:"is_default,omitempty"`     // 是否默认组
	Status      int64         `gorm:"column:status" json:"status,omitempty"`             // 状态
	Tags        string        `gorm:"column:tags" json:"tags,omitempty"`                 // 标签
	Location    string        `gorm:"column:location" json:"location,omitempty"`         // 位置
}

func (g Group) TableName() string {
	return "group"
}

func (g *Group) Create() error {
	if g.Avatar == "" {
		g.Avatar = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/default.png"
	}
	err := DataBase().Table(g.TableName()).
		Where("name = ? and  creator_id = ? and owner_id = ? and deleted = ?",
			g.Name, g.CreatorID, g.CreatorID, 0).
		First(g).
		Error

	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("query group failed: %s", err.Error())
		return err
	}
	if err == gorm.ErrRecordNotFound {
		err = DataBase().Table(g.TableName()).Create(g).Error
		if err != nil {
			log.Errorf("create group [%s] failed: %s", g.Name, err.Error())
			return errors.ErrGroupIsAlreadyExist
		}
	}

	return nil
}

func CreateGroup(g *Group) error {
	err := DataBase().Table(g.TableName()).
		Where("name = ? and  creator_id = ? and owner_id = ? and deleted = ?",
			g.Name, g.CreatorID, g.CreatorID, 0).
		First(g).
		Error
	if err == nil && g.OwnerID != 0 {
		return nil
	}
	if g.Avatar == "" {
		g.Avatar = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/default.png"
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("query group failed: %s", err.Error())
		return err
	}
	if err == gorm.ErrRecordNotFound {
		err = DataBase().Table(g.TableName()).Create(g).Error
		if err != nil {
			log.Errorf("create group [%s] failed: %s", g.Name, err.Error())
			return errors.ErrGroupIsAlreadyExist
		}
	}

	return nil
}

func (g *Group) UpdateAll(groupId int64) error {
	if groupId < 1 {
		return errors.ErrGroupIsNotExist
	}
	needUpdate := make(map[string]interface{})
	if g.ShortDesc != "" {
		needUpdate["short_desc"] = g.ShortDesc
	}
	if g.Gtype != "" {
		needUpdate["gtype"] = g.Gtype
	}
	if g.Avatar != "" {
		needUpdate["avatar"] = g.Avatar
	}
	if g.Name != "" {
		needUpdate["name"] = g.Name
	}
	if g.Description != "" {
		needUpdate["description"] = g.Description
	}
	if err := DataBase().Table(g.TableName()).
		Where("id = ? and deleted = ?", groupId, 0).
		Updates(needUpdate).
		Error; err != nil {
		log.Errorf("update group [%d] all failed : [%s]", groupId, err)
		return fmt.Errorf("update group [%d] all failed : [%s]", groupId, err)
	}
	return nil
}

func (g *Group) UpdateDesc() error {
	if err := DataBase().Table(g.TableName()).
		Update("short_desc", g.ShortDesc).
		Where("id = ? and deleted = ?", g.ID, 0).
		Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateGroupType() error {
	if err := DataBase().Table(g.TableName()).
		Update("gtype", g.Gtype).
		Where("id = ? and deleted = ?", g.ID, 0).
		Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateAvatar() error {
	if err := DataBase().Table(g.TableName()).
		Update("avatar", g.Avatar).
		Where("id = ? and deleted = ?", g.ID, 0).
		Error; err != nil {
		log.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) GetByName() error {
	if err := DataBase().Table(g.TableName()).
		Where("name = ? and deleted = ?", g.Name, 0).
		First(g).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) GetByID() error {
	if err := DataBase().Table(g.TableName()).
		Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) Delete() error {
	if err := DataBase().Table(g.TableName()).
		Update("deleted", 1).
		Where("id = ? and deleted = ?", g.ID, 0).
		Error; err != nil {
		log.Errorf("update group [%s] deleted failed ", g.Name)
		return fmt.Errorf("deleted group [%s] failed ", g.Name)
	}
	return nil
}

// GroupMember 代表组成员
// role: 1-admin, 2-member, 3-viewer
// status: 1-有效, 0-无效
type GroupMember struct {
	IDBase
	GroupID  int64  `gorm:"column:group_id" json:"group_id,omitempty"` // 组ID
	UserID   int64  `gorm:"column:user_id" json:"user_id,omitempty"`   // 用户ID
	Nickname string `gorm:"column:nickname" json:"nickname,omitempty"` // 昵称
	Role     int64  `gorm:"column:role" json:"role,omitempty"`         // 角色
	Status   int64  `gorm:"column:status" json:"status,omitempty"`     // 状态
}

func (g GroupMember) TableName() string {
	return "group_member"
}

func (g *GroupMember) Create() error {
	err := DataBase().Table(g.TableName()).
		Where("group_id = ? and  user_id = ? and deleted = 0", g.GroupID, g.UserID).
		First(g).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Errorf("query group member failed: %s", err.Error())
			return err
		}
		err = DataBase().Table(g.TableName()).Create(g).Error
		if err != nil {
			return errors.ErrGroupIsAlreadyExist
		}
	} else {
		log.Errorf("group [%d] member [%d] is exist : ", g.GroupID, g.UserID)
		return errors.ErrGroupIsAlreadyExist
	}
	err = IncGroupProfileMembers(context.Background(), g.GroupID)
	if err != nil {
		log.Errorf("inc group [%d] members failed : %s", g.GroupID, err.Error())
		return err
	}
	log.Infof("group [%d] member [%d] created", g.GroupID, g.UserID)
	return nil
}

func (g *GroupMember) IsInGroup() (bool, error) {
	err := DataBase().Table(g.TableName()).
		Where("group_id = ? and  user_id = ? and deleted = 0", g.GroupID, g.UserID).
		First(g).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Errorf("query group member failed: %s", err.Error())
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (g *GroupMember) Delete() error {
	if err := DataBase().Table(g.TableName()).
		Where("user_id = ? and group_id = ? and deleted = 0", g.UserID, g.GroupID).
		Update("deleted", 1).
		Error; err != nil {
		return fmt.Errorf("group [%d] member [%d] failed %s", g.GroupID, g.UserID, err.Error())
	}
	return nil
}

func GetGroupMembers(groupID int, offset, number int) (list []*GroupMember, err error) {
	list = make([]*GroupMember, 0)
	err = DataBase().Table(GroupMember{}.TableName()).
		Where("group_id = ? and deleted = 0", groupID).
		Offset(offset).
		Limit(number).
		Scan(list).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return list, nil
}

func GetUserGroups(userID int, offset, pageSize int) (list []*Group, err error) {
	log.Infof("get user groups: %d, offset: %d, pageSize: %d", userID, offset, pageSize)
	list = make([]*Group, 0)
	err = DataBase().Model(Group{}).
		Where("creator_id = ? and deleted = 0", userID).
		Order("create_at desc").
		Offset((offset - 1) * pageSize).
		Limit(pageSize).
		Scan(&list).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return list, nil
}

func GetUserDefaultGroup(userID int) (g *Group, ok bool, err error) {
	userInfo := new(User)
	userInfo.ID = uint(userID)
	err = userInfo.GetById()
	if err != nil {
		return nil, false, err
	}
	g = new(Group)
	err = DataBase().Model(Group{}).
		Where("owner_id = ? and is_default = ?  and deleted = 0", userID, true).
		Scan(g).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, false, nil
	}
	newGroup := &Group{
		Name:        "默认的群组",
		OwnerID:     int64(userID),
		ShortDesc:   "默认的群组",
		CreatorID:   int64(userID),
		VisableType: api.ScopeType_AllPublic,
		IsDefault:   true,
		Gtype:       "",
		Members:     1,
		IDBase: IDBase{
			Base: Base{
				CreateAt: time.Now(),
				UpdateAt: time.Now(),
			},
		},
	}
	err = CreateGroup(newGroup)
	if err != nil {
		return nil, false, err
	}

	err = CreateGroupProfile(context.Background(), int64(newGroup.ID), "默认的群组", 0, false, 1)
	if err != nil {
		log.Errorf("create group profile failed: %s", err.Error())
	}
	mem := &GroupMember{
		GroupID:  int64(newGroup.ID),
		UserID:   int64(userID),
		Nickname: userInfo.Name,
		Role:     1,
		IDBase: IDBase{
			Base: Base{
				CreateAt: time.Now(),
				UpdateAt: time.Now(),
			},
		},
	}
	err = mem.Create()
	if err != nil {
		return nil, false, err
	}
	return g, true, nil
}

// GetUserFollowedGroups
func GetUserJoinedGroups(userID int, offset, pageSize int) (list []*Group, err error) {
	groupIds := make([]int, 0)
	err = DataBase().Model(&GroupMember{}).
		Select("group_id").
		Where("user_id = ? and deleted = 0", userID).
		Order("create_at desc").
		Scan(groupIds).
		Offset((offset - 1) * pageSize).
		Limit(pageSize).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	list = make([]*Group, 0)
	err = DataBase().Model(&Group{}).
		Select("*").
		Where("id in (?)", groupIds).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return list, nil
}

func GetGroupMemberInfoList(groupID int, offset, pageSize int) (users []*User, err error) {
	list := make([]int64, 0, pageSize)
	err = DataBase().Table(GroupMember{}.TableName()).
		Select("user_id").
		Where("group_id = ? and deleted = 0", groupID).
		Scan(&list).
		Offset((offset - 1) * pageSize).
		Limit(pageSize).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	users, err = GetUsersByIds(list)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func GetGroupsByIdsOrderByActive(groupIds []int, offset, pageSize int) (list []*Group, total int64, err error) {
	total = 0
	err = DataBase().Model(Group{}).
		Where("id in (?)", groupIds).
		Count(&total).
		Error
	if err != nil {
		return nil, 0, err
	}
	list = make([]*Group, 0)
	err = DataBase().Model(Group{}).
		Where("id in (?)", groupIds).
		Order("update_at desc").
		Offset((offset - 1) * pageSize).
		Limit(pageSize).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		log.Errorf("get groups by ids order by active failed: %s", err.Error())
		return nil, 0, err
	}
	return list, total, nil
}

func GetUserFollowedGroups(userID int, offset, pageSize int) (list []*Group, total int64, err error) {
	groupIds := make([]int, 0)
	err = DataBase().Model(&WatchItem{}).
		Select("distinct group_id").
		Where("user_id = ? and deleted = 0 and watch_item_type = ? and watch_type = ?",
			userID, WatchItemTypeGroup, WatchTypeIsWatch).
		Scan(&groupIds).
		Error
	if err != nil {
		return nil, 0, err
	}
	list, total, err = GetGroupsByIdsOrderByActive(groupIds, offset, pageSize)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return list, total, nil
}

func GetUserFollowedGroupIds(ctx context.Context, userID int) ([]int64, int64, error) {
	groupIds := make([]int64, 0)
	err := DataBase().Model(&WatchItem{}).
		Select("distinct group_id").
		Where("user_id = ? and deleted = 0 and watch_item_type = ? and watch_type = ?",
			userID, WatchItemTypeGroup, WatchTypeIsWatch).
		Scan(&groupIds).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		log.Errorf("get user [%d] followed group ids failed: %s", userID, err.Error())
		return nil, 0, err
	}
	log.Infof("get user [%d] followed group ids success: %v", userID, groupIds)
	return groupIds, int64(len(groupIds)), nil
}

func GetGroupByName(name string, offset, pageSize int) (groups []*Group, total int64, err error) {
	groups = make([]*Group, 0)
	err = DataBase().Model(Group{}).
		Where("name like ? and deleted = 0", "%"+name+"%").
		Count(&total).
		Error
	if err != nil {
		return nil, 0, err
	}
	err = DataBase().Model(Group{}).
		Where("name like ? and deleted = 0", "%"+name+"%").
		Offset((offset - 1) * pageSize).
		Limit(pageSize).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return groups, total, nil
}

// 根据group id 列表获取group 列表
func GetGroupsByIds(groupIds []int64) (groups []*Group, err error) {
	groups = make([]*Group, 0)
	err = DataBase().Model(Group{}).
		Where("id in (?)", groupIds).
		Scan(&groups).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return groups, nil
}

// GroupProfile 代表组的统计信息和扩展信息
type GroupProfile struct {
	IDBase
	GroupID        int64  `gorm:"column:group_id" json:"group_id,omitempty"`                 // 组ID
	Desc           string `gorm:"column:desc" json:"desc,omitempty"`                         // 组简介
	Members        int64  `gorm:"column:members" json:"members,omitempty"`                   // 成员数
	DefaultStoryId int64  `gorm:"column:default_story_id" json:"default_story_id,omitempty"` // 默认故事ID
	StoryCount     int64  `gorm:"column:story_count" json:"story_count,omitempty"`           // 故事数
	IsVerified     bool   `gorm:"column:is_verified" json:"is_verified,omitempty"`           // 是否认证
	Followers      int64  `gorm:"column:followers" json:"followers,omitempty"`               // 关注数
	BackgroundUrl  string `gorm:"column:background_url" json:"background_url,omitempty"`     // 背景图
}

func (g GroupProfile) TableName() string {
	return "group_profile"
}

func (g *GroupProfile) GetByGroupID() error {
	err := DataBase().Table(g.TableName()).Where("group_id = ? and deleted = 0", g.GroupID).First(g).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	return nil
}

func CreateGroupProfile(ctx context.Context, groupID int64, desc string, defaultStoryId int64, isVerified bool, followers int64) error {
	profile := &GroupProfile{
		GroupID:        groupID,
		Desc:           desc,
		Members:        1,
		DefaultStoryId: defaultStoryId,
		StoryCount:     0,
		IsVerified:     false,
		Followers:      1,
		IDBase: IDBase{
			Base: Base{
				CreateAt: time.Now(),
				UpdateAt: time.Now(),
			},
		},
	}
	err := DataBase().Table(profile.TableName()).Create(profile).Error
	if err != nil {
		return err
	}
	return nil
}

func GetGroupProfile(ctx context.Context, groupID int64) (profile *GroupProfile, err error) {
	profile = new(GroupProfile)
	err = DataBase().Table(profile.TableName()).Where("group_id = ? and deleted = 0", groupID).First(profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("get group profile failed: %s", err.Error())
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get group profile failed: %s", err.Error())
		return nil, nil
	}
	return profile, nil
}

// 根据groupIds 列表获取group profile 列表
func GetGroupProfiles(ctx context.Context, groupIds []int64) (profiles []*GroupProfile, err error) {
	profiles = make([]*GroupProfile, 0)
	err = DataBase().Table(GroupProfile{}.TableName()).Where("group_id in (?) and deleted = 0", groupIds).Find(&profiles).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Errorf("get group profiles failed: %s", err.Error())
			return nil, nil
		}
		log.Errorf("get group profiles failed: %s", err.Error())
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	log.Infof("get group profiles success: %v", profiles)
	return profiles, nil
}
func IncGroupProfileMembers(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("members", gorm.Expr("members + 1")).Error
	if err != nil {
		log.Errorf("inc group profile members failed: %s", err.Error())
		return err
	}
	return nil
}

func DecGroupProfileMembers(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("members", gorm.Expr("members - 1")).Error
	if err != nil {
		log.Errorf("dec group profile members failed: %s", err.Error())
		return err
	}
	return nil
}

func IncGroupProfileStoryCount(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("story_count", gorm.Expr("story_count + 1")).Error
	if err != nil {
		log.Errorf("inc group profile story count failed: %s", err.Error())
		return err
	}
	return nil
}

func DecGroupProfileStoryCount(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("story_count", gorm.Expr("story_count - 1")).Error
	if err != nil {
		log.Errorf("dec group profile story count failed: %s", err.Error())
		return err
	}
	return nil
}

func IncGroupProfileFollowers(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("followers", gorm.Expr("followers + 1")).Error
	if err != nil {
		log.Errorf("inc group profile followers failed: %s", err.Error())
		return err
	}
	return nil
}

func DecGroupProfileFollowers(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("followers", gorm.Expr("followers - 1")).Error
	if err != nil {
		log.Errorf("dec group profile followers failed: %s", err.Error())
		return err
	}
	return nil
}

// 根据groupId和userId获取用户加入小组的信息
func GetGroupMemberByGroupAndUser(ctx context.Context, groupId int64, userId int64) (member *GroupMember, err error) {
	member = new(GroupMember)
	err = DataBase().Table((&GroupMember{}).TableName()).
		Where("group_id = ? and user_id = ? and deleted = 0", groupId, userId).
		First(member).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Errorf("get group member by group and user failed: %s", err.Error())
			return nil, nil
		}
		log.Errorf("get group member by group and user failed: %s", err.Error())
		return nil, err
	}
	return member, nil
}

func UpdateGroupProfile(ctx context.Context, groupId int64, desc string, followers int64) error {
	needUpdate := make(map[string]interface{})
	if desc != "" {
		needUpdate["desc"] = desc
	}
	err := DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Updates(needUpdate).Error
	if err != nil {
		log.Errorf("update group profile failed: %s", err.Error())
		return err
	}
	return nil
}

// 新增：分页获取Group列表
func GetGroupList(ctx context.Context, offset, limit int) ([]*Group, error) {
	var groups []*Group
	err := DataBase().Model(&Group{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&groups).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return groups, nil
}

// 新增：通过Name唯一查询
func GetGroupByNameUnique(ctx context.Context, name string) (*Group, error) {
	group := &Group{}
	err := DataBase().Model(group).
		WithContext(ctx).
		Where("name = ?", name).
		First(group).Error
	if err != nil {
		return nil, err
	}
	return group, nil
}
