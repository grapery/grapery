package models

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/errors"
)

/*
用户有默认的自己的一个group,叫做场地或者空间
Group
用户参与的组织：
1.类似于一个大家庭，一个小家庭，情侣的二人世界
2.学校，学院，系，实验室，班级，同学合作小组，
3.一些关系比较近的大学校友同学，或者一些开放组织
4.一些公共组织，类似于气候变化，共同地理位置
5.一些大型公司，或者对抗大型公司恶性事件而结成的组织
6.私密学术讨论组织
*/
type Group struct {
	IDBase
	Name        string          `json:"name,omitempty"`
	ShortDesc   string          `json:"short_desc,omitempty"`
	Gtype       string          `json:"gtype,omitempty"`
	CreatorID   uint64          `json:"creator_id,omitempty"`
	OwnerID     uint64          `json:"owner_id,omitempty"`
	Members     uint64          `json:"members,omitempty"`
	VisableType api.VisibleType `json:"visable_type,omitempty"`
	Description string          `json:"description,omitempty"`
	Avatar      string          `json:"avatar,omitempty"`
	IsDefault   bool            `json:"is_default,omitempty"`
}

func (g Group) TableName() string {
	return "group"
}

func (g *Group) Create() error {
	err := DataBase().Table(g.TableName()).Where("name = ? and  user_id = ? and deleted = ?", g.Name, g.CreatorID, 0).Find(g).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Errorf("query group failed: %s", err.Error())
			return err
		}
		err = DataBase().Table(g.TableName()).Create(g).First(g).Error
		if err != nil {
			log.Errorf("create group [%s] failed: %s", g.Name, err.Error())
			return errors.ErrGroupIsAlreadyExist
		}
	} else {
		log.Errorf("group [%s] is exist : ", g.ID)
		return errors.ErrGroupIsAlreadyExist
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
	if err := DataBase().Table(g.TableName()).Update("short_desc", g.ShortDesc).Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateGroupType() error {
	if err := DataBase().Table(g.TableName()).Update("gtype", g.Gtype).Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateAvatar() error {
	if err := DataBase().Table(g.TableName()).Update("avatar", g.Avatar).Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) GetByName() error {
	if err := DataBase().Table(g.TableName()).Where("name = ? and deleted = ?",
		g.Name, 0).Find(g).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) GetByID() error {
	if err := DataBase().Table(g.TableName()).Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) Delete() error {
	if err := DataBase().Table(g.TableName()).Update("deleted", 1).Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("update group [%s] deleted failed ", g.Name)
		return fmt.Errorf("deleted group [%s] failed ", g.Name)
	}
	return nil
}

type GroupMember struct {
	IDBase
	GroupID  uint64 `json:"group_id,omitempty"`
	UserID   uint64 `json:"user_id,omitempty"`
	Nickname string
	Role     int64
}

func (g GroupMember) TableName() string {
	return "group_member"
}

func (g *GroupMember) Create() error {
	err := DataBase().Table(g.TableName()).Where("group_id = ? and  user_id = ? and deleted = ?", g.GroupID, g.UserID, 0).Find(g).Error
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
	return nil
}

func (g *GroupMember) IsInGroup() (bool, error) {
	err := DataBase().Table(g.TableName()).Where("group_id = ? and  user_id = ? and deleted = ?", g.GroupID, g.UserID, 0).Find(g).Error
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
	if err := DataBase().Table(g.TableName()).Update("deleted", 1).Where("user_id = ? and group_id = ? and deleted = ?", g.UserID, g.GroupID, 1).Error; err != nil {
		return fmt.Errorf("group [%d] member [%d] failed %s", g.GroupID, g.UserID, err.Error())
	}
	return nil
}

func GetGroupMembers(groupID int, offset, number int) (list []*GroupMember, err error) {
	list = make([]*GroupMember, 0)
	err = DataBase().Table(GroupMember{}.TableName()).Where("group_id = ? and deleted = 0", groupID).
		Scan(list).Offset(offset).Limit(number).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetUserGroups(userID int, offset, number int) (list []*Group, err error) {
	list = make([]*Group, 0)
	err = DataBase().Model(Group{}).Where("creator_id = ? and deleted = 0", userID).
		Scan(&list).Offset(offset).Limit(number).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetUserDefaultGroup(userID int) (g *Group, err error) {
	g = new(Group)
	err = DataBase().Model(Group{}).Where("creator_id = ? and is_default = ?  and deleted = 0", true, userID).
		Scan(g).Error
	if err != nil {
		return nil, err
	}
	return g, nil
}

// GetUserFollowedGroups
func GetUserJoinedGroups(userID int, offset, number int) (list []*Group, err error) {
	groupIds := make([]int, 0)
	err = DataBase().Model(&GroupMember{}).Select("group_id").Where("user_id = ? and deleted = 0", userID).
		Scan(groupIds).Offset(offset).Limit(number).Error
	if err != nil {
		return nil, err
	}
	list = make([]*Group, 0)
	err = DataBase().Model(&Group{}).Select("*").Where(" group_id in (?)", groupIds).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetGroupMemberInfoList(groupID int, offset, number int) (users []*User, err error) {
	list := make([]int, 0, number)
	err = DataBase().Table(GroupMember{}.TableName()).Select("user_id").Where("group_id = ? and deleted = 0", groupID).
		Scan(&list).Offset(offset).Limit(number).Error
	if err != nil {
		return nil, err
	}
	users, err = GetUsersByIds(list)
	if err != nil {
		return nil, err
	}
	return users, nil
}
