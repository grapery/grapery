package domain

// UserFeedback 用户提交的反馈（持久化）
type UserFeedback struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Category    string `json:"category"`
	Content     string `json:"content"`
	ContactInfo string `json:"contactInfo,omitempty"`
	Status      string `json:"status"`
	Response    string `json:"response,omitempty"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt,omitempty"`
}
