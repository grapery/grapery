package service

import (
	"context"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func (s *Service) ListFragmentGenerationAssets(ctx context.Context, viewerID, fragmentID string, query FragmentAssetQuery) ([]*domain.FragmentGenerationAsset, error) {
	fragment, err := s.repo.FragmentByID(ctx, fragmentID)
	if err != nil {
		return nil, err
	}
	if fragment == nil {
		return nil, domain.ErrNotFound
	}
	if !s.canViewerReadFragmentAssets(ctx, fragment, viewerID) {
		return nil, domain.ErrForbidden
	}
	assets, err := s.repo.ListFragmentGenerationAssets(ctx, fragmentID)
	if err != nil {
		return nil, err
	}
	return filterFragmentAssets(assets, query), nil
}

func (s *Service) canViewerReadFragmentAssets(ctx context.Context, fragment *domain.Fragment, viewerID string) bool {
	if fragment == nil {
		return false
	}
	ownerID := strings.TrimSpace(fragment.UserID)
	if ownerID == "" {
		ownerID = strings.TrimSpace(fragment.CreatorID)
	}
	if strings.TrimSpace(viewerID) == ownerID {
		return true
	}
	visibility := strings.TrimSpace(fragment.Visibility)
	if visibility == "" {
		visibility = domain.FragmentVisibilityPublic
	}
	switch visibility {
	case domain.FragmentVisibilityPrivate:
		return false
	case domain.FragmentVisibilityFollowers, domain.FragmentVisibilityFollowersLegacy:
		if viewerID == "" || ownerID == "" {
			return false
		}
		ok, err := s.repo.IsFollowing(ctx, viewerID, ownerID)
		return err == nil && ok
	default:
		return true
	}
}

func filterFragmentAssets(items []*domain.FragmentGenerationAsset, q FragmentAssetQuery) []*domain.FragmentGenerationAsset {
	kind := strings.TrimSpace(q.Kind)
	entityKind := strings.TrimSpace(q.EntityKind)
	entityKey := strings.TrimSpace(q.EntityKey)
	if kind == "" && entityKind == "" && entityKey == "" {
		return items
	}
	out := make([]*domain.FragmentGenerationAsset, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if kind != "" && !strings.EqualFold(strings.TrimSpace(item.Kind), kind) {
			continue
		}
		if entityKind != "" && !strings.EqualFold(strings.TrimSpace(item.EntityKind), entityKind) {
			continue
		}
		if entityKey != "" && !strings.EqualFold(strings.TrimSpace(item.EntityKey), entityKey) {
			continue
		}
		out = append(out, item)
	}
	return out
}
