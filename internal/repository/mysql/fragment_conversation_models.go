package mysql

// FragmentConversationMessageDB stores creator chat turns for AI fragment creation/editing.
type FragmentConversationMessageDB struct {
	ID              string `gorm:"primaryKey;size:36"`
	FragmentID      string `gorm:"size:36;not null;index:idx_fcm_fragment_seq;uniqueIndex:idx_fcm_fragment_client"`
	UserID          string `gorm:"size:36;not null;index:idx_fcm_user"`
	Role            string `gorm:"size:20;not null"`
	MessageType     string `gorm:"column:message_type;size:40;not null"`
	Text            string `gorm:"type:text;not null"`
	TaskID          string `gorm:"size:36;index"`
	ClientMessageID string `gorm:"size:64;uniqueIndex:idx_fcm_fragment_client"`
	Sequence        int    `gorm:"type:int;not null;index:idx_fcm_fragment_seq"`
	CreatedAt       int64  `gorm:"type:bigint;autoCreateTime;index"`
}

func (FragmentConversationMessageDB) TableName() string {
	return "fragment_conversation_messages"
}
