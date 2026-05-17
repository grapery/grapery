package common

import "strings"

// SubscriptionBillingPeriodGrantTokens 将 iap_products.quota_limit（单个计费周期的订阅点数/token 配额）
// 换算为「每次 Apple/Google 完成一期订阅付款」应写入 memberships 的订阅部分 token 上限。
//
// 约定（与 scripts/membership_iap_seed / iap_grapery_seed 一致）：quota_limit 表示「按月展示的档位配额」对应的 token；
// 年付 SKU（P1Y）在一次续费时一次性发放 12 个月的定额（unused 不结转）；季付（P3M）×3。
func SubscriptionBillingPeriodGrantTokens(quotaLimit int, duration *string) int {
	if quotaLimit <= 0 {
		return 0
	}
	if duration == nil {
		return quotaLimit
	}
	d := strings.TrimSpace(*duration)
	switch {
	case strings.HasPrefix(d, "P1Y"):
		return quotaLimit * 12
	case strings.HasPrefix(d, "P3M"):
		return quotaLimit * 3
	case strings.HasPrefix(d, "P1M"):
		return quotaLimit
	default:
		return quotaLimit
	}
}
