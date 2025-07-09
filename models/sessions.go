package models

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Session 聊天会话
// 声明类型！
type UserSession struct {
	ID             int64     `gorm:"primaryKey;column:id" json:"id,omitempty"`                     // 数据库主键
	UserID         int64     `gorm:"column:user_id;index" json:"user_id,omitempty"`                // 用户ID
	SessionId      string    `gorm:"column:session_id;size:64;index" json:"session_id"`            // 会话唯一ID
	ConversationId string    `gorm:"column:conversation_id;size:64;index" json:"conversation_id"`  // 会话ID
	RoleId         string    `gorm:"column:role_id;size:64" json:"role_id"`                        // 角色ID
	BotId          string    `gorm:"column:bot_id;size:64" json:"bot_id"`                          // 机器人ID
	Summary        string    `gorm:"column:summary;size:1024" json:"summary"`                      // 会话摘要
	MsgCount       int       `gorm:"column:msg_count" json:"msg_count"`                            // 消息数量
	StartTime      int64     `gorm:"column:start_time" json:"start_time"`                          // 会话开始时间（时间戳）
	EndTime        int64     `gorm:"column:end_time" json:"end_time"`                              // 会话结束时间（时间戳）
	Name           string    `gorm:"column:name;size:128" json:"name,omitempty"`                   // 会话名称
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at,omitempty"` // 创建时间
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at,omitempty"` // 更新时间
	Deleted        bool      `gorm:"column:deleted" json:"deleted,omitempty"`                      // 是否已删除
	LastClearAt    int64     `gorm:"column:last_clear_at" json:"last_clear_at,omitempty"`          // 上次清空时间（时间戳）
}

func (s UserSession) TableName() string {
	return "user_sessions"
}

// Create 创建会话
func (s *UserSession) Create(ctx context.Context) error {
	err := DataBase().WithContext(ctx).Model(s).Create(s).Error
	if err != nil {
		return fmt.Errorf("create session failed: %w", err)
	}
	return nil
}

// GetById 根据ID获取会话
func (s *UserSession) GetById(ctx context.Context, id int64) error {
	err := DataBase().WithContext(ctx).Model(s).Where("id = ? AND deleted = 0", id).First(s).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get user_session [%d] failed: %w", id, err)
	}
	return nil
}

// UpdateById 根据ID更新会话（只更新非零字段）
func (s *UserSession) UpdateById(ctx context.Context, id int64, updates map[string]interface{}) error {
	err := DataBase().WithContext(ctx).Model(s).Where("id = ? AND deleted = 0", id).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("update user_session [%d] failed: %w", id, err)
	}
	return nil
}

// DeleteById 逻辑删除会话
func (s *UserSession) DeleteById(ctx context.Context, id int64) error {
	err := DataBase().WithContext(ctx).Model(s).Where("id = ? AND deleted = 0", id).Update("deleted", true).Error
	if err != nil {
		return fmt.Errorf("delete user_session [%d] failed: %w", id, err)
	}
	return nil
}

// ListByUserId 获取用户所有会话
func ListUserSessionsByUserId(ctx context.Context, userId int64) ([]*UserSession, error) {
	var sessions []*UserSession
	err := DataBase().WithContext(ctx).Model(&UserSession{}).Where("user_id = ? AND deleted = 0", userId).Order("updated_at desc").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("list user_sessions by user failed: %w", err)
	}
	return sessions, nil
}

// ListAllUserSessions 获取所有会话（可选）
func ListAllUserSessions(ctx context.Context) ([]*UserSession, error) {
	var sessions []*UserSession
	err := DataBase().WithContext(ctx).Model(&UserSession{}).Where("deleted = 0").Order("updated_at desc").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("list all user_sessions failed: %w", err)
	}
	return sessions, nil
}

// GetByRoleId 根据RoleId获取会话
func (s *UserSession) GetByRoleId(ctx context.Context, roleId string) error {
	err := DataBase().WithContext(ctx).Model(s).Where("role_id = ? AND deleted = 0", roleId).Order("updated_at desc").First(s).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get user_session by role_id [%s] failed: %w", roleId, err)
	}
	return nil
}

// ListUserSessionsByRoleId 根据RoleId获取所有会话，按更新时间倒序
func ListUserSessionsByRoleId(ctx context.Context, roleId string) ([]*UserSession, error) {
	var sessions []*UserSession
	err := DataBase().WithContext(ctx).Model(&UserSession{}).Where("role_id = ? AND deleted = 0", roleId).Order("updated_at desc").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("list user_sessions by role_id failed: %w", err)
	}
	return sessions, nil
}

// UpdateByRoleId 根据RoleId更新会话（只更新非零字段）
func (s *UserSession) UpdateByRoleId(ctx context.Context, roleId string, updates map[string]interface{}) error {
	err := DataBase().WithContext(ctx).Model(s).Where("role_id = ? AND deleted = 0", roleId).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("update user_session by role_id [%s] failed: %w", roleId, err)
	}
	return nil
}

// DeleteByRoleId 逻辑删除RoleId对应的会话
func (s *UserSession) DeleteByRoleId(ctx context.Context, roleId string) error {
	err := DataBase().WithContext(ctx).Model(s).Where("role_id = ? AND deleted = 0", roleId).Update("deleted", true).Error
	if err != nil {
		return fmt.Errorf("delete user_session by role_id [%s] failed: %w", roleId, err)
	}
	return nil
}

