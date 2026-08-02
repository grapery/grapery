package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/cache"
	"go.uber.org/zap"
)

// 故事角色缓存键前缀
const (
	StoryRoleDetailPrefix      = "story_role:detail:"       // 角色详情
	StoryRoleListPrefix        = "story_role:list:"         // 角色列表
	StoryRoleUserCreatedPrefix = "story_role:user_created:" // 用户创建的角色
	StoryRoleStoryboardsPrefix = "story_role:storyboards:"  // 角色故事板
	ChatContextPrefix          = "chat:context:"            // 聊天上下文
	ChatMessagePrefix          = "chat:message:"            // 聊天消息
	StoryRoleLikeCountPrefix   = "story_role:like_count:"   // 角色点赞数
	StoryRoleFollowCountPrefix = "story_role:follow_count:" // 角色关注数
)

// 缓存过期时间（秒）
const (
	StoryRoleDetailTTL      = 300 // 5分钟
	StoryRoleListTTL        = 180 // 3分钟
	StoryRoleUserCreatedTTL = 120 // 2分钟
	StoryRoleStoryboardsTTL = 300 // 5分钟
	ChatContextTTL          = 600 // 10分钟
	ChatMessageTTL          = 180 // 3分钟
	StoryRoleLikeCountTTL   = 60  // 1分钟
	StoryRoleFollowCountTTL = 60  // 1分钟
)

// StoryRoleCache 故事角色缓存管理器
type StoryRoleCache struct{}

// NewStoryRoleCache 创建故事角色缓存管理器
func NewStoryRoleCache() *StoryRoleCache {
	return &StoryRoleCache{}
}

// GetStoryRoleDetail 获取角色详情缓存
func (src *StoryRoleCache) GetStoryRoleDetail(ctx context.Context, roleID int64) (*models.StoryRole, error) {
	key := fmt.Sprintf("%s%d", StoryRoleDetailPrefix, roleID)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var role models.StoryRole
	if err := json.Unmarshal(data, &role); err != nil {
		logger.Error("unmarshal story role detail failed", zap.Error(err))
		return nil, err
	}

	return &role, nil
}

