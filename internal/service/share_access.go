package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// SharePreview is a minimal payload for OG / WeChat in-app browser landing.
type SharePreview struct {
	Kind        ShareKind `json:"kind"`
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	ImageURL    string    `json:"imageUrl,omitempty"`
	OpenPath    string    `json:"openPath"`
}

// GetFragmentByID loads a fragment by id (share / visibility checks).
func (s *Service) GetFragmentByID(ctx context.Context, id string) (*domain.Fragment, error) {
	return s.repo.FragmentByID(ctx, id)
}

// CanViewerSeeFragment determines whether viewer may read fragment content.
// shareGrant is true when a valid signed share link is presented.
func (s *Service) CanViewerSeeFragment(ctx context.Context, viewerUserID string, f *domain.Fragment, shareGrant bool) bool {
	if f == nil {
		return false
	}
	if shareGrant {
		return f.Status != "deleted"
	}
	ownerID := f.UserID
	if ownerID == "" {
		ownerID = f.CreatorID
	}
	if viewerUserID != "" && viewerUserID == ownerID {
		return true
	}
	if f.IsDraft || f.Status == "deleted" {
		return false
	}
	switch domain.NormalizeFragmentVisibility(f.Visibility) {
	case domain.FragmentVisibilityPublic:
		return true
	case domain.FragmentVisibilityPrivate:
		return false
	case domain.FragmentVisibilityFollowers, domain.FragmentVisibilityFollowersLegacy:
		if viewerUserID == "" || ownerID == "" {
			return false
		}
		following, err := s.repo.IsFollowing(ctx, viewerUserID, ownerID)
		return err == nil && following
	default:
		return false
	}
}

// BuildSharePreview resolves preview metadata for a share landing page.
func (s *Service) BuildSharePreview(ctx context.Context, kind ShareKind, id string) (*SharePreview, error) {
	switch kind {
	case ShareKindFragment:
		return s.sharePreviewFragment(ctx, id)
	case ShareKindStory:
		return s.sharePreviewStory(ctx, id)
	case ShareKindStoryboard:
		return s.sharePreviewStoryboard(ctx, id)
	case ShareKindCharacter:
		return s.sharePreviewCharacter(ctx, id)
	default:
		return nil, domain.ErrNotFound
	}
}

func (s *Service) sharePreviewFragment(ctx context.Context, id string) (*SharePreview, error) {
	f, err := s.repo.FragmentByID(ctx, id)
	if err != nil || f == nil {
		return nil, domain.ErrNotFound
	}
	title := strings.TrimSpace(f.Caption)
	if title == "" {
		title = strings.TrimSpace(f.Content)
	}
	if title == "" {
		title = "Fragment"
	}
	if len([]rune(title)) > 80 {
		title = string([]rune(title)[:80]) + "…"
	}
	img := firstImageURL(f.MediaURLs, f.ImageUrls)
	return &SharePreview{
		Kind:        ShareKindFragment,
		ID:          id,
		Title:       title,
		Description: strings.TrimSpace(f.Content),
		ImageURL:    img,
		OpenPath:    "/fragments/" + id,
	}, nil
}

func (s *Service) sharePreviewStory(ctx context.Context, id string) (*SharePreview, error) {
	st, err := s.GetStory(ctx, id)
	if err != nil {
		return nil, err
	}
	img := shareFirstString(st.CoverImage, st.BackgroundImage)
	return &SharePreview{
		Kind:        ShareKindStory,
		ID:          id,
		Title:       strings.TrimSpace(st.Title),
		Description: strings.TrimSpace(st.Description),
		ImageURL:    img,
		OpenPath:    "/stories/" + id,
	}, nil
}

func (s *Service) sharePreviewStoryboard(ctx context.Context, id string) (*SharePreview, error) {
	sb, err := s.GetStoryboard(ctx, id)
	if err != nil {
		return nil, err
	}
	img := ""
	if len(sb.StoryboardScenes) > 0 {
		for _, sc := range sb.StoryboardScenes {
			if sc.Image != "" {
				img = sc.Image
				break
			}
		}
	}
	return &SharePreview{
		Kind:        ShareKindStoryboard,
		ID:          id,
		Title:       strings.TrimSpace(sb.Title),
		Description: strings.TrimSpace(sb.Content),
		ImageURL:    img,
		OpenPath:    "/storyboards/" + id,
	}, nil
}

func (s *Service) sharePreviewCharacter(ctx context.Context, id string) (*SharePreview, error) {
	ch, err := s.GetCharacter(ctx, id)
	if err != nil {
		return nil, err
	}
	return &SharePreview{
		Kind:        ShareKindCharacter,
		ID:          id,
		Title:       strings.TrimSpace(ch.Name),
		Description: strings.TrimSpace(ch.Description),
		ImageURL:    strings.TrimSpace(ch.Avatar),
		OpenPath:    "/characters/" + id,
	}, nil
}

func shareFirstString(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func firstImageURL(media []string, legacyJSON string) string {
	for _, u := range media {
		u = strings.TrimSpace(u)
		if u != "" {
			return u
		}
	}
	raw := strings.TrimSpace(legacyJSON)
	if raw == "" {
		return ""
	}
	var urls []string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil {
		return ""
	}
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			return u
		}
	}
	return ""
}
