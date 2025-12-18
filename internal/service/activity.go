package service

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// GetGroupActivitiesWithFilter 获取带过滤条件的群组活动
func (s *Service) GetGroupActivitiesWithFilter(ctx context.Context, req *domain.ActivityListRequest) ([]*domain.GroupActivity, int, error) {
	s.logger.Info("fetching group activities with filter",
		zap.String("groupID", req.GroupID),
		zap.String("timeRange", string(req.TimeRange)),
		zap.String("date", req.Date),
		zap.Int("limit", req.Limit))

	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	var activities []*domain.GroupActivity
	var err error

	// If specific date is provided, filter by that date
	if req.Date != "" {
		activities, err = s.repo.GroupActivitiesByDate(ctx, req.GroupID, req.Date, req.Limit, req.Offset)
		if err != nil {
			s.logger.Error("failed to fetch activities by date",
				zap.String("groupID", req.GroupID),
				zap.String("date", req.Date),
				zap.Error(err))
			return nil, 0, err
		}
	} else {
		// Filter by time range
		startTime, endTime := s.getTimeRangeBounds(req.TimeRange)
		activities, err = s.repo.GroupActivitiesByTimeRange(ctx, req.GroupID, startTime, endTime, req.Limit, req.Offset)
		if err != nil {
			s.logger.Error("failed to fetch activities by time range",
				zap.String("groupID", req.GroupID),
				zap.String("timeRange", string(req.TimeRange)),
				zap.Error(err))
			return nil, 0, err
		}
	}

	s.logger.Info("successfully fetched group activities with filter",
		zap.String("groupID", req.GroupID),
		zap.Int("count", len(activities)))

	return activities, len(activities), nil
}

// GetGroupActivityHeatmap 获取群组活动热力图数据
func (s *Service) GetGroupActivityHeatmap(ctx context.Context, groupID string, timeRange domain.ActivityTimeRange) (*domain.ActivityHeatmapResponse, error) {
	s.logger.Info("fetching group activity heatmap",
		zap.String("groupID", groupID),
		zap.String("timeRange", string(timeRange)))

	startTime, endTime := s.getTimeRangeBounds(timeRange)

	heatmapData, err := s.repo.GroupActivityHeatmap(ctx, groupID, startTime, endTime)
	if err != nil {
		s.logger.Error("failed to fetch activity heatmap",
			zap.String("groupID", groupID),
			zap.Error(err))
		return nil, err
	}

	// Calculate total count
	totalCount := 0
	for _, data := range heatmapData {
		totalCount += data.Count
	}

	// Fill in missing dates with zero counts
	heatmapData = s.fillMissingDates(heatmapData, startTime, endTime)

	loc := s.getChinaTimezone()
	response := &domain.ActivityHeatmapResponse{
		TimeRange:   timeRange,
		StartDate:   time.Unix(startTime, 0).In(loc).Format("2006-01-02"),
		EndDate:     time.Unix(endTime, 0).In(loc).Format("2006-01-02"),
		HeatmapData: make([]domain.ActivityHeatmapData, len(heatmapData)),
		TotalCount:  totalCount,
	}

	for i, data := range heatmapData {
		response.HeatmapData[i] = *data
	}

	s.logger.Info("successfully fetched activity heatmap",
		zap.String("groupID", groupID),
		zap.Int("dataPoints", len(heatmapData)),
		zap.Int("totalCount", totalCount))

	return response, nil
}

// getChinaTimezone returns the China timezone (Asia/Shanghai)
func (s *Service) getChinaTimezone() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// Fallback to UTC+8 if timezone loading fails
		loc = time.FixedZone("CST", 8*60*60)
	}
	return loc
}

