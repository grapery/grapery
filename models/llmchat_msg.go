package models

import (
	"context"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LLMChatMsg 聊天消息
// 声明类型！
type LLMChatMsg struct {
	ID             int64          `gorm:"primaryKey;column:id" json:"id,omitempty"`                              // 消息主键
	SessionID      string         `gorm:"column:session_id;size:64;index" json:"session_id,omitempty"`           // 会话ID（string类型）
	MessageId      string         `gorm:"column:message_id;size:64;index" json:"message_id"`                     // 消息唯一ID
	ConversationId string         `gorm:"column:conversation_id;size:64;index" json:"conversation_id,omitempty"` // 会话ID（string类型）
	UserID         int64          `gorm:"column:user_id;index" json:"user_id,omitempty"`                         // 用户ID
	Content        string         `gorm:"column:content;type:text" json:"content,omitempty"`                     // 消息内容
	LLmContent     string         `gorm:"column:llm_content;type:text" json:"llm_content,omitempty"`             // 消息内容
	MsgType        string         `gorm:"column:msg_type;size:16" json:"msg_type,omitempty"`                     // 消息类型 user/ai
	Status         string         `gorm:"column:status;size:16" json:"status,omitempty"`                         // 消息状态 sent/failed/pending/interrupted
	Like           int            `gorm:"column:like;default:0" json:"like"`                                     // 点赞状态：0=无操作，1=喜欢，-1=点踩，默认0
	Attachments    datatypes.JSON `gorm:"column:attachments;type:json" json:"attachments,omitempty"`             // 附件数据，如{"image":"dasdasdsa.jpg"}
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at,omitempty"`          // 创建时间
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at,omitempty"`          // 更新时间
	Deleted        bool           `gorm:"column:deleted" json:"deleted,omitempty"`                               // 是否已删除
}

func (m LLMChatMsg) TableName() string {
	return "llmchat_msgs"
}

func UpdateLike(ctx context.Context, msgID int64, like int) error {
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("id = ?", msgID).Update("like", like).Error
	if err != nil {
		return fmt.Errorf("update like failed: %w", err)
	}
	return nil
}

// Create 创建消息
func (m *LLMChatMsg) Create(ctx context.Context) error {
	err := DataBase().WithContext(ctx).Model(m).Create(m).Error
	if err != nil {
		return fmt.Errorf("create llmchat_msg failed: %w", err)
	}
	return nil
}

// GetById 根据ID获取消息
func (m *LLMChatMsg) GetById(ctx context.Context, id int64) error {
	err := DataBase().WithContext(ctx).Model(m).Where("id = ? AND deleted = 0", id).First(m).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get llmchat_msg [%d] failed: %w", id, err)
	}
	return nil
}

// ListBySessionId 获取会话下所有消息
func ListMsgsBySessionId(ctx context.Context, sessionId int64) ([]*LLMChatMsg, error) {
	var msgs []*LLMChatMsg
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("session_id = ? AND deleted = 0", sessionId).Order("created_at asc").Find(&msgs).Error
	if err != nil {
		return nil, fmt.Errorf("list msgs by session failed: %w", err)
	}
	return msgs, nil
}

// ListByUserId 获取用户所有消息
func ListMsgsByUserId(ctx context.Context, userId int64) ([]*LLMChatMsg, error) {
	var msgs []*LLMChatMsg
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("user_id = ? AND deleted = 0", userId).Order("created_at asc").Find(&msgs).Error
	if err != nil {
		return nil, fmt.Errorf("list msgs by user failed: %w", err)
	}
	return msgs, nil
}

// UpdateStatus 更新消息状态
func (m *LLMChatMsg) UpdateStatus(ctx context.Context, status string) error {
	err := DataBase().WithContext(ctx).Model(m).Where("id = ?", m.ID).Update("status", status).Error
	if err != nil {
		return fmt.Errorf("update msg status failed: %w", err)
	}
	return nil
}

// GetBySessionID 根据SessionID获取一条消息（如需多条请用List）
func (m *LLMChatMsg) GetBySessionID(ctx context.Context, sessionID string) error {
	err := DataBase().WithContext(ctx).Model(m).Where("session_id = ? AND deleted = 0", sessionID).Order("created_at desc").First(m).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get llmchat_msg by session_id [%s] failed: %w", sessionID, err)
	}
	return nil
}

// ListBySessionID 根据SessionID获取所有消息，按创建时间倒序
func ListLLMChatMsgsBySessionID(ctx context.Context, sessionID string) ([]*LLMChatMsg, error) {
	var msgs []*LLMChatMsg
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("session_id = ? AND deleted = 0", sessionID).Order("created_at desc").Find(&msgs).Error
	if err != nil {
		return nil, fmt.Errorf("list llmchat_msgs by session_id failed: %w", err)
	}
	return msgs, nil
}

// UpdateBySessionID 根据SessionID批量更新消息（只更新非零字段）
func UpdateLLMChatMsgsBySessionID(ctx context.Context, sessionID string, updates map[string]interface{}) error {
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("session_id = ? AND deleted = 0", sessionID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("update llmchat_msgs by session_id [%s] failed: %w", sessionID, err)
	}
	return nil
}

