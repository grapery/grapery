package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/email"
	"go.uber.org/zap"
)

// 缓存过期时间常量
const (
	userCacheTTL          = 30 * time.Minute
	passwordResetTokenTTL = 15 * time.Minute
	emailVerifyTokenTTL   = 24 * time.Hour
	emailVerifyCodeTTL    = 10 * time.Minute

	emailVerifyMaxAttempts     = 5
	emailVerifySendLimitTTL    = 1 * time.Minute
	emailVerifySendLimitMax    = 1 // per email per minute
	emailVerifyIPLimitTTL      = 1 * time.Minute
	emailVerifyIPLimitMax      = 5 // per IP per minute

	// 通用实体缓存过期时间
	entityCacheTTL      = 30 * time.Minute // 单个实体缓存（用户、角色、群组、故事板等）
	listCacheTTL        = 10 * time.Minute // 列表缓存（较短，因为数据变化频繁）
	styleConfigCacheTTL = 1 * time.Hour    // 风格配置缓存（变化较少）
	activityCacheTTL    = 5 * time.Minute  // 活动流缓存（变化频繁）
	commentCacheTTL     = 15 * time.Minute // 评论缓存
	groupMemberCacheTTL = 15 * time.Minute // 群组成员缓存
)

// EmailVerificationSendRequest send verification code request
type EmailVerificationSendRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// EmailVerificationConfirmRequest verify email with code
type EmailVerificationConfirmRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type emailVerifyCodeData struct {
	Email     string `json:"email"`
	CodeHash  string `json:"codeHash"`
	CreatedAt int64  `json:"createdAt"`
	Attempts  int    `json:"attempts"`
}

func (s *Service) emailVerifySecret() string {
	if v := os.Getenv("EMAIL_VERIFICATION_SECRET"); v != "" {
		return v
	}
	// Fallback: JWT secret if present.
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return v
	}
	// Last resort (dev): constant, but should not be used in production.
	return "email-verify-dev-secret"
}

func (s *Service) hashEmailVerificationCode(emailAddr, code string) string {
	secret := s.emailVerifySecret()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strings.ToLower(emailAddr)))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generate6DigitCode() (string, error) {
	// crypto/rand to avoid predictable codes
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// 0..999999
	n := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % 1000000
	return fmt.Sprintf("%06d", n), nil
}

func (s *Service) emailVerifyRateLimit(ctx context.Context, emailAddr, ip string) error {
	c := s.getCache()
	if c == nil {
		return nil // allow if no cache (degraded)
	}

	if emailAddr != "" {
		emailKey := cache.EmailVerifySendLimitKey(strings.ToLower(emailAddr))
		n, err := c.Incr(ctx, emailKey)
		if err == nil && n == 1 {
			_ = c.Expire(ctx, emailKey, emailVerifySendLimitTTL)
		}
		if err == nil && n > emailVerifySendLimitMax {
			return errors.New("too many requests, please try again later")
		}
	}

	if ip != "" {
		ipKey := cache.EmailVerifyIPLimitKey(ip)
		n, err := c.Incr(ctx, ipKey)
		if err == nil && n == 1 {
			_ = c.Expire(ctx, ipKey, emailVerifyIPLimitTTL)
		}
		if err == nil && n > emailVerifyIPLimitMax {
			return errors.New("too many requests, please try again later")
		}
	}

	return nil
}

