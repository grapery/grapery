package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

func (s *FragmentGenerationService) recordFragmentConversationTurn(
	ctx context.Context,
	fragmentID, userID, role, messageType, text, taskID, clientMessageID string,
) {
	if s == nil || s.repo == nil {
		return
	}
	fragmentID = strings.TrimSpace(fragmentID)
	userID = strings.TrimSpace(userID)
	text = strings.TrimSpace(text)
	if fragmentID == "" || userID == "" || text == "" {
		return
	}
	if role == "" {
		role = domain.FragmentConversationRoleStatus
	}
	if messageType == "" {
		messageType = domain.FragmentConversationTypeStatus
	}
	clientID := strings.TrimSpace(clientMessageID)
	msg := &domain.FragmentConversationMessage{
		ID:              uuid.New().String(),
		FragmentID:      fragmentID,
		UserID:          userID,
		Role:            role,
		MessageType:     messageType,
		Text:            text,
		TaskID:          strings.TrimSpace(taskID),
		ClientMessageID: clientID,
	}
	if err := s.repo.AppendFragmentConversationMessage(ctx, msg); err != nil {
		s.logger.Warn("failed to persist fragment conversation message",
			zap.String("fragmentID", fragmentID),
			zap.String("messageType", messageType),
			zap.Error(err))
	}
}

func (s *FragmentGenerationService) recordFragmentAnalyzeConversation(
	ctx context.Context,
	fragmentID, userID, userInput, assistantMessage string,
) {
	userInput = strings.TrimSpace(userInput)
	assistantMessage = strings.TrimSpace(assistantMessage)
	if userInput != "" {
		s.recordFragmentConversationTurn(ctx, fragmentID, userID,
			domain.FragmentConversationRoleUser,
			domain.FragmentConversationTypeUserInput,
			userInput, "", uuid.New().String())
	}
	if assistantMessage != "" {
		s.recordFragmentConversationTurn(ctx, fragmentID, userID,
			domain.FragmentConversationRoleAssistant,
			domain.FragmentConversationTypeAnalyzeReply,
			assistantMessage, "", uuid.New().String())
	}
}

func (s *FragmentGenerationService) recordFragmentGenerationUserInput(
	ctx context.Context,
	fragmentID, userID, taskID, userInput string,
) {
	if taskID == "" {
		return
	}
	s.recordFragmentConversationTurn(ctx, fragmentID, userID,
		domain.FragmentConversationRoleUser,
		domain.FragmentConversationTypeUserInput,
		userInput, taskID, taskID+":user_input")
}

func (s *FragmentGenerationService) recordFragmentGenerationOutputs(
	ctx context.Context,
	fragmentID, userID, taskID string,
	result *domain.FragmentGenerationResult,
) {
	if result == nil {
		return
	}
	if content := strings.TrimSpace(result.Content); content != "" {
		s.recordFragmentConversationTurn(ctx, fragmentID, userID,
			domain.FragmentConversationRoleAssistant,
			domain.FragmentConversationTypeStory,
			content, taskID, taskID+":story")
	}
	if len(result.ScenePlan) > 0 {
		lines := make([]string, 0, len(result.ScenePlan))
		for i, scene := range result.ScenePlan {
			caption := strings.TrimSpace(scene.SceneDesc)
			if caption == "" {
				continue
			}
			pageIndex := fragmentScenePlanDisplayPageIndex(i, scene)
			lines = append(lines, "第"+strconv.Itoa(pageIndex)+"页："+caption)
		}
		if len(lines) > 0 {
			s.recordFragmentConversationTurn(ctx, fragmentID, userID,
				domain.FragmentConversationRoleStatus,
				domain.FragmentConversationTypeImagePlan,
				strings.Join(lines, "\n"), taskID, taskID+":image_plan")
		}
	}
}

func fragmentScenePlanDisplayPageIndex(zeroBasedIndex int, scene domain.FragmentScenePlan) int {
	if scene.Index > 0 {
		return scene.Index
	}
	return zeroBasedIndex + 1
}

func (s *FragmentGenerationService) recordFragmentGenerationFailure(
	ctx context.Context,
	fragmentID, userID, taskID, reason string,
) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}
	s.recordFragmentConversationTurn(ctx, fragmentID, userID,
		domain.FragmentConversationRoleStatus,
		domain.FragmentConversationTypeError,
		reason, taskID, taskID+":error")
}
