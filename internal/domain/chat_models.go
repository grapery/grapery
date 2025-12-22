package domain

// ChatThread summarizes a conversation channel
type ChatThread struct {
	ID                   string   `json:"id"`
	UserID               string   `json:"-"`
	CharacterID          string   `json:"characterId"`
	CharacterName        string   `json:"characterName"`
	CharacterAvatar      string   `json:"characterAvatar"`
	StoryTitle           string   `json:"storyTitle"`
	LastMessage          string   `json:"lastMessage"`
	LastMessageTime      int64    `json:"lastMessageTime"`
	UnreadCount          int      `json:"unreadCount"`
	MessageCount         int      `json:"messageCount"`
	InteractionFrequency float64  `json:"interactionFrequency"`
	CreatedAt            int64    `json:"createdAt"`
	UpdatedAt            int64    `json:"updatedAt"`
	SelectedStoryboardID string   `json:"selectedStoryboardId,omitempty"` // Current selected storyboard leaf node
	ContextStoryboardIDs []string `json:"contextStoryboardIds,omitempty"` // Recent 5 storyboards the character participated in
	TotalTokensUsed      int64    `json:"totalTokensUsed"`                // Total tokens consumed in this thread
	IsArchived           bool     `json:"isArchived"`                     // Whether the thread is archived
	Summary              string   `json:"summary,omitempty"`              // AI-generated summary of the conversation

	// Relations
	User      *User      `json:"user,omitempty"`
	Character *Character `json:"character,omitempty"`
}

// ChatMessage contains rich chat content
type ChatMessage struct {
	ID           string             `json:"id"`
	ThreadID     string             `json:"threadId"`
	SenderID     string             `json:"senderId"`
	SenderName   string             `json:"senderName"`
	SenderAvatar string             `json:"senderAvatar"`
	Content      string             `json:"content"`
	Image        string             `json:"image,omitempty"`
	Timestamp    int64              `json:"timestamp"`
	IsUser       bool               `json:"isUser"`
	IsArchived   bool               `json:"isArchived"`           // Whether the message is archived
	Reactions    []*MessageReaction `json:"reactions,omitempty"`  // User reactions to this message
	TokenUsage   *TokenUsage        `json:"tokenUsage,omitempty"` // Token usage statistics

	// Relations
	Thread *ChatThread `json:"thread,omitempty"`
}

// MessageReaction represents a user reaction to a message
type MessageReaction struct {
	ID           string `json:"id"`
	MessageID    string `json:"messageId"`
	UserID       string `json:"userId"`
	ReactionType string `json:"reactionType"`        // like, dislike, emoji
	EmojiCode    string `json:"emojiCode,omitempty"` // Emoji code for emoji reactions
	CreatedAt    int64  `json:"createdAt"`
}

// TokenUsage tracks token consumption for a message
type TokenUsage struct {
	ID           string `json:"id"`
	MessageID    string `json:"messageId"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	TotalTokens  int    `json:"totalTokens"`
	Model        string `json:"model,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
}

// StoryboardBranch represents a storyboard branch selection for a chat thread
type StoryboardBranch struct {
	ID           string `json:"id"`
	ThreadID     string `json:"threadId"`
	StoryboardID string `json:"storyboardId"` // Leaf node storyboard ID
	CharacterID  string `json:"characterId"`
	SelectedAt   int64  `json:"selectedAt"`
	CreatedAt    int64  `json:"createdAt"`
}
