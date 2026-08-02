package mysql

// StoryboardConversationMessageDB stores creator chat turns for AI storyboard creation/editing.
type StoryboardConversationMessageDB struct {
	ID              string `gorm:"primaryKey;size:36"`
	StoryboardID    string `gorm:"size:36;not null;index:idx_scm_storyboard_seq;uniqueIndex:idx_scm_storyboard_client"`
	UserID          string `gorm:"size:36;not null;index:idx_scm_user"`
	Role            string `gorm:"size:20;not null"`
	MessageType     string `gorm:"column:message_type;size:40;not null"`
	Text            string `gorm:"type:text;not null"`
	TaskID          string `gorm:"size:36;index"`
	ClientMessageID string `gorm:"size:64;uniqueIndex:idx_scm_storyboard_client"`
	Sequence        int    `gorm:"type:int;not null;index:idx_scm_storyboard_seq"`
	CreatedAt       int64  `gorm:"type:bigint;autoCreateTime;index"`
}

func (StoryboardConversationMessageDB) TableName() string {
	return "storyboard_conversation_messages"
}
