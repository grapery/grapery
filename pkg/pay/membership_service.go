package pay

import (
	"context"
	"fmt"
	"time"

	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

// MembershipService 会员服务接口
type MembershipService interface {
	// 套餐管理
	CreatePackagePlan(ctx context.Context, plan *models.PackagePlan) error
	GetPackagePlan(ctx context.Context, id uint) (*models.PackagePlan, error)
	GetPackagePlanBySKU(ctx context.Context, sku string) (*models.PackagePlan, error)
	GetActivePackagePlans(ctx context.Context) ([]*models.PackagePlan, error)
	GetPackagePlansByType(ctx context.Context, packageType models.PackageType) ([]*models.PackagePlan, error)
	GetPackagePlansByLevel(ctx context.Context, level models.PackageLevel) ([]*models.PackagePlan, error)
	GetPackagePlansByBillingCycle(ctx context.Context, billingCycle models.BillingCycle) ([]*models.PackagePlan, error)
	GetHotPackagePlans(ctx context.Context, limit int) ([]*models.PackagePlan, error)
	GetRecommendPackagePlans(ctx context.Context, limit int) ([]*models.PackagePlan, error)
	GetPopularPackagePlans(ctx context.Context, limit int) ([]*models.PackagePlan, error)
	UpdatePackagePlan(ctx context.Context, plan *models.PackagePlan) error
	DeletePackagePlan(ctx context.Context, id uint, updatedBy int64) error
	GetPackagePlanStats(ctx context.Context) (map[string]interface{}, error)

	// 用户订阅管理
	CreateUserSubscription(ctx context.Context, userID int64, packagePlanID uint, orderID uint, paymentMethod models.PaymentMethod, paymentProvider string) (*models.UserSubscription, error)
	GetUserSubscription(ctx context.Context, id uint) (*models.UserSubscription, error)
	GetUserActiveSubscription(ctx context.Context, userID int64) (*models.UserSubscription, error)
	GetUserSubscriptions(ctx context.Context, userID int64, offset, limit int) ([]*models.UserSubscription, error)
	GetUserSubscriptionsByStatus(ctx context.Context, userID int64, status models.UserSubscriptionStatus, offset, limit int) ([]*models.UserSubscription, error)
	UpdateUserSubscriptionStatus(ctx context.Context, id uint, status models.UserSubscriptionStatus) error
	UpdateUserSubscriptionQuota(ctx context.Context, id uint, quotaUsed int) error
	ConsumeUserSubscriptionQuota(ctx context.Context, id uint, amount int) error
	CancelUserSubscription(ctx context.Context, id uint, reason string, canceledBy int64) error
	PauseUserSubscription(ctx context.Context, id uint, reason string) error
	ResumeUserSubscription(ctx context.Context, id uint) error
	UpgradeUserSubscription(ctx context.Context, id uint, newPackagePlanID uint) error
	DowngradeUserSubscription(ctx context.Context, id uint, newPackagePlanID uint) error
	GetExpiredUserSubscriptions(ctx context.Context) ([]*models.UserSubscription, error)
	GetUserSubscriptionStats(ctx context.Context, userID int64) (map[string]interface{}, error)

	// 用户活动记录管理
	CreateUserActivity(ctx context.Context, activity *models.UserActivity) error
	GetUserActivity(ctx context.Context, id uint) (*models.UserActivity, error)
	GetUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*models.UserActivity, error)
	GetUserActivitiesByType(ctx context.Context, userID int64, activityType models.ActivityType, offset, limit int) ([]*models.UserActivity, error)
	GetUserActivitiesByLevel(ctx context.Context, userID int64, level models.ActivityLevel, offset, limit int) ([]*models.UserActivity, error)
	GetUnreadUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*models.UserActivity, error)
	GetUnresolvedUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*models.UserActivity, error)
	GetHighPriorityUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*models.UserActivity, error)
	MarkUserActivityAsRead(ctx context.Context, id uint) error
	MarkUserActivityAsResolved(ctx context.Context, id uint, resolvedBy int64, resolutionNote string) error
	GetUserActivityStats(ctx context.Context, userID int64) (map[string]interface{}, error)
	GetRecentUserActivities(ctx context.Context, userID int64, days int, limit int) ([]*models.UserActivity, error)

	// 会员权限检查
	IsUserVIP(ctx context.Context, userID int64) (bool, error)
	GetUserVIPInfo(ctx context.Context, userID int64) (*models.UserSubscription, error)
	CheckUserPermission(ctx context.Context, userID int64, permission string) (bool, error)
	ConsumeUserQuota(ctx context.Context, userID int64, amount int) error
	GetUserQuota(ctx context.Context, userID int64) (used int, limit int, err error)
	GetUserMaxRoles(ctx context.Context, userID int64) (int, error)
	GetUserMaxContexts(ctx context.Context, userID int64) (int, error)
	GetUserAvailableModels(ctx context.Context, userID int64) ([]string, error)

	// 订阅流程管理
	ProcessSubscriptionPurchase(ctx context.Context, userID int64, packagePlanID uint, paymentMethod models.PaymentMethod, paymentProvider string) (*models.Order, *models.UserSubscription, error)
	ProcessSubscriptionRenewal(ctx context.Context, subscriptionID uint) error
	ProcessSubscriptionUpgrade(ctx context.Context, subscriptionID uint, newPackagePlanID uint) error
	ProcessSubscriptionDowngrade(ctx context.Context, subscriptionID uint, newPackagePlanID uint) error
	ProcessSubscriptionCancellation(ctx context.Context, subscriptionID uint, reason string, canceledBy int64) error
	ProcessSubscriptionPause(ctx context.Context, subscriptionID uint, reason string) error
	ProcessSubscriptionResume(ctx context.Context, subscriptionID uint) error

	// 系统管理
	ProcessExpiredSubscriptions(ctx context.Context) error
	ProcessTrialExpirations(ctx context.Context) error
	ProcessAutoRenewals(ctx context.Context) error
	GetSystemStats(ctx context.Context) (map[string]interface{}, error)
	GetRevenueStats(ctx context.Context, startTime, endTime time.Time) (map[string]interface{}, error)
}

// MembershipServiceImpl 会员服务实现
type MembershipServiceImpl struct {
	paymentService PaymentService
	logger         *zap.Logger
}

// NewMembershipService 创建会员服务实例
func NewMembershipService(paymentService PaymentService) MembershipService {
	return &MembershipServiceImpl{
		paymentService: paymentService,
		logger:         log.Log(),
	}
}

