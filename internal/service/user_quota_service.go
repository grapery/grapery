package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// UserQuotaService 用户配额服务
// 提供用户配额、会员信息、使用统计等功能
type UserQuotaService struct {
	repo   domain.Repository
	logger *zap.Logger
}

// NewUserQuotaService 创建用户配额服务
func NewUserQuotaService(repo domain.Repository, logger *zap.Logger) *UserQuotaService {
	return &UserQuotaService{
		repo:   repo,
		logger: logger,
	}
}

// GetUserQuotaInfo 获取用户配额信息
func (s *UserQuotaService) GetUserQuotaInfo(ctx context.Context, userID string) (*domain.UserQuotaInfo, error) {
	// 获取 token 余额
	tokenBalance, err := s.repo.GetTokenBalance(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get token balance",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get token balance: %w", err)
	}

	// 获取活跃订阅
	activeSubscription, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil && err.Error() != "not found" {
		s.logger.Warn("failed to get active subscription",
			zap.String("userID", userID),
			zap.Error(err))
	}

	// 设置配额和使用量
	tokenQuota := 0 // 免费用户的默认配额
	var resetDate *time.Time

	if activeSubscription != nil {
		// 从订阅计划获取配额
		plan, err := s.repo.GetSubscriptionPlan(ctx, activeSubscription.PlanID)
		if err == nil {
			tokenQuota = plan.TokenQuota
		}

		// 设置重置日期为订阅结束日期
		if activeSubscription.EndDate > 0 {
			t := time.Unix(activeSubscription.EndDate, 0)
			resetDate = &t
		}
	}

	// 如果没有订阅，使用免费配额
	if tokenQuota == 0 {
		tokenQuota = 1000 // 免费用户每月 1000 tokens
		// 设置重置日期为下个月1号
		now := time.Now()
		nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		resetDate = &nextMonth
	}

	// 计算 token 使用量（从 token 交易记录中获取）
	tokenUsed := 0
	if tokenQuota > tokenBalance {
		tokenUsed = tokenQuota - tokenBalance
	}

	tokenRemaining := tokenBalance
	if tokenRemaining < 0 {
		tokenRemaining = 0
	}

	return &domain.UserQuotaInfo{
		UserID:           userID,
		TokenBalance:     tokenBalance,
		TokenQuota:       tokenQuota,
		TokenUsed:        tokenUsed,
		TokenRemaining:   tokenRemaining,
		StorageUsed:      0,                        // TODO: 从存储服务获取
		StorageQuota:     100 * 1024 * 1024 * 1024, // 100GB
		StorageRemaining: 100 * 1024 * 1024 * 1024, // 100GB
		ResetDate:        resetDate,
	}, nil
}

