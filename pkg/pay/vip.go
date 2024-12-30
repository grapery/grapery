package pay

import (
	"time"
)

// 实现会员系统，用户购买会员后，可以享受会员特权
// 会员特权包括：
// 1. 使用AI助手
// 2. 创建更多角色，一般用户创建2个角色，会员用户创建5个角色
// 3. 创建更多聊天上下文，一般用户创建5个，会员用户创建100个
// 4. 使用更多模型，一般用户使用gpt-3.5-turbo，会员用户使用gpt-4o
// 5. 管理会员自动续费
// 6. 管理会员用量和额度
// 7. 管理会员到期时间
// 8. 管理会员状态
// 9. 管理会员订单以及订单流转状态

// VIPLevel 会员等级
type VIPLevel int

const (
	VIPLevelNone  VIPLevel = iota // 非会员
	VIPLevelBasic                 // 基础会员
	VIPLevelPro                   // 高级会员
)

// VIPStatus 会员状态
type VIPStatus int

const (
	VIPStatusInactive VIPStatus = iota // 未激活
	VIPStatusActive                    // 已激活
	VIPStatusExpired                   // 已过期
)

// VIPInfo 会员信息
type VIPInfo struct {
	UserID     int64     `json:"user_id"`
	Level      VIPLevel  `json:"level"`
	Status     VIPStatus `json:"status"`
	ExpireTime time.Time `json:"expire_time"`
	AutoRenew  bool      `json:"auto_renew"`
	QuotaLimit int       `json:"quota_limit"` // 使用额度上限
	QuotaUsed  int       `json:"quota_used"`  // 已使用额度
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// VIPOrderStatus 订单状态
type VIPOrderStatus int

const (
	OrderStatusPending  VIPOrderStatus = iota // 待支付
	OrderStatusPaid                           // 已支付
	OrderStatusCanceled                       // 已取消
	OrderStatusRefunded                       // 已退款
)

// VIPOrder 会员订单
type VIPOrder struct {
	OrderID     string         `json:"order_id"`
	UserID      int64          `json:"user_id"`
	Level       VIPLevel       `json:"level"`
	Amount      float64        `json:"amount"`
	Status      VIPOrderStatus `json:"status"`
	PaymentTime *time.Time     `json:"payment_time"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// VIPService 会员服务接口
type VIPService interface {
	// 会员信息管理
	GetVIPInfo(userID int64) (*VIPInfo, error)
	UpdateVIPStatus(userID int64, status VIPStatus) error
	UpdateAutoRenew(userID int64, autoRenew bool) error

	// 额度管理
	GetQuota(userID int64) (used int, limit int, err error)
	ConsumeQuota(userID int64, amount int) error

	// 订单管理
	CreateOrder(userID int64, level VIPLevel) (*VIPOrder, error)
	GetOrder(orderID string) (*VIPOrder, error)
	UpdateOrderStatus(orderID string, status VIPOrderStatus) error

	// 权限检查
	CanUseAI(userID int64) bool
	GetMaxRoles(userID int64) int
	GetMaxContexts(userID int64) int
	GetAvailableModels(userID int64) []string
}

// VIPConfig 会员配置
type VIPConfig struct {
	BasicVIP struct {
		MaxRoles    int
		MaxContexts int
		Models      []string
		QuotaLimit  int
	}
	ProVIP struct {
		MaxRoles    int
		MaxContexts int
		Models      []string
		QuotaLimit  int
	}
}

// DefaultVIPConfig 默认会员配置
var DefaultVIPConfig = VIPConfig{
	BasicVIP: struct {
		MaxRoles    int
		MaxContexts int
		Models      []string
		QuotaLimit  int
	}{
		MaxRoles:    2,
		MaxContexts: 5,
		Models:      []string{"gpt-3.5-turbo"},
		QuotaLimit:  1000,
	},
	ProVIP: struct {
		MaxRoles    int
		MaxContexts int
		Models      []string
		QuotaLimit  int
	}{
		MaxRoles:    5,
		MaxContexts: 100,
		Models:      []string{"gpt-3.5-turbo", "gpt-4"},
		QuotaLimit:  5000,
	},
}
