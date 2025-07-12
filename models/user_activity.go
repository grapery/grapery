package models

import (
	"context"
	"encoding/json"
	"time"
)

// ActivityType 活动类型
type ActivityType string

const (
	ActivityTypeSubscriptionCreated    ActivityType = "subscription_created"     // 订阅创建
	ActivityTypeSubscriptionRenewed    ActivityType = "subscription_renewed"     // 订阅续费
	ActivityTypeSubscriptionCanceled   ActivityType = "subscription_canceled"    // 订阅取消
	ActivityTypeSubscriptionUpgraded   ActivityType = "subscription_upgraded"    // 订阅升级
	ActivityTypeSubscriptionDowngraded ActivityType = "subscription_downgraded"  // 订阅降级
	ActivityTypeSubscriptionPaused     ActivityType = "subscription_paused"      // 订阅暂停
	ActivityTypeSubscriptionResumed    ActivityType = "subscription_resumed"     // 订阅恢复
	ActivityTypeSubscriptionExpired    ActivityType = "subscription_expired"     // 订阅过期
	ActivityTypePaymentSuccess         ActivityType = "payment_success"          // 支付成功
	ActivityTypePaymentFailed          ActivityType = "payment_failed"           // 支付失败
	ActivityTypePaymentRefunded        ActivityType = "payment_refunded"         // 支付退款
	ActivityTypePaymentPartialRefunded ActivityType = "payment_partial_refunded" // 部分退款
	ActivityTypeRecharge               ActivityType = "recharge"                 // 充值
	ActivityTypeQuotaConsumed          ActivityType = "quota_consumed"           // 额度消耗
	ActivityTypeQuotaReplenished       ActivityType = "quota_replenished"        // 额度补充
	ActivityTypeTrialStarted           ActivityType = "trial_started"            // 试用开始
	ActivityTypeTrialEnded             ActivityType = "trial_ended"              // 试用结束
	ActivityTypePackagePurchased       ActivityType = "package_purchased"        // 套餐购买
	ActivityTypePackageUpgraded        ActivityType = "package_upgraded"         // 套餐升级
	ActivityTypePackageDowngraded      ActivityType = "package_downgraded"       // 套餐降级
	ActivityTypeSupportRequest         ActivityType = "support_request"          // 客服请求
	ActivityTypeAccountSuspended       ActivityType = "account_suspended"        // 账户暂停
	ActivityTypeAccountReactivated     ActivityType = "account_reactivated"      // 账户恢复
)

// ActivityLevel 活动级别
type ActivityLevel string

const (
	ActivityLevelInfo    ActivityLevel = "info"    // 信息
	ActivityLevelWarning ActivityLevel = "warning" // 警告
	ActivityLevelError   ActivityLevel = "error"   // 错误
	ActivityLevelSuccess ActivityLevel = "success" // 成功
)

// UserActivity 用户活动记录模型
type UserActivity struct {
	IDBase
	UserID         int64         `gorm:"column:user_id;not null;index" json:"user_id"`                       // 用户ID
	ActivityType   ActivityType  `gorm:"column:activity_type;size:50;not null" json:"activity_type"`         // 活动类型
	ActivityLevel  ActivityLevel `gorm:"column:activity_level;size:20;default:'info'" json:"activity_level"` // 活动级别
	Title          string        `gorm:"column:title;size:255;not null" json:"title"`                        // 活动标题
	Description    string        `gorm:"column:description;type:text" json:"description"`                    // 活动描述
	Amount         int64         `gorm:"column:amount;default:0" json:"amount"`                              // 涉及金额（分）
	Currency       string        `gorm:"column:currency;size:10;default:'CNY'" json:"currency"`              // 货币类型
	OrderID        *uint         `gorm:"column:order_id;index" json:"order_id"`                              // 关联订单ID
	SubscriptionID *uint         `gorm:"column:subscription_id;index" json:"subscription_id"`                // 关联订阅ID
	PaymentID      *uint         `gorm:"column:payment_id;index" json:"payment_id"`                          // 关联支付ID
	PackagePlanID  *uint         `gorm:"column:package_plan_id;index" json:"package_plan_id"`                // 关联套餐ID
	IPAddress      string        `gorm:"column:ip_address;size:45" json:"ip_address"`                        // IP地址
	UserAgent      string        `gorm:"column:user_agent;size:500" json:"user_agent"`                       // 用户代理
	DeviceInfo     string        `gorm:"column:device_info;size:500" json:"device_info"`                     // 设备信息
	Location       string        `gorm:"column:location;size:255" json:"location"`                           // 地理位置
	Metadata       string        `gorm:"column:metadata;type:text" json:"metadata"`                          // 元数据（JSON）
	Notes          string        `gorm:"column:notes;type:text" json:"notes"`                                // 备注
	ProcessedBy    int64         `gorm:"column:processed_by" json:"processed_by"`                            // 处理人ID
	ProcessedAt    *time.Time    `gorm:"column:processed_at" json:"processed_at"`                            // 处理时间
	IsRead         bool          `gorm:"column:is_read;default:false" json:"is_read"`                        // 是否已读
	ReadAt         *time.Time    `gorm:"column:read_at" json:"read_at"`                                      // 阅读时间
	IsResolved     bool          `gorm:"column:is_resolved;default:false" json:"is_resolved"`                // 是否已解决
	ResolvedAt     *time.Time    `gorm:"column:resolved_at" json:"resolved_at"`                              // 解决时间
	ResolvedBy     int64         `gorm:"column:resolved_by" json:"resolved_by"`                              // 解决人ID
	ResolutionNote string        `gorm:"column:resolution_note;type:text" json:"resolution_note"`            // 解决方案
}

