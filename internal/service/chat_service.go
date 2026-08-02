package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ChatService provides minimal character / DM chat.
type ChatService struct {
	repo *mysql.ChatRepository
	svc  *Service
	log  *zap.Logger
}

func NewChatService(db *gorm.DB, svc *Service, log *zap.Logger) *ChatService {
	return &ChatService{
		repo: mysql.NewChatRepository(db),
		svc:  svc,
		log:  log,
	}
}

func (c *ChatService) ListSessions(ctx context.Context, userID string, limit, offset int) ([]*domain.ChatSession, int64, error) {
	return c.repo.ListSessions(ctx, userID, limit, offset)
}

func (c *ChatService) GetSession(ctx context.Context, userID, sessionID string) (*domain.ChatSession, error) {
	s, err := c.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}
	return s, nil
}

type StartChatRequest struct {
	CharacterID string `json:"characterId"`
	PeerUserID  string `json:"peerUserId"`
}

func (c *ChatService) StartSession(ctx context.Context, userID string, req StartChatRequest) (*domain.ChatSession, error) {
	characterID := strings.TrimSpace(req.CharacterID)
	peerUserID := strings.TrimSpace(req.PeerUserID)
	if characterID == "" && peerUserID == "" {
		return nil, fmt.Errorf("characterId or peerUserId is required")
	}
	if characterID != "" && peerUserID != "" {
		return nil, fmt.Errorf("provide only one of characterId or peerUserId")
	}

	if characterID != "" {
		if existing, err := c.repo.FindCharacterSession(ctx, userID, characterID); err == nil {
			return existing, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		ch, err := c.svc.GetCharacter(ctx, characterID)
		if err != nil || ch == nil {
			return nil, fmt.Errorf("character not found")
		}
		title := ch.Name
		avatar := ch.Avatar
		if avatar == "" {
			avatar = ch.Portrait
		}
		s := &domain.ChatSession{
			OwnerUserID:  userID,
			SessionType:  domain.ChatSessionTypeCharacter,
			CharacterID:  characterID,
			Title:        title,
			Avatar:       avatar,
			CreatedAt:    time.Now().UnixMilli(),
		}
		if err := c.repo.CreateSession(ctx, s); err != nil {
			return nil, err
		}
		s.CharacterName = title
		s.CharacterAvatar = avatar
		return s, nil
	}

	if peerUserID == userID {
		return nil, fmt.Errorf("cannot start a chat with yourself")
	}
	if existing, err := c.repo.FindDirectSession(ctx, userID, peerUserID); err == nil {
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	peer, err := c.svc.GetUser(ctx, peerUserID)
	if err != nil || peer == nil {
		return nil, fmt.Errorf("user not found")
	}
	title := peer.DisplayName
	if title == "" {
		title = peer.Username
	}
	s := &domain.ChatSession{
		OwnerUserID: userID,
		SessionType: domain.ChatSessionTypeDirect,
		PeerUserID:  peerUserID,
		Title:       title,
		Avatar:      peer.Avatar,
		CreatedAt:   time.Now().UnixMilli(),
	}
	if err := c.repo.CreateSession(ctx, s); err != nil {
		return nil, err
	}
	s.CharacterName = title
	s.CharacterAvatar = peer.Avatar
	return s, nil
}

func (c *ChatService) ListMessages(ctx context.Context, userID, sessionID string, before int64, limit int) ([]*domain.ChatMessage, error) {
	if _, err := c.repo.GetSession(ctx, sessionID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}
	return c.repo.ListMessages(ctx, sessionID, before, limit)
}

type SendChatMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type SendChatMessageResult struct {
	UserMessage      *domain.ChatMessage `json:"userMessage"`
	AssistantMessage *domain.ChatMessage `json:"assistantMessage,omitempty"`
}

func (c *ChatService) SendMessage(ctx context.Context, userID, sessionID, content string) (*SendChatMessageResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	session, err := c.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}

	userMsg := &domain.ChatMessage{
		SessionID: sessionID,
		SenderID:  userID,
		Role:      "user",
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
		Status:    "sent",
	}
	if err := c.repo.CreateMessage(ctx, userMsg); err != nil {
		return nil, err
	}

	result := &SendChatMessageResult{UserMessage: userMsg}

	// Character sessions get a lightweight assistant reply (no LLM dependency for MVP).
	if session.SessionType == domain.ChatSessionTypeCharacter {
		name := session.Title
		if name == "" {
			name = "Character"
		}
		reply := fmt.Sprintf("(%s) I heard you: %s", name, truncateChatText(content, 120))
		assistantMsg := &domain.ChatMessage{
			SessionID: sessionID,
			SenderID:  session.CharacterID,
			Role:      "assistant",
			Content:   reply,
			Timestamp: time.Now().UnixMilli(),
			Status:    "sent",
		}
		if err := c.repo.CreateMessage(ctx, assistantMsg); err != nil {
			c.log.Warn("failed to persist assistant chat reply", zap.Error(err))
		} else {
			result.AssistantMessage = assistantMsg
		}
	}

	return result, nil
}

func truncateChatText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
