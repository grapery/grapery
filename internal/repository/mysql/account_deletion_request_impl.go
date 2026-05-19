package mysql

import (
	"context"
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

func adrModelToDomain(m *AccountDeletionRequest) *domain.AccountDeletionRequestRow {
	if m == nil {
		return nil
	}
	return &domain.AccountDeletionRequestRow{
		ID:                  m.ID,
		UserID:              m.UserID,
		Status:              m.Status,
		ScheduledDeletionAt: m.ScheduledDeletionAt,
		Reason:              m.Reason,
		Feedback:            m.Feedback,
		RequestedAt:         m.RequestedAt,
		ProcessedAt:         m.ProcessedAt,
		CancelledAt:         m.CancelledAt,
		CancelledReason:     m.CancelledReason,
	}
}

// CreateAccountDeletionRequest persists a deletion request row.
func (r *Repository) CreateAccountDeletionRequest(ctx context.Context, req *domain.AccountDeletionRequestRow) error {
	if req == nil || req.ID == "" || req.UserID == "" {
		return domain.ErrInvalidInput
	}
	row := &AccountDeletionRequest{
		ID:                  req.ID,
		UserID:              req.UserID,
		Status:              req.Status,
		ScheduledDeletionAt: req.ScheduledDeletionAt,
		Reason:              req.Reason,
		Feedback:            req.Feedback,
		RequestedAt:         req.RequestedAt,
		ProcessedAt:         req.ProcessedAt,
		CancelledAt:         req.CancelledAt,
		CancelledReason:     req.CancelledReason,
	}
	return r.db.WithContext(ctx).Create(row).Error
}

// GetActiveAccountDeletionRequestByUser returns the latest pending/processing deletion request for a user.
func (r *Repository) GetActiveAccountDeletionRequestByUser(ctx context.Context, userID string) (*domain.AccountDeletionRequestRow, error) {
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}
	var m AccountDeletionRequest
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []string{
			string(common.DeletionStatusPending),
			string(common.DeletionStatusProcessing),
		}).
		Order("requested_at DESC").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return adrModelToDomain(&m), nil
}

// ListDueAccountDeletionRequests lists pending rows whose grace period elapsed (scheduled_deletion_at <= now).
func (r *Repository) ListDueAccountDeletionRequests(ctx context.Context, nowUnix int64, limit int) ([]*domain.AccountDeletionRequestRow, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []AccountDeletionRequest
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_deletion_at <= ?", string(common.DeletionStatusPending), nowUnix).
		Order("scheduled_deletion_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.AccountDeletionRequestRow, 0, len(rows))
	for i := range rows {
		out = append(out, adrModelToDomain(&rows[i]))
	}
	return out, nil
}

// UpdateAccountDeletionRequestStatus updates status and optional timestamps.
func (r *Repository) UpdateAccountDeletionRequestStatus(ctx context.Context, id, status string, processedAt *int64, cancelledAt *int64, cancelledReason string) error {
	if id == "" || status == "" {
		return domain.ErrInvalidInput
	}
	updates := map[string]interface{}{
		"status": status,
	}
	if processedAt != nil {
		updates["processed_at"] = *processedAt
	}
	if cancelledAt != nil {
		updates["cancelled_at"] = *cancelledAt
	}
	if cancelledReason != "" {
		updates["cancelled_reason"] = cancelledReason
	}
	return r.db.WithContext(ctx).Model(&AccountDeletionRequest{}).Where("id = ?", id).Updates(updates).Error
}

// ClaimPendingDueAccountDeletionRequest atomically marks a due pending deletion request as processing.
func (r *Repository) ClaimPendingDueAccountDeletionRequest(ctx context.Context, requestID string, nowUnix int64) (bool, error) {
	if requestID == "" {
		return false, domain.ErrInvalidInput
	}
	res := r.db.WithContext(ctx).Model(&AccountDeletionRequest{}).
		Where("id = ? AND status = ? AND scheduled_deletion_at <= ?", requestID,
			string(common.DeletionStatusPending), nowUnix).
		Updates(map[string]interface{}{
			"status":       string(common.DeletionStatusProcessing),
			"processed_at": nowUnix,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
