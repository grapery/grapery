package common

import "strings"

// SubscriptionChangeKind 订阅权益变更类型（Apple IAP / 主库 memberships）。
type SubscriptionChangeKind string

const (
	ChangeInitial            SubscriptionChangeKind = "initial"
	ChangeUpgrade            SubscriptionChangeKind = "upgrade"
	ChangeDowngradeScheduled SubscriptionChangeKind = "downgrade_scheduled"
	ChangeRenewal            SubscriptionChangeKind = "renewal"
	ChangeCancelRenewal      SubscriptionChangeKind = "cancel_renewal"
	ChangeExpired            SubscriptionChangeKind = "expired"
	ChangeRevoked            SubscriptionChangeKind = "revoked"
)

// MembershipTierRank 用于比较档位高低（premium > basic > free）。
func MembershipTierRank(tier string) int {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "premium", "prime", "ultra":
		return 2
	case "basic", "pro":
		return 1
	default:
		return 0
	}
}

// DetectSubscriptionChangeKind 由旧/新 SKU 推断变更类型（verify 路径；Webhook 可覆盖）。
func DetectSubscriptionChangeKind(oldProductID, newProductID string) SubscriptionChangeKind {
	oldPID := strings.TrimSpace(oldProductID)
	newPID := strings.TrimSpace(newProductID)
	if newPID == "" {
		return ChangeRenewal
	}
	if oldPID == "" {
		return ChangeInitial
	}
	if oldPID == newPID {
		return ChangeRenewal
	}
	oldTier := MembershipTierFromIAPProductID(oldPID)
	newTier := MembershipTierFromIAPProductID(newPID)
	oldRank := MembershipTierRank(oldTier)
	newRank := MembershipTierRank(newTier)
	if newRank > oldRank {
		return ChangeUpgrade
	}
	if newRank < oldRank {
		return ChangeDowngradeScheduled
	}
	// 同档不同计费周期（月付/年付）
	return ChangeRenewal
}

// NormalizeAppleNotificationAction 将 ASC V2（及 legacy）通知映射为内部 action。
func NormalizeAppleNotificationAction(notificationType, subtype string) SubscriptionChangeKind {
	nt := strings.ToUpper(strings.TrimSpace(notificationType))
	st := strings.ToUpper(strings.TrimSpace(subtype))
	switch nt {
	case "SUBSCRIBED":
		switch st {
		case "INITIAL_BUY":
			return ChangeInitial
		case "UPGRADE":
			return ChangeUpgrade
		case "DOWNGRADE":
			return ChangeDowngradeScheduled
		default:
			return ChangeRenewal
		}
	case "DID_RENEW":
		return ChangeRenewal
	case "DID_CHANGE_RENEWAL_STATUS", "DID_CHANGE_RENEWAL_PREF":
		switch st {
		case "AUTO_RENEW_DISABLED":
			return ChangeCancelRenewal
		case "AUTO_RENEW_ENABLED":
			return ChangeRenewal
		default:
			return ChangeCancelRenewal
		}
	case "EXPIRED":
		return ChangeExpired
	case "REFUND", "REVOKE":
		return ChangeRevoked
	case "INITIAL_BUY":
		return ChangeInitial
	case "DID_FAIL_TO_RENEW":
		return ChangeExpired
	default:
		return ""
	}
}
