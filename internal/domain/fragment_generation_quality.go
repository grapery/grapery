package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	FragmentQualityUnreviewed = "unreviewed"
	FragmentQualityPassed     = "passed"
	FragmentQualityBlocked    = "blocked"
)

// Nil quality identifies legacy data. New generation always records an explicit
// outcome; completion of image rendering alone is not publication approval.
type FragmentGenerationQuality struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ValidateFragmentPublicationQuality reads server-owned metadata, never a
// client-submitted replacement. Legacy documents without quality remain usable.
func ValidateFragmentPublicationQuality(metadata string) error {
	if strings.TrimSpace(metadata) == "" {
		return nil
	}
	var envelope struct {
		PublicationManifest *FragmentPublicationManifest `json:"publicationManifest"`
		BillingStatus       string                       `json:"billingStatus"`
		Quality             *FragmentGenerationQuality   `json:"quality"`
		ComicDocument       *FragmentComicDocument       `json:"comicDocument"`
	}
	if err := json.Unmarshal([]byte(metadata), &envelope); err != nil {
		return fmt.Errorf("invalid fragment generation metadata: %w", err)
	}
	if envelope.BillingStatus != "" && envelope.BillingStatus != "settled" {
		return fmt.Errorf("fragment billing %s: settlement is required before publication", envelope.BillingStatus)
	}
	if manifest := envelope.PublicationManifest; manifest != nil {
		if manifest.Quality == nil || manifest.Quality.Status != FragmentQualityPassed || len(manifest.Pages) == 0 {
			return fmt.Errorf("publication selection has not passed audit")
		}
		selected := map[int]bool{}
		for _, page := range manifest.Pages {
			if page.Quality == nil || page.Quality.Status != FragmentQualityPassed || strings.TrimSpace(page.ImageURL) == "" || selected[page.PageIndex] {
				return fmt.Errorf("publication page has not passed audit")
			}
			selected[page.PageIndex] = true
		}
		return nil
	}
	qualities := []*FragmentGenerationQuality{envelope.Quality}
	if doc := envelope.ComicDocument; doc != nil {
		qualities = append(qualities, doc.Quality)
		for _, page := range doc.Pages {
			qualities = append(qualities, page.Quality)
		}
	}
	for _, quality := range qualities {
		if quality != nil && quality.Status != FragmentQualityPassed {
			return fmt.Errorf("fragment quality %s: %s", quality.Status, quality.Reason)
		}
	}
	return nil
}
