package service

import (
	"context"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"strings"
	"testing"
)

func TestPreparePublicationAuditsFlattenedPagesWithoutDestroyingBackground(t *testing.T) {
	original := &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{{BackgroundImageURL: "background", FlattenedImageURL: "old-flat", Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}}}}
	doc, _, _, err := auditPreparedFragmentDocument(context.Background(), original, []string{"new-flat"}, func(page domain.FragmentComicPageDocument, url string) fragmentPageAuditResult {
		if page.BackgroundImageURL != "background" || url != "new-flat" {
			t.Fatal("wrong audit target")
		}
		return fragmentPageAuditResult{Provider: "test", Expected: 1, Audited: 1}
	})
	if err != nil || doc.Pages[0].FlattenedImageURL != "new-flat" || doc.Pages[0].BackgroundImageURL != "background" {
		t.Fatalf("accepted manifest: %#v %v", doc, err)
	}
	if original.Pages[0].FlattenedImageURL != "old-flat" {
		t.Fatal("audit mutated source document")
	}
	failed, _, _, err := auditPreparedFragmentDocument(context.Background(), original, []string{"new-flat"}, func(domain.FragmentComicPageDocument, string) fragmentPageAuditResult {
		return fragmentPageAuditResult{Expected: 1, Skipped: "provider unavailable"}
	})
	if err != nil || failed.Quality.Status != domain.FragmentQualityUnreviewed || failed.Pages[0].FlattenedImageURL != "old-flat" || failed.Pages[0].BackgroundImageURL != "background" {
		t.Fatal("unreviewed audit accepted changed media or destroyed source")
	}
	blocked, _, _, _ := auditPreparedFragmentDocument(context.Background(), original, []string{"new-flat"}, func(domain.FragmentComicPageDocument, string) fragmentPageAuditResult {
		return fragmentPageAuditResult{Provider: "test", Expected: 1, Audited: 1, Issues: []domain.FragmentConsistencyIssue{{Severity: "high"}}}
	})
	if blocked.Quality.Status != domain.FragmentQualityBlocked || blocked.Pages[0].FlattenedImageURL != "old-flat" {
		t.Fatal("high-severity image accepted")
	}
}
func TestPreparePublicationManifestBoundsAndOrigin(t *testing.T) {
	owns := func(value string) bool { return strings.HasPrefix(value, "https://app.oss.example/") }
	if err := validatePublicationImageURLs([]string{"https://app.oss.example/a"}, 1, owns); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		urls []string
		n    int
	}{{[]string{"https://localhost/a"}, 1}, {[]string{"https://app.oss.example/a"}, 2}, {[]string{"https://app.oss.example/a", "https://app.oss.example/a"}, 2}, {nil, 0}, {make([]string, 9), 9}} {
		if validatePublicationImageURLs(tc.urls, tc.n, owns) == nil {
			t.Fatal("invalid manifest accepted")
		}
	}
}

