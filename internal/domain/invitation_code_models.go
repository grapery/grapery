package domain

// InvitationCode 邀请码模型
type InvitationCode struct {
	ID          string `json:"id"`
	Code        string `json:"code"`        // 邀请码（唯一）
	CreatedBy   string `json:"createdBy"`  // 创建者用户ID
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

