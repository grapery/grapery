package mysql

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ResetAccountDeletionRequestToPending moves a deletion request row back to pending and clears processed_at.
func (r *Repository) ResetAccountDeletionRequestToPending(ctx context.Context, requestID string) error {
	if requestID == "" {
		return domain.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Model(&AccountDeletionRequest{}).Where("id = ?", requestID).Updates(map[string]interface{}{
		"status":       string(common.DeletionStatusPending),
		"processed_at": nil,
	}).Error
}
