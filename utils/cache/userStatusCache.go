package cache

import (
	"context"
	"encoding/json"
	"fmt"

	api "github.com/grapery/common-protoc/gen"
	log "github.com/sirupsen/logrus"
)

const (
	// 用户状态缓存TTL设置
	UserStatusCacheTTL = 600 // 10分钟

	// 缓存键前缀
	StoryRoleUserStatusPrefix  = "story_role_user_status:"
	StoryUserStatusPrefix      = "story_user_status:"
	StoryboardUserStatusPrefix = "storyboard_user_status:"
	GroupUserStatusPrefix      = "group_user_status:"
)

// UserStatusCache 处理用户状态相关的缓存操作
type UserStatusCache struct{}

// GetStoryRoleUserStatusKey 生成故事角色用户状态缓存键
func (c *UserStatusCache) GetStoryRoleUserStatusKey(roleID, userID int64) string {
	return fmt.Sprintf("%s%d_%d", StoryRoleUserStatusPrefix, roleID, userID)
}

// GetStoryUserStatusKey 生成故事用户状态缓存键
func (c *UserStatusCache) GetStoryUserStatusKey(storyID, userID int64) string {
	return fmt.Sprintf("%s%d_%d", StoryUserStatusPrefix, storyID, userID)
}

// GetStoryboardUserStatusKey 生成分镜用户状态缓存键
func (c *UserStatusCache) GetStoryboardUserStatusKey(storyboardID, userID int64) string {
	return fmt.Sprintf("%s%d_%d", StoryboardUserStatusPrefix, storyboardID, userID)
}

// GetGroupUserStatusKey 生成小组用户状态缓存键
func (c *UserStatusCache) GetGroupUserStatusKey(groupID, userID int64) string {
	return fmt.Sprintf("%s%d_%d", GroupUserStatusPrefix, groupID, userID)
}

// CacheUserStatus 缓存用户状态
func (c *UserStatusCache) CacheUserStatus(ctx context.Context, key string, status *api.WhatCurrentUserStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		log.Errorf("缓存用户状态序列化失败: %v", err)
		return err
	}
	return SetBytes(ctx, key, data, UserStatusCacheTTL)
}

// GetCachedUserStatus 获取缓存的用户状态
func (c *UserStatusCache) GetCachedUserStatus(ctx context.Context, key string) (*api.WhatCurrentUserStatus, error) {
	data, err := GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var status api.WhatCurrentUserStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		log.Errorf("反序列化缓存用户状态失败: %v", err)
		return nil, err
	}
	return &status, nil
}

// InvalidateStoryRoleUserStatusCache 清除故事角色用户状态缓存
func (c *UserStatusCache) InvalidateStoryRoleUserStatusCache(ctx context.Context, roleID, userID int64) error {
	key := c.GetStoryRoleUserStatusKey(roleID, userID)
	return DelCache(ctx, key)
}

// InvalidateStoryUserStatusCache 清除故事用户状态缓存
func (c *UserStatusCache) InvalidateStoryUserStatusCache(ctx context.Context, storyID, userID int64) error {
	key := c.GetStoryUserStatusKey(storyID, userID)
	return DelCache(ctx, key)
}

// InvalidateStoryboardUserStatusCache 清除分镜用户状态缓存
func (c *UserStatusCache) InvalidateStoryboardUserStatusCache(ctx context.Context, storyboardID, userID int64) error {
	key := c.GetStoryboardUserStatusKey(storyboardID, userID)
	return DelCache(ctx, key)
}

// InvalidateGroupUserStatusCache 清除小组用户状态缓存
func (c *UserStatusCache) InvalidateGroupUserStatusCache(ctx context.Context, groupID, userID int64) error {
	key := c.GetGroupUserStatusKey(groupID, userID)
	return DelCache(ctx, key)
}

// NewUserStatusCache 创建新的用户状态缓存实例
func NewUserStatusCache() *UserStatusCache {
	return &UserStatusCache{}
}

// 全局实例
var userStatusCache = NewUserStatusCache()

// GetUserStatusCache 返回全局用户状态缓存实例
func GetUserStatusCache() *UserStatusCache {
	return userStatusCache
}
