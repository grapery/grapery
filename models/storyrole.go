package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StoryRole 代表故事中的角色
// status: 1-有效, 0-无效
type StoryRole struct {
	ID uint `gorm:"primary_key,column:id;autoIncrement:1000000" json:"id,omitempty"`
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

	LikeCount     int64 `gorm:"column:like_count" json:"like_count,omitempty"`         // 点赞数
	FollowCount   int64 `gorm:"column:follow_count" json:"follow_count,omitempty"`     // 关注数
	StoryboardNum int64 `gorm:"column:storyboard_num" json:"storyboard_num,omitempty"` // 参与故事板数
	Version       int64 `gorm:"column:version" json:"version,omitempty"`               // 版本号
	BranchId      int64 `gorm:"column:branch_id" json:"branch_id,omitempty"`           // 分支ID

	CharacterDetail string `gorm:"column:character_detail" json:"character_detail,omitempty"` // 角色详细信息
	EditScope       int64  `gorm:"column:edit_scope" json:"edit_scope,omitempty"`             // 可编辑范围,1:自己可以编辑，2:小组可编辑
	ActScope        int64  `gorm:"column:act_scope" json:"act_scope,omitempty"`               // 可交互范围,1:自己可以交互，2:小组可交互,3:所有人可交互
	Resume          string `gorm:"column:resume" json:"resume,omitempty"`                     // 角色简历
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
	log.Log().Info("[GetStoryRoleByCreatorId] 查询角色", zap.Int64("creatorId", creatorId))
	if err := DataBase().Model(&StoryRole{}).
		Where("creator_id = ?", creatorId).
		Find(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryRoleByCreatorId] 未找到角色", zap.Int64("creatorId", creatorId))
			return nil, nil
		}
		log.Log().Error("[GetStoryRoleByCreatorId] 查询失败", zap.Error(err), zap.Int64("creatorId", creatorId))
		return nil, err
	}
	log.Log().Info("[GetStoryRoleByCreatorId] 查询成功", zap.Int64("creatorId", creatorId), zap.Int("count", len(roles)))
	return roles, nil
}

func CreateStoryRole(ctx context.Context, role *StoryRole) (int64, error) {
	log.Log().Info("[CreateStoryRole] 创建角色参数", zap.Any("role", role))
	if err := DataBase().Model(role).
		WithContext(ctx).
		Create(role).Error; err != nil {
		log.Log().Error("[CreateStoryRole] 创建角色失败", zap.Error(err), zap.Any("role", role))
		return 0, err
	}
	log.Log().Info("[CreateStoryRole] 创建角色成功", zap.Int64("role_id", int64(role.ID)))
	return int64(role.ID), nil
}

func GetStoryRole(ctx context.Context, storyID int64) ([]*StoryRole, error) {
	var roles []*StoryRole
	log.Log().Info("[GetStoryRole] 查询故事角色", zap.Int64("storyID", storyID))
	if err := DataBase().Model(&StoryRole{}).
		Where("story_id = ?", storyID).
		Find(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryRole] 未找到角色", zap.Int64("storyID", storyID))
			return nil, nil
		}
		log.Log().Error("[GetStoryRole] 查询失败", zap.Error(err), zap.Int64("storyID", storyID))
		return nil, err
	}
	log.Log().Info("[GetStoryRole] 查询成功", zap.Int64("storyID", storyID), zap.Int("count", len(roles)))
	return roles, nil
}

func GetStoryRoleByID(ctx context.Context, roleID int64) (*StoryRole, error) {
	var role StoryRole
	log.Log().Info("[GetStoryRoleByID] 查询角色", zap.Int64("roleID", roleID))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryRoleByID] 未找到角色", zap.Int64("roleID", roleID))
			return nil, nil
		}
		log.Log().Error("[GetStoryRoleByID] 查询失败", zap.Error(err), zap.Int64("roleID", roleID))
		return nil, err
	}
	log.Log().Info("[GetStoryRoleByID] 查询成功", zap.Int64("roleID", roleID))
	return &role, nil
}

func UpdateStoryRole(ctx context.Context, roleID int64, needUpdateFields map[string]interface{}) error {
	if len(needUpdateFields) == 0 {
		return nil
	}
	needUpdateFields["update_at"] = time.Now()
	log.Log().Info("[UpdateStoryRole] 更新角色", zap.Int64("roleID", roleID), zap.Any("fields", needUpdateFields))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Updates(needUpdateFields).Error; err != nil {
		log.Log().Error("[UpdateStoryRole] 更新失败", zap.Error(err), zap.Int64("roleID", roleID), zap.Any("fields", needUpdateFields))
		return err
	}
	log.Log().Info("[UpdateStoryRole] 更新成功", zap.Int64("roleID", roleID))
	return nil
}