// SendEmailVerificationCode sends a 6-digit verification code to the email if user exists and not verified.
// Always returns nil for non-existent emails (anti-enumeration).
func (s *Service) SendEmailVerificationCode(ctx context.Context, req *EmailVerificationSendRequest, ip string) error {
	// Temporary override: allow disabling email sending (e.g., during SMTP instability).
	// This keeps the API responsive and avoids request timeouts.
	if v := strings.TrimSpace(os.Getenv("DISABLE_EMAIL_VERIFICATION_SEND")); v == "1" || strings.EqualFold(v, "true") {
		s.logger.Warn("email verification send disabled by env", zap.String("email", req.Email))
		return nil
	}
	// Rate limit (best-effort)
	if err := s.emailVerifyRateLimit(ctx, req.Email, ip); err != nil {
		return err
	}

	user, err := s.repo.UserByEmail(ctx, req.Email)
	if err != nil || user == nil {
		// Silent success to prevent enumeration
		return nil
	}
	if user.EmailVerified {
		return nil
	}

	code, err := generate6DigitCode()
	if err != nil {
		s.logger.Error("failed to generate verification code", zap.Error(err))
		return errors.New("failed to generate verification code")
	}

	data := &emailVerifyCodeData{
		Email:     strings.ToLower(req.Email),
		CodeHash:  s.hashEmailVerificationCode(req.Email, code),
		CreatedAt: time.Now().Unix(),
		Attempts:  0,
	}

	c := s.getCache()
	if c != nil {
		_ = c.Set(ctx, cache.EmailVerifyCodeKey(strings.ToLower(req.Email)), data, emailVerifyCodeTTL)
	}

	username := user.DisplayName
	if username == "" {
		username = user.Username
	}
	if err := email.VerificationCodeEmail([]string{user.Email}, username, code, int(emailVerifyCodeTTL.Minutes())); err != nil {
		s.logger.Error("failed to send verification code email", zap.Error(err))
		// Relaxed behavior: do not fail the request if email sending is unavailable.
		// Caller can retry later; verification code remains in cache if available.
		return nil
	}
	return nil
}

// VerifyEmailByCode verifies the email using the stored code, and flips user.EmailVerified to true.
func (s *Service) VerifyEmailByCode(ctx context.Context, req *EmailVerificationConfirmRequest) error {
	c := s.getCache()
	if c == nil {
		return errors.New("cache not available")
	}

	key := cache.EmailVerifyCodeKey(strings.ToLower(req.Email))
	var data emailVerifyCodeData
	if err := c.Get(ctx, key, &data); err != nil {
		return errors.New("invalid or expired code")
	}

	if data.Attempts >= emailVerifyMaxAttempts {
		_ = c.Delete(ctx, key)
		return errors.New("too many attempts")
	}

	expected := s.hashEmailVerificationCode(req.Email, req.Code)
	if !hmac.Equal([]byte(expected), []byte(data.CodeHash)) {
		data.Attempts++
		_ = c.Set(ctx, key, &data, emailVerifyCodeTTL)
		return errors.New("invalid or expired code")
	}

	user, err := s.repo.UserByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return errors.New("invalid code")
	}

	if !user.EmailVerified {
		user.EmailVerified = true
		user.UpdatedAt = time.Now().Unix()
		if err := s.repo.UpdateUser(ctx, user); err != nil {
			return errors.New("failed to verify email")
		}
		s.invalidateUserCache(ctx, user.ID)
	}

	_ = c.Delete(ctx, key)
	return nil
}

