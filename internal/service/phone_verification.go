package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

type smsOTPStoredData struct {
	Phone    string `json:"phone"`
	CodeHash string `json:"codeHash"`
	Attempts int    `json:"attempts"`
}

const (
	smsPhoneOTPTTL = 5 * time.Minute
	// 每个 IP、每个登录用户、每个手机号：滑动窗口内最多发送次数（窗口长度见 smsPhoneSendWindow）
	smsPhoneSendWindow       = 1 * time.Minute
	smsPhoneSendMaxPerWindow = 1
	smsPhoneAttemptMax       = 5
)

var errSMSSendRateLimited = errors.New("verification code can only be sent once per minute per account, phone number, and IP; please try again later")

// AttachPhoneVerificationRequirement sets RequiresPhoneVerification from PendingOAuthPhoneSMS
// (only first-time Apple/WeChat-registered accounts until SMS completes).
func (s *Service) AttachPhoneVerificationRequirement(_ context.Context, u *domain.User) error {
	if u == nil {
		return nil
	}
	u.RequiresPhoneVerification = u.PendingOAuthPhoneSMS
	return nil
}

func (s *Service) smsCodeSecret() (string, error) {
	if v := os.Getenv("SMS_VERIFICATION_SECRET"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v), nil
	}
	return s.emailVerifySecret()
}

func (s *Service) hashSMSCode(userID, phone, code string) (string, error) {
	secret, err := s.smsCodeSecret()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(userID))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write([]byte(strings.TrimSpace(phone)))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// NormalizeChinaPhone trims input and returns domestic 11-digit mobile (no +86), or error.
func NormalizeChinaPhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "+86")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 11 {
		return "", errors.New("phone must be 11 digits")
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return "", errors.New("phone must contain only digits")
		}
	}
	if s[0] != '1' {
		return "", errors.New("invalid china mobile")
	}
	return s, nil
}

// userMayUsePhoneSMSGate returns nil only when first-time Apple/WeChat signup still has SMS pending.
func (s *Service) userMayUsePhoneSMSGate(ctx context.Context, userID string) error {
	me, err := s.repo.UserByID(ctx, userID)
	if err != nil || me == nil {
		return errors.New("user not found")
	}
	if !me.PendingOAuthPhoneSMS {
		return errors.New("phone SMS verification is not required for this account")
	}
	if me.PhoneVerifiedAt != nil && *me.PhoneVerifiedAt > 0 {
		return errors.New("phone already verified")
	}
	return nil
}

// SendPhoneSMSVerificationCode sends an OTP to the given phone for an authenticated user.
func (s *Service) SendPhoneSMSVerificationCode(ctx context.Context, userID, rawPhone, clientIP string) error {
	c := s.getCache()
	if c == nil {
		return errors.New("SMS verification requires Redis cache")
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

	if err := s.userMayUsePhoneSMSGate(ctx, userID); err != nil {
		return err
	}

	// Rate limits: IP / user / phone each max once per smsPhoneSendWindow.
	// If a later dimension fails, DECR earlier keys so one failed attempt does not consume unrelated quotas.
	// If the failing dimension itself exceeded max, do not DECR that key (keep blocking).
	undoKeys := make([]string, 0, 3)
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
			_ = c.Expire(ctx, key, smsPhoneSendWindow)
		}
		if n > smsPhoneSendMaxPerWindow {
			rollbackEarlier()
			return errSMSSendRateLimited
		}
		undoKeys = append(undoKeys, key)
		return nil
	}

	if clientIP != "" {
		if err := tryIncrLimit(cache.SMSPhoneIPLimitKey(clientIP)); err != nil {
			return err
		}
	}
	if err := tryIncrLimit(cache.SMSPhoneSendUserKey(userID)); err != nil {
		return err
	}
	if err := tryIncrLimit(cache.SMSPhoneSendPhoneKey(phone)); err != nil {
		return err
	}

	code, err := generate6DigitCode()
	if err != nil {
		rollbackEarlier()
		return err
	}
	hash, err := s.hashSMSCode(userID, phone, code)
	if err != nil {
		rollbackEarlier()
		return err
	}

	otpKey := cache.SMSPhoneOTPKey(userID, phone)
	data := smsOTPStoredData{
		Phone:    phone,
		CodeHash: hash,
		Attempts: 0,
	}
	if err := c.Set(ctx, otpKey, data, smsPhoneOTPTTL); err != nil {
		rollbackEarlier()
		return err
	}

	if err := SendAliyunOTPCode(phone, code); err != nil {
		_ = c.Delete(ctx, otpKey)
		rollbackEarlier()
		s.logger.Warn("aliyun sms send failed", zap.Error(err))
		return err
	}

	return nil
}

// VerifyPhoneSMSCode verifies the OTP and binds the phone to the user.
func (s *Service) VerifyPhoneSMSCode(ctx context.Context, userID, rawPhone, code string) error {
	phone, err := NormalizeChinaPhone(rawPhone)
	if err != nil {
		return err
	}
	if len(code) != 6 {
		return errors.New("invalid code")
	}

	if err := s.userMayUsePhoneSMSGate(ctx, userID); err != nil {
		return err
	}

	c := s.getCache()
	if c == nil {
		return errors.New("SMS verification requires Redis cache")
	}

	otpKey := cache.SMSPhoneOTPKey(userID, phone)
	var stored smsOTPStoredData
	if err := c.Get(ctx, otpKey, &stored); err != nil || stored.CodeHash == "" {
		return errors.New("code expired or not found, please request a new one")
	}
	if stored.Attempts >= smsPhoneAttemptMax {
		return errors.New("too many incorrect attempts")
	}

	wantHash, err := s.hashSMSCode(userID, phone, code)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(wantHash), []byte(stored.CodeHash)) {
		stored.Attempts++
		_ = c.Set(ctx, otpKey, stored, smsPhoneOTPTTL)
		return errors.New("invalid verification code")
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
	user.Phone = phone
	user.PhoneVerifiedAt = &now
	user.PendingOAuthPhoneSMS = false
	user.UpdatedAt = now
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	_ = c.Delete(ctx, otpKey)
	s.invalidateUserCache(ctx, userID)

	return nil
}
