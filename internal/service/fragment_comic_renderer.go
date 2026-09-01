package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	xdraw "golang.org/x/image/draw"
)

// generateFragmentComicPageFromPanels renders deterministic panel assets first
// and then composites them into the layout owned by ComicDocument v2. Text in
// overlay mode intentionally remains outside the pixels for exact App editing.
func (s *FragmentGenerationService) generateFragmentComicPageFromPanels(
	ctx context.Context,
	userID, taskID, language string,
	bible *domain.FragmentVisualBible,
	scene *domain.FragmentScenePlan,
	referenceAssets []domain.FragmentReferenceAsset,
	userRefURLs []string,
	policy *domain.FragmentConsistencyPolicy,
) (string, []string, int, error) {
	if scene == nil || scene.ComicPage == nil || len(scene.ComicPage.Panels) == 0 {
		return "", nil, 0, fmt.Errorf("comic page plan is empty")
	}
	layout := scene.ComicPage.Layout
	if layout == nil || len(layout.Panels) != len(scene.ComicPage.Panels) {
		generated := fragmentComicLayoutForPanelCount(len(scene.ComicPage.Panels), domain.FragmentComicPageAspectDefault)
		layout = &generated
		scene.ComicPage.Layout = layout
	}

	panelURLs := make([]string, len(scene.ComicPage.Panels))
	copy(panelURLs, scene.PanelImageURLs)
	scene.PanelImageURLs = panelURLs
	totalTokens := 0
	for panelIndex, panel := range scene.ComicPage.Panels {
		if err := ctx.Err(); err != nil {
			return "", panelURLs, totalTokens, err
		}
		if s.fragmentGenRepo != nil && !s.fragmentGenerationTaskCanContinue(ctx, taskID) {
			return "", panelURLs, totalTokens, context.Canceled
		}
		if strings.TrimSpace(panelURLs[panelIndex]) != "" {
			continue
		}
		panelScene := domain.FragmentScenePlan{
			Index:          panelIndex,
			SceneDesc:      panel.SceneDesc,
			ImagePrompt:    buildFragmentPanelArtworkPrompt(panel, language),
			ReferenceKeys:  panel.ReferenceKeys,
			EntityBindings: panel.EntityBindings,
		}
		refs := mergeFragmentSceneReferenceAssets(userRefURLs, panelScene, referenceAssets, fragmentMaxSceneReferenceImages)
		payload := map[string]interface{}{
			"prompt":      buildFragmentPanelRenderPrompt(bible, panelScene, language),
			"aspectRatio": fragmentPanelAspectRatio(layout.Panels[panelIndex].Rect, layout.PageAspectRatio),
			"seed":        fragmentStoryImageSeed(policy),
			"options":     cloneFragmentProviderOptions(policy),
		}
		if len(refs) > 0 {
			payload["referenceImages"] = refs
		}
		raw, _ := json.Marshal(payload)
		request := &domain.AITask{
			ID: uuid.NewString(), UserID: userID,
			Type: domain.AITaskGenerateFragmentImages, Status: domain.AITaskStatusProcessing,
			Input: string(raw), RelatedEntityID: taskID, RelatedEntityType: "fragment_comic_panel",
		}
		url, tokens, err := s.generateFragmentSceneImageWithRetry(ctx, request, scene.Index*100+panelIndex)
		totalTokens += tokens
		if err != nil {
			return "", panelURLs, totalTokens, fmt.Errorf("render panel %d: %w", panelIndex+1, err)
		}
		panelURLs[panelIndex] = url
	}

	pageBytes, err := compositeFragmentComicPage(ctx, layout, panelURLs)
	if err != nil {
		return "", panelURLs, totalTokens, err
	}
	ossClient := aliyun.GetGlobalClient()
	if ossClient == nil {
		return "", panelURLs, totalTokens, fmt.Errorf("comic page compositor storage unavailable")
	}
	objectKey := fmt.Sprintf("fragments/%s/comic-pages/page-%02d-%s.jpg", taskID, scene.Index+1, uuid.NewString())
	pageURL, err := ossClient.UploadBytes(objectKey, pageBytes)
	if err != nil {
		return "", panelURLs, totalTokens, fmt.Errorf("upload composited comic page: %w", err)
	}
	return pageURL, panelURLs, totalTokens, nil
}

