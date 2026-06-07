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
	"github.com/grapestree/fgrapery/grapery/internal/config"
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

func (s *Service) effectiveSystemAnonymousUserID() string {
	v := strings.TrimSpace(s.accountDeletionCfg.SystemAnonymousUserID)
	if v == "" {
		return config.DefaultSystemAnonymousUserID
	}
	return v
}

func (s *Service) effectiveAccountDeletionGraceSeconds() int64 {
	if s.accountDeletionCfg.GracePeriodSeconds <= 0 {
		return 7 * 24 * 3600
	}
	return int64(s.accountDeletionCfg.GracePeriodSeconds)
}

// SessionUser loads the persisted user profile (minimal helper for middleware).
func (s *Service) SessionUser(ctx context.Context, userID string) (*domain.User, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("not authenticated")
	}
	return s.GetUser(ctx, userID)
}

// EnsureActiveSessionUser allows active accounts and grace-period pending_deletion (for cancel/read paths).
func (s *Service) EnsureActiveSessionUser(ctx context.Context, userID string) error {
	u, err := s.SessionUser(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("account no longer available")
	}
	switch u.Status {
	case string(common.StatusActive), string(common.StatusPendingDeletion):
		return nil
	default:
		return errors.New("account no longer available")
	}
}

func finalizeAnonymizedUserInTx(ctx context.Context, tx domain.Repository, u *domain.User, userID string) error {
	if u == nil {
		return domain.ErrNotFound
	}
	emailNorm := normalizeEmailForDeletionBlock(u.Email)
	phoneNorm := normalizePhoneForDeletionBlock(u.Phone)

	deadHash, err := auth.HashPassword(uuid.New().String() + uuid.New().String())
	if err != nil {
		return err
	}
	blockedUntil := time.Now().Unix() + accountDeletionCooldownSec

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
	return tx.DeleteUser(ctx, userID)
}

func storyIsPublishedPublic(st *domain.Story) bool {
	if st == nil {
		return false
	}
	if strings.TrimSpace(strings.ToLower(st.Status)) != strings.ToLower(string(common.ContentStatusPublished)) {
		return false
	}
	return strings.TrimSpace(strings.ToLower(st.Visibility)) == strings.ToLower(string(domain.StoryVisibilityPublic))
}

func rowToDeletionStatus(row *domain.AccountDeletionRequestRow, userStatus string) *domain.AccountDeletionStatus {
	if row == nil {
		return &domain.AccountDeletionStatus{
			IsPending:  false,
			UserStatus: userStatus,
		}
	}
	ok := row.Status == string(common.DeletionStatusPending) || row.Status == string(common.DeletionStatusProcessing)
	return &domain.AccountDeletionStatus{
		IsPending:             ok,
		UserStatus:            userStatus,
		DeletionRequestStatus: row.Status,
		ScheduledDeletionAt:   &row.ScheduledDeletionAt,
		GracePeriodEndsAt:     &row.ScheduledDeletionAt,
		Reason:                row.Reason,
	}
}

// AccountDeletionSubmit is the authenticated intent to schedule account deletion (risk ACK + OTP proof enforced server-side first time).
type AccountDeletionSubmit struct {
	RiskAcknowledged bool
}

// RequestAccountDeletion starts grace-period deletion for an active account.
func (s *Service) RequestAccountDeletion(ctx context.Context, userID string, submit *AccountDeletionSubmit) (*domain.AccountDeletionStatus, error) {
	if submit == nil || !submit.RiskAcknowledged {
		return nil, ErrAccountDeletionRiskAckRequired
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, domain.ErrNotFound
	}

	switch u.Status {
	case string(common.StatusPendingDeletion):
		row, err := s.repo.GetActiveAccountDeletionRequestByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if row != nil && row.Status == string(common.DeletionStatusPending) {
			return rowToDeletionStatus(row, u.Status), nil
		}
		return nil, fmt.Errorf("account is %s", u.Status)
	case string(common.StatusActive):
	default:
		return nil, fmt.Errorf("account is %s", u.Status)
	}
	if prev, err := s.repo.GetActiveAccountDeletionRequestByUser(ctx, userID); err != nil {
		return nil, err
	} else if prev != nil && prev.Status == string(common.DeletionStatusPending) {
		return rowToDeletionStatus(prev, u.Status), nil
	}

	if err := s.assertAccountDeletionSMSProof(ctx, userID); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	grace := s.effectiveAccountDeletionGraceSeconds()
	scheduled := now + grace
	row := &domain.AccountDeletionRequestRow{
		ID:                  uuid.New().String(),
		UserID:              userID,
		Status:              string(common.DeletionStatusPending),
		ScheduledDeletionAt: scheduled,
		RequestedAt:         now,
	}
	if err := s.repo.WithTransaction(ctx, func(tx domain.Repository) error {
		if err := tx.CreateAccountDeletionRequest(ctx, row); err != nil {
			return err
		}
		u.Status = string(common.StatusPendingDeletion)
		u.UpdatedAt = now
		return tx.UpdateUser(ctx, u)
	}); err != nil {
		return nil, err
	}

	s.clearAccountDeletionSMSProof(ctx, userID)

	mailTo := strings.TrimSpace(u.Email)
	display := u.DisplayName
	if mailTo != "" && strings.Contains(mailTo, "@") && !strings.HasPrefix(strings.ToLower(mailTo), "del+") {
		go func(addr, name string, at int64) {
			if err := emailpkg.AccountDeletionGracePeriodScheduledEmail([]string{addr}, name, at); err != nil {
				s.logger.Warn("account deletion grace email failed", zap.Error(err))
			}
		}(mailTo, display, scheduled)
	}

	st := rowToDeletionStatus(row, string(common.StatusPendingDeletion))
	s.invalidateUserCache(ctx, userID)
	return st, nil
}

