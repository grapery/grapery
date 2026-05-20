package service

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/email"
	"go.uber.org/zap"
)

const (
	acctBindPhoneOTPTTL        = 5 * time.Minute
	acctBindEmailCodeTTL       = 10 * time.Minute
	acctBindVerifyAttemptMax   = 3
	acctBindModifyLockTTL      = 7 * 24 * time.Hour
	acctBindSendWindow         = time.Minute
	acctBindSendMaxPerWindow   = int64(1)
	acctBindPhoneChangeDailyMax = int64(3)
)

// Stable sentinel messages for iOS ServerMessageLocalization mapping.
var (
	ErrContactOTPExpired              = errors.New("account contact verification code expired or not sent")
	ErrAccountContactInvalidCode      = errors.New("invalid account contact verification code")
	ErrAccountContactVerifyLocked     = errors.New("account contact verification locked for 7 days")
	ErrAccountContactModifyLocked     = errors.New("account contact modification locked")
	ErrAccountContactEmailRegistered  = errors.New("email already registered")
	ErrAccountContactInvalidEmail     = errors.New("invalid email address")
	ErrAccountContactCacheRequired    = errors.New("contact verification temporarily unavailable")
	ErrAccountContactInvalidCodeFmt   = errors.New("verification code must be 6 digits")
	ErrAccountContactPhoneChangeDailyLimit = errors.New("phone number can only be changed up to 3 times per day")
)

type accountContactOTPStored struct {
	Target   string `json:"target"`
	CodeHash string `json:"codeHash"`
	Attempts int    `json:"attempts"`
}

type accountContactLockStored struct {
	LockedUntil int64 `json:"lockedUntil"`
}

// AccountContactLockedError is returned when modify is blocked (7-day lock).
type AccountContactLockedError struct {
	LockedUntil int64
	VerifyLock  bool // true = just hit 3 wrong attempts; false = already locked
}

func (e *AccountContactLockedError) Error() string {
	if e.VerifyLock {
		return ErrAccountContactVerifyLocked.Error()
	}
	return ErrAccountContactModifyLocked.Error()
}

// AccountContactInvalidCodeError carries remaining attempts after a wrong OTP.
type AccountContactInvalidCodeError struct {
	AttemptsRemaining int
}

func (e *AccountContactInvalidCodeError) Error() string {
	return ErrAccountContactInvalidCode.Error()
}

func normalizeAccountContactEmail(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "", ErrAccountContactInvalidEmail
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return "", ErrAccountContactInvalidEmail
	}
	return s, nil
}

func (s *Service) accountContactGetLock(ctx context.Context, lockKey string) (int64, bool, error) {
	c := s.getCache()
	if c == nil {
		return 0, false, ErrAccountContactCacheRequired
	}
	var stored accountContactLockStored
	if err := c.Get(ctx, lockKey, &stored); err != nil {
		return 0, false, nil
	}
	if stored.LockedUntil <= 0 {
		return 0, true, nil
	}
	if time.Now().Unix() >= stored.LockedUntil {
		_ = c.Delete(ctx, lockKey)
		return 0, false, nil
	}
	return stored.LockedUntil, true, nil
}

func (s *Service) accountContactSetModifyLock(ctx context.Context, lockKey string) (int64, error) {
	c := s.getCache()
	if c == nil {
		return 0, ErrAccountContactCacheRequired
	}
	until := time.Now().Add(acctBindModifyLockTTL).Unix()
	stored := accountContactLockStored{LockedUntil: until}
	if err := c.Set(ctx, lockKey, stored, acctBindModifyLockTTL); err != nil {
		return 0, err
	}
	return until, nil
}

func (s *Service) accountContactAssertNotLocked(ctx context.Context, lockKey string) error {
	until, locked, err := s.accountContactGetLock(ctx, lockKey)
	if err != nil {
		return err
	}
	if locked {
		return &AccountContactLockedError{LockedUntil: until, VerifyLock: false}
	}
	return nil
}

