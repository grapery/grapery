package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ========== Writers Room Operations ==========

// WritersRoomByStoryID retrieves the writers room for a story
func (r *Repository) WritersRoomByStoryID(ctx context.Context, storyID string) (*domain.WritersRoom, error) {
	var dbRoom WritersRoomDB
	err := r.db.WithContext(ctx).Where("story_id = ?", storyID).First(&dbRoom).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Room doesn't exist yet
		}
		r.log.Error("failed to get writers room by story ID", zap.Error(err), zap.String("storyId", storyID))
		return nil, fmt.Errorf("failed to get writers room: %w", err)
	}

	return dbRoom.ToDomain(), nil
}

// CreateWritersRoom creates a new writers room
func (r *Repository) CreateWritersRoom(ctx context.Context, room *domain.WritersRoom) error {
	dbRoom := WritersRoomDB{
		ID:               room.ID,
		StoryID:          room.StoryID,
		Title:            room.Title,
		LastMessage:      room.LastMessage,
		LastMessageTime:  room.LastMessageTime,
		MessageCount:     room.MessageCount,
		ParticipantCount: room.ParticipantCount,
		CreatedAt:        room.CreatedAt,
		UpdatedAt:        room.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&dbRoom).Error; err != nil {
		r.log.Error("failed to create writers room", zap.Error(err))
		return fmt.Errorf("failed to create writers room: %w", err)
	}

	return nil
}

// UpdateWritersRoom updates an existing writers room
func (r *Repository) UpdateWritersRoom(ctx context.Context, room *domain.WritersRoom) error {
	dbRoom := WritersRoomDB{
		ID:               room.ID,
		StoryID:          room.StoryID,
		Title:            room.Title,
		LastMessage:      room.LastMessage,
		LastMessageTime:  room.LastMessageTime,
		MessageCount:     room.MessageCount,
		ParticipantCount: room.ParticipantCount,
		CreatedAt:        room.CreatedAt,
		UpdatedAt:        time.Now().Unix(),
	}

	if err := r.db.WithContext(ctx).Model(&dbRoom).Updates(map[string]interface{}{
		"title":             dbRoom.Title,
		"last_message":      dbRoom.LastMessage,
		"last_message_time": dbRoom.LastMessageTime,
		"message_count":     dbRoom.MessageCount,
		"participant_count": dbRoom.ParticipantCount,
		"updated_at":        dbRoom.UpdatedAt,
	}).Error; err != nil {
		r.log.Error("failed to update writers room", zap.Error(err))
		return fmt.Errorf("failed to update writers room: %w", err)
	}

	return nil
}

// DeleteWritersRoom deletes a writers room
func (r *Repository) DeleteWritersRoom(ctx context.Context, roomID string) error {
	if err := r.db.WithContext(ctx).Delete(&WritersRoomDB{}, "id = ?", roomID).Error; err != nil {
		r.log.Error("failed to delete writers room", zap.Error(err), zap.String("roomId", roomID))
		return fmt.Errorf("failed to delete writers room: %w", err)
	}
	return nil
}

// ========== Writers Room Participants ==========

// WritersRoomParticipants retrieves all participants of a writers room
func (r *Repository) WritersRoomParticipants(ctx context.Context, roomID string) ([]*domain.WritersRoomParticipant, error) {
	var dbParticipants []WritersRoomParticipantDB
	err := r.db.WithContext(ctx).Where("room_id = ?", roomID).
		Preload("User").
		Find(&dbParticipants).Error
	if err != nil {
		r.log.Error("failed to get writers room participants", zap.Error(err))
		return nil, fmt.Errorf("failed to get writers room participants: %w", err)
	}

	participants := make([]*domain.WritersRoomParticipant, len(dbParticipants))
	for i, dbp := range dbParticipants {
		participants[i] = dbp.ToDomain()
	}

	return participants, nil
}

// AddWritersRoomParticipant adds a participant to a writers room
func (r *Repository) AddWritersRoomParticipant(ctx context.Context, participant *domain.WritersRoomParticipant) error {
	dbParticipant := WritersRoomParticipantDB{
		ID:         participant.ID,
		RoomID:     participant.RoomID,
		UserID:     participant.UserID,
		Role:       participant.Role,
		JoinedAt:   participant.JoinedAt,
		LastReadAt: participant.LastReadAt,
	}

	if err := r.db.WithContext(ctx).Create(&dbParticipant).Error; err != nil {
		r.log.Error("failed to add writers room participant", zap.Error(err))
		return fmt.Errorf("failed to add writers room participant: %w", err)
	}

	return nil
}