// GetUserMembershipInfo 获取用户会员信息
func (s *UserQuotaService) GetUserMembershipInfo(ctx context.Context, userID string) (*domain.UserMembershipInfo, error) {
	// 获取活跃订阅
	activeSubscription, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil {
		// 没有活跃订阅，返回免费用户信息
		return s.getFreeUserInfo(ctx, userID)
	}

	// 获取订阅计划
	plan, err := s.repo.GetSubscriptionPlan(ctx, activeSubscription.PlanID)
	if err != nil {
		s.logger.Error("failed to get subscription plan",
			zap.String("planID", activeSubscription.PlanID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	// 获取配额信息
	quotaInfo, err := s.GetUserQuotaInfo(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get quota info, using defaults",
			zap.String("userID", userID),
			zap.Error(err))
		quotaInfo = &domain.UserQuotaInfo{
			TokenBalance:   0,
			TokenQuota:     plan.TokenQuota,
			TokenUsed:      0,
			TokenRemaining: plan.TokenQuota,
		}
	}

	// 计算剩余天数
	daysRemaining := 0
	canRenew := false
	if activeSubscription.EndDate > 0 {
		endDate := time.Unix(activeSubscription.EndDate, 0)
		daysRemaining = int(endDate.Sub(time.Now()).Hours() / 24)
		if daysRemaining <= 7 {
			canRenew = true
		}
	}

	// 获取会员权益
	benefits := s.getMembershipBenefits(plan.Name)

	return &domain.UserMembershipInfo{
		UserID:        userID,
		Tier:          plan.Name,
		TierName:      s.getTierName(plan.Name),
		Status:        activeSubscription.Status,
		StatusName:    s.getStatusName(activeSubscription.Status),
		StartDate:     activeSubscription.StartDate,
		EndDate:       &activeSubscription.EndDate,
		AutoRenew:     false, // SubscriptionOrder doesn't have AutoRenew field
		DaysRemaining: daysRemaining,
		CanRenew:      canRenew,
		QuotaInfo:     quotaInfo,
		Benefits:      benefits,
		CreatedAt:     activeSubscription.CreatedAt,
		UpdatedAt:     activeSubscription.UpdatedAt,
	}, nil
}

// getFreeUserInfo 获取免费用户信息
func (s *UserQuotaService) getFreeUserInfo(ctx context.Context, userID string) (*domain.UserMembershipInfo, error) {
	quotaInfo, err := s.GetUserQuotaInfo(ctx, userID)
	if err != nil {
		quotaInfo = &domain.UserQuotaInfo{
			TokenBalance:   0,
			TokenQuota:     1000,
			TokenUsed:      0,
			TokenRemaining: 1000,
		}
	}

	return &domain.UserMembershipInfo{
		UserID:        userID,
		Tier:          "free",
		TierName:      "免费用户",
		Status:        "active",
		StatusName:    "正常",
		StartDate:     time.Now().Unix(),
		EndDate:       nil,
		AutoRenew:     false,
		DaysRemaining: 0,
		CanRenew:      true,
		QuotaInfo:     quotaInfo,
		Benefits:      s.getMembershipBenefits("free"),
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}, nil
}

// GetUserRechargeInfo 获取用户充值信息
func (s *UserQuotaService) GetUserRechargeInfo(ctx context.Context, userID string) (*domain.UserRechargeInfo, error) {
	// 获取订单列表
	orders, _, err := s.repo.ListSubscriptionOrders(ctx, userID, 10, 0)
	if err != nil {
		s.logger.Error("failed to list subscription orders",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	// 计算总充值金额和次数
	var totalRecharged int64
	rechargeCount := 0
	var lastRechargeAt *int64

	for _, order := range orders {
		if order.Status == "paid" {
			// 将金额转换为分（假设原金额是元）
			totalRecharged += int64(order.Amount * 100)
			rechargeCount++

			if order.PaidAt != nil && (lastRechargeAt == nil || *order.PaidAt > *lastRechargeAt) {
				lastRechargeAt = order.PaidAt
			}
		}
	}

	// 可用的支付方式
	paymentMethods := []string{"alipay", "wechat"}
	if rechargeCount > 0 {
		paymentMethods = append(paymentMethods, "stripe", "paypal")
	}

	return &domain.UserRechargeInfo{
		UserID:            userID,
		TotalRecharged:    totalRecharged,
		TotalRechargedCNY: float64(totalRecharged) / 100,
		RechargeCount:     rechargeCount,
		LastRechargeAt:    lastRechargeAt,
		RecentOrders:      orders,
		PaymentMethods:    paymentMethods,
	}, nil
}

// GetUserUsageStatistics 获取用户使用统计
func (s *UserQuotaService) GetUserUsageStatistics(ctx context.Context, userID string, period string) (*domain.UserUsageStatistics, error) {
	now := time.Now()
	var startDate, endDate time.Time

	// 根据周期设置时间范围
	switch period {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.Add(24 * time.Hour)
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		endDate = startDate.Add(7 * 24 * time.Hour)
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0)
	case "year":
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(1, 0, 0)
	default: // "all"
		startDate = time.Time{}
		endDate = now
	}

	startTimestamp := startDate.Unix()
	endTimestamp := endDate.Unix()

	// 从 AI 生成记录中获取统计数据
	stats := &domain.UserUsageStatistics{
		UserID:    userID,
		Period:    period,
		StartDate: startTimestamp,
		EndDate:   endTimestamp,
	}

	// 获取 AI 生成记录（需要添加 ListAIGenerationRecordsByTimeRange 方法）
	records, err := s.repo.ListAIGenerationRecordsByTimeRange(ctx, userID, startTimestamp, endTimestamp)
	if err == nil {
		for _, record := range records {
			stats.TotalTokensUsed += record.TotalTokens

			switch record.Type {
			case "text":
				stats.TextGeneratedCount++
				stats.TextTokensUsed += record.TotalTokens
			case "image":
				stats.ImageGeneratedCount += record.ImageCount
				stats.ImageTokensUsed += record.TotalTokens
			case "video":
				stats.VideoGeneratedCount += record.VideoCount
				stats.VideoTokensUsed += record.TotalTokens
			}
		}
	}

	// TODO: 获取其他统计数据（故事、片段、角色等）
	// 这些需要从相应的 repository 方法获取

	return stats, nil
}

// GetTokenUsageHistory 获取 Token 使用历史
func (s *UserQuotaService) GetTokenUsageHistory(ctx context.Context, userID string, page, pageSize int) (*domain.TokenUsageHistory, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	transactions, total, err := s.repo.ListTokenTransactions(ctx, userID, pageSize, offset)
	if err != nil {
		s.logger.Error("failed to list token transactions",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list token transactions: %w", err)
	}

	details := make([]*domain.TokenUsageDetail, len(transactions))
	for i, txn := range transactions {
		details[i] = &domain.TokenUsageDetail{
			ID:          txn.ID,
			Type:        txn.Type,
			Amount:      txn.Amount,
			Balance:     txn.Balance,
			Source:      txn.Source,
			Description: txn.Description,
			RelatedID:   txn.RelatedID,
			CreatedAt:   txn.CreatedAt,
		}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &domain.TokenUsageHistory{
		Details:    details,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUserDashboardInfo 获取用户主页综合信息
func (s *UserQuotaService) GetUserDashboardInfo(ctx context.Context, userID string) (*domain.UserDashboardInfo, error) {
	// 获取用户信息
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 获取配额信息
	quotaInfo, err := s.GetUserQuotaInfo(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get quota info",
			zap.String("userID", userID),
			zap.Error(err))
		quotaInfo = nil
	}

	// 获取会员信息
	membership, err := s.GetUserMembershipInfo(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get membership info",
			zap.String("userID", userID),
			zap.Error(err))
		membership = nil
	}

	// 获取使用统计（本月）
	usageStats, err := s.GetUserUsageStatistics(ctx, userID, "month")
	if err != nil {
		s.logger.Warn("failed to get usage statistics",
			zap.String("userID", userID),
			zap.Error(err))
		usageStats = nil
	}

	// TODO: 获取最近活动

	return &domain.UserDashboardInfo{
		User:             user,
		QuotaInfo:        quotaInfo,
		Membership:       membership,
		UsageStatistics:  usageStats,
		RecentActivities: nil, // TODO: 实现最近活动获取
	}, nil
}

// GetMembershipTiers 获取所有会员等级信息
func (s *UserQuotaService) GetMembershipTiers(ctx context.Context) ([]*domain.MembershipTier, error) {
	plans, err := s.repo.ListSubscriptionPlans(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription plans: %w", err)
	}

	tiers := make([]*domain.MembershipTier, len(plans))
	for i, plan := range plans {
		tiers[i] = &domain.MembershipTier{
			Tier:          plan.Name,
			TierName:      plan.Name,
			DisplayName:   plan.Name,
			Description:   fmt.Sprintf("%s 会员计划", plan.Name),
			Price:         plan.Price,
			Currency:      plan.Currency,
			TokenQuota:    plan.TokenQuota,
			StorageQuota:  plan.StorageQuota,
			MaxStories:    plan.MaxStories,
			MaxCharacters: plan.MaxCharacters,
			Benefits:      s.getMembershipBenefits(plan.Name),
			Features:      []string{}, // TODO: 从 plan.Features 解析
			SortOrder:     plan.SortOrder,
			IsActive:      plan.IsActive,
			TrialDays:     0,          // TODO: 从配置获取
			UpgradeFrom:   []string{}, // TODO: 根据等级关系设置
		}
	}

	return tiers, nil
}

// ============== 辅助方法 ==============

// getTierName 获取等级中文名称
func (s *UserQuotaService) getTierName(tier string) string {
	names := map[string]string{
		"free":       "免费用户",
		"basic":      "基础会员",
		"pro":        "专业会员",
		"enterprise": "企业会员",
	}
	if name, ok := names[tier]; ok {
		return name
	}
	return tier
}

// getStatusName 获取状态中文名称
func (s *UserQuotaService) getStatusName(status string) string {
	names := map[string]string{
		"active":    "正常",
		"expired":   "已过期",
		"cancelled": "已取消",
		"pending":   "待支付",
	}
	if name, ok := names[status]; ok {
		return name
	}
	return status
}

// getMembershipBenefits 获取会员权益列表
func (s *UserQuotaService) getMembershipBenefits(tier string) []domain.MembershipBenefit {
	allBenefits := map[string][]domain.MembershipBenefit{
		"free": {
			{ID: "ai_text", Name: "AI 文本生成", Description: "每月 1000 tokens", Icon: "text.bubble", Enabled: true, Value: "1000 tokens/月"},
			{ID: "basic_support", Name: "基础支持", Description: "社区支持", Icon: "message", Enabled: true, Value: "社区支持"},
		},
		"basic": {
			{ID: "ai_text", Name: "AI 文本生成", Description: "每月 10000 tokens", Icon: "text.bubble", Enabled: true, Value: "10000 tokens/月"},
			{ID: "ai_image", Name: "AI 图片生成", Description: "每月 100 张", Icon: "photo", Enabled: true, Value: "100 张/月"},
			{ID: "storage", Name: "云存储", Description: "10GB 存储空间", Icon: "icloud", Enabled: true, Value: "10GB"},
			{ID: "priority_support", Name: "优先支持", Description: "更快响应", Icon: "star.fill", Enabled: true, Value: "24小时内"},
		},
		"pro": {
			{ID: "ai_text", Name: "AI 文本生成", Description: "每月 50000 tokens", Icon: "text.bubble", Enabled: true, Value: "50000 tokens/月"},
			{ID: "ai_image", Name: "AI 图片生成", Description: "每月 500 张", Icon: "photo", Enabled: true, Value: "500 张/月"},
			{ID: "ai_video", Name: "AI 视频生成", Description: "每月 10 个", Icon: "video", Enabled: true, Value: "10 个/月"},
			{ID: "storage", Name: "云存储", Description: "100GB 存储空间", Icon: "icloud", Enabled: true, Value: "100GB"},
			{ID: "priority_support", Name: "专属支持", Description: "专属客服", Icon: "star.fill", Enabled: true, Value: "12小时内"},
			{ID: "advanced_features", Name: "高级功能", Description: "解锁所有高级功能", Icon: "wand.and.stars", Enabled: true, Value: "全部"},
		},
		"enterprise": {
			{ID: "ai_text", Name: "AI 文本生成", Description: "无限 tokens", Icon: "text.bubble", Enabled: true, Value: "无限"},
			{ID: "ai_image", Name: "AI 图片生成", Description: "无限生成", Icon: "photo", Enabled: true, Value: "无限"},
			{ID: "ai_video", Name: "AI 视频生成", Description: "无限生成", Icon: "video", Enabled: true, Value: "无限"},
			{ID: "storage", Name: "云存储", Description: "1TB 存储空间", Icon: "icloud", Enabled: true, Value: "1TB"},
			{ID: "api_access", Name: "API 访问", Description: "开发者 API", Icon: "code", Enabled: true, Value: "完整 API"},
			{ID: "dedicated_support", Name: "专属顾问", Description: "一对一技术支持", Icon: "person.2", Enabled: true, Value: "7x24"},
			{ID: "custom_integration", Name: "定制集成", Description: "定制化解决方案", Icon: "gear", Enabled: true, Value: "支持"},
		},
	}

	if benefits, ok := allBenefits[tier]; ok {
		return benefits
	}
	return allBenefits["free"]
}
