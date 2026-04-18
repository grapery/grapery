package domain

// InvitationCode 邀请码模型
type InvitationCode struct {
	ID          string `json:"id"`
	Code        string `json:"code"`        // 邀请码（唯一）
	CreatedBy   string `json:"createdBy"`   // 创建者用户ID
	UsedBy      string `json:"usedBy"`      // 使用者用户ID（如果已使用）
	UsedAt      int64  `json:"usedAt"`      // 使用时间
	IsActive    bool   `json:"isActive"`    // 是否启用
	MaxUses     int    `json:"maxUses"`     // 最大使用次数（0表示无限制）
	CurrentUses int    `json:"currentUses"` // 当前使用次数
	ExpiresAt   int64  `json:"expiresAt"`   // 过期时间（0表示永不过期）
	Description string `json:"description"` // 描述信息
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`

	// Relations
	Creator *User `json:"creator,omitempty"`
	User    *User `json:"user,omitempty"` // 使用者信息
}

// MARK: - Referral System (StoryCreationAppUI Design)

// UserReferral 用户邀请记录
type UserReferral struct {
	ID           string `json:"id"`
	ReferrerID   string `json:"referrerId"`   // 邀请人用户ID
	RefereeID    string `json:"refereeId"`    // 被邀请人用户ID
	ReferralCode string `json:"referralCode"` // 使用的邀请码
	PointsEarned int    `json:"pointsEarned"` // 获得的积分
	Status       string `json:"status"`       // pending, completed, rewarded
	CreatedAt    int64  `json:"createdAt"`
	RewardedAt   int64  `json:"rewardedAt,omitempty"`

	// Relations
	Referrer *User `json:"referrer,omitempty"`
	Referee  *User `json:"referee,omitempty"`
}

// ReferralStatus 邀请状态
type ReferralStatus string

const (
	ReferralStatusPending   ReferralStatus = "pending"
	ReferralStatusCompleted ReferralStatus = "completed"
	ReferralStatusRewarded  ReferralStatus = "rewarded"
)

// ReferralStats 用户邀请统计
type ReferralStats struct {
	TotalReferrals  int `json:"totalReferrals"`  // 总邀请人数
	PointsEarned    int `json:"pointsEarned"`    // 通过邀请获得的积分
	PendingReferrals int `json:"pendingReferrals"` // 待完成邀请数
}

// CreateReferralRequest 创建邀请请求
type CreateReferralRequest struct {
	ReferralCode string `json:"referralCode" binding:"required"` // 邀请码
}

// ReferralResponse 邀请响应
type ReferralResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	PointsEarned int    `json:"pointsEarned,omitempty"`
}

// InviteShareContent 邀请分享内容
type InviteShareContent struct {
	Title       string `json:"title"`       // 分享标题
	Description string `json:"description"` // 分享描述
	Link        string `json:"link"`        // 分享链接
	ReferralCode string `json:"referralCode"` // 邀请码
}

