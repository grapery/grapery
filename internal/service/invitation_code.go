package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateInvitationCodeRequest 创建邀请码请求
type CreateInvitationCodeRequest struct {
	MaxUses     int    `json:"maxUses"`     // 最大使用次数（0表示无限制）
	ExpiresAt   int64  `json:"expiresAt"`   // 过期时间（Unix时间戳，0表示永不过期）
	Description string `json:"description"` // 描述信息
}

// UpdateInvitationCodeRequest 更新邀请码请求
type UpdateInvitationCodeRequest struct {
	IsActive    *bool   `json:"isActive"`
	MaxUses     *int    `json:"maxUses"`
	ExpiresAt   *int64  `json:"expiresAt"`
	Description *string `json:"description"`
}

// GenerateInvitationCode 生成随机邀请码
func generateInvitationCode() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:16] // 取前16个字符
}

// CreateInvitationCode 创建邀请码
func (s *Service) CreateInvitationCode(ctx context.Context, userID string, req CreateInvitationCodeRequest) (*domain.InvitationCode, error) {
	s.logger.Info("creating invitation code",
		zap.String("userID", userID),
		zap.Int("maxUses", req.MaxUses),
	)

	// 生成唯一邀请码
	var code string
	for {
		code = generateInvitationCode()
		// 检查是否已存在
		_, err := s.repo.GetInvitationCodeByCode(ctx, code)
		if err == domain.ErrNotFound {
			break // 找到唯一码
		}
		if err != nil {
			return nil, errors.New("failed to generate invitation code")
		}
		// 如果已存在，继续生成
	}

	now := time.Now().Unix()
	invCode := &domain.InvitationCode{
		ID:          uuid.New().String(),
		Code:        code,
		CreatedBy:   userID,
		IsActive:    true,
		MaxUses:     req.MaxUses,
		CurrentUses: 0,
		ExpiresAt:   req.ExpiresAt,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateInvitationCode(ctx, invCode); err != nil {
		s.logger.Error("failed to create invitation code", zap.Error(err))
		return nil, errors.New("failed to create invitation code")
	}

	s.logger.Info("invitation code created successfully", zap.String("code", code))
	return invCode, nil
}

// GetInvitationCode 获取邀请码信息
func (s *Service) GetInvitationCode(ctx context.Context, code string) (*domain.InvitationCode, error) {
	invCode, err := s.repo.GetInvitationCodeByCode(ctx, code)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("invitation code not found")
		}
		return nil, errors.New("failed to get invitation code")
	}
	return invCode, nil
}

// ListInvitationCodes 列出邀请码
func (s *Service) ListInvitationCodes(ctx context.Context, userID string, limit, offset int) ([]*domain.InvitationCode, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	codes, err := s.repo.ListInvitationCodes(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to list invitation codes", zap.Error(err))
		return nil, errors.New("failed to list invitation codes")
	}
	return codes, nil
}

// UpdateInvitationCode 更新邀请码
func (s *Service) UpdateInvitationCode(ctx context.Context, userID, codeID string, req UpdateInvitationCodeRequest) (*domain.InvitationCode, error) {
	s.logger.Info("updating invitation code",
		zap.String("userID", userID),
		zap.String("codeID", codeID),
	)

	// 获取邀请码
	invCode, err := s.repo.GetInvitationCodeByID(ctx, codeID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("invitation code not found")
		}
		return nil, errors.New("failed to get invitation code")
	}

	// 验证权限（只有创建者可以更新）
	if invCode.CreatedBy != userID {
		return nil, errors.New("unauthorized: you can only update your own invitation codes")
	}

	// 更新字段
	if req.IsActive != nil {
		invCode.IsActive = *req.IsActive
	}
	if req.MaxUses != nil {
		invCode.MaxUses = *req.MaxUses
	}
	if req.ExpiresAt != nil {
		invCode.ExpiresAt = *req.ExpiresAt
	}
	if req.Description != nil {
		invCode.Description = *req.Description
	}
	invCode.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateInvitationCode(ctx, invCode); err != nil {
		s.logger.Error("failed to update invitation code", zap.Error(err))
		return nil, errors.New("failed to update invitation code")
	}

	s.logger.Info("invitation code updated successfully")
	return invCode, nil
}

// DeleteInvitationCode 删除邀请码
func (s *Service) DeleteInvitationCode(ctx context.Context, userID, codeID string) error {
	s.logger.Info("deleting invitation code",
		zap.String("userID", userID),
		zap.String("codeID", codeID),
	)

	// 获取邀请码
	invCode, err := s.repo.GetInvitationCodeByID(ctx, codeID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("invitation code not found")
		}
		return errors.New("failed to get invitation code")
	}

	// 验证权限（只有创建者可以删除）
	if invCode.CreatedBy != userID {
		return errors.New("unauthorized: you can only delete your own invitation codes")
	}

	if err := s.repo.DeleteInvitationCode(ctx, codeID); err != nil {
		s.logger.Error("failed to delete invitation code", zap.Error(err))
		return errors.New("failed to delete invitation code")
	}

	s.logger.Info("invitation code deleted successfully")
	return nil
}

// ValidateInvitationCode 验证邀请码（公开接口）
func (s *Service) ValidateInvitationCode(ctx context.Context, code string) error {
	return s.repo.ValidateInvitationCode(ctx, code)
}

