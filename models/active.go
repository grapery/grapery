package models

import (
	"context"
	"fmt"
	"time"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/log"
	"gorm.io/gorm"
)

// Active 用户活跃/动态记录
type Active struct {
	IDBase
	UserId            int64          `gorm:"column:user_id" json:"user_id,omitempty"`                         // 用户ID
	ActiveType        api.ActiveType `gorm:"column:active_type" json:"active_type,omitempty"`                 // 活跃类型
	GroupId           int64          `gorm:"column:group_id" json:"group_id,omitempty"`                       // 群组ID
	StoryId           int64          `gorm:"column:story_id" json:"story_id,omitempty"`                       // 故事ID
	StoryBoardId      int64          `gorm:"column:storyboard_id" json:"storyboard_id,omitempty"`             // 故事板ID
	StoryBoardSceneId int64          `gorm:"column:storyboard_scene_id" json:"storyboard_scene_id,omitempty"` // 场景ID
	StoryRoleId       int64          `gorm:"column:story_role_id" json:"story_role_id,omitempty"`             // 角色ID
	Content           string         `gorm:"column:content" json:"content,omitempty"`                         // 内容
	Status            int64          `gorm:"column:status" json:"status,omitempty"`                           // 状态
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
		Where("user_id = ? and deleted = 0", userID).
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
		Where("created_at < ? and  created_at > ? and deleted = 0", end, start).
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
		Where("creator_id = ? and active_type = ? and deleted = 0", creatorID, activeType).
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
		Where("group_id = ? and deleted = 0", groupID).
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
		Where("story_id = ? and deleted = 0", storyID).
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
		Where("storyboard_id = ? and deleted = 0", storyBoardID).
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
		Where("storyboard_scene_id = ? and deleted = 0", storyBoardSceneID).
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
		Where("story_role_id = ? and deleted = 0", storyRoleID).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get story role [%d] active failed ", storyRoleID))
		return nil, err
	}
	return ret, nil
}

// 按时间倒序获取用户关注的小组的活动
func GetActiveByFollowingGroupID(userID int64, groupIds []int64, page, pageSize int) ([]*Active, int64, error) {
	var ret = make([]*Active, 0)
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and group_id in (?) and deleted = 0", userID, groupIds).
		Order("create_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following group [%v] active failed ", userID, groupIds))
		return nil, 0, err
	}
	var total int64
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and group_id in (?) and deleted = 0", userID, groupIds).
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
		Where("user_id = ? and story_id in (?) and deleted = 0", userID, storyIds).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following story [%v] active failed ", userID, storyIds))
		return nil, 0, err
	}
	var total int64
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and story_id in (?) and deleted = 0", userID, storyIds).
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
		Where("user_id = ? and story_role_id in (?) and deleted = 0", userID, storyRoleIds).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("created_at desc").
		Scan(ret).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following story role [%v] active failed ", userID, storyRoleIds))
		return nil, 0, err
	}
	var total int64
	if err := DataBase().Model(Active{}).
		Where("user_id = ? and story_role_id in (?) and deleted = 0", userID, storyRoleIds).
		Count(&total).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("get user [%d] following story role [%v] active count failed ", userID, storyRoleIds))
		return nil, 0, err
	}
	return ret, total, nil
}

// 新增：分页获取Active列表
func GetActiveList(ctx context.Context, offset, limit int) ([]*Active, error) {
	var actives []*Active
	err := DataBase().Model(&Active{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&actives).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return actives, nil
}

// 新增：通过主键唯一查询
func GetActiveByID(ctx context.Context, id int64) (*Active, error) {
	active := &Active{}
	err := DataBase().Model(active).
		WithContext(ctx).
		Where("id = ?", id).
		First(active).Error
	if err != nil {
		return nil, err
	}
	return active, nil
}
