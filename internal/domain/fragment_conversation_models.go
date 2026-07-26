package domain

const (
	FragmentConversationRoleUser         = "user"
	FragmentConversationRoleAssistant    = "assistant"
	FragmentConversationRoleStatus       = "status"
	FragmentConversationTypeUserInput    = "user_input"
	FragmentConversationTypeAnalyzeReply = "analyze_reply"
	FragmentConversationTypeStory        = "story"
	FragmentConversationTypeImagePlan    = "image_plan"
	FragmentConversationTypeStatus       = "status"
	FragmentConversationTypeError        = "error"
	FragmentConversationTypeSystem       = "system"
)

// FragmentConversationMessage stores one user-facing turn in fragment creation chat.
type FragmentConversationMessage struct {
	ID              string `json:"id"`
	FragmentID      string `json:"fragmentId"`
	UserID          string `json:"userId"`
	Role            string `json:"role"`
	MessageType     string `json:"type"`
	Text            string `json:"text"`
	TaskID          string `json:"taskId,omitempty"`
	ClientMessageID string `json:"clientMessageId,omitempty"`
	Sequence        int    `json:"sequence,omitempty"`
	CreatedAt       int64  `json:"createdAt,omitempty"`
}
