package active

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/cache"
	"github.com/grapery/grapery/utils/log"
	redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	ErrStartAfterEnd   = errors.New("start time after end time")
	ErrUnsupportedSpan = errors.New("unsupported time range")
)

const (
	userHeatmapCachePrefix  = "heatmap:user"
	groupHeatmapCachePrefix = "heatmap:group"
	heatmapCacheTTLSeconds  = int64(300)
)

type HeatmapResult struct {
	Items      []*api.HeatmapDataItem
	TotalCount int64
	Days       int
}

type GroupHeatmapResult struct {
	Items       []*api.HeatmapDataItem
	TotalCount  int64
	Days        int
	MemberCount int64
}

func BuildUserHeatmap(ctx context.Context, userID int64, startTS, endTS int64) (*HeatmapResult, error) {
	startDay, endExclusive, days, err := resolveHeatmapRange(startTS, endTS)
	if err != nil {
		return nil, err
	}
	cacheKey := heatmapCacheKey(userHeatmapCachePrefix, userID, startDay, endExclusive, days)
	if payload, ok, err := getHeatmapFromCache(ctx, cacheKey); err != nil {
		log.Log().Warn("get user heatmap cache failed",
			zap.Int64("user_id", userID),
			zap.Int64("start_ts", startTS),
			zap.Int64("end_ts", endTS),
			zap.Error(err))
	} else if ok {
		return &HeatmapResult{
			Items:      payload.Items,
			TotalCount: payload.TotalCount,
			Days:       payload.Days,
		}, nil
	}
	entries, err := models.GetUserActiveHeatmapEntries(ctx, userID, startDay, endExclusive)
	if err != nil {
		return nil, err
	}
	items, total := buildHeatmapItems(entries, startDay, days)
	result := &HeatmapResult{
		Items:      items,
		TotalCount: total,
		Days:       days,
	}
	if err := setHeatmapCache(ctx, cacheKey, cachedHeatmapPayload{
		Items:      items,
		TotalCount: total,
		Days:       days,
	}); err != nil {
		log.Log().Warn("set user heatmap cache failed",
			zap.Int64("user_id", userID),
			zap.String("cache_key", cacheKey),
			zap.Error(err))
	}
	return result, nil
}

func BuildGroupHeatmap(ctx context.Context, groupID int64, startTS, endTS int64) (*GroupHeatmapResult, error) {
	startDay, endExclusive, days, err := resolveHeatmapRange(startTS, endTS)
	if err != nil {
		return nil, err
	}
	cacheKey := heatmapCacheKey(groupHeatmapCachePrefix, groupID, startDay, endExclusive, days)
	if payload, ok, err := getHeatmapFromCache(ctx, cacheKey); err != nil {
		log.Log().Warn("get group heatmap cache failed",
			zap.Int64("group_id", groupID),
			zap.Int64("start_ts", startTS),
			zap.Int64("end_ts", endTS),
			zap.Error(err))
	} else if ok {
		return &GroupHeatmapResult{
			Items:       payload.Items,
			TotalCount:  payload.TotalCount,
			Days:        payload.Days,
			MemberCount: payload.MemberCount,
		}, nil
	}
	entries, err := models.GetGroupActiveHeatmapEntries(ctx, groupID, startDay, endExclusive)
	if err != nil {
		return nil, err
	}
	memberCount, err := models.CountGroupActiveMembers(ctx, groupID, startDay, endExclusive)
	if err != nil {
		return nil, err
	}
	items, total := buildHeatmapItems(entries, startDay, days)
	result := &GroupHeatmapResult{
		Items:       items,
		TotalCount:  total,
		Days:        days,
		MemberCount: memberCount,
	}
	if err := setHeatmapCache(ctx, cacheKey, cachedHeatmapPayload{
		Items:       items,
		TotalCount:  total,
		Days:        days,
		MemberCount: memberCount,
	}); err != nil {
		log.Log().Warn("set group heatmap cache failed",
			zap.Int64("group_id", groupID),
			zap.String("cache_key", cacheKey),
			zap.Error(err))
	}
	return result, nil
}

type cachedHeatmapPayload struct {
	Items       []*api.HeatmapDataItem `json:"items"`
	TotalCount  int64                  `json:"total_count"`
	Days        int                    `json:"days"`
	MemberCount int64                  `json:"member_count,omitempty"`
}

func resolveHeatmapRange(startTS, endTS int64) (time.Time, time.Time, int, error) {
	loc := time.Local

	var endTime time.Time
	if endTS > 0 {
		endTime = time.Unix(endTS, 0).In(loc)
	} else {
		endTime = time.Now().In(loc)
	}
	endDay := truncateToDay(endTime)

	days := 7
	if startTS > 0 {
		startTime := time.Unix(startTS, 0).In(loc)
		startDay := truncateToDay(startTime)
		if startDay.After(endDay) {
			return time.Time{}, time.Time{}, 0, ErrStartAfterEnd
		}
		span := int(endDay.Sub(startDay).Hours()/24) + 1
		normalized := normalizeHeatmapDays(span)
		if normalized == 0 {
			return time.Time{}, time.Time{}, 0, ErrUnsupportedSpan
		}
		days = normalized
	}

	startDay := endDay.AddDate(0, 0, -(days - 1))
	endExclusive := endDay.AddDate(0, 0, 1)
	return startDay, endExclusive, days, nil
}

func buildHeatmapItems(entries []models.HeatmapEntry, start time.Time, days int) ([]*api.HeatmapDataItem, int64) {
	counts := make(map[string]int64, len(entries))
	var total int64
	var maxCount int64
	for _, entry := range entries {
		counts[entry.Date] = entry.Count
		total += entry.Count
		if entry.Count > maxCount {
			maxCount = entry.Count
		}
	}

	items := make([]*api.HeatmapDataItem, 0, days)
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		dateStr := day.Format("2006-01-02")
		count := counts[dateStr]
		items = append(items, &api.HeatmapDataItem{
			Date:  dateStr,
			Count: count,
			Level: calculateHeatmapLevel(count, maxCount),
		})
	}
	return items, total
}

func getHeatmapFromCache(ctx context.Context, key string) (*cachedHeatmapPayload, bool, error) {
	if cache.GetCacheClient() == nil {
		return nil, false, nil
	}
	data, err := cache.GetBytes(ctx, key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var payload cachedHeatmapPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false, err
	}
	return &payload, true, nil
}

func setHeatmapCache(ctx context.Context, key string, payload cachedHeatmapPayload) error {
	if cache.GetCacheClient() == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return cache.SetBytes(ctx, key, data, heatmapCacheTTLSeconds)
}

func heatmapCacheKey(prefix string, id int64, startDay, endExclusive time.Time, days int) string {
	endDay := endExclusive.AddDate(0, 0, -1)
	return fmt.Sprintf("%s:%d:%s:%s:%d", prefix, id, startDay.Format("20060102"), endDay.Format("20060102"), days)
}

func normalizeHeatmapDays(span int) int {
	switch {
	case span <= 0:
		return 0
	case span >= 7 && span <= 8:
		return 7
	case span >= 29 && span <= 32:
		return 30
	case span >= 178 && span <= 186:
		return 180
	default:
		return 0
	}
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func calculateHeatmapLevel(count, max int64) int64 {
	if count <= 0 || max <= 0 {
		return 0
	}

	ratio := float64(count) / float64(max)

	switch {
	case ratio >= 0.8:
		return 4
	case ratio >= 0.6:
		return 3
	case ratio >= 0.4:
		return 2
	case ratio > 0:
		return 1
	default:
		return 0
	}
}
