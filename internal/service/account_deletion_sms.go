package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

const (
	acctDeletionOTPTTL             = 10 * time.Minute
	acctDeletionProofTTL           = 15 * time.Minute
	acctDeletionSendWindow         = time.Minute
	acctDeletionSendMaxPerWindow   = int64(1)
	acctDeletionSMSAttemptMax      = 5
)

var acctDelSMSDecimal6 = regexp.MustCompile(`^[0-9]{6}$`)

// Account deletion sentinel errors for HTTP routing.
var (
	ErrAccountDeletionRiskAckRequired           = errors.New("risk acknowledgement required for account deletion")
	ErrAccountDeletionSMSProofMissing           = errors.New("sms verification required before deleting account")
	ErrAccountDeletionVerifiedPhoneRequired     = errors.New("a verified mainland China mobile number must be bound to delete this account")
	ErrAccountDeletionSMSOnlyForActiveAccounts = errors.New("account deletion verification texts can only be requested for active accounts")
	ErrAccountDeletionSMSCacheRequired          = errors.New("account deletion sms requires redis cache")
	ErrAccountDeletionInvalidDeletionSMSCodeFmt = errors.New("verification code must be 6 digits")
)

type acctDeletionOTPStored struct {
	PhoneNorm string `json:"phoneNorm"`
	CodeHash  string `json:"codeHash"`
	Attempts  int    `json:"attempts"`
}

// ClearAccountDeletionSMSState removes OTP / proof Redis keys for a user (e.g. after cancellation).
func (s *Service) ClearAccountDeletionSMSState(ctx context.Context, userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	c := s.getCache()
	if c == nil {
		return
	}
	_ = c.Delete(ctx, cache.AccountDeletionOTPKey(userID), cache.AccountDeletionProofKey(userID))
}

func (s *Service) SendAccountDeletionSMS(ctx context.Context, userID string, clientIP string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.ErrInvalidInput
	}
	c := s.getCache()
	if c == nil {
		return ErrAccountDeletionSMSCacheRequired
	}
	me, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if me == nil {
		return domain.ErrNotFound
	}
	if me.Status != string(common.StatusActive) {
		return ErrAccountDeletionSMSOnlyForActiveAccounts
	}
	if strings.TrimSpace(me.Phone) == "" || me.PhoneVerifiedAt == nil || *me.PhoneVerifiedAt <= 0 {
		return ErrAccountDeletionVerifiedPhoneRequired
	}
	phoneNorm, err := NormalizeChinaPhone(me.Phone)
	if err != nil {
		return ErrAccountDeletionVerifiedPhoneRequired
	}

	var undoKeys []string
	rollbackEarlier := func() {
		for i := len(undoKeys) - 1; i >= 0; i-- {
			_, _ = c.Decr(ctx, undoKeys[i])
		}
		undoKeys = undoKeys[:0]
	}
	tryIncrLimit := func(key string) error {
		n, errIncr := c.Incr(ctx, key)
		if errIncr != nil {
			rollbackEarlier()
			return errIncr
		}
		if n == 1 {
			_ = c.Expire(ctx, key, acctDeletionSendWindow)
		}
		if n > acctDeletionSendMaxPerWindow {
			rollbackEarlier()
			return ErrSMSSendRateLimited
		}
		undoKeys = append(undoKeys, key)
		return nil
	}

	if clientIP != "" {
		if err := tryIncrLimit(cache.AccountDeletionSMSIPLimitKey(clientIP)); err != nil {
			return err
		}
	}
	if err := tryIncrLimit(cache.AccountDeletionSMSUserSendKey(userID)); err != nil {
		return err
	}
	if err := tryIncrLimit(cache.AccountDeletionSMSPhoneSendKey(phoneNorm)); err != nil {
		return err
	}

	code, err := generate6DigitCode()
	if err != nil {
		rollbackEarlier()
		return err
	}
	hash, err := s.hashSMSCode(userID, phoneNorm, code)
	if err != nil {
		rollbackEarlier()
		return err
	}

	otpKey := cache.AccountDeletionOTPKey(userID)
	stored := acctDeletionOTPStored{PhoneNorm: phoneNorm, CodeHash: hash, Attempts: 0}
	if err := c.Set(ctx, otpKey, stored, acctDeletionOTPTTL); err != nil {
		rollbackEarlier()
		return err
	}

	if err := SendAliyunOTPCode(phoneNorm, code); err != nil {
		_ = c.Delete(ctx, otpKey)
		rollbackEarlier()
		s.logger.Warn("account deletion sms send failed", zap.Error(err))
		return err
	}

	proofKey := cache.AccountDeletionProofKey(userID)
	_ = c.Delete(ctx, proofKey)
	return nil
}

