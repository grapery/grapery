package models

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type StoryRole struct {
	IDBase
	StoryID              int64  `json:"story_id"`
	CharacterName        string `json:"character_name"`
	CharacterAvatar      string `json:"character_avatar"`
	CharacterID          string `json:"character_id"`
	CharacterType        string `json:"character_type"`
	CharacterPrompt      string `json:"character_prompt"`
	CharacterRefImages   string `json:"character_ref_images"`
	CharacterDescription string `json:"character_description"`
	CreatorID            int64  `json:"creator_id"`
	Status               int    `json:"status"`
	LikeCount            int64  `json:"like_count"`
	FollowCount          int64  `json:"follow_count"`
	StoryboardNum        int64  `json:"storyboard_num"`
	Version              int64  `json:"version"`
	BranchId             int64  `json:"branch_id"`
}

func (s StoryRole) String() string {
	roleJson, _ := json.Marshal(s)
	return string(roleJson)
}

func (s StoryRole) TableName() string {
	return "story_role"
}

func GetStoryRoleByCreatorId(ctx context.Context, creatorId int64) ([]*StoryRole, error) {
	var roles []*StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("creator_id = ?", creatorId).
		Find(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return roles, nil
}

func CreateStoryRole(ctx context.Context, role *StoryRole) (int64, error) {
	if err := DataBase().Model(role).
		WithContext(ctx).
		Create(role).Error; err != nil {
		return 0, err
	}
	return int64(role.ID), nil
}

func GetStoryRole(ctx context.Context, storyID int64) ([]*StoryRole, error) {
	var roles []*StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("story_id = ?", storyID).
		Find(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return roles, nil
}

func GetStoryRoleByID(ctx context.Context, roleID int64) (*StoryRole, error) {
	var role StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func UpdateStoryRole(ctx context.Context, roleID int64, needUpdateFields map[string]interface{}) error {
	if len(needUpdateFields) == 0 {
		return nil
	}
	needUpdateFields["update_at"] = time.Now()
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Updates(needUpdateFields).Error; err != nil {
		return err
	}
	return nil
}

func GetStoryRoleByName(ctx context.Context, name string, storyId int64) (*StoryRole, error) {
	var role StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("character_name = ?", name).
		Where("story_id = ?", storyId).
		First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func GetStoryRolesByName(ctx context.Context, name string, storyId int64, offset, number int) ([]*StoryRole, int64, error) {
	var roles []*StoryRole
	var total int64
	if err := DataBase().Model(&StoryRole{}).
		Where("story_id = ?", storyId).
		Where("character_name like ?", "%"+name+"%").
		Count(&total).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	if err := DataBase().Model(&StoryRole{}).
		Where("story_id = ?", storyId).
		Where("character_name like ?", "%"+name+"%").
		Offset(offset).
		Limit(number).
		Find(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return roles, total, nil
}

func IncreaseStoryRoleLikeCount(ctx context.Context, roleID int64, count int64) error {
	return DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("like_count", gorm.Expr("like_count + ?", count)).Error
}

func DecreaseStoryRoleLikeCount(ctx context.Context, roleID int64, count int64) error {
	return DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("like_count", gorm.Expr("like_count - ?", count)).Error
}

func IncreaseStoryRoleFollowCount(ctx context.Context, roleID int64, count int64) error {
	return DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("follow_count", gorm.Expr("follow_count + ?", count)).Error
}

func DecreaseStoryRoleFollowCount(ctx context.Context, roleID int64, count int64) error {
	return DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("follow_count", gorm.Expr("follow_count - ?", count)).Error
}

func IncreaseStoryRoleStoryboardNum(ctx context.Context, roleID int64, count int64) error {
	return DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("storyboard_num", gorm.Expr("storyboard_num + ?", count)).Error
}

func GetUserFollowedStoryRoleIds(ctx context.Context, userId int) ([]int64, error) {
	var roleIds []int64
	err := DataBase().Model(&WatchItem{}).
		Select("distinct role_id").
		Where("user_id = ? and deleted = 0 and watch_item_type = ? and watch_type = ?",
			userId, WatchItemTypeStoryRole, WatchTypeIsWatch).
		Pluck("role_id", &roleIds).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return roleIds, nil
}

// 根据角色ID列表获取角色列表
func GetStoryRolesByIDs(ctx context.Context, roleIds []int64) ([]*StoryRole, error) {
	var roles []*StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("id in (?)", roleIds).
		Find(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return roles, nil
}
