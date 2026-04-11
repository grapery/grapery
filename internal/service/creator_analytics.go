package service

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const creatorAnalyticsCacheTTL = 90 * time.Second

// CreatorAnalyticsDTO is the JSON payload for GET /api/v1/me/creator-analytics
type CreatorAnalyticsDTO struct {
	Range string `json:"range"`

	Summary CreatorAnalyticsSummaryDTO `json:"summary"`

	TrendLabels []string    `json:"trendLabels"`
	TrendSeries TrendSeriesDTO `json:"trendSeries"`

	FollowerCurve      []float64 `json:"followerCurve"`
	TotalFollowers     int       `json:"totalFollowers"`
	FollowerGrowthPct  *float64  `json:"followerGrowthPct,omitempty"`

	Interaction InteractionDTO `json:"interaction"`

	HotContent []HotContentDTO `json:"hotContent"`
}

type CreatorAnalyticsSummaryDTO struct {
	TotalViews        int64 `json:"totalViews"`
	TotalLikes        int64 `json:"totalLikes"`
	TotalBookmarks    int64 `json:"totalBookmarks"`
	NewFollowersHint  int64 `json:"newFollowersHint"`
	ViewsTrendPct     *int  `json:"viewsTrendPct,omitempty"`
	LikesTrendPct     *int  `json:"likesTrendPct,omitempty"`
	BookmarksTrendPct *int  `json:"bookmarksTrendPct,omitempty"`
	FollowersTrendPct *int  `json:"followersTrendPct,omitempty"`
}

type TrendSeriesDTO struct {
	Stories   []float64 `json:"stories"`
	Likes     []float64 `json:"likes"`
	Bookmarks []float64 `json:"bookmarks"`
}

type InteractionDTO struct {
	LikesPct    int `json:"likesPct"`
	CommentsPct int `json:"commentsPct"`
	BookmarksPct int `json:"bookmarksPct"`
	SharesPct   int `json:"sharesPct"`
	RepostsPct  int `json:"repostsPct"`
}

type HotContentDTO struct {
	Rank          int     `json:"rank"`
	StoryboardID  string  `json:"storyboardId"`
	StoryID       string  `json:"storyId"`
	Title         string  `json:"title"`
	Reads         int     `json:"reads"`
	Likes         int     `json:"likes"`
	GrowthPercent float64 `json:"growthPercent"`
	Up            bool    `json:"up"`
}

// GetMyCreatorAnalytics returns cached creator analytics for the authenticated user.
func (s *Service) GetMyCreatorAnalytics(ctx context.Context, userID string, rangeKey string) (*CreatorAnalyticsDTO, error) {
	if userID == "" {
		return nil, fmt.Errorf("unauthorized")
	}
	if rangeKey != "7d" && rangeKey != "30d" {
		rangeKey = "7d"
	}

	cacheKey := cache.CreatorAnalyticsKey(userID, rangeKey)
	if c, ok := s.cache.(cache.Cache); ok && c != nil {
		var dto CreatorAnalyticsDTO
		if err := c.Get(ctx, cacheKey, &dto); err == nil && dto.Range != "" {
			return &dto, nil
		}
	}

	dto, err := s.buildCreatorAnalyticsDTO(ctx, userID, rangeKey)
	if err != nil {
		return nil, err
	}

	if c, ok := s.cache.(cache.Cache); ok && c != nil {
		if err := c.Set(ctx, cacheKey, dto, creatorAnalyticsCacheTTL); err != nil {
			s.logger.Warn("creator analytics cache set failed", zap.Error(err))
		}
	}

	return dto, nil
}

// CreatorAnalyticsCacheInvalidate best-effort; call after major profile/content events if needed.
func (s *Service) CreatorAnalyticsCacheInvalidate(ctx context.Context, userID string) {
	if c, ok := s.cache.(cache.Cache); ok && c != nil {
		_ = c.Delete(ctx, cache.CreatorAnalyticsKey(userID, "7d"), cache.CreatorAnalyticsKey(userID, "30d"))
	}
}

