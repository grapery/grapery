package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QuotaReservation 配额预留记录
type QuotaReservation struct {
	ReservationID   string    `json:"reservation_id"`
	UserID          string    `json:"user_id"`
	EstimatedTokens int       `json:"estimated_tokens"`
	ActualTokens    int       `json:"actual_tokens"`
	SourceType      string    `json:"source_type"`      // ai_text_generation, ai_image_generation, etc.
	Status          string    `json:"status"`           // pending, confirmed, released, expired
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
	ReleasedAt      *time.Time `json:"released_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// QuotaReservationService 配额预留服务
type QuotaReservationService struct {
	logger     *zap.Logger
	repo       domain.Repository
	reservations map[string]*QuotaReservation // 内存存储（生产环境应使用 Redis）
	mu         map[string]*time.Time         // 预留锁
}

// NewQuotaReservationService 创建配额预留服务
func NewQuotaReservationService(logger *zap.Logger, repo domain.Repository) *QuotaReservationService {
	return &QuotaReservationService{
		logger:       logger,
		repo:         repo,
		reservations: make(map[string]*QuotaReservation),
		mu:           make(map[string]*time.Time),
	}
}

// ReserveQuota 预留配额
// 在 AI 调用前预留配额，确保用户有足够的额度
func (s *QuotaReservationService) ReserveQuota(ctx context.Context, userID string, estimatedTokens int, sourceType string, metadata map[string]interface{}) (*QuotaReservation, error) {
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

	// 2. 预先扣减配额（作为预留）
	// 这会将配额标记为"已使用但未确认"状态
	_, err = s.repo.UpdateTokenBalance(ctx, userID, -estimatedTokens, sourceType+"_reservation", fmt.Sprintf("Reserved %d tokens for %s", estimatedTokens, reservationID))
	if err != nil {
		s.logger.Error("failed to reserve token balance",
			zap.String("userID", userID),
			zap.Int("estimatedTokens", estimatedTokens),
			zap.Error(err))
		return nil, fmt.Errorf("failed to reserve tokens: %w", err)
	}

	// 3. 创建预留记录
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

	// 4. 存储预留记录
	s.reservations[reservationID] = reservation
	s.mu[reservationID] = &now

	s.logger.Info("quota reserved successfully",
		zap.String("reservationID", reservationID),
		zap.String("userID", userID),
		zap.Int("estimatedTokens", estimatedTokens),
		zap.String("sourceType", sourceType))

	// 5. 启动过期清理
	go s.monitorExpiration(reservationID)

	return reservation, nil
}

// ConfirmQuota 确认配额扣减
// AI 调用成功后调用，确认实际消耗的 token 数量
// 如果实际消耗少于预留数量，多余的会被退还
func (s *QuotaReservationService) ConfirmQuota(ctx context.Context, reservationID string, actualTokens int) error {
	reservation, exists := s.reservations[reservationID]
	if !exists {
		return fmt.Errorf("reservation not found: %s", reservationID)
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

	// 更新预留记录
	reservation.ActualTokens = actualTokens
	reservation.Status = "confirmed"
	reservation.ConfirmedAt = &now

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
func (s *QuotaReservationService) ReleaseQuota(ctx context.Context, reservationID string) error {
	reservation, exists := s.reservations[reservationID]
	if !exists {
		return fmt.Errorf("reservation not found: %s", reservationID)
	}

	if reservation.Status != "pending" {
		return fmt.Errorf("reservation is not in pending status: %s", reservationID)
	}

	now := time.Now()

	// 退还预留的配额
	_, err := s.repo.UpdateTokenBalance(ctx, reservation.UserID, reservation.EstimatedTokens, reservation.SourceType+"_release", fmt.Sprintf("Released %d tokens from failed request %s", reservation.EstimatedTokens, reservationID))
	if err != nil {
		s.logger.Error("failed to release reserved tokens",
			zap.String("reservationID", reservationID),
			zap.Int("estimatedTokens", reservation.EstimatedTokens),
			zap.Error(err))
		return fmt.Errorf("failed to release tokens: %w", err)
	}

	// 更新预留记录
	reservation.Status = "released"
	reservation.ReleasedAt = &now

	s.logger.Info("quota reservation released",
		zap.String("reservationID", reservationID),
		zap.String("userID", reservation.UserID),
		zap.Int("tokensReleased", reservation.EstimatedTokens))

	return nil
}

// monitorExpiration 监控预留过期
func (s *QuotaReservationService) monitorExpiration(reservationID string) {
	reservation, exists := s.reservations[reservationID]
	if !exists {
		return
	}

	duration := time.Until(reservation.ExpiresAt)
	if duration <= 0 {
		return
	}

	<-time.After(duration)

	// 检查是否仍处于 pending 状态
	if reservation.Status == "pending" {
		s.logger.Warn("quota reservation expired, releasing tokens",
			zap.String("reservationID", reservationID),
			zap.String("userID", reservation.UserID),
			zap.Int("estimatedTokens", reservation.EstimatedTokens))

		// 自动释放预留
		ctx := context.Background()
		_ = s.ReleaseQuota(ctx, reservationID)
	}
}

// GetReservation 获取预留记录
func (s *QuotaReservationService) GetReservation(reservationID string) (*QuotaReservation, bool) {
	reservation, exists := s.reservations[reservationID]
	return reservation, exists
}

// CleanupExpiredReservations 清理过期的预留记录
func (s *QuotaReservationService) CleanupExpiredReservations() {
	now := time.Now()
	for id, reservation := range s.reservations {
		if now.After(reservation.ExpiresAt) && reservation.Status == "pending" {
			s.logger.Warn("found expired reservation",
				zap.String("reservationID", id),
				zap.String("userID", reservation.UserID))
		}
		// 清理已确认或释放的旧记录（超过1小时）
		if (reservation.Status == "confirmed" || reservation.Status == "released") && now.Sub(reservation.ConfirmedAt != nil ? *reservation.ConfirmedAt : reservation.ReleasedAt != nil ? *reservation.ReleasedAt : reservation.CreatedAt) > time.Hour {
			delete(s.reservations, id)
			delete(s.mu, id)
		}
	}
}
