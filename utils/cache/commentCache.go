package cache

import (
	"context"
	"encoding/json"
	"fmt"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

const (
	// Cache TTL settings
	CommentListCacheTTL = 300  // 5 minutes
	UserInfoCacheTTL    = 1800 // 30 minutes

	// Cache key prefixes
	StoryCommentsPrefix      = "story_comments:"
	StoryBoardCommentsPrefix = "storyboard_comments:"
	CommentRepliesPrefix     = "comment_replies:"
	UserInfoPrefix           = "user_info:"
)

// CommentCache handles comment-related caching operations
type CommentCache struct{}

// GetStoryCommentsKey generates cache key for story comments
func (c *CommentCache) GetStoryCommentsKey(storyID int64, offset, pageSize int64) string {
	return fmt.Sprintf("%s%d_%d_%d", StoryCommentsPrefix, storyID, offset, pageSize)
}

// GetStoryBoardCommentsKey generates cache key for storyboard comments
func (c *CommentCache) GetStoryBoardCommentsKey(boardID int64, offset, pageSize int64) string {
	return fmt.Sprintf("%s%d_%d_%d", StoryBoardCommentsPrefix, boardID, offset, pageSize)
}

// GetCommentRepliesKey generates cache key for comment replies
func (c *CommentCache) GetCommentRepliesKey(commentID int64) string {
	return fmt.Sprintf("%s%d", CommentRepliesPrefix, commentID)
}

// GetUserInfoKey generates cache key for user info
func (c *CommentCache) GetUserInfoKey(userID int64) string {
	return fmt.Sprintf("%s%d", UserInfoPrefix, userID)
}

// CacheStoryComments caches story comments list
func (c *CommentCache) CacheStoryComments(ctx context.Context, storyID int64, offset, pageSize int64, comments []*api.StoryComment) error {
	key := c.GetStoryCommentsKey(storyID, offset, pageSize)
	data, err := json.Marshal(comments)
	if err != nil {
		log.Errorf("Failed to marshal story comments for cache: %v", err)
		return err
	}
	return SetBytes(ctx, key, data, CommentListCacheTTL)
}

// GetCachedStoryComments retrieves cached story comments list
func (c *CommentCache) GetCachedStoryComments(ctx context.Context, storyID int64, offset, pageSize int64) ([]*api.StoryComment, error) {
	key := c.GetStoryCommentsKey(storyID, offset, pageSize)
	data, err := GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var comments []*api.StoryComment
	err = json.Unmarshal(data, &comments)
	if err != nil {
		log.Errorf("Failed to unmarshal cached story comments: %v", err)
		return nil, err
	}
	return comments, nil
}

// CacheStoryBoardComments caches storyboard comments list
func (c *CommentCache) CacheStoryBoardComments(ctx context.Context, boardID int64, offset, pageSize int64, comments []*api.StoryComment) error {
	key := c.GetStoryBoardCommentsKey(boardID, offset, pageSize)
	data, err := json.Marshal(comments)
	if err != nil {
		log.Errorf("Failed to marshal storyboard comments for cache: %v", err)
		return err
	}
	return SetBytes(ctx, key, data, CommentListCacheTTL)
}

// GetCachedStoryBoardComments retrieves cached storyboard comments list
func (c *CommentCache) GetCachedStoryBoardComments(ctx context.Context, boardID int64, offset, pageSize int64) ([]*api.StoryComment, error) {
	key := c.GetStoryBoardCommentsKey(boardID, offset, pageSize)
	data, err := GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var comments []*api.StoryComment
	err = json.Unmarshal(data, &comments)
	if err != nil {
		log.Errorf("Failed to unmarshal cached storyboard comments: %v", err)
		return nil, err
	}
	return comments, nil
}

// CacheCommentReplies caches comment replies list
func (c *CommentCache) CacheCommentReplies(ctx context.Context, commentID int64, replies []*api.StoryComment) error {
	key := c.GetCommentRepliesKey(commentID)
	data, err := json.Marshal(replies)
	if err != nil {
		log.Errorf("Failed to marshal comment replies for cache: %v", err)
		return err
	}
	return SetBytes(ctx, key, data, CommentListCacheTTL)
}

// GetCachedCommentReplies retrieves cached comment replies list
func (c *CommentCache) GetCachedCommentReplies(ctx context.Context, commentID int64) ([]*api.StoryComment, error) {
	key := c.GetCommentRepliesKey(commentID)
	data, err := GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var replies []*api.StoryComment
	err = json.Unmarshal(data, &replies)
	if err != nil {
		log.Errorf("Failed to unmarshal cached comment replies: %v", err)
		return nil, err
	}
	return replies, nil
}

// CacheUserInfo caches user information
func (c *CommentCache) CacheUserInfo(ctx context.Context, userID int64, user *models.User) error {
	key := c.GetUserInfoKey(userID)
	data, err := json.Marshal(user)
	if err != nil {
		log.Errorf("Failed to marshal user info for cache: %v", err)
		return err
	}
	return SetBytes(ctx, key, data, UserInfoCacheTTL)
}

// GetCachedUserInfo retrieves cached user information
func (c *CommentCache) GetCachedUserInfo(ctx context.Context, userID int64) (*models.User, error) {
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

// InvalidateStoryCommentsCache invalidates story comments cache
func (c *CommentCache) InvalidateStoryCommentsCache(ctx context.Context, storyID int64) error {
	// We need to invalidate all possible combinations of offset and pageSize
	// For simplicity, we'll use a pattern-based deletion if supported, or delete common combinations
	pattern := fmt.Sprintf("%s%d_*", StoryCommentsPrefix, storyID)
	return c.deleteByPattern(ctx, pattern)
}

// InvalidateStoryBoardCommentsCache invalidates storyboard comments cache
func (c *CommentCache) InvalidateStoryBoardCommentsCache(ctx context.Context, boardID int64) error {
	pattern := fmt.Sprintf("%s%d_*", StoryBoardCommentsPrefix, boardID)
	return c.deleteByPattern(ctx, pattern)
}

// InvalidateCommentRepliesCache invalidates comment replies cache
func (c *CommentCache) InvalidateCommentRepliesCache(ctx context.Context, commentID int64) error {
	key := c.GetCommentRepliesKey(commentID)
	return DelCache(ctx, key)
}

// InvalidateUserInfoCache invalidates user info cache
func (c *CommentCache) InvalidateUserInfoCache(ctx context.Context, userID int64) error {
	key := c.GetUserInfoKey(userID)
	return DelCache(ctx, key)
}

// deleteByPattern deletes cache entries by pattern
func (c *CommentCache) deleteByPattern(ctx context.Context, pattern string) error {
	// 使用 Redis SCAN 命令实现模式匹配删除
	// 这里需要根据实际的 Redis 客户端实现
	// 暂时使用简单的键删除策略
	log.Warnf("Pattern-based cache deletion for pattern: %s", pattern)

	// 对于评论缓存，我们可以删除常见的分页组合
	// 这是一个简化的实现，生产环境建议使用 Redis SCAN
	commonPageSizes := []int64{10, 20, 50, 100}
	commonOffsets := []int64{0, 10, 20, 50, 100}

	// 提取 storyID 或 boardID
	var baseID int64
	if _, err := fmt.Sscanf(pattern, "story_comments:%d_*", &baseID); err == nil {
		// 删除故事评论缓存
		for _, pageSize := range commonPageSizes {
			for _, offset := range commonOffsets {
				key := c.GetStoryCommentsKey(baseID, offset, pageSize)
				DelCache(ctx, key)
			}
		}
	} else if _, err := fmt.Sscanf(pattern, "storyboard_comments:%d_*", &baseID); err == nil {
		// 删除故事板评论缓存
		for _, pageSize := range commonPageSizes {
			for _, offset := range commonOffsets {
				key := c.GetStoryBoardCommentsKey(baseID, offset, pageSize)
				DelCache(ctx, key)
			}
		}
	}

	return nil
}

// NewCommentCache creates a new comment cache instance
func NewCommentCache() *CommentCache {
	return &CommentCache{}
}

// Global instance
var commentCache = NewCommentCache()

// GetCommentCache returns the global comment cache instance
func GetCommentCache() *CommentCache {
	return commentCache
}
