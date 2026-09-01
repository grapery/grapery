package service

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// phoneLoginOTPSentinel namespaces the OTP HMAC for the unauthenticated phone-login
// flow so it can never collide with the authenticated OAuth-binding OTP (keyed by userID).
const phoneLoginOTPSentinel = "phone-login"

const (
	smsPhoneLoginSendWindow       = 1 * time.Minute
	smsPhoneLoginSendMaxPerWindow = 1
	smsPhoneLoginIPWindow         = 1 * time.Minute
	smsPhoneLoginIPMaxPerWindow   = 5
)

// debugPhoneLoginBypassCode returns an explicitly configured local development
// code. It is deliberately disabled in release mode and when either environment
// variable is missing, so production keeps using the normal SMS/Redis flow.
func debugPhoneLoginBypassCode() string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("GIN_MODE")), "release") {
		return ""
	}
	enabled := strings.TrimSpace(os.Getenv("GRAPERY_DEBUG_PHONE_LOGIN_BYPASS"))
	if enabled != "1" && !strings.EqualFold(enabled, "true") {
		return ""
	}
	code := strings.TrimSpace(os.Getenv("GRAPERY_DEBUG_PHONE_LOGIN_CODE"))
	if len(code) != 6 || !smsCodeDecimal6.MatchString(code) {
		return ""
	}
	return code
}

// SendPhoneLoginSMSCode sends a login OTP to a mainland-China phone for the unauthenticated
// phone-number login flow. A single phone maps to a single account (enforced at verify time);
// here we only normalize, rate-limit (per phone + per IP) and dispatch the code.
func (s *Service) SendPhoneLoginSMSCode(ctx context.Context, rawPhone, clientIP string) error {
	log := s.logger.With(
		zap.String("flow", "phone_login_sms_send"),
		zap.String("client_ip", clientIP),
		zap.String("raw_phone_masked", utils.MaskChinaPhone(rawPhone)),
	)
	log.Info("phone login sms send: request accepted")

	phone, err := NormalizeChinaPhone(rawPhone)
	if err != nil {
		log.Warn("phone login sms send: invalid phone format", zap.Error(err))
		return err
	}
	log = log.With(zap.String("phone_masked", utils.MaskChinaPhone(phone)))
	if debugPhoneLoginBypassCode() != "" {
		log.Warn("phone login sms send: skipped by explicit development bypass")
		return nil
	}

	c := s.getCache()
	if c == nil {
		log.Warn("phone login sms send: redis cache unavailable")
		return errors.New("SMS verification requires Redis cache")
	}

	// Rate limits: phone once per window, IP a few times per window. Roll back the phone
	// counter if the IP dimension trips so a blocked IP does not burn the phone quota.
	phoneKey := cache.SMSPhoneLoginSendPhoneKey(phone)
	n, err := c.Incr(ctx, phoneKey)
	if err != nil {
		log.Error("phone login sms send: incr phone limit failed", zap.Error(err))
		return err
	}
	if n == 1 {
		_ = c.Expire(ctx, phoneKey, smsPhoneLoginSendWindow)
	}
	if n > smsPhoneLoginSendMaxPerWindow {
		log.Warn("phone login sms send: rate limited (phone)")
		return ErrSMSSendRateLimited
	}

	if clientIP != "" {
		ipKey := cache.SMSPhoneLoginIPLimitKey(clientIP)
		ipN, errIP := c.Incr(ctx, ipKey)
		if errIP != nil {
			_, _ = c.Decr(ctx, phoneKey)
			log.Error("phone login sms send: incr ip limit failed", zap.Error(errIP))
			return errIP
		}
		if ipN == 1 {
			_ = c.Expire(ctx, ipKey, smsPhoneLoginIPWindow)
		}
		if ipN > smsPhoneLoginIPMaxPerWindow {
			_, _ = c.Decr(ctx, phoneKey)
			log.Warn("phone login sms send: rate limited (ip)")
			return ErrSMSSendRateLimited
		}
	}

	code, err := generate6DigitCode()
	if err != nil {
		_, _ = c.Decr(ctx, phoneKey)
		log.Error("phone login sms send: generate otp failed", zap.Error(err))
		return err
	}
	hash, err := s.hashSMSCode(phoneLoginOTPSentinel, phone, code)
	if err != nil {
		_, _ = c.Decr(ctx, phoneKey)
		log.Error("phone login sms send: hash otp failed", zap.Error(err))
		return err
	}

	otpKey := cache.SMSPhoneLoginOTPKey(phone)
	data := smsOTPStoredData{
		Phone:    phone,
		CodeHash: hash,
		Attempts: 0,
	}
	if err := c.Set(ctx, otpKey, data, smsPhoneOTPTTL); err != nil {
		_, _ = c.Decr(ctx, phoneKey)
		log.Error("phone login sms send: redis set otp failed", zap.Error(err))
		return err
	}

	log.Info("phone login sms send: otp stored, calling aliyun")
	if err := SendAliyunOTPCode(phone, code); err != nil {
		_ = c.Delete(ctx, otpKey)
		_, _ = c.Decr(ctx, phoneKey)
		log.Warn("phone login sms send: aliyun dispatch failed", zap.Error(err))
		return err
	}

	log.Info("phone login sms send: completed successfully")
	return nil
}

