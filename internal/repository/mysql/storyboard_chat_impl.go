package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// StoryboardChatSession database model
type StoryboardChatSession struct {
	ID                 string         `gorm:"primaryKey;size:36"`
	UserID             string         `gorm:"size:36;not null;index"`
	User               User           `gorm:"foreignKey:UserID"`
	StoryID            string         `gorm:"size:36;not null;index"`
	Story              Story          `gorm:"foreignKey:StoryID"`
	StoryboardID       *string        `gorm:"size:36;index"`
	Storyboard         *Storyboard    `gorm:"foreignKey:StoryboardID"`
	CurrentStep        int            `gorm:"default:1"`
	Status             string         `gorm:"size:20;default:'active';index"`
	SelectedCharacters string         `gorm:"type:json"` // JSON array of character IDs
	SelectedStyle      string         `gorm:"size:100"`
	CreatedAt          time.Time      `gorm:"autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"autoUpdateTime"`
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

func (StoryboardChatSession) TableName() string {
	return "storyboard_chat_sessions"
}

// StoryboardChatMessage database model
type StoryboardChatMessage struct {
	ID          string         `gorm:"primaryKey;size:36"`
	SessionID   string         `gorm:"size:36;not null;index"`
	Session     StoryboardChatSession `gorm:"foreignKey:SessionID"`
	MessageType string         `gorm:"size:50;not null"`
	Status      string         `gorm:"size:20;not null"`
	Step        int            `gorm:"not null"`
	Data        string         `gorm:"type:json"`
	Actions     string         `gorm:"type:json"`
	Content     string         `gorm:"type:text"`
	IsUser      bool           `gorm:"default:false"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (StoryboardChatMessage) TableName() string {
	return "storyboard_chat_messages"
}

// ========== StoryboardChatSession Methods ==========

func (r *Repository) CreateStoryboardChatSession(ctx context.Context, session *domain.StoryboardChatSession) error {
	selectedCharactersJSON := "[]"
	if len(session.SelectedCharacters) > 0 {
		jsonBytes, err := json.Marshal(session.SelectedCharacters)
		if err != nil {
			return fmt.Errorf("failed to marshal selected characters: %w", err)
		}
		selectedCharactersJSON = string(jsonBytes)
	}

	var storyboardID *string
	if session.StoryboardID != "" {
		storyboardID = &session.StoryboardID
	}

	dbSession := StoryboardChatSession{
		ID:                 uuid.New().String(),
		UserID:             session.UserID,
		StoryID:            session.StoryID,
		StoryboardID:       storyboardID,
		CurrentStep:        session.CurrentStep,
		Status:             session.Status,
		SelectedCharacters: selectedCharactersJSON,
		SelectedStyle:      session.SelectedStyle,
	}

	if err := r.db.WithContext(ctx).Create(&dbSession).Error; err != nil {
		return fmt.Errorf("failed to create storyboard chat session: %w", err)
	}

	session.ID = dbSession.ID
	session.CreatedAt = dbSession.CreatedAt.Unix()
	session.UpdatedAt = dbSession.UpdatedAt.Unix()
	return nil
}

func (r *Repository) GetStoryboardChatSession(ctx context.Context, id string) (*domain.StoryboardChatSession, error) {
	var session StoryboardChatSession
	if err := r.db.WithContext(ctx).
		Preload("Story").
		Preload("Storyboard").
		First(&session, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to get storyboard chat session: %w", err)
	}
	return r.storyboardChatSessionToDomain(session), nil
}

func (r *Repository) GetActiveStoryboardChatSession(ctx context.Context, userID, storyID string) (*domain.StoryboardChatSession, error) {
	var session StoryboardChatSession
	if err := r.db.WithContext(ctx).
		Preload("Story").
		Preload("Storyboard").
		Where("user_id = ? AND story_id = ? AND status = ?", userID, storyID, domain.SessionStatusActive).
		Order("created_at DESC").
		First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active storyboard chat session: %w", err)
	}
	return r.storyboardChatSessionToDomain(session), nil
}

func (r *Repository) ListStoryboardChatSessions(ctx context.Context, userID string, limit, offset int) ([]*domain.StoryboardChatSession, error) {
	var sessions []StoryboardChatSession
	if err := r.db.WithContext(ctx).
		Preload("Story").
		Preload("Storyboard").
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to list storyboard chat sessions: %w", err)
	}

	result := make([]*domain.StoryboardChatSession, len(sessions))
	for i, s := range sessions {
		result[i] = r.storyboardChatSessionToDomain(s)
	}
	return result, nil
}

func (r *Repository) UpdateStoryboardChatSession(ctx context.Context, session *domain.StoryboardChatSession) error {
	updates := map[string]interface{}{
		"current_step": session.CurrentStep,
		"status":       session.Status,
		"updated_at":   time.Now(),
	}

	if session.StoryboardID != "" {
		updates["storyboard_id"] = session.StoryboardID
	}

	if session.SelectedStyle != "" {
		updates["selected_style"] = session.SelectedStyle
	}

	if len(session.SelectedCharacters) > 0 {
		jsonBytes, err := json.Marshal(session.SelectedCharacters)
		if err != nil {
			return fmt.Errorf("failed to marshal selected characters: %w", err)
		}
		updates["selected_characters"] = string(jsonBytes)
	}

	if err := r.db.WithContext(ctx).Model(&StoryboardChatSession{}).
		Where("id = ?", session.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update storyboard chat session: %w", err)
	}
	return nil
}

func (r *Repository) DeleteStoryboardChatSession(ctx context.Context, id string) error {
	// First delete all messages in the session
	if err := r.db.WithContext(ctx).Where("session_id = ?", id).Delete(&StoryboardChatMessage{}).Error; err != nil {
		return fmt.Errorf("failed to delete storyboard chat messages: %w", err)
	}

	// Then delete the session
	if err := r.db.WithContext(ctx).Delete(&StoryboardChatSession{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete storyboard chat session: %w", err)
	}
	return nil
}

// ========== StoryboardChatMessage Methods ==========

func (r *Repository) CreateStoryboardChatMessage(ctx context.Context, msg *domain.StoryboardChatMessage) error {
	dataJSON := "{}"
	if len(msg.Data) > 0 {
		dataJSON = string(msg.Data)
	}

	actionsJSON := "[]"
	if len(msg.Actions) > 0 {
		jsonBytes, err := json.Marshal(msg.Actions)
		if err != nil {
			return fmt.Errorf("failed to marshal actions: %w", err)
		}
		actionsJSON = string(jsonBytes)
	}

	dbMsg := StoryboardChatMessage{
		ID:          uuid.New().String(),
		SessionID:   msg.SessionID,
		MessageType: msg.MessageType,
		Status:      msg.Status,
		Step:        msg.Step,
		Data:        dataJSON,
		Actions:     actionsJSON,
		Content:     msg.Content,
		IsUser:      msg.IsUser,
	}

	if err := r.db.WithContext(ctx).Create(&dbMsg).Error; err != nil {
		return fmt.Errorf("failed to create storyboard chat message: %w", err)
	}

	msg.ID = dbMsg.ID
	msg.Timestamp = dbMsg.CreatedAt.Unix()
	return nil
}

func (r *Repository) GetStoryboardChatMessage(ctx context.Context, id string) (*domain.StoryboardChatMessage, error) {
	var msg StoryboardChatMessage
	if err := r.db.WithContext(ctx).First(&msg, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to get storyboard chat message: %w", err)
	}
	return r.storyboardChatMessageToDomain(msg), nil
}

func (r *Repository) ListStoryboardChatMessages(ctx context.Context, sessionID string, limit, offset int) ([]*domain.StoryboardChatMessage, error) {
	var messages []StoryboardChatMessage
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to list storyboard chat messages: %w", err)
	}

	result := make([]*domain.StoryboardChatMessage, len(messages))
	for i, m := range messages {
		result[i] = r.storyboardChatMessageToDomain(m)
	}
	return result, nil
}

func (r *Repository) GetLastStoryboardChatMessage(ctx context.Context, sessionID string) (*domain.StoryboardChatMessage, error) {
	var msg StoryboardChatMessage
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		First(&msg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last storyboard chat message: %w", err)
	}
	return r.storyboardChatMessageToDomain(msg), nil
}

func (r *Repository) DeleteStoryboardChatMessage(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&StoryboardChatMessage{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete storyboard chat message: %w", err)
	}
	return nil
}

// ========== Converters ==========

func (r *Repository) storyboardChatSessionToDomain(s StoryboardChatSession) *domain.StoryboardChatSession {
	session := &domain.StoryboardChatSession{
		ID:            s.ID,
		UserID:        s.UserID,
		StoryID:       s.StoryID,
		CurrentStep:   s.CurrentStep,
		Status:        s.Status,
		SelectedStyle: s.SelectedStyle,
		CreatedAt:     s.CreatedAt.Unix(),
		UpdatedAt:     s.UpdatedAt.Unix(),
	}

	if s.StoryboardID != nil {
		session.StoryboardID = *s.StoryboardID
	}

	// Parse selected characters
	if s.SelectedCharacters != "" && s.SelectedCharacters != "[]" {
		var characters []string
		if err := json.Unmarshal([]byte(s.SelectedCharacters), &characters); err == nil {
			session.SelectedCharacters = characters
		}
	}

	// Convert story if loaded
	if s.Story.ID != "" {
		domainStory := r.storyToDomain(s.Story)
		session.Story = &domainStory
	}

	// Convert storyboard if loaded (without context, we skip the full conversion)
	if s.Storyboard != nil && s.Storyboard.ID != "" {
		// Simple conversion without child nodes for session context
		parentID := ""
		if s.Storyboard.ParentID != nil {
			parentID = *s.Storyboard.ParentID
		}
		session.Storyboard = &domain.Storyboard{
			ID:             s.Storyboard.ID,
			StoryID:        s.Storyboard.StoryID,
			ParentID:       parentID,
			CreatorID:      s.Storyboard.CreatorID,
			Title:          s.Storyboard.Title,
			Content:        s.Storyboard.Content,
			WorkflowStatus: s.Storyboard.WorkflowStatus,
			CurrentStep:    s.Storyboard.CurrentStep,
		}
	}

	return session
}

func (r *Repository) storyboardChatMessageToDomain(m StoryboardChatMessage) *domain.StoryboardChatMessage {
	msg := &domain.StoryboardChatMessage{
		ID:          m.ID,
		SessionID:   m.SessionID,
		MessageType: m.MessageType,
		Status:      m.Status,
		Step:        m.Step,
		Content:     m.Content,
		IsUser:      m.IsUser,
		Timestamp:   m.CreatedAt.Unix(),
	}

	// Parse data JSON
	if m.Data != "" && m.Data != "{}" {
		msg.Data = json.RawMessage(m.Data)
	}

	// Parse actions JSON
	if m.Actions != "" && m.Actions != "[]" {
		var actions []domain.ChatAction
		if err := json.Unmarshal([]byte(m.Actions), &actions); err == nil {
			msg.Actions = actions
		}
	}

	return msg
}

