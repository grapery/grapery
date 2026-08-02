package mysql

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

func (r *Repository) AppendStoryboardConversationMessage(ctx context.Context, msg *domain.StoryboardConversationMessage) error {
	if r == nil || r.db == nil || msg == nil {
		return nil
	}
	storyboardID := strings.TrimSpace(msg.StoryboardID)
	if storyboardID == "" {
		return nil
	}
	clientID := strings.TrimSpace(msg.ClientMessageID)
	if clientID != "" {
		var count int64
		if err := r.db.WithContext(ctx).Model(&StoryboardConversationMessageDB{}).
			Where("storyboard_id = ? AND client_message_id = ?", storyboardID, clientID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}
	id := strings.TrimSpace(msg.ID)
	if id == "" {
		id = uuid.New().String()
	}
	var maxSeq int
	if err := r.db.WithContext(ctx).Model(&StoryboardConversationMessageDB{}).
		Where("storyboard_id = ?", storyboardID).
		Select("COALESCE(MAX(sequence), 0)").
		Scan(&maxSeq).Error; err != nil {
		return err
	}
	row := &StoryboardConversationMessageDB{
		ID:              id,
		StoryboardID:    storyboardID,
		UserID:          strings.TrimSpace(msg.UserID),
		Role:            strings.TrimSpace(msg.Role),
		MessageType:     strings.TrimSpace(msg.MessageType),
		Text:            strings.TrimSpace(msg.Text),
		TaskID:          strings.TrimSpace(msg.TaskID),
		ClientMessageID: clientID,
		Sequence:        maxSeq + 1,
	}
	if msg.Sequence > 0 {
		row.Sequence = msg.Sequence
	}
	if msg.CreatedAt > 0 {
		row.CreatedAt = msg.CreatedAt
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpsertStoryboardConversationMessages(ctx context.Context, messages []*domain.StoryboardConversationMessage) error {
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		if err := r.AppendStoryboardConversationMessage(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

const (
	defaultStoryboardConversationPageSize = 50
	maxStoryboardConversationPageSize     = 200
)

func normalizeStoryboardConversationPageLimit(limit int) int {
	if limit <= 0 {
		return defaultStoryboardConversationPageSize
	}
	if limit > maxStoryboardConversationPageSize {
		return maxStoryboardConversationPageSize
	}
	return limit
}

func (r *Repository) ListStoryboardConversationMessages(ctx context.Context, storyboardID string) ([]*domain.StoryboardConversationMessage, error) {
	messages, _, err := r.ListStoryboardConversationMessagesPage(ctx, storyboardID, 0, 0)
	return messages, err
}

func (r *Repository) ListStoryboardConversationMessagesPage(ctx context.Context, storyboardID string, limit int, beforeCreatedAt int64) ([]*domain.StoryboardConversationMessage, bool, error) {
	if r == nil || r.db == nil {
		return nil, false, nil
	}
	storyboardID = strings.TrimSpace(storyboardID)
	if storyboardID == "" {
		return nil, false, nil
	}
	pageLimit := normalizeStoryboardConversationPageLimit(limit)
	fetchAll := limit <= 0 && beforeCreatedAt <= 0
	if fetchAll {
		var rows []StoryboardConversationMessageDB
		if err := r.db.WithContext(ctx).
			Where("storyboard_id = ?", storyboardID).
			Order("sequence ASC, created_at ASC").
			Find(&rows).Error; err != nil {
			return nil, false, err
		}
		out := make([]*domain.StoryboardConversationMessage, 0, len(rows))
		for i := range rows {
			out = append(out, domainStoryboardConversationMessageFromDB(&rows[i]))
		}
		return out, false, nil
	}

	q := r.db.WithContext(ctx).Where("storyboard_id = ?", storyboardID)
	if beforeCreatedAt > 0 {
		q = q.Where("created_at < ?", beforeCreatedAt)
	}
	var rows []StoryboardConversationMessageDB
	if err := q.Order("created_at DESC, sequence DESC").Limit(pageLimit + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > pageLimit
	if hasMore {
		rows = rows[:pageLimit]
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	out := make([]*domain.StoryboardConversationMessage, 0, len(rows))
	for i := range rows {
		out = append(out, domainStoryboardConversationMessageFromDB(&rows[i]))
	}
	return out, hasMore, nil
}

func domainStoryboardConversationMessageFromDB(row *StoryboardConversationMessageDB) *domain.StoryboardConversationMessage {
	if row == nil {
		return nil
	}
	return &domain.StoryboardConversationMessage{
		ID:              row.ID,
		StoryboardID:    row.StoryboardID,
		UserID:          row.UserID,
		Role:            row.Role,
		MessageType:     row.MessageType,
		Text:            row.Text,
		TaskID:          row.TaskID,
		ClientMessageID: row.ClientMessageID,
		Sequence:        row.Sequence,
		CreatedAt:       row.CreatedAt,
	}
}

func deleteStoryboardConversationMessages(ctx context.Context, db *gorm.DB, storyboardID string) error {
	if db == nil || strings.TrimSpace(storyboardID) == "" {
		return nil
	}
	return db.WithContext(ctx).Where("storyboard_id = ?", storyboardID).Delete(&StoryboardConversationMessageDB{}).Error
}