func (s *Service) buildCreatorAnalyticsDTO(ctx context.Context, userID, rangeKey string) (*CreatorAnalyticsDTO, error) {
	agg, err := s.repo.CreatorAnalyticsAggregate(ctx, userID)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.TopCreatorStoryboards(ctx, userID, 5)
	if err != nil {
		top = nil
	}

	totalLikes := agg.TotalStoryLikes + agg.TotalStoryboardLikes
	seed := hashUser(userID)
	if rangeKey == "30d" {
		seed ^= 0xBADC0FFE
	}

	dto := &CreatorAnalyticsDTO{
		Range:          rangeKey,
		TrendLabels:    lastNDayLabels(7),
		TotalFollowers: user.Followers,
		Summary: CreatorAnalyticsSummaryDTO{
			TotalViews:       agg.TotalViews,
			TotalLikes:       totalLikes,
			TotalBookmarks:   agg.TotalSaves,
			NewFollowersHint: syntheticNewFollowers(user.Followers, seed),
		},
		TrendSeries: TrendSeriesDTO{
			Stories:   syntheticMetricSeries(7, float64(agg.TotalViews), seed, 1),
			Likes:     syntheticMetricSeries(7, float64(totalLikes), seed, 2),
			Bookmarks: syntheticMetricSeries(7, float64(agg.TotalSaves), seed, 3),
		},
		FollowerCurve: syntheticFollowerCurve(6, user.Followers, seed),
		Interaction:   buildInteractionDTO(agg),
		HotContent:    buildHotContent(top),
	}

	// Optional trend chips: lightweight pseudo deltas (stable per user) until time-series exists.
	dto.Summary.ViewsTrendPct = ptrInt(syntheticTrendPct(seed, 11, 40))
	dto.Summary.LikesTrendPct = ptrInt(syntheticTrendPct(seed, 13, 35))
	dto.Summary.BookmarksTrendPct = ptrInt(syntheticTrendPct(seed, 17, 25))
	dto.Summary.FollowersTrendPct = ptrInt(syntheticTrendPct(seed, 19, 45))
	g := float64(syntheticTrendPct(seed, 23, 50))
	dto.FollowerGrowthPct = &g

	return dto, nil
}

func ptrInt(v int) *int { return &v }

func hashUser(userID string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	return h.Sum32()
}

func syntheticTrendPct(seed uint32, salt int, max int) int {
	x := int((seed + uint32(salt)*131) % uint32(max+1))
	if x < 3 {
		x = 3
	}
	return x
}

func syntheticNewFollowers(followers int, seed uint32) int64 {
	if followers <= 0 {
		return 0
	}
	// Placeholder until follower events exist: small fraction of total, stable per user.
	p := float64((seed%40)+10) / 100.0
	v := int64(math.Round(float64(followers) * p))
	if v < 1 && followers > 0 {
		return 1
	}
	if v > int64(followers) {
		return int64(followers)
	}
	return v
}

func syntheticMetricSeries(n int, base float64, seed uint32, lane int) []float64 {
	if n < 2 {
		n = 2
	}
	norm := base
	if norm < 1 {
		norm = 1
	}
	cap := 1.0
	if norm > 0 {
		cap = math.Min(1.0, math.Log10(norm+10)/5.0)
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		j := float64(i) / float64(n-1)
		jit := float64((seed+uint32(i*7)+uint32(lane*97))%50) / 500.0
		v := cap * (0.55 + 0.45*j + jit)
		if v > 1 {
			v = 1
		}
		out[i] = v
	}
	return out
}

func syntheticFollowerCurve(n int, followers int, seed uint32) []float64 {
	if n < 2 {
		n = 2
	}
	base := float64(followers)
	if base < 1 {
		base = 1
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		j := float64(i) / float64(n-1)
		jit := float64((seed+uint32(i*11))%30) / 200.0
		// Slight dip then rise toward current follower level (normalized 0..1)
		v := 0.72 - 0.22*math.Sin(j*math.Pi) + 0.35*j + jit
		if v > 1 {
			v = 1
		}
		if v < 0.08 {
			v = 0.08
		}
		out[i] = v
	}
	return out
}

func buildInteractionDTO(agg *domain.CreatorAnalyticsAggregate) InteractionDTO {
	likes := agg.TotalStoryLikes + agg.TotalStoryboardLikes
	comments := agg.TotalComments
	bookmarks := agg.TotalSaves
	shares := agg.TotalShares
	reposts := agg.TotalForks
	total := likes + comments + bookmarks + shares + reposts
	if total <= 0 {
		return InteractionDTO{LikesPct: 20, CommentsPct: 20, BookmarksPct: 20, SharesPct: 20, RepostsPct: 20}
	}
	p := func(x int64) int {
		return int(math.Round(100.0 * float64(x) / float64(total)))
	}
	d := InteractionDTO{
		LikesPct:     p(likes),
		CommentsPct:  p(comments),
		BookmarksPct: p(bookmarks),
		SharesPct:    p(shares),
		RepostsPct:   p(reposts),
	}
	// Fix rounding drift
	sum := d.LikesPct + d.CommentsPct + d.BookmarksPct + d.SharesPct + d.RepostsPct
	if sum != 100 && sum > 0 {
		d.LikesPct += 100 - sum
	}
	return d
}

func buildHotContent(rows []*domain.CreatorAnalyticsStoryboardRow) []HotContentDTO {
	out := make([]HotContentDTO, 0, len(rows))
	for i, r := range rows {
		if r == nil {
			continue
		}
		g := float64(syntheticTrendPct(hashUser(r.ID), 31+i, 25))
		out = append(out, HotContentDTO{
			Rank:          i + 1,
			StoryboardID:  r.ID,
			StoryID:       r.StoryID,
			Title:         r.Title,
			Reads:         r.Views,
			Likes:         r.Likes,
			GrowthPercent: g,
			Up:            true,
		})
	}
	return out
}

func lastNDayLabels(n int) []string {
	if n <= 0 {
		n = 7
	}
	now := time.Now().UTC()
	labels := make([]string, n)
	for i := n - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		labels[n-1-i] = d.Format("1/2")
	}
	return labels
}