func (ua UserActivity) TableName() string {
	return "user_activities"
}

// GetMetadata 获取元数据
func (ua *UserActivity) GetMetadata() (map[string]interface{}, error) {
	if ua.Metadata == "" {
		return map[string]interface{}{}, nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(ua.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
func (ua *UserActivity) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	ua.Metadata = string(data)
	return nil
}

// IsHighPriority 检查是否为高优先级活动
func (ua *UserActivity) IsHighPriority() bool {
	return ua.ActivityLevel == ActivityLevelError || ua.ActivityLevel == ActivityLevelWarning
}

// IsPaymentRelated 检查是否为支付相关活动
func (ua *UserActivity) IsPaymentRelated() bool {
	return ua.ActivityType == ActivityTypePaymentSuccess ||
		ua.ActivityType == ActivityTypePaymentFailed ||
		ua.ActivityType == ActivityTypePaymentRefunded ||
		ua.ActivityType == ActivityTypePaymentPartialRefunded ||
		ua.ActivityType == ActivityTypeRecharge
}

// IsSubscriptionRelated 检查是否为订阅相关活动
func (ua *UserActivity) IsSubscriptionRelated() bool {
	return ua.ActivityType == ActivityTypeSubscriptionCreated ||
		ua.ActivityType == ActivityTypeSubscriptionRenewed ||
		ua.ActivityType == ActivityTypeSubscriptionCanceled ||
		ua.ActivityType == ActivityTypeSubscriptionUpgraded ||
		ua.ActivityType == ActivityTypeSubscriptionDowngraded ||
		ua.ActivityType == ActivityTypeSubscriptionPaused ||
		ua.ActivityType == ActivityTypeSubscriptionResumed ||
		ua.ActivityType == ActivityTypeSubscriptionExpired
}

// CreateUserActivity 创建用户活动记录
func CreateUserActivity(ctx context.Context, activity *UserActivity) error {
	return DataBase().WithContext(ctx).Create(activity).Error
}

// GetUserActivity 获取用户活动记录
func GetUserActivity(ctx context.Context, id uint) (*UserActivity, error) {
	var activity UserActivity
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&activity).Error
	if err != nil {
		return nil, err
	}
	return &activity, nil
}

// GetUserActivities 获取用户活动记录列表
func GetUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*UserActivity, error) {
	var activities []*UserActivity
	err := DataBase().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// GetUserActivitiesByType 根据类型获取用户活动记录
func GetUserActivitiesByType(ctx context.Context, userID int64, activityType ActivityType, offset, limit int) ([]*UserActivity, error) {
	var activities []*UserActivity
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND activity_type = ?", userID, activityType).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// GetUserActivitiesByLevel 根据级别获取用户活动记录
func GetUserActivitiesByLevel(ctx context.Context, userID int64, level ActivityLevel, offset, limit int) ([]*UserActivity, error) {
	var activities []*UserActivity
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND activity_level = ?", userID, level).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// GetUnreadUserActivities 获取未读的用户活动记录
func GetUnreadUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*UserActivity, error) {
	var activities []*UserActivity
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND is_read = ?", userID, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// GetUnresolvedUserActivities 获取未解决的用户活动记录
func GetUnresolvedUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*UserActivity, error) {
	var activities []*UserActivity
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND is_resolved = ?", userID, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// GetHighPriorityUserActivities 获取高优先级的用户活动记录
func GetHighPriorityUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*UserActivity, error) {
	var activities []*UserActivity
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND activity_level IN (?, ?)", userID, ActivityLevelError, ActivityLevelWarning).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// MarkUserActivityAsRead 标记用户活动为已读
func MarkUserActivityAsRead(ctx context.Context, id uint) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&UserActivity{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		}).Error
}

// MarkUserActivityAsResolved 标记用户活动为已解决
func MarkUserActivityAsResolved(ctx context.Context, id uint, resolvedBy int64, resolutionNote string) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&UserActivity{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_resolved":     true,
			"resolved_at":     &now,
			"resolved_by":     resolvedBy,
			"resolution_note": resolutionNote,
		}).Error
}

// UpdateUserActivity 更新用户活动记录
func UpdateUserActivity(ctx context.Context, activity *UserActivity) error {
	return DataBase().WithContext(ctx).Save(activity).Error
}

