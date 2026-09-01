package service

import (
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestBuildFragmentComicDocumentPreservesMultimodalFactsAndEditableText(t *testing.T) {
	request := domain.FragmentGenerationRequest{
		UserInput: "小猫头鹰来到水之塔",
		Language:  "zh-Hans",
		Style:     "manga",
		ImageUrls: []string{"https://example.com/owl.png"},
	}
	page := buildFallbackFragmentComicPagePlan(domain.FragmentScenePlan{
		SceneDesc:   "小猫头鹰看见水之塔",
		ImagePrompt: "owl sees the water tower",
	}, 3, request.Language)
	page.Panels[0].ComicTexts = []domain.FragmentComicText{{
		Type: "dialogue", Text: "就在前面", Speaker: "owl", RenderMode: "overlay",
	}}
	scenes := []domain.FragmentScenePlan{{SceneDesc: "小猫头鹰看见水之塔", ComicPage: &page}}
	evidence := []domain.FragmentVisualEvidence{{
		ImageURL: "https://example.com/owl.png", Summary: "一只戴黑框眼镜的小猫头鹰", Confidence: .9,
		Entities: []domain.FragmentVisualEvidenceEntity{{Kind: "character"}},
	}}

	doc := buildFragmentComicDocument(request, "fragment-1", "扩写后的故事", "3:4", nil, evidence, scenes, nil)
	if doc.SchemaVersion != 2 || doc.Revision != 1 || len(doc.Pages) != 1 {
		t.Fatalf("unexpected document header: %#v", doc)
	}
	if len(doc.CreativeContext.Inputs) != 1 || doc.CreativeContext.Inputs[0].Role != "character_reference" {
		t.Fatalf("expected image evidence to become a character reference: %#v", doc.CreativeContext.Inputs)
	}
	if len(doc.CreativeContext.Facts) != 3 {
		t.Fatalf("expected text, image, and agent-expansion facts, got %#v", doc.CreativeContext.Facts)
	}
	text := doc.Pages[0].Plan.Panels[0].ComicTexts[0]
	if text.ID == "" || text.RenderMode != "overlay" {
		t.Fatalf("expected stable editable overlay text, got %#v", text)
	}
	if got := len(doc.Pages[0].Plan.Layout.Panels); got != 3 {
		t.Fatalf("expected deterministic three-panel layout, got %d", got)
	}
	if doc.Pages[0].Status != "planned" {
		t.Fatal("a page without artwork must not be marked rendered_editable")
	}
	if page.Panels[0].ComicTexts[0].ID != "" {
		t.Fatal("building a document mutated the input scene")
	}
}

func TestFragmentLegacyMigrationDoesNotOverlayExistingLettering(t *testing.T) {
	plan := buildFallbackFragmentComicPagePlan(domain.FragmentScenePlan{SceneDesc: "arrival"}, 2, "zh-Hans")
	plan.Panels[0].ComicTexts = []domain.FragmentComicText{{Type: "dialogue", Text: "到了"}}
	trace := &domain.FragmentGenerationTrace{Scenes: []domain.FragmentScenePlan{{ComicPage: &plan}}}
	doc := fragmentComicDocumentFromExistingDraft(domain.FragmentGenerationRequest{}, "draft", "story", "3:4", []string{"old.jpg"}, trace)
	if doc.Pages[0].TextLayers[0].RenderMode != "image" || doc.Pages[0].FlattenedImageURL != "old.jpg" {
		t.Fatalf("legacy lettering must remain baked in the existing bitmap: %#v", doc.Pages[0])
	}
	if trace.Scenes[0].ComicPage.Panels[0].ComicTexts[0].RenderMode != "" {
		t.Fatal("migration changed stored trace by aliasing its slices")
	}
}

func TestFragmentDocumentMergeDoesNotMutateIncomingTextIDs(t *testing.T) {
	current := &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{{
		Plan: domain.FragmentComicPagePlan{Panels: []domain.FragmentComicPanelPlan{{ComicTexts: []domain.FragmentComicText{{ID: "original"}}}}},
	}}}
	previous := &domain.FragmentComicDocument{Pages: make([]domain.FragmentComicPageDocument, 2)}
	for _, replacement := range []int{0, 2} {
		_ = applyFragmentComicDocumentEdit(previous, current, 2, replacement)
		if current.Pages[0].Plan.Panels[0].ComicTexts[0].ID != "original" {
			t.Fatalf("edit mode %d mutated input document", replacement)
		}
	}
}