// MARK: - Referral System (StoryCreationAppUI Design)

const (
	// ReferralPointsReward 邀请奖励积分
	ReferralPointsReward = 100
)

// GetOrCreateReferralCode 获取或创建用户专属邀请码
func (s *Service) GetOrCreateReferralCode(ctx context.Context, userID string) (string, error) {
	s.logger.Info("getting or creating referral code", zap.String("userID", userID))

	// 先尝试获取用户信息
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return "", errors.New("user not found")
	}

	// 如果用户已有邀请码，直接返回
	if user.ReferralCode != "" {
		return user.ReferralCode, nil
	}

	// 生成新的专属邀请码（基于用户ID生成短码）
	referralCode := GenerateUserReferralCode(userID)

	// 更新用户的邀请码
	user.ReferralCode = referralCode
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("failed to update user referral code", zap.Error(err))
		return "", errors.New("failed to create referral code")
	}

	s.logger.Info("referral code created", zap.String("userID", userID), zap.String("code", referralCode))
	return referralCode, nil
}

// UseReferralCode 使用邀请码（新用户注册时调用）
func (s *Service) UseReferralCode(ctx context.Context, refereeID, referralCode string) (*domain.ReferralResponse, error) {
	s.logger.Info("using referral code",
		zap.String("refereeID", refereeID),
		zap.String("referralCode", referralCode))

	// 查找邀请码对应的用户
	referrer, err := s.repo.GetUserByReferralCode(ctx, referralCode)
	if err != nil {
		if err == domain.ErrNotFound {
			return &domain.ReferralResponse{
				Success: false,
				Message: "invalid referral code",
			}, nil
		}
		return nil, errors.New("failed to validate referral code")
	}

	// 不能使用自己的邀请码
	if referrer.ID == refereeID {
		return &domain.ReferralResponse{
			Success: false,
			Message: "cannot use your own referral code",
		}, nil
	}

	// 检查是否已经被邀请过
	existing, _ := s.repo.GetUserReferralByReferee(ctx, refereeID)
	if existing != nil {
		return &domain.ReferralResponse{
			Success: false,
			Message: "already referred",
		}, nil
	}

	// 创建邀请记录
	now := time.Now().Unix()
	referral := &domain.UserReferral{
		ID:           uuid.New().String(),
		ReferrerID:   referrer.ID,
		RefereeID:    refereeID,
		ReferralCode: referralCode,
		PointsEarned: ReferralPointsReward,
		Status:       string(domain.ReferralStatusRewarded),
		CreatedAt:    now,
		RewardedAt:   now,
	}

	if err := s.repo.CreateUserReferral(ctx, referral); err != nil {
		s.logger.Error("failed to create referral record", zap.Error(err))
		return nil, errors.New("failed to process referral")
	}

	// 给邀请人增加积分
	if err := s.repo.AddUserPoints(ctx, referrer.ID, ReferralPointsReward); err != nil {
		s.logger.Error("failed to add points to referrer", zap.Error(err))
		// 不返回错误，记录已经创建
	}

	s.logger.Info("referral processed successfully",
		zap.String("referrerID", referrer.ID),
		zap.String("refereeID", refereeID),
		zap.Int("points", ReferralPointsReward))

	return &domain.ReferralResponse{
		Success:      true,
		Message:      "referral successful",
		PointsEarned: ReferralPointsReward,
	}, nil
}

// GetReferralStats 获取用户邀请统计
func (s *Service) GetReferralStats(ctx context.Context, userID string) (*domain.ReferralStats, error) {
	stats, err := s.repo.GetReferralStats(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get referral stats", zap.Error(err))
		return nil, errors.New("failed to get referral stats")
	}
	return stats, nil
}

// GetReferrals 获取用户邀请列表
func (s *Service) GetReferrals(ctx context.Context, userID string, limit, offset int) ([]*domain.UserReferral, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	referrals, err := s.repo.GetReferralsByUser(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get referrals", zap.Error(err))
		return nil, errors.New("failed to get referrals")
	}
	return referrals, nil
}

// GetInviteShareContent 获取邀请分享内容
func (s *Service) GetInviteShareContent(ctx context.Context, userID string) (*domain.InviteShareContent, error) {
	// 获取或创建邀请码
	referralCode, err := s.GetOrCreateReferralCode(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Username
	}

	return &domain.InviteShareContent{
		Title:        displayName + " 邀请你一起玩未择",
		Description:  "来未择，用 AI 创作你的故事世界！注册即送新人礼包。",
		Link:         "https://weize.app/invite/" + referralCode,
		ReferralCode: referralCode,
	}, nil
}

// AddPoints 增加用户积分
func (s *Service) AddPoints(ctx context.Context, userID string, points int) error {
	if points <= 0 {
		return errors.New("points must be positive")
	}

	if err := s.repo.AddUserPoints(ctx, userID, points); err != nil {
		s.logger.Error("failed to add points",
			zap.String("userID", userID),
			zap.Int("points", points),
			zap.Error(err))
		return errors.New("failed to add points")
	}

	s.logger.Info("points added successfully",
		zap.String("userID", userID),
		zap.Int("points", points))
	return nil
}

// GetUserPoints 获取用户积分
func (s *Service) GetUserPoints(ctx context.Context, userID string) (int, error) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return 0, errors.New("user not found")
	}
	return user.Points, nil
}