// GetAccountDeletionStatus returns current deletion workflow status for UI.
func (s *Service) GetAccountDeletionStatus(ctx context.Context, userID string) (*domain.AccountDeletionStatus, error) {
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil || u == nil {
		return nil, domain.ErrNotFound
	}
	row, err := s.repo.GetActiveAccountDeletionRequestByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if row != nil {
		return rowToDeletionStatus(row, u.Status), nil
	}
	return &domain.AccountDeletionStatus{
		IsPending:  false,
		UserStatus: u.Status,
	}, nil
}

// CancelAccountDeletion restores an active login during grace window.
func (s *Service) CancelAccountDeletion(ctx context.Context, userID string) error {
	row, err := s.repo.GetActiveAccountDeletionRequestByUser(ctx, userID)
	if err != nil {
		return err
	}
	if row == nil {
		return domain.ErrNotFound
	}
	if row.Status != string(common.DeletionStatusPending) {
		return fmt.Errorf("deletion cannot be cancelled in status %s", row.Status)
	}
	now := time.Now().Unix()
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil || u == nil {
		return domain.ErrNotFound
	}
	if err := s.repo.WithTransaction(ctx, func(tx domain.Repository) error {
		if err := tx.UpdateAccountDeletionRequestStatus(ctx, row.ID, string(common.DeletionStatusCancelled), nil, &now, ""); err != nil {
			return err
		}
		u.Status = string(common.StatusActive)
		u.UpdatedAt = now
		return tx.UpdateUser(ctx, u)
	}); err != nil {
		return err
	}
	s.invalidateUserCache(ctx, userID)
	s.ClearAccountDeletionSMSState(ctx, userID)
	return nil
}

func (s *Service) terminationDeleteDraftFragments(ctx context.Context, uid string) error {
	if s.terminationFragmentRepo == nil {
		s.logger.Warn("fragment repository not wired; skipping fragment deletion during account termination")
		return nil
	}
	for i := 0; i < 1000; i++ {
		batch, _, err := s.terminationFragmentRepo.ListDraftsByCreatorID(ctx, uid, 100, 0)
		if err != nil || len(batch) == 0 {
			return err
		}
		for _, fr := range batch {
			if fr == nil || fr.ID == "" {
				continue
			}
			if err := s.terminationFragmentRepo.Delete(ctx, fr.ID); err != nil {
				s.logger.Warn("terminate draft fragment failed", zap.String("fragmentId", fr.ID), zap.Error(err))
			}
		}
	}
	return nil
}

func (s *Service) terminationDeleteNonPublishedPublicFragments(ctx context.Context, uid string) error {
	if s.terminationFragmentRepo == nil {
		return nil
	}
	for i := 0; i < 1000; i++ {
		batch, _, err := s.terminationFragmentRepo.ListByCreatorID(ctx, uid, 100, 0)
		if err != nil || len(batch) == 0 {
			return err
		}
		for _, fr := range batch {
			if fr == nil || fr.ID == "" {
				continue
			}
			vis := domain.NormalizeFragmentVisibility(fr.Visibility)
			if vis == domain.FragmentVisibilityPublic {
				continue
			}
			if err := s.terminationFragmentRepo.Delete(ctx, fr.ID); err != nil {
				s.logger.Warn("terminate fragment failed", zap.String("fragmentId", fr.ID), zap.Error(err))
			}
		}
	}
	return nil
}