// SetStoryRoleDetail 设置角色详情缓存
func (src *StoryRoleCache) SetStoryRoleDetail(ctx context.Context, roleID int64, role *models.StoryRole) error {
	key := fmt.Sprintf("%s%d", StoryRoleDetailPrefix, roleID)
	data, err := json.Marshal(role)
	if err != nil {
		logger.Error("marshal story role detail failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryRoleDetailTTL)
}

// DeleteStoryRoleDetail 删除角色详情缓存
func (src *StoryRoleCache) DeleteStoryRoleDetail(ctx context.Context, roleID int64) error {
	key := fmt.Sprintf("%s%d", StoryRoleDetailPrefix, roleID)
	return cache.DelCache(ctx, key)
}

// GetStoryRoleList 获取角色列表缓存
func (src *StoryRoleCache) GetStoryRoleList(ctx context.Context, storyID int64, searchKey string, offset, pageSize int) ([]*models.StoryRole, int64, error) {
	key := fmt.Sprintf("%s%d:%s:%d:%d", StoryRoleListPrefix, storyID, searchKey, offset, pageSize)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, 0, err
	}

	var result struct {
		Roles []*models.StoryRole `json:"roles"`
		Total int64               `json:"total"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		logger.Error("unmarshal story role list failed", zap.Error(err))
		return nil, 0, err
	}

	return result.Roles, result.Total, nil
}

// SetStoryRoleList 设置角色列表缓存
func (src *StoryRoleCache) SetStoryRoleList(ctx context.Context, storyID int64, searchKey string, offset, pageSize int, roles []*models.StoryRole, total int64) error {
	key := fmt.Sprintf("%s%d:%s:%d:%d", StoryRoleListPrefix, storyID, searchKey, offset, pageSize)

	result := struct {
		Roles []*models.StoryRole `json:"roles"`
		Total int64               `json:"total"`
	}{
		Roles: roles,
		Total: total,
	}

	data, err := json.Marshal(result)
	if err != nil {
		logger.Error("marshal story role list failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryRoleListTTL)
}

// DeleteStoryRoleList 删除角色列表缓存
func (src *StoryRoleCache) DeleteStoryRoleList(ctx context.Context, storyID int64) error {
	key := fmt.Sprintf("%s%d", StoryRoleListPrefix, storyID)
	return cache.DelCache(ctx, key)
}

// GetUserCreatedRoles 获取用户创建的角色缓存
func (src *StoryRoleCache) GetUserCreatedRoles(ctx context.Context, userID, storyID int64, offset, pageSize int) ([]*models.StoryRole, int64, error) {
	key := fmt.Sprintf("%s%d:%d:%d:%d", StoryRoleUserCreatedPrefix, userID, storyID, offset, pageSize)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, 0, err
	}

	var result struct {
		Roles []*models.StoryRole `json:"roles"`
		Total int64               `json:"total"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		logger.Error("unmarshal user created roles failed", zap.Error(err))
		return nil, 0, err
	}

	return result.Roles, result.Total, nil
}

// SetUserCreatedRoles 设置用户创建的角色缓存
func (src *StoryRoleCache) SetUserCreatedRoles(ctx context.Context, userID, storyID int64, offset, pageSize int, roles []*models.StoryRole, total int64) error {
	key := fmt.Sprintf("%s%d:%d:%d:%d", StoryRoleUserCreatedPrefix, userID, storyID, offset, pageSize)

	result := struct {
		Roles []*models.StoryRole `json:"roles"`
		Total int64               `json:"total"`
	}{
		Roles: roles,
		Total: total,
	}

	data, err := json.Marshal(result)
	if err != nil {
		logger.Error("marshal user created roles failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryRoleUserCreatedTTL)
}

// DeleteUserCreatedRoles 删除用户创建的角色缓存
func (src *StoryRoleCache) DeleteUserCreatedRoles(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%s%d", StoryRoleUserCreatedPrefix, userID)
	return cache.DelCache(ctx, key)
}

// GetStoryRoleStoryboards 获取角色故事板缓存
func (src *StoryRoleCache) GetStoryRoleStoryboards(ctx context.Context, roleID int64, offset, pageSize int) ([]*models.StoryBoard, int64, error) {
	key := fmt.Sprintf("%s%d:%d:%d", StoryRoleStoryboardsPrefix, roleID, offset, pageSize)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, 0, err
	}

	var result struct {
		Boards []*models.StoryBoard `json:"boards"`
		Total  int64                `json:"total"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		logger.Error("unmarshal story role storyboards failed", zap.Error(err))
		return nil, 0, err
	}

	return result.Boards, result.Total, nil
}

// SetStoryRoleStoryboards 设置角色故事板缓存
func (src *StoryRoleCache) SetStoryRoleStoryboards(ctx context.Context, roleID int64, offset, pageSize int, boards []*models.StoryBoard, total int64) error {
	key := fmt.Sprintf("%s%d:%d:%d", StoryRoleStoryboardsPrefix, roleID, offset, pageSize)

	result := struct {
		Boards []*models.StoryBoard `json:"boards"`
		Total  int64                `json:"total"`
	}{
		Boards: boards,
		Total:  total,
	}

	data, err := json.Marshal(result)
	if err != nil {
		logger.Error("marshal story role storyboards failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, StoryRoleStoryboardsTTL)
}

// DeleteStoryRoleStoryboards 删除角色故事板缓存
func (src *StoryRoleCache) DeleteStoryRoleStoryboards(ctx context.Context, roleID int64) error {
	key := fmt.Sprintf("%s%d", StoryRoleStoryboardsPrefix, roleID)
	return cache.DelCache(ctx, key)
}

// GetChatContext 获取聊天上下文缓存
func (src *StoryRoleCache) GetChatContext(ctx context.Context, userID, roleID int64) (*models.ChatContext, error) {
	key := fmt.Sprintf("%s%d:%d", ChatContextPrefix, userID, roleID)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var chatCtx models.ChatContext
	if err := json.Unmarshal(data, &chatCtx); err != nil {
		logger.Error("unmarshal chat context failed", zap.Error(err))
		return nil, err
	}

	return &chatCtx, nil
}

// SetChatContext 设置聊天上下文缓存
func (src *StoryRoleCache) SetChatContext(ctx context.Context, userID, roleID int64, chatCtx *models.ChatContext) error {
	key := fmt.Sprintf("%s%d:%d", ChatContextPrefix, userID, roleID)
	data, err := json.Marshal(chatCtx)
	if err != nil {
		logger.Error("marshal chat context failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, ChatContextTTL)
}

// DeleteChatContext 删除聊天上下文缓存
func (src *StoryRoleCache) DeleteChatContext(ctx context.Context, userID, roleID int64) error {
	key := fmt.Sprintf("%s%d:%d", ChatContextPrefix, userID, roleID)
	return cache.DelCache(ctx, key)
}

// GetStoryRoleLikeCount 获取角色点赞数缓存
func (src *StoryRoleCache) GetStoryRoleLikeCount(ctx context.Context, roleID int64) (int64, error) {
	key := fmt.Sprintf("%s%d", StoryRoleLikeCountPrefix, roleID)
	countStr, err := cache.GetString(ctx, key)
	if err != nil {
		return 0, err
	}
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// SetStoryRoleLikeCount 设置角色点赞数缓存
func (src *StoryRoleCache) SetStoryRoleLikeCount(ctx context.Context, roleID int64, count int64) error {
	key := fmt.Sprintf("%s%d", StoryRoleLikeCountPrefix, roleID)
	return cache.SetString(ctx, key, strconv.FormatInt(count, 10), StoryRoleLikeCountTTL)
}

// DeleteStoryRoleLikeCount 删除角色点赞数缓存
func (src *StoryRoleCache) DeleteStoryRoleLikeCount(ctx context.Context, roleID int64) error {
	key := fmt.Sprintf("%s%d", StoryRoleLikeCountPrefix, roleID)
	return cache.DelCache(ctx, key)
}

// GetStoryRoleFollowCount 获取角色关注数缓存
func (src *StoryRoleCache) GetStoryRoleFollowCount(ctx context.Context, roleID int64) (int64, error) {
	key := fmt.Sprintf("%s%d", StoryRoleFollowCountPrefix, roleID)
	countStr, err := cache.GetString(ctx, key)
	if err != nil {
		return 0, err
	}
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// SetStoryRoleFollowCount 设置角色关注数缓存
func (src *StoryRoleCache) SetStoryRoleFollowCount(ctx context.Context, roleID int64, count int64) error {
	key := fmt.Sprintf("%s%d", StoryRoleFollowCountPrefix, roleID)
	return cache.SetString(ctx, key, strconv.FormatInt(count, 10), StoryRoleFollowCountTTL)
}

// DeleteStoryRoleFollowCount 删除角色关注数缓存
func (src *StoryRoleCache) DeleteStoryRoleFollowCount(ctx context.Context, roleID int64) error {
	key := fmt.Sprintf("%s%d", StoryRoleFollowCountPrefix, roleID)
	return cache.DelCache(ctx, key)
}

// InvalidateStoryRoleCache 清除角色相关所有缓存
func (src *StoryRoleCache) InvalidateStoryRoleCache(ctx context.Context, roleID int64) error {
	// 删除角色详情缓存
	if err := src.DeleteStoryRoleDetail(ctx, roleID); err != nil {
		logger.Warn("delete story role detail cache failed", zap.Error(err))
	}

	// 删除角色点赞数缓存
	if err := src.DeleteStoryRoleLikeCount(ctx, roleID); err != nil {
		logger.Warn("delete story role like count cache failed", zap.Error(err))
	}

	// 删除角色关注数缓存
	if err := src.DeleteStoryRoleFollowCount(ctx, roleID); err != nil {
		logger.Warn("delete story role follow count cache failed", zap.Error(err))
	}

	// 删除角色故事板缓存
	if err := src.DeleteStoryRoleStoryboards(ctx, roleID); err != nil {
		logger.Warn("delete story role storyboards cache failed", zap.Error(err))
	}

	return nil
}

// InvalidateStoryRoleListCache 清除角色列表相关缓存
func (src *StoryRoleCache) InvalidateStoryRoleListCache(ctx context.Context, storyID int64) error {
	// 删除角色列表缓存
	if err := src.DeleteStoryRoleList(ctx, storyID); err != nil {
		logger.Warn("delete story role list cache failed", zap.Error(err))
	}

	return nil
}

// InvalidateUserCreatedRolesCache 清除用户创建角色相关缓存
func (src *StoryRoleCache) InvalidateUserCreatedRolesCache(ctx context.Context, userID int64) error {
	// 删除用户创建的角色缓存
	if err := src.DeleteUserCreatedRoles(ctx, userID); err != nil {
		logger.Warn("delete user created roles cache failed", zap.Error(err))
	}

	return nil
}
