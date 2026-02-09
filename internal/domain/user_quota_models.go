package domain

import "time"

// UserQuotaInfo 用户配额信息（用于个人主页展示）
type UserQuotaInfo struct {
	UserID          string    `json:"userId"`
	TokenBalance    int       `json:"tokenBalance"`     // 当前 token 余额
	TokenQuota      int       `json:"tokenQuota"`       // token 配额总量
	TokenUsed       int       `json:"tokenUsed"`        // 已使用 token 数量
	TokenRemaining  int       `json:"tokenRemaining"`   // 剩余 token 数量
	StorageUsed     int64     `json:"storageUsed"`      // 已使用存储空间（字节）
	StorageQuota    int64     `json:"storageQuota"`     // 存储配额（字节）
	StorageRemaining int64    `json:"storageRemaining"` // 剩余存储空间（字节）
	ResetDate       *time.Time `json:"resetDate,omitempty"` // 配额重置日期
}

// UserMembershipInfo 用户会员信息（用于 VIP 页面展示）
type UserMembershipInfo struct {
	UserID          string             `json:"userId"`
	Tier            string             `json:"tier"`             // free, basic, pro, enterprise
	TierName        string             `json:"tierName"`         // 会员等级名称
	Status          string             `json:"status"`           // active, expired, cancelled
	StatusName      string             `json:"statusName"`       // 状态名称
	StartDate       int64              `json:"startDate"`        // 会员开始日期
	EndDate         *int64             `json:"endDate,omitempty"` // 会员结束日期
	AutoRenew       bool               `json:"autoRenew"`        // 是否自动续费
	DaysRemaining   int                `json:"daysRemaining"`    // 剩余天数
	CanRenew        bool               `json:"canRenew"`         // 是否可以续费
	QuotaInfo       *UserQuotaInfo     `json:"quotaInfo"`        // 配额信息
	Benefits        []MembershipBenefit `json:"benefits"`         // 会员权益列表
	CreatedAt       int64              `json:"createdAt"`
	UpdatedAt       int64              `json:"updatedAt"`
}

// MembershipBenefit 会员权益
type MembershipBenefit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`        // 权益名称
	Description string `json:"description"` // 权益描述
	Icon        string `json:"icon"`        // 权益图标
	Enabled     bool   `json:"enabled"`     // 是否启用
	Value       string `json:"value"`       // 权益值（如 "1000 tokens/月"）
}

// UserRechargeInfo 用户充值信息
type UserRechargeInfo struct {
	UserID              string                  `json:"userId"`
	TotalRecharged      int64                   `json:"totalRecharged"`       // 总充值金额（分为单位）
	TotalRechargedCNY   float64                 `json:"totalRechargedCNY"`    // 总充值金额（人民币）
	RechargeCount       int                     `json:"rechargeCount"`        // 充值次数
	LastRechargeAt      *int64                  `json:"lastRechargeAt,omitempty"` // 最后充值时间
	RecentOrders        []*SubscriptionOrder    `json:"recentOrders"`         // 最近订单
	PaymentMethods      []string                `json:"paymentMethods"`       // 可用的支付方式
}

// UserUsageStatistics 用户使用统计
type UserUsageStatistics struct {
	UserID              string `json:"userId"`
	Period              string `json:"period"` // today, week, month, year, all

	// AI 生成统计
	TextGeneratedCount  int    `json:"textGeneratedCount"`  // 文本生成次数
	ImageGeneratedCount int    `json:"imageGeneratedCount"` // 图片生成次数
	VideoGeneratedCount int    `json:"videoGeneratedCount"` // 视频生成次数

	// Token 使用统计
	TotalTokensUsed     int    `json:"totalTokensUsed"`     // 总使用 token 数
	TextTokensUsed      int    `json:"textTokensUsed"`      // 文本生成使用 token
	ImageTokensUsed     int    `json:"imageTokensUsed"`     // 图片生成使用 token
	VideoTokensUsed     int    `json:"videoTokensUsed"`     // 视频生成使用 token

	// 创作统计
	StoriesCreated      int    `json:"storiesCreated"`      // 创建故事数
	FragmentsCreated    int    `json:"fragmentsCreated"`    // 创建片段数
	CharactersCreated   int    `json:"charactersCreated"`   // 创建角色数
	StoryboardsCreated  int    `json:"storyboardsCreated"`  // 创建故事板数

	// 互动统计
	LikesGiven          int    `json:"likesGiven"`          // 点赞数
	CommentsMade        int    `json:"commentsMade"`        // 评论数
	SharesCount         int    `json:"sharesCount"`         // 分享数

	StartDate           int64  `json:"startDate"`           // 统计开始时间
	EndDate             int64  `json:"endDate"`             // 统计结束时间
}

// UserDashboardInfo 用户主页综合信息
type UserDashboardInfo struct {
	User            *User               `json:"user"`
	QuotaInfo       *UserQuotaInfo      `json:"quotaInfo"`
	Membership      *UserMembershipInfo `json:"membership,omitempty"`
	UsageStatistics *UserUsageStatistics `json:"usageStatistics,omitempty"`
	RecentActivities []*UserActivity    `json:"recentActivities,omitempty"`
}

// TokenUsageDetail Token 使用明细
type TokenUsageDetail struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // consume, recharge, refund, gift
	Amount      int       `json:"amount"`      // 数量
	Balance     int       `json:"balance"`     // 交易后余额
	Source      string    `json:"source"`      // 来源
	Description string    `json:"description"` // 描述
	RelatedID   string    `json:"relatedId,omitempty"` // 关联ID
	CreatedAt   int64     `json:"createdAt"`
}

// TokenUsageHistory Token 使用历史（带分页）
type TokenUsageHistory struct {
	Details     []*TokenUsageDetail `json:"details"`
	TotalCount  int64              `json:"totalCount"`
	Page        int                `json:"page"`
	PageSize    int                `json:"pageSize"`
	TotalPages  int                `json:"totalPages"`
}

// MembershipTier 会员等级配置
type MembershipTier struct {
	Tier           string               `json:"tier"`            // free, basic, pro, enterprise
	TierName       string               `json:"tierName"`        // 等级名称
	DisplayName    string               `json:"displayName"`     // 显示名称
	Description    string               `json:"description"`     // 描述
	Price          float64              `json:"price"`           // 月费
	Currency       string               `json:"currency"`        // 货币
	TokenQuota     int                  `json:"tokenQuota"`      // Token 配额
	StorageQuota   int64                `json:"storageQuota"`    // 存储配额
	MaxStories     int                  `json:"maxStories"`      // 最大故事数
	MaxCharacters  int                  `json:"maxCharacters"`   // 最大角色数
	Benefits       []MembershipBenefit  `json:"benefits"`         // 权益列表
	Features       []string             `json:"features"`        // 功能列表
	SortOrder      int                  `json:"sortOrder"`       // 排序
	IsActive       bool                 `json:"isActive"`        // 是否启用
	TrialDays      int                  `json:"trialDays"`       // 试用天数
	UpgradeFrom    []string             `json:"upgradeFrom"`     // 可从此等级升级
}
