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

func (g *Group) UpdateAll() error {
	if err := DataBase().Table(g.TableName()).
		Update("short_desc", g.ShortDesc).
		Update("gtype", g.Gtype).
		Update("avatar", g.Avatar).
		Update("name", g.Name).
		Error; err != nil {
		log.Errorf("update group [%d] all failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] all failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateDesc() error {
	if err := DataBase().Table(g.TableName()).
		Update("short_desc", g.ShortDesc).Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateGroupType() error {
	if err := DataBase().Table(g.TableName()).
		Update("gtype", g.Gtype).Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateAvatar() error {
	if err := DataBase().Table(g.TableName()).
		Update("avatar", g.Avatar).Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) GetByName() error {
	if err := DataBase().Table(g.TableName()).Where("name = ? and deleted = ?",
		g.Name, 0).First(g).Error; err != nil {
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

type GroupProfile struct {
	IDBase
	GroupID        int64 `json:"group_id,omitempty"`
	Desc           string
	Members        int64
	DefaultStoryId int64
	StoryCount     int64
	IsVerified     bool
	Followers      int64
}

func (g *GroupProfile) TableName() string {
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
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return profile, nil
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

func UpdateGroupProfile(ctx context.Context, groupId int64, desc string, defaultStoryId int64, isVerified bool, followers int64) error {
	needUpdate := make(map[string]interface{})
	if desc != "" {
		needUpdate["desc"] = desc
	}
	if defaultStoryId != 0 {
		needUpdate["default_story_id"] = defaultStoryId
	}
	if isVerified {
		needUpdate["is_verified"] = isVerified
	}
	return DataBase().Table((&GroupProfile{}).TableName()).
		Where("group_id = ? and deleted = 0", groupId).
		Updates(needUpdate).Error
}