func buildFragmentPanelArtworkPrompt(panel domain.FragmentComicPanelPlan, language string) string {
	var builder strings.Builder
	builder.WriteString("Generate one borderless full-bleed comic panel artwork, not a page and not a collage. ")
	builder.WriteString(panel.ImagePrompt)
	fmt.Fprintf(&builder, ". Dramatic intent: %s. New visible information: %s. Shot: %s; camera: %s; composition: %s. ", panel.DramaticIntent, panel.NewInformation, panel.ShotType, panel.CameraAngle, panel.Composition)
	for _, binding := range panel.EntityBindings {
		fmt.Fprintf(&builder, "Entity %s: region %s, depth %s, scale %s, pose %s, expression %s, emotion %s, gaze toward %s; staging purpose %s. ", binding.Key, binding.Region, binding.Depth, binding.RelativeScale, binding.Pose, binding.Expression, binding.Emotion, binding.GazeTarget, binding.StagingIntent)
	}
	for _, relation := range panel.Relations {
		fmt.Fprintf(&builder, "Required staging relation: %s %s %s, shown as %s. ", relation.Subject, relation.Relation, strings.Join(relation.Object, ","), relation.VisualExpression)
	}
	builder.WriteString("No outer border, no gutters, no speech balloons, no captions, no subtitles, no random letters, no pseudo-text. ")
	if len(fragmentImageModeComicTexts(panel.ComicTexts)) > 0 {
		for _, item := range fragmentImageModeComicTexts(panel.ComicTexts) {
			fmt.Fprintf(&builder, "Draw only the exact %s sound effect %q at %s, with tone %s. ", generationLanguageName(language), sanitizeComicPromptText(item.Text), item.Position, item.Tone)
		}
	}
	return strings.TrimSpace(builder.String())
}

func buildFragmentPanelRenderPrompt(bible *domain.FragmentVisualBible, scene domain.FragmentScenePlan, language string) string {
	var builder strings.Builder
	if bible != nil && bible.StyleBible != nil {
		if value := strings.TrimSpace(bible.StyleBible.ArtStyle); value != "" {
			fmt.Fprintf(&builder, "Global art style: %s.\n", value)
		}
		if value := strings.TrimSpace(bible.StyleBible.LineQuality); value != "" {
			fmt.Fprintf(&builder, "Line and rendering discipline: %s.\n", value)
		}
		if value := strings.TrimSpace(bible.StyleBible.Palette); value != "" {
			fmt.Fprintf(&builder, "Series color script: %s.\n", value)
		}
	}
	fmt.Fprintf(&builder, "Narrative beat (%s): %s.\n", generationLanguageName(language), scene.SceneDesc)
	fmt.Fprintf(&builder, "Visual execution: %s.\n", scene.ImagePrompt)
	writeFragmentActiveEntities(&builder, bible, scene.ReferenceKeys)
	if len(scene.ReferenceKeys) > 1 {
		builder.WriteString("Do not merge or swap identities, clothing, props, positions, or ownership.\n")
	}
	builder.WriteString(fullBleedCanvasDirective())
	return strings.TrimSpace(builder.String())
}

func fragmentImageModeComicTexts(items []domain.FragmentComicText) []domain.FragmentComicText {
	out := make([]domain.FragmentComicText, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.RenderMode), "image") {
			out = append(out, item)
		}
	}
	return out
}

func fragmentPanelAspectRatio(rect domain.FragmentComicRect, pageAspectRatio string) string {
	if rect.Height <= 0 || rect.Width <= 0 {
		return "1:1"
	}
	width, height := fragmentComicCanvasSize(pageAspectRatio)
	ratio := rect.Width * float64(width) / (rect.Height * float64(height))
	best, distance := "1:1", math.Inf(1)
	for _, candidate := range []string{"1:1", "3:4", "4:3", "9:16", "16:9"} {
		w, h := fragmentComicCanvasSize(candidate)
		if delta := math.Abs(math.Log(ratio / (float64(w) / float64(h)))); delta < distance {
			best, distance = candidate, delta
		}
	}
	return best
}

