package domain

// ChatThread summarizes a conversation channel
type ChatThread struct {
	ID                   string  `json:"id"`
	UserID               string  `json:"-"`
	CharacterID          string  `json:"characterId"`
	CharacterName        string  `json:"characterName"`
	CharacterAvatar      string  `json:"characterAvatar"`
	StoryTitle           string  `json:"storyTitle"`
	LastMessage          string  `json:"lastMessage"`
	LastMessageTime      int64   `json:"lastMessageTime"`
	UnreadCount          int     `json:"unreadCount"`
	MessageCount         int     `json:"messageCount"`
	InteractionFrequency float64 `json:"interactionFrequency"`
	CreatedAt            int64   `json:"createdAt"`

	// Relations
	User      *User      `json:"user,omitempty"`
	Character *Character `json:"character,omitempty"`
}

// ChatMessage contains rich chat content
type ChatMessage struct {
	ID           string `json:"id"`
	ThreadID     string `json:"threadId"`
	SenderID     string `json:"senderId"`
	SenderName   string `json:"senderName"`
	SenderAvatar string `json:"senderAvatar"`
	Content      string `json:"content"`
	Image        string `json:"image,omitempty"`
	Timestamp    int64  `json:"timestamp"`
	IsUser       bool   `json:"isUser"`

	// Relations
	Thread *ChatThread `json:"thread,omitempty"`
}