// getTimeRangeBounds returns start and end timestamps for a given time range
// Uses China timezone (Asia/Shanghai) for consistent date calculations
func (s *Service) getTimeRangeBounds(timeRange domain.ActivityTimeRange) (int64, int64) {
	loc := s.getChinaTimezone()
	now := time.Now().In(loc)
	// Get start of today in China timezone
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)

	var startTime, endTime time.Time

	switch timeRange {
	case domain.TimeRangeToday:
		// Start of today to end of today
		startTime = today
		endTime = endOfDay
	case domain.TimeRangeWeek:
		// Last 7 days (including today) - use AddDate for correct month boundary handling
		startTime = today.AddDate(0, 0, -6)
		endTime = endOfDay
	case domain.TimeRangeMonth:
		// Last 30 days (including today)
		startTime = today.AddDate(0, 0, -29)
		endTime = endOfDay
	default:
		// Default to week
		startTime = today.AddDate(0, 0, -6)
		endTime = endOfDay
	}

	s.logger.Debug("time range bounds calculated",
		zap.String("timeRange", string(timeRange)),
		zap.Time("startTime", startTime),
		zap.Time("endTime", endTime))

	return startTime.Unix(), endTime.Unix()
}

// fillMissingDates fills in dates with zero counts for continuous heatmap display
// Uses China timezone for consistent date formatting
func (s *Service) fillMissingDates(data []*domain.ActivityHeatmapData, startTime, endTime int64) []*domain.ActivityHeatmapData {
	loc := s.getChinaTimezone()

	// Create a map of existing dates
	existingDates := make(map[string]int)
	for _, d := range data {
		existingDates[d.Date] = d.Count
	}

	// Generate all dates in range using China timezone
	var result []*domain.ActivityHeatmapData
	current := time.Unix(startTime, 0).In(loc)
	end := time.Unix(endTime, 0).In(loc)

	for !current.After(end) {
		dateStr := current.Format("2006-01-02")
		count := existingDates[dateStr] // Will be 0 if not exists
		result = append(result, &domain.ActivityHeatmapData{
			Date:  dateStr,
			Count: count,
		})
		current = current.AddDate(0, 0, 1)
	}

	return result
}