// 套餐管理实现
func (m *MembershipServiceImpl) CreatePackagePlan(ctx context.Context, plan *models.PackagePlan) error {
	m.logger.Info("开始创建套餐计划",
		zap.String("package_name", plan.Name),
		zap.String("sku", plan.SKU),
		zap.Int("package_type", int(plan.PackageType)),
		zap.Int("package_level", int(plan.PackageLevel)),
		zap.String("billing_cycle", string(plan.BillingCycle)),
		zap.Int64("price", plan.Price),
		zap.String("currency", plan.Currency),
	)

	err := models.CreatePackagePlan(ctx, plan)
	if err != nil {
		m.logger.Error("创建套餐计划失败",
			zap.String("package_name", plan.Name),
			zap.String("sku", plan.SKU),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("套餐计划创建成功",
		zap.Uint("package_id", plan.ID),
		zap.String("package_name", plan.Name),
		zap.String("sku", plan.SKU),
	)
	return nil
}

func (m *MembershipServiceImpl) GetPackagePlan(ctx context.Context, id uint) (*models.PackagePlan, error) {
	m.logger.Info("开始获取套餐计划", zap.Uint("package_id", id))

	plan, err := models.GetPackagePlan(ctx, id)
	if err != nil {
		m.logger.Error("获取套餐计划失败",
			zap.Uint("package_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	if plan == nil {
		m.logger.Warn("套餐计划不存在", zap.Uint("package_id", id))
		return nil, nil
	}

	m.logger.Info("套餐计划获取成功",
		zap.Uint("package_id", id),
		zap.String("package_name", plan.Name),
		zap.String("sku", plan.SKU),
	)
	return plan, nil
}

func (m *MembershipServiceImpl) GetPackagePlanBySKU(ctx context.Context, sku string) (*models.PackagePlan, error) {
	m.logger.Info("开始根据SKU获取套餐计划", zap.String("sku", sku))

	plan, err := models.GetPackagePlanBySKU(ctx, sku)
	if err != nil {
		m.logger.Error("根据SKU获取套餐计划失败",
			zap.String("sku", sku),
			zap.Error(err),
		)
		return nil, err
	}

	if plan == nil {
		m.logger.Warn("根据SKU未找到套餐计划", zap.String("sku", sku))
		return nil, nil
	}

	m.logger.Info("根据SKU获取套餐计划成功",
		zap.String("sku", sku),
		zap.Uint("package_id", plan.ID),
		zap.String("package_name", plan.Name),
	)
	return plan, nil
}

func (m *MembershipServiceImpl) GetActivePackagePlans(ctx context.Context) ([]*models.PackagePlan, error) {
	m.logger.Info("开始获取所有活跃套餐计划")

	plans, err := models.GetActivePackagePlans(ctx)
	if err != nil {
		m.logger.Error("获取活跃套餐计划失败", zap.Error(err))
		return nil, err
	}

	m.logger.Info("获取活跃套餐计划成功",
		zap.Int("count", len(plans)),
	)
	return plans, nil
}

func (m *MembershipServiceImpl) GetPackagePlansByType(ctx context.Context, packageType models.PackageType) ([]*models.PackagePlan, error) {
	m.logger.Info("开始根据类型获取套餐计划",
		zap.Int("package_type", int(packageType)),
	)

	plans, err := models.GetPackagePlansByType(ctx, packageType)
	if err != nil {
		m.logger.Error("根据类型获取套餐计划失败",
			zap.Int("package_type", int(packageType)),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("根据类型获取套餐计划成功",
		zap.Int("package_type", int(packageType)),
		zap.Int("count", len(plans)),
	)
	return plans, nil
}

func (m *MembershipServiceImpl) GetPackagePlansByLevel(ctx context.Context, level models.PackageLevel) ([]*models.PackagePlan, error) {
	m.logger.Info("开始根据等级获取套餐计划",
		zap.Int("package_level", int(level)),
	)

	plans, err := models.GetPackagePlansByLevel(ctx, level)
	if err != nil {
		m.logger.Error("根据等级获取套餐计划失败",
			zap.Int("package_level", int(level)),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("根据等级获取套餐计划成功",
		zap.Int("package_level", int(level)),
		zap.Int("count", len(plans)),
	)
	return plans, nil
}

func (m *MembershipServiceImpl) GetPackagePlansByBillingCycle(ctx context.Context, billingCycle models.BillingCycle) ([]*models.PackagePlan, error) {
	m.logger.Info("开始根据计费周期获取套餐计划",
		zap.String("billing_cycle", string(billingCycle)),
	)

	plans, err := models.GetPackagePlansByBillingCycle(ctx, billingCycle)
	if err != nil {
		m.logger.Error("根据计费周期获取套餐计划失败",
			zap.String("billing_cycle", string(billingCycle)),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("根据计费周期获取套餐计划成功",
		zap.String("billing_cycle", string(billingCycle)),
		zap.Int("count", len(plans)),
	)
	return plans, nil
}

func (m *MembershipServiceImpl) GetHotPackagePlans(ctx context.Context, limit int) ([]*models.PackagePlan, error) {
	m.logger.Info("开始获取热门套餐计划", zap.Int("limit", limit))

	plans, err := models.GetHotPackagePlans(ctx, limit)
	if err != nil {
		m.logger.Error("获取热门套餐计划失败",
			zap.Int("limit", limit),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("获取热门套餐计划成功",
		zap.Int("limit", limit),
		zap.Int("count", len(plans)),
	)
	return plans, nil
}

func (m *MembershipServiceImpl) GetRecommendPackagePlans(ctx context.Context, limit int) ([]*models.PackagePlan, error) {
	m.logger.Info("开始获取推荐套餐计划", zap.Int("limit", limit))

	plans, err := models.GetRecommendPackagePlans(ctx, limit)
	if err != nil {
		m.logger.Error("获取推荐套餐计划失败",
			zap.Int("limit", limit),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("获取推荐套餐计划成功",
		zap.Int("limit", limit),
		zap.Int("count", len(plans)),
	)
	return plans, nil
}

func (m *MembershipServiceImpl) GetPopularPackagePlans(ctx context.Context, limit int) ([]*models.PackagePlan, error) {
	m.logger.Info("开始获取畅销套餐计划", zap.Int("limit", limit))

	plans, err := models.GetPopularPackagePlans(ctx, limit)
	if err != nil {
		m.logger.Error("获取畅销套餐计划失败",
			zap.Int("limit", limit),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("获取畅销套餐计划成功",
		zap.Int("limit", limit),
		zap.Int("count", len(plans)),
	)
	return plans, nil
}

func (m *MembershipServiceImpl) UpdatePackagePlan(ctx context.Context, plan *models.PackagePlan) error {
	m.logger.Info("开始更新套餐计划",
		zap.Uint("package_id", plan.ID),
		zap.String("package_name", plan.Name),
		zap.String("sku", plan.SKU),
	)

	err := models.UpdatePackagePlan(ctx, plan)
	if err != nil {
		m.logger.Error("更新套餐计划失败",
			zap.Uint("package_id", plan.ID),
			zap.String("package_name", plan.Name),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("套餐计划更新成功",
		zap.Uint("package_id", plan.ID),
		zap.String("package_name", plan.Name),
	)
	return nil
}

func (m *MembershipServiceImpl) DeletePackagePlan(ctx context.Context, id uint, updatedBy int64) error {
	m.logger.Info("开始删除套餐计划",
		zap.Uint("package_id", id),
		zap.Int64("updated_by", updatedBy),
	)

	err := models.DeletePackagePlan(ctx, id, updatedBy)
	if err != nil {
		m.logger.Error("删除套餐计划失败",
			zap.Uint("package_id", id),
			zap.Int64("updated_by", updatedBy),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("套餐计划删除成功",
		zap.Uint("package_id", id),
		zap.Int64("updated_by", updatedBy),
	)
	return nil
}

func (m *MembershipServiceImpl) GetPackagePlanStats(ctx context.Context) (map[string]interface{}, error) {
	m.logger.Info("开始获取套餐计划统计信息")

	stats, err := models.GetPackagePlanStats(ctx)
	if err != nil {
		m.logger.Error("获取套餐计划统计信息失败", zap.Error(err))
		return nil, err
	}

	m.logger.Info("获取套餐计划统计信息成功",
		zap.Any("stats", stats),
	)
	return stats, nil
}

// 用户订阅管理实现
func (m *MembershipServiceImpl) CreateUserSubscription(ctx context.Context, userID int64, packagePlanID uint, orderID uint, paymentMethod models.PaymentMethod, paymentProvider string) (*models.UserSubscription, error) {
	m.logger.Info("开始创建用户订阅",
		zap.Int64("user_id", userID),
		zap.Uint("package_plan_id", packagePlanID),
		zap.Uint("order_id", orderID),
		zap.Int("payment_method", int(paymentMethod)),
		zap.String("payment_provider", paymentProvider),
	)

	// 获取套餐计划信息
	packagePlan, err := models.GetPackagePlan(ctx, packagePlanID)
	if err != nil {
		m.logger.Error("获取套餐计划失败",
			zap.Int64("user_id", userID),
			zap.Uint("package_plan_id", packagePlanID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get package plan: %w", err)
	}

	m.logger.Info("成功获取套餐计划信息",
		zap.Uint("package_plan_id", packagePlanID),
		zap.String("package_name", packagePlan.Name),
		zap.Int("free_trial_days", packagePlan.FreeTrialDays),
		zap.Int64("duration", packagePlan.Duration),
	)

	// 获取订单信息
	order, err := models.GetOrder(ctx, orderID)
	if err != nil {
		m.logger.Error("获取订单信息失败",
			zap.Int64("user_id", userID),
			zap.Uint("order_id", orderID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	m.logger.Info("成功获取订单信息",
		zap.Uint("order_id", orderID),
		zap.Int64("total_amount", order.TotalAmount),
		zap.String("currency", order.Currency),
	)

	// 计算订阅时间
	now := time.Now()
	var startTime, endTime time.Time
	var trialStartTime, trialEndTime *time.Time

	if packagePlan.FreeTrialDays > 0 {
		trialStartTime = &now
		trialEnd := now.AddDate(0, 0, packagePlan.FreeTrialDays)
		trialEndTime = &trialEnd
		startTime = trialEnd
		m.logger.Info("设置试用期",
			zap.Int64("user_id", userID),
			zap.Int("free_trial_days", packagePlan.FreeTrialDays),
			zap.Time("trial_start_time", now),
			zap.Time("trial_end_time", trialEnd),
		)
	} else {
		startTime = now
		m.logger.Info("无试用期，直接开始订阅",
			zap.Int64("user_id", userID),
			zap.Time("start_time", now),
		)
	}

	if packagePlan.Duration > 0 {
		endTime = startTime.Add(time.Duration(packagePlan.Duration) * time.Second)
		m.logger.Info("设置订阅结束时间",
			zap.Int64("user_id", userID),
			zap.Int64("duration_seconds", packagePlan.Duration),
			zap.Time("end_time", endTime),
		)
	} else {
		// 永久订阅
		endTime = time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)
		m.logger.Info("设置为永久订阅",
			zap.Int64("user_id", userID),
			zap.Time("end_time", endTime),
		)
	}

	// 创建用户订阅
	subscription := &models.UserSubscription{
		UserID:          userID,
		PackagePlanID:   packagePlanID,
		OrderID:         orderID,
		Status:          models.UserSubscriptionStatusActive,
		StartTime:       startTime,
		EndTime:         endTime,
		AutoRenew:       true,
		PaymentMethod:   paymentMethod,
		PaymentProvider: paymentProvider,
		Amount:          order.TotalAmount,
		Currency:        order.Currency,
		QuotaLimit:      packagePlan.QuotaLimit,
		QuotaUsed:       0,
		MaxRoles:        packagePlan.MaxRoles,
		MaxContexts:     packagePlan.MaxContexts,
		AvailableModels: packagePlan.AvailableModels,
		Features:        packagePlan.Features,
		TrialStartTime:  trialStartTime,
		TrialEndTime:    trialEndTime,
	}

	m.logger.Info("开始创建用户订阅记录",
		zap.Int64("user_id", userID),
		zap.Uint("package_plan_id", packagePlanID),
		zap.Uint("order_id", orderID),
		zap.Int("quota_limit", packagePlan.QuotaLimit),
		zap.Int("max_roles", packagePlan.MaxRoles),
		zap.Int("max_contexts", packagePlan.MaxContexts),
	)

	if err := models.CreateUserSubscription(ctx, subscription); err != nil {
		m.logger.Error("创建用户订阅失败",
			zap.Int64("user_id", userID),
			zap.Uint("package_plan_id", packagePlanID),
			zap.Uint("order_id", orderID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create user subscription: %w", err)
	}

	m.logger.Info("用户订阅创建成功",
		zap.Uint("subscription_id", subscription.ID),
		zap.Int64("user_id", userID),
		zap.Uint("package_plan_id", packagePlanID),
		zap.Uint("order_id", orderID),
	)

	// 创建活动记录
	activityTitle := "订阅创建"
	activityDesc := fmt.Sprintf("成功创建套餐订阅：%s", packagePlan.Name)
	if packagePlan.FreeTrialDays > 0 {
		activityTitle = "试用开始"
		activityDesc = fmt.Sprintf("开始试用套餐：%s，试用期%d天", packagePlan.Name, packagePlan.FreeTrialDays)
	}

	m.logger.Info("开始创建订阅活动记录",
		zap.Int64("user_id", userID),
		zap.Uint("subscription_id", subscription.ID),
		zap.String("activity_title", activityTitle),
	)

	if err := models.CreateSubscriptionActivity(ctx, userID, models.ActivityTypeSubscriptionCreated, subscription.ID, activityTitle, activityDesc, order.TotalAmount); err != nil {
		m.logger.Warn("创建订阅活动记录失败",
			zap.Int64("user_id", userID),
			zap.Uint("subscription_id", subscription.ID),
			zap.Error(err),
		)
	} else {
		m.logger.Info("订阅活动记录创建成功",
			zap.Int64("user_id", userID),
			zap.Uint("subscription_id", subscription.ID),
		)
	}

	// 更新套餐销售数量
	m.logger.Info("开始更新套餐销售数量",
		zap.Uint("package_plan_id", packagePlanID),
	)

	if err := models.IncrementPackagePlanSoldCount(ctx, packagePlanID); err != nil {
		m.logger.Warn("更新套餐销售数量失败",
			zap.Uint("package_plan_id", packagePlanID),
			zap.Error(err),
		)
	} else {
		m.logger.Info("套餐销售数量更新成功",
			zap.Uint("package_plan_id", packagePlanID),
		)
	}

	m.logger.Info("用户订阅创建流程完成",
		zap.Int64("user_id", userID),
		zap.Uint("package_plan_id", packagePlanID),
		zap.Uint("subscription_id", subscription.ID),
		zap.Uint("order_id", orderID),
	)
	return subscription, nil
}

func (m *MembershipServiceImpl) GetUserSubscription(ctx context.Context, id uint) (*models.UserSubscription, error) {
	m.logger.Info("开始获取用户订阅", zap.Uint("subscription_id", id))

	subscription, err := models.GetUserSubscription(ctx, id)
	if err != nil {
		m.logger.Error("获取用户订阅失败",
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	if subscription == nil {
		m.logger.Warn("用户订阅不存在", zap.Uint("subscription_id", id))
		return nil, nil
	}

	m.logger.Info("用户订阅获取成功",
		zap.Uint("subscription_id", id),
		zap.Int64("user_id", subscription.UserID),
		zap.Uint("package_plan_id", subscription.PackagePlanID),
		zap.Int("status", int(subscription.Status)),
	)
	return subscription, nil
}

func (m *MembershipServiceImpl) GetUserActiveSubscription(ctx context.Context, userID int64) (*models.UserSubscription, error) {
	m.logger.Info("开始获取用户活跃订阅", zap.Int64("user_id", userID))

	subscription, err := models.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		m.logger.Error("获取用户活跃订阅失败",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	if subscription == nil {
		m.logger.Info("用户无活跃订阅", zap.Int64("user_id", userID))
		return nil, nil
	}

	m.logger.Info("用户活跃订阅获取成功",
		zap.Int64("user_id", userID),
		zap.Uint("subscription_id", subscription.ID),
		zap.Uint("package_plan_id", subscription.PackagePlanID),
		zap.Time("end_time", subscription.EndTime),
	)
	return subscription, nil
}

func (m *MembershipServiceImpl) GetUserSubscriptions(ctx context.Context, userID int64, offset, limit int) ([]*models.UserSubscription, error) {
	m.logger.Info("开始获取用户所有订阅",
		zap.Int64("user_id", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
	)

	subscriptions, err := models.GetUserSubscriptionsByUserID(ctx, userID, offset, limit)
	if err != nil {
		m.logger.Error("获取用户所有订阅失败",
			zap.Int64("user_id", userID),
			zap.Int("offset", offset),
			zap.Int("limit", limit),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("用户所有订阅获取成功",
		zap.Int64("user_id", userID),
		zap.Int("count", len(subscriptions)),
	)
	return subscriptions, nil
}

func (m *MembershipServiceImpl) GetUserSubscriptionsByStatus(ctx context.Context, userID int64, status models.UserSubscriptionStatus, offset, limit int) ([]*models.UserSubscription, error) {
	m.logger.Info("开始根据状态获取用户订阅",
		zap.Int64("user_id", userID),
		zap.Int("status", int(status)),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
	)

	subscriptions, err := models.GetUserSubscriptionsByStatus(ctx, userID, status, offset, limit)
	if err != nil {
		m.logger.Error("根据状态获取用户订阅失败",
			zap.Int64("user_id", userID),
			zap.Int("status", int(status)),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("根据状态获取用户订阅成功",
		zap.Int64("user_id", userID),
		zap.Int("status", int(status)),
		zap.Int("count", len(subscriptions)),
	)
	return subscriptions, nil
}

func (m *MembershipServiceImpl) UpdateUserSubscriptionStatus(ctx context.Context, id uint, status models.UserSubscriptionStatus) error {
	m.logger.Info("开始更新用户订阅状态",
		zap.Uint("subscription_id", id),
		zap.Int("status", int(status)),
	)

	err := models.UpdateUserSubscriptionStatus(ctx, id, status)
	if err != nil {
		m.logger.Error("更新用户订阅状态失败",
			zap.Uint("subscription_id", id),
			zap.Int("status", int(status)),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("用户订阅状态更新成功",
		zap.Uint("subscription_id", id),
		zap.Int("status", int(status)),
	)
	return nil
}

func (m *MembershipServiceImpl) UpdateUserSubscriptionQuota(ctx context.Context, id uint, quotaUsed int) error {
	m.logger.Info("开始更新用户订阅额度",
		zap.Uint("subscription_id", id),
		zap.Int("quota_used", quotaUsed),
	)

	err := models.UpdateUserSubscriptionQuota(ctx, id, quotaUsed)
	if err != nil {
		m.logger.Error("更新用户订阅额度失败",
			zap.Uint("subscription_id", id),
			zap.Int("quota_used", quotaUsed),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("用户订阅额度更新成功",
		zap.Uint("subscription_id", id),
		zap.Int("quota_used", quotaUsed),
	)
	return nil
}

func (m *MembershipServiceImpl) ConsumeUserSubscriptionQuota(ctx context.Context, id uint, amount int) error {
	m.logger.Info("开始消耗用户订阅额度",
		zap.Uint("subscription_id", id),
		zap.Int("amount", amount),
	)

	subscription, err := models.GetUserSubscription(ctx, id)
	if err != nil {
		m.logger.Error("获取用户订阅失败",
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get user subscription: %w", err)
	}

	m.logger.Info("获取用户订阅成功，检查额度",
		zap.Uint("subscription_id", id),
		zap.Int("current_quota_used", subscription.QuotaUsed),
		zap.Int("quota_limit", subscription.QuotaLimit),
		zap.Int("requested_amount", amount),
	)

	if !subscription.HasQuota() {
		m.logger.Warn("用户订阅额度不足",
			zap.Uint("subscription_id", id),
			zap.Int("current_quota_used", subscription.QuotaUsed),
			zap.Int("quota_limit", subscription.QuotaLimit),
			zap.Int("requested_amount", amount),
		)
		return fmt.Errorf("insufficient quota: used=%d, limit=%d", subscription.QuotaUsed, subscription.QuotaLimit)
	}

	if err := models.ConsumeUserSubscriptionQuota(ctx, id, amount); err != nil {
		m.logger.Error("消耗用户订阅额度失败",
			zap.Uint("subscription_id", id),
			zap.Int("amount", amount),
			zap.Error(err),
		)
		return fmt.Errorf("failed to consume quota: %w", err)
	}

	m.logger.Info("用户订阅额度消耗成功",
		zap.Uint("subscription_id", id),
		zap.Int("amount", amount),
		zap.Int("remaining_quota", subscription.GetRemainingQuota()-amount),
	)

	// 创建额度消耗活动记录
	activityTitle := "额度消耗"
	activityDesc := fmt.Sprintf("消耗额度：%d，剩余额度：%d", amount, subscription.GetRemainingQuota()-amount)

	m.logger.Info("开始创建额度消耗活动记录",
		zap.Int64("user_id", subscription.UserID),
		zap.Uint("subscription_id", id),
		zap.String("activity_title", activityTitle),
	)

	if err := models.CreateQuotaActivity(ctx, subscription.UserID, models.ActivityTypeQuotaConsumed, id, activityTitle, activityDesc, amount); err != nil {
		m.logger.Warn("创建额度消耗活动记录失败",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
	} else {
		m.logger.Info("额度消耗活动记录创建成功",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) CancelUserSubscription(ctx context.Context, id uint, reason string, canceledBy int64) error {
	m.logger.Info("开始取消用户订阅",
		zap.Uint("subscription_id", id),
		zap.String("reason", reason),
		zap.Int64("canceled_by", canceledBy),
	)

	subscription, err := models.GetUserSubscription(ctx, id)
	if err != nil {
		m.logger.Error("获取用户订阅失败",
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get user subscription: %w", err)
	}

	m.logger.Info("获取用户订阅成功，开始取消",
		zap.Uint("subscription_id", id),
		zap.Int64("user_id", subscription.UserID),
		zap.String("reason", reason),
	)

	if err := models.CancelUserSubscription(ctx, id, reason, canceledBy); err != nil {
		m.logger.Error("取消用户订阅失败",
			zap.Uint("subscription_id", id),
			zap.String("reason", reason),
			zap.Error(err),
		)
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	m.logger.Info("用户订阅取消成功",
		zap.Uint("subscription_id", id),
		zap.Int64("user_id", subscription.UserID),
		zap.String("reason", reason),
	)

	// 创建订阅取消活动记录
	activityTitle := "订阅取消"
	activityDesc := fmt.Sprintf("取消订阅，原因：%s", reason)

	m.logger.Info("开始创建订阅取消活动记录",
		zap.Int64("user_id", subscription.UserID),
		zap.Uint("subscription_id", id),
		zap.String("activity_title", activityTitle),
	)

	if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionCanceled, id, activityTitle, activityDesc, 0); err != nil {
		m.logger.Warn("创建订阅取消活动记录失败",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
	} else {
		m.logger.Info("订阅取消活动记录创建成功",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) PauseUserSubscription(ctx context.Context, id uint, reason string) error {
	subscription, err := models.GetUserSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user subscription: %w", err)
	}

	if err := models.PauseUserSubscription(ctx, id, reason); err != nil {
		return fmt.Errorf("failed to pause subscription: %w", err)
	}

	// 创建订阅暂停活动记录
	activityTitle := "订阅暂停"
	activityDesc := fmt.Sprintf("暂停订阅，原因：%s", reason)
	if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionPaused, id, activityTitle, activityDesc, 0); err != nil {
		m.logger.Warn("创建订阅暂停活动记录失败",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
	} else {
		m.logger.Info("订阅暂停活动记录创建成功",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) ResumeUserSubscription(ctx context.Context, id uint) error {
	subscription, err := models.GetUserSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user subscription: %w", err)
	}

	if err := models.ResumeUserSubscription(ctx, id); err != nil {
		return fmt.Errorf("failed to resume subscription: %w", err)
	}

	// 创建订阅恢复活动记录
	activityTitle := "订阅恢复"
	activityDesc := "恢复订阅服务"
	if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionResumed, id, activityTitle, activityDesc, 0); err != nil {
		m.logger.Warn("Failed to create subscription activity",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) UpgradeUserSubscription(ctx context.Context, id uint, newPackagePlanID uint) error {
	subscription, err := models.GetUserSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user subscription: %w", err)
	}

	if err := models.UpgradeUserSubscription(ctx, id, newPackagePlanID); err != nil {
		return fmt.Errorf("failed to upgrade subscription: %w", err)
	}

	// 创建订阅升级活动记录
	activityTitle := "订阅升级"
	activityDesc := fmt.Sprintf("升级订阅套餐，从套餐ID %d 升级到套餐ID %d", subscription.PackagePlanID, newPackagePlanID)
	if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionUpgraded, id, activityTitle, activityDesc, 0); err != nil {
		m.logger.Warn("Failed to create subscription activity",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) DowngradeUserSubscription(ctx context.Context, id uint, newPackagePlanID uint) error {
	subscription, err := models.GetUserSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user subscription: %w", err)
	}

	if err := models.DowngradeUserSubscription(ctx, id, newPackagePlanID); err != nil {
		return fmt.Errorf("failed to downgrade subscription: %w", err)
	}

	// 创建订阅降级活动记录
	activityTitle := "订阅降级"
	activityDesc := fmt.Sprintf("降级订阅套餐，从套餐ID %d 降级到套餐ID %d", subscription.PackagePlanID, newPackagePlanID)
	if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionDowngraded, id, activityTitle, activityDesc, 0); err != nil {
		m.logger.Warn("Failed to create subscription activity",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", id),
			zap.Error(err),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) GetExpiredUserSubscriptions(ctx context.Context) ([]*models.UserSubscription, error) {
	return models.GetExpiredUserSubscriptions(ctx)
}

func (m *MembershipServiceImpl) GetUserSubscriptionStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	return models.GetUserSubscriptionStats(ctx, userID)
}

// 用户活动记录管理实现
func (m *MembershipServiceImpl) CreateUserActivity(ctx context.Context, activity *models.UserActivity) error {
	return models.CreateUserActivity(ctx, activity)
}

func (m *MembershipServiceImpl) GetUserActivity(ctx context.Context, id uint) (*models.UserActivity, error) {
	return models.GetUserActivity(ctx, id)
}

func (m *MembershipServiceImpl) GetUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*models.UserActivity, error) {
	return models.GetUserActivities(ctx, userID, offset, limit)
}

func (m *MembershipServiceImpl) GetUserActivitiesByType(ctx context.Context, userID int64, activityType models.ActivityType, offset, limit int) ([]*models.UserActivity, error) {
	return models.GetUserActivitiesByType(ctx, userID, activityType, offset, limit)
}

func (m *MembershipServiceImpl) GetUserActivitiesByLevel(ctx context.Context, userID int64, level models.ActivityLevel, offset, limit int) ([]*models.UserActivity, error) {
	return models.GetUserActivitiesByLevel(ctx, userID, level, offset, limit)
}

func (m *MembershipServiceImpl) GetUnreadUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*models.UserActivity, error) {
	return models.GetUnreadUserActivities(ctx, userID, offset, limit)
}

func (m *MembershipServiceImpl) GetUnresolvedUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*models.UserActivity, error) {
	return models.GetUnresolvedUserActivities(ctx, userID, offset, limit)
}

func (m *MembershipServiceImpl) GetHighPriorityUserActivities(ctx context.Context, userID int64, offset, limit int) ([]*models.UserActivity, error) {
	return models.GetHighPriorityUserActivities(ctx, userID, offset, limit)
}

func (m *MembershipServiceImpl) MarkUserActivityAsRead(ctx context.Context, id uint) error {
	return models.MarkUserActivityAsRead(ctx, id)
}

func (m *MembershipServiceImpl) MarkUserActivityAsResolved(ctx context.Context, id uint, resolvedBy int64, resolutionNote string) error {
	return models.MarkUserActivityAsResolved(ctx, id, resolvedBy, resolutionNote)
}

func (m *MembershipServiceImpl) GetUserActivityStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	return models.GetUserActivityStats(ctx, userID)
}

func (m *MembershipServiceImpl) GetRecentUserActivities(ctx context.Context, userID int64, days int, limit int) ([]*models.UserActivity, error) {
	return models.GetRecentUserActivities(ctx, userID, days, limit)
}

// 会员权限检查实现
func (m *MembershipServiceImpl) IsUserVIP(ctx context.Context, userID int64) (bool, error) {
	subscription, err := models.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	return subscription != nil && subscription.IsActive(), nil
}

func (m *MembershipServiceImpl) GetUserVIPInfo(ctx context.Context, userID int64) (*models.UserSubscription, error) {
	return models.GetUserActiveSubscriptionByUserID(ctx, userID)
}

func (m *MembershipServiceImpl) CheckUserPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	subscription, err := models.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if subscription == nil || !subscription.IsActive() {
		return false, nil
	}

	// 根据权限类型进行检查
	switch permission {
	case "ai_generation":
		return subscription.HasQuota(), nil
	case "advanced_models":
		models, err := subscription.GetAvailableModels()
		if err != nil {
			return false, err
		}
		return len(models) > 1, nil // 有多个模型可用
	case "unlimited_contexts":
		return subscription.MaxContexts > 10, nil
	case "multiple_roles":
		return subscription.MaxRoles > 2, nil
	default:
		return false, fmt.Errorf("unknown permission: %s", permission)
	}
}

func (m *MembershipServiceImpl) ConsumeUserQuota(ctx context.Context, userID int64, amount int) error {
	subscription, err := models.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if subscription == nil {
		return fmt.Errorf("no active subscription found for user: %d", userID)
	}
	return m.ConsumeUserSubscriptionQuota(ctx, subscription.ID, amount)
}

func (m *MembershipServiceImpl) GetUserQuota(ctx context.Context, userID int64) (used int, limit int, err error) {
	subscription, err := models.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	if subscription == nil {
		return 0, 0, nil
	}
	return subscription.QuotaUsed, subscription.QuotaLimit, nil
}

func (m *MembershipServiceImpl) GetUserMaxRoles(ctx context.Context, userID int64) (int, error) {
	subscription, err := models.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if subscription == nil {
		return 1, nil // 默认值
	}
	return subscription.MaxRoles, nil
}

func (m *MembershipServiceImpl) GetUserMaxContexts(ctx context.Context, userID int64) (int, error) {
	subscription, err := models.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if subscription == nil {
		return 5, nil // 默认值
	}
	return subscription.MaxContexts, nil
}

func (m *MembershipServiceImpl) GetUserAvailableModels(ctx context.Context, userID int64) ([]string, error) {
	subscription, err := models.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		return []string{"gpt-3.5-turbo"}, err // 默认模型
	}
	if subscription == nil {
		return []string{"gpt-3.5-turbo"}, nil // 默认模型
	}
	return subscription.GetAvailableModels()
}

// 订阅流程管理实现
func (m *MembershipServiceImpl) ProcessSubscriptionPurchase(ctx context.Context, userID int64, packagePlanID uint, paymentMethod models.PaymentMethod, paymentProvider string) (*models.Order, *models.UserSubscription, error) {
	// 获取套餐计划
	packagePlan, err := models.GetPackagePlan(ctx, packagePlanID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get package plan: %w", err)
	}
	_ = packagePlan

	// 创建订单
	order, err := m.paymentService.CreateOrder(ctx, userID, uint(packagePlanID), nil, 1, paymentMethod)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 创建用户订阅
	subscription, err := m.CreateUserSubscription(ctx, userID, packagePlanID, order.ID, paymentMethod, paymentProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create user subscription: %w", err)
	}

	return order, subscription, nil
}

func (m *MembershipServiceImpl) ProcessSubscriptionRenewal(ctx context.Context, subscriptionID uint) error {
	subscription, err := models.GetUserSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// 检查是否需要续费
	if !subscription.AutoRenew || subscription.IsExpired() {
		return fmt.Errorf("subscription is not eligible for renewal")
	}

	// 获取套餐计划
	packagePlan, err := models.GetPackagePlan(ctx, subscription.PackagePlanID)
	if err != nil {
		return fmt.Errorf("failed to get package plan: %w", err)
	}

	// 创建续费订单
	order, err := m.paymentService.CreateOrder(ctx, subscription.UserID, subscription.PackagePlanID, nil, 1, subscription.PaymentMethod)
	if err != nil {
		return fmt.Errorf("failed to create renewal order: %w", err)
	}

	// 更新订阅时间
	now := time.Now()
	subscription.StartTime = now
	if packagePlan.Duration > 0 {
		subscription.EndTime = now.Add(time.Duration(packagePlan.Duration) * time.Second)
	}
	subscription.OrderID = order.ID
	subscription.Amount = order.TotalAmount

	if err := models.UpdateUserSubscription(ctx, subscription); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// 创建续费活动记录
	activityTitle := "订阅续费"
	activityDesc := fmt.Sprintf("成功续费套餐：%s", packagePlan.Name)
	if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionRenewed, subscriptionID, activityTitle, activityDesc, order.TotalAmount); err != nil {
		m.logger.Warn("Failed to create subscription activity",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", subscriptionID),
			zap.Error(err),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) ProcessSubscriptionUpgrade(ctx context.Context, subscriptionID uint, newPackagePlanID uint) error {
	subscription, err := models.GetUserSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// 获取新套餐计划
	newPackagePlan, err := models.GetPackagePlan(ctx, newPackagePlanID)
	if err != nil {
		return fmt.Errorf("failed to get new package plan: %w", err)
	}

	// 检查是否可以升级
	if newPackagePlan.PackageLevel <= models.PackageLevel(newPackagePlan.PackageLevel) {
		return fmt.Errorf("cannot upgrade to same or lower level package")
	}

	// 创建升级订单
	order, err := m.paymentService.CreateOrder(ctx, subscription.UserID, newPackagePlanID, nil, 1, subscription.PaymentMethod)
	if err != nil {
		return fmt.Errorf("failed to create upgrade order: %w", err)
	}

	// 更新订阅信息
	subscription.PackagePlanID = newPackagePlanID
	subscription.OrderID = order.ID
	subscription.Amount = order.TotalAmount
	subscription.QuotaLimit = newPackagePlan.QuotaLimit
	subscription.MaxRoles = newPackagePlan.MaxRoles
	subscription.MaxContexts = newPackagePlan.MaxContexts
	subscription.AvailableModels = newPackagePlan.AvailableModels
	subscription.Features = newPackagePlan.Features
	subscription.Status = models.UserSubscriptionStatusActive

	if err := models.UpdateUserSubscription(ctx, subscription); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// 创建升级活动记录
	activityTitle := "订阅升级"
	activityDesc := fmt.Sprintf("升级到套餐：%s", newPackagePlan.Name)
	if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionUpgraded, subscriptionID, activityTitle, activityDesc, order.TotalAmount); err != nil {
		m.logger.Warn("Failed to create subscription activity",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", subscriptionID),
			zap.Error(err),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) ProcessSubscriptionDowngrade(ctx context.Context, subscriptionID uint, newPackagePlanID uint) error {
	subscription, err := models.GetUserSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// 获取新套餐计划
	newPackagePlan, err := models.GetPackagePlan(ctx, newPackagePlanID)
	if err != nil {
		return fmt.Errorf("failed to get new package plan: %w", err)
	}

	// 检查是否可以降级
	if newPackagePlan.PackageLevel >= models.PackageLevel(newPackagePlan.PackageLevel) {
		return fmt.Errorf("cannot downgrade to same or higher level package")
	}

	// 更新订阅信息（降级通常不需要额外付费）
	subscription.PackagePlanID = newPackagePlanID
	subscription.QuotaLimit = newPackagePlan.QuotaLimit
	subscription.MaxRoles = newPackagePlan.MaxRoles
	subscription.MaxContexts = newPackagePlan.MaxContexts
	subscription.AvailableModels = newPackagePlan.AvailableModels
	subscription.Features = newPackagePlan.Features
	subscription.Status = models.UserSubscriptionStatusActive

	if err := models.UpdateUserSubscription(ctx, subscription); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// 创建降级活动记录
	activityTitle := "订阅降级"
	activityDesc := fmt.Sprintf("降级到套餐：%s", newPackagePlan.Name)
	if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionDowngraded, subscriptionID, activityTitle, activityDesc, 0); err != nil {
		m.logger.Warn("Failed to create subscription activity",
			zap.Int64("user_id", subscription.UserID),
			zap.Uint("subscription_id", subscriptionID),
			zap.Error(err),
		)
	}

	return nil
}

func (m *MembershipServiceImpl) ProcessSubscriptionCancellation(ctx context.Context, subscriptionID uint, reason string, canceledBy int64) error {
	return m.CancelUserSubscription(ctx, subscriptionID, reason, canceledBy)
}

func (m *MembershipServiceImpl) ProcessSubscriptionPause(ctx context.Context, subscriptionID uint, reason string) error {
	return m.PauseUserSubscription(ctx, subscriptionID, reason)
}

func (m *MembershipServiceImpl) ProcessSubscriptionResume(ctx context.Context, subscriptionID uint) error {
	return m.ResumeUserSubscription(ctx, subscriptionID)
}

// 系统管理实现
func (m *MembershipServiceImpl) ProcessExpiredSubscriptions(ctx context.Context) error {
	expiredSubscriptions, err := models.GetExpiredUserSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get expired subscriptions: %w", err)
	}

	for _, subscription := range expiredSubscriptions {
		if err := models.UpdateUserSubscriptionStatus(ctx, subscription.ID, models.UserSubscriptionStatusExpired); err != nil {
			m.logger.Error("Failed to update expired subscription",
				zap.Uint("subscription_id", subscription.ID),
				zap.Error(err),
			)
			continue
		}

		// 创建过期活动记录
		activityTitle := "订阅过期"
		activityDesc := "订阅已过期，服务已暂停"
		if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeSubscriptionExpired, subscription.ID, activityTitle, activityDesc, 0); err != nil {
			m.logger.Warn("Failed to create subscription activity",
				zap.Int64("user_id", subscription.UserID),
				zap.Uint("subscription_id", subscription.ID),
				zap.Error(err),
			)
		}
	}

	m.logger.Info("Processed expired subscriptions",
		zap.Int("count", len(expiredSubscriptions)),
	)
	return nil
}

func (m *MembershipServiceImpl) ProcessTrialExpirations(ctx context.Context) error {
	// 获取试用期即将到期的订阅
	var trialSubscriptions []*models.UserSubscription
	err := models.DataBase().WithContext(ctx).
		Where("trial_end_time <= ? AND trial_end_time > ? AND status = ?",
			time.Now().AddDate(0, 0, 1), time.Now(), models.UserSubscriptionStatusActive).
		Find(&trialSubscriptions).Error
	if err != nil {
		return fmt.Errorf("failed to get trial subscriptions: %w", err)
	}

	for _, subscription := range trialSubscriptions {
		// 创建试用结束活动记录
		activityTitle := "试用结束"
		activityDesc := "免费试用期已结束，请升级到付费套餐继续使用"
		if err := models.CreateSubscriptionActivity(ctx, subscription.UserID, models.ActivityTypeTrialEnded, subscription.ID, activityTitle, activityDesc, 0); err != nil {
			m.logger.Warn("Failed to create subscription activity",
				zap.Int64("user_id", subscription.UserID),
				zap.Uint("subscription_id", subscription.ID),
				zap.Error(err),
			)
		}
	}

	m.logger.Info("Processed trial expirations",
		zap.Int("count", len(trialSubscriptions)),
	)
	return nil
}

func (m *MembershipServiceImpl) ProcessAutoRenewals(ctx context.Context) error {
	// 获取需要自动续费的订阅
	var autoRenewSubscriptions []*models.UserSubscription
	err := models.DataBase().WithContext(ctx).
		Where("auto_renew = ? AND status = ? AND end_time <= ? AND end_time > ?",
			true, models.UserSubscriptionStatusActive, time.Now().AddDate(0, 0, 1), time.Now()).
		Find(&autoRenewSubscriptions).Error
	if err != nil {
		return fmt.Errorf("failed to get auto-renew subscriptions: %w", err)
	}

	for _, subscription := range autoRenewSubscriptions {
		if err := m.ProcessSubscriptionRenewal(ctx, subscription.ID); err != nil {
			m.logger.Error("Failed to process auto-renewal for subscription",
				zap.Uint("subscription_id", subscription.ID),
				zap.Error(err),
			)
			continue
		}
	}

	m.logger.Info("Processed auto-renewals",
		zap.Int("count", len(autoRenewSubscriptions)),
	)
	return nil
}

func (m *MembershipServiceImpl) GetSystemStats(ctx context.Context) (map[string]interface{}, error) {
	// 获取套餐统计
	packageStats, err := models.GetPackagePlanStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get package stats: %w", err)
	}

	// 获取订阅统计
	var subscriptionStats struct {
		TotalSubscriptions   int64 `json:"total_subscriptions"`
		ActiveSubscriptions  int64 `json:"active_subscriptions"`
		ExpiredSubscriptions int64 `json:"expired_subscriptions"`
		TrialSubscriptions   int64 `json:"trial_subscriptions"`
	}

	err = models.DataBase().WithContext(ctx).
		Model(&models.UserSubscription{}).
		Select(`
			COUNT(*) as total_subscriptions,
			SUM(CASE WHEN status = 1 AND end_time > NOW() THEN 1 ELSE 0 END) as active_subscriptions,
			SUM(CASE WHEN end_time <= NOW() THEN 1 ELSE 0 END) as expired_subscriptions,
			SUM(CASE WHEN trial_end_time IS NOT NULL AND trial_end_time > NOW() THEN 1 ELSE 0 END) as trial_subscriptions
		`).
		Scan(&subscriptionStats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription stats: %w", err)
	}

	// 获取活动统计
	var activityStats struct {
		TotalActivities        int64 `json:"total_activities"`
		UnresolvedActivities   int64 `json:"unresolved_activities"`
		HighPriorityActivities int64 `json:"high_priority_activities"`
	}

	err = models.DataBase().WithContext(ctx).
		Model(&models.UserActivity{}).
		Select(`
			COUNT(*) as total_activities,
			SUM(CASE WHEN is_resolved = 0 THEN 1 ELSE 0 END) as unresolved_activities,
			SUM(CASE WHEN activity_level IN ('error', 'warning') THEN 1 ELSE 0 END) as high_priority_activities
		`).
		Scan(&activityStats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get activity stats: %w", err)
	}

	return map[string]interface{}{
		"package_stats":      packageStats,
		"subscription_stats": subscriptionStats,
		"activity_stats":     activityStats,
	}, nil
}

func (m *MembershipServiceImpl) GetRevenueStats(ctx context.Context, startTime, endTime time.Time) (map[string]interface{}, error) {
	// 获取收入统计
	var revenueStats struct {
		TotalRevenue        int64 `json:"total_revenue"`
		SubscriptionRevenue int64 `json:"subscription_revenue"`
		OneTimeRevenue      int64 `json:"one_time_revenue"`
		RefundAmount        int64 `json:"refund_amount"`
		NetRevenue          int64 `json:"net_revenue"`
		TotalOrders         int64 `json:"total_orders"`
		SuccessfulOrders    int64 `json:"successful_orders"`
		FailedOrders        int64 `json:"failed_orders"`
	}

	err := models.DataBase().WithContext(ctx).
		Model(&models.Order{}).
		Select(`
			SUM(total_amount) as total_revenue,
			SUM(CASE WHEN product_id IN (SELECT id FROM products WHERE product_type = 1) THEN total_amount ELSE 0 END) as subscription_revenue,
			SUM(CASE WHEN product_id IN (SELECT id FROM products WHERE product_type = 2) THEN total_amount ELSE 0 END) as one_time_revenue,
			SUM(refund_amount) as refund_amount,
			SUM(total_amount - refund_amount) as net_revenue,
			COUNT(*) as total_orders,
			SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END) as successful_orders,
			SUM(CASE WHEN status = 5 THEN 1 ELSE 0 END) as failed_orders
		`).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Scan(&revenueStats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue stats: %w", err)
	}

	return map[string]interface{}{
		"total_revenue":        revenueStats.TotalRevenue,
		"subscription_revenue": revenueStats.SubscriptionRevenue,
		"one_time_revenue":     revenueStats.OneTimeRevenue,
		"refund_amount":        revenueStats.RefundAmount,
		"net_revenue":          revenueStats.NetRevenue,
		"total_orders":         revenueStats.TotalOrders,
		"successful_orders":    revenueStats.SuccessfulOrders,
		"failed_orders":        revenueStats.FailedOrders,
		"success_rate":         float64(revenueStats.SuccessfulOrders) / float64(revenueStats.TotalOrders) * 100,
	}, nil
}