func (s *Service) terminationDeleteEligibleStories(ctx context.Context, uid string) error {
	const page = 120
	for pass := 0; pass < 5000; pass++ {
		batch, err := s.repo.StoriesByUser(ctx, uid, page, 0)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		deletedAny := false
		for _, st := range batch {
			if st == nil || st.ID == "" {
				continue
			}
			if storyIsPublishedPublic(st) {
				continue
			}
			if err := s.repo.TerminateOwnedStoryAndDependents(ctx, st.ID); err != nil {
				s.logger.Warn("terminate story failed", zap.String("storyId", st.ID), zap.Error(err))
				continue
			}
			deletedAny = true
		}
		if !deletedAny {
			break
		}
	}
	return nil
}

// ProcessDueAccountDeletionBatch claims and finalizes overdue deletion requests (best-effort, idempotent per user).
func (s *Service) ProcessDueAccountDeletionBatch(ctx context.Context, limit int) {
	if limit <= 0 {
		limit = 25
	}
	nowUnix := time.Now().Unix()
	due, err := s.repo.ListDueAccountDeletionRequests(ctx, nowUnix, limit)
	if err != nil {
		s.logger.Warn("list due account deletions failed", zap.Error(err))
		return
	}
	sysUID := s.effectiveSystemAnonymousUserID()

	for _, req := range due {
		if req == nil || req.ID == "" || req.UserID == "" {
			continue
		}
		claimed, err := s.repo.ClaimPendingDueAccountDeletionRequest(ctx, req.ID, nowUnix)
		if err != nil || !claimed {
			continue
		}

		uid := req.UserID
		u, err := s.repo.UserByID(ctx, uid)
		if err != nil || u == nil {
			now := time.Now().Unix()
			_ = s.repo.UpdateAccountDeletionRequestStatus(ctx, req.ID, string(common.DeletionStatusCancelled), nil, &now, "user_missing")
			continue
		}
		if u.Status != string(common.StatusPendingDeletion) {
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}

		sysUser, err := s.repo.UserByID(ctx, sysUID)
		if err != nil || sysUser == nil || sysUser.Status == string(common.StatusDeleted) {
			s.logger.Warn("system anonymous user missing; rolling back deletion claim", zap.String("sysId", sysUID))
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}

		mailTo := strings.TrimSpace(u.Email)
		origDisplay := u.DisplayName

		if err := s.terminationDeleteDraftFragments(ctx, uid); err != nil {
			s.logger.Warn("draft fragment termination failed", zap.String("userId", uid), zap.Error(err))
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}
		if err := s.terminationDeleteNonPublishedPublicFragments(ctx, uid); err != nil {
			s.logger.Warn("non-public fragment termination failed", zap.String("userId", uid), zap.Error(err))
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}
		if err := s.terminationDeleteEligibleStories(ctx, uid); err != nil {
			s.logger.Warn("story subtree termination failed", zap.String("userId", uid), zap.Error(err))
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}

		_ = s.CancelMembership(ctx, uid)

		if err := s.repo.ApplyAccountDeletionContentReassignment(ctx, uid, sysUID); err != nil {
			s.logger.Warn("apply reassignment failed", zap.String("userId", uid), zap.Error(err))
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}
		if err := s.repo.ApplyAccountDeletionUserSocialGraphPurge(ctx, uid); err != nil {
			s.logger.Warn("social purge failed", zap.String("userId", uid), zap.Error(err))
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}

		u2, err := s.repo.UserByID(ctx, uid)
		if err != nil || u2 == nil {
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}
		proc := nowUnix
		if err := s.repo.WithTransaction(ctx, func(tx domain.Repository) error {
			return finalizeAnonymizedUserInTx(ctx, tx, u2, uid)
		}); err != nil {
			s.logger.Error("finalize anonymized account failed", zap.String("userId", uid), zap.Error(err))
			_ = s.repo.ResetAccountDeletionRequestToPending(ctx, req.ID)
			continue
		}

		if err := s.repo.UpdateAccountDeletionRequestStatus(ctx, req.ID, string(common.DeletionStatusCompleted), &proc, nil, ""); err != nil {
			s.logger.Error("mark deletion completed failed", zap.String("requestId", req.ID), zap.Error(err))
		}

		s.invalidateUserCache(ctx, uid)
		s.invalidateUserCache(ctx, sysUID)

		if mailTo != "" && strings.Contains(mailTo, "@") && !strings.HasPrefix(strings.ToLower(mailTo), "del+") {
			go func(addr, name string) {
				if err := emailpkg.AccountDeletedEmail([]string{addr}, name); err != nil {
					s.logger.Warn("account deleted confirmation email failed", zap.Error(err))
				}
			}(mailTo, origDisplay)
		}
		s.logger.Info("account deletion completed after grace window", zap.String("userID", uid))
	}
}
