package models

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
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

func (g *Group) Create(ctx context.Context) error {
	if g.Avatar == "" {
		g.Avatar = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/cool_rest.jpg"
	}
	err := DataBase().Table(g.TableName()).WithContext(ctx).
		Where("name = ? and  creator_id = ? and owner_id = ? and deleted = ?",
			g.Name, g.CreatorID, g.CreatorID, 0).
		First(g).
		Error

	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("query group failed: %s", err)
		return err
	}
	if err == gorm.ErrRecordNotFound {
		err = DataBase().Table(g.TableName()).WithContext(ctx).Create(g).Error
		if err != nil {
			log.Errorf("create group [%s] failed: %s", g.Name, err)
			return errors.ErrGroupIsAlreadyExist
		}
	}

	return nil
}

func CreateGroup(ctx context.Context, g *Group) error {
	err := DataBase().Table(g.TableName()).WithContext(ctx).
		Where("name = ? and  creator_id = ? and owner_id = ? and deleted = ?",
			g.Name, g.CreatorID, g.CreatorID, 0).WithContext(ctx).
		First(g).
		Error
	if err == nil && g.OwnerID != 0 {
		return nil
	}
	if g.Avatar == "" {
		g.Avatar = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/cool_rest.jpg"
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("query group failed: %s", err.Error())
		return err
	}
	if err == gorm.ErrRecordNotFound {
		err = DataBase().Table(g.TableName()).WithContext(ctx).Create(g).Error
		if err != nil {
			log.Errorf("create group [%s] failed: %s", g.Name, err.Error())
			return errors.ErrGroupIsAlreadyExist
		}
	}

	return nil
}

func (g *Group) UpdateAll(ctx context.Context, groupId int64) error {
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
	if err := DataBase().Table(g.TableName()).WithContext(ctx).
		Where("id = ? and deleted = ?", groupId, 0).
		Updates(needUpdate).
		Error; err != nil {
		log.Errorf("update group [%d] all failed : [%s]", groupId, err)
		return fmt.Errorf("update group [%d] all failed : [%s]", groupId, err)
	}
	return nil
}

func (g *Group) UpdateDesc(ctx context.Context) error {
	if err := DataBase().Table(g.TableName()).WithContext(ctx).
		Update("short_desc", g.ShortDesc).
		Where("id = ? and deleted = ?", g.ID, 0).
		Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateGroupType(ctx context.Context) error {
	if err := DataBase().Table(g.TableName()).WithContext(ctx).
		Update("gtype", g.Gtype).
		Where("id = ? and deleted = ?", g.ID, 0).
		Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateAvatar(ctx context.Context) error {
	if err := DataBase().Table(g.TableName()).WithContext(ctx).
		Update("avatar", g.Avatar).
		Where("id = ? and deleted = ?", g.ID, 0).
		Error; err != nil {
		log.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) GetByName(ctx context.Context) error {
	if err := DataBase().Table(g.TableName()).WithContext(ctx).
		Where("name = ? and deleted = ?", g.Name, 0).
		First(g).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) GetByID(ctx context.Context) error {
	if err := DataBase().Table(g.TableName()).WithContext(ctx).
		Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) Delete(ctx context.Context) error {
	if err := DataBase().Table(g.TableName()).WithContext(ctx).
		Update("deleted", 1).
		Where("id = ? and deleted = ?", g.ID, 0).
		Error; err != nil {
		log.Errorf("update group [%s] deleted failed ", g.Name)
		return fmt.Errorf("deleted group [%s] failed ", g.Name)
	}
	return nil
}

type GroupMember struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	GroupID  int64               `gorm:"column:group_id" json:"group_id,omitempty"` // 组ID
	UserID   int64               `gorm:"column:user_id" json:"user_id,omitempty"`   // 用户ID
	Nickname string              `gorm:"column:nickname" json:"nickname,omitempty"` // 昵称
	Role     api.GroupMemberType `gorm:"column:role" json:"role,omitempty"`         // 角色
	Status   int64               `gorm:"column:status" json:"status,omitempty"`     // 状态
}

func (g GroupMember) TableName() string {
	return "group_member"
}

func (g *GroupMember) Create(ctx context.Context) error {
	created, err := JoinGroupMember(ctx, g)
	if err != nil {
		return err
	}
	if !created {
		return errors.ErrGroupIsAlreadyExist
	}
	log.Infof("group [%d] member [%d] created", g.GroupID, g.UserID)
	return nil
}

func (g *GroupMember) IsInGroup(ctx context.Context) (bool, error) {
	err := DataBase().Table(g.TableName()).WithContext(ctx).
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

func (g *GroupMember) Delete(ctx context.Context) error {
	_, err := LeaveGroupMember(ctx, g.GroupID, g.UserID)
	return err
}

func JoinGroupMember(ctx context.Context, member *GroupMember) (bool, error) {
	var created bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing GroupMember
		err := tx.Table(member.TableName()).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND user_id = ?", member.GroupID, member.UserID).
			First(&existing).Error
		switch {
		case err == nil:
			if !existing.Deleted {
				return nil
			}
			updates := map[string]interface{}{
				"deleted":  false,
				"role":     member.Role,
				"nickname": member.Nickname,
				"status":   member.Status,
			}
			if err := tx.Table(member.TableName()).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
				return err
			}
			created = true
		case err != nil && err != gorm.ErrRecordNotFound:
			return err
		default:
			if err := tx.Table(member.TableName()).Create(member).Error; err != nil {
				return err
			}
			created = true
		}

		if !created {
			return nil
		}
		return tx.Table((&GroupProfile{}).TableName()).
			Where("group_id = ? and deleted = 0", member.GroupID).
			Update("members", gorm.Expr("CASE WHEN members IS NULL THEN 1 ELSE members + 1 END")).Error
	})
	if err != nil {
		return false, err
	}
	return created, nil
}