// GetGroupActivities 获取群组活动流（带缓存）
func (s *Service) GetGroupActivities(ctx context.Context, groupID string, limit int) ([]*domain.GroupActivity, error) {
	s.logger.Info("fetching group activities",
		zap.String("groupID", groupID),
		zap.Int("limit", limit))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.GroupActivitiesKey(groupID, limit)
		var cachedActivities []*domain.GroupActivity
		if err := c.Get(ctx, cacheKey, &cachedActivities); err == nil {
			s.logger.Debug("group activities cache hit",
				zap.String("groupID", groupID),
				zap.Int("count", len(cachedActivities)))
			return cachedActivities, nil
		} else {
			s.logger.Debug("group activities cache miss",
				zap.String("groupID", groupID),
				zap.Error(err))
		}
	}

	activities, err := s.repo.GroupActivities(ctx, groupID, limit)
	if err != nil {
		s.logger.Error("failed to fetch group activities",
			zap.String("groupID", groupID),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存
	if c != nil && len(activities) > 0 {
		cacheKey := cache.GroupActivitiesKey(groupID, limit)
		if err := c.Set(ctx, cacheKey, activities, activityCacheTTL); err != nil {
			s.logger.Warn("failed to cache group activities",
				zap.String("groupID", groupID),
				zap.Error(err))
		} else {
			s.logger.Debug("group activities cached",
				zap.String("groupID", groupID),
				zap.Int("count", len(activities)))
		}
	}

	s.logger.Info("successfully fetched group activities",
		zap.String("groupID", groupID),
		zap.Int("count", len(activities)))

	return activities, nil
}

// GetUserActivities 获取用户活动流（基于关注的用户和故事）
func (s *Service) GetUserActivities(ctx context.Context, userID string, limit int) ([]*domain.GroupActivity, error) {
	s.logger.Info("fetching user activities",
		zap.String("userID", userID),
		zap.Int("limit", limit))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 获取用户参与的群组
	groups, err := s.repo.GroupsByUser(ctx, userID)
	if err != nil {
		s.logger.Error("failed to fetch user groups",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	if len(groups) == 0 {
		s.logger.Info("user has no groups, returning empty activities",
			zap.String("userID", userID))
		return []*domain.GroupActivity{}, nil
	}

	// 并发获取所有群组的活动
	var (
		mu            sync.Mutex
		wg            sync.WaitGroup
		allActivities []*domain.GroupActivity
		perGroupLimit = limit / len(groups)
	)

	if perGroupLimit < 5 {
		perGroupLimit = 5 // 每个群组至少获取5条
	}

	s.logger.Debug("fetching activities from user groups",
		zap.String("userID", userID),
		zap.Int("groupCount", len(groups)),
		zap.Int("perGroupLimit", perGroupLimit))

	for _, group := range groups {
		wg.Add(1)
		go func(groupID string) {
			defer wg.Done()

			activities, err := s.repo.GroupActivities(ctx, groupID, perGroupLimit)
			if err != nil {
				s.logger.Warn("failed to fetch activities for group",
					zap.String("groupID", groupID),
					zap.Error(err))
				return
			}

			mu.Lock()
			allActivities = append(allActivities, activities...)
			mu.Unlock()
		}(group.ID)
	}

	wg.Wait()

	// 按时间戳降序排序
	sort.Slice(allActivities, func(i, j int) bool {
		return allActivities[i].Timestamp > allActivities[j].Timestamp
	})

	// 限制返回数量
	if len(allActivities) > limit {
		allActivities = allActivities[:limit]
	}

	s.logger.Info("successfully fetched user activities",
		zap.String("userID", userID),
		zap.Int("totalCount", len(allActivities)))

	return allActivities, nil
}

// GetGlobalActivities 获取全局活动流（推荐内容）
func (s *Service) GetGlobalActivities(ctx context.Context, limit int) ([]*domain.GroupActivity, error) {
	s.logger.Info("fetching global activities",
		zap.Int("limit", limit))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 获取所有公开群组
	groups, err := s.repo.ListGroups(ctx, 50, 0) // 获取前50个群组
	if err != nil {
		s.logger.Error("failed to fetch groups for global activities",
			zap.Error(err))
		return nil, err
	}

	if len(groups) == 0 {
		s.logger.Info("no groups available for global activities")
		return []*domain.GroupActivity{}, nil
	}

	// 并发获取各群组的活动
	var (
		mu            sync.Mutex
		wg            sync.WaitGroup
		allActivities []*domain.GroupActivity
		perGroupLimit = 5 // 每个群组获取5条活动
	)

	s.logger.Debug("fetching activities from public groups",
		zap.Int("groupCount", len(groups)),
		zap.Int("perGroupLimit", perGroupLimit))

	// 限制并发数，避免过多goroutine
	maxConcurrent := 10
	sem := make(chan struct{}, maxConcurrent)

	for _, group := range groups {
		// 只获取公开群组的活动
		if !group.Public {
			continue
		}

		wg.Add(1)
		go func(groupID string) {
			defer wg.Done()
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			activities, err := s.repo.GroupActivities(ctx, groupID, perGroupLimit)
			if err != nil {
				s.logger.Warn("failed to fetch activities for global feed",
					zap.String("groupID", groupID),
					zap.Error(err))
				return
			}

			mu.Lock()
			allActivities = append(allActivities, activities...)
			mu.Unlock()
		}(group.ID)
	}

	wg.Wait()

	// 按时间戳降序排序
	sort.Slice(allActivities, func(i, j int) bool {
		return allActivities[i].Timestamp > allActivities[j].Timestamp
	})

	// 限制返回数量
	if len(allActivities) > limit {
		allActivities = allActivities[:limit]
	}

	s.logger.Info("successfully fetched global activities",
		zap.Int("totalCount", len(allActivities)),
		zap.Int("groupsProcessed", len(groups)))

	return allActivities, nil
}

// ========== 群组活动记录方法 ==========

// RecordGroupStoryCreated 记录群组内创建新故事的活动
func (s *Service) RecordGroupStoryCreated(ctx context.Context, groupID, userID, storyID, storyTitle string) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get user for activity recording", zap.Error(err))
		return
	}

	activity := &domain.GroupActivity{
		ID:         uuid.New().String(),
		GroupID:    groupID,
		Type:       "story_created",
		UserID:     userID,
		UserName:   user.DisplayName,
		UserAvatar: user.Avatar,
		StoryID:    storyID,
		StoryTitle: storyTitle,
		Message:    "created a new story",
		Timestamp:  time.Now().Unix(),
	}

	if err := s.repo.CreateGroupActivity(ctx, activity); err != nil {
		s.logger.Warn("failed to record story created activity",
			zap.String("groupID", groupID),
			zap.String("storyID", storyID),
			zap.Error(err))
	} else {
		// 使活动流缓存失效
		c := s.getCache()
		if c != nil {
			for limit := 20; limit <= 100; limit += 20 {
				_ = c.Delete(ctx, cache.GroupActivitiesKey(groupID, limit))
			}
			s.logger.Debug("group activities cache invalidated",
				zap.String("groupID", groupID))
		}
	}
}

// RecordGroupStoryboardCreated 记录群组故事内创建新故事板的活动
func (s *Service) RecordGroupStoryboardCreated(ctx context.Context, groupID, userID, storyID, storyTitle string) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get user for activity recording", zap.Error(err))
		return
	}

	activity := &domain.GroupActivity{
		ID:         uuid.New().String(),
		GroupID:    groupID,
		Type:       "storyboard_created",
		UserID:     userID,
		UserName:   user.DisplayName,
		UserAvatar: user.Avatar,
		StoryID:    storyID,
		StoryTitle: storyTitle,
		Message:    "created a new storyboard",
		Timestamp:  time.Now().Unix(),
	}

	if err := s.repo.CreateGroupActivity(ctx, activity); err != nil {
		s.logger.Warn("failed to record storyboard created activity",
			zap.String("groupID", groupID),
			zap.String("storyID", storyID),
			zap.Error(err))
	} else {
		// 使活动流缓存失效
		c := s.getCache()
		if c != nil {
			for limit := 20; limit <= 100; limit += 20 {
				_ = c.Delete(ctx, cache.GroupActivitiesKey(groupID, limit))
			}
			s.logger.Debug("group activities cache invalidated",
				zap.String("groupID", groupID))
		}
	}
}

// RecordGroupMemberJoined 记录新成员加入群组的活动
func (s *Service) RecordGroupMemberJoined(ctx context.Context, groupID, userID string) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get user for activity recording", zap.Error(err))
		return
	}

	userName := user.DisplayName
	if userName == "" {
		userName = user.Username
	}

	activity := &domain.GroupActivity{
		ID:         uuid.New().String(),
		GroupID:    groupID,
		Type:       "member_joined",
		UserID:     userID,
		UserName:   userName,
		UserAvatar: user.Avatar,
		Message:    "joined the group",
		Timestamp:  time.Now().Unix(),
	}

	if err := s.repo.CreateGroupActivity(ctx, activity); err != nil {
		s.logger.Warn("failed to record member joined activity",
			zap.String("groupID", groupID),
			zap.String("userID", userID),
			zap.Error(err))
	} else {
		// 使活动流缓存失效
		c := s.getCache()
		if c != nil {
			for limit := 20; limit <= 100; limit += 20 {
				_ = c.Delete(ctx, cache.GroupActivitiesKey(groupID, limit))
			}
			s.logger.Debug("group activities cache invalidated",
				zap.String("groupID", groupID))
		}
	}
}
