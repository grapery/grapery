package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ChatSessionDB persists chat_sessions.
type ChatSessionDB struct {
	ID            string `gorm:"primaryKey;type:varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"`
	OwnerUserID   string `gorm:"column:owner_user_id;type:varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;index;not null"`
	SessionType   string `gorm:"column:session_type;size:20;index;not null"` // character | direct
	CharacterID   string `gorm:"column:character_id;type:varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;index"`
	PeerUserID    string `gorm:"column:peer_user_id;type:varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;index"`
	Title         string `gorm:"size:200"`
	Avatar        string `gorm:"size:500"`
	LastMessage   string `gorm:"type:text"`
	LastMessageAt int64  `gorm:"column:last_message_at;type:bigint;index"`
	UnreadCount   int    `gorm:"column:unread_count;default:0"`
	CreatedAt     int64  `gorm:"type:bigint;index"`
	UpdatedAt     int64  `gorm:"type:bigint"`
}

func (ChatSessionDB) TableName() string { return "chat_sessions" }

// ChatMessageDB persists chat_messages.
type ChatMessageDB struct {
	ID        string `gorm:"primaryKey;type:varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"`
	SessionID string `gorm:"column:session_id;type:varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;index;not null"`
	SenderID  string `gorm:"column:sender_id;type:varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;index"`
	Role      string `gorm:"size:20;not null"`
	Content   string `gorm:"type:text;not null"`
	CreatedAt int64  `gorm:"type:bigint;index"`
}

func (ChatMessageDB) TableName() string { return "chat_messages" }

// ChatRepository provides chat persistence without expanding domain.Repository.
type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) ListSessions(ctx context.Context, ownerUserID string, limit, offset int) ([]*domain.ChatSession, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	var total int64
	q := r.db.WithContext(ctx).Model(&ChatSessionDB{}).Where("owner_user_id = ?", ownerUserID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ChatSessionDB
	if err := q.Order("last_message_at DESC, updated_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*domain.ChatSession, 0, len(rows))
	for i := range rows {
		out = append(out, sessionFromDB(&rows[i]))
	}
	return out, total, nil
}

func (r *ChatRepository) GetSession(ctx context.Context, sessionID, ownerUserID string) (*domain.ChatSession, error) {
	var row ChatSessionDB
	err := r.db.WithContext(ctx).Where("id = ? AND owner_user_id = ?", sessionID, ownerUserID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return sessionFromDB(&row), nil
}

func (r *ChatRepository) FindCharacterSession(ctx context.Context, ownerUserID, characterID string) (*domain.ChatSession, error) {
	var row ChatSessionDB
	err := r.db.WithContext(ctx).
		Where("owner_user_id = ? AND session_type = ? AND character_id = ?", ownerUserID, domain.ChatSessionTypeCharacter, characterID).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return sessionFromDB(&row), nil
}

func (r *ChatRepository) FindDirectSession(ctx context.Context, ownerUserID, peerUserID string) (*domain.ChatSession, error) {
	var row ChatSessionDB
	err := r.db.WithContext(ctx).
		Where("owner_user_id = ? AND session_type = ? AND peer_user_id = ?", ownerUserID, domain.ChatSessionTypeDirect, peerUserID).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return sessionFromDB(&row), nil
}

func (r *ChatRepository) CreateSession(ctx context.Context, s *domain.ChatSession) error {
	now := time.Now().UnixMilli()
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.CreatedAt == 0 {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	row := ChatSessionDB{
		ID:            s.ID,
		OwnerUserID:   s.OwnerUserID,
		SessionType:   s.SessionType,
		CharacterID:   s.CharacterID,
		PeerUserID:    s.PeerUserID,
		Title:         s.Title,
		Avatar:        s.Avatar,
		LastMessage:   s.LastMessage,
		LastMessageAt: s.LastMessageAt,
		UnreadCount:   s.UnreadCount,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *ChatRepository) UpdateSessionMeta(ctx context.Context, sessionID, lastMessage string, lastMessageAt int64) error {
	return r.db.WithContext(ctx).Model(&ChatSessionDB{}).Where("id = ?", sessionID).Updates(map[string]interface{}{
		"last_message":    lastMessage,
		"last_message_at": lastMessageAt,
		"updated_at":      time.Now().UnixMilli(),
	}).Error
}

func (r *ChatRepository) ListMessages(ctx context.Context, sessionID string, before int64, limit int) ([]*domain.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	q := r.db.WithContext(ctx).Model(&ChatMessageDB{}).Where("session_id = ?", sessionID)
	if before > 0 {
		q = q.Where("created_at < ?", before)
	}
	var rows []ChatMessageDB
	if err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.ChatMessage, 0, len(rows))
	for i := range rows {
		out = append(out, messageFromDB(&rows[i]))
	}
	return out, nil
}

func (r *ChatRepository) CreateMessage(ctx context.Context, m *domain.ChatMessage) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.Timestamp == 0 {
		m.Timestamp = time.Now().UnixMilli()
	}
	if m.Status == "" {
		m.Status = "sent"
	}
	row := ChatMessageDB{
		ID:        m.ID,
		SessionID: m.SessionID,
		SenderID:  m.SenderID,
		Role:      m.Role,
		Content:   m.Content,
		CreatedAt: m.Timestamp,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	return r.UpdateSessionMeta(ctx, m.SessionID, truncate(m.Content, 200), m.Timestamp)
}

func sessionFromDB(row *ChatSessionDB) *domain.ChatSession {
	s := &domain.ChatSession{
		ID:              row.ID,
		OwnerUserID:     row.OwnerUserID,
		SessionType:     row.SessionType,
		CharacterID:     row.CharacterID,
		PeerUserID:      row.PeerUserID,
		Title:           row.Title,
		Avatar:          row.Avatar,
		LastMessage:     row.LastMessage,
		LastMessageAt:   row.LastMessageAt,
		UnreadCount:     row.UnreadCount,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		CharacterName:   row.Title,
		CharacterAvatar: row.Avatar,
		LastMessageTime: row.LastMessageAt,
	}
	return s
}

func messageFromDB(row *ChatMessageDB) *domain.ChatMessage {
	return &domain.ChatMessage{
		ID:        row.ID,
		SessionID: row.SessionID,
		SenderID:  row.SenderID,
		Role:      row.Role,
		Content:   row.Content,
		Timestamp: row.CreatedAt,
		Status:    "sent",
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// EnsureChatTables is used by migrations.
func EnsureChatTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}
	return autoMigrateIgnoringDuplicatedStoriesSourceFragmentIndex(db, nil, &ChatSessionDB{}, &ChatMessageDB{})
}
