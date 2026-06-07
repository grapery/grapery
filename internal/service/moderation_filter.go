package service

// ExcludeByBlockedAuthors removes items whose author is in the blocked set.
func ExcludeByBlockedAuthors[T any](items []T, total int64, blocked map[string]struct{}, authorID func(T) string) ([]T, int64) {
	if len(blocked) == 0 || len(items) == 0 {
		return items, total
	}
	filtered := make([]T, 0, len(items))
	removed := 0
	for _, item := range items {
		author := authorID(item)
		if author != "" {
			if _, isBlocked := blocked[author]; isBlocked {
				removed++
				continue
			}
		}
		filtered = append(filtered, item)
	}
	if removed > 0 && total >= int64(removed) {
		total -= int64(removed)
	}
	return filtered, total
}

// BlockedIDSet converts a slice of user IDs into a lookup set for feed filtering.
func BlockedIDSet(ids []string) map[string]struct{} {
	if len(ids) == 0 {
		return nil
	}
	blocked := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			blocked[id] = struct{}{}
		}
	}
	return blocked
}
