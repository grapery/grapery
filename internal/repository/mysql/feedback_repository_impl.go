package mysql

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// FeedbackRepositoryImpl implements domain.FeedbackRepository
type FeedbackRepositoryImpl struct {
	db *gorm.DB
}

// NewFeedbackRepository creates a feedback repository
func NewFeedbackRepository(db *gorm.DB) domain.FeedbackRepository {
	return &FeedbackRepositoryImpl{db: db}
}

func feedbackModelToDomain(m *UserFeedback) *domain.UserFeedback {
	if m == nil {
		return nil
	}
	return &domain.UserFeedback{
		ID:          m.ID,
		UserID:      m.UserID,
		Category:    m.Category,
		Content:     m.Content,
		ContactInfo: m.ContactInfo,
		Status:      m.Status,
		Response:    m.Response,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// CreateFeedback persists a feedback row
func (r *FeedbackRepositoryImpl) CreateFeedback(ctx context.Context, fb *domain.UserFeedback) error {
	if fb == nil {
		return fmt.Errorf("feedback is nil")
	}
	m := &UserFeedback{
		ID:          fb.ID,
		UserID:      fb.UserID,
		Category:    fb.Category,
		Content:     fb.Content,
		ContactInfo: fb.ContactInfo,
		Status:      fb.Status,
		Response:    fb.Response,
	}
	if m.Status == "" {
		m.Status = "received"
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	fb.CreatedAt = m.CreatedAt
	fb.UpdatedAt = m.UpdatedAt
	return nil
}

// ListFeedbackByUserID returns feedback for a user with total count
func (r *FeedbackRepositoryImpl) ListFeedbackByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.UserFeedback, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := r.db.WithContext(ctx).Model(&UserFeedback{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []UserFeedback
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*domain.UserFeedback, 0, len(rows))
	for i := range rows {
		out = append(out, feedbackModelToDomain(&rows[i]))
	}
	return out, total, nil
}
