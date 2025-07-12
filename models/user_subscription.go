package models

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// UserSubscriptionStatus 用户订阅状态
type UserSubscriptionStatus int

const (
	UserSubscriptionStatusActive      UserSubscriptionStatus = iota + 1 // 活跃
	UserSubscriptionStatusExpired                                       // 已过期
	UserSubscriptionStatusCanceled                                      // 已取消
	UserSubscriptionStatusPending                                       // 待支付
	UserSubscriptionStatusFailed                                        // 支付失败
	UserSubscriptionStatusTrial                                         // 试用中
	UserSubscriptionStatusPaused                                        // 暂停
	UserSubscriptionStatusUpgrading                                     // 升级中
	UserSubscriptionStatusDowngrading                                   // 降级中
)

// UserSubscription 用户订阅记录模型
type UserSubscription struct {
	IDBase
	UserID          int64                  `gorm:"column:user_id;not null;index" json:"user_id"`                     // 用户ID
	PackagePlanID   uint                   `gorm:"column:package_plan_id;not null;index" json:"package_plan_id"`     // 套餐计划ID
	OrderID         uint                   `gorm:"column:order_id;not null;index" json:"order_id"`                   // 订单ID
	Status          UserSubscriptionStatus `gorm:"column:status;default:1" json:"status"`                            // 订阅状态
	StartTime       time.Time              `gorm:"column:start_time;not null" json:"start_time"`                     // 开始时间
	EndTime         time.Time              `gorm:"column:end_time;not null" json:"end_time"`                         // 结束时间
	AutoRenew       bool                   `gorm:"column:auto_renew;default:true" json:"auto_renew"`                 // 自动续费
	PaymentMethod   PaymentMethod          `gorm:"column:payment_method;not null" json:"payment_method"`             // 支付方式
	PaymentProvider string                 `gorm:"column:payment_provider;size:50;not null" json:"payment_provider"` // 支付提供商
	ProviderSubID   string                 `gorm:"column:provider_sub_id;size:255" json:"provider_sub_id"`           // 第三方订阅ID
	Amount          int64                  `gorm:"column:amount;not null" json:"amount"`                             // 订阅金额（分）
	Currency        string                 `gorm:"column:currency;size:10;default:'CNY'" json:"currency"`            // 货币类型

	// 服务能力配置（从套餐计划复制，便于查询）
	QuotaLimit      int    `gorm:"column:quota_limit;default:1000" json:"quota_limit"`        // 额度限制
	QuotaUsed       int    `gorm:"column:quota_used;default:0" json:"quota_used"`             // 已使用额度
	MaxRoles        int    `gorm:"column:max_roles;default:2" json:"max_roles"`               // 最大角色数
	MaxContexts     int    `gorm:"column:max_contexts;default:5" json:"max_contexts"`         // 最大上下文数
	AvailableModels string `gorm:"column:available_models;type:text" json:"available_models"` // 可用模型（JSON数组）
	Features        string `gorm:"column:features;type:text" json:"features"`                 // 功能特性（JSON对象）

	// 订阅管理
	CancelReason    string     `gorm:"column:cancel_reason;size:255" json:"cancel_reason"` // 取消原因
	CanceledAt      *time.Time `gorm:"column:canceled_at" json:"canceled_at"`              // 取消时间
	CanceledBy      int64      `gorm:"column:canceled_by" json:"canceled_by"`              // 取消者ID
	NextBillingDate *time.Time `gorm:"column:next_billing_date" json:"next_billing_date"`  // 下次计费时间
	TrialStartTime  *time.Time `gorm:"column:trial_start_time" json:"trial_start_time"`    // 试用开始时间
	TrialEndTime    *time.Time `gorm:"column:trial_end_time" json:"trial_end_time"`        // 试用结束时间
	PauseStartTime  *time.Time `gorm:"column:pause_start_time" json:"pause_start_time"`    // 暂停开始时间
	PauseEndTime    *time.Time `gorm:"column:pause_end_time" json:"pause_end_time"`        // 暂停结束时间
	UpgradeFromID   *uint      `gorm:"column:upgrade_from_id" json:"upgrade_from_id"`      // 升级前订阅ID
	DowngradeToID   *uint      `gorm:"column:downgrade_to_id" json:"downgrade_to_id"`      // 降级后订阅ID

	// 统计信息
	TotalPaid    int64 `gorm:"column:total_paid;default:0" json:"total_paid"`       // 总支付金额
	PaymentCount int   `gorm:"column:payment_count;default:0" json:"payment_count"` // 支付次数
	RefundCount  int   `gorm:"column:refund_count;default:0" json:"refund_count"`   // 退款次数
	RefundAmount int64 `gorm:"column:refund_amount;default:0" json:"refund_amount"` // 退款金额

	// 元数据
	Metadata string `gorm:"column:metadata;type:text" json:"metadata"` // 元数据（JSON）
	Notes    string `gorm:"column:notes;type:text" json:"notes"`       // 备注
}

