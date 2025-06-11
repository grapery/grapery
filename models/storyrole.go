package models

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// StoryRole 代表故事中的角色
// status: 1-有效, 0-无效
type StoryRole struct {
	IDBase
	StoryID              int64  `gorm:"column:story_id" json:"story_id,omitempty"`                           // 故事ID
	CharacterName        string `gorm:"column:character_name" json:"character_name,omitempty"`               // 角色名
	CharacterAvatar      string `gorm:"column:character_avatar" json:"character_avatar,omitempty"`           // 角色头像
	CharacterID          string `gorm:"column:character_id" json:"character_id,omitempty"`                   // 角色唯一ID
	CharacterType        string `gorm:"column:character_type" json:"character_type,omitempty"`               // 角色类型
	CharacterPrompt      string `gorm:"column:character_prompt" json:"character_prompt,omitempty"`           // 角色生成提示词
	CharacterRefImages   string `gorm:"column:character_ref_images" json:"character_ref_images,omitempty"`   // 角色参考图片
	CharacterDescription string `gorm:"column:character_description" json:"character_description,omitempty"` // 角色描述
	CreatorID            int64  `gorm:"column:creator_id" json:"creator_id,omitempty"`                       // 创建者ID
	Status               int    `gorm:"column:status" json:"status,omitempty"`                               // 状态
	LikeCount            int64  `gorm:"column:like_count" json:"like_count,omitempty"`                       // 点赞数
	FollowCount          int64  `gorm:"column:follow_count" json:"follow_count,omitempty"`                   // 关注数
	StoryboardNum        int64  `gorm:"column:storyboard_num" json:"storyboard_num,omitempty"`               // 参与故事板数
	Version              int64  `gorm:"column:version" json:"version,omitempty"`                             // 版本号
	BranchId             int64  `gorm:"column:branch_id" json:"branch_id,omitempty"`                         // 分支ID
	PosterURL            string `gorm:"column:poster_url" json:"poster_url,omitempty"`                       // 角色海报
	CharacterDetail      string `gorm:"column:character_detail" json:"character_detail,omitempty"`           // 角色详细信息
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

func UpdateStoryRolePosterURL(ctx context.Context, roleID int64, posterURL string) error {
	return DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		Where("status = ?", 1).
		WithContext(ctx).
		Update("poster_url", posterURL).Error
}

func UpdateStoryRoleCharacterDetail(ctx context.Context, roleID int64, characterDetail string) error {
	return DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("character_detail", characterDetail).Error
}

func GetStoryRoleCharacterDetail(ctx context.Context, roleID int64) (string, error) {
	var characterDetail string
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		Pluck("character_detail", &characterDetail).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return characterDetail, nil
}

// 新增：分页获取StoryRole列表
func GetStoryRoleList(ctx context.Context, offset, limit int) ([]*StoryRole, error) {
	var roles []*StoryRole
	err := DataBase().Model(&StoryRole{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&roles).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return roles, nil
}

// 新增：通过CharacterName唯一查询
func GetStoryRoleByCharacterName(ctx context.Context, name string, storyId int64) (*StoryRole, error) {
	role := &StoryRole{}
	err := DataBase().Model(role).
		WithContext(ctx).
		Where("character_name = ? and story_id = ?", name, storyId).
		First(role).Error
	if err != nil {
		return nil, err
	}
	return role, nil
}
