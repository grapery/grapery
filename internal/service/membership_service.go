package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// ========== Membership Service Methods ==========

// ListMembershipPlans 列出可用会员方案
func (s *Service) ListMembershipPlans(ctx context.Context) ([]domain.MembershipPlan, error) {
	// 使用现有的 ListSubscriptionPlans 方法
	plans, err := s.repo.ListSubscriptionPlans(ctx, true)
	if err != nil {
		s.logger.Error("failed to list membership plans", zap.Error(err))
		return nil, err
	}

	// 转换为 MembershipPlan 格式（仅对外提供月付 / 年付）
	result := make([]domain.MembershipPlan, 0, len(plans))
	for _, p := range plans {
		period := normalizeBillingPeriod(p.BillingPeriod)
		if period == string(domain.PeriodQuarterly) {
			continue
		}
		features := parsePlanFeatures(p.Features)
		tier := domain.CanonicalMembershipTier(p.MembershipTier, p.Name)

		result = append(result, domain.MembershipPlan{
			ID:           p.ID,
			Tier:         tier,
			Period:       period,
			IAPProductID: p.IAPProductID,
			Price:        p.Price,
			PerMonth:     perMonthPrice(p.Price, period),
			AIQuota:      p.TokenQuota,
			Features:     features,
			IsActive:     p.IsActive,
			SortOrder:    p.SortOrder,
		})
	}

	return result, nil
}

// GetUserMembership 获取用户会员信息
func (s *Service) GetUserMembership(ctx context.Context, userID string) (*domain.UserMembership, error) {
	membership, err := s.repo.Membership(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user membership", zap.Error(err), zap.String("userId", userID))
		// 返回免费用户
		now := time.Now().Unix()
		return &domain.UserMembership{
			BaseModel: common.BaseModel{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:          userID,
			Tier:            domain.TierTypeFree,
			StartedAt:       now,
			ExpiresAt:       0,
			AutoRenew:       false,
			AIUsedThisMonth: 0,
			AILimit:         10,
		}, nil
	}

	now := time.Now().Unix()
	return &domain.UserMembership{
		BaseModel: common.BaseModel{
			ID:        membership.ID,
			CreatedAt: membership.CreatedAt,
			UpdatedAt: now,
		},
		UserID:          membership.UserID,
		Tier:            domain.CanonicalMembershipTier(membership.Tier, ""),
		StartedAt:       membership.StartDate,
		ExpiresAt:       getExpiresAt(membership.EndDate),
		AutoRenew:       membership.AutoRenew,
		AIUsedThisMonth: membership.TokenUsed,
		AILimit:         membership.TokenQuota,
	}, nil
}

// SubscribeMembership 订阅会员方案
func (s *Service) SubscribeMembership(ctx context.Context, userID string, req domain.SubscribeRequest) (*domain.SubscribeResponse, error) {
	// 获取方案信息
	plans, err := s.repo.ListSubscriptionPlans(ctx, true)
	if err != nil {
		s.logger.Error("failed to list subscription plans", zap.Error(err))
		return nil, err
	}

	wantTier := domain.CanonicalMembershipTier(string(req.Tier), "")
	wantPeriod := normalizeBillingPeriod(string(req.Period))
	if wantPeriod == string(domain.PeriodQuarterly) {
		return nil, domain.ErrInvalidInput
	}

	var selectedPlan *domain.SubscriptionPlan
	for _, p := range plans {
		if p == nil {
			continue
		}
		if domain.CanonicalMembershipTier(p.MembershipTier, p.Name) != wantTier {
			continue
		}
		bp := normalizeBillingPeriod(p.BillingPeriod)
		if bp == string(domain.PeriodQuarterly) {
			continue
		}
		if bp != wantPeriod {
			continue
		}
		selectedPlan = p
		break
	}

	if selectedPlan == nil {
		return nil, domain.ErrNotFound
	}

	// 创建订单
	now := time.Now().Unix()
	order := &domain.SubscriptionOrder{
		ID:            uuid.New().String(),
		UserID:        userID,
		PlanID:        selectedPlan.ID,
		Status:        string(common.OrderStatusPending),
		Amount:        selectedPlan.Price,
		Currency:      selectedPlan.Currency,
		PaymentMethod: "pending",
		StartDate:     now,
		EndDate:       now + int64(30*24*60*60), // 30 days
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateSubscriptionOrder(ctx, order); err != nil {
		s.logger.Error("failed to create subscription order", zap.Error(err))
		return nil, err
	}

	return &domain.SubscribeResponse{
		OrderID: order.ID,
		Status:  order.Status,
	}, nil
}

// CancelMembership 取消会员订阅
func (s *Service) CancelMembership(ctx context.Context, userID string) error {
	membership, err := s.repo.Membership(ctx, userID)
	if err != nil {
		return err
	}

	// 设置不自动续费
	membership.AutoRenew = false
	membership.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateMembership(ctx, membership); err != nil {
		s.logger.Error("failed to cancel membership", zap.Error(err))
		return err
	}

	return nil
}

// GetMembershipUsage 获取会员使用量
func (s *Service) GetMembershipUsage(ctx context.Context, userID string) (*MembershipUsage, error) {
	membership, err := s.repo.Membership(ctx, userID)
	if err != nil {
		// 如果没有会员信息，返回免费用户的使用量
		return &MembershipUsage{
			UserID:          userID,
			Tier:            string(domain.TierTypeFree),
			AIUsedThisMonth: 0,
			AILimit:         10, // 免费用户限制
			PeriodStart:     time.Now().Unix(),
			PeriodEnd:       time.Now().AddDate(0, 1, 0).Unix(),
		}, nil
	}

	return &MembershipUsage{
		UserID: userID,
		Tier:   string(domain.CanonicalMembershipTier(membership.Tier, "")),
		AIUsedThisMonth: membership.TokenUsed,
		AILimit:         membership.TokenQuota,
		PeriodStart:     membership.StartDate,
		PeriodEnd:       getExpiresAt(membership.EndDate),
	}, nil
}

// Helper function
func getExpiresAt(endDate *int64) int64 {
	if endDate == nil {
		return 0
	}
	return *endDate
}

func parsePlanFeatures(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	var features []string
	if err := json.Unmarshal([]byte(raw), &features); err != nil {
		return []string{}
	}
	return features
}

func normalizeBillingPeriod(raw string) string {
	period := strings.ToLower(strings.TrimSpace(raw))
	switch period {
	case string(domain.PeriodMonthly), string(domain.PeriodQuarterly), string(domain.PeriodYearly):
		return period
	default:
		return string(domain.PeriodMonthly)
	}
}

func perMonthPrice(price float64, period string) float64 {
	switch period {
	case string(domain.PeriodQuarterly):
		return price / 3
	case string(domain.PeriodYearly):
		return price / 12
	default:
		return price
	}
}

// MembershipUsage 会员使用量
type MembershipUsage struct {
	UserID          string `json:"userId"`
	Tier            string `json:"tier"`
	AIUsedThisMonth int    `json:"aiUsedThisMonth"`
	AILimit         int    `json:"aiLimit"`
	PeriodStart     int64  `json:"periodStart"`
	PeriodEnd       int64  `json:"periodEnd"`
}