func (s *Service) accountContactIncrSendLimits(ctx context.Context, keys ...string) error {
	c := s.getCache()
	if c == nil {
		return ErrAccountContactCacheRequired
	}
	undoKeys := make([]string, 0, len(keys))
	rollback := func() {
		for i := len(undoKeys) - 1; i >= 0; i-- {
			_, _ = c.Decr(ctx, undoKeys[i])
		}
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		n, err := c.Incr(ctx, key)
		if err != nil {
			rollback()
			return err
		}
		if n == 1 {
			_ = c.Expire(ctx, key, acctBindSendWindow)
		}
		if n > acctBindSendMaxPerWindow {
			rollback()
			return ErrSMSSendRateLimited
		}
		undoKeys = append(undoKeys, key)
	}
	return nil
}

func chinaTimeLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func chinaCalendarDayKey(t time.Time) string {
	return t.In(chinaTimeLocation()).Format("20060102")
}

func untilEndOfChinaDay(t time.Time) time.Duration {
	loc := chinaTimeLocation()
	local := t.In(loc)
	end := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc)
	d := end.Sub(t)
	if d <= 0 {
		return time.Second
	}
	return d
}

// accountContactIsPhoneNumberChange is true when the user already has a phone and the new number differs.
func accountContactIsPhoneNumberChange(currentPhone, newNormalizedPhone string) bool {
	cur := strings.TrimSpace(currentPhone)
	if cur == "" {
		return false
	}
	cn, err := NormalizeChinaPhone(cur)
	if err != nil {
		cn = strings.TrimPrefix(strings.TrimPrefix(cur, "+86"), " ")
	}
	return cn != newNormalizedPhone
}

func (s *Service) accountContactPhoneChangeCountToday(ctx context.Context, userID string) (int64, error) {
	c := s.getCache()
	if c == nil {
		return 0, ErrAccountContactCacheRequired
	}
	key := cache.AccountBindPhoneChangeDailyKey(userID, chinaCalendarDayKey(time.Now()))
	raw, err := c.Eval(ctx, `return tonumber(redis.call('GET', KEYS[1]) or '0')`, []string{key})
	if err != nil {
		return 0, err
	}
	switch v := raw.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, nil
	}
}

func (s *Service) accountContactAssertPhoneChangeDailyQuota(ctx context.Context, userID string) error {
	n, err := s.accountContactPhoneChangeCountToday(ctx, userID)
	if err != nil {
		return err
	}
	if n >= acctBindPhoneChangeDailyMax {
		return ErrAccountContactPhoneChangeDailyLimit
	}
	return nil
}

func (s *Service) accountContactRecordPhoneChange(ctx context.Context, userID string) error {
	c := s.getCache()
	if c == nil {
		return ErrAccountContactCacheRequired
	}
	now := time.Now()
	key := cache.AccountBindPhoneChangeDailyKey(userID, chinaCalendarDayKey(now))
	n, err := c.Incr(ctx, key)
	if err != nil {
		return err
	}
	if n == 1 {
		_ = c.Expire(ctx, key, untilEndOfChinaDay(now))
	}
	return nil
}

// SendAccountContactPhoneSMS sends OTP for settings phone bind/change.
func (s *Service) SendAccountContactPhoneSMS(ctx context.Context, userID, rawPhone, clientIP string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user not found")
	}
	c := s.getCache()
	if c == nil {
		return ErrAccountContactCacheRequired
	}
	if err := s.accountContactAssertNotLocked(ctx, cache.AccountBindPhoneLockKey(userID)); err != nil {
		return err
	}

	phone, err := NormalizeChinaPhone(rawPhone)
	if err != nil {
		return err
	}

	other, err := s.repo.UserByPhone(ctx, phone)
	if err != nil {
		return err
	}
	if other != nil && other.ID != userID {
		return errors.New("this phone number is already bound to another account")
	}

	blocked, err := s.repo.IsAccountReRegistrationBlocked(ctx, "", phone)
	if err != nil {
		return err
	}
	if blocked {
		return errors.New("this phone number cannot be bound within 30 days after account deletion")
	}

	me, err := s.repo.UserByID(ctx, userID)
	if err != nil || me == nil {
		return errors.New("user not found")
	}
	if accountContactIsPhoneNumberChange(me.Phone, phone) {
		if err := s.accountContactAssertPhoneChangeDailyQuota(ctx, userID); err != nil {
			return err
		}
	}

	limitKeys := []string{
		cache.AccountBindPhoneSendUserKey(userID),
		cache.AccountBindPhoneSendPhoneKey(phone),
	}
	if clientIP != "" {
		limitKeys = append(limitKeys, cache.AccountBindPhoneSendIPKey(clientIP))
	}
	if err := s.accountContactIncrSendLimits(ctx, limitKeys...); err != nil {
		return err
	}

	code, err := generate6DigitCode()
	if err != nil {
		return err
	}
	hash, err := s.hashSMSCode(userID, phone, code)
	if err != nil {
		return err
	}

	otpKey := cache.AccountBindPhoneOTPKey(userID, phone)
	data := accountContactOTPStored{
		Target:   phone,
		CodeHash: hash,
		Attempts: 0,
	}
	if err := c.Set(ctx, otpKey, data, acctBindPhoneOTPTTL); err != nil {
		return err
	}

	if err := SendAliyunOTPCode(phone, code); err != nil {
		_ = c.Delete(ctx, otpKey)
		s.logger.Warn("account contact sms send failed", zap.Error(err))
		return err
	}
	return nil
}

