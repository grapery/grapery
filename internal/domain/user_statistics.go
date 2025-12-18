package domain

import "time"

// UserStatistics 用户统计数据
type UserStatistics struct {
	ID           uint      `json:"id"`
	Date         time.Time `json:"date"`          // 统计日期 (YYYY-MM-DD)
	DAU          int       `json:"dau"`           // Daily Active Users
	WAU          int       `json:"wau"`           // Weekly Active Users
	MAU          int       `json:"mau"`           // Monthly Active Users
	NewUsers     int       `json:"newUsers"`      // 当日新增用户数
	TotalUsers   int       `json:"totalUsers"`    // 当日总用户数
	GrowthRateYoY float64  `json:"growthRateYoY"` // 同比增长率 (Year-over-Year)
	GrowthRateMoM float64  `json:"growthRateMoM"` // 环比增长率 (Month-over-Month)
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

