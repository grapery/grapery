package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// WritersRoomService handles writers room business logic
type WritersRoomService struct {
	repo   domain.Repository
	logger *zap.Logger
}

// NewWritersRoomService creates a new writers room service
func NewWritersRoomService(repo domain.Repository, logger *zap.Logger) *WritersRoomService {
	return &WritersRoomService{
		repo:   repo,
		logger: logger,
	}
}

// GetOrCreateRoom gets or creates a writers room for a story
func (s *WritersRoomService) GetOrCreateRoom(ctx context.Context, storyID string) (*domain.WritersRoom, error) {
	// Try to get existing room
	room, err := s.repo.WritersRoomByStoryID(ctx, storyID)
	if err != nil {
		s.logger.Error("failed to get writers room", zap.Error(err), zap.String("storyId", storyID))
		return nil, fmt.Errorf("failed to get writers room: %w", err)
	}

	// If room exists, sync participants and return
	if room != nil {
		// Sync participants from story contributors
		if err := s.SyncParticipants(ctx, room.ID); err != nil {
			s.logger.Warn("failed to sync participants", zap.Error(err))
			// Continue anyway, don't fail the request
		}
		return room, nil
	}

	// Get story details
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		s.logger.Error("story not found", zap.Error(err), zap.String("storyId", storyID))
		return nil, fmt.Errorf("story not found: %w", err)
	}

	// Create new room
	newRoom := &domain.WritersRoom{
		ID:               uuid.New().String(),
		StoryID:          storyID,
		Title:            fmt.Sprintf("%s Writers Room", story.Title),
		LastMessage:      "",
		LastMessageTime:  0,
		MessageCount:     0,
		ParticipantCount: 0,
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	if err := s.repo.CreateWritersRoom(ctx, newRoom); err != nil {
		s.logger.Error("failed to create writers room", zap.Error(err))
		return nil, fmt.Errorf("failed to create writers room: %w", err)
	}

	s.logger.Info("writers room created",
		zap.String("roomId", newRoom.ID),
		zap.String("storyId", storyID))

	// Sync participants
	if err := s.SyncParticipants(ctx, newRoom.ID); err != nil {
		s.logger.Warn("failed to sync participants", zap.Error(err))
	}

	return newRoom, nil
}

// GetRoom gets a writers room with permission check
func (s *WritersRoomService) GetRoom(ctx context.Context, roomID, userID string) (*domain.WritersRoom, error) {
	room, err := s.repo.WritersRoomByStoryID(ctx, roomID)
	if err != nil {
		s.logger.Error("failed to get writers room", zap.Error(err), zap.String("roomId", roomID))
		return nil, fmt.Errorf("failed to get writers room: %w", err)
	}

	if room == nil {
		return nil, fmt.Errorf("writers room not found")
	}

	// Check if user is a participant in the room
	isParticipant, err := s.repo.IsWritersRoomParticipant(ctx, roomID, userID)
	if err != nil {
		s.logger.Error("failed to check participant", zap.Error(err))
		return nil, fmt.Errorf("failed to check participant: %w", err)
	}

	if !isParticipant {
		// Check if user is a story contributor
		isContributor, err := s.repo.IsStoryContributor(ctx, room.StoryID, userID)
		if err != nil {
			s.logger.Error("failed to check story contributor", zap.Error(err))
			return nil, fmt.Errorf("failed to check story contributor: %w", err)
		}

		if !isContributor {
			return nil, fmt.Errorf("unauthorized: you don't have access to this writers room")
		}
	}

	return room, nil
}

