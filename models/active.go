package models

import (
	"fmt"
	"time"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/log"
)

type Active struct {
	IDBase
	UserId            int64          `json:"user_id,omitempty"`
	ActiveType        api.ActiveType `json:"active_type,omitempty"`
	GroupId           int64          `json:"group_id,omitempty"`
	StoryId           int64          `json:"story_id,omitempty"`
	StoryBoardId      int64          `json:"storyboard_id,omitempty"`
	StoryBoardSceneId int64          `json:"storyboard_scene_id,omitempty"`
	StoryRoleId       int64          `json:"story_role_id,omitempty"`
	Content           string         `json:"content,omitempty"`
	Status            int64          `json:"status,omitempty"`
}

func (a Active) TableName() string {
	return "active"
}

func (a *Active) Create() error {
	if err := DataBase().Model(Active{}).Create(a).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("create new active [%d] failed : [%s]", a.ID, err.Error()))
		return fmt.Errorf("create new active [%d] failed: %s", a.ID, err.Error())
	}
	return nil
}

func (a *Active) Get() error {
	if err := DataBase().Model(Active{}).First(a).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get active [%d] failed : [%s]", a.ID, err.Error()))
		return err
	}
	return nil
}

func (a *Active) Delete() error {
	if err := DataBase().Model(Active{}).Update("deleted", 1); err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("update active [%d] deleted failed ", a.ID))
		return fmt.Errorf("deleted active [%d] failed ", a.ID)
	}
	log.Log().WithOptions(logFieldModels).Info(fmt.Sprintf("delete active [%d] success", a.ID))
	return nil
}

// 按时间倒序获取用户的活动
func GetActiveByUserID(userID int64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and delete = 0", userID).
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] active failed ", userID))
		return nil, err
	}
	return ret, nil
}

// 按时间倒序获取活动
func GetActiveListByTimeRange(start time.Time, end time.Time) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("created_at < ? and  created_at > ? and delete = 0", end, start).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).
			Error(fmt.Sprintf("get active in range [%s--%s] failed ", start.String(), end.String()))
		return nil, err
	}
	return ret, nil
}

// 按时间倒序获取用户的活动
func GetActiveListByActiveType(creatorID int64, activeType uint) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().
		Model(Active{}).
		Where("creator_id = ? and active_type = ? and delete = 0", creatorID, activeType).
		Order("created_at desc").
		Scan(ret).
		Error; err != nil {
		log.Log().WithOptions(logFieldModels).
			Error(fmt.Sprintf("get user [%d] active type [%d] failed ", creatorID, activeType))
		return nil, err
	}
	return ret, nil
}

// 按时间倒序获取群组的活动
func GetActiveByGroupID(groupID int64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("group_id = ? and delete = 0", groupID).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get group [%d] active failed ", groupID))
		return nil, err
	}
	return ret, nil
}

// 按时间倒序获取故事的活动
func GetActiveByStoryID(storyID int64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("story_id = ? and delete = 0", storyID).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get story [%d] active failed ", storyID))
		return nil, err
	}
	return ret, nil
}

// 按时间倒序获取故事板的活动
func GetActiveByStoryBoardID(storyBoardID int64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("storyboard_id = ? and delete = 0", storyBoardID).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get storyboard [%d] active failed ", storyBoardID))
		return nil, err
	}
	return ret, nil
}

// 按时间倒序获取故事场景的活动
func GetActiveByStoryBoardSceneID(storyBoardSceneID int64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("storyboard_scene_id = ? and delete = 0", storyBoardSceneID).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get storyboard scene [%d] active failed ", storyBoardSceneID))
		return nil, err
	}
	return ret, nil
}

// 按时间倒序获取故事角色的活动
func GetActiveByStoryRoleID(storyRoleID int64) (*[]*Active, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("story_role_id = ? and delete = 0", storyRoleID).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get story role [%d] active failed ", storyRoleID))
		return nil, err
	}
	return ret, nil
}