func LeaveGroupMember(ctx context.Context, groupID, userID int64) (bool, error) {
	var removed bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var member GroupMember
		err := tx.Table(member.TableName()).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND user_id = ?", groupID, userID).
			First(&member).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if member.Deleted {
			return nil
		}
		if err := tx.Table(member.TableName()).Where("id = ?", member.ID).
			Updates(map[string]interface{}{"deleted": true}).Error; err != nil {
			return err
		}
		removed = true
		return tx.Table((&GroupProfile{}).TableName()).
			Where("group_id = ? and deleted = 0", groupID).
			Update("members", gorm.Expr("CASE WHEN members > 0 THEN members - 1 ELSE 0 END")).Error
	})
	if err != nil {
		return false, err
	}
	return removed, nil
}

func GetGroupMembers(ctx context.Context, groupID int, offset, number int) (list []*GroupMember, total int64, hasMore bool, err error) {
	list = make([]*GroupMember, 0)
	err = DataBase().Table(GroupMember{}.TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupID).
		Count(&total).
		Error
	if err != nil {
		return nil, 0, false, err
	}
	err = DataBase().Table(GroupMember{}.TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupID).
		Offset((offset - 1) * number).
		Limit(number).
		Scan(list).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, false, nil
		}
		return nil, 0, false, err
	}
	total = int64(len(list))
	hasMore = total > int64(offset+number)
	return list, total, hasMore, nil
}

func GetUserGroups(ctx context.Context, userID int, offset, pageSize int) (list []*Group, total int64, hasMore bool, err error) {
	log.Infof("get user groups: %d, offset: %d, pageSize: %d", userID, offset, pageSize)
	list = make([]*Group, 0)

	// 参数验证
	if offset < 1 {
		offset = 1
	}
	if pageSize < 1 {
		pageSize = 10 // 默认页面大小
	}

	// 获取用户加入的所有组ID（去重）
	type GroupMemberResult struct {
		GroupID  int64     `json:"group_id"`
		CreateAt time.Time `json:"create_at"`
	}

	groupMemberResults := make([]GroupMemberResult, 0)
	err = DataBase().Model(&GroupMember{}).WithContext(ctx).
		Select("group_id, MAX(create_at) as create_at").
		Where("user_id = ? and deleted = 0", userID).
		Group("group_id").
		Order("create_at desc").
		Scan(&groupMemberResults).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, false, nil
		}
		return nil, 0, false, err
	}

	// 提取组ID
	groupIds := make([]int64, 0, len(groupMemberResults))
	for _, result := range groupMemberResults {
		groupIds = append(groupIds, result.GroupID)
	}

	total = int64(len(groupIds))
	if total == 0 {
		return nil, 0, false, nil
	}

	// 分页处理
	start := (offset - 1) * pageSize
	if start >= len(groupIds) {
		return nil, total, false, nil
	}

	end := start + pageSize
	if end > len(groupIds) {
		end = len(groupIds)
	}

	pagedGroupIds := groupIds[start:end]

	// 根据组ID获取组信息
	err = DataBase().Model(Group{}).WithContext(ctx).
		Where("id in (?) and deleted = 0", pagedGroupIds).
		Order("create_at desc").
		Scan(&list).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, total, false, nil
		}
		return nil, 0, false, err
	}
	hasMore = total > int64(offset+pageSize)
	return list, total, hasMore, nil
}

