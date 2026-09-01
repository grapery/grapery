package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestCompositeFragmentComicPageRejectsInvalidGeometryBeforeDownload(t *testing.T) {
	for _, rect := range []domain.FragmentComicRect{
		{X: -1, Width: .4, Height: .4},
		{Width: 0, Height: .4},
		{Width: 2, Height: .4},
		{X: math.NaN(), Width: .4, Height: .4},
	} {
		layout := &domain.FragmentComicLayout{Panels: []domain.FragmentComicPanelLayout{{Rect: rect}}}
		if _, err := compositeFragmentComicPage(t.Context(), layout, []string{"invalid://must-not-fetch"}); err == nil {
			t.Fatalf("invalid rectangle accepted: %#v", rect)
		}
	}
}

func TestFragmentPanelAspectRatioUsesPageGeometry(t *testing.T) {
	for _, test := range []struct{ page, want string }{
		{"3:4", "3:4"}, {"9:16", "9:16"}, {"4:3", "4:3"}, {"16:9", "16:9"},
	} {
		t.Run(test.page, func(t *testing.T) {
			got := fragmentPanelAspectRatio(domain.FragmentComicRect{Width: .4, Height: .4}, test.page)
			if got != test.want {
				t.Fatalf("normalized square on %s page must be %s, got %s", test.page, test.want, got)
			}
		})
	}
}

func TestFragmentPanelCancellationPreservesCachedAssets(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	scene := domain.FragmentScenePlan{
		ComicPage:      &domain.FragmentComicPagePlan{Panels: make([]domain.FragmentComicPanelPlan, 3)},
		PanelImageURLs: []string{"https://example.com/completed.png"},
	}
	service := &FragmentGenerationService{}
	_, urls, tokens, err := service.generateFragmentComicPageFromPanels(ctx, "u", "t", "zh", nil, &scene, nil, nil, nil)
	if !errors.Is(err, context.Canceled) || tokens != 0 || len(urls) != 3 || urls[0] != scene.PanelImageURLs[0] {
		t.Fatalf("cancellation must preserve completed assets without generating more: %v %v %d", err, urls, tokens)
	}
}

func TestCompositeFragmentComicPageUsesDeterministicLayout(t *testing.T) {
	panel := image.NewRGBA(image.Rect(0, 0, 120, 80))
	for y := 0; y < panel.Bounds().Dy(); y++ {
		for x := 0; x < panel.Bounds().Dx(); x++ {
			panel.Set(x, y, color.RGBA{R: 220, G: uint8(x), B: uint8(y), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, panel); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "image/png")
		_, _ = writer.Write(encoded.Bytes())
	}))
	defer server.Close()

	layout := fragmentComicLayoutForPanelCount(3, "3:4")
	page, err := compositeFragmentComicPage(t.Context(), &layout, []string{server.URL, server.URL, server.URL})
	if err != nil {
		t.Fatalf("composite failed: %v", err)
	}
	decoded, _, err := image.Decode(bytes.NewReader(page))
	if err != nil {
		t.Fatalf("decode composite: %v", err)
	}
	if decoded.Bounds().Dx() != 1200 || decoded.Bounds().Dy() != 1600 {
		t.Fatalf("expected deterministic 3:4 canvas, got %v", decoded.Bounds())
	}
	for _, item := range layout.Panels {
		rect := fragmentComicPixelRect(item.Rect, 1200, 1600)
		if got := color.RGBAModel.Convert(decoded.At(rect.Min.X, rect.Min.Y)).(color.RGBA); got.R > 20 || got.G > 20 || got.B > 20 {
			t.Fatalf("expected black panel border at %v, got %#v", rect.Min, got)
		}
	}
}

func TestFragmentPanelArtworkPromptKeepsOverlayTextOutOfPixels(t *testing.T) {
	panel := domain.FragmentComicPanelPlan{
		ImagePrompt: "hero confronts the guardian",
		ComicTexts: []domain.FragmentComicText{
			{Type: "dialogue", Text: "停下", RenderMode: "overlay"},
			{Type: "sfx", Text: "砰", RenderMode: "image", Position: "right"},
		},
	}
	prompt := buildFragmentPanelArtworkPrompt(panel, "zh-Hans")
	if bytes.Contains([]byte(prompt), []byte("停下")) {
		t.Fatalf("overlay dialogue leaked into image prompt: %s", prompt)
	}
	if !bytes.Contains([]byte(prompt), []byte("砰")) {
		t.Fatalf("image-mode SFX missing from image prompt: %s", prompt)
	}
}
