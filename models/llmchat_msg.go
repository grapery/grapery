package models

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// LLMChatMsg 聊天消息
// 声明类型！
type LLMChatMsg struct {
	ID        int64     `gorm:"primaryKey;column:id" json:"id,omitempty"`
	SessionID int64     `gorm:"column:session_id;index" json:"session_id,omitempty"`
	UserID    int64     `gorm:"column:user_id;index" json:"user_id,omitempty"`
	Content   string    `gorm:"column:content;type:text" json:"content,omitempty"`
	MsgType   string    `gorm:"column:msg_type;size:16" json:"msg_type,omitempty"` // user/ai
	Status    string    `gorm:"column:status;size:16" json:"status,omitempty"`     // sent/failed/pending/interrupted
	ParentID  *int64    `gorm:"column:parent_id;index" json:"parent_id,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at,omitempty"`
	Deleted   bool      `gorm:"column:deleted" json:"deleted,omitempty"`
}

func (m LLMChatMsg) TableName() string {
	return "llmchat_msgs"
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