func (us UserSubscription) TableName() string {
	return "user_subscriptions"
}

// GetAvailableModels 获取可用模型列表
func (us *UserSubscription) GetAvailableModels() ([]string, error) {
	if us.AvailableModels == "" {
		return []string{"gpt-3.5-turbo"}, nil // 默认模型
	}
	var models []string
	err := json.Unmarshal([]byte(us.AvailableModels), &models)
	return models, err
}

// SetAvailableModels 设置可用模型列表
func (us *UserSubscription) SetAvailableModels(models []string) error {
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	us.AvailableModels = string(data)
	return nil
}

// GetFeatures 获取功能特性
func (us *UserSubscription) GetFeatures() (map[string]interface{}, error) {
	if us.Features == "" {
		return map[string]interface{}{}, nil
	}
	var features map[string]interface{}
	err := json.Unmarshal([]byte(us.Features), &features)
	return features, err
}

// SetFeatures 设置功能特性
func (us *UserSubscription) SetFeatures(features map[string]interface{}) error {
	data, err := json.Marshal(features)
	if err != nil {
		return err
	}
	us.Features = string(data)
	return nil
}

// GetMetadata 获取元数据
func (us *UserSubscription) GetMetadata() (map[string]interface{}, error) {
	if us.Metadata == "" {
		return map[string]interface{}{}, nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(us.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
func (us *UserSubscription) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	us.Metadata = string(data)
	return nil
}

// IsActive 检查订阅是否活跃
func (us *UserSubscription) IsActive() bool {
	return us.Status == UserSubscriptionStatusActive && time.Now().Before(us.EndTime)
}

// IsExpired 检查订阅是否过期
func (us *UserSubscription) IsExpired() bool {
	return time.Now().After(us.EndTime)
}

// IsTrial 检查是否在试用期
func (us *UserSubscription) IsTrial() bool {
	if us.TrialStartTime == nil || us.TrialEndTime == nil {
		return false
	}
	now := time.Now()
	return now.After(*us.TrialStartTime) && now.Before(*us.TrialEndTime)
}

// IsPaused 检查是否暂停
func (us *UserSubscription) IsPaused() bool {
	return us.Status == UserSubscriptionStatusPaused
}

// HasQuota 检查是否有剩余额度
func (us *UserSubscription) HasQuota() bool {
	return us.QuotaUsed < us.QuotaLimit
}

// GetRemainingQuota 获取剩余额度
func (us *UserSubscription) GetRemainingQuota() int {
	remaining := us.QuotaLimit - us.QuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetQuotaUsagePercentage 获取额度使用百分比
func (us *UserSubscription) GetQuotaUsagePercentage() float64 {
	if us.QuotaLimit == 0 {
		return 0
	}
	return float64(us.QuotaUsed) / float64(us.QuotaLimit) * 100
}

// GetDaysRemaining 获取剩余天数
func (us *UserSubscription) GetDaysRemaining() int {
	remaining := us.EndTime.Sub(time.Now())
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Hours() / 24)
}

// CreateUserSubscription 创建用户订阅
func CreateUserSubscription(ctx context.Context, subscription *UserSubscription) error {
	return DataBase().WithContext(ctx).Create(subscription).Error
}

// GetUserSubscription 获取用户订阅
func GetUserSubscription(ctx context.Context, id uint) (*UserSubscription, error) {
	var subscription UserSubscription
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetUserActiveSubscriptionByUserID 获取用户活跃订阅
func GetUserActiveSubscriptionByUserID(ctx context.Context, userID int64) (*UserSubscription, error) {
	var subscription UserSubscription
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND status = ? AND end_time > ?", userID, UserSubscriptionStatusActive, time.Now()).
		Order("end_time DESC").
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetUserSubscriptionsByUserID 获取用户所有订阅
func GetUserSubscriptionsByUserID(ctx context.Context, userID int64, offset, limit int) ([]*UserSubscription, error) {
	var subscriptions []*UserSubscription
	err := DataBase().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// GetUserSubscriptionsByStatus 根据状态获取用户订阅
func GetUserSubscriptionsByStatus(ctx context.Context, userID int64, status UserSubscriptionStatus, offset, limit int) ([]*UserSubscription, error) {
	var subscriptions []*UserSubscription
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, status).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// UpdateUserSubscriptionStatus 更新用户订阅状态
func UpdateUserSubscriptionStatus(ctx context.Context, id uint, status UserSubscriptionStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	now := time.Now()
	switch status {
	case UserSubscriptionStatusCanceled:
		updates["canceled_at"] = &now
	case UserSubscriptionStatusPaused:
		updates["pause_start_time"] = &now
	}

	return DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateUserSubscriptionQuota 更新用户订阅额度使用情况
func UpdateUserSubscriptionQuota(ctx context.Context, id uint, quotaUsed int) error {
	return DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Where("id = ?", id).
		Update("quota_used", quotaUsed).Error
}

// ConsumeUserSubscriptionQuota 消耗用户订阅额度
func ConsumeUserSubscriptionQuota(ctx context.Context, id uint, amount int) error {
	return DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Where("id = ? AND quota_used + ? <= quota_limit", id, amount).
		Update("quota_used", gorm.Expr("quota_used + ?", amount)).Error
}

// CancelUserSubscription 取消用户订阅
func CancelUserSubscription(ctx context.Context, id uint, reason string, canceledBy int64) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        UserSubscriptionStatusCanceled,
			"cancel_reason": reason,
			"canceled_at":   &now,
			"canceled_by":   canceledBy,
			"auto_renew":    false,
		}).Error
}

// PauseUserSubscription 暂停用户订阅
func PauseUserSubscription(ctx context.Context, id uint, reason string) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":           UserSubscriptionStatusPaused,
			"pause_start_time": &now,
			"notes":            reason,
		}).Error
}

