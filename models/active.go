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

type ActiveList []*Active

func (a ActiveList) Len() int {
	return len(a)
}

func (a ActiveList) Less(i, j int) bool {
	return a[i].CreateAt.Unix() > a[j].CreateAt.Unix()
}

func (a ActiveList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a *Active) Create() error {
	a.CreateAt = time.Now()
	a.UpdateAt = time.Now()
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

// 按时间倒序获取用户关注的小组的活动
func GetActiveByFollowingGroupID(userID int64, groupIds []int64, page, pageSize int) (*[]*Active, int64, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and group_id in (?) and delete = 0", userID, groupIds).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following group [%v] active failed ", userID, groupIds))
		return nil, 0, err
	}
	var total int64
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and group_id in (?) and delete = 0", userID, groupIds).
		Count(&total).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following group [%v] active count failed ", userID, groupIds))
		return nil, 0, err
	}
	return ret, total, nil
}

// 按时间倒序获取用户关注的故事的活动
func GetActiveByFollowingStoryID(userID int64, storyIds []int64, page, pageSize int) (*[]*Active, int64, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and story_id in (?) and delete = 0", userID, storyIds).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following story [%v] active failed ", userID, storyIds))
		return nil, 0, err
	}
	var total int64
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and story_id in (?) and delete = 0", userID, storyIds).
		Count(&total).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following story [%v] active count failed ", userID, storyIds))
		return nil, 0, err
	}
	return ret, total, nil
}

// 按时间倒序获取用户关注的故事角色的活动
func GetActiveByFollowingStoryRoleID(userID int64, storyRoleIds []int64, page, pageSize int) (*[]*Active, int64, error) {
	var ret = new([]*Active)
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and story_role_id in (?) and delete = 0", userID, storyRoleIds).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following story role [%v] active failed ", userID, storyRoleIds))
		return nil, 0, err
	}
	var total int64
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and story_role_id in (?) and delete = 0", userID, storyRoleIds).
		Count(&total).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following story role [%v] active count failed ", userID, storyRoleIds))
		return nil, 0, err
	}
	return ret, total, nil
}
