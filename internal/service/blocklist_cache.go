package service

import (
	"context"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/recommendation"
	"go.uber.org/zap"
)

type blockedListCachePayload struct {
	Users []*domain.BlockedUser `json:"users"`
	Total int64                 `json:"total"`
}

const (
	blockedIDsCacheTTL    = 24 * time.Hour
	blockedIDsEmptyMember = "__blocked_empty__"
)

// invalidateBlockedUserCaches clears paginated block-list JSON caches and recommendation
// feeds. Blocked-ID membership is updated via writeThroughBlockedID and must not be deleted
// here — deleting the SET after write-through invites stale DB repopulation on repeat
// block/unblock.
func (s *Service) invalidateBlockedUserCaches(ctx context.Context, blockerID string) {
	if blockerID == "" {
		return
	}
	c := s.getCache()
	if c == nil {
		return
	}
	s.bumpSocialListCacheGen(ctx, blockerID)
	for limit := 20; limit <= 100; limit += 20 {
		for offset := 0; offset < 200; offset += limit {
			_ = c.Delete(ctx, cache.UserBlockedListKey(blockerID, limit, offset))
		}
	}
	recommendation.InvalidateAllForUser(ctx, c, blockerID)
}

func (s *Service) writeThroughBlockedID(ctx context.Context, blockerID, blockedID string, block bool) {
	c := s.getCache()
	if c == nil || blockerID == "" || blockedID == "" {
		return
	}
	key := cache.UserBlockedIDsKey(blockerID)
	if block {
		_ = c.SRem(ctx, key, blockedIDsEmptyMember)
		_ = c.SAdd(ctx, key, blockedID)
	} else {
		_ = c.SRem(ctx, key, blockedID)
		if members, err := c.SMembers(ctx, key); err == nil && len(members) == 0 {
			_ = c.SAdd(ctx, key, blockedIDsEmptyMember)
		}
	}
	_ = c.Expire(ctx, key, blockedIDsCacheTTL)
}

func (s *Service) populateBlockedIDsCache(ctx context.Context, blockerID string, ids []string) {
	c := s.getCache()
	if c == nil || blockerID == "" {
		return
	}
	key := cache.UserBlockedIDsKey(blockerID)
	exists, err := c.Exists(ctx, key)
	if err == nil && exists {
		// A concurrent block/unblock may have write-through updated the SET while we read DB.
		return
	}
	if len(ids) == 0 {
		_ = c.SAdd(ctx, key, blockedIDsEmptyMember)
	} else {
		members := make([]interface{}, len(ids))
		for i, id := range ids {
			members[i] = id
		}
		_ = c.SAdd(ctx, key, members...)
	}
	if err := c.Expire(ctx, key, blockedIDsCacheTTL); err != nil {
		s.logger.Warn("failed to set blocked ids cache ttl",
			zap.String("blockerID", blockerID),
			zap.Error(err))
	}
}

func (s *Service) readBlockedIDsFromCache(ctx context.Context, blockerID string) ([]string, bool) {
	c := s.getCache()
	if c == nil || blockerID == "" {
		return nil, false
	}
	key := cache.UserBlockedIDsKey(blockerID)
	exists, err := c.Exists(ctx, key)
	if err != nil || !exists {
		return nil, false
	}
	members, err := c.SMembers(ctx, key)
	if err != nil {
		return nil, false
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		if member == blockedIDsEmptyMember {
			continue
		}
		ids = append(ids, member)
	}
	return ids, true
}
