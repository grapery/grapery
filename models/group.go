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

type Group struct {
	IDBase
	Name        string        `json:"name,omitempty"`
	ShortDesc   string        `json:"short_desc,omitempty"`
	Gtype       string        `json:"gtype,omitempty"`
	CreatorID   int64         `json:"creator_id,omitempty"`
	OwnerID     int64         `json:"owner_id,omitempty"`
	Members     int64         `json:"members,omitempty"`
	VisableType api.ScopeType `json:"visable_type,omitempty"`
	Description string        `json:"description,omitempty"`
	Avatar      string        `json:"avatar,omitempty"`
	IsDefault   bool          `json:"is_default,omitempty"`
	Status      int64         `json:"status,omitempty"`
	Tags        string        `json:"tags,omitempty"`
	Location    string        `json:"location,omitempty"`
}

func (g Group) TableName() string {
	return "group"
}

func (g *Group) Create() error {
	if g.Avatar == "" {
		g.Avatar = "https://grapery-1301865260.cos.ap-shanghai.myqcloud.com/avator/tmp3evp1xxl.png"
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
		g.Avatar = "https://grapery-1301865260.cos.ap-shanghai.myqcloud.com/avator/tmp3evp1xxl.png"
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

type GroupMember struct {
	IDBase
	GroupID  int64 `json:"group_id,omitempty"`
	UserID   int64 `json:"user_id,omitempty"`
	Nickname string
	Role     int64
	Status   int64
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
		Scan(list).
		Offset(offset).
		Limit(number).
		Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetUserGroups(userID int, offset, number int) (list []*Group, err error) {
	list = make([]*Group, 0)
	err = DataBase().Model(Group{}).
		Where("creator_id = ? and deleted = 0", userID).
		Scan(&list).
		Offset(offset).
		Limit(number).
		Error
	if err != nil {
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
func GetUserJoinedGroups(userID int, offset, number int) (list []*Group, err error) {
	groupIds := make([]int, 0)
	err = DataBase().Model(&GroupMember{}).
		Select("group_id").
		Where("user_id = ? and deleted = 0", userID).
		Scan(groupIds).
		Offset(offset).
		Limit(number).
		Error
	if err != nil {
		return nil, err
	}
	list = make([]*Group, 0)
	err = DataBase().Model(&Group{}).
		Select("*").
		Where("id in (?)", groupIds).
		Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetGroupMemberInfoList(groupID int, offset, number int) (users []*User, err error) {
	list := make([]int64, 0, number)
	err = DataBase().Table(GroupMember{}.TableName()).
		Select("user_id").
		Where("group_id = ? and deleted = 0", groupID).
		Scan(&list).
		Offset(offset).
		Limit(number).
		Error
	if err != nil {
		return nil, err
	}
	users, err = GetUsersByIds(list)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func GetGroupsByIdsOrderByActive(groupIds []int, offset, number int) (list []*Group, total int64, err error) {
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
		Offset(offset).
		Limit(number).
		Error
	if err != nil {
		log.Errorf("get groups by ids order by active failed: %s", err.Error())
		return nil, 0, err
	}
	return list, total, nil
}

func GetUserFollowedGroups(userID int, offset, number int) (list []*Group, total int64, err error) {
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
	list, total, err = GetGroupsByIdsOrderByActive(groupIds, offset, number)
	if err != nil {
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
		log.Errorf("get user [%d] followed group ids failed: %s", userID, err.Error())
		return nil, 0, err
	}
	log.Infof("get user [%d] followed group ids success: %v", userID, groupIds)
	return groupIds, int64(len(groupIds)), nil
}

func GetGroupByName(name string, offset, number int) (groups []*Group, total int64, err error) {
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
		Offset(offset).
		Limit(number).
		Error
	if err != nil {
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
		return nil, err
	}
	return groups, nil
}

type GroupProfile struct {
	IDBase
	GroupID        int64  `json:"group_id,omitempty"`
	Desc           string `json:"desc,omitempty"`
	Members        int64  `json:"members,omitempty"`
	DefaultStoryId int64  `json:"default_story_id,omitempty"`
	StoryCount     int64  `json:"story_count,omitempty"`
	IsVerified     bool   `json:"is_verified,omitempty"`
	Followers      int64  `json:"followers,omitempty"`
	BackgroundUrl  string `json:"background_url,omitempty"`
}

func (g GroupProfile) TableName() string {
	return "group_profile"
}

func (g *GroupProfile) GetByGroupID() error {
	return DataBase().Table(g.TableName()).Where("group_id = ? and deleted = 0", g.GroupID).First(g).Error
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
			return nil, nil
		}
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	return profiles, nil
}
func IncGroupProfileMembers(ctx context.Context, groupId int64) error {
	return DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("members", gorm.Expr("members + 1")).Error
}

func DecGroupProfileMembers(ctx context.Context, groupId int64) error {
	return DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("members", gorm.Expr("members - 1")).Error
}

func IncGroupProfileStoryCount(ctx context.Context, groupId int64) error {
	return DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("story_count", gorm.Expr("story_count + 1")).Error
}

func DecGroupProfileStoryCount(ctx context.Context, groupId int64) error {
	return DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("story_count", gorm.Expr("story_count - 1")).Error
}

func IncGroupProfileFollowers(ctx context.Context, groupId int64) error {
	return DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("followers", gorm.Expr("followers + 1")).Error
}

func DecGroupProfileFollowers(ctx context.Context, groupId int64) error {
	return DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Update("followers", gorm.Expr("followers - 1")).Error
}

// 根据groupId和userId获取用户加入小组的信息
func GetGroupMemberByGroupAndUser(ctx context.Context, groupId int64, userId int64) (member *GroupMember, err error) {
	member = new(GroupMember)
	err = DataBase().Table((&GroupMember{}).TableName()).
		Where("group_id = ? and user_id = ? and deleted = 0", groupId, userId).
		First(member).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return member, nil
}

func UpdateGroupProfile(ctx context.Context, groupId int64, desc string, followers int64) error {
	needUpdate := make(map[string]interface{})
	if desc != "" {
		needUpdate["desc"] = desc
	}
	return DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Updates(needUpdate).Error
}
