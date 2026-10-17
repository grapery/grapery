package mysql

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"strings"
)

func (r *Repository) ApplyGenerationCharge(ctx context.Context, userID, operationID, relatedID, source string, tokens int) (*domain.TokenTransaction, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(operationID) == "" || tokens <= 0 {
		return nil, fmt.Errorf("invalid generation charge")
	}
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(userID+":"+source+":"+operationID)).String()
	return r.updateTokenBalance(ctx, userID, -tokens, source, "AI generation settlement", id, relatedID)
}
func (r *Repository) GenerationChargedTokens(ctx context.Context, userID, relatedID string) (int, error) {
	var total int
	err := r.db.WithContext(ctx).Model(&TokenTransaction{}).Where("user_id = ? AND related_id = ?", userID, relatedID).Select("COALESCE(-SUM(amount), 0)").Scan(&total).Error
	return total, err
}

func (r *Repository) ClaimFailedGenerationImage(ctx context.Context, recordID string) (bool, error) {
	result := r.db.WithContext(ctx).Model(&AIGenerationRecord{}).Where("id = ? AND status = ?", recordID, domain.AITaskStatusFailed).Updates(map[string]interface{}{"status": domain.AITaskStatusProcessing})
	return result.RowsAffected == 1, result.Error
}
