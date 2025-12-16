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
	IsActive    *bool  `json:"isActive"`
	MaxUses     *int   `json:"maxUses"`
	ExpiresAt   *int64 `json:"expiresAt"`
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

