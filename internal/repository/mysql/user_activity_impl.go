package mysql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// CreateUserActivity 创建用户活动记录
func (r *Repository) CreateUserActivity(ctx context.Context, activity *domain.UserActivity) error {
	dbActivity := &UserActivity{
		ID:          uuid.New().String(),
		UserID:      activity.UserID,
		Type:        activity.Type,
		TargetID:    activity.TargetID,
		TargetType:  activity.TargetType,
		TargetTitle: activity.TargetTitle,
		Message:     activity.Message,
		CreatedAt:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(dbActivity).Error; err != nil {
		return err
	}

	activity.ID = dbActivity.ID
	activity.CreatedAt = dbActivity.CreatedAt.Unix()
	return nil
}

// UserActivitiesByUserID 获取用户活动列表
func (r *Repository) UserActivitiesByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.UserActivity, error) {
	var activities []UserActivity
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&activities).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.UserActivity, len(activities))
	for i, a := range activities {
		result[i] = r.userActivityToDomain(&a)
	}
	return result, nil
}

// DeleteUserActivity 删除用户活动记录
func (r *Repository) DeleteUserActivity(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&UserActivity{}, "id = ?", id).Error
}

// UserActivitiesByTimeRange 按时间范围获取用户活动
func (r *Repository) UserActivitiesByTimeRange(ctx context.Context, userID string, startTime, endTime int64, limit, offset int) ([]*domain.UserActivity, error) {
	var activities []UserActivity

	startT := time.Unix(startTime, 0)
	endT := time.Unix(endTime, 0)

	query := r.db.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", userID).
		Where("created_at >= ?", startT).
		Where("created_at <= ?", endT).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	} else {
		query = query.Limit(50)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.UserActivity, len(activities))
	for i, a := range activities {
		result[i] = r.userActivityToDomain(&a)
	}
	return result, nil
}

// UserActivitiesByDate 按日期获取用户活动
func (r *Repository) UserActivitiesByDate(ctx context.Context, userID string, date string, limit, offset int) ([]*domain.UserActivity, error) {
	var activities []UserActivity

	// Parse date string to get start and end of day
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}

	startOfDay := parsedDate
	endOfDay := parsedDate.Add(24*time.Hour - time.Second)

	query := r.db.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", userID).
		Where("created_at >= ?", startOfDay).
		Where("created_at <= ?", endOfDay).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	} else {
		query = query.Limit(50)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.UserActivity, len(activities))
	for i, a := range activities {
		result[i] = r.userActivityToDomain(&a)
	}
	return result, nil
}

// UserActivityHeatmap 获取用户活动热力图数据
// Uses China timezone (UTC+8) for date grouping
func (r *Repository) UserActivityHeatmap(ctx context.Context, userID string, startTime, endTime int64) ([]*domain.ActivityHeatmapData, error) {
	type DateCount struct {
		Date  string `gorm:"column:date"`
		Count int    `gorm:"column:count"`
	}

	var dateCounts []DateCount

	startT := time.Unix(startTime, 0)
	endT := time.Unix(endTime, 0)

	// Group activities by date and count using DATE_FORMAT with China timezone conversion
	// CONVERT_TZ converts from UTC ('+00:00') to China Standard Time ('+08:00')
	err := r.db.WithContext(ctx).
		Model(&UserActivity{}).
		Select("DATE_FORMAT(CONVERT_TZ(created_at, '+00:00', '+08:00'), '%Y-%m-%d') as date, COUNT(*) as count").
		Where("user_id = ?", userID).
		Where("created_at >= ?", startT).
		Where("created_at <= ?", endT).
		Where("deleted_at IS NULL").
		Group("DATE_FORMAT(CONVERT_TZ(created_at, '+00:00', '+08:00'), '%Y-%m-%d')").
		Order("date ASC").
		Scan(&dateCounts).Error

	if err != nil {
		return nil, err
	}

	result := make([]*domain.ActivityHeatmapData, len(dateCounts))
	for i, dc := range dateCounts {
		result[i] = &domain.ActivityHeatmapData{
			Date:  dc.Date,
			Count: dc.Count,
		}
	}
	return result, nil
}

// userActivityToDomain 转换用户活动到 domain
func (r *Repository) userActivityToDomain(activity *UserActivity) *domain.UserActivity {
	result := &domain.UserActivity{
		BaseModel: common.BaseModel{
			ID:        activity.ID,
			CreatedAt: activity.CreatedAt.Unix(),
		},
		UserID:      activity.UserID,
		Type:        activity.Type,
		TargetID:    activity.TargetID,
		TargetType:  activity.TargetType,
		TargetTitle: activity.TargetTitle,
		Message:     activity.Message,
	}

	if activity.User.ID != "" {
		result.User = r.userToDomainPtr(&activity.User)
	}

	return result
}
