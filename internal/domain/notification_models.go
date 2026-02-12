package domain

import "github.com/grapestree/fgrapery/grapery/internal/common"

// Notification 通知
type Notification struct {
	// Base model fields
	common.BaseModel

	UserID      string `json:"userId"`
	Type        string `json:"type"` // like, comment, follow, mention, system
	Title       string `json:"title"`
	Content     string `json:"content"`
	Link        string `json:"link,omitempty"`
	Read        bool   `json:"read"`
	ActorID     string `json:"actorId,omitempty"` // 触发通知的用户ID
	ActorName   string `json:"actorName,omitempty"`
	ActorAvatar string `json:"actorAvatar,omitempty"`

	// Relations
	User  *User `json:"user,omitempty"`
	Actor *User `json:"actor,omitempty"`
}