func GetStoryRoleByName(ctx context.Context, name string, storyId int64) (*StoryRole, error) {
	var role StoryRole
	log.Log().Info("[GetStoryRoleByName] 查询角色", zap.String("name", name), zap.Int64("storyId", storyId))
	if err := DataBase().Model(&StoryRole{}).
		Where("character_name = ?", name).
		Where("story_id = ?", storyId).
		First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryRoleByName] 未找到角色", zap.String("name", name), zap.Int64("storyId", storyId))
			return nil, nil
		}
		log.Log().Error("[GetStoryRoleByName] 查询失败", zap.Error(err), zap.String("name", name), zap.Int64("storyId", storyId))
		return nil, err
	}
	log.Log().Info("[GetStoryRoleByName] 查询成功", zap.String("name", name), zap.Int64("storyId", storyId))
	return &role, nil
}

func GetStoryRolesByName(ctx context.Context, name string, storyId int64, offset, number int) ([]*StoryRole, int64, error) {
	var roles []*StoryRole
	var total int64
	log.Log().Info("[GetStoryRolesByName] 查询角色列表", zap.String("name", name), zap.Int64("storyId", storyId), zap.Int("offset", offset), zap.Int("number", number))
	query := DataBase().Model(&StoryRole{})
	if storyId > 0 {
		query = query.Where("story_id = ?", storyId)
	}
	if name != "" {
		query = query.Where("character_name like ?", "%"+name+"%")
	} else {
		return nil, 0, nil
	}
	if err := query.Count(&total).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryRolesByName] 未找到角色", zap.String("name", name), zap.Int64("storyId", storyId))
			return nil, 0, nil
		}
		log.Log().Error("[GetStoryRolesByName] 统计总数失败", zap.Error(err), zap.String("name", name), zap.Int64("storyId", storyId))
		return nil, 0, err
	}
	queryData := DataBase().Model(&StoryRole{})
	if storyId > 0 {
		queryData = queryData.Where("story_id = ?", storyId)
	}
	if name != "" {
		queryData = queryData.Where("character_name like ?", "%"+name+"%").
			Offset(offset).
			Limit(number)
	} else {
		return nil, 0, nil
	}
	if err := queryData.Find(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryRolesByName] 未找到角色", zap.String("name", name), zap.Int64("storyId", storyId))
			return nil, 0, nil
		}
		log.Log().Error("[GetStoryRolesByName] 查询失败", zap.Error(err), zap.String("name", name), zap.Int64("storyId", storyId))
		return nil, 0, err
	}
	log.Log().Info("[GetStoryRolesByName] 查询成功", zap.String("name", name), zap.Int64("storyId", storyId), zap.Int64("total", total), zap.Int("count", len(roles)))
	return roles, total, nil
}

func IncreaseStoryRoleLikeCount(ctx context.Context, roleID int64, count int64) error {
	log.Log().Info("[IncreaseStoryRoleLikeCount] 增加角色点赞数", zap.Int64("roleID", roleID), zap.Int64("count", count))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("like_count", gorm.Expr("like_count + ?", count)).Error; err != nil {
		log.Log().Error("[IncreaseStoryRoleLikeCount] 增加失败", zap.Error(err), zap.Int64("roleID", roleID), zap.Int64("count", count))
		return err
	}
	log.Log().Info("[IncreaseStoryRoleLikeCount] 增加成功", zap.Int64("roleID", roleID))
	return nil
}

func DecreaseStoryRoleLikeCount(ctx context.Context, roleID int64, count int64) error {
	log.Log().Info("[DecreaseStoryRoleLikeCount] 减少角色点赞数", zap.Int64("roleID", roleID), zap.Int64("count", count))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("like_count", gorm.Expr("like_count - ?", count)).Error; err != nil {
		log.Log().Error("[DecreaseStoryRoleLikeCount] 减少失败", zap.Error(err), zap.Int64("roleID", roleID), zap.Int64("count", count))
		return err
	}
	log.Log().Info("[DecreaseStoryRoleLikeCount] 减少成功", zap.Int64("roleID", roleID))
	return nil
}

func IncreaseStoryRoleFollowCount(ctx context.Context, roleID int64, count int64) error {
	log.Log().Info("[IncreaseStoryRoleFollowCount] 增加角色关注数", zap.Int64("roleID", roleID), zap.Int64("count", count))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("follow_count", gorm.Expr("follow_count + ?", count)).Error; err != nil {
		log.Log().Error("[IncreaseStoryRoleFollowCount] 增加失败", zap.Error(err), zap.Int64("roleID", roleID), zap.Int64("count", count))
		return err
	}
	log.Log().Info("[IncreaseStoryRoleFollowCount] 增加成功", zap.Int64("roleID", roleID))
	return nil
}

