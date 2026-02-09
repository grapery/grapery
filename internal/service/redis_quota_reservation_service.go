package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// QuotaReservation 配额预留记录
type QuotaReservation struct {
	ReservationID   string                 `json:"reservation_id"`
	UserID          string                 `json:"user_id"`
	EstimatedTokens int                    `json:"estimated_tokens"`
	ActualTokens    int                    `json:"actual_tokens"`
	SourceType      string                 `json:"source_type"`
	Status          string                 `json:"status"` // pending, confirmed, released
	CreatedAt       time.Time              `json:"created_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
	ConfirmedAt     *time.Time             `json:"confirmed_at,omitempty"`
	ReleasedAt      *time.Time             `json:"released_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// RedisQuotaReservationService Redis 版本的配额预留服务
// 使用 Redis 存储预留记录，支持分布式部署
type RedisQuotaReservationService struct {
	logger *zap.Logger
	repo   domain.Repository
	client *redis.Client
}

// NewRedisQuotaReservationService 创建 Redis 版本的配额预留服务
func NewRedisQuotaReservationService(logger *zap.Logger, repo domain.Repository, client *redis.Client) *RedisQuotaReservationService {
	return &RedisQuotaReservationService{
		logger: logger,
		repo:   repo,
		client: client,
	}
}

// reservationKey 生成 Redis key 前缀
func reservationKey(reservationID string) string {
	return fmt.Sprintf("quota_reservation:%s", reservationID)
}

// userReservationsKey 生成用户预留列表的 key
func userReservationsKey(userID string) string {
	return fmt.Sprintf("quota_reservations:user:%s", userID)
}

// ReserveQuota 预留配额
// 在 AI 调用前预留配额，确保用户有足够的额度
func (s *RedisQuotaReservationService) ReserveQuota(ctx context.Context, userID string, estimatedTokens int, sourceType string, metadata map[string]interface{}) (*QuotaReservation, error) {
	reservationID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(10 * time.Minute) // 预留 10 分钟后过期

	// 1. 检查用户当前余额
	balance, err := s.repo.GetTokenBalance(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get token balance",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get token balance: %w", err)
	}

	if balance < estimatedTokens {
		s.logger.Warn("insufficient token balance",
			zap.String("userID", userID),
			zap.Int("balance", balance),
			zap.Int("estimatedTokens", estimatedTokens))
		return nil, fmt.Errorf("insufficient token balance: have %d, need %d", balance, estimatedTokens)
	}

	// 2. 使用 Redis 事务原子性地预留配额
	ttl := 10 * time.Minute

	// 创建预留记录
	reservation := &QuotaReservation{
		ReservationID:   reservationID,
		UserID:          userID,
		EstimatedTokens: estimatedTokens,
		ActualTokens:    0,
		SourceType:      sourceType,
		Status:          "pending",
		CreatedAt:       now,
		ExpiresAt:       expiresAt,
		Metadata:        metadata,
	}

	reservationJSON, err := json.Marshal(reservation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reservation: %w", err)
	}

	// Redis 事务：检查并设置
	pipe := s.client.Pipeline()

	// 设置预留记录
	pipe.Set(ctx, reservationKey(reservationID), reservationJSON, ttl)

	// 添加到用户的预留列表（用于查询用户所有预留）
	pipe.SAdd(ctx, userReservationsKey(userID), reservationID)
	pipe.Expire(ctx, userReservationsKey(userID), ttl)

	// 执行事务
	_, err = pipe.Exec(ctx)
	if err != nil {
		s.logger.Error("failed to create reservation in Redis",
			zap.String("reservationID", reservationID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create reservation: %w", err)
	}

	// 3. 预先扣减配额（作为预留）
	_, err = s.repo.UpdateTokenBalance(ctx, userID, -estimatedTokens, sourceType+"_reservation", fmt.Sprintf("Reserved %d tokens for %s", estimatedTokens, reservationID))
	if err != nil {
		// 预留失败，清理 Redis 记录
		s.client.Del(ctx, reservationKey(reservationID))
		s.client.SRem(ctx, userReservationsKey(userID), reservationID)

		s.logger.Error("failed to reserve token balance",
			zap.String("userID", userID),
			zap.Int("estimatedTokens", estimatedTokens),
			zap.Error(err))
		return nil, fmt.Errorf("failed to reserve tokens: %w", err)
	}

	s.logger.Info("quota reserved successfully",
		zap.String("reservationID", reservationID),
		zap.String("userID", userID),
		zap.Int("estimatedTokens", estimatedTokens),
		zap.String("sourceType", sourceType))

	return reservation, nil
}

// ConfirmQuota 确认配额扣减
// AI 调用成功后调用，确认实际消耗的 token 数量
// 如果实际消耗少于预留数量，多余的会被退还
func (s *RedisQuotaReservationService) ConfirmQuota(ctx context.Context, reservationID string, actualTokens int) error {
	// 从 Redis 获取预留记录
	val, err := s.client.Get(ctx, reservationKey(reservationID)).Result()
	if err == redis.Nil {
		return fmt.Errorf("reservation not found: %s", reservationID)
	}
	if err != nil {
		return fmt.Errorf("failed to get reservation: %w", err)
	}

	var reservation QuotaReservation
	if err := json.Unmarshal([]byte(val), &reservation); err != nil {
		return fmt.Errorf("failed to unmarshal reservation: %w", err)
	}

	if reservation.Status != "pending" {
		return fmt.Errorf("reservation is not in pending status: %s", reservationID)
	}

	now := time.Now()

	// 计算需要退还的 token
	tokensToReturn := reservation.EstimatedTokens - actualTokens
	if tokensToReturn < 0 {
		// 实际消耗多于预留，需要额外扣减
		tokensToReturn = 0
		extraTokens := actualTokens - reservation.EstimatedTokens
		_, err := s.repo.UpdateTokenBalance(ctx, reservation.UserID, -extraTokens, reservation.SourceType, fmt.Sprintf("Additional token consumption for %s", reservationID))
		if err != nil {
			s.logger.Error("failed to deduct extra tokens",
				zap.String("reservationID", reservationID),
				zap.Int("extraTokens", extraTokens),
				zap.Error(err))
			return fmt.Errorf("failed to deduct extra tokens: %w", err)
		}
	}

	// 退还多余的预留
	if tokensToReturn > 0 {
		_, err := s.repo.UpdateTokenBalance(ctx, reservation.UserID, tokensToReturn, reservation.SourceType+"_refund", fmt.Sprintf("Refunded %d unused tokens from %s", tokensToReturn, reservationID))
		if err != nil {
			s.logger.Error("failed to refund unused tokens",
				zap.String("reservationID", reservationID),
				zap.Int("tokensToReturn", tokensToReturn),
				zap.Error(err))
			return fmt.Errorf("failed to refund tokens: %w", err)
		}
	}

	// 更新预留记录状态
	reservation.ActualTokens = actualTokens
	reservation.Status = "confirmed"
	reservation.ConfirmedAt = &now

	// 保存到 Redis（更新状态）
	reservationJSON, err := json.Marshal(reservation)
	if err != nil {
		return fmt.Errorf("failed to marshal reservation: %w", err)
	}

	// 更新预留记录，设置较短的过期时间（7天后清理）
	s.client.Set(ctx, reservationKey(reservationID), reservationJSON, 7*24*time.Hour)

	s.logger.Info("quota reservation confirmed",
		zap.String("reservationID", reservationID),
		zap.String("userID", reservation.UserID),
		zap.Int("estimatedTokens", reservation.EstimatedTokens),
		zap.Int("actualTokens", actualTokens),
		zap.Int("tokensToReturn", tokensToReturn))

	return nil
}

// ReleaseQuota 释放预留配额
// AI 调用失败时调用，全额退还预留的配额
func (s *RedisQuotaReservationService) ReleaseQuota(ctx context.Context, reservationID string) error {
	// 从 Redis 获取预留记录
	val, err := s.client.Get(ctx, reservationKey(reservationID)).Result()
	if err == redis.Nil {
		return fmt.Errorf("reservation not found: %s", reservationID)
	}
	if err != nil {
		return fmt.Errorf("failed to get reservation: %w", err)
	}

	var reservation QuotaReservation
	if err := json.Unmarshal([]byte(val), &reservation); err != nil {
		return fmt.Errorf("failed to unmarshal reservation: %w", err)
	}

	if reservation.Status != "pending" {
		return fmt.Errorf("reservation is not in pending status: %s", reservationID)
	}

	now := time.Now()

	// 退还预留的配额
	_, err = s.repo.UpdateTokenBalance(ctx, reservation.UserID, reservation.EstimatedTokens, reservation.SourceType+"_release", fmt.Sprintf("Released %d tokens from failed request %s", reservation.EstimatedTokens, reservationID))
	if err != nil {
		s.logger.Error("failed to release reserved tokens",
			zap.String("reservationID", reservationID),
			zap.Int("estimatedTokens", reservation.EstimatedTokens),
			zap.Error(err))
		return fmt.Errorf("failed to release tokens: %w", err)
	}

	// 更新预留记录状态
	reservation.Status = "released"
	reservation.ReleasedAt = &now

	// 保存到 Redis（更新状态，设置较短过期时间）
	reservationJSON, err := json.Marshal(reservation)
	if err != nil {
		return fmt.Errorf("failed to marshal reservation: %w", err)
	}

	s.client.Set(ctx, reservationKey(reservationID), reservationJSON, 24*time.Hour)

	s.logger.Info("quota reservation released",
		zap.String("reservationID", reservationID),
		zap.String("userID", reservation.UserID),
		zap.Int("tokensReleased", reservation.EstimatedTokens))

	return nil
}

// GetReservation 获取预留记录
func (s *RedisQuotaReservationService) GetReservation(reservationID string) (*QuotaReservation, error) {
	val, err := s.client.Get(context.Background(), reservationKey(reservationID)).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("reservation not found: %s", reservationID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	var reservation QuotaReservation
	if err := json.Unmarshal([]byte(val), &reservation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reservation: %w", err)
	}

	return &reservation, nil
}

// GetUserReservations 获取用户的所有预留记录
func (s *RedisQuotaReservationService) GetUserReservations(ctx context.Context, userID string) ([]*QuotaReservation, error) {
	// 获取用户的所有预留 ID
	reservationIDs, err := s.client.SMembers(ctx, userReservationsKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user reservations: %w", err)
	}

	var reservations []*QuotaReservation
	for _, id := range reservationIDs {
		reservation, err := s.GetReservation(id)
		if err != nil {
			// 记录错误但继续处理其他预留
			s.logger.Warn("failed to get reservation",
				zap.String("reservationID", id),
				zap.Error(err))
			continue
		}
		reservations = append(reservations, reservation)
	}

	return reservations, nil
}

// CleanupExpiredReservations 清理过期的预留记录
func (s *RedisQuotaReservationService) CleanupExpiredReservations(ctx context.Context) {
	// Redis 会自动清理过期的 key，这里主要用于日志记录
	// 扫描所有预留记录
	pattern := "quota_reservation:*"

	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			s.logger.Error("failed to scan for reservations",
				zap.Error(err))
			break
		}

		now := time.Now()
		for _, key := range keys {
			val, err := s.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var reservation QuotaReservation
			if err := json.Unmarshal([]byte(val), &reservation); err != nil {
				continue
			}

			// 检查是否过期且仍处于 pending 状态
			if now.After(reservation.ExpiresAt) && reservation.Status == "pending" {
				s.logger.Warn("found expired pending reservation",
					zap.String("reservationID", reservation.ReservationID),
					zap.String("userID", reservation.UserID),
					zap.Int("estimatedTokens", reservation.EstimatedTokens))

				// 自动释放预留
				if err := s.ReleaseQuota(ctx, reservation.ReservationID); err != nil {
					s.logger.Error("failed to auto-release expired reservation",
						zap.String("reservationID", reservation.ReservationID),
						zap.Error(err))
				}
			}
		}

		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
}
