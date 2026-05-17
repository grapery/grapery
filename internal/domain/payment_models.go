package domain

import "github.com/grapestree/fgrapery/grapery/internal/common"

// MembershipStatus 会员状态类型 - 使用 common.MembershipStatus 作为类型别名以保持向后兼容
type MembershipStatus = common.MembershipStatus

const (
	MembershipStatusActive    MembershipStatus = common.MembershipStatusActive
	MembershipStatusExpired   MembershipStatus = common.MembershipStatusExpired
	MembershipStatusCancelled MembershipStatus = common.MembershipStatusCancelled
)

// OrderStatus 订单状态类型 - 使用 common.OrderStatus 作为类型别名以保持向后兼容
type OrderStatus = common.OrderStatus

const (
	OrderStatusPending   OrderStatus = common.OrderStatusPending
	OrderStatusPaid      OrderStatus = common.OrderStatusPaid
	OrderStatusFailed    OrderStatus = common.OrderStatusFailed
	OrderStatusRefunded  OrderStatus = common.OrderStatusRefunded
	OrderStatusCancelled OrderStatus = common.OrderStatusCancelled
)

// MembershipTierType 会员等级类型（字符串枚举）
type MembershipTierType string

const (
	TierTypeFree    MembershipTierType = "free"
	TierTypeBasic   MembershipTierType = "basic"   // 基础会员（兼容历史 tier / 套餐名 pro）
	TierTypePremium MembershipTierType = "premium" // 高级会员（兼容 prime / ultra）
)

// MembershipPeriod 会员周期
type MembershipPeriod string

const (
	PeriodMonthly   MembershipPeriod = "monthly"
	PeriodQuarterly MembershipPeriod = "quarterly"
	PeriodYearly    MembershipPeriod = "yearly"
)

// Membership 会员信息
type Membership struct {
	ID           string `json:"id"`
	UserID       string `json:"userId"`
	Tier         string `json:"tier"`   // free | basic | premium（兼容存量的 pro / prime / ultra）
	Status       string `json:"status"` // active, expired, cancelled
	StartDate    int64  `json:"startDate"`
	EndDate      *int64 `json:"endDate,omitempty"`
	AutoRenew    bool   `json:"autoRenew"`
	TokenQuota   int    `json:"tokenQuota"` // AI token 配额
	TokenUsed    int    `json:"tokenUsed"`
	StorageQuota int64  `json:"storageQuota"` // 存储配额（字节）
	StorageUsed  int64  `json:"storageUsed"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`

	// Relations
	User *User `json:"user,omitempty"`
}

// MembershipPlan 会员方案（新版，用于前端展示）
type MembershipPlan struct {
	ID           string             `json:"id"`
	Tier         MembershipTierType `json:"tier"`
	Period       string             `json:"period"` // monthly, quarterly, yearly
	IAPProductID string             `json:"iapProductId,omitempty"`
	Price        float64            `json:"price"`
	PerMonth     float64            `json:"perMonth"` // 月均价格
	AIQuota      int                `json:"aiQuota"`  // -1 为无限制
	Features     []string           `json:"features"`
	IsActive     bool               `json:"isActive"`
	SortOrder    int                `json:"sortOrder"`
}

// UserMembership 用户会员信息（新版，用于前端展示）
type UserMembership struct {
	common.BaseModel

	UserID          string             `json:"userId"`
	Tier            MembershipTierType `json:"tier"`
	StartedAt       int64              `json:"startedAt"`
	ExpiresAt       int64              `json:"expiresAt"`
	AutoRenew       bool               `json:"autoRenew"`
	AIUsedThisMonth int                `json:"aiUsedThisMonth"`
	AILimit         int                `json:"aiLimit"` // -1 为无限制

	// Relations
	User *User `json:"user,omitempty"`
}

// SubscriptionPlan 订阅计划
type SubscriptionPlan struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"` // Free, Basic, Pro, Enterprise
	IAPProductID   string  `json:"iapProductId,omitempty"`
	MembershipTier string  `json:"membershipTier,omitempty"` // free | basic | premium（兼容 pro / prime / ultra）
	BillingPeriod  string  `json:"billingPeriod,omitempty"`  // monthly, quarterly, yearly
	Price          float64 `json:"price"`                    // 月费
	Currency       string  `json:"currency"`                 // USD, CNY
	TokenQuota     int     `json:"tokenQuota"`               // 每月Token配额
	StorageQuota   int64   `json:"storageQuota"`             // 存储配额（字节）
	MaxStories     int     `json:"maxStories"`               // 最大故事数
	MaxCharacters  int     `json:"maxCharacters"`            // 最大角色数
	Features       string  `json:"features"`                 // JSON格式的功能列表
	IsActive       bool    `json:"isActive"`
	SortOrder      int     `json:"sortOrder"`
	CreatedAt      int64   `json:"createdAt"`
	UpdatedAt      int64   `json:"updatedAt"`
}

// SubscriptionOrder 订阅订单
type SubscriptionOrder struct {
	ID            string  `json:"id"`
	UserID        string  `json:"userId"`
	PlanID        string  `json:"planId"`
	Status        string  `json:"status"` // pending, paid, failed, refunded, cancelled
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	PaymentMethod string  `json:"paymentMethod"`       // alipay, wechat, stripe, paypal
	PaymentID     string  `json:"paymentId,omitempty"` // 第三方支付ID
	StartDate     int64   `json:"startDate"`
	EndDate       int64   `json:"endDate"`
	InvoiceURL    string  `json:"invoiceUrl,omitempty"`
	CreatedAt     int64   `json:"createdAt"`
	PaidAt        *int64  `json:"paidAt,omitempty"`
	UpdatedAt     int64   `json:"updatedAt"`

	// Relations
	User *User             `json:"user,omitempty"`
	Plan *SubscriptionPlan `json:"plan,omitempty"`
}

// TokenTransaction Token消费/充值记录
type TokenTransaction struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Type        string `json:"type"`                // consume, recharge, refund, gift
	Amount      int    `json:"amount"`              // 正数为充值，负数为消费
	Balance     int    `json:"balance"`             // 交易后余额
	Source      string `json:"source"`              // ai_generation, render, subscription, manual
	RelatedID   string `json:"relatedId,omitempty"` // 关联的任务ID或订单ID
	Description string `json:"description"`
	CreatedAt   int64  `json:"createdAt"`

	// Relations
	User *User `json:"user,omitempty"`
}

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	Tier   MembershipTierType `json:"tier"`
	Period MembershipPeriod   `json:"period"`
}

// SubscribeResponse 订阅响应
type SubscribeResponse struct {
	OrderID    string `json:"orderId"`
	PaymentURL string `json:"paymentUrl,omitempty"`
	Status     string `json:"status"`
}