// DeleteBySessionID 根据SessionID批量逻辑删除消息
func DeleteLLMChatMsgsBySessionID(ctx context.Context, sessionID string) error {
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("session_id = ? AND deleted = 0", sessionID).Update("deleted", true).Error
	if err != nil {
		return fmt.Errorf("delete llmchat_msgs by session_id [%s] failed: %w", sessionID, err)
	}
	return nil
}

// GetByUserID 根据UserID获取一条消息
func (m *LLMChatMsg) GetByUserID(ctx context.Context, userID int64) error {
	err := DataBase().WithContext(ctx).Model(m).Where("user_id = ? AND deleted = 0", userID).Order("created_at desc").First(m).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get llmchat_msg by user_id [%d] failed: %w", userID, err)
	}
	return nil
}

// ListByUserID 根据UserID获取所有消息，按创建时间倒序
func ListLLMChatMsgsByUserID(ctx context.Context, userID int64) ([]*LLMChatMsg, error) {
	var msgs []*LLMChatMsg
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("user_id = ? AND deleted = 0", userID).Order("created_at desc").Find(&msgs).Error
	if err != nil {
		return nil, fmt.Errorf("list llmchat_msgs by user_id failed: %w", err)
	}
	return msgs, nil
}

// UpdateByUserID 根据UserID批量更新消息（只更新非零字段）
func UpdateLLMChatMsgsByUserID(ctx context.Context, userID int64, updates map[string]interface{}) error {
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("user_id = ? AND deleted = 0", userID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("update llmchat_msgs by user_id [%d] failed: %w", userID, err)
	}
	return nil
}

// DeleteByUserID 根据UserID批量逻辑删除消息
func DeleteLLMChatMsgsByUserID(ctx context.Context, userID int64) error {
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("user_id = ? AND deleted = 0", userID).Update("deleted", true).Error
	if err != nil {
		return fmt.Errorf("delete llmchat_msgs by user_id [%d] failed: %w", userID, err)
	}
	return nil
}

// BatchInsertLLMChatMsgs 批量插入消息
func BatchInsertLLMChatMsgs(ctx context.Context, msgs []*LLMChatMsg) error {
	if len(msgs) == 0 {
		return nil
	}
	return DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Create(&msgs).Error
}

// ListLLMChatMsgsBySessionIDWithPage 根据SessionID分页查询消息，按创建时间倒序
// 返回：消息列表、总数、错误
func ListLLMChatMsgsBySessionIDWithPage(ctx context.Context, sessionID string, page, pageSize int) ([]*LLMChatMsg, int64, error) {
	var msgs []*LLMChatMsg
	var total int64
	db := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("session_id = ? AND deleted = 0", sessionID)
	db.Count(&total)
	err := db.Order("created_at asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&msgs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list llmchat_msgs by session_id with page failed: %w", err)
	}
	return msgs, total, nil
}

// ListLLMChatMsgsByUserIDWithPage 根据UserID分页查询消息，按创建时间倒序
// 返回：消息列表、总数、错误
func ListLLMChatMsgsByUserIDWithPage(ctx context.Context, userID int64, page, pageSize int) ([]*LLMChatMsg, int64, error) {
	var msgs []*LLMChatMsg
	var total int64
	db := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("user_id = ? AND deleted = 0", userID)
	db.Count(&total)
	err := db.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&msgs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list llmchat_msgs by user_id with page failed: %w", err)
	}
	return msgs, total, nil
}

// GetByMessageId 根据MessageId获取消息
func (m *LLMChatMsg) GetByMessageId(ctx context.Context, messageId string) error {
	err := DataBase().WithContext(ctx).Model(m).Where("message_id = ? AND deleted = 0", messageId).Order("created_at desc").First(m).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get llmchat_msg by message_id [%s] failed: %w", messageId, err)
	}
	return nil
}

// ListByMessageId 根据MessageId获取所有消息，按创建时间倒序
func ListLLMChatMsgsByMessageId(ctx context.Context, messageId string) ([]*LLMChatMsg, error) {
	var msgs []*LLMChatMsg
	err := DataBase().WithContext(ctx).Model(&LLMChatMsg{}).Where("message_id = ? AND deleted = 0", messageId).Order("created_at desc").Find(&msgs).Error
	if err != nil {
		return nil, fmt.Errorf("list llmchat_msgs by message_id failed: %w", err)
	}
	return msgs, nil
}

// UpdateByMessageId 根据MessageId更新消息（只更新非零字段）
func (m *LLMChatMsg) UpdateByMessageId(ctx context.Context, messageId string, updates map[string]interface{}) error {
	err := DataBase().WithContext(ctx).Model(m).
		Where("message_id = ? AND deleted = 0", messageId).
		Where("status = ?", "pending").
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("update llmchat_msg by message_id [%s] failed: %w", messageId, err)
	}
	return nil
}

// DeleteByMessageId 逻辑删除MessageId对应的消息
func (m *LLMChatMsg) DeleteByMessageId(ctx context.Context, messageId string) error {
	err := DataBase().WithContext(ctx).Model(m).Where("message_id = ? AND deleted = 0", messageId).Update("deleted", true).Error
	if err != nil {
		return fmt.Errorf("delete llmchat_msg by message_id [%s] failed: %w", messageId, err)
	}
	return nil
}