// GetUserActivityStats 获取用户活动统计信息
func GetUserActivityStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	var stats struct {
		TotalActivities        int64 `json:"total_activities"`
		UnreadActivities       int64 `json:"unread_activities"`
		UnresolvedActivities   int64 `json:"unresolved_activities"`
		HighPriorityActivities int64 `json:"high_priority_activities"`
		PaymentActivities      int64 `json:"payment_activities"`
		SubscriptionActivities int64 `json:"subscription_activities"`
		ErrorActivities        int64 `json:"error_activities"`
		WarningActivities      int64 `json:"warning_activities"`
		SuccessActivities      int64 `json:"success_activities"`
	}

	err := DataBase().WithContext(ctx).
		Model(&UserActivity{}).
		Select(`
			COUNT(*) as total_activities,
			SUM(CASE WHEN is_read = 0 THEN 1 ELSE 0 END) as unread_activities,
			SUM(CASE WHEN is_resolved = 0 THEN 1 ELSE 0 END) as unresolved_activities,
			SUM(CASE WHEN activity_level IN ('error', 'warning') THEN 1 ELSE 0 END) as high_priority_activities,
			SUM(CASE WHEN activity_type LIKE '%payment%' OR activity_type = 'recharge' THEN 1 ELSE 0 END) as payment_activities,
			SUM(CASE WHEN activity_type LIKE '%subscription%' THEN 1 ELSE 0 END) as subscription_activities,
			SUM(CASE WHEN activity_level = 'error' THEN 1 ELSE 0 END) as error_activities,
			SUM(CASE WHEN activity_level = 'warning' THEN 1 ELSE 0 END) as warning_activities,
			SUM(CASE WHEN activity_level = 'success' THEN 1 ELSE 0 END) as success_activities
		`).
		Where("user_id = ?", userID).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_activities":         stats.TotalActivities,
		"unread_activities":        stats.UnreadActivities,
		"unresolved_activities":    stats.UnresolvedActivities,
		"high_priority_activities": stats.HighPriorityActivities,
		"payment_activities":       stats.PaymentActivities,
		"subscription_activities":  stats.SubscriptionActivities,
		"error_activities":         stats.ErrorActivities,
		"warning_activities":       stats.WarningActivities,
		"success_activities":       stats.SuccessActivities,
	}, nil
}

// GetRecentUserActivities 获取最近的用户活动记录
func GetRecentUserActivities(ctx context.Context, userID int64, days int, limit int) ([]*UserActivity, error) {
	var activities []*UserActivity
	startTime := time.Now().AddDate(0, 0, -days)

	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND created_at >= ?", userID, startTime).
		Order("created_at DESC").
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// GetUserActivitiesByDateRange 根据日期范围获取用户活动记录
func GetUserActivitiesByDateRange(ctx context.Context, userID int64, startTime, endTime time.Time, offset, limit int) ([]*UserActivity, error) {
	var activities []*UserActivity
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startTime, endTime).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// CreateSubscriptionActivity 创建订阅相关活动记录
func CreateSubscriptionActivity(ctx context.Context, userID int64, activityType ActivityType, subscriptionID uint, title, description string, amount int64) error {
	activity := &UserActivity{
		UserID:         userID,
		ActivityType:   activityType,
		ActivityLevel:  ActivityLevelInfo,
		Title:          title,
		Description:    description,
		Amount:         amount,
		Currency:       "CNY",
		SubscriptionID: &subscriptionID,
	}
	return CreateUserActivity(ctx, activity)
}

// CreatePaymentActivity 创建支付相关活动记录
func CreatePaymentActivity(ctx context.Context, userID int64, activityType ActivityType, paymentID uint, orderID uint, title, description string, amount int64) error {
	activity := &UserActivity{
		UserID:       userID,
		ActivityType: activityType,
		ActivityLevel: func() ActivityLevel {
			switch activityType {
			case ActivityTypePaymentSuccess, ActivityTypeRecharge:
				return ActivityLevelSuccess
			case ActivityTypePaymentFailed:
				return ActivityLevelError
			case ActivityTypePaymentRefunded, ActivityTypePaymentPartialRefunded:
				return ActivityLevelWarning
			default:
				return ActivityLevelInfo
			}
		}(),
		Title:       title,
		Description: description,
		Amount:      amount,
		Currency:    "CNY",
		PaymentID:   &paymentID,
		OrderID:     &orderID,
	}
	return CreateUserActivity(ctx, activity)
}

// CreateQuotaActivity 创建额度相关活动记录
func CreateQuotaActivity(ctx context.Context, userID int64, activityType ActivityType, subscriptionID uint, title, description string, amount int) error {
	activity := &UserActivity{
		UserID:         userID,
		ActivityType:   activityType,
		ActivityLevel:  ActivityLevelInfo,
		Title:          title,
		Description:    description,
		SubscriptionID: &subscriptionID,
	}
	return CreateUserActivity(ctx, activity)
}
