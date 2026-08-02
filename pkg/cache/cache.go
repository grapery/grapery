package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/cache"
	"go.uber.org/zap"
)

var logger, _ = zap.NewDevelopment()

// 缓存键前缀
const (
	GroupInfoPrefix    = "group:info:"    // 群组基本信息
	GroupProfilePrefix = "group:profile:" // 群组详细信息
	GroupMembersPrefix = "group:members:" // 群组成员列表
	UserGroupPrefix    = "user:group:"    // 用户群组状态
	GroupStoryPrefix   = "group:story:"   // 群组故事列表
	GroupActivePrefix  = "group:active:"  // 群组活动列表
	GroupSearchPrefix  = "group:search:"  // 群组搜索结果
)

// 缓存过期时间（秒）
const (
	GroupInfoTTL    = 300 // 5分钟
	GroupProfileTTL = 600 // 10分钟
	GroupMembersTTL = 180 // 3分钟
	UserGroupTTL    = 120 // 2分钟
	GroupStoryTTL   = 60  // 1分钟
	GroupActiveTTL  = 300 // 5分钟
	GroupSearchTTL  = 180 // 3分钟
)

// GroupCache 群组缓存管理器
type GroupCache struct{}

// NewGroupCache 创建群组缓存管理器
func NewGroupCache() *GroupCache {
	return &GroupCache{}
}

// GetGroupInfo 获取群组信息缓存
func (gc *GroupCache) GetGroupInfo(ctx context.Context, groupID int64) (*models.Group, error) {
	key := fmt.Sprintf("%s%d", GroupInfoPrefix, groupID)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var group models.Group
	if err := json.Unmarshal(data, &group); err != nil {
		logger.Error("unmarshal group info failed", zap.Error(err))
		return nil, err
	}

	return &group, nil
}

