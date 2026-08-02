package domain

// ShareEventType distinguishes share funnel stages.
type ShareEventType string

const (
	ShareEventIssue ShareEventType = "issue"
	ShareEventOpen  ShareEventType = "open"
)

// ShareEvent records a share issuance or open for admin analytics.
type ShareEvent struct {
	ID        string         `json:"id" gorm:"primaryKey;size:36"`
	EventType ShareEventType `json:"eventType" gorm:"size:16;not null;index:idx_share_events_type_created"`
	Kind      string         `json:"kind" gorm:"size:32;not null;index:idx_share_events_kind_created"`
	ContentID string         `json:"contentId" gorm:"size:64;not null;index:idx_share_events_content"`
	UserID    string         `json:"userId,omitempty" gorm:"size:36;index:idx_share_events_user"`
	Platform  string         `json:"platform,omitempty" gorm:"size:32"`
	Source    string         `json:"source,omitempty" gorm:"size:32"`
	CreatedAt int64          `json:"createdAt" gorm:"type:bigint;autoCreateTime;index:idx_share_events_type_created;index:idx_share_events_kind_created"`
}

func (ShareEvent) TableName() string { return "share_events" }
