package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ValidateFragmentPublicationAuthority runs inside the caller's write transaction.
// Persisted task and quality facts are the authority; request metadata is not.
func ValidateFragmentPublicationAuthority(ctx context.Context, tx *gorm.DB, stored *FragmentDB, proposed *domain.Fragment) error {
	if err := domain.ValidateFragmentPublicationQuality(stored.GenerationMetadata); err != nil {
		return err
	}
	if err := domain.ValidateFragmentPublicationQuality(proposed.GenerationMetadata); err != nil {
		return err
	}
	newGeneration := false
	if stored.SourceType == string(domain.FragmentSourcePanelGeneration) && (stored.SourceID != "" || stored.GenerationTaskID != "") {
		id := stored.SourceID
		if id == "" {
			id = stored.GenerationTaskID
		}
		var task FragmentPanelGenerationTaskDB
		if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", id, stored.UserID).First(&task).Error; err != nil {
			return fmt.Errorf("panel generation authority unavailable: %w", err)
		}
		if task.Status != "completed" {
			return fmt.Errorf("panel generation is not completed")
		}
	} else if stored.GenerationTaskID != "" {
		var task FragmentGenerationTaskDB
		if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", stored.GenerationTaskID, stored.UserID).First(&task).Error; err != nil {
			return fmt.Errorf("generation authority unavailable: %w", err)
		}
		if task.Status != "completed" {
			return fmt.Errorf("generation is not completed")
		}
		var req domain.FragmentGenerationRequest
		var result domain.FragmentGenerationResult
		if json.Unmarshal([]byte(task.RequestJSON), &req) != nil {
			return fmt.Errorf("invalid generation request authority")
		}
		if req.BillingVersion >= 1 {
			newGeneration = true
			if json.Unmarshal([]byte(task.ResultJSON), &result) != nil || result.GenerationTrace == nil || result.GenerationTrace.BillingStatus != "settled" {
				return fmt.Errorf("generation bill is not settled")
			}
		}
	}
	var trace domain.FragmentGenerationTrace
	if strings.TrimSpace(stored.GenerationMetadata) == "" {
		if newGeneration {
			return fmt.Errorf("new generation quality authority is unavailable")
		}
		return nil
	}
	if err := json.Unmarshal([]byte(stored.GenerationMetadata), &trace); err != nil {
		return err
	}
	if trace.PublicationManifest == nil && trace.Quality == nil && (trace.ComicDocument == nil || trace.ComicDocument.Quality == nil) {
		if newGeneration {
			return fmt.Errorf("new generation has no explicit quality outcome")
		}
		return nil
	}
	if newGeneration && trace.PublicationManifest == nil {
		return fmt.Errorf("prepare the current page selection before publication")
	}
	if stored.IsDraft && stored.GenerationTaskID != "" && proposed.ExpectedGenerationTaskID != stored.GenerationTaskID {
		return fmt.Errorf("generation changed; reopen the current draft")
	}
	if proposed.Content != stored.Content {
		return fmt.Errorf("generated text changed; edit the current draft before publication")
	}
	if proposed.GenerationTaskID != stored.GenerationTaskID {
		return fmt.Errorf("generation task cannot be replaced on publication")
	}
	urls := proposed.MediaURLs
	if len(urls) == 0 && proposed.ImageUrls != "" {
		if err := json.Unmarshal([]byte(proposed.ImageUrls), &urls); err != nil {
			return err
		}
	}
	return ValidateFragmentMediaManifest(&trace, urls)
}

func ValidateFragmentMediaManifest(trace *domain.FragmentGenerationTrace, urls []string) error {
	if trace == nil || trace.ComicDocument == nil || len(trace.ComicDocument.Pages) == 0 {
		return fmt.Errorf("generated publication manifest is unavailable")
	}
	if manifest := trace.PublicationManifest; manifest != nil {
		if manifest.Quality == nil || manifest.Quality.Status != domain.FragmentQualityPassed || len(manifest.Pages) == 0 || len(urls) != len(manifest.Pages) {
			return fmt.Errorf("publication selection has not passed audit")
		}
		known := map[int]bool{}
		for _, page := range trace.ComicDocument.Pages {
			known[page.PageIndex] = true
		}
		seen := map[int]bool{}
		for i, page := range manifest.Pages {
			if !known[page.PageIndex] || seen[page.PageIndex] || page.Quality == nil || page.Quality.Status != domain.FragmentQualityPassed || strings.TrimSpace(page.ImageURL) == "" || urls[i] != page.ImageURL {
				return fmt.Errorf("publication page selection or accepted media changed")
			}
			seen[page.PageIndex] = true
		}
		return nil
	}
	pages := trace.ComicDocument.Pages
	if len(urls) != len(pages) {
		return fmt.Errorf("publication image count changed")
	}
	for i, page := range pages {
		if page.Quality == nil || page.Quality.Status != domain.FragmentQualityPassed {
			return fmt.Errorf("page %d has not passed publication audit", i+1)
		}
		value := strings.TrimSpace(urls[i])
		if value == "" || (value != strings.TrimSpace(page.BackgroundImageURL) && value != strings.TrimSpace(page.FlattenedImageURL)) {
			return fmt.Errorf("page %d is not a server-accepted publication image", i+1)
		}
	}
	return nil
}
