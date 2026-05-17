package domain

import "strings"

// CanonicalMembershipTier maps DB/API tier strings and legacy plan name prefixes to free | basic | premium.
func CanonicalMembershipTier(tierRaw, planNameFallback string) MembershipTierType {
	raw := strings.ToLower(strings.TrimSpace(tierRaw))
	if raw == "" {
		raw = strings.ToLower(strings.TrimSpace(planNameFallback))
	}
	switch raw {
	case string(TierTypeFree):
		return TierTypeFree
	case string(TierTypeBasic), "pro":
		return TierTypeBasic
	case string(TierTypePremium), "prime", "ultra":
		return TierTypePremium
	default:
		n := strings.ToLower(strings.TrimSpace(planNameFallback))
		switch {
		case strings.HasPrefix(n, "pro_"), strings.HasPrefix(n, "basic_"):
			return TierTypeBasic
		case strings.HasPrefix(n, "prime_"), strings.HasPrefix(n, "premium_"), strings.HasPrefix(n, "ultra_"):
			return TierTypePremium
		default:
			return TierTypeFree
		}
	}
}
