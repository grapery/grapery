package models

import (
	"context"

	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

type ChatContext struct {
	IDBase
	UserID      int64  `json:"user_id,omitempty"`
	RoleID      int64  `json:"role_id,omitempty"`
	Title       string `json:"title,omitempty"`
	Content     string `json:"content,omitempty"`
	Status      int64  `json:"status,omitempty"` // 1: open, 2: close, 3: delete
	UseAgent    int64  `json:"use_agent,omitempty"`
	AgentPrompt string `json:"agent_prompt,omitempty"`
}

func (c ChatContext) TableName() string {
	return "chat_context"
}

func CreateChatContext(ctx context.Context, chatContext *ChatContext) error {
	return DataBase().Create(chatContext).WithContext(ctx).Error
}

func GetChatContextByID(ctx context.Context, id int64) (*ChatContext, error) {
	var chatContext ChatContext
	err := DataBase().Model(&ChatContext{}).
		Where("id = ?", id).
		Where("status = ?", 1).
		WithContext(ctx).
		First(&chatContext).Error
	return &chatContext, err
}

func GetChatContextByUserID(ctx context.Context, userID int64, page, size int) ([]*ChatContext, int, error) {
	var chatContexts []*ChatContext
	err := DataBase().Model(&ChatContext{}).WithContext(ctx).
		Where("user_id = ?", userID).
		Where("status = ?", 1).
		Order("update_at DESC").
		Offset((page) * size).
		Limit(size).
		Find(&chatContexts).Error
	if err != nil {
		log.Log().Error("get chat context by user id failed", zap.Error(err))
		return nil, 0, err
	}
	var total int64
	err = DataBase().Model(&ChatContext{}).
		Where("user_id = ?", userID).
		Where("status = ?", 1).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return chatContexts, int(total), nil
}

func GetChatContextByUserIDAndRoleID(ctx context.Context, userID int64, roleID int64) (*ChatContext, error) {
	var chatContext = new(ChatContext)
	err := DataBase().Where("user_id = ?", userID).
		Where("role_id = ?", roleID).
		Where("status = ?", 1).
		WithContext(ctx).
		First(&chatContext).Error
	return chatContext, err
}

func GetChatContextByRoleID(ctx context.Context, roleID int64) ([]*ChatContext, error) {
	var chatContexts []*ChatContext
	err := DataBase().
		Where("role_id = ?", roleID).
		Where("status = ?", 1).
		WithContext(ctx).
		Find(&chatContexts).Error
	return chatContexts, err
}

func UpdateChatContext(ctx context.Context, id int64, updates map[string]interface{}) error {
	return DataBase().Model(&ChatContext{}).
		Where("id = ?", id).
		WithContext(ctx).
		Updates(updates).Error
}

func DeleteChatContext(ctx context.Context, id int64) error {
	return DataBase().Model(&ChatContext{}).Update("status", -1).
		Where("id = ?", id).
		WithContext(ctx).Error
}

type ChatMessage struct {
	IDBase
	ChatContextID int64  `json:"chat_context_id,omitempty"`
	UserID        int64  `json:"user_id,omitempty"`
	RoleID        int64  `json:"role_id,omitempty"`
	Sender        int64  `json:"sender,omitempty"`
	MessageType   int64  `json:"message_type,omitempty"`
	Content       string `json:"content,omitempty"`
	Status        int64  `json:"status,omitempty"`
	NeedRender    int64  `json:"is_need_render,omitempty"`
	Prompt        string `json:"prompt,omitempty"`
	AfterRender   string `json:"after_render,omitempty"`
	UUID          string `json:"uuid,omitempty"`
	SendTime      int64  `json:"send_time,omitempty"`
	ReceiveTime   int64  `json:"receive_time,omitempty"`
	MessageID     string `json:"message_id,omitempty"`
}

func (c ChatMessage) TableName() string {
	return "chat_message"
}

func CreateChatMessage(ctx context.Context, chatMessage *ChatMessage) error {
	return DataBase().Create(chatMessage).WithContext(ctx).Error
}

func GetChatMessageByChatContextID(ctx context.Context, chatContextID int64, page, size int) ([]*ChatMessage, int, error) {
	var chatMessages []*ChatMessage
	err := DataBase().Where("chat_context_id = ?", chatContextID).
		WithContext(ctx).
		Offset((page - 1) * size).
		Limit(size).
		Find(&chatMessages).Error
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = DataBase().Model(&ChatMessage{}).
		Where("chat_context_id = ?", chatContextID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return chatMessages, int(total), nil
}

func GetChatContextLastMessage(ctx context.Context, chatContextID int64) (*ChatMessage, error) {
	var chatMessage ChatMessage
	err := DataBase().Where("chat_context_id = ?", chatContextID).
		Order("id DESC").
		Limit(1).
		WithContext(ctx).
		First(&chatMessage).Error
	return &chatMessage, err
}

func GetChatMessageBySender(ctx context.Context, chatContextID, userID int64, page, size int) ([]*ChatMessage, int, error) {
	var chatMessages []*ChatMessage
	err := DataBase().Where("chat_context_id = ?", chatContextID).
		Where("sender = ?", userID).
		WithContext(ctx).
		Offset((page - 1) * size).
		Limit(size).
		Find(&chatMessages).Error
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = DataBase().Model(&ChatMessage{}).
		Where("chat_context_id = ?", chatContextID).
		Where("sender = ?", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return chatMessages, int(total), nil
}

// 根据用户的id获取消息
func GetChatMessageByUserID(ctx context.Context, userID int64, page, size int) ([]*ChatMessage, int, error) {
	var chatMessages []*ChatMessage
	err := DataBase().Where("user_id = ?", userID).
		WithContext(ctx).
		Offset((page - 1) * size).
		Limit(size).
		Find(&chatMessages).Error
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = DataBase().Model(&ChatMessage{}).
		Where("user_id = ?", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return chatMessages, int(total), nil
}

// 根角色的id获取消息
func GetChatMessageByRoleID(ctx context.Context, roleID int64, page, size int) ([]*ChatMessage, int, error) {
	var chatMessages []*ChatMessage
	err := DataBase().Where("role_id = ?", roleID).
		WithContext(ctx).
		Offset((page - 1) * size).
		Limit(size).
		Find(&chatMessages).Error
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = DataBase().Model(&ChatMessage{}).
		Where("role_id = ?", roleID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return chatMessages, int(total), nil
}

// 根据chat_context_id获取批量的消息
func GetChatMessageByChatContextIDBatch(ctx context.Context, chatContextID int64, page, size int) ([]*ChatMessage, int, error) {
	var chatMessages []*ChatMessage
	err := DataBase().Where("chat_context_id = ?", chatContextID).
		WithContext(ctx).
		Offset((page - 1) * size).
		Limit(size).
		Find(&chatMessages).Error
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = DataBase().Model(&ChatMessage{}).
		Where("chat_context_id = ?", chatContextID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	return chatMessages, int(total), nil
}