func DecreaseStoryRoleFollowCount(ctx context.Context, roleID int64, count int64) error {
	log.Log().Info("[DecreaseStoryRoleFollowCount] 减少角色关注数", zap.Int64("roleID", roleID), zap.Int64("count", count))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("follow_count", gorm.Expr("follow_count - ?", count)).Error; err != nil {
		log.Log().Error("[DecreaseStoryRoleFollowCount] 减少失败", zap.Error(err), zap.Int64("roleID", roleID), zap.Int64("count", count))
		return err
	}
	log.Log().Info("[DecreaseStoryRoleFollowCount] 减少成功", zap.Int64("roleID", roleID))
	return nil
}

func IncreaseStoryRoleStoryboardNum(ctx context.Context, roleID int64, count int64) error {
	log.Log().Info("[IncreaseStoryRoleStoryboardNum] 增加角色故事板数", zap.Int64("roleID", roleID), zap.Int64("count", count))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("storyboard_num", gorm.Expr("storyboard_num + ?", count)).Error; err != nil {
		log.Log().Error("[IncreaseStoryRoleStoryboardNum] 增加失败", zap.Error(err), zap.Int64("roleID", roleID), zap.Int64("count", count))
		return err
	}
	log.Log().Info("[IncreaseStoryRoleStoryboardNum] 增加成功", zap.Int64("roleID", roleID))
	return nil
}

func IncreaseStoryRoleStoryboardNumBatch(ctx context.Context, boardId int64, roleIds []int64, count int64) error {
	// 去重ID，避免重复操作
	roleIds = UniqueInt64s(roleIds)
	if len(roleIds) == 0 {
		log.Log().Warn("[IncreaseStoryRoleStoryboardNum] ID列表为空")
		return nil
	}

	log.Log().Info("[IncreaseStoryRoleStoryboardNum] 增加角色故事板数", zap.Int64("boardId", boardId), zap.Int64s("roleIds", roleIds), zap.Int64("count", count))
	if err := DataBase().Model(&StoryRole{}).
		Where("id in (?)", roleIds).
		WithContext(ctx).
		Update("storyboard_num", gorm.Expr("storyboard_num + ?", count)).Error; err != nil {
		log.Log().Error("[IncreaseStoryRoleStoryboardNum] 增加失败", zap.Error(err), zap.Int64("boardId", boardId), zap.Int64s("roleIds", roleIds), zap.Int64("count", count))
		return err
	}
	log.Log().Info("[IncreaseStoryRoleStoryboardNum] 增加成功", zap.Int64("boardId", boardId), zap.Int64s("roleIds", roleIds), zap.Int64("count", count))
	return nil
}

func DecreaseStoryRoleStoryboardNumBatch(ctx context.Context, boardId int64, roleIds []int64, count int64) error {
	// 去重ID，避免重复操作
	roleIds = UniqueInt64s(roleIds)
	if len(roleIds) == 0 {
		log.Log().Warn("[DecreaseStoryRoleStoryboardNum] ID列表为空")
		return nil
	}

	log.Log().Info("[DecreaseStoryRoleStoryboardNum] 减少角色故事板数", zap.Int64("boardId", boardId), zap.Int64s("roleIds", roleIds), zap.Int64("count", count))
	if err := DataBase().Model(&StoryRole{}).
		Where("id in (?)", roleIds).
		WithContext(ctx).
		Update("storyboard_num", gorm.Expr("storyboard_num - ?", count)).Error; err != nil {
		log.Log().Error("[DecreaseStoryRoleStoryboardNum] 减少失败", zap.Error(err), zap.Int64("boardId", boardId), zap.Int64s("roleIds", roleIds), zap.Int64("count", count))
		return err
	}
	log.Log().Info("[DecreaseStoryRoleStoryboardNum] 减少成功", zap.Int64("boardId", boardId), zap.Int64s("roleIds", roleIds), zap.Int64("count", count))
	return nil
}

func GetUserFollowedStoryRoleIds(ctx context.Context, userId int) ([]int64, error) {
	var roleIds []int64
	log.Log().Info("[GetUserFollowedStoryRoleIds] 查询用户关注的角色ID", zap.Int("userId", userId))
	err := DataBase().Model(&WatchItem{}).
		Select("distinct role_id").
		Where("user_id = ? and deleted = 0 and watch_item_type = ? and watch_type = ?",
			userId, WatchItemTypeStoryRole, WatchTypeIsWatching).
		Pluck("role_id", &roleIds).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetUserFollowedStoryRoleIds] 未找到关注角色", zap.Int("userId", userId))
			return nil, nil
		}
		log.Log().Error("[GetUserFollowedStoryRoleIds] 查询失败", zap.Error(err), zap.Int("userId", userId))
		return nil, err
	}
	log.Log().Info("[GetUserFollowedStoryRoleIds] 查询成功", zap.Int("userId", userId), zap.Int("count", len(roleIds)))
	return roleIds, nil
}

