package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	emailpkg "github.com/grapestree/fgrapery/grapery/internal/email"
	"go.uber.org/zap"
)

const accountDeletionCooldownSec = int64(30 * 24 * 3600)

func normalizeEmailForDeletionBlock(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizePhoneForDeletionBlock(raw string) string {
	if raw == "" {
		return ""
	}
	if p, err := NormalizeChinaPhone(raw); err == nil {
		return p
	}
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// EnsureActiveSessionUser returns an error if the JWT subject is missing, unknown, or not an active account.
func (s *Service) EnsureActiveSessionUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("not authenticated")
	}
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil || u.Status != string(common.StatusActive) {
		return errors.New("account no longer available")
	}
	return nil
}

// DeleteMyAccount anonymizes the user, soft-deletes the row, removes OAuth bindings and devices,
// and records a cooldown so the same email/phone cannot re-register or re-bind for 30 days.
func (s *Service) DeleteMyAccount(ctx context.Context, userID string) error {
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return domain.ErrNotFound
	}
	if u.Status != string(common.StatusActive) {
		return fmt.Errorf("account is %s", u.Status)
	}

	emailNorm := normalizeEmailForDeletionBlock(u.Email)
	phoneNorm := normalizePhoneForDeletionBlock(u.Phone)
	origDisplay := u.DisplayName
	mailTo := strings.TrimSpace(u.Email)

	deadHash, err := auth.HashPassword(uuid.New().String() + uuid.New().String())
	if err != nil {
		return err
	}

	blockedUntil := time.Now().Unix() + accountDeletionCooldownSec

	err = s.repo.WithTransaction(ctx, func(tx domain.Repository) error {
		if err := tx.CreateAccountDeletionBlock(ctx, userID, emailNorm, phoneNorm, blockedUntil); err != nil {
			return err
		}

		logins, err := tx.GetThirdPartyLoginsByUserID(ctx, userID)
		if err != nil {
			return err
		}
		for _, lp := range logins {
			if err := tx.DeleteThirdPartyLogin(ctx, lp.ID); err != nil {
				return err
			}
		}

		devices, err := tx.GetUserDevicesByUserID(ctx, userID)
		if err != nil {
			return err
		}
		for _, d := range devices {
			if err := tx.DeleteUserDevice(ctx, d.ID); err != nil {
				return err
			}
		}

		anonEmail := fmt.Sprintf("del+%s@del.invalid", userID)
		anonUser := "del_" + strings.ReplaceAll(userID, "-", "")
		if len(anonUser) > 50 {
			anonUser = anonUser[:50]
		}
		referral := "d" + strings.ReplaceAll(uuid.New().String(), "-", "")[:19]

		u.Email = anonEmail
		u.Username = anonUser
		u.PasswordHash = deadHash
		u.DisplayName = "Deleted User"
		u.Avatar = ""
		u.Background = ""
		u.Bio = ""
		u.Location = ""
		u.Website = ""
		u.AIPromptPreferences = ""
		u.Phone = ""
		u.PhoneVerifiedAt = nil
		u.PendingOAuthPhoneSMS = false
		u.EmailVerified = false
		u.ReferralCode = referral
		u.Status = string(common.StatusDeleted)
		u.UpdatedAt = time.Now().Unix()

		if err := tx.UpdateUser(ctx, u); err != nil {
			return err
		}
		if err := tx.DeleteUser(ctx, userID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.invalidateUserCache(ctx, userID)

	if mailTo != "" && strings.Contains(mailTo, "@") && !strings.HasPrefix(strings.ToLower(mailTo), "del+") {
		go func(addr, name string) {
			if err := emailpkg.AccountDeletedEmail([]string{addr}, name); err != nil {
				s.logger.Warn("account deleted confirmation email failed", zap.Error(err))
			}
		}(mailTo, origDisplay)
	}

	s.logger.Info("user account deleted", zap.String("userID", userID))
	return nil
}