// SetGroupInfo 设置群组信息缓存
func (gc *GroupCache) SetGroupInfo(ctx context.Context, groupID int64, group *models.Group) error {
	key := fmt.Sprintf("%s%d", GroupInfoPrefix, groupID)
	data, err := json.Marshal(group)
	if err != nil {
		logger.Error("marshal group info failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, GroupInfoTTL)
}

// DeleteGroupInfo 删除群组信息缓存
func (gc *GroupCache) DeleteGroupInfo(ctx context.Context, groupID int64) error {
	key := fmt.Sprintf("%s%d", GroupInfoPrefix, groupID)
	return cache.DelCache(ctx, key)
}

// GetGroupProfile 获取群组详细信息缓存
func (gc *GroupCache) GetGroupProfile(ctx context.Context, groupID int64) (*models.GroupProfile, error) {
	key := fmt.Sprintf("%s%d", GroupProfilePrefix, groupID)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var profile models.GroupProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		logger.Error("unmarshal group profile failed", zap.Error(err))
		return nil, err
	}

	return &profile, nil
}

// SetGroupProfile 设置群组详细信息缓存
func (gc *GroupCache) SetGroupProfile(ctx context.Context, groupID int64, profile *models.GroupProfile) error {
	key := fmt.Sprintf("%s%d", GroupProfilePrefix, groupID)
	data, err := json.Marshal(profile)
	if err != nil {
		logger.Error("marshal group profile failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, GroupProfileTTL)
}

// DeleteGroupProfile 删除群组详细信息缓存
func (gc *GroupCache) DeleteGroupProfile(ctx context.Context, groupID int64) error {
	key := fmt.Sprintf("%s%d", GroupProfilePrefix, groupID)
	return cache.DelCache(ctx, key)
}

// GetGroupMembers 获取群组成员列表缓存
func (gc *GroupCache) GetGroupMembers(ctx context.Context, groupID int64, offset, pageSize int) ([]*models.User, error) {
	key := fmt.Sprintf("%s%d:%d:%d", GroupMembersPrefix, groupID, offset, pageSize)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var members []*models.User
	if err := json.Unmarshal(data, &members); err != nil {
		logger.Error("unmarshal group members failed", zap.Error(err))
		return nil, err
	}

	return members, nil
}

// SetGroupMembers 设置群组成员列表缓存
func (gc *GroupCache) SetGroupMembers(ctx context.Context, groupID int64, offset, pageSize int, members []*models.User) error {
	key := fmt.Sprintf("%s%d:%d:%d", GroupMembersPrefix, groupID, offset, pageSize)
	data, err := json.Marshal(members)
	if err != nil {
		logger.Error("marshal group members failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, GroupMembersTTL)
}

// DeleteGroupMembers 删除群组成员列表缓存
func (gc *GroupCache) DeleteGroupMembers(ctx context.Context, groupID int64) error {
	// 注意：这里需要实现模式删除，简化处理
	key := fmt.Sprintf("%s%d", GroupMembersPrefix, groupID)
	return cache.DelCache(ctx, key)
}

// GetUserGroupStatus 获取用户在群组中的状态缓存
func (gc *GroupCache) GetUserGroupStatus(ctx context.Context, userID, groupID int64) (bool, error) {
	key := fmt.Sprintf("%s%d:%d", UserGroupPrefix, userID, groupID)
	status, err := cache.GetString(ctx, key)
	if err != nil {
		return false, err
	}

	return status == "1", nil
}

// SetUserGroupStatus 设置用户在群组中的状态缓存
func (gc *GroupCache) SetUserGroupStatus(ctx context.Context, userID, groupID int64, isInGroup bool) error {
	key := fmt.Sprintf("%s%d:%d", UserGroupPrefix, userID, groupID)
	status := "0"
	if isInGroup {
		status = "1"
	}

	return cache.SetString(ctx, key, status, UserGroupTTL)
}

// DeleteUserGroupStatus 删除用户在群组中的状态缓存
func (gc *GroupCache) DeleteUserGroupStatus(ctx context.Context, userID, groupID int64) error {
	key := fmt.Sprintf("%s%d:%d", UserGroupPrefix, userID, groupID)
	return cache.DelCache(ctx, key)
}

// GetGroupStories 获取群组故事列表缓存
func (gc *GroupCache) GetGroupStories(ctx context.Context, groupID int64, page, pageSize int) ([]*models.Story, int64, bool, error) {
	key := fmt.Sprintf("%s%d:%d:%d", GroupStoryPrefix, groupID, page, pageSize)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, 0, false, err
	}

	var result struct {
		Stories []*models.Story `json:"stories"`
		Total   int64           `json:"total"`
		HasMore bool            `json:"has_more"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		logger.Error("unmarshal group stories failed", zap.Error(err))
		return nil, 0, false, err
	}

	return result.Stories, result.Total, result.HasMore, nil
}

// SetGroupStories 设置群组故事列表缓存
func (gc *GroupCache) SetGroupStories(ctx context.Context, groupID int64, page, pageSize int, stories []*models.Story, total int64, hasMore bool) error {
	key := fmt.Sprintf("%s%d:%d:%d", GroupStoryPrefix, groupID, page, pageSize)

	result := struct {
		Stories []*models.Story `json:"stories"`
		Total   int64           `json:"total"`
		HasMore bool            `json:"has_more"`
	}{
		Stories: stories,
		Total:   total,
		HasMore: hasMore,
	}

	data, err := json.Marshal(result)
	if err != nil {
		logger.Error("marshal group stories failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, GroupStoryTTL)
}

// DeleteGroupStories 删除群组故事列表缓存
func (gc *GroupCache) DeleteGroupStories(ctx context.Context, groupID int64) error {
	key := fmt.Sprintf("%s%d", GroupStoryPrefix, groupID)
	return cache.DelCache(ctx, key)
}

// GetGroupSearch 获取群组搜索结果缓存
func (gc *GroupCache) GetGroupSearch(ctx context.Context, name string, offset, pageSize int) ([]*models.Group, int64, error) {
	key := fmt.Sprintf("%s%s:%d:%d", GroupSearchPrefix, name, offset, pageSize)
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		return nil, 0, err
	}

	var result struct {
		Groups []*models.Group `json:"groups"`
		Total  int64           `json:"total"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		logger.Error("unmarshal group search failed", zap.Error(err))
		return nil, 0, err
	}

	return result.Groups, result.Total, nil
}

// SetGroupSearch 设置群组搜索结果缓存
func (gc *GroupCache) SetGroupSearch(ctx context.Context, name string, offset, pageSize int, groups []*models.Group, total int64) error {
	key := fmt.Sprintf("%s%s:%d:%d", GroupSearchPrefix, name, offset, pageSize)

	result := struct {
		Groups []*models.Group `json:"groups"`
		Total  int64           `json:"total"`
	}{
		Groups: groups,
		Total:  total,
	}

	data, err := json.Marshal(result)
	if err != nil {
		logger.Error("marshal group search failed", zap.Error(err))
		return err
	}

	return cache.SetBytes(ctx, key, data, GroupSearchTTL)
}

// DeleteGroupSearch 删除群组搜索结果缓存
func (gc *GroupCache) DeleteGroupSearch(ctx context.Context, name string) error {
	key := fmt.Sprintf("%s%s", GroupSearchPrefix, name)
	return cache.DelCache(ctx, key)
}

// InvalidateGroupCache 清除群组相关所有缓存
func (gc *GroupCache) InvalidateGroupCache(ctx context.Context, groupID int64) error {
	// 删除群组信息缓存
	if err := gc.DeleteGroupInfo(ctx, groupID); err != nil {
		logger.Warn("delete group info cache failed", zap.Error(err))
	}

	// 删除群组详细信息缓存
	if err := gc.DeleteGroupProfile(ctx, groupID); err != nil {
		logger.Warn("delete group profile cache failed", zap.Error(err))
	}

	// 删除群组成员缓存
	if err := gc.DeleteGroupMembers(ctx, groupID); err != nil {
		logger.Warn("delete group members cache failed", zap.Error(err))
	}

	// 删除群组故事缓存
	if err := gc.DeleteGroupStories(ctx, groupID); err != nil {
		logger.Warn("delete group stories cache failed", zap.Error(err))
	}

	return nil
}

// InvalidateUserGroupCache 清除用户群组相关缓存
func (gc *GroupCache) InvalidateUserGroupCache(ctx context.Context, userID, groupID int64) error {
	// 删除用户群组状态缓存
	if err := gc.DeleteUserGroupStatus(ctx, userID, groupID); err != nil {
		logger.Warn("delete user group status cache failed", zap.Error(err))
	}

	// 删除群组成员缓存（因为成员状态发生变化）
	if err := gc.DeleteGroupMembers(ctx, groupID); err != nil {
		logger.Warn("delete group members cache failed", zap.Error(err))
	}

	return nil
}