// SyncParticipants syncs story contributors to writers room participants
func (s *WritersRoomService) SyncParticipants(ctx context.Context, roomID string) error {
	// Get room details to find story ID
	room, err := s.repo.WritersRoomByStoryID(ctx, roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return fmt.Errorf("room not found")
	}

	// Get story contributors
	limit := 100
	offset := 0
	contributors, err := s.repo.GetStoryContributors(ctx, room.StoryID, limit, offset)
	if err != nil {
		return err
	}

	// Get existing participants
	existingParticipants, err := s.repo.WritersRoomParticipants(ctx, roomID)
	if err != nil {
		return err
	}

	// Create a map of existing participants
	existingMap := make(map[string]*domain.WritersRoomParticipant)
	for _, p := range existingParticipants {
		existingMap[p.UserID] = p
	}

	// Add new participants
	for _, contributor := range contributors {
		// Skip if already a participant
		if _, exists := existingMap[contributor.UserID]; exists {
			continue
		}

		// Determine role
		role := domain.WritersRoomRoleMember
		if contributor.Role == domain.StoryRoleOwner {
			role = domain.WritersRoomRoleOwner
		} else if contributor.Role == domain.StoryRoleCollaborator {
			role = domain.WritersRoomRoleAdmin
		}

		// Create participant
		participant := &domain.WritersRoomParticipant{
			ID:        uuid.New().String(),
			RoomID:    roomID,
			UserID:    contributor.UserID,
			Role:      role,
			JoinedAt:  time.Now().Unix(),
			LastReadAt: time.Now().Unix(),
		}

		if err := s.repo.AddWritersRoomParticipant(ctx, participant); err != nil {
			s.logger.Error("failed to add participant",
				zap.Error(err),
				zap.String("userId", contributor.UserID),
				zap.String("roomId", roomID))
			// Continue with next participant
		}
	}

	// Update participant count
	if err := s.repo.UpdateWritersRoom(ctx, &domain.WritersRoom{
		ID:               roomID,
		ParticipantCount: len(contributors),
	}); err != nil {
		s.logger.Warn("failed to update participant count", zap.Error(err))
	}

	return nil
}

