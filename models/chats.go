package models

import (
	"context"

	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ChatContext 聊天上下文/会话
// status: 1-open, 2-close, 3-delete
type ChatContext struct {
	IDBase
	UserID      int64  `gorm:"column:user_id" json:"user_id,omitempty"`           // 用户ID
	RoleID      int64  `gorm:"column:role_id" json:"role_id,omitempty"`           // 角色ID
	Title       string `gorm:"column:title" json:"title,omitempty"`               // 会话标题
	Content     string `gorm:"column:content" json:"content,omitempty"`           // 会话内容
	Status      int64  `gorm:"column:status" json:"status,omitempty"`             // 会话状态
	UseAgent    int64  `gorm:"column:use_agent" json:"use_agent,omitempty"`       // 是否使用Agent
	AgentPrompt string `gorm:"column:agent_prompt" json:"agent_prompt,omitempty"` // Agent提示词
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

// ChatMessage 聊天消息
type ChatMessage struct {
	IDBase
	ChatContextID int64  `gorm:"column:chat_context_id" json:"chat_context_id,omitempty"` // 会话ID
	UserID        int64  `gorm:"column:user_id" json:"user_id,omitempty"`                 // 用户ID
	RoleID        int64  `gorm:"column:role_id" json:"role_id,omitempty"`                 // 角色ID
	Sender        int64  `gorm:"column:sender" json:"sender,omitempty"`                   // 发送者ID
	MessageType   int64  `gorm:"column:message_type" json:"message_type,omitempty"`       // 消息类型
	Content       string `gorm:"column:content" json:"content,omitempty"`                 // 消息内容
	Status        int64  `gorm:"column:status" json:"status,omitempty"`                   // 消息状态
	NeedRender    int64  `gorm:"column:is_need_render" json:"is_need_render,omitempty"`   // 是否需要渲染
	Prompt        string `gorm:"column:prompt" json:"prompt,omitempty"`                   // 渲染前prompt
	AfterRender   string `gorm:"column:after_render" json:"after_render,omitempty"`       // 渲染后内容
	UUID          string `gorm:"column:uuid" json:"uuid,omitempty"`                       // 唯一标识
	SendTime      int64  `gorm:"column:send_time" json:"send_time,omitempty"`             // 发送时间
	ReceiveTime   int64  `gorm:"column:receive_time" json:"receive_time,omitempty"`       // 接收时间
	MessageID     string `gorm:"column:message_id" json:"message_id,omitempty"`           // 消息ID
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

// 新增：分页获取ChatContext列表
func GetChatContextList(ctx context.Context, offset, limit int) ([]*ChatContext, error) {
	var ctxs []*ChatContext
	err := DataBase().Model(&ChatContext{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&ctxs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return ctxs, nil
}

// 新增：通过Title唯一查询
func GetChatContextByTitle(ctx context.Context, title string) (*ChatContext, error) {
	chatCtx := &ChatContext{}
	err := DataBase().Model(chatCtx).
		WithContext(ctx).
		Where("title = ?", title).
		First(chatCtx).Error
	if err != nil {
		return nil, err
	}
	return chatCtx, nil
}
