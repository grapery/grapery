package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// FeedbackService handles user feedback submissions
type FeedbackService interface {
	SubmitFeedback(ctx context.Context, userID, category, content, contactInfo string) (*domain.UserFeedback, error)
	ListMyFeedback(ctx context.Context, userID string, limit, offset int) ([]*domain.UserFeedback, int64, error)
}

type feedbackService struct {
	repo   domain.FeedbackRepository
	logger *zap.Logger
}

// NewFeedbackService constructs FeedbackService
func NewFeedbackService(repo domain.FeedbackRepository, logger *zap.Logger) FeedbackService {
	return &feedbackService{repo: repo, logger: logger}
}

func (s *feedbackService) SubmitFeedback(ctx context.Context, userID, category, content, contactInfo string) (*domain.UserFeedback, error) {
	category = strings.TrimSpace(category)
	content = strings.TrimSpace(content)
	contactInfo = strings.TrimSpace(contactInfo)
	if category == "" || content == "" {
		return nil, fmt.Errorf("category and content are required")
	}
	if len(content) > 20000 {
		return nil, fmt.Errorf("content too long")
	}
	fb := &domain.UserFeedback{
		ID:          utils.GenerateID(),
		UserID:      userID,
		Category:    category,
		Content:     content,
		ContactInfo: contactInfo,
		Status:      "received",
	}
	if err := s.repo.CreateFeedback(ctx, fb); err != nil {
		s.logger.Error("create feedback failed", zap.Error(err), zap.String("userID", userID))
		return nil, fmt.Errorf("failed to save feedback: %w", err)
	}
	// Notification email is intentionally deferred; feedback is persisted only for now.
	s.logger.Info("feedback saved",
		zap.String("feedbackID", fb.ID),
		zap.String("userID", userID),
		zap.String("category", fb.Category),
	)
	return fb, nil
}

func (s *feedbackService) ListMyFeedback(ctx context.Context, userID string, limit, offset int) ([]*domain.UserFeedback, int64, error) {
	return s.repo.ListFeedbackByUserID(ctx, userID, limit, offset)
}