// RemoveWritersRoomParticipant removes a participant from a writers room
func (r *Repository) RemoveWritersRoomParticipant(ctx context.Context, roomID, userID string) error {
	if err := r.db.WithContext(ctx).Delete(&WritersRoomParticipantDB{}, "room_id = ? AND user_id = ?", roomID, userID).Error; err != nil {
		r.log.Error("failed to remove writers room participant", zap.Error(err))
		return fmt.Errorf("failed to remove writers room participant: %w", err)
	}
	return nil
}

// IsWritersRoomParticipant checks if a user is a participant in a writers room
func (r *Repository) IsWritersRoomParticipant(ctx context.Context, roomID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&WritersRoomParticipantDB{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	if err != nil {
		r.log.Error("failed to check writers room participant", zap.Error(err))
		return false, fmt.Errorf("failed to check writers room participant: %w", err)
	}

	return count > 0, nil
}

// UpdateParticipantLastRead updates the last read timestamp for a participant
func (r *Repository) UpdateParticipantLastRead(ctx context.Context, roomID, userID string) error {
	now := time.Now().Unix()
	if err := r.db.WithContext(ctx).Model(&WritersRoomParticipantDB{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Update("last_read_at", now).Error; err != nil {
		r.log.Error("failed to update participant last read", zap.Error(err))
		return fmt.Errorf("failed to update participant last read: %w", err)
	}
	return nil
}

// IncrementParticipantCount increments the participant count for a room
func (r *Repository) IncrementParticipantCount(ctx context.Context, roomID string) error {
	if err := r.db.WithContext(ctx).Model(&WritersRoomDB{}).
		Where("id = ?", roomID).
		UpdateColumn("participant_count", gorm.Expr("participant_count + 1")).Error; err != nil {
		r.log.Error("failed to increment participant count", zap.Error(err))
		return fmt.Errorf("failed to increment participant count: %w", err)
	}
	return nil
}

// DecrementParticipantCount decrements the participant count for a room
func (r *Repository) DecrementParticipantCount(ctx context.Context, roomID string) error {
	if err := r.db.WithContext(ctx).Model(&WritersRoomDB{}).
		Where("id = ?", roomID).
		UpdateColumn("participant_count", gorm.Expr("GREATEST(participant_count - 1, 0)")).Error; err != nil {
		r.log.Error("failed to decrement participant count", zap.Error(err))
		return fmt.Errorf("failed to decrement participant count: %w", err)
	}
	return nil
}

// ========== Writers Room Messages ==========

// WritersRoomMessages retrieves messages from a writers room with pagination
func (r *Repository) WritersRoomMessages(ctx context.Context, roomID string, limit, offset int) ([]*domain.WritersRoomMessage, error) {
	var dbMessages []WritersRoomMessageDB

	query := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.
		Preload("Sender.User").
		Find(&dbMessages).Error

	if err != nil {
		r.log.Error("failed to get writers room messages", zap.Error(err))
		return nil, fmt.Errorf("failed to get writers room messages: %w", err)
	}

	// Reverse to get chronological order
	messages := make([]*domain.WritersRoomMessage, len(dbMessages))
	for i := 0; i < len(dbMessages); i++ {
		messages[len(dbMessages)-1-i] = dbMessages[i].ToDomain()
	}

	return messages, nil
}

// WritersRoomMessageByID retrieves a single message by ID
func (r *Repository) WritersRoomMessageByID(ctx context.Context, messageID string) (*domain.WritersRoomMessage, error) {
	var dbMessage WritersRoomMessageDB
	err := r.db.WithContext(ctx).
		Preload("Sender.User").
		Where("id = ?", messageID).
		First(&dbMessage).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.log.Error("failed to get writers room message", zap.Error(err))
		return nil, fmt.Errorf("failed to get writers room message: %w", err)
	}

	return dbMessage.ToDomain(), nil
}

// CreateWritersRoomMessage creates a new message in the writers room
func (r *Repository) CreateWritersRoomMessage(ctx context.Context, msg *domain.WritersRoomMessage) error {
	// Convert attachments to JSON
	var attachmentsJSON string
	if len(msg.Attachments) > 0 {
		data, err := json.Marshal(msg.Attachments)
		if err != nil {
			return fmt.Errorf("failed to marshal attachments: %w", err)
		}
		attachmentsJSON = string(data)
	}

	// Convert mentions to JSON
	var mentionsJSON string
	if len(msg.Mentions) > 0 {
		data, err := json.Marshal(msg.Mentions)
		if err != nil {
			return fmt.Errorf("failed to marshal mentions: %w", err)
		}
		mentionsJSON = string(data)
	}

	dbMessage := WritersRoomMessageDB{
		ID:               msg.ID,
		RoomID:           msg.RoomID,
		SenderID:         msg.SenderID,
		Content:          msg.Content,
		MessageType:      msg.MessageType,
		AttachmentsJSON:  attachmentsJSON,
		MentionsJSON:     mentionsJSON,
		ReplyToMessageID: msg.ReplyToMessageID,
		CreatedAt:        msg.CreatedAt,
		UpdatedAt:        msg.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&dbMessage).Error; err != nil {
		r.log.Error("failed to create writers room message", zap.Error(err))
		return fmt.Errorf("failed to create writers room message: %w", err)
	}

	return nil
}

// DeleteWritersRoomMessage deletes a message from the writers room
func (r *Repository) DeleteWritersRoomMessage(ctx context.Context, messageID string) error {
	if err := r.db.WithContext(ctx).Delete(&WritersRoomMessageDB{}, "id = ?", messageID).Error; err != nil {
		r.log.Error("failed to delete writers room message", zap.Error(err))
		return fmt.Errorf("failed to delete writers room message: %w", err)
	}
	return nil
}

// IncrementMessageCount increments the message count for a room
func (r *Repository) IncrementMessageCount(ctx context.Context, roomID string) error {
	if err := r.db.WithContext(ctx).Model(&WritersRoomDB{}).
		Where("id = ?", roomID).
		UpdateColumn("message_count", gorm.Expr("message_count + 1")).Error; err != nil {
		r.log.Error("failed to increment message count", zap.Error(err))
		return fmt.Errorf("failed to increment message count: %w", err)
	}
	return nil
}

// UpdateRoomLastMessage updates the last message info for a room
func (r *Repository) UpdateRoomLastMessage(ctx context.Context, roomID string, lastMessage string, lastMessageTime int64) error {
	if err := r.db.WithContext(ctx).Model(&WritersRoomDB{}).
		Where("id = ?", roomID).
		Updates(map[string]interface{}{
			"last_message":      lastMessage,
			"last_message_time": lastMessageTime,
			"updated_at":        time.Now().Unix(),
		}).Error; err != nil {
		r.log.Error("failed to update room last message", zap.Error(err))
		return fmt.Errorf("failed to update room last message: %w", err)
	}
	return nil
}

// ========== Writers Room Message Reactions ==========

// WritersRoomMessageReactions retrieves all reactions for a message
func (r *Repository) WritersRoomMessageReactions(ctx context.Context, messageID string) ([]*domain.WritersRoomMessageReaction, error) {
	var dbReactions []WritersRoomMessageReactionDB
	err := r.db.WithContext(ctx).Where("message_id = ?", messageID).
		Preload("User").
		Find(&dbReactions).Error
	if err != nil {
		r.log.Error("failed to get message reactions", zap.Error(err))
		return nil, fmt.Errorf("failed to get message reactions: %w", err)
	}

	reactions := make([]*domain.WritersRoomMessageReaction, len(dbReactions))
	for i, dbr := range dbReactions {
		reactions[i] = dbr.ToDomain()
	}

	return reactions, nil
}

// CreateWritersRoomMessageReaction adds a reaction to a message
func (r *Repository) CreateWritersRoomMessageReaction(ctx context.Context, reaction *domain.WritersRoomMessageReaction) error {
	dbReaction := WritersRoomMessageReactionDB{
		ID:           uuid.New().String(),
		MessageID:    reaction.MessageID,
		UserID:       reaction.UserID,
		ReactionType: reaction.ReactionType,
		EmojiCode:    reaction.EmojiCode,
		CreatedAt:    reaction.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&dbReaction).Error; err != nil {
		r.log.Error("failed to create message reaction", zap.Error(err))
		return fmt.Errorf("failed to create message reaction: %w", err)
	}

	return nil
}

// DeleteWritersRoomMessageReaction removes a reaction from a message
func (r *Repository) DeleteWritersRoomMessageReaction(ctx context.Context, messageID, userID string) error {
	if err := r.db.WithContext(ctx).Delete(&WritersRoomMessageReactionDB{}, "message_id = ? AND user_id = ?", messageID, userID).Error; err != nil {
		r.log.Error("failed to delete message reaction", zap.Error(err))
		return fmt.Errorf("failed to delete message reaction: %w", err)
	}
	return nil
}

// ========== Message Read Receipts ==========

// MessageReadReceipts retrieves all read receipts for a message
func (r *Repository) MessageReadReceipts(ctx context.Context, messageID string) ([]*domain.MessageReadReceipt, error) {
	var dbReceipts []MessageReadReceiptDB
	err := r.db.WithContext(ctx).Where("message_id = ?", messageID).
		Preload("User").
		Find(&dbReceipts).Error
	if err != nil {
		r.log.Error("failed to get message read receipts", zap.Error(err))
		return nil, fmt.Errorf("failed to get message read receipts: %w", err)
	}

	receipts := make([]*domain.MessageReadReceipt, len(dbReceipts))
	for i, dbr := range dbReceipts {
		receipts[i] = dbr.ToDomain()
	}

	return receipts, nil
}

// CreateMessageReadReceipt creates a read receipt for a message
func (r *Repository) CreateMessageReadReceipt(ctx context.Context, receipt *domain.MessageReadReceipt) error {
	dbReceipt := MessageReadReceiptDB{
		ID:        uuid.New().String(),
		MessageID: receipt.MessageID,
		UserID:    receipt.UserID,
		ReadAt:    receipt.ReadAt,
	}

	if err := r.db.WithContext(ctx).Create(&dbReceipt).Error; err != nil {
		r.log.Error("failed to create message read receipt", zap.Error(err))
		return fmt.Errorf("failed to create message read receipt: %w", err)
	}

	return nil
}

// UpdateMessageReadReceipt updates or creates a read receipt for a message
func (r *Repository) UpdateMessageReadReceipt(ctx context.Context, messageID, userID string) error {
	now := time.Now().Unix()

	// Try to update first
	result := r.db.WithContext(ctx).Model(&MessageReadReceiptDB{}).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Update("read_at", now)

	if result.Error != nil {
		r.log.Error("failed to update message read receipt", zap.Error(result.Error))
		return fmt.Errorf("failed to update message read receipt: %w", result.Error)
	}

	// If no rows were affected, create a new receipt
	if result.RowsAffected == 0 {
		return r.CreateMessageReadReceipt(ctx, &domain.MessageReadReceipt{
			MessageID: messageID,
			UserID:    userID,
			ReadAt:    now,
		})
	}

	return nil
}

// MarkMessageAsRead marks a message as read by a user
func (r *Repository) MarkMessageAsRead(ctx context.Context, messageID, userID string) error {
	return r.UpdateMessageReadReceipt(ctx, messageID, userID)
}

// WritersRoomUnreadCount calculates unread message count for a user in a room
func (r *Repository) WritersRoomUnreadCount(ctx context.Context, roomID, userID string) (int, error) {
	var participant WritersRoomParticipantDB
	err := r.db.WithContext(ctx).Where("room_id = ? AND user_id = ?", roomID, userID).First(&participant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil // User not in room, no unread messages
		}
		r.log.Error("failed to get participant for unread count", zap.Error(err))
		return 0, fmt.Errorf("failed to get participant: %w", err)
	}

	// Count messages created after participant's last read time
	var count int64
	err = r.db.WithContext(ctx).Model(&WritersRoomMessageDB{}).
		Where("room_id = ? AND created_at > ?", roomID, participant.LastReadAt).
		Count(&count).Error
	if err != nil {
		r.log.Error("failed to count unread messages", zap.Error(err))
		return 0, fmt.Errorf("failed to count unread messages: %w", err)
	}

	return int(count), nil
}

// WritersRoomMessageReactionByID gets a reaction by ID
func (r *Repository) WritersRoomMessageReactionByID(ctx context.Context, reactionID string) (*domain.WritersRoomMessageReaction, error) {
	var dbReaction WritersRoomMessageReactionDB
	err := r.db.WithContext(ctx).Where("id = ?", reactionID).
		Preload("User").
		First(&dbReaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.log.Error("failed to get writers room message reaction", zap.Error(err), zap.String("reactionId", reactionID))
		return nil, fmt.Errorf("failed to get writers room message reaction: %w", err)
	}

	return dbReaction.ToDomain(), nil
}
