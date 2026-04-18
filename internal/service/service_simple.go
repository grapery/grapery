package service

import (
	"context"
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// 简化的 Service 方法，用于临时兼容旧代码

// ErrUnauthorized 未授权错误
var ErrUnauthorized = errors.New("unauthorized: user not authenticated")

// ErrAuthUserNotFound JWT 已通过校验但 users 表中无对应行（账号删除、库不一致、错误环境签发的 token 等）。
var ErrAuthUserNotFound = errors.New("auth user not found")

// EnsureUserExists returns ErrAuthUserNotFound when userID is absent from users.
func (s *Service) EnsureUserExists(ctx context.Context, userID string) error {
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		if s.logger != nil {
			s.logger.Warn("user id from JWT not found in users table",
				zap.String("userID", userID))
		}
		return ErrAuthUserNotFound
	}
	return nil
}

// CurrentUser 获取当前用户
// 从 context 中获取用户 ID，然后从数据库查询完整的用户信息
func (s *Service) CurrentUser(ctx context.Context) (*domain.User, error) {
	// 从 context 获取用户 ID（由 AuthMiddleware 注入）
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, ErrUnauthorized
	}

	// 从数据库获取用户信息
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
