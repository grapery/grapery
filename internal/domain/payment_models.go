package domain

// Membership 会员信息
type Membership struct {
	ID           string `json:"id"`
	UserID       string `json:"userId"`
	Tier         string `json:"tier"`   // free, basic, pro, enterprise
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

// SubscriptionPlan 订阅计划
type SubscriptionPlan struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`          // Free, Basic, Pro, Enterprise
	Price         float64 `json:"price"`         // 月费
	Currency      string  `json:"currency"`      // USD, CNY
	TokenQuota    int     `json:"tokenQuota"`    // 每月Token配额
	StorageQuota  int64   `json:"storageQuota"`  // 存储配额（字节）
	MaxStories    int     `json:"maxStories"`    // 最大故事数
	MaxCharacters int     `json:"maxCharacters"` // 最大角色数
	Features      string  `json:"features"`      // JSON格式的功能列表
	IsActive      bool    `json:"isActive"`
	SortOrder     int     `json:"sortOrder"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
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