func GetUserDefaultGroup(ctx context.Context, userID int) (g *Group, ok bool, err error) {
	userInfo := new(User)
	userInfo.ID = uint(userID)
	err = userInfo.GetById(ctx)
	if err != nil {
		return nil, false, err
	}
	g = new(Group)
	err = DataBase().Model(Group{}).WithContext(ctx).
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
		VisableType: api.ScopeType_PROTECT_SCOPE,
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
	err = CreateGroup(ctx, newGroup)
	if err != nil {
		return nil, false, err
	}

	err = CreateGroupProfile(ctx, int64(newGroup.ID), "默认的群组", 0, false, 1)
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
	err = mem.Create(ctx)
	if err != nil {
		return nil, false, err
	}
	return g, true, nil
}

// GetUserFollowedGroups
func GetUserJoinedGroups(ctx context.Context, userID int, offset, pageSize int) (list []*Group, total int64, err error) {
	groupIds := make([]int, 0)
	err = DataBase().Model(&GroupMember{}).WithContext(ctx).
		Select("group_id").
		Where("user_id = ? and deleted = 0", userID).
		Order("create_at desc").
		Scan(groupIds).
		Offset((offset - 1) * pageSize).
		Limit(pageSize).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	// 去重ID，避免重复查询
	groupIds = UniqueInts(groupIds)
	if len(groupIds) == 0 {
		return nil, 0, nil
	}

	list = make([]*Group, 0)
	err = DataBase().Model(&Group{}).WithContext(ctx).
		Select("*").
		Where("id in (?)", groupIds).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return list, int64(len(groupIds)), nil
}