// LoginWithPhoneSMS verifies the login OTP and signs the user in by phone number.
// If no account owns the phone yet, a new account is auto-created and bound to it.
// A phone number is unique across all accounts (DB unique index + UserByPhone lookup here),
// guaranteeing one phone ↔ one user for phone/WeChat/Apple alike.
func (s *Service) LoginWithPhoneSMS(ctx context.Context, rawPhone, code string, loginInfo *LoginInfo) (*LoginResponse, error) {
	log := s.logger.With(
		zap.String("flow", "phone_login_sms_verify"),
		zap.String("raw_phone_masked", utils.MaskChinaPhone(rawPhone)),
	)
	log.Info("phone login verify: request accepted")

	phone, err := NormalizeChinaPhone(rawPhone)
	if err != nil {
		log.Warn("phone login verify: invalid phone format", zap.Error(err))
		return nil, err
	}
	log = log.With(zap.String("phone_masked", utils.MaskChinaPhone(phone)))

	code = strings.TrimSpace(code)
	if len(code) != 6 || !smsCodeDecimal6.MatchString(code) {
		log.Warn("phone login verify: invalid code format", zap.Int("code_len", len(code)))
		return nil, errors.New("verification code must be 6 digits")
	}

	if err := s.verifyPhoneLoginCode(ctx, phone, code, log); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	user, err := s.repo.UserByPhone(ctx, phone)
	if err != nil {
		log.Error("phone login verify: lookup phone owner failed", zap.Error(err))
		return nil, err
	}

	isNewUser := false
	if user == nil {
		// First sign-in with this phone → auto-register a phone-only account.
		blocked, errBlk := s.repo.IsAccountReRegistrationBlocked(ctx, "", phone)
		if errBlk != nil {
			log.Error("phone login verify: re-registration block check failed", zap.Error(errBlk))
			return nil, errBlk
		}
		if blocked {
			return nil, errors.New("this phone number cannot be used to register within 30 days after account deletion")
		}

		user, err = s.createPhoneLoginUser(ctx, phone, now)
		if err != nil {
			log.Error("phone login verify: auto-register failed", zap.Error(err))
			return nil, err
		}
		isNewUser = true
		log.Info("phone login verify: new user auto-registered", zap.String("user_id", user.ID))
	} else {
		if user.Status != string(common.StatusActive) && user.Status != string(common.StatusPendingDeletion) {
			return nil, fmt.Errorf("account is %s", user.Status)
		}
		user.LastLoginAt = &now
		if user.PhoneVerifiedAt == nil || *user.PhoneVerifiedAt <= 0 {
			user.PhoneVerifiedAt = &now
		}
		user.PendingOAuthPhoneSMS = false
		user.UpdatedAt = now
		if err := s.repo.UpdateUser(ctx, user); err != nil {
			log.Warn("phone login verify: update existing user failed", zap.Error(err))
			// 不阻塞登录流程
		}
	}

	accessToken, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		log.Error("phone login verify: generate access token failed", zap.Error(err))
		return nil, errors.New("failed to generate authentication token")
	}
	deviceID := ""
	if loginInfo != nil {
		deviceID = normalizeAuthDeviceID(loginInfo.DeviceID)
	}
	refreshToken, err := auth.GenerateRefreshTokenForDevice(user.ID, deviceID)
	if err != nil {
		log.Error("phone login verify: generate refresh token failed", zap.Error(err))
		return nil, errors.New("failed to generate refresh token")
	}

	_ = s.AttachPhoneVerificationRequirement(ctx, user)
	s.cacheUser(ctx, user)
	s.invalidateUserCache(ctx, user.ID)

	if s.metrics != nil {
		if isNewUser {
			s.metrics.RecordUserRegistration("phone")
		}
		s.metrics.RecordUserLogin("phone")
	}
	if s.userStatsService != nil {
		_ = s.userStatsService.RecordActiveUser(ctx, user.ID)
	}

	log.Info("phone login verify: completed successfully",
		zap.String("user_id", user.ID),
		zap.Bool("is_new_user", isNewUser),
	)

	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    24 * 3600,
	}, nil
}

