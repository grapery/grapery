package mysql

import (
	"time"

	"gorm.io/gorm"
)

// UserStatistics 用户统计数据表
type UserStatistics struct {
	ID            uint           `gorm:"primaryKey;autoIncrement"`
	Date          time.Time      `gorm:"type:date;uniqueIndex;not null"` // 统计日期 (YYYY-MM-DD)
	DAU           int            `gorm:"default:0"`                      // Daily Active Users
	WAU           int            `gorm:"default:0"`                      // Weekly Active Users
	MAU           int            `gorm:"default:0"`                      // Monthly Active Users
	NewUsers      int            `gorm:"default:0"`                      // 当日新增用户数
	TotalUsers    int            `gorm:"default:0"`                      // 当日总用户数
	GrowthRateYoY float64        `gorm:"type:decimal(10,4);default:0"`   // 同比增长率 (Year-over-Year)
	GrowthRateMoM float64        `gorm:"type:decimal(10,4);default:0"`   // 环比增长率 (Month-over-Month)
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (UserStatistics) TableName() string {
	return "user_statistics"
}