func compositeFragmentComicPage(ctx context.Context, layout *domain.FragmentComicLayout, panelURLs []string) ([]byte, error) {
	if layout == nil || len(layout.Panels) == 0 || len(layout.Panels) != len(panelURLs) {
		return nil, fmt.Errorf("comic page layout and panel assets do not match")
	}
	width, height := fragmentComicCanvasSize(layout.PageAspectRatio)
	for _, panel := range layout.Panels {
		r := panel.Rect
		if !validFragmentComicRect(r) || fragmentComicPixelRect(r, width, height).Inset(maxInt(2, width/500)).Empty() {
			return nil, fmt.Errorf("invalid comic panel rectangle at index %d", panel.Index)
		}
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	client := &http.Client{Timeout: 30 * time.Second}
	for index, panelLayout := range layout.Panels {
		panelImage, err := downloadFragmentComicPanel(ctx, client, panelURLs[index])
		if err != nil {
			return nil, fmt.Errorf("download panel %d: %w", index+1, err)
		}
		rect := fragmentComicPixelRect(panelLayout.Rect, width, height)
		draw.Draw(canvas, rect, image.NewUniform(color.Black), image.Point{}, draw.Src)
		inner := rect.Inset(maxInt(2, width/500))
		drawFragmentComicCover(canvas, inner, panelImage)
	}
	var output bytes.Buffer
	if err := jpeg.Encode(&output, canvas, &jpeg.Options{Quality: 92}); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func downloadFragmentComicPanel(ctx context.Context, client *http.Client, rawURL string) (image.Image, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	const maxBytes = 24 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("comic panel image exceeds size limit")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 32_000_000 {
		return nil, fmt.Errorf("comic panel image exceeds pixel limit")
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	return decoded, err
}

func validFragmentComicRect(r domain.FragmentComicRect) bool {
	return r.X >= 0 && r.Y >= 0 && r.Width > 0 && r.Height > 0 &&
		r.X+r.Width <= 1 && r.Y+r.Height <= 1
}

func drawFragmentComicCover(destination *image.RGBA, target image.Rectangle, source image.Image) {
	sourceBounds := source.Bounds()
	targetRatio := float64(target.Dx()) / float64(maxInt(target.Dy(), 1))
	sourceRatio := float64(sourceBounds.Dx()) / float64(maxInt(sourceBounds.Dy(), 1))
	crop := sourceBounds
	if sourceRatio > targetRatio {
		cropWidth := int(float64(sourceBounds.Dy()) * targetRatio)
		start := sourceBounds.Min.X + (sourceBounds.Dx()-cropWidth)/2
		crop = image.Rect(start, sourceBounds.Min.Y, start+cropWidth, sourceBounds.Max.Y)
	} else if sourceRatio < targetRatio {
		cropHeight := int(float64(sourceBounds.Dx()) / targetRatio)
		start := sourceBounds.Min.Y + (sourceBounds.Dy()-cropHeight)/2
		crop = image.Rect(sourceBounds.Min.X, start, sourceBounds.Max.X, start+cropHeight)
	}
	xdraw.CatmullRom.Scale(destination, target, source, crop, draw.Over, nil)
}

func fragmentComicPixelRect(rect domain.FragmentComicRect, width, height int) image.Rectangle {
	return image.Rect(
		int(rect.X*float64(width)), int(rect.Y*float64(height)),
		int((rect.X+rect.Width)*float64(width)), int((rect.Y+rect.Height)*float64(height)),
	)
}

func fragmentComicCanvasSize(aspectRatio string) (int, int) {
	switch domain.NormalizeFragmentAspectRatio(aspectRatio) {
	case "1:1":
		return 1200, 1200
	case "4:3":
		return 1400, 1050
	case "16:9":
		return 1600, 900
	case "9:16":
		return 900, 1600
	default:
		return 1200, 1600
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
