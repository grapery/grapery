package service

import "github.com/grapestree/fgrapery/grapery/internal/domain"

// VoyagerNotificationType maps server-side notification types to values accepted by
// Voyager iOS NotificationType (Codable enum). DB and push-preference logic keep server types.
func VoyagerNotificationType(server string) string {
	switch server {
	case "follow", "like", "comment", "mention", "system", "update", "story_update", "announcement", "ai_complete":
		return server
	case "fragment_generation_complete", "storyboard_generation_complete":
		return "ai_complete"
	case "story_follow_storyboard", "storyboard", "fork":
		return "story_update"
	case "reply":
		return "comment"
	case "group_invite":
		return "system"
	default:
		return "system"
	}
}

// CloneNotificationsWithVoyagerTypes returns a shallow copy of each notification with Type remapped for API/SSE JSON.
func CloneNotificationsWithVoyagerTypes(in []*domain.Notification) []*domain.Notification {
	if len(in) == 0 {
		return nil
	}
	out := make([]*domain.Notification, len(in))
	for i, n := range in {
		if n == nil {
			continue
		}
		c := *n
		c.Type = VoyagerNotificationType(n.Type)
		out[i] = &c
	}
	return out
}
