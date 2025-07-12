package models

import (
	"context"
	"fmt"
	"time"
)

// LLMMsgFeedback 消息反馈
// 声明类型！
type LLMMsgFeedback struct {
	ID        int64     `gorm:"primaryKey;column:id" json:"id,omitempty"`
	MsgID     string    `gorm:"column:msg_id;index" json:"msg_id,omitempty"`
	UserID    int64     `gorm:"column:user_id;index" json:"user_id,omitempty"`
	Type      int       `gorm:"column:type;size:16" json:"type,omitempty"` // like/dislike/complaint
	Content   string    `gorm:"column:content;type:text" json:"content,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at,omitempty"`
	Deleted   bool      `gorm:"column:deleted" json:"deleted,omitempty"`
}

func (f LLMMsgFeedback) TableName() string {
	return "llmmsg_feedbacks"
}

// Create 创建反馈
func (f *LLMMsgFeedback) Create(ctx context.Context) error {
	err := DataBase().WithContext(ctx).Model(f).Create(f).Error
	if err != nil {
		return fmt.Errorf("create feedback failed: %w", err)
	}
	return nil
}

// ListByMsgId 获取消息的所有反馈
func ListFeedbacksByMsgId(ctx context.Context, msgId int64) ([]*LLMMsgFeedback, error) {
	var fbs []*LLMMsgFeedback
	err := DataBase().WithContext(ctx).Model(&LLMMsgFeedback{}).Where("msg_id = ? AND deleted = 0", msgId).Order("created_at asc").Find(&fbs).Error
	if err != nil {
		return nil, fmt.Errorf("list feedbacks by msg failed: %w", err)
	}
	return fbs, nil
}
