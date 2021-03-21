package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

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
	VisableType api.VisibleType `json:"visable_type,omitempty"`
	Description string          `json:"description,omitempty"`
	Avatar      string          `json:"avatar,omitempty"`
}

func (g Group) TableName() string {
	return "group"
}

func (g *Group) Create() error {
	err := database.Model(g).Where("name = ? and  user_id = ? and deleted = ?", g.Name, g.CreatorID, 0).Find(g).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Errorf("query group failed: %s", err.Error())
			return err
		}
		err = database.Model(g).Create(g).First(g).Error
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

func (g *Group) UpdateDesc() error {
	if err := database.Model(g).Update("short_desc", g.ShortDesc).Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateGroupType() error {
	if err := database.Model(g).Update("gtype", g.Gtype).Error; err != nil {
		log.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] desc failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) UpdateAvatar() error {
	if err := database.Model(g).Update("avatar", g.Avatar).Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
		return fmt.Errorf("update group [%d] avatar failed : [%s]", g.ID, err)
	}
	return nil
}

func (g *Group) GetByName() error {
	if err := database.Model(g).Where("name = ? and deleted = ?",
		g.Name, 0).Find(g).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) GetByID() error {
	if err := database.Model(g).Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("get group [%s] info failed : [%s]", g.Name, err)
		return fmt.Errorf("get group [%s] info failed ", g.Name)
	}
	return nil
}

func (g *Group) Delete() error {
	if err := database.Model(g).Update("deleted", 1).Where("id = ? and deleted = ?", g.ID, 0).Error; err != nil {
		log.Errorf("update group [%s] deleted failed ", g.Name)
		return fmt.Errorf("deleted group [%s] failed ", g.Name)
	}
	return nil
}

type GroupMember struct {
	IDBase
	GroupID uint64 `json:"group_id,omitempty"`
	UserID  uint64 `json:"user_id,omitempty"`
}

func (g GroupMember) TableName() string {
	return "group_member"
}

func (g *GroupMember) Create() error {
	err := database.Model(g).Where("group_id = ? and  user_id = ? and deleted = ?", g.GroupID, g.UserID, 0).Find(g).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Errorf("query group member failed: %s", err.Error())
			return err
		}
		err = database.Model(g).Create(g).Error
		if err != nil {
			return errors.ErrGroupIsAlreadyExist
		}
	} else {
		log.Errorf("group [%d] member [%d] is exist : ", g.GroupID, g.UserID)
		return errors.ErrGroupIsAlreadyExist
	}
	return nil
}

func (g *GroupMember) IsInOneGroup() (bool, error) {
	err := database.Model(g).Where("group_id = ? and  user_id = ? and deleted = ?", g.GroupID, g.UserID, 0).Find(g).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Errorf("query group member failed: %s", err.Error())
			return false, nil
		}
		return false, nil
	}
	return true, nil
}

func (g *GroupMember) Delete() error {
	if err := database.Model(g).Update("deleted", 1).Where("user_id = ? and group_id = ? and deleted = ?", g.UserID, g.GroupID, 1).Error; err != nil {
		return fmt.Errorf("group [%d] member [%d] failed %s", g.GroupID, g.UserID, err.Error())
	}
	return nil
}

func GetGroupMembers(groupID int, offset, number int) (list []*GroupMember, err error) {
	list = make([]*GroupMember, 0)
	err = database.Model(&GroupMember{}).Where("group_id = ? and deleted = 0", groupID).
		Scan(list).Offset(offset).Limit(number).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetUserGroups(userID int, offset, number int) (list []*GroupMember, err error) {
	list = make([]*GroupMember, 0)
	err = database.Model(&GroupMember{}).Where("user_id = ? and deleted = 0", userID).
		Scan(list).Offset(offset).Limit(number).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}