// PasswordResetData 密码重置数据结构
type PasswordResetData struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"createdAt"`
}

// getCache 获取缓存实例（类型安全）
func (s *Service) getCache() cache.Cache {
	if s.cache == nil {
		return nil
	}
	if c, ok := s.cache.(cache.Cache); ok {
		return c
	}
	return nil
}

// cacheUser 缓存用户信息
func (s *Service) cacheUser(ctx context.Context, user *domain.User) {
	c := s.getCache()
	if c == nil {
		return
	}

	key := cache.UserKey(user.ID)
	if err := c.Set(ctx, key, user, userCacheTTL); err != nil {
		s.logger.Warn("failed to cache user", zap.String("userID", user.ID), zap.Error(err))
	}
}

// invalidateUserCache 清除用户缓存
func (s *Service) invalidateUserCache(ctx context.Context, userID string) {
	c := s.getCache()
	if c == nil {
		return
	}

	key := cache.UserKey(userID)
	if err := c.Delete(ctx, key); err != nil {
		s.logger.Warn("failed to invalidate user cache", zap.String("userID", userID), zap.Error(err))
	}
}

// storePasswordResetToken 存储密码重置 token
func (s *Service) storePasswordResetToken(ctx context.Context, token string, data *PasswordResetData) error {
	c := s.getCache()
	if c == nil {
		return errors.New("cache not available")
	}

	key := cache.PasswordResetKey(token)
	return c.Set(ctx, key, data, passwordResetTokenTTL)
}

// getPasswordResetToken 获取密码重置 token 数据
func (s *Service) getPasswordResetToken(ctx context.Context, token string) (*PasswordResetData, error) {
	c := s.getCache()
	if c == nil {
		return nil, errors.New("cache not available")
	}

	key := cache.PasswordResetKey(token)
	var data PasswordResetData
	if err := c.Get(ctx, key, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// deletePasswordResetToken 删除密码重置 token
func (s *Service) deletePasswordResetToken(ctx context.Context, token string) {
	c := s.getCache()
	if c == nil {
		return
	}

	key := cache.PasswordResetKey(token)
	if err := c.Delete(ctx, key); err != nil {
		s.logger.Warn("failed to delete password reset token", zap.Error(err))
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username       string `json:"username" binding:"required,min=3,max=50"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	DisplayName    string `json:"displayName" binding:"required,min=1,max=100"`
	DateOfBirth    string `json:"dateOfBirth,omitempty"` // YYYY-MM-DD
	AgreeTerms     bool   `json:"agreeTerms" binding:"required"`
	InvitationCode string `json:"invitationCode,omitempty"` // 邀请码（可选）
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User         *domain.User `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresIn    int64        `json:"expiresIn"` // 秒
}

// PasswordResetRequest 密码重置请求
type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// PasswordResetConfirm 密码重置确认
type PasswordResetConfirm struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// LoginInfo 登录时的客户端信息
type LoginInfo struct {
	IPAddress string // 客户端 IP 地址
	Location  string // 地理位置（可选，可通过 IP 地址查询）
	Device    string // 设备类型
	OS        string // 操作系统
	Browser   string // 浏览器
	UserAgent string // 完整的 User-Agent 字符串
}

// Register 用户注册
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*LoginResponse, error) {
	s.logger.Info("user registration attempt", zap.String("email", req.Email))

	// 验证用户协议
	if !req.AgreeTerms {
		return nil, errors.New("must agree to terms and conditions")
	}

	// 验证邀请码（如果提供）
	if req.InvitationCode != "" {
		if err := s.repo.ValidateInvitationCode(ctx, req.InvitationCode); err != nil {
			s.logger.Warn("invalid invitation code", zap.String("code", req.InvitationCode), zap.Error(err))
			return nil, errors.New("invalid or expired invitation code")
		}
	}

	// 检查用户名是否已存在
	existingUser, err := s.repo.UserByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, errors.New("username already taken")
	}

	// 检查邮箱是否已存在
	existingUser, err = s.repo.UserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("email already registered")
	}

	// 加密密码
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, errors.New("failed to process password")
	}

	// 解析生日
	var dob *int64
	if req.DateOfBirth != "" {
		parsed, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err == nil {
			ts := parsed.Unix()
			dob = &ts
		}
	}

	// 创建用户
	user := &domain.User{
		ID:            uuid.New().String(),
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  passwordHash,
		DisplayName:   req.DisplayName,
		DateOfBirth:   dob,
		Status:        "active",
		EmailVerified: false,
		Followers:     0,
		Following:     0,
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, errors.New("failed to create user account")
	}

	// 标记邀请码已使用（如果提供了邀请码）
	if req.InvitationCode != "" {
		if err := s.repo.UseInvitationCode(ctx, req.InvitationCode, user.ID); err != nil {
			s.logger.Warn("failed to mark invitation code as used", zap.String("code", req.InvitationCode), zap.Error(err))
			// 不阻塞注册流程，但记录警告
		}
	}

	// 创建默认用户设置
	settings := &domain.UserSettings{
		ID:                 uuid.New().String(),
		UserID:             user.ID,
		Language:           "en",
		Theme:              "auto",
		EmailNotifications: true,
		PushNotifications:  true,
		ShowAdultContent:   false,
		ProfileVisibility:  "public",
		AllowComments:      true,
		AllowMessages:      true,
		ShowOnlineStatus:   true,
		UpdatedAt:          time.Now().Unix(),
	}

	if err := s.repo.CreateUserSettings(ctx, settings); err != nil {
		s.logger.Warn("failed to create user settings", zap.Error(err))
		// 不阻塞注册流程
	}

	// 创建默认会员信息（免费会员）
	membership := &domain.Membership{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		Tier:         "free",
		Status:       "active",
		StartDate:    time.Now().Unix(),
		AutoRenew:    false,
		TokenQuota:   10000, // 免费配额
		TokenUsed:    0,
		StorageQuota: 1024 * 1024 * 100, // 100MB
		StorageUsed:  0,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	if err := s.repo.CreateMembership(ctx, membership); err != nil {
		s.logger.Warn("failed to create membership", zap.Error(err))
		// 不阻塞注册流程
	}

	// 生成 Token
	accessToken, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		s.logger.Error("failed to generate access token", zap.Error(err))
		return nil, errors.New("failed to generate authentication token")
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID)
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.Error(err))
		return nil, errors.New("failed to generate refresh token")
	}

	// 缓存用户信息
	s.cacheUser(ctx, user)

	// Record metrics
	if s.metrics != nil {
		source := "web"
		if req.InvitationCode != "" {
			source = "invitation"
		}
		s.metrics.RecordUserRegistration(source)
		// 更新用户总数
		s.metrics.UserCount.Inc()
	}

	// 记录活跃用户（注册也算活跃）
	if s.userStatsService != nil {
		_ = s.userStatsService.RecordActiveUser(ctx, user.ID)
	}

	s.logger.Info("user registered successfully",
		zap.String("userID", user.ID),
		zap.String("username", user.Username))

	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    24 * 3600, // 24小时
	}, nil
}

// Login 用户登录
// loginInfo 包含登录时的客户端信息（IP、User-Agent 等），可选
func (s *Service) Login(ctx context.Context, req *LoginRequest, loginInfo *LoginInfo) (*LoginResponse, error) {
	s.logger.Info("user login attempt", zap.String("email", req.Email))

	// 查找用户
	user, err := s.repo.UserByEmail(ctx, req.Email)
	if err != nil || user == nil {
		s.logger.Warn("user not found", zap.String("email", req.Email))
		return nil, errors.New("invalid email or password")
	}

	// 验证密码
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		s.logger.Warn("invalid password", zap.String("email", req.Email))
		return nil, errors.New("invalid email or password")
	}

	// 检查用户状态
	if user.Status != "active" {
		return nil, fmt.Errorf("account is %s", user.Status)
	}

	// 更新最后登录时间
	now := time.Now().Unix()
	user.LastLoginAt = &now
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Warn("failed to update last login time", zap.Error(err))
		// 不阻塞登录流程
	}

	// 生成 Token
	accessToken, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		s.logger.Error("failed to generate access token", zap.Error(err))
		return nil, errors.New("failed to generate authentication token")
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID)
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.Error(err))
		return nil, errors.New("failed to generate refresh token")
	}

	// 缓存用户信息
	s.cacheUser(ctx, user)

	// Record metrics
	if s.metrics != nil {
		s.metrics.RecordUserLogin("password")
	}

	// 记录活跃用户
	if s.userStatsService != nil {
		_ = s.userStatsService.RecordActiveUser(ctx, user.ID)
	}

	s.logger.Info("user logged in successfully",
		zap.String("userID", user.ID),
		zap.String("username", user.Username))

	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    24 * 3600, // 24小时
	}, nil
}

// RequestPasswordReset 请求密码重置
func (s *Service) RequestPasswordReset(ctx context.Context, req *PasswordResetRequest) error {
	s.logger.Info("password reset requested", zap.String("email", req.Email))

	// 查找用户
	user, err := s.repo.UserByEmail(ctx, req.Email)
	if err != nil || user == nil {
		// 为了安全，不暴露用户是否存在
		s.logger.Warn("password reset requested for non-existent email", zap.String("email", req.Email))
		return nil // 返回成功，但不发送邮件
	}

	// 生成重置 Token（使用 UUID）
	resetToken := uuid.New().String()

	// 存储到 Redis，15分钟过期
	resetData := &PasswordResetData{
		UserID:    user.ID,
		Email:     user.Email,
		CreatedAt: time.Now().Unix(),
	}

	if err := s.storePasswordResetToken(ctx, resetToken, resetData); err != nil {
		s.logger.Error("failed to store password reset token", zap.Error(err))
		// 如果缓存不可用，记录日志但仍然继续（降级处理）
		// 在生产环境中，可能需要考虑其他存储方式
	}

	s.logger.Info("password reset token generated", zap.String("userID", user.ID))

	// Send reset email (do not leak whether email exists; failures are logged but response remains success).
	resetBaseURL := os.Getenv("PASSWORD_RESET_BASE_URL")
	if resetBaseURL == "" {
		resetBaseURL = "https://rankquantity.xyz/reset-password"
	}
	resetURL := fmt.Sprintf("%s?token=%s", resetBaseURL, resetToken)
	username := user.DisplayName
	if username == "" {
		username = user.Username
	}
	if err := email.PasswordResetEmail([]string{user.Email}, username, resetURL); err != nil {
		s.logger.Error("failed to send password reset email", zap.Error(err))
		// Keep silent to client; also avoid failing the request on SMTP issues.
	}

	return nil
}

// ResetPassword 重置密码
func (s *Service) ResetPassword(ctx context.Context, req *PasswordResetConfirm) error {
	s.logger.Info("password reset confirmation", zap.String("token", req.Token))

	// 从 Redis 获取重置信息
	resetData, err := s.getPasswordResetToken(ctx, req.Token)
	if err != nil {
		s.logger.Warn("invalid or expired password reset token", zap.String("token", req.Token), zap.Error(err))
		return errors.New("invalid or expired reset token")
	}

	// 验证 token 是否过期（双重检查）
	if time.Since(time.Unix(resetData.CreatedAt, 0)) > passwordResetTokenTTL {
		s.logger.Warn("password reset token expired", zap.String("token", req.Token))
		s.deletePasswordResetToken(ctx, req.Token)
		return errors.New("reset token has expired")
	}

	// 获取用户
	user, err := s.repo.UserByID(ctx, resetData.UserID)
	if err != nil || user == nil {
		s.logger.Error("user not found for reset", zap.String("userID", resetData.UserID))
		return errors.New("user not found")
	}

	// 验证邮箱匹配（额外安全检查）
	if user.Email != resetData.Email {
		s.logger.Error("email mismatch in password reset",
			zap.String("userID", resetData.UserID),
			zap.String("expected", resetData.Email),
			zap.String("actual", user.Email))
		return errors.New("invalid reset token")
	}

	// 加密新密码
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash new password", zap.Error(err))
		return errors.New("failed to process new password")
	}

	// 更新密码
	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("failed to update password", zap.Error(err))
		return errors.New("failed to update password")
	}

	// 删除重置 Token
	s.deletePasswordResetToken(ctx, req.Token)

	// 清除用户缓存
	s.invalidateUserCache(ctx, user.ID)

	s.logger.Info("password reset successfully", zap.String("userID", user.ID))

	return nil
}

// ChangePassword 修改密码（已登录用户）
func (s *Service) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error {
	s.logger.Info("password change request", zap.String("userID", userID))

	// 获取用户
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	// 验证旧密码
	if !auth.CheckPassword(req.OldPassword, user.PasswordHash) {
		s.logger.Warn("invalid old password", zap.String("userID", userID))
		return errors.New("invalid old password")
	}

	// 加密新密码
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash new password", zap.Error(err))
		return errors.New("failed to process new password")
	}

	// 更新密码
	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("failed to update password", zap.Error(err))
		return errors.New("failed to update password")
	}

	// 清除用户缓存
	s.invalidateUserCache(ctx, userID)

	s.logger.Info("password changed successfully", zap.String("userID", userID))

	return nil
}

// RefreshToken 刷新访问令牌
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// 解析刷新令牌
	claims, err := auth.ParseToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// 获取用户
	user, err := s.repo.UserByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	// 检查用户状态
	if user.Status != "active" {
		return nil, fmt.Errorf("account is %s", user.Status)
	}

	// 生成新的访问令牌
	accessToken, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	// 生成新的刷新令牌
	newRefreshToken, err := auth.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    24 * 3600,
	}, nil
}
