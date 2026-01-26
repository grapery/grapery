package domain

// WritersRoomMessageType enum constants
const (
	WritersRoomMsgTypeText    = "text"
	WritersRoomMsgTypeImage   = "image"
	WritersRoomMsgTypeMixed   = "mixed"
	WritersRoomMsgTypeSystem  = "system"
)

// WritersRoomParticipantRole enum constants
const (
	WritersRoomRoleOwner   = "owner"
	WritersRoomRoleAdmin   = "admin"
	WritersRoomRoleMember  = "member"
)

// WritersRoom represents a collaborative chat room for story participants
type WritersRoom struct {
	ID               string `json:"id"`
	StoryID          string `json:"storyId"`
	Title            string `json:"title"`
	LastMessage      string `json:"lastMessage"`
	LastMessageTime  int64  `json:"lastMessageTime"`
	MessageCount     int    `json:"messageCount"`
	ParticipantCount int    `json:"participantCount"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`

	// Relations
	Story        *Story                   `json:"story,omitempty"`
	Participants []*WritersRoomParticipant `json:"participants,omitempty"`
}

// WritersRoomParticipant represents a user who can access the writers room
type WritersRoomParticipant struct {
	ID        string `json:"id"`
	RoomID    string `json:"roomId"`
	UserID    string `json:"userId"`
	Role      string `json:"role"` // owner, admin, member
	JoinedAt  int64  `json:"joinedAt"`
	LastReadAt int64  `json:"lastReadAt"`

	// Flattened fields for client display (populated from User relation)
	Name   string `json:"name,omitempty"`
	Avatar string `json:"avatar,omitempty"`

	// Relations
	User *User         `json:"user,omitempty"`
	Room *WritersRoom `json:"room,omitempty"`
}

// WritersRoomMessage represents a message in the writers room
type WritersRoomMessage struct {
	ID               string `json:"id"`
	RoomID           string `json:"roomId"`
	SenderID         string `json:"senderId"`
	Content          string `json:"content"`
	MessageType      string `json:"messageType"` // text, image, mixed, system
	Attachments      []string `json:"attachments,omitempty"`
	Mentions         []string `json:"mentions,omitempty"`
	ReplyToMessageID *string  `json:"replyToMessageId,omitempty"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`

	// Flattened fields for client display (populated from User relation)
	SenderName   string `json:"senderName,omitempty"`
	SenderAvatar string `json:"senderAvatar,omitempty"`

	// Relations
	Sender         *WritersRoomParticipant `json:"sender,omitempty"`
	Room           *WritersRoom         `json:"room,omitempty"`
	ReplyToMessage *WritersRoomMessage  `json:"replyToMessage,omitempty"`
	Reactions      []*MessageReaction    `json:"reactions,omitempty"`
	ReadReceipts   []*MessageReadReceipt `json:"readReceipts,omitempty"`
}

// WritersRoomMessageReaction represents a user reaction to a message
type WritersRoomMessageReaction struct {
	ID           string `json:"id"`
	MessageID    string `json:"messageId"`
	UserID       string `json:"userId"`
	ReactionType string `json:"reactionType"` // like, emoji
	EmojiCode    string `json:"emojiCode,omitempty"`
	CreatedAt    int64  `json:"createdAt"`

	// Flattened fields for client display
	UserName string `json:"userName,omitempty"`

	// Relations
	User    *User                `json:"user,omitempty"`
	Message *WritersRoomMessage  `json:"message,omitempty"`
}

// MessageReadReceipt represents when a user has read a message
type MessageReadReceipt struct {
	ID        string `json:"id"`
	MessageID string `json:"messageId"`
	UserID    string `json:"userId"`
	ReadAt    int64  `json:"readAt"`

	// Flattened fields for client display
	UserName string `json:"userName,omitempty"`

	// Relations
	User    *User                `json:"user,omitempty"`
	Message *WritersRoomMessage  `json:"message,omitempty"`
}

// ========== Request Types ===========

// WritersRoomSendMessageRequest request to send a message in writers room
type WritersRoomSendMessageRequest struct {
	RoomID          string   `json:"roomId"`
	Content         string   `json:"content"`
	MessageType     string   `json:"messageType"` // text, image, mixed
	Attachments     []string `json:"attachments,omitempty"`
	Mentions        []string `json:"mentions,omitempty"`
	ReplyToMessageID string   `json:"replyToMessageId,omitempty"`
}

// CreateWritersRoomRequest request to create a writers room
type CreateWritersRoomRequest struct {
	StoryID string `json:"storyId"`
	Title   string `json:"title"`
}

// AddReactionRequest request to add reaction to a message
type AddReactionRequest struct {
	MessageID    string `json:"messageId"`
	ReactionType string `json:"reactionType"`
	EmojiCode    string `json:"emojiCode,omitempty"`
}

// ========== Response Types ===========

// WritersRoomMessageResponse enhanced message response with sender info
type WritersRoomMessageResponse struct {
	ID             string                   `json:"id"`
	RoomID         string                   `json:"roomId"`
	Sender         MessageSenderInfo         `json:"sender"`
	Content        string                   `json:"content"`
	MessageType    string                   `json:"messageType"`
	Attachments    []string                 `json:"attachments,omitempty"`
	Mentions       []MentionInfo            `json:"mentions,omitempty"`
	ReplyToMessageID *string                `json:"replyToMessageId,omitempty"`
	ReplyToMessage *WritersRoomMessage      `json:"replyToMessage,omitempty"`
	Reactions      []ReactionSummary        `json:"reactions,omitempty"`
	ReadReceipts   []ReadReceiptInfo       `json:"readReceipts,omitempty"`
	CreatedAt      int64                   `json:"createdAt"`
	IsMine         bool                    `json:"isMine"`
}

// MessageSenderInfo message sender information
type MessageSenderInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

// MentionInfo mention information with user details
type MentionInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReactionSummary summary of reactions for a message
type ReactionSummary struct {
	ReactionType string   `json:"reactionType"`
	EmojiCode    string   `json:"emojiCode,omitempty"`
	Count        int      `json:"count"`
	Users        []string `json:"users,omitempty"` // user IDs who reacted
}

// ReadReceiptInfo read receipt information
type ReadReceiptInfo struct {
	UserID string `json:"userId"`
	UserName string `json:"userName"`
	ReadAt  int64  `json:"readAt"`
}

// MessagesListResponse response for listing messages
type MessagesListResponse struct {
	Messages    []*WritersRoomMessageResponse `json:"messages"`
	Count       int                         `json:"count"`
	HasMore     bool                        `json:"hasMore"`
	UnreadCount int                         `json:"unreadCount"`
}
