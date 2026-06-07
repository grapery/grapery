package domain

// BlockedUser is a user the viewer has blocked, with display metadata for settings UI.
type BlockedUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Username    string `json:"username"`
	BlockedAt   int64  `json:"blockedAt"`
}
