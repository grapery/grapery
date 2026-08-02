package domain

// ChatSessionType distinguishes character roleplay vs user DM.
const (
	ChatSessionTypeCharacter = "character"
	ChatSessionTypeDirect    = "direct"
)

// ChatSession is a conversation between the owner and a character or peer user.
type ChatSession struct {
	ID               string `json:"id"`
	OwnerUserID      string `json:"ownerUserId"`
	SessionType      string `json:"sessionType"` // character | direct
	CharacterID      string `json:"characterId,omitempty"`
	PeerUserID       string `json:"peerUserId,omitempty"`
	Title            string `json:"title,omitempty"`
	Avatar           string `json:"avatar,omitempty"`
	LastMessage      string `json:"lastMessage,omitempty"`
	LastMessageAt    int64  `json:"lastMessageAt,omitempty"`
	UnreadCount      int    `json:"unreadCount"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
	// Convenience aliases for web UI ChatSession shape
	CharacterName    string `json:"characterName,omitempty"`
	CharacterAvatar  string `json:"characterAvatar,omitempty"`
	LastMessageTime  int64  `json:"lastMessageTime,omitempty"`
}

// ChatMessage is one turn in a chat session.
type ChatMessage struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	SenderID  string `json:"senderId,omitempty"`
	Role      string `json:"role"` // user | assistant | peer
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status,omitempty"`
}
