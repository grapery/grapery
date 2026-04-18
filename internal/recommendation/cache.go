package recommendation

import (
	"context"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
)

const (
	FeedTypeFragments   = "fragments"
	FeedTypeStoryboards = "storyboards"
)

func FeedCacheKey(feedType, userID string, page, limit int) string {
	return fmt.Sprintf("reco:%s:for_you:%s:%d:%d", feedType, userID, page, limit)
}

func FeedIndexKey(feedType, userID string) string {
	return fmt.Sprintf("reco:index:%s:for_you:%s", feedType, userID)
}

func TrackFeedCacheKey(ctx context.Context, c cache.Cache, feedType, userID, key string, ttl time.Duration) {
	if c == nil || userID == "" || key == "" {
		return
	}
	indexKey := FeedIndexKey(feedType, userID)
	_ = c.SAdd(ctx, indexKey, key)
	_ = c.Expire(ctx, indexKey, ttl+30*time.Second)
}

func InvalidateFeedCache(ctx context.Context, c cache.Cache, feedType, userID string) {
	if c == nil || userID == "" {
		return
	}
	indexKey := FeedIndexKey(feedType, userID)
	keys, err := c.SMembers(ctx, indexKey)
	if err == nil && len(keys) > 0 {
		_ = c.Delete(ctx, keys...)
	}
	_ = c.Delete(ctx, indexKey)
}

func InvalidateAllForUser(ctx context.Context, c cache.Cache, userID string) {
	InvalidateFeedCache(ctx, c, FeedTypeFragments, userID)
	InvalidateFeedCache(ctx, c, FeedTypeStoryboards, userID)
}
