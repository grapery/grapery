package mysql

import (
	"context"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// CountAllUsers 统计所有用户总数
func (r *Repository) CountAllUsers(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("status = ?", string(common.StatusActive)).Count(&count).Error
	return int(count), err
}

// CountNewUsersByDate 统计指定日期新增的用户数
func (r *Repository) CountNewUsersByDate(ctx context.Context, date time.Time) (int, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).
		Where("created_at >= ? AND created_at < ?", startOfDay.Unix(), endOfDay.Unix()).
		Count(&count).Error
	return int(count), err
}

// GetUserStatisticsByDate 根据日期获取用户统计数据
func (r *Repository) GetUserStatisticsByDate(ctx context.Context, date time.Time) (*domain.UserStatistics, error) {
	// 将日期转换为当天的开始时间（用于查询）
	dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	var stats UserStatistics
	err := r.db.WithContext(ctx).
		Where("DATE(date) = ?", dateStart.Format("2006-01-02")).
		First(&stats).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.userStatisticsToDomain(&stats), nil
}

// SaveUserStatistics 保存或更新用户统计数据
func (r *Repository) SaveUserStatistics(ctx context.Context, stats *domain.UserStatistics) error {
	dbStats := r.userStatisticsFromDomain(stats)
	// 确保日期是当天的开始时间
	dbStats.Date = time.Date(stats.Date.Year(), stats.Date.Month(), stats.Date.Day(), 0, 0, 0, 0, stats.Date.Location())

	// 使用 ON DUPLICATE KEY UPDATE 或先查询后更新
	var existing UserStatistics
	err := r.db.WithContext(ctx).
		Where("DATE(date) = ?", dbStats.Date.Format("2006-01-02")).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		return r.db.WithContext(ctx).Create(dbStats).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录（保留 created_at，避免零值写入 MySQL）
	dbStats.ID = existing.ID
	return r.db.WithContext(ctx).Omit("CreatedAt").Save(dbStats).Error
}

// userStatisticsToDomain 转换数据库模型到领域模型
func (r *Repository) userStatisticsToDomain(m *UserStatistics) *domain.UserStatistics {
	return &domain.UserStatistics{
		ID:            m.ID,
		Date:          m.Date,
		DAU:           m.DAU,
		WAU:           m.WAU,
		MAU:           m.MAU,
		NewUsers:      m.NewUsers,
		TotalUsers:    m.TotalUsers,
		GrowthRateYoY: m.GrowthRateYoY,
		GrowthRateMoM: m.GrowthRateMoM,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// userStatisticsFromDomain 转换领域模型到数据库模型
func (r *Repository) userStatisticsFromDomain(d *domain.UserStatistics) *UserStatistics {
	return &UserStatistics{
		ID:            d.ID,
		Date:          d.Date,
		DAU:           d.DAU,
		WAU:           d.WAU,
		MAU:           d.MAU,
		NewUsers:      d.NewUsers,
		TotalUsers:    d.TotalUsers,
		GrowthRateYoY: d.GrowthRateYoY,
		GrowthRateMoM: d.GrowthRateMoM,
	}
}
