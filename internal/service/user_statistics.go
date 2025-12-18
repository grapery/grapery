package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"go.uber.org/zap"
)

// UserStatisticsService 用户统计服务
type UserStatisticsService struct {
	repo   domain.Repository
	cache  cache.Cache
	logger *zap.Logger
	metrics *telemetry.Metrics
}

// NewUserStatisticsService 创建用户统计服务
func NewUserStatisticsService(repo domain.Repository, cache cache.Cache, logger *zap.Logger, metrics *telemetry.Metrics) *UserStatisticsService {
	return &UserStatisticsService{
		repo:    repo,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// RecordActiveUser 记录活跃用户（登录时调用）
func (s *UserStatisticsService) RecordActiveUser(ctx context.Context, userID string) error {
	if s.cache == nil {
		return nil
	}

	now := time.Now()
	
	// Redis key 格式：
	// - DAU: "dau:YYYY-MM-DD"
	// - WAU: "wau:YYYY-WW" (周数)
	// - MAU: "mau:YYYY-MM"
	
	dateKey := now.Format("2006-01-02")
	weekKey := fmt.Sprintf("%d-W%02d", now.Year(), getWeekOfYear(now))
	monthKey := now.Format("2006-01")
	
	// 使用 Redis Set 存储活跃用户ID（自动去重）
	dauKey := fmt.Sprintf("dau:%s", dateKey)
	wauKey := fmt.Sprintf("wau:%s", weekKey)
	mauKey := fmt.Sprintf("mau:%s", monthKey)
	
	// 添加到对应的集合中
	if err := s.cache.SAdd(ctx, dauKey, userID); err != nil {
		s.logger.Warn("failed to record DAU", zap.String("userID", userID), zap.Error(err))
	}
	if err := s.cache.SAdd(ctx, wauKey, userID); err != nil {
		s.logger.Warn("failed to record WAU", zap.String("userID", userID), zap.Error(err))
	}
	if err := s.cache.SAdd(ctx, mauKey, userID); err != nil {
		s.logger.Warn("failed to record MAU", zap.String("userID", userID), zap.Error(err))
	}
	
	// 设置过期时间（DAU: 2天, WAU: 8天, MAU: 32天）
	s.cache.Expire(ctx, dauKey, 48*time.Hour)
	s.cache.Expire(ctx, wauKey, 8*24*time.Hour)
	s.cache.Expire(ctx, mauKey, 32*24*time.Hour)
	
	return nil
}

// GetActiveUserCounts 获取活跃用户数量
func (s *UserStatisticsService) GetActiveUserCounts(ctx context.Context, date time.Time) (dau, wau, mau int, err error) {
	if s.cache == nil {
		return 0, 0, 0, fmt.Errorf("cache not available")
	}
	
	dateKey := date.Format("2006-01-02")
	weekKey := fmt.Sprintf("%d-W%02d", date.Year(), getWeekOfYear(date))
	monthKey := date.Format("2006-01")
	
	dauKey := fmt.Sprintf("dau:%s", dateKey)
	wauKey := fmt.Sprintf("wau:%s", weekKey)
	mauKey := fmt.Sprintf("mau:%s", monthKey)
	
	// 获取集合大小
	dauMembers, err := s.cache.SMembers(ctx, dauKey)
	if err == nil {
		dau = len(dauMembers)
	}
	
	wauMembers, err := s.cache.SMembers(ctx, wauKey)
	if err == nil {
		wau = len(wauMembers)
	}
	
	mauMembers, err := s.cache.SMembers(ctx, mauKey)
	if err == nil {
		mau = len(mauMembers)
	}
	
	return dau, wau, mau, nil
}

// CalculateGrowthRate 计算同比增长率和环比增长率
func (s *UserStatisticsService) CalculateGrowthRate(ctx context.Context, date time.Time) (yoyRate, momRate float64, err error) {
	// 获取当日总用户数
	todayTotal, err := s.repo.CountAllUsers(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get today total users: %w", err)
	}
	
	// 计算同比（去年同期）
	lastYearDate := date.AddDate(-1, 0, 0)
	lastYearStats, err := s.repo.GetUserStatisticsByDate(ctx, lastYearDate)
	if err != nil {
		s.logger.Warn("failed to get last year statistics", zap.Error(err))
		yoyRate = 0
	} else if lastYearStats != nil && lastYearStats.TotalUsers > 0 {
		yoyRate = float64(todayTotal-lastYearStats.TotalUsers) / float64(lastYearStats.TotalUsers) * 100
	}
	
	// 计算环比（上个月同期）
	lastMonthDate := date.AddDate(0, -1, 0)
	lastMonthStats, err := s.repo.GetUserStatisticsByDate(ctx, lastMonthDate)
	if err != nil {
		s.logger.Warn("failed to get last month statistics", zap.Error(err))
		momRate = 0
	} else if lastMonthStats != nil && lastMonthStats.TotalUsers > 0 {
		momRate = float64(todayTotal-lastMonthStats.TotalUsers) / float64(lastMonthStats.TotalUsers) * 100
	}
	
	return yoyRate, momRate, nil
}

// PersistStatistics 将统计数据持久化到数据库
func (s *UserStatisticsService) PersistStatistics(ctx context.Context, date time.Time) error {
	// 获取活跃用户数
	dau, wau, mau, err := s.GetActiveUserCounts(ctx, date)
	if err != nil {
		s.logger.Warn("failed to get active user counts", zap.Error(err))
		dau, wau, mau = 0, 0, 0
	}
	
	// 获取新增用户数（当日注册的用户）
	newUsers, err := s.repo.CountNewUsersByDate(ctx, date)
	if err != nil {
		s.logger.Warn("failed to get new users count", zap.Error(err))
		newUsers = 0
	}
	
	// 获取总用户数
	totalUsers, err := s.repo.CountAllUsers(ctx)
	if err != nil {
		s.logger.Warn("failed to get total users count", zap.Error(err))
		totalUsers = 0
	}
	
	// 计算增长率
	yoyRate, momRate, err := s.CalculateGrowthRate(ctx, date)
	if err != nil {
		s.logger.Warn("failed to calculate growth rate", zap.Error(err))
	}
	
	// 保存或更新统计数据
	stats := &domain.UserStatistics{
		Date:         date,
		DAU:          dau,
		WAU:          wau,
		MAU:          mau,
		NewUsers:     newUsers,
		TotalUsers:   totalUsers,
		GrowthRateYoY: yoyRate,
		GrowthRateMoM: momRate,
	}
	
	if err := s.repo.SaveUserStatistics(ctx, stats); err != nil {
		return fmt.Errorf("failed to save user statistics: %w", err)
	}
	
	// 更新 Prometheus metrics
	if s.metrics != nil {
		s.metrics.RecordUserCount(float64(totalUsers))
		s.metrics.RecordDailyActiveUsers(float64(dau))
		s.metrics.RecordWeeklyActiveUsers(float64(wau))
		s.metrics.RecordMonthlyActiveUsers(float64(mau))
		s.metrics.RecordUserGrowthRate("yoy", yoyRate)
		s.metrics.RecordUserGrowthRate("mom", momRate)
	}
	
	s.logger.Info("user statistics persisted",
		zap.String("date", date.Format("2006-01-02")),
		zap.Int("dau", dau),
		zap.Int("wau", wau),
		zap.Int("mau", mau),
		zap.Int("newUsers", newUsers),
		zap.Int("totalUsers", totalUsers),
		zap.Float64("yoyRate", yoyRate),
		zap.Float64("momRate", momRate))
	
	return nil
}

// getWeekOfYear 获取一年中的第几周
func getWeekOfYear(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