// VerifyAccountContactPhoneSMS verifies OTP and binds phone on the user record.
func (s *Service) VerifyAccountContactPhoneSMS(ctx context.Context, userID, rawPhone, code string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user not found")
	}
	if err := s.accountContactAssertNotLocked(ctx, cache.AccountBindPhoneLockKey(userID)); err != nil {
		return err
	}

	phone, err := NormalizeChinaPhone(rawPhone)
	if err != nil {
		return err
	}
	code = strings.TrimSpace(code)
	if len(code) != 6 || !smsCodeDecimal6.MatchString(code) {
		return ErrAccountContactInvalidCodeFmt
	}

	c := s.getCache()
	if c == nil {
		return ErrContactOTPExpired
	}

	otpKey := cache.AccountBindPhoneOTPKey(userID, phone)
	var stored accountContactOTPStored
	if err := c.Get(ctx, otpKey, &stored); err != nil || stored.CodeHash == "" {
		return ErrContactOTPExpired
	}

	wantHash, err := s.hashSMSCode(userID, phone, code)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(wantHash), []byte(stored.CodeHash)) {
		stored.Attempts++
		if stored.Attempts >= acctBindVerifyAttemptMax {
			_ = c.Delete(ctx, otpKey)
			until, lockErr := s.accountContactSetModifyLock(ctx, cache.AccountBindPhoneLockKey(userID))
			if lockErr != nil {
				return lockErr
			}
			return &AccountContactLockedError{LockedUntil: until, VerifyLock: true}
		}
		_ = c.Set(ctx, otpKey, stored, acctBindPhoneOTPTTL)
		return &AccountContactInvalidCodeError{AttemptsRemaining: acctBindVerifyAttemptMax - stored.Attempts}
	}

	other, err := s.repo.UserByPhone(ctx, phone)
	if err != nil {
		return err
	}
	if other != nil && other.ID != userID {
		return errors.New("this phone number is already bound to another account")
	}

	blocked, err := s.repo.IsAccountReRegistrationBlocked(ctx, "", phone)
	if err != nil {
		return err
	}
	if blocked {
		return errors.New("this phone number cannot be bound within 30 days after account deletion")
	}

	now := time.Now().Unix()
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}
	isChange := accountContactIsPhoneNumberChange(user.Phone, phone)
	if isChange {
		if err := s.accountContactAssertPhoneChangeDailyQuota(ctx, userID); err != nil {
			return err
		}
	}

	user.Phone = phone
	user.PhoneVerifiedAt = &now
	user.PendingOAuthPhoneSMS = false
	user.UpdatedAt = now
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if isChange {
		if err := s.accountContactRecordPhoneChange(ctx, userID); err != nil {
			s.logger.Warn("account contact phone change daily counter failed", zap.Error(err))
		}
	}

	_ = c.Delete(ctx, otpKey)
	s.invalidateUserCache(ctx, userID)
	return nil
}