func TestPreparePublicationDeletionUndoAndPersistedReorderPreserveCanonicalAuthority(t *testing.T) {
	passed := &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}
	blocked := &domain.FragmentGenerationQuality{Status: domain.FragmentQualityBlocked}
	doc := &domain.FragmentComicDocument{Quality: blocked, Pages: []domain.FragmentComicPageDocument{{PageIndex: 0, BackgroundImageURL: "a", FlattenedImageURL: "old-flat-a", Quality: passed}, {PageIndex: 1, BackgroundImageURL: "b", Quality: blocked}, {PageIndex: 2, BackgroundImageURL: "c", Quality: passed}}}
	selected, err := projectDraftPublicationPages(doc, nil, []string{"a", "c"})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	manifest, _, _, err := auditPublicationSelection(context.Background(), doc, selected, []string{"flat-a", "flat-c"}, func(page domain.FragmentComicPageDocument, url string) fragmentPageAuditResult {
		calls++
		if page.PageIndex == 1 {
			t.Fatal("deleted page was audited")
		}
		return fragmentPageAuditResult{Provider: "test", Expected: 1, Audited: 1}
	})
	if err != nil || calls != 2 || manifest.Quality.Status != domain.FragmentQualityPassed || len(manifest.Pages) != 2 {
		t.Fatalf("selected publication failed %#v %v", manifest, err)
	}
	if len(doc.Pages) != 3 || doc.Pages[1].Quality.Status != domain.FragmentQualityBlocked || doc.Pages[0].FlattenedImageURL != "old-flat-a" || doc.Quality.Status != domain.FragmentQualityBlocked {
		t.Fatal("canonical/undo authority overwritten")
	}
	assets := retainPublicationAssetIdentities(nil, manifest)
	restored, err := projectDraftPublicationPages(doc, assets, []string{"flat-a", "b", "flat-c"})
	if err != nil || len(restored) != 3 || restored[1].PageIndex != 1 {
		t.Fatalf("undo lost identity %v", err)
	}
	if publicationSelectionMatches(manifest, restored, []string{"flat-a", "b", "flat-c"}) {
		t.Fatal("undo reused different selection")
	}
	reordered, err := projectDraftPublicationPages(doc, assets, []string{"flat-c", "flat-a"})
	if err != nil || reordered[0].PageIndex != 2 || reordered[1].PageIndex != 0 {
		t.Fatal("persisted reorder lost")
	}
	if err := validatePublicationCandidateIdentities(doc, assets, reordered, []string{"flat-c", "flat-a"}); err != nil {
		t.Fatal(err)
	}
	if validatePublicationCandidateIdentities(doc, assets, selected, []string{"c", "a"}) == nil {
		t.Fatal("candidate-only reorder accepted")
	}
	if _, err := projectDraftPublicationPages(doc, assets, []string{"arbitrary"}); err == nil {
		t.Fatal("replacement became source")
	}
	if _, err := projectDraftPublicationPages(doc, assets, []string{"a", "old-flat-a"}); err == nil {
		t.Fatal("same page selected twice")
	}
	history := retainPublicationAssetIdentities(assets, &domain.FragmentPublicationManifest{Quality: passed, Pages: []domain.FragmentPublicationPage{{PageIndex: 0, ImageURL: "newer-flat-a", Quality: passed}}})
	for _, url := range []string{"old-flat-a", "flat-a", "newer-flat-a"} {
		if pages, err := projectDraftPublicationPages(doc, history, []string{url}); err != nil || pages[0].PageIndex != 0 {
			t.Fatalf("undo alias %s lost", url)
		}
	}
}

func TestExistingDraftProjectionResolvesAcceptedAliasesAndPreservesLettering(t *testing.T) {
	trace := &domain.FragmentGenerationTrace{ComicDocument: &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{
		{PageIndex: 0, BackgroundImageURL: "a", TextLayers: []domain.FragmentComicText{{Text: "dialogue-a"}}},
		{PageIndex: 1, BackgroundImageURL: "b", TextLayers: []domain.FragmentComicText{{Text: "dialogue-b"}}},
		{PageIndex: 2, BackgroundImageURL: "c", TextLayers: []domain.FragmentComicText{{Text: "dialogue-c"}}},
	}}, PublicationAssets: []domain.FragmentPublicationAsset{{PageIndex: 0, ImageURL: "flat-a"}, {PageIndex: 2, ImageURL: "flat-c"}}}
	projected := fragmentComicDocumentFromExistingDraft(domain.FragmentGenerationRequest{}, "draft", "story", "3:4", []string{"flat-c", "flat-a"}, trace)
	if projected == nil || len(projected.Pages) != 2 || projected.Pages[0].BackgroundImageURL != "c" || projected.Pages[0].FlattenedImageURL != "flat-c" || projected.Pages[0].TextLayers[0].Text != "dialogue-c" || projected.Pages[1].TextLayers[0].Text != "dialogue-a" || projected.Pages[1].PageIndex != 1 {
		t.Fatal("accepted image aliases lost page identity/lettering")
	}
	if len(trace.ComicDocument.Pages) != 3 || trace.ComicDocument.Pages[2].PageIndex != 2 || trace.ComicDocument.Pages[2].FlattenedImageURL != "" {
		t.Fatal("projection mutated canonical source")
	}
	if got := fragmentComicDocumentFromExistingDraft(domain.FragmentGenerationRequest{}, "draft", "story", "3:4", []string{"replacement"}, trace); got != nil {
		t.Fatal("unknown image silently became continuation authority")
	}
}
