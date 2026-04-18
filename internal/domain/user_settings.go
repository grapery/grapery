package domain

import "time"

// DeviceInfo 设备信息
type DeviceInfo struct {
	ID           string `json:"id"`
	DeviceID     string `json:"deviceId"`
	DeviceType   string `json:"deviceType"` // ios, android, web
	DeviceName   string `json:"deviceName"`
	OS           string `json:"os,omitempty"`
	Browser      string `json:"browser,omitempty"`
	IPAddress    string `json:"ipAddress"`
	Location     string `json:"location,omitempty"`
	IsCurrent    bool   `json:"isCurrent"`
	LastActiveAt int64  `json:"lastActiveAt"`
	LoginMethod  string `json:"loginMethod,omitempty"`
	LoggedInAt   int64  `json:"loggedInAt"`
}

// AccountDeletionStatus 账号删除状态响应
type AccountDeletionStatus struct {
	IsPending           bool   `json:"isPending"`
	Status              string `json:"status,omitempty"`
	ScheduledDeletionAt *int64 `json:"scheduledDeletionAt,omitempty"`
	GracePeriodEndsAt   *int64 `json:"gracePeriodEndsAt,omitempty"`
	Reason              string `json:"reason,omitempty"`
}

// ToDeviceInfo 将 LoginHistory 转换为 DeviceInfo
func (lh *LoginHistory) ToDeviceInfo(currentDeviceID string) *DeviceInfo {
	return &DeviceInfo{
		ID:           lh.ID,
		DeviceID:     lh.DeviceID,
		DeviceType:   lh.DeviceType,
		DeviceName:   lh.DeviceName,
		IPAddress:    lh.IPAddress,
		Location:     lh.Location,
		IsCurrent:    lh.DeviceID == currentDeviceID,
		LastActiveAt: lh.LastActiveAt,
		LoginMethod:  lh.LoginMethod,
		LoggedInAt:   lh.LoggedInAt,
	}
}

// TimeToInt64 将 time.Time 转换为 int64 时间戳
func TimeToInt64(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	result := t.Unix()
	return &result
}