// GetByBotId 根据BotId获取会话
func (s *UserSession) GetByBotId(ctx context.Context, botId string) error {
	err := DataBase().WithContext(ctx).Model(s).Where("bot_id = ? AND deleted = 0", botId).Order("updated_at desc").First(s).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get user_session by bot_id [%s] failed: %w", botId, err)
	}
	return nil
}

// ListUserSessionsByBotId 根据BotId获取所有会话，按更新时间倒序
func ListUserSessionsByBotId(ctx context.Context, botId string) ([]*UserSession, error) {
	var sessions []*UserSession
	err := DataBase().WithContext(ctx).Model(&UserSession{}).Where("bot_id = ? AND deleted = 0", botId).Order("updated_at desc").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("list user_sessions by bot_id failed: %w", err)
	}
	return sessions, nil
}

// UpdateByBotId 根据BotId更新会话（只更新非零字段）
func (s *UserSession) UpdateByBotId(ctx context.Context, botId string, updates map[string]interface{}) error {
	err := DataBase().WithContext(ctx).Model(s).Where("bot_id = ? AND deleted = 0", botId).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("update user_session by bot_id [%s] failed: %w", botId, err)
	}
	return nil
}

// DeleteByBotId 逻辑删除BotId对应的会话
func (s *UserSession) DeleteByBotId(ctx context.Context, botId string) error {
	err := DataBase().WithContext(ctx).Model(s).Where("bot_id = ? AND deleted = 0", botId).Update("deleted", true).Error
	if err != nil {
		return fmt.Errorf("delete user_session by bot_id [%s] failed: %w", botId, err)
	}
	return nil
}

// GetBySessionId 根据SessionId获取会话
func (s *UserSession) GetBySessionId(ctx context.Context, sessionId string) error {
	err := DataBase().WithContext(ctx).Model(s).Where("session_id = ? AND deleted = 0", sessionId).Order("updated_at desc").First(s).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get user_session by session_id [%s] failed: %w", sessionId, err)
	}
	return nil
}

// ListUserSessionsBySessionId 根据SessionId获取所有会话，按更新时间倒序
func ListUserSessionsBySessionId(ctx context.Context, sessionId string) ([]*UserSession, error) {
	var sessions []*UserSession
	err := DataBase().WithContext(ctx).Model(&UserSession{}).Where("session_id = ? AND deleted = 0", sessionId).Order("updated_at desc").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("list user_sessions by session_id failed: %w", err)
	}
	return sessions, nil
}

// UpdateBySessionId 根据SessionId更新会话（只更新非零字段）
func (s *UserSession) UpdateBySessionId(ctx context.Context, sessionId string, updates map[string]interface{}) error {
	err := DataBase().WithContext(ctx).Model(s).Where("session_id = ? AND deleted = 0", sessionId).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("update user_session by session_id [%s] failed: %w", sessionId, err)
	}
	return nil
}

// DeleteBySessionId 逻辑删除SessionId对应的会话
func (s *UserSession) DeleteBySessionId(ctx context.Context, sessionId string) error {
	err := DataBase().WithContext(ctx).Model(s).Where("session_id = ? AND deleted = 0", sessionId).Update("deleted", true).Error
	if err != nil {
		return fmt.Errorf("delete user_session by session_id [%s] failed: %w", sessionId, err)
	}
	return nil
}

// GetByConversationId 根据ConversationId获取会话
func (s *UserSession) GetByConversationId(ctx context.Context, conversationId string) error {
	err := DataBase().WithContext(ctx).Model(s).Where("conversation_id = ? AND deleted = 0", conversationId).Order("updated_at desc").First(s).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get user_session by conversation_id [%s] failed: %w", conversationId, err)
	}
	return nil
}

// ListUserSessionsByConversationId 根据ConversationId获取所有会话，按更新时间倒序
func ListUserSessionsByConversationId(ctx context.Context, conversationId string) ([]*UserSession, error) {
	var sessions []*UserSession
	err := DataBase().WithContext(ctx).Model(&UserSession{}).Where("conversation_id = ? AND deleted = 0", conversationId).Order("updated_at desc").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("list user_sessions by conversation_id failed: %w", err)
	}
	return sessions, nil
}

// UpdateByConversationId 根据ConversationId更新会话（只更新非零字段）
func (s *UserSession) UpdateByConversationId(ctx context.Context, conversationId string, updates map[string]interface{}) error {
	err := DataBase().WithContext(ctx).Model(s).Where("conversation_id = ? AND deleted = 0", conversationId).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("update user_session by conversation_id [%s] failed: %w", conversationId, err)
	}
	return nil
}

// DeleteByConversationId 逻辑删除ConversationId对应的会话
func (s *UserSession) DeleteByConversationId(ctx context.Context, conversationId string) error {
	err := DataBase().WithContext(ctx).Model(s).Where("conversation_id = ? AND deleted = 0", conversationId).Update("deleted", true).Error
	if err != nil {
		return fmt.Errorf("delete user_session by conversation_id [%s] failed: %w", conversationId, err)
	}
	return nil
}
