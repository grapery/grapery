package mysql

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

func (r *Repository) AppendFragmentConversationMessage(ctx context.Context, msg *domain.FragmentConversationMessage) error {
	if r == nil || r.db == nil || msg == nil {
		return nil
	}
	fragmentID := strings.TrimSpace(msg.FragmentID)
	if fragmentID == "" {
		return nil
	}
	clientID := strings.TrimSpace(msg.ClientMessageID)
	if clientID != "" {
		var count int64
		if err := r.db.WithContext(ctx).Model(&FragmentConversationMessageDB{}).
			Where("fragment_id = ? AND client_message_id = ?", fragmentID, clientID).
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
	if err := r.db.WithContext(ctx).Model(&FragmentConversationMessageDB{}).
		Where("fragment_id = ?", fragmentID).
		Select("COALESCE(MAX(sequence), 0)").
		Scan(&maxSeq).Error; err != nil {
		return err
	}
	row := &FragmentConversationMessageDB{
		ID:              id,
		FragmentID:      fragmentID,
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

func (r *Repository) UpsertFragmentConversationMessages(ctx context.Context, messages []*domain.FragmentConversationMessage) error {
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		if err := r.AppendFragmentConversationMessage(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

const (
	defaultFragmentConversationPageSize = 50
	maxFragmentConversationPageSize     = 200
)

func normalizeFragmentConversationPageLimit(limit int) int {
	if limit <= 0 {
		return defaultFragmentConversationPageSize
	}
	if limit > maxFragmentConversationPageSize {
		return maxFragmentConversationPageSize
	}
	return limit
}

func (r *Repository) ListFragmentConversationMessages(ctx context.Context, fragmentID string) ([]*domain.FragmentConversationMessage, error) {
	messages, _, err := r.ListFragmentConversationMessagesPage(ctx, fragmentID, 0, 0)
	return messages, err
}

func (r *Repository) ListFragmentConversationMessagesPage(ctx context.Context, fragmentID string, limit int, beforeCreatedAt int64) ([]*domain.FragmentConversationMessage, bool, error) {
	if r == nil || r.db == nil {
		return nil, false, nil
	}
	fragmentID = strings.TrimSpace(fragmentID)
	if fragmentID == "" {
		return nil, false, nil
	}
	pageLimit := normalizeFragmentConversationPageLimit(limit)
	fetchAll := limit <= 0 && beforeCreatedAt <= 0
	if fetchAll {
		var rows []FragmentConversationMessageDB
		if err := r.db.WithContext(ctx).
			Where("fragment_id = ?", fragmentID).
			Order("sequence ASC, created_at ASC").
			Find(&rows).Error; err != nil {
			return nil, false, err
		}
		out := make([]*domain.FragmentConversationMessage, 0, len(rows))
		for i := range rows {
			out = append(out, domainFragmentConversationMessageFromDB(&rows[i]))
		}
		return out, false, nil
	}

	q := r.db.WithContext(ctx).Where("fragment_id = ?", fragmentID)
	if beforeCreatedAt > 0 {
		q = q.Where("created_at < ?", beforeCreatedAt)
	}
	var rows []FragmentConversationMessageDB
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
	out := make([]*domain.FragmentConversationMessage, 0, len(rows))
	for i := range rows {
		out = append(out, domainFragmentConversationMessageFromDB(&rows[i]))
	}
	return out, hasMore, nil
}

func domainFragmentConversationMessageFromDB(row *FragmentConversationMessageDB) *domain.FragmentConversationMessage {
	if row == nil {
		return nil
	}
	return &domain.FragmentConversationMessage{
		ID:              row.ID,
		FragmentID:      row.FragmentID,
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

func deleteFragmentConversationMessages(ctx context.Context, db *gorm.DB, fragmentID string) error {
	if db == nil || strings.TrimSpace(fragmentID) == "" {
		return nil
	}
	return db.WithContext(ctx).Where("fragment_id = ?", fragmentID).Delete(&FragmentConversationMessageDB{}).Error
}
