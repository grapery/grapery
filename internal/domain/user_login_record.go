package domain

import "time"

// UserLoginRecord 用户登录记录
type UserLoginRecord struct {
	ID        uint      `json:"id"`
	UserID    string    `json:"userId"`
	IPAddress string    `json:"ipAddress"` // IPv4 or IPv6 address
	Location  string    `json:"location"`  // 地理位置
	Device    string    `json:"device"`    // 设备类型
	OS        string    `json:"os"`        // 操作系统
	Browser   string    `json:"browser"`   // 浏览器
	UserAgent string    `json:"userAgent"` // 完整的 User-Agent
	LoginAt   time.Time `json:"loginAt"`   // 登录时间
	CreatedAt time.Time `json:"createdAt"`
}

