package recommendation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/config"
)

const (
	seenStoryboardsKeyPrefix = "reco:for_you_seen:storyboards:"
	seenFragmentsKeyPrefix   = "reco:for_you_seen:fragments:"
)

func seenStoryboardsKey(userID string) string {
	return seenStoryboardsKeyPrefix + userID
}

func seenFragmentsKey(userID string) string {
	return seenFragmentsKeyPrefix + userID
}

func seenMaxEntries(cfg config.RecommendationConfig) int {
	if cfg.SeenMaxEntries > 0 {
		return cfg.SeenMaxEntries
	}
	return 5000
}

func seenKeyTTL(cfg config.RecommendationConfig) time.Duration {
	if cfg.SeenTTLDays <= 0 {
		return 0
	}
	return time.Duration(cfg.SeenTTLDays) * 24 * time.Hour
}

// RecordStoryboardSeen records that the user opened a storyboard detail (for for_you exclusion).
func RecordStoryboardSeen(ctx context.Context, c cache.Cache, cfg config.RecommendationConfig, userID, storyboardID string) {
	if c == nil || userID == "" || strings.TrimSpace(storyboardID) == "" {
		return
	}
	recordSeen(ctx, c, cfg, seenStoryboardsKey(userID), storyboardID, FeedTypeStoryboards, userID)
}

// RecordFragmentSeen records that the user opened a fragment detail (for for_you exclusion).
func RecordFragmentSeen(ctx context.Context, c cache.Cache, cfg config.RecommendationConfig, userID, fragmentID string) {
	if c == nil || userID == "" || strings.TrimSpace(fragmentID) == "" {
		return
	}
	recordSeen(ctx, c, cfg, seenFragmentsKey(userID), fragmentID, FeedTypeFragments, userID)
}

func recordSeen(ctx context.Context, c cache.Cache, cfg config.RecommendationConfig, key, entityID, feedType, userID string) {
	score := float64(time.Now().UnixMilli())
	if err := c.ZAdd(ctx, key, &redis.Z{Score: score, Member: entityID}); err != nil {
		return
	}
	max := seenMaxEntries(cfg)
	if max > 0 {
		n, err := c.ZCard(ctx, key)
		if err == nil && n > int64(max) {
			remove := n - int64(max)
			if remove > 0 {
				_ = c.ZRemRangeByRank(ctx, key, 0, remove-1)
			}
		}
	}
	if ttl := seenKeyTTL(cfg); ttl > 0 {
		_ = c.Expire(ctx, key, ttl)
	}
	InvalidateFeedCache(ctx, c, feedType, userID)
}

// ListSeenStoryboardIDs returns storyboard IDs the user has recently opened (newest first).
func ListSeenStoryboardIDs(ctx context.Context, c cache.Cache, cfg config.RecommendationConfig, userID string) ([]string, error) {
	if c == nil || userID == "" {
		return nil, nil
	}
	key := seenStoryboardsKey(userID)
	max := seenMaxEntries(cfg)
	if max <= 0 {
		max = 5000
	}
	// ZSET is capped at max; fetch all members (newest first by score).
	ids, err := c.ZRevRange(ctx, key, 0, int64(max-1))
	if err != nil {
		return nil, fmt.Errorf("list seen storyboards: %w", err)
	}
	return ids, nil
}

// ListSeenFragmentIDs returns fragment IDs the user has recently opened (newest first).
func ListSeenFragmentIDs(ctx context.Context, c cache.Cache, cfg config.RecommendationConfig, userID string) ([]string, error) {
	if c == nil || userID == "" {
		return nil, nil
	}
	key := seenFragmentsKey(userID)
	max := seenMaxEntries(cfg)
	if max <= 0 {
		max = 5000
	}
	ids, err := c.ZRevRange(ctx, key, 0, int64(max-1))
	if err != nil {
		return nil, fmt.Errorf("list seen fragments: %w", err)
	}
	return ids, nil
}

// FilterExcludedOrderedIDs drops IDs present in exclude, preserving order.
func FilterExcludedOrderedIDs(ids []string, exclude map[string]struct{}) []string {
	if len(exclude) == 0 {
		return ids
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, skip := exclude[id]; skip {
			continue
		}
		out = append(out, id)
	}
	return out
}

// SeenIDsToSet builds a lookup set for exclusion (nil if ids empty).
func SeenIDsToSet(ids []string) map[string]struct{} {
	if len(ids) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out[id] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
