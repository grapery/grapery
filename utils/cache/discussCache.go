package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

const (
	// Cache TTL settings
	DiscussCacheTTL         = 300  // 5 minutes
	DiscussListCacheTTL     = 180  // 3 minutes
	DiscussUserInfoCacheTTL = 1800 // 30 minutes

	// Cache key prefixes
	DiscussPrefix         = "discuss:"
	DiscussListPrefix     = "discuss_list:"
	DiscussReplyPrefix    = "discuss_reply:"
	DiscussUserInfoPrefix = "discuss_user_info:"
)

// DiscussCache handles discuss-related caching operations
type DiscussCache struct{}

// GetDiscussKey generates cache key for discuss
func (c *DiscussCache) GetDiscussKey(discussID int64) string {
	return fmt.Sprintf("%s%d", DiscussPrefix, discussID)
}

// GetDiscussListKey generates cache key for discuss list
func (c *DiscussCache) GetDiscussListKey(storyID int64, offset, pageSize int64) string {
	return fmt.Sprintf("%s%d_%d_%d", DiscussListPrefix, storyID, offset, pageSize)
}

// GetDiscussReplyKey generates cache key for discuss replies
func (c *DiscussCache) GetDiscussReplyKey(discussID int64) string {
	return fmt.Sprintf("%s%d", DiscussReplyPrefix, discussID)
}

// GetUserInfoKey generates cache key for user info
func (c *DiscussCache) GetUserInfoKey(userID int64) string {
	return fmt.Sprintf("%s%d", DiscussUserInfoPrefix, userID)
}

// CacheDiscuss caches discuss detail
func (c *DiscussCache) CacheDiscuss(ctx context.Context, discussID int64, discuss interface{}) error {
	key := c.GetDiscussKey(discussID)
	data, err := json.Marshal(discuss)
	if err != nil {
		log.Errorf("Failed to marshal discuss for cache: %v", err)
		return err
	}
	return SetBytes(ctx, key, data, DiscussCacheTTL)
}

// GetCachedDiscuss retrieves cached discuss detail
func (c *DiscussCache) GetCachedDiscuss(ctx context.Context, discussID int64) (interface{}, error) {
	key := c.GetDiscussKey(discussID)
	data, err := GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var discuss interface{}
	err = json.Unmarshal(data, &discuss)
	if err != nil {
		log.Errorf("Failed to unmarshal cached discuss: %v", err)
		return nil, err
	}
	return discuss, nil
}

// CacheDiscussList caches discuss list
func (c *DiscussCache) CacheDiscussList(ctx context.Context, storyID int64, offset, pageSize int64, discusses interface{}) error {
	key := c.GetDiscussListKey(storyID, offset, pageSize)
	data, err := json.Marshal(discusses)
	if err != nil {
		log.Errorf("Failed to marshal discuss list for cache: %v", err)
		return err
	}
	return SetBytes(ctx, key, data, DiscussListCacheTTL)
}

// GetCachedDiscussList retrieves cached discuss list
func (c *DiscussCache) GetCachedDiscussList(ctx context.Context, storyID int64, offset, pageSize int64) (interface{}, error) {
	key := c.GetDiscussListKey(storyID, offset, pageSize)
	data, err := GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var discusses interface{}
	err = json.Unmarshal(data, &discusses)
	if err != nil {
		log.Errorf("Failed to unmarshal cached discuss list: %v", err)
		return nil, err
	}
	return discusses, nil
}

// CacheUserInfo caches user information
func (c *DiscussCache) CacheUserInfo(ctx context.Context, userID int64, user *models.User) error {
	key := c.GetUserInfoKey(userID)
	data, err := json.Marshal(user)
	if err != nil {
		log.Errorf("Failed to marshal user info for cache: %v", err)
		return err
	}
	return SetBytes(ctx, key, data, DiscussUserInfoCacheTTL)
}

// GetCachedUserInfo retrieves cached user information
func (c *DiscussCache) GetCachedUserInfo(ctx context.Context, userID int64) (*models.User, error) {
	key := c.GetUserInfoKey(userID)
	data, err := GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = json.Unmarshal(data, &user)
	if err != nil {
		log.Errorf("Failed to unmarshal cached user info: %v", err)
		return nil, err
	}
	return &user, nil
}

// InvalidateDiscussCache invalidates discuss cache
func (c *DiscussCache) InvalidateDiscussCache(ctx context.Context, discussID int64) error {
	key := c.GetDiscussKey(discussID)
	return DelCache(ctx, key)
}

// InvalidateDiscussListCache invalidates discuss list cache
func (c *DiscussCache) InvalidateDiscussListCache(ctx context.Context, storyID int64) error {
	// 删除该故事下的所有讨论列表缓存
	pattern := fmt.Sprintf("%s%d_*", DiscussListPrefix, storyID)
	return c.deleteByPattern(ctx, pattern)
}

// InvalidateDiscussRepliesCache invalidates discuss replies cache
func (c *DiscussCache) InvalidateDiscussRepliesCache(ctx context.Context, discussID int64) error {
	key := c.GetDiscussReplyKey(discussID)
	return DelCache(ctx, key)
}

// InvalidateUserInfoCache invalidates user info cache
func (c *DiscussCache) InvalidateUserInfoCache(ctx context.Context, userID int64) error {
	key := c.GetUserInfoKey(userID)
	return DelCache(ctx, key)
}

// deleteByPattern deletes cache entries by pattern
func (c *DiscussCache) deleteByPattern(ctx context.Context, pattern string) error {
	// 对于讨论列表缓存，我们可以删除常见的分页组合
	// 这是一个简化的实现，生产环境建议使用 Redis SCAN
	commonPageSizes := []int64{10, 20, 50, 100}
	commonOffsets := []int64{0, 10, 20, 50, 100}

	// 提取 storyID
	var baseID int64
	if _, err := fmt.Sscanf(pattern, "discuss_list:%d_*", &baseID); err == nil {
		// 删除讨论列表缓存
		for _, pageSize := range commonPageSizes {
			for _, offset := range commonOffsets {
				key := c.GetDiscussListKey(baseID, offset, pageSize)
				DelCache(ctx, key)
			}
		}
	}

	return nil
}

// NewDiscussCache creates a new discuss cache instance
func NewDiscussCache() *DiscussCache {
	return &DiscussCache{}
}

// Global instance
var discussCache = NewDiscussCache()

// GetDiscussCache returns the global discuss cache instance
func GetDiscussCache() *DiscussCache {
	return discussCache
}