// 根据角色ID列表获取角色列表
func GetStoryRolesByIDs(ctx context.Context, roleIds []int64) ([]*StoryRole, error) {
	// 去重ID，避免重复查询
	roleIds = UniqueInt64s(roleIds)
	if len(roleIds) == 0 {
		log.Log().Warn("[GetStoryRolesByIDs] ID列表为空")
		return nil, nil
	}

	var roles []*StoryRole
	log.Log().Info("[GetStoryRolesByIDs] 批量查询角色", zap.Int64s("roleIds", roleIds))
	if err := DataBase().Model(&StoryRole{}).
		Where("id in (?)", roleIds).
		Find(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryRolesByIDs] 未找到角色", zap.Int64s("roleIds", roleIds))
			return nil, nil
		}
		log.Log().Error("[GetStoryRolesByIDs] 查询失败", zap.Error(err), zap.Int64s("roleIds", roleIds))
		return nil, err
	}
	log.Log().Info("[GetStoryRolesByIDs] 查询成功", zap.Int("count", len(roles)))
	return roles, nil
}

func UpdateStoryRolePosterURL(ctx context.Context, roleID int64, posterURL string) error {
	log.Log().Info("[UpdateStoryRolePosterURL] 更新角色海报", zap.Int64("roleID", roleID), zap.String("posterURL", posterURL))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		Where("status = ?", 1).
		WithContext(ctx).
		Update("poster_url", posterURL).Error; err != nil {
		log.Log().Error("[UpdateStoryRolePosterURL] 更新失败", zap.Error(err), zap.Int64("roleID", roleID), zap.String("posterURL", posterURL))
		return err
	}
	log.Log().Info("[UpdateStoryRolePosterURL] 更新成功", zap.Int64("roleID", roleID))
	return nil
}

func UpdateStoryRoleCharacterDetail(ctx context.Context, roleID int64, characterDetail string) error {
	log.Log().Info("[UpdateStoryRoleCharacterDetail] 更新角色详情", zap.Int64("roleID", roleID))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Update("character_detail", characterDetail).Error; err != nil {
		log.Log().Error("[UpdateStoryRoleCharacterDetail] 更新失败", zap.Error(err), zap.Int64("roleID", roleID))
		return err
	}
	log.Log().Info("[UpdateStoryRoleCharacterDetail] 更新成功", zap.Int64("roleID", roleID))
	return nil
}

func GetStoryRoleCharacterDetail(ctx context.Context, roleID int64) (string, error) {
	var characterDetail string
	log.Log().Info("[GetStoryRoleCharacterDetail] 查询角色详情", zap.Int64("roleID", roleID))
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		Pluck("character_detail", &characterDetail).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryRoleCharacterDetail] 未找到角色详情", zap.Int64("roleID", roleID))
			return "", nil
		}
		log.Log().Error("[GetStoryRoleCharacterDetail] 查询失败", zap.Error(err), zap.Int64("roleID", roleID))
		return "", err
	}
	log.Log().Info("[GetStoryRoleCharacterDetail] 查询成功", zap.Int64("roleID", roleID))
	return characterDetail, nil
}

// 新增：分页获取StoryRole列表
func GetStoryRoleList(ctx context.Context, offset, limit int) ([]*StoryRole, error) {
	var roles []*StoryRole
	log.Log().Info("[GetStoryRoleList] 分页获取角色列表", zap.Int("offset", offset), zap.Int("limit", limit))
	err := DataBase().Model(&StoryRole{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&roles).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("[GetStoryRoleList] 查询失败", zap.Error(err), zap.Int("offset", offset), zap.Int("limit", limit))
		return nil, err
	}
	log.Log().Info("[GetStoryRoleList] 查询成功", zap.Int("count", len(roles)))
	return roles, nil
}

// 新增：通过CharacterName唯一查询
func GetStoryRoleByCharacterName(ctx context.Context, name string, storyId int64) (*StoryRole, error) {
	role := &StoryRole{}
	log.Log().Info("[GetStoryRoleByCharacterName] 查询角色", zap.String("name", name), zap.Int64("storyId", storyId))
	err := DataBase().Model(role).
		WithContext(ctx).
		Where("character_name = ? and story_id = ?", name, storyId).
		First(role).Error
	if err != nil {
		log.Log().Error("[GetStoryRoleByCharacterName] 查询失败", zap.Error(err), zap.String("name", name), zap.Int64("storyId", storyId))
		return nil, err
	}
	log.Log().Info("[GetStoryRoleByCharacterName] 查询成功", zap.String("name", name), zap.Int64("storyId", storyId))
	return role, nil
}
