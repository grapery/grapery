package http

import (
	"encoding/json"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"testing"
)

func TestFragmentPublicationSelectionResponseDoesNotRequireGenerationTrace(t *testing.T) {
	quality := &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}
	manifest := &domain.FragmentPublicationManifest{Quality: quality, Pages: []domain.FragmentPublicationPage{{PageIndex: 2, ImageURL: "accepted-c", Quality: quality}}}
	assets := []domain.FragmentPublicationAsset{{PageIndex: 2, ImageURL: "accepted-c"}}
	task := &domain.FragmentGenerationTask{Status: "completed", Request: domain.FragmentGenerationRequest{IncludeGenerationTrace: false, ImageCount: 3}, Result: &domain.FragmentGenerationResult{ImageUrls: []string{"accepted-c"}, Quality: quality, PublicationManifest: manifest, PublicationAssets: assets, ComicDocument: &domain.FragmentComicDocument{Pages: make([]domain.FragmentComicPageDocument, 3)}}}
	response := fragmentGenerationResultResponse(task)
	if _, ok := response["generationTrace"]; ok {
		t.Fatal("test must exercise trace-free response")
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		PublicationManifest *domain.FragmentPublicationManifest `json:"publicationManifest"`
		PublicationAssets   []domain.FragmentPublicationAsset   `json:"publicationAssets"`
		ComicDocument       *domain.FragmentComicDocument       `json:"comicDocument"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.PublicationManifest == nil || len(decoded.PublicationManifest.Pages) != 1 || decoded.PublicationManifest.Pages[0].PageIndex != 2 || len(decoded.PublicationAssets) != 1 || len(decoded.ComicDocument.Pages) != 3 {
		t.Fatal("selection aliases lost or canonical source truncated")
	}
}
