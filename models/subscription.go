package models

import (
	"context"
	"encoding/json"
	"time"
)

// SubscriptionStatus 订阅状态
type SubscriptionStatus int

const (
	SubscriptionStatusActive   SubscriptionStatus = iota + 1 // 活跃
	SubscriptionStatusExpired                                // 已过期
	SubscriptionStatusCanceled                               // 已取消
	SubscriptionStatusPending                                // 待支付
	SubscriptionStatusFailed                                 // 支付失败
)

// Subscription 订阅模型
type Subscription struct {
	IDBase
	UserID          int64              `gorm:"column:user_id;not null;index" json:"user_id"`              // 用户ID
	ProductID       uint               `gorm:"column:product_id;not null" json:"product_id"`              // 商品ID
	OrderID         uint               `gorm:"column:order_id;not null" json:"order_id"`                  // 订单ID
	Status          SubscriptionStatus `gorm:"column:status;default:1" json:"status"`                     // 订阅状态
	StartTime       time.Time          `gorm:"column:start_time;not null" json:"start_time"`              // 开始时间
	EndTime         time.Time          `gorm:"column:end_time;not null" json:"end_time"`                  // 结束时间
	AutoRenew       bool               `gorm:"column:auto_renew;default:true" json:"auto_renew"`          // 自动续费
	PaymentMethod   string             `gorm:"column:payment_method;size:50" json:"payment_method"`       // 支付方式
	PaymentProvider string             `gorm:"column:payment_provider;size:50" json:"payment_provider"`   // 支付提供商
	ProviderSubID   string             `gorm:"column:provider_sub_id;size:255" json:"provider_sub_id"`    // 第三方订阅ID
	Amount          int64              `gorm:"column:amount;not null" json:"amount"`                      // 订阅金额（分）
	Currency        string             `gorm:"column:currency;size:10;default:'CNY'" json:"currency"`     // 货币类型
	QuotaLimit      int                `gorm:"column:quota_limit;default:1000" json:"quota_limit"`        // 额度限制
	QuotaUsed       int                `gorm:"column:quota_used;default:0" json:"quota_used"`             // 已使用额度
	MaxRoles        int                `gorm:"column:max_roles;default:2" json:"max_roles"`               // 最大角色数
	MaxContexts     int                `gorm:"column:max_contexts;default:5" json:"max_contexts"`         // 最大上下文数
	AvailableModels string             `gorm:"column:available_models;type:text" json:"available_models"` // 可用模型（JSON数组）
	CancelReason    string             `gorm:"column:cancel_reason;size:255" json:"cancel_reason"`        // 取消原因
	CanceledAt      *time.Time         `gorm:"column:canceled_at" json:"canceled_at"`                     // 取消时间
	CanceledBy      int64              `gorm:"column:canceled_by" json:"canceled_by"`                     // 取消者ID
	NextBillingDate *time.Time         `gorm:"column:next_billing_date" json:"next_billing_date"`         // 下次计费时间
	Metadata        string             `gorm:"column:metadata;type:text" json:"metadata"`                 // 元数据（JSON）
}

func (s Subscription) TableName() string {
	return "subscriptions"
}

// CreateSubscription 创建订阅
func CreateSubscription(ctx context.Context, subscription *Subscription) error {
	return DataBase().WithContext(ctx).Create(subscription).Error
}

// GetSubscription 获取订阅信息
func GetSubscription(ctx context.Context, id uint) (*Subscription, error) {
	var subscription Subscription
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetUserActiveSubscription 获取用户活跃订阅
func GetUserActiveSubscription(ctx context.Context, userID int64) (*Subscription, error) {
	var subscription Subscription
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND status = ? AND end_time > ?", userID, SubscriptionStatusActive, time.Now()).
		Order("end_time DESC").
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetUserSubscriptions 获取用户所有订阅
func GetUserSubscriptions(ctx context.Context, userID int64, offset, limit int) ([]*Subscription, error) {
	var subscriptions []*Subscription
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

// UpdateSubscriptionStatus 更新订阅状态
func UpdateSubscriptionStatus(ctx context.Context, id uint, status SubscriptionStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == SubscriptionStatusCanceled {
		now := time.Now()
		updates["canceled_at"] = &now
	}

	return DataBase().WithContext(ctx).
		Model(&Subscription{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateSubscriptionQuota 更新订阅额度使用情况
func UpdateSubscriptionQuota(ctx context.Context, id uint, quotaUsed int) error {
	return DataBase().WithContext(ctx).
		Model(&Subscription{}).
		Where("id = ?", id).
		Update("quota_used", quotaUsed).Error
}

// CancelSubscription 取消订阅
func CancelSubscription(ctx context.Context, id uint, reason string, canceledBy int64) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&Subscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        SubscriptionStatusCanceled,
			"cancel_reason": reason,
			"canceled_at":   &now,
			"canceled_by":   canceledBy,
			"auto_renew":    false,
		}).Error
}

// GetExpiredSubscriptions 获取已过期的订阅
func GetExpiredSubscriptions(ctx context.Context) ([]*Subscription, error) {
	var subscriptions []*Subscription
	err := DataBase().WithContext(ctx).
		Where("status = ? AND end_time <= ?", SubscriptionStatusActive, time.Now()).
		Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// GetSubscriptionsByProvider 根据支付提供商获取订阅
func GetSubscriptionsByProvider(ctx context.Context, provider, providerSubID string) (*Subscription, error) {
	var subscription Subscription
	err := DataBase().WithContext(ctx).
		Where("payment_provider = ? AND provider_sub_id = ?", provider, providerSubID).
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// UpdateSubscription 更新订阅信息
func UpdateSubscription(ctx context.Context, subscription *Subscription) error {
	return DataBase().WithContext(ctx).Save(subscription).Error
}

// GetAvailableModels 获取可用模型列表
func (s *Subscription) GetAvailableModels() ([]string, error) {
	if s.AvailableModels == "" {
		return []string{"gpt-3.5-turbo"}, nil // 默认模型
	}

	// 解析JSON数组
	var models []string
	if err := json.Unmarshal([]byte(s.AvailableModels), &models); err != nil {
		return []string{"gpt-3.5-turbo"}, nil // 解析失败时返回默认模型
	}

	return models, nil
}