// VerifyAccountDeletionSMS verifies the 6-digit code and grants a short-lived proof for DELETE /auth/account.
func (s *Service) VerifyAccountDeletionSMS(ctx context.Context, userID, rawCode string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.ErrInvalidInput
	}
	code := strings.TrimSpace(rawCode)
	if len(code) != 6 || !acctDelSMSDecimal6.MatchString(code) {
		return ErrAccountDeletionInvalidDeletionSMSCodeFmt
	}

	c := s.getCache()
	if c == nil {
		return ErrAccountDeletionSMSCacheRequired
	}

	me, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if me == nil {
		return domain.ErrNotFound
	}
	if me.Status != string(common.StatusActive) {
		return ErrAccountDeletionSMSOnlyForActiveAccounts
	}
	if strings.TrimSpace(me.Phone) == "" || me.PhoneVerifiedAt == nil || *me.PhoneVerifiedAt <= 0 {
		return ErrAccountDeletionVerifiedPhoneRequired
	}
	phoneNorm, err := NormalizeChinaPhone(me.Phone)
	if err != nil {
		return ErrAccountDeletionVerifiedPhoneRequired
	}

	otpKey := cache.AccountDeletionOTPKey(userID)
	var stored acctDeletionOTPStored
	if err := c.Get(ctx, otpKey, &stored); err != nil || stored.CodeHash == "" {
		return errors.New("code expired or not found, please request a new deletion verification code")
	}
	if stored.PhoneNorm != phoneNorm {
		return errors.New("phone mismatch for deletion verification, please request a new code")
	}
	if stored.Attempts >= acctDeletionSMSAttemptMax {
		_ = c.Delete(ctx, otpKey)
		return errors.New("too many incorrect deletion verification attempts")
	}

	wantHash, err := s.hashSMSCode(userID, phoneNorm, code)
	if err != nil {
		return err
	}
	wantB := []byte(strings.TrimSpace(strings.ToLower(wantHash)))
	gotB := []byte(strings.TrimSpace(strings.ToLower(stored.CodeHash)))
	if subtle.ConstantTimeCompare(wantB, gotB) != 1 {
		stored.Attempts++
		_ = c.Set(ctx, otpKey, stored, acctDeletionOTPTTL)
		return errors.New("invalid verification code")
	}

	proofKey := cache.AccountDeletionProofKey(userID)
	if err := c.Set(ctx, proofKey, true, acctDeletionProofTTL); err != nil {
		return err
	}
	_ = c.Delete(ctx, otpKey)
	return nil
}

// assertAccountDeletionSMSProof verifies the short-lived Redis proof exists (does not delete it).
func (s *Service) assertAccountDeletionSMSProof(ctx context.Context, userID string) error {
	c := s.getCache()
	if c == nil {
		return ErrAccountDeletionSMSCacheRequired
	}
	ok, err := c.Exists(ctx, cache.AccountDeletionProofKey(userID))
	if err != nil {
		return err
	}
	if !ok {
		return ErrAccountDeletionSMSProofMissing
	}
	return nil
}

// clearAccountDeletionSMSProof deletes proof after DELETE /account completed scheduling.
func (s *Service) clearAccountDeletionSMSProof(ctx context.Context, userID string) {
	c := s.getCache()
	if c == nil {
		return
	}
	_ = c.Delete(ctx, cache.AccountDeletionProofKey(userID))
}
