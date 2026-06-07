package cache

import "context"

const (
	PrefixMembership      = "membership:"
	PrefixMembershipUsage = "membership:usage:"
)

func MembershipKey(userID string) string {
	return PrefixMembership + userID
}

func MembershipUsageKey(userID string) string {
	return PrefixMembershipUsage + userID
}

// InvalidateStory drops the GetStory read-through cache entry (best-effort).
func InvalidateStory(ctx context.Context, c Cache, storyID string) {
	if c == nil || storyID == "" {
		return
	}
	_ = c.Delete(ctx, StoryKey(storyID))
}

// InvalidateMembership drops membership profile and usage cache entries (best-effort).
func InvalidateMembership(ctx context.Context, c Cache, userID string) {
	if c == nil || userID == "" {
		return
	}
	_ = c.Delete(ctx, MembershipKey(userID), MembershipUsageKey(userID))
}
