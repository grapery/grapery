package mysql

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CreateAccountDeletionBlock inserts a cooldown record for email and/or phone identifiers.
func (r *Repository) CreateAccountDeletionBlock(ctx context.Context, userID, emailNorm, phoneNorm string, blockedUntil int64) error {
	row := &AccountDeletionBlock{
		ID:           uuid.New().String(),
		UserID:       userID,
		EmailNorm:    emailNorm,
		PhoneNorm:    phoneNorm,
		BlockedUntil: blockedUntil,
	}
	return r.db.WithContext(ctx).Create(row).Error
}

// IsAccountReRegistrationBlocked is true when emailNorm or phoneNorm (if non-empty) is still within a deletion cooldown.
func (r *Repository) IsAccountReRegistrationBlocked(ctx context.Context, emailNorm, phoneNorm string) (bool, error) {
	now := time.Now().Unix()
	q := r.db.WithContext(ctx).Model(&AccountDeletionBlock{}).Where("blocked_until > ?", now)
	if emailNorm != "" && phoneNorm != "" {
		q = q.Where("(email_norm = ? OR phone_norm = ?)", emailNorm, phoneNorm)
	} else if emailNorm != "" {
		q = q.Where("email_norm = ?", emailNorm)
	} else if phoneNorm != "" {
		q = q.Where("phone_norm = ?", phoneNorm)
	} else {
		return false, nil
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