// ResumeUserSubscription 恢复用户订阅
func ResumeUserSubscription(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         UserSubscriptionStatusActive,
			"pause_end_time": time.Now(),
		}).Error
}

// UpgradeUserSubscription 升级用户订阅
func UpgradeUserSubscription(ctx context.Context, id uint, newPackagePlanID uint) error {
	return DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          UserSubscriptionStatusUpgrading,
			"package_plan_id": newPackagePlanID,
		}).Error
}

// DowngradeUserSubscription 降级用户订阅
func DowngradeUserSubscription(ctx context.Context, id uint, newPackagePlanID uint) error {
	return DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          UserSubscriptionStatusDowngrading,
			"package_plan_id": newPackagePlanID,
		}).Error
}

// GetExpiredUserSubscriptions 获取已过期的用户订阅
func GetExpiredUserSubscriptions(ctx context.Context) ([]*UserSubscription, error) {
	var subscriptions []*UserSubscription
	err := DataBase().WithContext(ctx).
		Where("status = ? AND end_time <= ?", UserSubscriptionStatusActive, time.Now()).
		Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// GetUserSubscriptionsByProvider 根据支付提供商获取用户订阅
func GetUserSubscriptionsByProvider(ctx context.Context, provider, providerSubID string) (*UserSubscription, error) {
	var subscription UserSubscription
	err := DataBase().WithContext(ctx).
		Where("payment_provider = ? AND provider_sub_id = ?", provider, providerSubID).
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// UpdateUserSubscription 更新用户订阅信息
func UpdateUserSubscription(ctx context.Context, subscription *UserSubscription) error {
	return DataBase().WithContext(ctx).Save(subscription).Error
}

// GetUserSubscriptionStats 获取用户订阅统计信息
func GetUserSubscriptionStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	var stats struct {
		TotalSubscriptions    int64 `json:"total_subscriptions"`
		ActiveSubscriptions   int64 `json:"active_subscriptions"`
		ExpiredSubscriptions  int64 `json:"expired_subscriptions"`
		CanceledSubscriptions int64 `json:"canceled_subscriptions"`
		TotalPaid             int64 `json:"total_paid"`
		TotalRefunded         int64 `json:"total_refunded"`
		CurrentQuotaUsed      int   `json:"current_quota_used"`
		CurrentQuotaLimit     int   `json:"current_quota_limit"`
	}

	err := DataBase().WithContext(ctx).
		Model(&UserSubscription{}).
		Select(`
			COUNT(*) as total_subscriptions,
			SUM(CASE WHEN status = 1 AND end_time > NOW() THEN 1 ELSE 0 END) as active_subscriptions,
			SUM(CASE WHEN end_time <= NOW() THEN 1 ELSE 0 END) as expired_subscriptions,
			SUM(CASE WHEN status = 3 THEN 1 ELSE 0 END) as canceled_subscriptions,
			SUM(total_paid) as total_paid,
			SUM(refund_amount) as total_refunded,
			SUM(CASE WHEN status = 1 AND end_time > NOW() THEN quota_used ELSE 0 END) as current_quota_used,
			SUM(CASE WHEN status = 1 AND end_time > NOW() THEN quota_limit ELSE 0 END) as current_quota_limit
		`).
		Where("user_id = ?", userID).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_subscriptions":     stats.TotalSubscriptions,
		"active_subscriptions":    stats.ActiveSubscriptions,
		"expired_subscriptions":   stats.ExpiredSubscriptions,
		"canceled_subscriptions":  stats.CanceledSubscriptions,
		"total_paid":              stats.TotalPaid,
		"total_refunded":          stats.TotalRefunded,
		"current_quota_used":      stats.CurrentQuotaUsed,
		"current_quota_limit":     stats.CurrentQuotaLimit,
		"current_quota_remaining": stats.CurrentQuotaLimit - stats.CurrentQuotaUsed,
	}, nil
}
