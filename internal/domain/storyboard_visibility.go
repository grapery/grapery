package domain

import "strings"

// RedactStoryboardViewsUnlessCreator sets Views to 0 when the API consumer is not the storyboard creator.
// Storyboard.UserID is serialized as creatorId. viewerID is the authenticated caller; empty viewer always redacts.
func RedactStoryboardViewsUnlessCreator(sb *Storyboard, viewerID string) {
	if sb == nil {
		return
	}
	v := strings.TrimSpace(viewerID)
	c := strings.TrimSpace(sb.UserID)
	if v == "" || c == "" || v != c {
		sb.Views = 0
	}
}

// RedactStoryboardViewsUnlessCreatorMany applies RedactStoryboardViewsUnlessCreator to each item.
func RedactStoryboardViewsUnlessCreatorMany(list []*Storyboard, viewerID string) {
	for _, sb := range list {
		RedactStoryboardViewsUnlessCreator(sb, viewerID)
	}
}
