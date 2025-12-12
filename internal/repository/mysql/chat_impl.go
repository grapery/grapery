package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ========== ChatThread Methods ==========

func (r *Repository) ChatThreadByID(ctx context.Context, id string) (*domain.ChatThread, error) {
	var thread ChatThread
	if err := r.db.WithContext(ctx).
		Preload("Character").
		Preload("Character.Author").
		Preload("User").
		First(&thread, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat thread: %w", err)
	}
	dt := r.chatThreadToDomain(thread)
	return &dt, nil
}

func (r *Repository) ChatThreads(ctx context.Context, userID string) ([]*domain.ChatThread, error) {
	var threads []ChatThread
	if err := r.db.WithContext(ctx).
		Preload("Character").
		Preload("Character.Author").
		Where("user_id = ?", userID).
		Order("last_message_time DESC").
		Find(&threads).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat threads: %w", err)
	}

	result := make([]*domain.ChatThread, len(threads))
	for i, t := range threads {
		dt := r.chatThreadToDomain(t)
		result[i] = &dt
	}
	return result, nil
}

func (r *Repository) CreateChatThread(ctx context.Context, thread *domain.ChatThread) error {
	dbThread := ChatThread{
		ID:                   uuid.New().String(),
		CharacterID:          thread.CharacterID,
		UserID:               r.getCurrentUserID(ctx), // 从context获取当前用户ID
		StoryTitle:           thread.StoryTitle,
		LastMessage:          thread.LastMessage,
		LastMessageTime:      time.Unix(thread.LastMessageTime, 0),
		UnreadCount:          thread.UnreadCount,
		MessageCount:         thread.MessageCount,
		InteractionFrequency: thread.InteractionFrequency,
	}

	if err := r.db.WithContext(ctx).Create(&dbThread).Error; err != nil {
		return fmt.Errorf("failed to create chat thread: %w", err)
	}

	thread.ID = dbThread.ID
	thread.CreatedAt = dbThread.CreatedAt.Unix()
	return nil
}

func (r *Repository) UpdateChatThread(ctx context.Context, thread *domain.ChatThread) error {
	updates := map[string]interface{}{
		"last_message":          thread.LastMessage,
		"last_message_time":     thread.LastMessageTime,
		"unread_count":          thread.UnreadCount,
		"message_count":         thread.MessageCount,
		"interaction_frequency": thread.InteractionFrequency,
		"updated_at":            time.Now(),
	}

	if err := r.db.WithContext(ctx).Model(&ChatThread{}).
		Where("id = ?", thread.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update chat thread: %w", err)
	}
	return nil
}

func (r *Repository) DeleteChatThread(ctx context.Context, id string) error {
	// 先删除该thread下的所有消息
	if err := r.db.WithContext(ctx).Where("thread_id = ?", id).Delete(&ChatMessage{}).Error; err != nil {
		return fmt.Errorf("failed to delete chat messages: %w", err)
	}

	// 再删除thread本身
	if err := r.db.WithContext(ctx).Delete(&ChatThread{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete chat thread: %w", err)
	}
	return nil
}

// ========== ChatMessage Methods ==========

func (r *Repository) ChatMessageByID(ctx context.Context, id string) (*domain.ChatMessage, error) {
	var message ChatMessage
	if err := r.db.WithContext(ctx).First(&message, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat message: %w", err)
	}
	dm := r.chatMessageToDomain(message)
	return &dm, nil
}

func (r *Repository) ChatMessages(ctx context.Context, threadID string, limit, offset int) ([]*domain.ChatMessage, error) {
	var messages []ChatMessage
	query := r.db.WithContext(ctx).
		Where("thread_id = ?", threadID).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat messages: %w", err)
	}

	result := make([]*domain.ChatMessage, len(messages))
	for i, m := range messages {
		dm := r.chatMessageToDomain(m)
		result[i] = &dm
	}
	return result, nil
}

func (r *Repository) CreateChatMessage(ctx context.Context, msg *domain.ChatMessage) error {
	dbMsg := ChatMessage{
		ID:           uuid.New().String(),
		ThreadID:     msg.ThreadID,
		SenderID:     msg.SenderID,
		SenderName:   msg.SenderName,
		SenderAvatar: msg.SenderAvatar,
		Content:      msg.Content,
		Image:        msg.Image,
		IsUser:       msg.IsUser,
	}

	if err := r.db.WithContext(ctx).Create(&dbMsg).Error; err != nil {
		return fmt.Errorf("failed to create chat message: %w", err)
	}

	// 更新thread的最后消息和消息计数
	// 使用 gorm.Expr 来实现自增操作，避免 SQL 注入
	updates := map[string]interface{}{
		"last_message":      msg.Content,
		"last_message_time": dbMsg.CreatedAt,
		"message_count":     gorm.Expr("message_count + 1"),
	}

	// 如果是角色发的消息（非用户），增加未读数
	if !msg.IsUser {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
	}

	if err := r.db.WithContext(ctx).Model(&ChatThread{}).
		Where("id = ?", msg.ThreadID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update chat thread: %w", err)
	}

	msg.ID = dbMsg.ID
	msg.Timestamp = dbMsg.CreatedAt.Unix()
	return nil
}

func (r *Repository) DeleteChatMessage(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&ChatMessage{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete chat message: %w", err)
	}
	return nil
}

// ========== Helper Methods ==========

func (r *Repository) chatThreadToDomain(t ChatThread) domain.ChatThread {
	return domain.ChatThread{
		ID:                   t.ID,
		CharacterID:          t.CharacterID,
		CharacterName:        t.Character.Name,
		CharacterAvatar:      t.Character.Avatar,
		StoryTitle:           t.StoryTitle,
		LastMessage:          t.LastMessage,
		LastMessageTime:      t.LastMessageTime.Unix(),
		UnreadCount:          t.UnreadCount,
		MessageCount:         t.MessageCount,
		InteractionFrequency: t.InteractionFrequency,
		CreatedAt:            t.CreatedAt.Unix(),
	}
}

func (r *Repository) chatMessageToDomain(m ChatMessage) domain.ChatMessage {
	return domain.ChatMessage{
		ID:           m.ID,
		ThreadID:     m.ThreadID,
		SenderID:     m.SenderID,
		SenderName:   m.SenderName,
		SenderAvatar: m.SenderAvatar,
		Content:      m.Content,
		Image:        m.Image,
		Timestamp:    m.CreatedAt.Unix(),
		IsUser:       m.IsUser,
	}
}

// getCurrentUserID 从context获取当前用户ID
// 使用 auth 包提供的类型安全的 context key 来获取用户ID
// 用户ID由 AuthMiddleware 在认证成功后注入到 request context 中
func (r *Repository) getCurrentUserID(ctx context.Context) string {
	return auth.UserIDFromContext(ctx)
}
