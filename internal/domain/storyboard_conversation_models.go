package domain

const (
	StoryboardConversationRoleUser      = "user"
	StoryboardConversationRoleAssistant = "assistant"
	StoryboardConversationRoleStatus    = "status"

	StoryboardConversationTypeUserInput      = "user_input"
	StoryboardConversationTypeAnalyzeReply   = "analyze_reply"
	StoryboardConversationTypeOutline        = "outline"
	StoryboardConversationTypeScenePlan      = "scene_plan"
	StoryboardConversationTypeCharacterBrief = "character_brief"
	StoryboardConversationTypeStatus         = "status"
	StoryboardConversationTypeError          = "error"
	StoryboardConversationTypeSystem         = "system"
)

// StoryboardConversationMessage stores one user-facing turn in storyboard creation chat.
type StoryboardConversationMessage struct {
	ID              string `json:"id"`
	StoryboardID    string `json:"storyboardId"`
	UserID          string `json:"userId"`
	Role            string `json:"role"`
	MessageType     string `json:"type"`
	Text            string `json:"text"`
	TaskID          string `json:"taskId,omitempty"`
	ClientMessageID string `json:"clientMessageId,omitempty"`
	Sequence        int    `json:"sequence,omitempty"`
	CreatedAt       int64  `json:"createdAt,omitempty"`
}