func (s *Service) verifyPhoneLoginCode(ctx context.Context, phone, code string, log *zap.Logger) error {
	if bypassCode := debugPhoneLoginBypassCode(); bypassCode != "" && hmac.Equal([]byte(code), []byte(bypassCode)) {
		log.Warn("phone login verify: accepted explicit development bypass")
		return nil
	}

	c := s.getCache()
	if c == nil {
		log.Warn("phone login verify: redis cache unavailable")
		return errors.New("SMS verification requires Redis cache")
	}

	otpKey := cache.SMSPhoneLoginOTPKey(phone)
	var stored smsOTPStoredData
	if err := c.Get(ctx, otpKey, &stored); err != nil || stored.CodeHash == "" {
		log.Warn("phone login verify: otp missing or expired", zap.Error(err))
		return errors.New("code expired or not found, please request a new one")
	}
	if stored.Attempts >= smsPhoneAttemptMax {
		log.Warn("phone login verify: too many attempts", zap.Int("attempts", stored.Attempts))
		return errors.New("too many incorrect attempts")
	}

	wantHash, err := s.hashSMSCode(phoneLoginOTPSentinel, phone, code)
	if err != nil {
		log.Error("phone login verify: hash otp failed", zap.Error(err))
		return err
	}
	if !hmac.Equal([]byte(wantHash), []byte(stored.CodeHash)) {
		stored.Attempts++
		_ = c.Set(ctx, otpKey, stored, smsPhoneOTPTTL)
		log.Warn("phone login verify: incorrect code", zap.Int("attempts", stored.Attempts))
		return errors.New("invalid verification code")
	}

	// Code is correct — consume it before any account mutation.
	_ = c.Delete(ctx, otpKey)
	return nil
}

// createPhoneLoginUser persists a new phone-verified account (settings + free membership)
// in a single transaction, mirroring email/OAuth registration.
func (s *Service) createPhoneLoginUser(ctx context.Context, phone string, now int64) (*domain.User, error) {
	userID := uuid.New().String()
	username := "u" + strings.ReplaceAll(userID, "-", "")
	displayName := "用户" + phone[len(phone)-4:]
	verifiedAt := now

	user := &domain.User{
		BaseModel: common.BaseModel{
			ID:        userID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		SocialStats: common.SocialStats{
			Followers: 0,
			Following: 0,
		},
		Username:             username,
		DisplayName:          displayName,
		Phone:                phone,
		PhoneVerifiedAt:      &verifiedAt,
		PendingOAuthPhoneSMS: false,
		Status:               string(common.StatusActive),
		EmailVerified:        false,
		ReferralCode:         GenerateUserReferralCode(userID),
		LastLoginAt:          &now,
	}
	ApplyNewUserWelcomePoints(user)

	err := s.repo.WithTransaction(ctx, func(tx domain.Repository) error {
		if err := tx.CreateUser(ctx, user); err != nil {
			if regErr := registrationCreateUserError(err); regErr != nil {
				return regErr
			}
			return errors.New("failed to create user account")
		}

		settings := &domain.UserSettings{
			BaseModel: common.BaseModel{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:               user.ID,
			Language:             "zh",
			Theme:                "auto",
			EmailNotifications:   true,
			PushNotifications:    true,
			ShowAdultContent:     false,
			ProfileVisibility:    "public",
			AllowComments:        true,
			AllowMessages:        true,
			ShowOnlineStatus:     true,
			ShowPublicStories:    true,
			ShowPublicFragments:  true,
			ShowPublicBookmarks:  true,
			NotificationSettings: "{}",
		}
		if err := tx.CreateUserSettings(ctx, settings); err != nil {
			s.logger.Warn("phone login auto-register: create settings failed", zap.Error(err))
			// 不阻塞注册
		}

		membership := NewUserWelcomeMembership(user.ID, now)
		if err := tx.CreateMembership(ctx, membership); err != nil {
			s.logger.Warn("phone login auto-register: create membership failed", zap.Error(err))
			// 不阻塞注册
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}
