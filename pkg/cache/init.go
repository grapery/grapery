package cache

import (
	"sync"
)

var (
	groupCache      *GroupCache
	storyRoleCache  *StoryRoleCache
	storyBoardCache *StoryBoardCache
	once            sync.Once
)

// InitCache 初始化缓存管理器
func InitCache() {
	once.Do(func() {
		groupCache = NewGroupCache()
		storyRoleCache = NewStoryRoleCache()
		storyBoardCache = NewStoryBoardCache()
	})
}

// GetGroupCache 获取群组缓存管理器
func GetGroupCache() *GroupCache {
	if groupCache == nil {
		InitCache()
	}
	return groupCache
}

// GetStoryRoleCache 获取故事角色缓存管理器
func GetStoryRoleCache() *StoryRoleCache {
	if storyRoleCache == nil {
		InitCache()
	}
	return storyRoleCache
}

// GetStoryBoardCache 获取故事板缓存管理器
func GetStoryBoardCache() *StoryBoardCache {
	if storyBoardCache == nil {
		InitCache()
	}
	return storyBoardCache
}
