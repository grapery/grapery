package models

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Session 聊天会话
// 声明类型！
type Session struct {
	ID        int64     `gorm:"primaryKey;column:id" json:"id,omitempty"`
	UserID    int64     `gorm:"column:user_id;index" json:"user_id,omitempty"`
	Name      string    `gorm:"column:name;size:128" json:"name,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at,omitempty"`
	Deleted   bool      `gorm:"column:deleted" json:"deleted,omitempty"`
}

func (s Session) TableName() string {
	return "sessions"
}

// Create 创建会话
func (s *Session) Create(ctx context.Context) error {
	err := DataBase().WithContext(ctx).Model(s).Create(s).Error
	if err != nil {
		return fmt.Errorf("create session failed: %w", err)
	}
	return nil
}

// GetById 根据ID获取会话
func (s *Session) GetById(ctx context.Context, id int64) error {
	err := DataBase().WithContext(ctx).Model(s).Where("id = ? AND deleted = 0", id).First(s).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get session [%d] failed: %w", id, err)
	}
	return nil
}

// ListByUserId 获取用户所有会话
func ListSessionsByUserId(ctx context.Context, userId int64) ([]*Session, error) {
	var sessions []*Session
	err := DataBase().WithContext(ctx).Model(&Session{}).Where("user_id = ? AND deleted = 0", userId).Order("updated_at desc").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("list sessions by user failed: %w", err)
	}
	return sessions, nil
}
