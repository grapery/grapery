package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// The current draft decides selection and order; the full source document and
// previous accepted aliases decide identity. Arbitrary draft replacement URLs
// cannot become a new authority just by being submitted to prepare-publication.
func projectDraftPublicationPages(doc *domain.FragmentComicDocument, assets []domain.FragmentPublicationAsset, urls []string) ([]domain.FragmentComicPageDocument, error) {
	if doc == nil || len(urls) < 1 {
		return nil, fmt.Errorf("select existing pages")
	}
	byIndex := map[int]domain.FragmentComicPageDocument{}
	identities := map[string]int{}
	bind := func(raw string, index int) error {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		if previous, ok := identities[raw]; ok && previous != index {
			return fmt.Errorf("ambiguous publication image identity")
		}
		identities[raw] = index
		return nil
	}
	for _, page := range doc.Pages {
		if _, exists := byIndex[page.PageIndex]; exists {
			return nil, fmt.Errorf("duplicate canonical page identity")
		}
		byIndex[page.PageIndex] = page
		if err := bind(page.BackgroundImageURL, page.PageIndex); err != nil {
			return nil, err
		}
		if err := bind(page.FlattenedImageURL, page.PageIndex); err != nil {
			return nil, err
		}
	}
	for _, asset := range assets {
		if _, ok := byIndex[asset.PageIndex]; !ok {
			return nil, fmt.Errorf("accepted image has no canonical page")
		}
		if err := bind(asset.ImageURL, asset.PageIndex); err != nil {
			return nil, err
		}
	}
	selected := make([]domain.FragmentComicPageDocument, 0, len(urls))
	seen := map[int]bool{}
	for _, raw := range urls {
		index, ok := identities[strings.TrimSpace(raw)]
		if !ok || seen[index] {
			return nil, fmt.Errorf("draft media is not a distinct existing page selection")
		}
		seen[index] = true
		selected = append(selected, byIndex[index])
	}
	return selected, nil
}

func publicationSelectionMatches(manifest *domain.FragmentPublicationManifest, pages []domain.FragmentComicPageDocument, urls []string) bool {
	if manifest == nil || manifest.Quality == nil || manifest.Quality.Status != domain.FragmentQualityPassed || len(manifest.Pages) != len(pages) || len(urls) != len(pages) {
		return false
	}
	for i, page := range manifest.Pages {
		if page.PageIndex != pages[i].PageIndex || page.ImageURL != urls[i] || page.Quality == nil || page.Quality.Status != domain.FragmentQualityPassed {
			return false
		}
	}
	return true
}

func auditPublicationSelection(ctx context.Context, source *domain.FragmentComicDocument, selected []domain.FragmentComicPageDocument, urls []string, audit func(domain.FragmentComicPageDocument, string) fragmentPageAuditResult) (*domain.FragmentPublicationManifest, []domain.FragmentConsistencyIssue, int, error) {
	projected := *source
	projected.Pages = selected
	reviewed, issues, tokens, err := auditPreparedFragmentDocument(ctx, &projected, urls, audit)
	if err != nil {
		return nil, nil, tokens, err
	}
	manifest := &domain.FragmentPublicationManifest{Quality: reviewed.Quality, Pages: make([]domain.FragmentPublicationPage, 0, len(reviewed.Pages))}
	for i, page := range reviewed.Pages {
		manifest.Pages = append(manifest.Pages, domain.FragmentPublicationPage{PageIndex: page.PageIndex, ImageURL: urls[i], Quality: page.Quality})
	}
	return manifest, issues, tokens, nil
}

func retainPublicationAssetIdentities(previous []domain.FragmentPublicationAsset, manifest *domain.FragmentPublicationManifest) []domain.FragmentPublicationAsset {
	out := append([]domain.FragmentPublicationAsset(nil), previous...)
	if manifest == nil || manifest.Quality == nil || manifest.Quality.Status != domain.FragmentQualityPassed {
		return out
	}
	seen := map[string]bool{}
	for _, asset := range out {
		seen[asset.ImageURL] = true
	}
	for _, page := range manifest.Pages {
		if !seen[page.ImageURL] {
			out = append(out, domain.FragmentPublicationAsset{PageIndex: page.PageIndex, ImageURL: page.ImageURL})
			seen[page.ImageURL] = true
		}
	}
	return out
}

func validatePublicationCandidateIdentities(doc *domain.FragmentComicDocument, assets []domain.FragmentPublicationAsset, selected []domain.FragmentComicPageDocument, urls []string) error {
	if len(selected) != len(urls) {
		return fmt.Errorf("publication page count changed")
	}
	for i, url := range urls {
		identity, err := projectDraftPublicationPages(doc, assets, []string{url})
		if err == nil && identity[0].PageIndex != selected[i].PageIndex {
			return fmt.Errorf("publication candidate changes an existing page identity or order")
		}
	}
	return nil
}