// SendAccountContactEmailCode sends OTP to a new email for settings bind/change.
func (s *Service) SendAccountContactEmailCode(ctx context.Context, userID, rawEmail, clientIP string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user not found")
	}
	c := s.getCache()
	if c == nil {
		return ErrAccountContactCacheRequired
	}
	if err := s.accountContactAssertNotLocked(ctx, cache.AccountBindEmailLockKey(userID)); err != nil {
		return err
	}

	emailNorm, err := normalizeAccountContactEmail(rawEmail)
	if err != nil {
		return err
	}

	existing, err := s.repo.UserByEmail(ctx, emailNorm)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != userID {
		return ErrAccountContactEmailRegistered
	}

	emailBlock := normalizeEmailForDeletionBlock(emailNorm)
	blocked, err := s.repo.IsAccountReRegistrationBlocked(ctx, emailBlock, "")
	if err != nil {
		return err
	}
	if blocked {
		return errors.New("this email cannot be used within 30 days after account deletion")
	}

	limitKeys := []string{
		cache.AccountBindEmailSendUserKey(userID),
		cache.AccountBindEmailSendEmailKey(emailNorm),
	}
	if clientIP != "" {
		limitKeys = append(limitKeys, cache.AccountBindEmailSendIPKey(clientIP))
	}
	if err := s.accountContactIncrSendLimits(ctx, limitKeys...); err != nil {
		return err
	}

	code, err := generate6DigitCode()
	if err != nil {
		return err
	}
	hash, err := s.hashEmailVerificationCode(emailNorm, code)
	if err != nil {
		return err
	}

	otpKey := cache.AccountBindEmailCodeKey(userID, emailNorm)
	data := accountContactOTPStored{
		Target:   emailNorm,
		CodeHash: hash,
		Attempts: 0,
	}
	if err := c.Set(ctx, otpKey, data, acctBindEmailCodeTTL); err != nil {
		return err
	}

	me, err := s.repo.UserByID(ctx, userID)
	if err != nil || me == nil {
		return errors.New("user not found")
	}
	display := me.DisplayName
	if display == "" {
		display = me.Username
	}
	if err := email.VerificationCodeEmail([]string{emailNorm}, display, code, int(acctBindEmailCodeTTL.Minutes())); err != nil {
		_ = c.Delete(ctx, otpKey)
		s.logger.Warn("account contact email send failed", zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) ConfirmAccountContactEmail(ctx context.Context, userID, rawEmail, code string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user not found")
	}
	if err := s.accountContactAssertNotLocked(ctx, cache.AccountBindEmailLockKey(userID)); err != nil {
		return err
	}

	emailNorm, err := normalizeAccountContactEmail(rawEmail)
	if err != nil {
		return err
	}
	code = strings.TrimSpace(code)
	if len(code) != 6 || !smsCodeDecimal6.MatchString(code) {
		return ErrAccountContactInvalidCodeFmt
	}

	c := s.getCache()
	if c == nil {
		return ErrContactOTPExpired
	}

	otpKey := cache.AccountBindEmailCodeKey(userID, emailNorm)
	var stored accountContactOTPStored
	if err := c.Get(ctx, otpKey, &stored); err != nil || stored.CodeHash == "" {
		return ErrContactOTPExpired
	}

	expected, err := s.hashEmailVerificationCode(emailNorm, code)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected), []byte(stored.CodeHash)) {
		stored.Attempts++
		if stored.Attempts >= acctBindVerifyAttemptMax {
			_ = c.Delete(ctx, otpKey)
			until, lockErr := s.accountContactSetModifyLock(ctx, cache.AccountBindEmailLockKey(userID))
			if lockErr != nil {
				return lockErr
			}
			return &AccountContactLockedError{LockedUntil: until, VerifyLock: true}
		}
		_ = c.Set(ctx, otpKey, stored, acctBindEmailCodeTTL)
		return &AccountContactInvalidCodeError{AttemptsRemaining: acctBindVerifyAttemptMax - stored.Attempts}
	}

	existing, err := s.repo.UserByEmail(ctx, emailNorm)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != userID {
		return ErrAccountContactEmailRegistered
	}

	user, err := s.repo.UserByID(ctx, userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}
	user.Email = emailNorm
	user.EmailVerified = true
	user.UpdatedAt = time.Now().Unix()
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	_ = c.Delete(ctx, otpKey)
	s.invalidateUserCache(ctx, userID)
	return nil
}
