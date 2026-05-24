package pay

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	payservice "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type membershipRow struct {
	ID           string     `gorm:"column:id"`
	UserID       string     `gorm:"column:user_id"`
	Tier         string     `gorm:"column:tier"`
	Status       string     `gorm:"column:status"`
	StartDate    time.Time  `gorm:"column:start_date"`
	EndDate      *time.Time `gorm:"column:end_date"`
	TokenQuota   int        `gorm:"column:token_quota"`
	TokenUsed    int        `gorm:"column:token_used"`
	StorageQuota int64      `gorm:"column:storage_quota"`
	StorageUsed  int64      `gorm:"column:storage_used"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

// SubscriptionDisplayInfo 客户端展示用订阅摘要。
type SubscriptionDisplayInfo struct {
	Tier            string                            `json:"tier"`
	TierDisplayKey  string                            `json:"tier_display_key"`
	ProductID       string                            `json:"product_id,omitempty"`
	ExpiresAt       *time.Time                        `json:"expires_at,omitempty"`
	AutoRenewing    bool                              `json:"auto_renewing"`
	LifecycleStatus string                            `json:"lifecycle_status"`
	TokenQuota      int                               `json:"token_quota,omitempty"`
	TokenUsed       int                               `json:"token_used,omitempty"`
	PendingNotice   *paymodels.SubscriptionNoticePayload `json:"pending_notice,omitempty"`
}

func tierDisplayKey(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "premium":
		return "membership_tier_prime"
	case "basic":
		return "membership_tier_pro"
	default:
		return "membership_tier_free"
	}
}

func (h *IAPHandler) lookupAppleSubProductID(ctx context.Context, originalTx string) string {
	originalTx = strings.TrimSpace(originalTx)
	if originalTx == "" {
		return ""
	}
	var rows []paymodels.AppleSubscription
	if err := paymodels.DataBase().WithContext(ctx).
		Where("original_transaction_id = ?", originalTx).
		Limit(1).
		Find(&rows).Error; err != nil || len(rows) == 0 {
		return ""
	}
	return rows[0].ProductID
}

func (h *IAPHandler) lookupAppleSubAppUserID(ctx context.Context, originalTx string) string {
	originalTx = strings.TrimSpace(originalTx)
	if originalTx == "" {
		return ""
	}
	var rows []paymodels.AppleSubscription
	if err := paymodels.DataBase().WithContext(ctx).
		Where("original_transaction_id = ?", originalTx).
		Limit(1).
		Find(&rows).Error; err != nil || len(rows) == 0 {
		return ""
	}
	return strings.TrimSpace(rows[0].AppUserID)
}

// applyApplePurchaseGrants 购买/通知成功后写入主库权益（统一入口）。
func (h *IAPHandler) applyApplePurchaseGrants(ctx context.Context, userIDStr string, receipt *payservice.IAPReceipt) {
	if userIDStr == "" || receipt == nil || receipt.ProductID == "" {
		return
	}
	product, prodErr := h.productService.GetProductByProductID(ctx, receipt.ProductID)
	if prodErr != nil || product == nil {
		if prodErr != nil {
			h.logger.WithError(prodErr).Warn("product lookup failed for entitlements")
		}
		return
	}

	normalizedQuota := common.NormalizeIAPProductQuotaLimit(receipt.ProductID, product.QuotaLimit)
	grantTokens := common.SubscriptionBillingPeriodGrantTokens(normalizedQuota, product.Duration)

	if grantTokens > 0 && !product.IsSubscription() {
		if topUpErr := h.topUpUserTokens(ctx, userIDStr, grantTokens); topUpErr != nil {
			h.logger.WithError(topUpErr).Error("top up tokens after IAP failed")
		}
		return
	}

	oldProductID := h.lookupAppleSubProductID(ctx, receipt.OriginalTransactionID)
	kind := common.DetectSubscriptionChangeKind(oldProductID, receipt.ProductID)
	h.applySubscriptionEntitlements(ctx, userIDStr, receipt, kind, oldProductID)
}

// applySubscriptionEntitlements 按变更类型更新 memberships 并写入 pending notice。
func (h *IAPHandler) applySubscriptionEntitlements(
	ctx context.Context,
	userIDStr string,
	receipt *payservice.IAPReceipt,
	kind common.SubscriptionChangeKind,
	oldProductID string,
) {
	if userIDStr == "" || receipt == nil {
		return
	}
	if kind == "" {
		kind = common.DetectSubscriptionChangeKind(oldProductID, receipt.ProductID)
	}

	product, prodErr := h.productService.GetProductByProductID(ctx, receipt.ProductID)
	if (prodErr != nil || product == nil) && receipt.ProductID != "" {
		if prodErr != nil {
			h.logger.WithError(prodErr).Warn("product lookup failed for entitlements")
		}
		if kind != common.ChangeCancelRenewal && kind != common.ChangeExpired && kind != common.ChangeRevoked {
			return
		}
	}

	var grantTokens int
	membershipTier := common.MembershipTierFromIAPProductID(receipt.ProductID)
	if product != nil && product.IsSubscription() {
		normalized := common.NormalizeIAPProductQuotaLimit(receipt.ProductID, product.QuotaLimit)
		grantTokens = common.SubscriptionBillingPeriodGrantTokens(normalized, product.Duration)
	}

	txID := strings.TrimSpace(receipt.SubscriptionTransactionID)
	needsClaim := kind == common.ChangeInitial || kind == common.ChangeUpgrade ||
		kind == common.ChangeRenewal || kind == common.ChangeRevoked
	if needsClaim && txID != "" {
		claimed, claimErr := paymodels.TryClaimSubscriptionCreditGrant(ctx, txID, userIDStr, receipt.ProductID)
		if claimErr != nil {
			h.logger.WithError(claimErr).Error("claim subscription credit grant failed")
			return
		}
		if !claimed {
			return
		}
	}

	var applyErr error
	switch kind {
	case common.ChangeUpgrade:
		applyErr = h.applyMembershipQuota(ctx, userIDStr, grantTokens, membershipTier, quotaPreserveUsed)
	case common.ChangeRenewal, common.ChangeInitial:
		applyErr = h.applyMembershipQuota(ctx, userIDStr, grantTokens, membershipTier, quotaResetUsed)
	case common.ChangeExpired, common.ChangeRevoked:
		applyErr = h.applyMembershipQuota(ctx, userIDStr, 0, "free", quotaExpireToFree)
	case common.ChangeCancelRenewal, common.ChangeDowngradeScheduled:
		applyErr = nil
	default:
		if grantTokens > 0 {
			applyErr = h.applyMembershipQuota(ctx, userIDStr, grantTokens, membershipTier, quotaResetUsed)
		}
	}
	if applyErr != nil {
		h.logger.WithError(applyErr).Error("apply membership quota failed")
		return
	}

	noticeKind := paymodels.MapChangeKindToNoticeKind(string(kind))
	if noticeKind == "" {
		return
	}
	args := h.buildNoticeArgs(ctx, userIDStr, kind, oldProductID, receipt, membershipTier, grantTokens)
	if _, err := paymodels.CreateSubscriptionNotice(ctx, userIDStr, noticeKind, args); err != nil {
		h.logger.WithError(err).Warn("create subscription notice failed")
	}
}

type quotaApplyMode int

const (
	quotaResetUsed quotaApplyMode = iota
	quotaPreserveUsed
	quotaExpireToFree
)

func (h *IAPHandler) applyMembershipQuota(
	ctx context.Context,
	userIDStr string,
	subscriptionGrantTokens int,
	membershipTier string,
	mode quotaApplyMode,
) error {
	if h.mainDB == nil {
		h.logger.Warn("mainDB not configured, skipping membership quota update")
		return nil
	}
	if userIDStr == "" {
		return nil
	}

	tier := strings.TrimSpace(membershipTier)
	if tier == "" {
		tier = "free"
	}
	now := time.Now()

	return h.mainDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m membershipRow
		err := tx.Table("memberships").Where("user_id = ?", userIDStr).First(&m).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			totalQuota := common.DefaultFreeTierTokenQuota
			tokenUsed := 0
			if mode != quotaExpireToFree {
				totalQuota = common.DefaultFreeTierTokenQuota + subscriptionGrantTokens
			}
			m = membershipRow{
				ID:           uuid.New().String(),
				UserID:       userIDStr,
				Tier:         tier,
				Status:       string(common.MembershipStatusActive),
				StartDate:    now,
				TokenQuota:   totalQuota,
				TokenUsed:    tokenUsed,
				StorageQuota: common.DefaultFreeTierStorageBytes,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			return tx.Table("memberships").Create(&m).Error
		}
		if err != nil {
			return fmt.Errorf("query membership: %w", err)
		}

		updates := map[string]interface{}{"updated_at": now}

		switch mode {
		case quotaExpireToFree:
			updates["tier"] = "free"
			updates["status"] = string(common.MembershipStatusExpired)
			updates["token_quota"] = common.DefaultFreeTierTokenQuota
			if m.TokenUsed > common.DefaultFreeTierTokenQuota {
				updates["token_used"] = common.DefaultFreeTierTokenQuota
			}
		case quotaPreserveUsed:
			totalQuota := common.DefaultFreeTierTokenQuota + subscriptionGrantTokens
			updates["tier"] = tier
			updates["status"] = string(common.MembershipStatusActive)
			updates["token_quota"] = totalQuota
			tokenUsed := m.TokenUsed
			if tokenUsed > totalQuota {
				tokenUsed = totalQuota
			}
			updates["token_used"] = tokenUsed
		case quotaResetUsed:
			totalQuota := common.DefaultFreeTierTokenQuota + subscriptionGrantTokens
			updates["tier"] = tier
			updates["status"] = string(common.MembershipStatusActive)
			updates["token_quota"] = totalQuota
			updates["token_used"] = 0
		}

		return tx.Table("memberships").Where("user_id = ?", userIDStr).Updates(updates).Error
	})
}

func (h *IAPHandler) buildNoticeArgs(
	ctx context.Context,
	userIDStr string,
	kind common.SubscriptionChangeKind,
	oldProductID string,
	receipt *payservice.IAPReceipt,
	newTier string,
	grantTokens int,
) map[string]string {
	args := map[string]string{
		"new_tier": newTier,
		"old_tier": common.MembershipTierFromIAPProductID(oldProductID),
	}
	if receipt.ExpirationDate != nil {
		args["expires_date"] = receipt.ExpirationDate.Format(time.RFC3339)
	}
	if _, used := h.readMembershipTokens(ctx, userIDStr); kind == common.ChangeUpgrade {
		args["token_used"] = strconv.Itoa(used)
		credits := grantTokens / common.CreditToTokenRatio
		if credits <= 0 && grantTokens > 0 {
			credits = grantTokens
		}
		args["grant_credits"] = strconv.Itoa(credits)
	}
	return args
}

func (h *IAPHandler) readMembershipTokens(ctx context.Context, userIDStr string) (quota, used int) {
	if h.mainDB == nil || userIDStr == "" {
		return 0, 0
	}
	var m membershipRow
	if err := h.mainDB.WithContext(ctx).Table("memberships").
		Select("token_quota, token_used").
		Where("user_id = ?", userIDStr).
		First(&m).Error; err != nil {
		return 0, 0
	}
	return m.TokenQuota, m.TokenUsed
}

func (h *IAPHandler) buildSubscriptionDisplay(ctx context.Context, userIDStr string, sub *payservice.IAPSubscription) *SubscriptionDisplayInfo {
	display := &SubscriptionDisplayInfo{
		Tier:           "free",
		TierDisplayKey: tierDisplayKey("free"),
		LifecycleStatus: "free",
	}
	if quota, used := h.readMembershipTokens(ctx, userIDStr); quota > 0 || used > 0 {
		display.TokenQuota = quota
		display.TokenUsed = used
	}
	if mTier := h.readMembershipTier(ctx, userIDStr); mTier != "" {
		display.Tier = mTier
		display.TierDisplayKey = tierDisplayKey(mTier)
	}

	if notice, _ := paymodels.GetPendingSubscriptionNotice(ctx, userIDStr); notice != nil {
		display.PendingNotice = notice
	}

	if sub == nil {
		return display
	}

	display.ProductID = sub.ProductID
	display.ExpiresAt = sub.ExpiresDate
	ar := strings.ToLower(strings.TrimSpace(sub.AutoRenewStatus))
	display.AutoRenewing = ar == "on" || ar == "true" || ar == "1"

	st := strings.TrimSpace(sub.Status)
	if sub.IsInGracePeriod {
		display.LifecycleStatus = "grace"
	} else if st == "Expired" || (sub.ExpiresDate != nil && sub.ExpiresDate.Before(time.Now())) {
		display.LifecycleStatus = "expired"
	} else if st == "WillExpire" || !display.AutoRenewing {
		display.LifecycleStatus = "will_expire"
	} else if st == "Active" || display.AutoRenewing {
		display.LifecycleStatus = "active"
	}
	if tier := common.MembershipTierFromIAPProductID(sub.ProductID); tier != "free" {
		display.Tier = tier
		display.TierDisplayKey = tierDisplayKey(tier)
	}
	return display
}

func (h *IAPHandler) readMembershipTier(ctx context.Context, userIDStr string) string {
	if h.mainDB == nil || userIDStr == "" {
		return ""
	}
	var m membershipRow
	if err := h.mainDB.WithContext(ctx).Table("memberships").
		Select("tier").
		Where("user_id = ?", userIDStr).
		First(&m).Error; err != nil {
		return ""
	}
	return m.Tier
}

// applyEntitlementsFromAppleNotification Webhook 成功后同步主库权益。
func (h *IAPHandler) applyEntitlementsFromAppleNotification(ctx context.Context, signedPayload string, notificationType, subtype string) {
	fields, err := payservice.ExtractAppleTransactionFromNotificationPayload(signedPayload)
	if err != nil {
		h.logger.WithError(err).Debug("skip entitlements: no signedTransactionInfo in notification")
		// cancel_renewal may only have renewal info — still try status-only path below
	}

	kind := common.NormalizeAppleNotificationAction(notificationType, subtype)
	if kind == "" {
		return
	}

	var userIDStr string
	var oldProductID string
	receipt := &payservice.IAPReceipt{ProductID: "", Status: "Active"}

	if fields != nil {
		receipt.ProductID = fields.ProductID
		receipt.SubscriptionTransactionID = fields.TransactionID
		receipt.OriginalTransactionID = fields.OriginalTransactionID
		receipt.ExpirationDate = fields.ExpiresDate
		if !fields.PurchaseDate.IsZero() {
			receipt.CreationDate = fields.PurchaseDate
		}
		oldProductID = h.lookupAppleSubProductID(ctx, fields.OriginalTransactionID)
		userIDStr = h.lookupAppleSubAppUserID(ctx, fields.OriginalTransactionID)
	}

	if userIDStr == "" {
		h.logger.WithFields(logrus.Fields{
			"notification_type": notificationType,
			"subtype":           subtype,
		}).Warn("skip entitlements: app_user_id not found for Apple subscription")
		return
	}

	if kind == common.ChangeCancelRenewal {
		h.markAppleSubWillExpire(ctx, receipt.OriginalTransactionID)
	}

	if receipt.ProductID == "" && kind != common.ChangeCancelRenewal {
		return
	}

	h.applySubscriptionEntitlements(ctx, userIDStr, receipt, kind, oldProductID)
}

func (h *IAPHandler) markAppleSubWillExpire(ctx context.Context, originalTx string) {
	originalTx = strings.TrimSpace(originalTx)
	if originalTx == "" {
		return
	}
	_ = paymodels.DataBase().WithContext(ctx).Model(&paymodels.AppleSubscription{}).
		Where("original_transaction_id = ?", originalTx).
		Updates(map[string]interface{}{
			"status":            "WillExpire",
			"auto_renew_status": "Off",
			"updated_at":        time.Now(),
		}).Error
}

// ExpireStaleAppleSubscriptions 兜底：已过期但仍标记 Active/WillExpire 的订阅。
func (h *IAPHandler) ExpireStaleAppleSubscriptions(ctx context.Context) {
	var rows []paymodels.AppleSubscription
	now := time.Now()
	if err := paymodels.DataBase().WithContext(ctx).
		Where("status IN ? AND expires_date IS NOT NULL AND expires_date < ?", []string{"Active", "WillExpire"}, now).
		Limit(100).
		Find(&rows).Error; err != nil {
		h.logger.WithError(err).Warn("expire stale subscriptions query failed")
		return
	}
	for _, row := range rows {
		userIDStr := strings.TrimSpace(row.AppUserID)
		if userIDStr == "" {
			continue
		}
		receipt := &payservice.IAPReceipt{
			ProductID:             row.ProductID,
			OriginalTransactionID: row.OriginalTransactionID,
			ExpirationDate:        row.ExpiresDate,
		}
		h.applySubscriptionEntitlements(ctx, userIDStr, receipt, common.ChangeExpired, row.ProductID)
		_ = paymodels.DataBase().WithContext(ctx).Model(&paymodels.AppleSubscription{}).
			Where("id = ?", row.ID).
			Updates(map[string]interface{}{"status": "Expired", "updated_at": now}).Error
	}
}

// resetSubscriptionPeriodTokens 兼容旧调用；新逻辑请用 applyMembershipQuota。
func (h *IAPHandler) resetSubscriptionPeriodTokens(ctx context.Context, userIDStr string, subscriptionGrantTokens int, membershipTier string) error {
	return h.applyMembershipQuota(ctx, userIDStr, subscriptionGrantTokens, membershipTier, quotaResetUsed)
}