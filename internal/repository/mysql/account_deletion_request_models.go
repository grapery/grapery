package mysql

// AccountDeletionRequest persisted row for phased account deletion.
type AccountDeletionRequest struct {
	ID                  string `gorm:"primaryKey;size:36"`
	UserID              string `gorm:"column:user_id;size:36;not null;index"` // Multiple historical rows allowed; active row is pending/processing only
	Status              string `gorm:"size:20;not null;default:'pending';index"`   // pending, processing, completed, cancelled
	ScheduledDeletionAt int64  `gorm:"column:scheduled_deletion_at;type:bigint;not null;index"`
	Reason              string `gorm:"size:500"`
	Feedback            string `gorm:"type:text"`
	RequestedAt         int64  `gorm:"column:requested_at;type:bigint;not null"`
	ProcessedAt         *int64 `gorm:"column:processed_at"`
	CancelledAt         *int64 `gorm:"column:cancelled_at"`
	CancelledReason     string `gorm:"column:cancelled_reason;size:500"`
	CreatedAt           int64  `gorm:"type:bigint;autoCreateTime"`
	UpdatedAt           int64  `gorm:"type:bigint;autoUpdateTime"`

	User User `gorm:"foreignKey:UserID"`
}

func (AccountDeletionRequest) TableName() string {
	return "account_deletion_requests"
}