// ListMessages lists messages in a writers room with pagination
func (s *WritersRoomService) ListMessages(ctx context.Context, roomID, userID string, limit, offset int) (*domain.MessagesListResponse, error) {
	// Permission check
	_, err := s.GetRoom(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	// Get messages
	messages, err := s.repo.WritersRoomMessages(ctx, roomID, limit, offset)
	if err != nil {
		s.logger.Error("failed to list messages", zap.Error(err))
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Get unread count
	unreadCount, err := s.repo.WritersRoomUnreadCount(ctx, roomID, userID)
	if err != nil {
		s.logger.Warn("failed to get unread count", zap.Error(err))
		unreadCount = 0
	}

	// Enhance messages with additional info
	enhancedMessages := make([]*domain.WritersRoomMessageResponse, len(messages))
	for i, msg := range messages {
		enhancedMessages[i] = s.enrichMessage(ctx, msg, userID)
	}

	return &domain.MessagesListResponse{
		Messages:    enhancedMessages,
		Count:       len(messages),
		HasMore:     len(messages) == limit,
		UnreadCount: unreadCount,
	}, nil
}

// SendMessage sends a message to a writers room
func (s *WritersRoomService) SendMessage(ctx context.Context, req *domain.WritersRoomSendMessageRequest, userID string) (*domain.WritersRoomMessageResponse, error) {
	// Permission check
	_, err := s.GetRoom(ctx, req.RoomID, userID)
	if err != nil {
		return nil, err
	}

	// Get sender participant info
	participants, err := s.repo.WritersRoomParticipants(ctx, req.RoomID)
	if err != nil {
		s.logger.Error("failed to get participants", zap.Error(err))
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	var senderID string
	for _, p := range participants {
		if p.UserID == userID {
			senderID = p.ID
			break
		}
	}

	if senderID == "" {
		return nil, fmt.Errorf("user is not a participant in this room")
	}

	// Determine message type
	messageType := req.MessageType
	if messageType == "" {
		messageType = domain.WritersRoomMsgTypeText
		if len(req.Attachments) > 0 {
			messageType = domain.WritersRoomMsgTypeMixed
		}
	}

	// Create message
	message := &domain.WritersRoomMessage{
		ID:          uuid.New().String(),
		RoomID:      req.RoomID,
		SenderID:    senderID,
		Content:     req.Content,
		MessageType: messageType,
		Attachments:  req.Attachments,
		Mentions:     req.Mentions,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	if req.ReplyToMessageID != "" {
		message.ReplyToMessageID = &req.ReplyToMessageID
	}

	if err := s.repo.CreateWritersRoomMessage(ctx, message); err != nil {
		s.logger.Error("failed to create message", zap.Error(err))
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Update room stats
	if err := s.repo.IncrementMessageCount(ctx, req.RoomID); err != nil {
		s.logger.Warn("failed to increment message count", zap.Error(err))
	}

	// Update room last message
	previewContent := req.Content
	if len(previewContent) > 100 {
		previewContent = previewContent[:100] + "..."
	}
	if err := s.repo.UpdateRoomLastMessage(ctx, req.RoomID, previewContent, message.CreatedAt); err != nil {
		s.logger.Warn("failed to update room last message", zap.Error(err))
	}

	s.logger.Info("message sent",
		zap.String("messageId", message.ID),
		zap.String("roomId", req.RoomID),
		zap.String("userId", userID))

	// Return enriched message
	enriched := s.enrichMessage(ctx, message, userID)
	return enriched, nil
}

// DeleteMessage deletes a message from a writers room
func (s *WritersRoomService) DeleteMessage(ctx context.Context, messageID, userID string) error {
	// Get message
	message, err := s.repo.WritersRoomMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}
	if message == nil {
		return fmt.Errorf("message not found")
	}

	// Get sender participant info
	participants, err := s.repo.WritersRoomParticipants(ctx, message.RoomID)
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	// Check if user is the sender
	var isSender bool
	for _, p := range participants {
		if p.UserID == userID {
			isSender = (p.ID == message.SenderID)
			break
		}
	}

	if !isSender {
		return fmt.Errorf("unauthorized: you can only delete your own messages")
	}

	// Delete message
	if err := s.repo.DeleteWritersRoomMessage(ctx, messageID); err != nil {
		s.logger.Error("failed to delete message", zap.Error(err))
		return fmt.Errorf("failed to delete message: %w", err)
	}

	s.logger.Info("message deleted",
		zap.String("messageId", messageID),
		zap.String("userId", userID))

	return nil
}

// MarkMessageAsRead marks a message as read by a user
func (s *WritersRoomService) MarkMessageAsRead(ctx context.Context, messageID, userID string) error {
	// Get message to verify room access
	message, err := s.repo.WritersRoomMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}
	if message == nil {
		return fmt.Errorf("message not found")
	}

	// Permission check
	_, err = s.GetRoom(ctx, message.RoomID, userID)
	if err != nil {
		return err
	}

	// Mark as read
	if err := s.repo.MarkMessageAsRead(ctx, messageID, userID); err != nil {
		s.logger.Error("failed to mark message as read", zap.Error(err))
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	// Update participant last read
	if err := s.repo.UpdateParticipantLastRead(ctx, message.RoomID, userID); err != nil {
		s.logger.Warn("failed to update participant last read", zap.Error(err))
	}

	return nil
}

// MarkAllAsRead marks all messages in a room as read by a user
func (s *WritersRoomService) MarkAllAsRead(ctx context.Context, roomID, userID string) error {
	// Permission check
	_, err := s.GetRoom(ctx, roomID, userID)
	if err != nil {
		return err
	}

	// Update participant last read to now
	if err := s.repo.UpdateParticipantLastRead(ctx, roomID, userID); err != nil {
		s.logger.Error("failed to update participant last read", zap.Error(err))
		return fmt.Errorf("failed to update participant last read: %w", err)
	}

	return nil
}

// AddReaction adds a reaction to a message
func (s *WritersRoomService) AddReaction(ctx context.Context, messageID, userID, reactionType, emojiCode string) error {
	// Get message to verify room access
	message, err := s.repo.WritersRoomMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}
	if message == nil {
		return fmt.Errorf("message not found")
	}

	// Permission check
	_, err = s.GetRoom(ctx, message.RoomID, userID)
	if err != nil {
		return err
	}

	// Create reaction
	reaction := &domain.WritersRoomMessageReaction{
		ID:           uuid.New().String(),
		MessageID:    messageID,
		UserID:       userID,
		ReactionType:  reactionType,
		EmojiCode:     emojiCode,
		CreatedAt:    time.Now().Unix(),
	}

	if err := s.repo.CreateWritersRoomMessageReaction(ctx, reaction); err != nil {
		s.logger.Error("failed to create reaction", zap.Error(err))
		return fmt.Errorf("failed to create reaction: %w", err)
	}

	s.logger.Info("reaction added",
		zap.String("messageId", messageID),
		zap.String("userId", userID),
		zap.String("reactionType", reactionType))

	return nil
}

// RemoveReaction removes a reaction from a message
func (s *WritersRoomService) RemoveReaction(ctx context.Context, messageID, userID string) error {
	// Get message to verify room access
	message, err := s.repo.WritersRoomMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}
	if message == nil {
		return fmt.Errorf("message not found")
	}

	// Permission check
	_, err = s.GetRoom(ctx, message.RoomID, userID)
	if err != nil {
		return err
	}

	// Delete reaction
	if err := s.repo.DeleteWritersRoomMessageReaction(ctx, messageID, userID); err != nil {
		s.logger.Error("failed to delete reaction", zap.Error(err))
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	s.logger.Info("reaction removed",
		zap.String("messageId", messageID),
		zap.String("userId", userID))

	return nil
}

// UnreadCount gets unread message count for a user in a room
func (s *WritersRoomService) UnreadCount(ctx context.Context, roomID, userID string) (int, error) {
	// Permission check
	_, err := s.GetRoom(ctx, roomID, userID)
	if err != nil {
		return 0, err
	}

	return s.repo.WritersRoomUnreadCount(ctx, roomID, userID)
}

// enrichMessage enriches a message with additional information
func (s *WritersRoomService) enrichMessage(ctx context.Context, msg *domain.WritersRoomMessage, userID string) *domain.WritersRoomMessageResponse {
	response := &domain.WritersRoomMessageResponse{
		ID:             msg.ID,
		RoomID:         msg.RoomID,
		Sender:         domain.MessageSenderInfo{
			ID:     msg.SenderID,
			Name:   msg.SenderName,
			Avatar: msg.SenderAvatar,
		},
		Content:        msg.Content,
		MessageType:    msg.MessageType,
		Attachments:    msg.Attachments,
		ReplyToMessageID: msg.ReplyToMessageID,
		CreatedAt:      msg.CreatedAt,
		IsMine:         false, // Will be set below
	}

	// Check if message is from current user
	response.IsMine = (msg.SenderID == userID || msg.SenderName == "You")

	// Add mentions info
	if len(msg.Mentions) > 0 {
		mentionInfos := make([]domain.MentionInfo, 0, len(msg.Mentions))
		for _, uid := range msg.Mentions {
			user, err := s.repo.UserByID(ctx, uid)
			if err == nil && user != nil {
				mentionInfos = append(mentionInfos, domain.MentionInfo{
					ID:   user.ID,
					Name: user.DisplayName,
				})
			}
		}
		response.Mentions = mentionInfos
	}

	// Add reactions
	reactions, err := s.repo.WritersRoomMessageReactions(ctx, msg.ID)
	if err == nil && len(reactions) > 0 {
		// Group reactions by type
		reactionMap := make(map[string]*domain.ReactionSummary)
		for _, r := range reactions {
			key := r.ReactionType
			if r.EmojiCode != "" {
				key = r.EmojiCode
			}

			if reactionMap[key] == nil {
				reactionMap[key] = &domain.ReactionSummary{
					ReactionType: r.ReactionType,
					EmojiCode:     r.EmojiCode,
					Count:        0,
					Users:        []string{},
				}
			}
			reactionMap[key].Count++
			reactionMap[key].Users = append(reactionMap[key].Users, r.UserID)
		}

		// Convert to slice
		summaries := make([]domain.ReactionSummary, 0, len(reactionMap))
		for _, r := range reactionMap {
			summaries = append(summaries, *r)
		}
		response.Reactions = summaries
	}

	// Add read receipts
	receipts, err := s.repo.MessageReadReceipts(ctx, msg.ID)
	if err == nil {
		receiptInfos := make([]domain.ReadReceiptInfo, 0, len(receipts))
		for _, r := range receipts {
			receiptInfos = append(receiptInfos, domain.ReadReceiptInfo{
				UserID: r.UserID,
				UserName: r.UserName,
				ReadAt:  r.ReadAt,
			})
		}
		response.ReadReceipts = receiptInfos
	}

	// Load reply to message if exists
	if msg.ReplyToMessageID != nil && *msg.ReplyToMessageID != "" {
		replyMsg, err := s.repo.WritersRoomMessageByID(ctx, *msg.ReplyToMessageID)
		if err == nil && replyMsg != nil {
			response.ReplyToMessage = replyMsg
		}
	}

	return response
}
