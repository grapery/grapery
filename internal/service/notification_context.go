package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func truncateNotificationText(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" || maxRunes <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}

// populateTargetRichContext loads story / storyboard / fragment / character display fields into n
// (StoryID, StoryTitle, StoryCover, StoryboardID, StoryboardTitle, FragmentID) for inbox + push.
func (s *Service) populateTargetRichContext(ctx context.Context, n *domain.Notification, targetTypeKey, targetID string) {
	tt := strings.ToLower(strings.TrimSpace(targetTypeKey))
	tid := strings.TrimSpace(targetID)
	if tid == "" {
		return
	}
	switch tt {
	case "story":
		n.StoryID = tid
		if st, err := s.repo.StoryByID(ctx, tid); err == nil && st != nil {
			if strings.TrimSpace(n.StoryTitle) == "" {
				n.StoryTitle = strings.TrimSpace(st.Title)
			}
			if strings.TrimSpace(n.StoryCover) == "" {
				n.StoryCover = strings.TrimSpace(st.CoverImage)
			}
		}
	case "storyboard":
		n.StoryboardID = tid
		if sb, err := s.repo.StoryboardByID(ctx, tid); err == nil && sb != nil {
			n.StoryID = strings.TrimSpace(sb.StoryID)
			if strings.TrimSpace(n.StoryboardTitle) == "" {
				n.StoryboardTitle = strings.TrimSpace(sb.Title)
			}
			if n.StoryID != "" {
				if st, err2 := s.repo.StoryByID(ctx, n.StoryID); err2 == nil && st != nil {
					if strings.TrimSpace(n.StoryTitle) == "" {
						n.StoryTitle = strings.TrimSpace(st.Title)
					}
					if strings.TrimSpace(n.StoryCover) == "" {
						n.StoryCover = strings.TrimSpace(st.CoverImage)
					}
				}
			}
		}
	case "fragment":
		n.FragmentID = tid
		if f, err := s.repo.FragmentByID(ctx, tid); err == nil && f != nil {
			if strings.TrimSpace(n.StoryTitle) == "" {
				label := strings.TrimSpace(f.Caption)
				if label == "" {
					label = truncateNotificationText(strings.TrimSpace(f.Content), 36)
				}
				n.StoryTitle = label
			}
			if strings.TrimSpace(n.StoryCover) == "" {
				n.StoryCover = firstFragmentCoverURL(f)
			}
		}
	case "character":
		if ch, err := s.repo.CharacterByID(ctx, tid); err == nil && ch != nil {
			if strings.TrimSpace(n.StoryTitle) == "" {
				n.StoryTitle = strings.TrimSpace(ch.Name)
			}
			n.StoryID = strings.TrimSpace(ch.StoryID)
			if strings.TrimSpace(n.StoryCover) == "" && n.StoryID != "" {
				if st, err2 := s.repo.StoryByID(ctx, n.StoryID); err2 == nil && st != nil {
					n.StoryCover = strings.TrimSpace(st.CoverImage)
				}
			}
		}
	}
}

func firstFragmentCoverURL(f *domain.Fragment) string {
	if f == nil {
		return ""
	}
	if len(f.MediaURLs) > 0 {
		return strings.TrimSpace(f.MediaURLs[0])
	}
	return ""
}

// targetSummaryPhrase returns a short Chinese phrase describing the notification target for copy.
func (s *Service) targetSummaryPhrase(n *domain.Notification, targetTypeKey string) string {
	tt := strings.ToLower(strings.TrimSpace(targetTypeKey))
	switch tt {
	case "storyboard":
		parts := make([]string, 0, 2)
		if strings.TrimSpace(n.StoryTitle) != "" {
			parts = append(parts, fmt.Sprintf("故事《%s》", strings.TrimSpace(n.StoryTitle)))
		}
		if strings.TrimSpace(n.StoryboardTitle) != "" {
			parts = append(parts, fmt.Sprintf("分镜《%s》", strings.TrimSpace(n.StoryboardTitle)))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	case "story":
		if strings.TrimSpace(n.StoryTitle) != "" {
			return fmt.Sprintf("故事《%s》", strings.TrimSpace(n.StoryTitle))
		}
	case "fragment":
		if strings.TrimSpace(n.StoryTitle) != "" {
			return fmt.Sprintf("故事碎片「%s」", strings.TrimSpace(n.StoryTitle))
		}
		return "故事碎片"
	case "character":
		if strings.TrimSpace(n.StoryTitle) != "" {
			return fmt.Sprintf("角色「%s」", strings.TrimSpace(n.StoryTitle))
		}
		return "角色"
	}
	return s.getTargetTypeName(targetTypeKey)
}

func likeNotificationTitle(targetTypeKey string) string {
	switch strings.ToLower(strings.TrimSpace(targetTypeKey)) {
	case "storyboard":
		return "分镜获赞"
	case "fragment":
		return "碎片获赞"
	case "story":
		return "故事获赞"
	case "character":
		return "角色相关获赞"
	default:
		return "新点赞"
	}
}

func storyboardCreatedTitle() string { return "新的故事板" }
