package service

import (
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func fragmentGenerationQuality(issues []domain.FragmentConsistencyIssue, skipped, provider string, expected, audited int) *domain.FragmentGenerationQuality {
	for _, issue := range issues {
		if strings.EqualFold(strings.TrimSpace(issue.Severity), "high") {
			return &domain.FragmentGenerationQuality{Status: domain.FragmentQualityBlocked, Reason: "high_severity_consistency_issue"}
		}
	}
	if skipped != "" || provider == "" || expected < 1 || audited < expected {
		if skipped == "" {
			skipped = "audit_incomplete"
		}
		return &domain.FragmentGenerationQuality{Status: domain.FragmentQualityUnreviewed, Reason: skipped}
	}
	return &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}
}

func setFragmentDocumentQuality(doc *domain.FragmentComicDocument, quality *domain.FragmentGenerationQuality) {
	if doc == nil {
		return
	}
	doc.Quality = quality
	for i := range doc.Pages {
		doc.Pages[i].Quality = quality
	}
}

// Legacy pages (nil quality) are grandfathered. Every newly generated page
// carries its own result, so editing one page cannot clear another page's block.
func reconcileFragmentDocumentQuality(doc *domain.FragmentComicDocument) {
	if doc == nil {
		return
	}
	var quality *domain.FragmentGenerationQuality
	for _, page := range doc.Pages {
		if page.Quality == nil {
			continue
		}
		if quality == nil || page.Quality.Status == domain.FragmentQualityBlocked ||
			(page.Quality.Status != domain.FragmentQualityPassed && quality.Status == domain.FragmentQualityPassed) {
			quality = page.Quality
		}
	}
	if quality != nil {
		doc.Quality = quality
	}
}

// Do not drop malformed model issues and accidentally turn them into a clean
// audit. A baseline URL or an unknown page makes the result unreviewed.
func validateFragmentAuditTargets(issues []domain.FragmentConsistencyIssue, urls []string, count int) error {
	indices := map[string]int{}
	for i, url := range urls {
		if strings.TrimSpace(url) != "" {
			indices[strings.TrimSpace(url)] = i
		}
	}
	for _, issue := range issues {
		index := issue.SceneIndex
		if url := strings.TrimSpace(issue.ImageURL); url != "" {
			mapped, ok := indices[url]
			if !ok {
				return fmt.Errorf("unknown audited image")
			}
			index = mapped
		}
		if index < 0 || index >= count {
			return fmt.Errorf("invalid audited page")
		}
	}
	return nil
}