func GetGroupMemberInfoList(ctx context.Context, groupID int, offset, pageSize int) (users []*User, err error) {
	list := make([]int64, 0, pageSize)
	err = DataBase().Table(GroupMember{}.TableName()).WithContext(ctx).
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
	users, err = GetUsersByIds(ctx, list)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func GetGroupsByIdsOrderByActive(ctx context.Context, groupIds []int, offset, pageSize int) (list []*Group, total int64, err error) {
	// 去重ID，避免重复查询
	groupIds = UniqueInts(groupIds)
	if len(groupIds) == 0 {
		return nil, 0, nil
	}

	total = 0
	err = DataBase().Model(Group{}).WithContext(ctx).
		Where("id in (?)", groupIds).
		Count(&total).
		Error
	if err != nil {
		return nil, 0, err
	}
	list = make([]*Group, 0)
	err = DataBase().Model(Group{}).WithContext(ctx).
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

func GetUserFollowedGroups(ctx context.Context, userID int, offset, pageSize int) (list []*Group, total int64, err error) {
	groupIds := make([]int, 0)
	err = DataBase().Model(&WatchItem{}).WithContext(ctx).
		Select("distinct group_id").
		Where("user_id = ? and deleted = 0 and watch_item_type = ? and watch_type = ?",
			userID, WatchItemTypeGroup, WatchTypeIsWatching).
		Scan(&groupIds).
		Error
	if err != nil {
		return nil, 0, err
	}
	list, total, err = GetGroupsByIdsOrderByActive(ctx, groupIds, offset, pageSize)
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
			userID, WatchItemTypeGroup, WatchTypeIsWatching).
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
	// 去重ID，避免重复查询
	groupIds = UniqueInt64s(groupIds)
	if len(groupIds) == 0 {
		return nil, nil
	}

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

func DisableGroupMembers(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupId).
		Update("status", 0).Error
	if err != nil {
		log.Errorf("disable group members failed: %s", err)
		return err
	}
	log.Infof("disable group members success: %d", groupId)
	return nil
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

func (g *GroupProfile) GetByGroupID(ctx context.Context) error {
	err := DataBase().Table(g.TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", g.GroupID).First(g).Error
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
	err := DataBase().Table(profile.TableName()).WithContext(ctx).Create(profile).Error
	if err != nil {
		return err
	}
	return nil
}

func GetGroupProfile(ctx context.Context, groupID int64) (profile *GroupProfile, err error) {
	profile = new(GroupProfile)
	err = DataBase().Table(profile.TableName()).WithContext(ctx).Where("group_id = ? and deleted = 0", groupID).First(profile).Error
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
	err = DataBase().Table(GroupProfile{}.TableName()).WithContext(ctx).Where("group_id in (?) and deleted = 0", groupIds).Find(&profiles).Error
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
	err := DataBase().Table((&GroupProfile{}).TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupId).
		Update("members", gorm.Expr("members + 1")).Error
	if err != nil {
		log.Errorf("inc group profile members failed: %s", err)
		return err
	}
	return nil
}

func DecGroupProfileMembers(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupId).
		Update("members", gorm.Expr("members - 1")).Error
	if err != nil {
		log.Errorf("dec group profile members failed: %s", err)
		return err
	}
	return nil
}

func IncGroupProfileStoryCount(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupId).
		Update("story_count", gorm.Expr("story_count + 1")).Error
	if err != nil {
		log.Errorf("inc group profile story count failed: %s", err)
		return err
	}
	return nil
}

func DecGroupProfileStoryCount(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupId).
		Update("story_count", gorm.Expr("story_count - 1")).Error
	if err != nil {
		log.Errorf("dec group profile story count failed: %s", err)
		return err
	}
	return nil
}

func IncGroupProfileFollowers(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupId).
		Update("followers", gorm.Expr("followers + 1")).Error
	if err != nil {
		log.Errorf("inc group profile followers failed: %s", err)
		return err
	}
	return nil
}

func DecGroupProfileFollowers(ctx context.Context, groupId int64) error {
	err := DataBase().Table((&GroupProfile{}).TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupId).
		Update("followers", gorm.Expr("followers - 1")).Error
	if err != nil {
		log.Errorf("dec group profile followers failed: %s", err)
		return err
	}
	return nil
}

// 根据groupId和userId获取用户加入小组的信息
func GetGroupMemberByGroupAndUser(ctx context.Context, groupId int64, userId int64) (member *GroupMember, err error) {
	member = new(GroupMember)
	err = DataBase().Table((&GroupMember{}).TableName()).WithContext(ctx).
		Where("group_id = ? and user_id = ? and deleted = 0", groupId, userId).
		First(member).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Errorf("get group member by group and user failed: %s", err)
			return nil, nil
		}
		log.Errorf("get group member by group and user failed: %s", err)
		return nil, err
	}
	return member, nil
}

func UpdateGroupProfile(ctx context.Context, groupId int64, desc string, followers int64) error {
	needUpdate := make(map[string]interface{})
	if desc != "" {
		needUpdate["desc"] = desc
	}
	err := DataBase().Table((&GroupProfile{}).TableName()).WithContext(ctx).
		Where("group_id = ? and deleted = 0", groupId).
		Updates(needUpdate).Error
	if err != nil {
		log.Errorf("update group profile failed: %s", err)
		return err
	}
	return nil
}

// 新增：分页获取Group列表
func GetGroupList(ctx context.Context, offset, limit int) ([]*Group, error) {
	var groups []*Group
	err := DataBase().Model(&Group{}).WithContext(ctx).
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
		WithContext(ctx).WithContext(ctx).
		Where("name = ?", name).
		First(group).Error
	if err != nil {
		return nil, err
	}
	return group, nil
}