func TestFragmentComicDocumentAppendAndReplacePreserveExistingPages(t *testing.T) {
	page := func(intent string) domain.FragmentComicPageDocument {
		return domain.FragmentComicPageDocument{PageIntent: intent, Status: "rendered"}
	}
	previous := &domain.FragmentComicDocument{
		SchemaVersion: 2, Revision: 4,
		Pages: []domain.FragmentComicPageDocument{page("first"), page("second")},
	}
	previous.Pages[1].PageIndex = 1
	current := &domain.FragmentComicDocument{
		SchemaVersion: 2, Revision: 1,
		Pages: []domain.FragmentComicPageDocument{page("new")},
	}

	appended := applyFragmentComicDocumentEdit(previous, current, 2, 0)
	if appended.Revision != 5 || len(appended.Pages) != 3 || appended.Pages[2].PageIntent != "new" {
		t.Fatalf("append must preserve existing pages: %#v", appended.Pages)
	}
	replaced := applyFragmentComicDocumentEdit(previous, current, 0, 2)
	if len(replaced.Pages) != 2 || replaced.Pages[0].PageIntent != "first" || replaced.Pages[1].PageIntent != "new" {
		t.Fatalf("replace must only change the selected page: %#v", replaced.Pages)
	}
}

func TestFragmentDocumentReplacesSparseLegacyPageByPageIndex(t *testing.T) {
	previous := &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{
		{PageIndex: 2, PageIntent: "third"}, {PageIndex: 5, PageIntent: "sixth"},
	}}
	current := &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{{PageIntent: "replacement"}}}
	got := applyFragmentComicDocumentEdit(previous, current, 0, 6)
	if len(got.Pages) != 2 || got.Pages[1].PageIntent != "replacement" || got.Pages[0].PageIntent != "third" {
		t.Fatalf("replacement used array position rather than page identity: %#v", got.Pages)
	}
	appended := mergeFragmentComicDocuments(previous, current, 0)
	if appended.Pages[2].PageIndex != 6 {
		t.Fatal("append collided with a sparse legacy page index")
	}
}

func TestFragmentDocumentReconcilesDeletedAndReorderedImages(t *testing.T) {
	doc := &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{
		{PageIndex: 0, BackgroundImageURL: "a.jpg"},
		{PageIndex: 1, BackgroundImageURL: "b.jpg"},
		{PageIndex: 2, BackgroundImageURL: "c.jpg"},
	}}
	got := reconcileFragmentComicDocumentImages(doc, []string{"c.jpg", "b.jpg"})
	if len(got.Pages) != 2 || got.Pages[0].BackgroundImageURL != "c.jpg" || got.Pages[1].PageIndex != 1 {
		t.Fatalf("deleted page resurrected or reordered lettering moved: %#v", got.Pages)
	}
	if len(doc.Pages) != 3 || doc.Pages[2].PageIndex != 2 {
		t.Fatal("reconciliation mutated persisted source")
	}
}

func TestFragmentDocumentProgressUpdatesOnlyAppendedPage(t *testing.T) {
	doc := &domain.FragmentComicDocument{Pages: []domain.FragmentComicPageDocument{
		{PageIndex: 0, BackgroundImageURL: "old.jpg", Status: "rendered"},
		{PageIndex: 1, Status: "planned", TextLayers: []domain.FragmentComicText{{RenderMode: "overlay", Text: "到了"}}},
	}}
	scenes := []domain.FragmentScenePlan{{GeneratedImageURL: "new.jpg", PanelImageURLs: []string{"panel.jpg"}}}
	syncFragmentComicDocumentPageAssets(doc, scenes, 1)
	if doc.Pages[0].BackgroundImageURL != "old.jpg" || doc.Pages[1].BackgroundImageURL != "new.jpg" || doc.Pages[1].Status != "rendered_editable" || doc.Pages[1].FlattenedImageURL != "" {
		t.Fatalf("progress must preserve old page and publish new background without losing lettering: %#v", doc.Pages)
	}
}

func TestFragmentComicLayoutIsNormalizedAndNonOverlapping(t *testing.T) {
	for count := fragmentComicPageMinPanels; count <= fragmentComicPageMaxPanels; count++ {
		layout := fragmentComicLayoutForPanelCount(count, "3:4")
		if len(layout.Panels) != count || len(layout.ReadingOrder) != count {
			t.Fatalf("count %d produced invalid layout: %#v", count, layout)
		}
		for i, panel := range layout.Panels {
			r := panel.Rect
			if r.X < 0 || r.Y < 0 || r.Width <= 0 || r.Height <= 0 || r.X+r.Width > 1.000001 || r.Y+r.Height > 1.000001 {
				t.Fatalf("count %d panel %d outside normalized page: %#v", count, i, r)
			}
			for j := 0; j < i; j++ {
				other := layout.Panels[j].Rect
				if r.X < other.X+other.Width && r.X+r.Width > other.X && r.Y < other.Y+other.Height && r.Y+r.Height > other.Y {
					t.Fatalf("count %d panels %d and %d overlap", count, i, j)
				}
			}
		}
	}
}
