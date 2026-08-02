package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

func (s *Service) recordStoryboardConversationTurn(
	ctx context.Context,
	storyboardID, userID, role, messageType, text, taskID, clientMessageID string,
) {
	if s == nil || s.repo == nil {
		return
	}
	storyboardID = strings.TrimSpace(storyboardID)
	userID = strings.TrimSpace(userID)
	text = strings.TrimSpace(text)
	if storyboardID == "" || userID == "" || text == "" {
		return
	}
	if role == "" {
		role = domain.StoryboardConversationRoleStatus
	}
	if messageType == "" {
		messageType = domain.StoryboardConversationTypeStatus
	}
	msg := &domain.StoryboardConversationMessage{
		ID:              uuid.New().String(),
		StoryboardID:    storyboardID,
		UserID:          userID,
		Role:            role,
		MessageType:     messageType,
		Text:            text,
		TaskID:          strings.TrimSpace(taskID),
		ClientMessageID: strings.TrimSpace(clientMessageID),
	}
	if err := s.repo.AppendStoryboardConversationMessage(ctx, msg); err != nil {
		s.logger.Warn("failed to persist storyboard conversation message",
			zap.String("storyboardID", storyboardID),
			zap.String("messageType", messageType),
			zap.Error(err))
	}
}

func (s *Service) recordStoryboardAnalyzeConversation(
	ctx context.Context,
	storyboardID, userID, userInput, assistantMessage string,
) {
	userInput = strings.TrimSpace(userInput)
	assistantMessage = strings.TrimSpace(assistantMessage)
	if userInput != "" {
		s.recordStoryboardConversationTurn(ctx, storyboardID, userID,
			domain.StoryboardConversationRoleUser,
			domain.StoryboardConversationTypeUserInput,
			userInput, "", uuid.New().String())
	}
	if assistantMessage != "" {
		s.recordStoryboardConversationTurn(ctx, storyboardID, userID,
			domain.StoryboardConversationRoleAssistant,
			domain.StoryboardConversationTypeAnalyzeReply,
			assistantMessage, "", uuid.New().String())
	}
}

// ListStoryboardConversationPage returns a chronological page of creator chat history.
func (s *Service) ListStoryboardConversationPage(ctx context.Context, storyboardID string, limit int, beforeCreatedAt int64) ([]*domain.StoryboardConversationMessage, bool, error) {
	if s == nil || s.repo == nil {
		return nil, false, nil
	}
	return s.repo.ListStoryboardConversationMessagesPage(ctx, storyboardID, limit, beforeCreatedAt)
}

// SyncStoryboardConversationMessages upserts client-synced chat turns.
func (s *Service) SyncStoryboardConversationMessages(ctx context.Context, messages []*domain.StoryboardConversationMessage) error {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.UpsertStoryboardConversationMessages(ctx, messages)
}
