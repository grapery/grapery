package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
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

// ErrSMSSendRateLimited indicates send-sms throttling per IP/user/phone window is exceeded (HTTP handlers map to CodeRateLimitExceed).
var ErrSMSSendRateLimited = errors.New("verification code can only be sent once per minute per account, phone number, and IP; please try again later")

var mainlandChinaMobile11 = regexp.MustCompile(`^1[3-9]\d{9}$`)

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

// NormalizeChinaPhone trims input and returns domestic 11-digit mainland mobile (no +86), matching ^1[3-9]\d{9}$, or error.
func NormalizeChinaPhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "+86")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", errors.New("china mobile number must contain only digits")
		}
	}
	if len(s) != 11 {
		return "", errors.New("china mobile number must be 11 digits")
	}
	if !mainlandChinaMobile11.MatchString(s) {
		return "", errors.New("invalid mainland china mobile number")
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
	log := s.logger.With(
		zap.String("flow", "oauth_phone_sms_send"),
		zap.String("user_id", userID),
		zap.String("client_ip", clientIP),
		zap.String("raw_phone_masked", utils.MaskChinaPhone(rawPhone)),
	)
	log.Info("phone sms send: request accepted")

	c := s.getCache()
	if c == nil {
		log.Warn("phone sms send: redis cache unavailable")
		return errors.New("SMS verification requires Redis cache")
	}

	phone, err := NormalizeChinaPhone(rawPhone)
	if err != nil {
		log.Warn("phone sms send: invalid phone format", zap.Error(err))
		return err
	}
	log = log.With(zap.String("phone_masked", utils.MaskChinaPhone(phone)))

	other, err := s.repo.UserByPhone(ctx, phone)
	if err != nil {
		log.Error("phone sms send: lookup phone owner failed", zap.Error(err))
		return err
	}
	if other != nil && other.ID != userID {
		log.Warn("phone sms send: phone bound to another account", zap.String("other_user_id", other.ID))
		return errors.New("this phone number is already bound to another account")
	}

	if err := s.userMayUsePhoneSMSGate(ctx, userID); err != nil {
		log.Warn("phone sms send: gate check failed", zap.Error(err))
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
			return ErrSMSSendRateLimited
		}
		undoKeys = append(undoKeys, key)
		return nil
	}

	if clientIP != "" {
		if err := tryIncrLimit(cache.SMSPhoneIPLimitKey(clientIP)); err != nil {
			log.Warn("phone sms send: rate limited (ip)", zap.Error(err))
			return err
		}
	}
	if err := tryIncrLimit(cache.SMSPhoneSendUserKey(userID)); err != nil {
		log.Warn("phone sms send: rate limited (user)", zap.Error(err))
		return err
	}
	if err := tryIncrLimit(cache.SMSPhoneSendPhoneKey(phone)); err != nil {
		log.Warn("phone sms send: rate limited (phone)", zap.Error(err))
		return err
	}

	log.Debug("phone sms send: rate limit passed, generating otp")

	code, err := generate6DigitCode()
	if err != nil {
		rollbackEarlier()
		log.Error("phone sms send: generate otp failed", zap.Error(err))
		return err
	}
	hash, err := s.hashSMSCode(userID, phone, code)
	if err != nil {
		rollbackEarlier()
		log.Error("phone sms send: hash otp failed", zap.Error(err))
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
		log.Error("phone sms send: redis set otp failed", zap.Error(err))
		return err
	}

	log.Info("phone sms send: otp stored in redis, calling aliyun")
	if err := SendAliyunOTPCode(phone, code); err != nil {
		_ = c.Delete(ctx, otpKey)
		rollbackEarlier()
		log.Warn("phone sms send: aliyun dispatch failed", zap.Error(err))
		return err
	}

	log.Info("phone sms send: completed successfully")
	return nil
}

// smsCodeDecimal6 validates a 6-digit ASCII decimal OTP.
var smsCodeDecimal6 = regexp.MustCompile(`^[0-9]{6}$`)

// VerifyPhoneSMSCode verifies the OTP and binds the phone to the user.
func (s *Service) VerifyPhoneSMSCode(ctx context.Context, userID, rawPhone, code string) error {
	log := s.logger.With(
		zap.String("flow", "oauth_phone_sms_verify"),
		zap.String("user_id", userID),
		zap.String("raw_phone_masked", utils.MaskChinaPhone(rawPhone)),
	)
	log.Info("phone sms verify: request accepted")

	phone, err := NormalizeChinaPhone(rawPhone)
	if err != nil {
		log.Warn("phone sms verify: invalid phone format", zap.Error(err))
		return err
	}
	log = log.With(zap.String("phone_masked", utils.MaskChinaPhone(phone)))

	code = strings.TrimSpace(code)
	if len(code) != 6 || !smsCodeDecimal6.MatchString(code) {
		log.Warn("phone sms verify: invalid code format", zap.Int("code_len", len(code)))
		return errors.New("verification code must be 6 digits")
	}

	if err := s.userMayUsePhoneSMSGate(ctx, userID); err != nil {
		log.Warn("phone sms verify: gate check failed", zap.Error(err))
		return err
	}

	c := s.getCache()
	if c == nil {
		log.Warn("phone sms verify: redis cache unavailable")
		return errors.New("SMS verification requires Redis cache")
	}

	otpKey := cache.SMSPhoneOTPKey(userID, phone)
	var stored smsOTPStoredData
	if err := c.Get(ctx, otpKey, &stored); err != nil || stored.CodeHash == "" {
		log.Warn("phone sms verify: otp missing or expired", zap.Error(err))
		return errors.New("code expired or not found, please request a new one")
	}
	if stored.Attempts >= smsPhoneAttemptMax {
		log.Warn("phone sms verify: too many attempts", zap.Int("attempts", stored.Attempts))
		return errors.New("too many incorrect attempts")
	}

	wantHash, err := s.hashSMSCode(userID, phone, code)
	if err != nil {
		log.Error("phone sms verify: hash otp failed", zap.Error(err))
		return err
	}
	if !hmac.Equal([]byte(wantHash), []byte(stored.CodeHash)) {
		stored.Attempts++
		_ = c.Set(ctx, otpKey, stored, smsPhoneOTPTTL)
		log.Warn("phone sms verify: incorrect code", zap.Int("attempts", stored.Attempts))
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

	log.Info("phone sms verify: completed successfully")
	return nil
}
