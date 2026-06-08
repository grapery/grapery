package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"go.uber.org/zap"
)

const (
	socialListMetaFieldGen = "social_list_gen"
	socialListMetaTTL      = 7 * 24 * time.Hour
)

func (s *Service) readSocialListCacheGen(ctx context.Context, userID string) int64 {
	c := s.getCache()
	if c == nil || userID == "" {
		return 0
	}
	val, err := c.HGet(ctx, cache.UserSocialListMetaKey(userID), socialListMetaFieldGen)
	if err != nil || val == "" {
		return 0
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (s *Service) bumpSocialListCacheGen(ctx context.Context, userID string) {
	c := s.getCache()
	if c == nil || userID == "" {
		return
	}
	key := cache.UserSocialListMetaKey(userID)
	if _, err := c.HIncrBy(ctx, key, socialListMetaFieldGen, 1); err != nil {
		s.logger.Warn("failed to bump social list cache gen",
			zap.String("userID", userID),
			zap.Error(err))
		return
	}
	_ = c.Expire(ctx, key, socialListMetaTTL)
}

// setSocialListCacheIfUnchanged writes a paginated social list only when no concurrent
// follow/unfollow/block mutation happened during the DB read and no other reader populated the key.
func (s *Service) setSocialListCacheIfUnchanged(ctx context.Context, userID, cacheKey string, value interface{}, ttl time.Duration, genAtRead int64) {
	c := s.getCache()
	if c == nil || userID == "" || cacheKey == "" {
		return
	}
	if s.readSocialListCacheGen(ctx, userID) != genAtRead {
		return
	}
	exists, err := c.Exists(ctx, cacheKey)
	if err == nil && exists {
		return
	}
	if err := c.Set(ctx, cacheKey, value, ttl); err != nil {
		s.logger.Warn("failed to cache social list",
			zap.String("userID", userID),
			zap.String("cacheKey", cacheKey),
			zap.Error(err))
	}
}

func (s *Service) invalidateUserFollowListCaches(ctx context.Context, followerID, followeeID string) {
	c := s.getCache()
	if c == nil {
		return
	}
	if followerID != "" {
		s.bumpSocialListCacheGen(ctx, followerID)
	}
	if followeeID != "" && followeeID != followerID {
		s.bumpSocialListCacheGen(ctx, followeeID)
	}
	keys := make([]string, 0, 2)
	if followerID != "" {
		keys = append(keys, cache.UserKey(followerID))
	}
	if followeeID != "" {
		keys = append(keys, cache.UserKey(followeeID))
	}
	if len(keys) > 0 {
		_ = c.Delete(ctx, keys...)
	}
	for limit := 20; limit <= 100; limit += 20 {
		for offset := 0; offset < 200; offset += limit {
			if followerID != "" {
				_ = c.Delete(ctx, cache.UserFollowingKey(followerID)+fmt.Sprintf(":%d:%d", limit, offset))
			}
			if followeeID != "" {
				_ = c.Delete(ctx, cache.UserFollowersKey(followeeID)+fmt.Sprintf(":%d:%d", limit, offset))
			}
		}
	}
}
