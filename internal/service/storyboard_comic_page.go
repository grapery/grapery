package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"go.uber.org/zap"
)

// ComicPagePipelineOptions 专用于故事板「多格漫画页」流水线（与单图插画请求体分离）。
type ComicPagePipelineOptions struct {
	LayoutPreset    string
	PanelCount      int
	PageAspectRatio string
	DialogueMode    string
}

// ComicPageGenerationRequest POST /storyboards/:id/generate/comic-page
type ComicPageGenerationRequest struct {
	StoryboardID             string
	SceneID                  string
	SceneTitle               string
	SceneDescription         string
	ReferenceImages          []string
	SceneCharacters          []string
	CharacterReferenceImages []string
	StoryStyle               *domain.StyleConfig
	ComicStyle               string
	Pipeline                 ComicPagePipelineOptions
	// SkipPeerFailureGate 用户重试时跳过同批兄弟分镜失败闸门
	SkipPeerFailureGate bool
}

// NormalizeComicPagePipeline applies defaults and validates panel count for known layouts.
func NormalizeComicPagePipeline(o *ComicPagePipelineOptions) {
	if o == nil {
		return
	}
	if strings.TrimSpace(o.LayoutPreset) == "" {
		o.LayoutPreset = "strip5_top2_middle_wide_bottom2"
	}
	if strings.TrimSpace(o.PageAspectRatio) == "" {
		o.PageAspectRatio = "9:16"
	}
	if strings.TrimSpace(o.DialogueMode) == "" {
		o.DialogueMode = "auto"
	}
	switch strings.TrimSpace(o.LayoutPreset) {
	case "strip5_top2_middle_wide_bottom2":
		o.PanelCount = 5
	default:
		if o.PanelCount <= 0 {
			o.PanelCount = 5
		}
		if o.PanelCount > 12 {
			o.PanelCount = 12
		}
	}
}

// GenerateStoryboardComicPage 生成「单张内含多格」的漫画页（独立流水线，不复用 GenerateSceneImage 的请求体）。
func (s *Service) GenerateStoryboardComicPage(ctx context.Context, req *ComicPageGenerationRequest) (*domain.StoryboardImageGeneration, error) {
	if req == nil {
		return nil, fmt.Errorf("nil comic page request")
	}
	NormalizeComicPagePipeline(&req.Pipeline)

	s.logger.Info("starting storyboard comic page generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("sceneId", req.SceneID),
		zap.String("layoutPreset", req.Pipeline.LayoutPreset),
		zap.Int("panelCount", req.Pipeline.PanelCount),
		zap.String("aspectRatio", req.Pipeline.PageAspectRatio))

	storyboard, err := s.repo.StoryboardByID(ctx, req.StoryboardID)
	if err != nil {
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

	var storyStyle *domain.StyleConfig
	if req.StoryStyle != nil {
		storyStyle = req.StoryStyle
	} else if storyboard.StoryID != "" {
		if story, err := s.repo.StoryByID(ctx, storyboard.StoryID); err == nil && story.Style != nil {
			storyStyle = story.Style
		}
	}

	characterRefImages := req.CharacterReferenceImages
	if len(characterRefImages) == 0 && len(req.SceneCharacters) > 0 && storyboard.StoryID != "" {
		characterRefImages = s.getCharacterImagesForScene(ctx, storyboard.StoryID, req.SceneCharacters)
	}

	mergedSceneDescription := s.MergedStoryboardSceneDescriptionForImage(ctx, req.StoryboardID, req.SceneID, req.SceneDescription)

	isTransitionScene := len(req.SceneCharacters) == 0 && len(characterRefImages) == 0

	panelURL := strings.TrimSpace(s.previousStoryboardScenePanelImageURL(ctx, req.StoryboardID, req.SceneID))

	const maxSceneRefURLs = 6
	allReferenceImages := make([]string, 0, maxSceneRefURLs)
	seen := make(map[string]struct{})
	addRef := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		if len(allReferenceImages) >= maxSceneRefURLs {
			return
		}
		seen[u] = struct{}{}
		allReferenceImages = append(allReferenceImages, u)
	}
	if panelURL != "" {
		addRef(panelURL)
	}
	for _, u := range req.ReferenceImages {
		addRef(u)
	}
	if !isTransitionScene {
		for _, u := range characterRefImages {
			addRef(u)
		}
	}

	gen := &domain.StoryboardImageGeneration{
		ID:                       uuid.New().String(),
		StoryboardID:             req.StoryboardID,
		SceneID:                  req.SceneID,
		SceneTitle:               req.SceneTitle,
		SceneDescription:         mergedSceneDescription,
		ReferenceImages:          allReferenceImages,
		SceneCharacters:          req.SceneCharacters,
		CharacterReferenceImages: characterRefImages,
		StoryStyle:               storyStyle,
		IsTransitionScene:        isTransitionScene,
		ComicStyle:               strings.TrimSpace(req.ComicStyle),
		PipelineKind:             domain.StoryboardImagePipelineComicPage,
		SkipPeerFailureGate:      req.SkipPeerFailureGate,
		Status:                   domain.GenerationStatusPending,
		CreatedAt:                time.Now().Unix(),
	}

	if err := s.repo.CreateImageGeneration(ctx, gen); err != nil {
		return nil, fmt.Errorf("failed to create image generation record: %w", err)
	}

	pipe := req.Pipeline
	go s.processComicPageGeneration(context.Background(), gen, pipe)

	return gen, nil
}

func (s *Service) processComicPageGeneration(ctx context.Context, gen *domain.StoryboardImageGeneration, opts ComicPagePipelineOptions) {
	NormalizeComicPagePipeline(&opts)
	startTime := time.Now()

	sceneType := "comic_page"
	characterCount := len(gen.SceneCharacters)

	if s.metrics != nil {
		s.metrics.RecordStoryboardImageGeneration("processing", sceneType, 0)
		if characterCount > 0 {
			s.metrics.RecordImageGenerationCharacterRefs(sceneType, float64(characterCount))
			s.metrics.RecordImageGenerationWithCharacters("processing", characterCount)
		}
		hasStyle := gen.StoryStyle != nil
		s.metrics.RecordImageGenerationWithStyle("processing", hasStyle)
		if hasStyle {
			s.metrics.RecordStoryStyleConfigUsage(gen.StoryStyle.ID, "comic_page_generation")
		}
	}

	gen.Status = domain.GenerationStatusProcessing
	if err := s.repo.UpdateImageGeneration(ctx, gen); err != nil {
		s.logger.Warn("failed to update comic page generation status to processing",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	}

	if !gen.SkipPeerFailureGate && s.storyboardPeerSceneHasLatestFailedImageGen(ctx, gen.StoryboardID, gen.SceneID) {
		gen.Status = domain.GenerationStatusFailed
		gen.ErrorMessage = formatGenerationError(GenerationErrorCancelled,
			"another panel image failed first; stopping remaining image jobs in this batch")
		now := time.Now().Unix()
		gen.CompletedAt = &now
		_ = s.repo.UpdateImageGeneration(ctx, gen)
		return
	}

	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	geminiOK := s.geminiClient != nil
	if huoshanOK || geminiOK {
		promptGen := s.buildComicPageImageGenerationLLMPrompt(gen, opts)
		text, inTok, outTok, totTok, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, promptGen, "comic_page_image_prompt", 4096, 0.35, true, 0.35, 4096)
		if err != nil {
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(classifyGenerationError(err, GenerationErrorProvider), err.Error())
			_ = s.repo.UpdateImageGeneration(ctx, gen)
			if !gen.SkipPeerFailureGate {
				s.cancelInFlightSiblingStoryboardImageGenerations(ctx, gen.StoryboardID, gen.ID)
			}
			s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "comic_page_image_prompt", promptGen, 0, 0, domain.AITaskStatusFailed, err.Error())
			if s.metrics != nil {
				s.metrics.RecordStoryboardImageGeneration("failed", sceneType, time.Since(startTime))
			}
			return
		}

		promptDetails, combinedPrompt := s.parseImagePromptDetails(text, gen.SceneTitle, gen.SceneDescription)
		if promptDetails != nil {
			gen.PromptDetails = promptDetails
			gen.GeneratedPrompt = combinedPrompt
		} else {
			nb := storyboardSceneNarrativeBlock(gen.SceneTitle, gen.SceneDescription)
			gen.GeneratedPrompt = prependStoryboardImageNarrativeBlock(nb, capGeminiImageBeautyOutput(strings.TrimSpace(text)))
		}
		gen.InputTokens = inTok
		gen.OutputTokens = outTok
		gen.TotalTokens = totTok
		s.logger.Debug("comic page image prompt LLM done",
			zap.String("generationId", gen.ID),
			zap.String("provider", prov))
		s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "comic_page_image_prompt", promptGen, gen.InputTokens, gen.OutputTokens, domain.AITaskStatusCompleted, "")
	} else {
		nb := storyboardSceneNarrativeBlock(gen.SceneTitle, gen.SceneDescription)
		if nb != "" {
			gen.GeneratedPrompt = nb
		} else {
			gen.GeneratedPrompt = strings.TrimSpace(gen.SceneDescription)
		}
	}

	if gen.GeneratedPrompt != "" {
		gen.GeneratedPrompt = finalizeStoryboardImagePromptForAPI(gen)
	}

	aspect := strings.TrimSpace(opts.PageAspectRatio)
	if aspect == "" {
		aspect = "9:16"
	}

	narrativeRunes := utf8.RuneCountInString(strings.TrimSpace(gen.SceneDescription))
	refURLs, imgOp, refPrimary := selectStoryboardImageRefsAndOperation(gen, narrativeRunes)
	useRefs := len(refURLs) > 0 && imgOp == genapi.OperationImageToImage
	finalPrompt := appendStoryboardImageToImageConstraints(strings.TrimSpace(gen.GeneratedPrompt), useRefs)

	if s.genAPI != nil && finalPrompt != "" {
		genReq := &genapi.GenerateRequest{
			Prompt:          finalPrompt,
			AspectRatio:     aspect,
			Quality:         "high",
			OutputCount:     1,
			ReferenceImages: refURLs,
			Metadata:        s.usageRecordMetadataForStoryboard(ctx, gen.StoryboardID),
		}
		genReq.Operation = imgOp
		genReq.ReferenceImageURL = refPrimary
		imageProvider := s.imageProvider
		if imageProvider == "" {
			imageProvider = "huoshan"
		}
		resp, err := s.genAPI.GenerateImage(ctx, imageProvider, genReq)
		if err != nil {
			s.logger.Warn("comic page image API failed",
				zap.String("generationId", gen.ID),
				zap.Error(err))
		} else if resp != nil && len(resp.ImageURLs) > 0 {
			gen.GeneratedImageURL = s.uploadStoryboardSceneImageOSS(gen, resp.ImageURLs[0])
			if resp.Usage != nil {
				gen.TotalTokens += resp.Usage.TotalTokens
			}
		}
	}

	if latest, err := s.repo.GetImageGeneration(ctx, gen.ID); err == nil && latest != nil && latest.Status == domain.GenerationStatusFailed {
		return
	}

	if gen.GeneratedImageURL != "" {
		if !gen.SkipPeerFailureGate && s.storyboardPeerSceneHasLatestFailedImageGen(ctx, gen.StoryboardID, gen.SceneID) {
			gen.GeneratedImageURL = ""
			gen.ErrorMessage = formatGenerationError(GenerationErrorCancelled,
				"another panel image failed; discarding generated image for this batch")
		}
	}

	if gen.GeneratedImageURL == "" {
		gen.Status = domain.GenerationStatusFailed
		if gen.ErrorMessage == "" {
			gen.ErrorMessage = formatGenerationError(GenerationErrorProvider, "comic page generation failed: empty generated image url")
		}
	} else {
		gen.Status = domain.GenerationStatusCompleted
		gen.ErrorMessage = ""
	}
	now := time.Now().Unix()
	gen.CompletedAt = &now
	if err := s.repo.UpdateImageGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to update comic page generation final status",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	}

	if gen.Status == domain.GenerationStatusFailed && !gen.SkipPeerFailureGate {
		s.cancelInFlightSiblingStoryboardImageGenerations(ctx, gen.StoryboardID, gen.ID)
	}

	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	if gen.GeneratedImageURL != "" && gen.SceneID != "" {
		if err := s.repo.UpdateStoryboardSceneImage(ctx, gen.SceneID, gen.GeneratedImageURL); err != nil {
			s.logger.Warn("failed to sync comic page image to storyboard scene",
				zap.String("sceneId", gen.SceneID),
				zap.Error(err))
		}
	}

	s.afterSceneImageWritten(ctx, gen)

	if s.metrics != nil {
		duration := time.Since(startTime)
		s.metrics.RecordStoryboardImageGeneration(gen.Status, sceneType, duration)
		if characterCount > 0 {
			s.metrics.RecordImageGenerationWithCharacters(gen.Status, characterCount)
		}
		s.metrics.RecordImageGenerationWithStyle(gen.Status, gen.StoryStyle != nil)
		s.metrics.RecordAIGeneration("gemini", "comic_page")
		if gen.TotalTokens > 0 {
			s.metrics.RecordStoryboardTokenConsumed(gen.StoryboardID, float64(gen.TotalTokens))
		}
	}
}

func (s *Service) uploadStoryboardSceneImageOSS(gen *domain.StoryboardImageGeneration, originalURL string) string {
	originalImageURL := originalURL
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil {
		objectKey := fmt.Sprintf("storyboard/%s/scenes/%s.jpg", gen.StoryboardID, gen.SceneID)
		ossURL, err := ossClient.UploadFileFromURL(objectKey, originalImageURL)
		if err != nil {
			s.logger.Warn("failed to upload comic page image to OSS, using original URL",
				zap.String("sceneId", gen.SceneID),
				zap.Error(err))
			return originalImageURL
		}
		ossURL = strings.Split(ossURL, "?")[0]
		ossURL = strings.ReplaceAll(ossURL, "http://", "https://")
		return ossURL
	}
	return originalImageURL
}

// buildComicPageImageGenerationLLMPrompt 让文本模型为「一页多格漫画」输出结构化 JSON，
// 包含每格分镜描述、漫画文字层（对白/思想泡/拟声/旁白）及整页视觉参数。
func (s *Service) buildComicPageImageGenerationLLMPrompt(gen *domain.StoryboardImageGeneration, opts ComicPagePipelineOptions) string {
	var b strings.Builder

	dialogueMode := strings.TrimSpace(opts.DialogueMode)
	dialogueModeDesc := "include speech balloons and comic lettering where the scene calls for it"
	if dialogueMode == "none" {
		dialogueModeDesc = "no speech balloons or dialogue text — purely atmospheric / visual storytelling"
	}

	b.WriteString(fmt.Sprintf(`You are a professional manga/webtoon panel director and image prompt engineer.
Create a detailed prompt for a SINGLE OUTPUT IMAGE that is one multi-panel comic page containing exactly %d sequential panels.

Layout preset  : %s
Output ratio   : %s
Dialogue mode  : %s (%s)

Page-level rules:
- Clear gutters between panels (2–4%% of page width, dark or white).
- Consistent character design, palette, and line quality across all panels.
- Reading order: left-to-right, top-to-bottom (webtoon / manga right-to-left only if style demands).
- Lettering must be drawn inside the image by the image model — NO app-side overlay.

`,
		opts.PanelCount,
		strings.TrimSpace(opts.LayoutPreset),
		strings.TrimSpace(opts.PageAspectRatio),
		dialogueMode,
		dialogueModeDesc,
	))

	b.WriteString(fmt.Sprintf("## Scene narrative\nTitle: %s\nDescription: %s\n", gen.SceneTitle, gen.SceneDescription))

	if gen.StoryStyle != nil {
		b.WriteString("\n## Story style\n")
		b.WriteString(fmt.Sprintf("- Slug : %s\n", gen.StoryStyle.Style))
		if gen.StoryStyle.Description != "" {
			b.WriteString(fmt.Sprintf("- Notes: %s\n", gen.StoryStyle.Description))
		}
		b.WriteString("The entire comic page MUST match this style for art direction, palette, and line quality.\n")
	}

	if len(gen.ReferenceImages) > 0 {
		b.WriteString("\n## Reference images (character identity anchors)\n")
		b.WriteString("Use the provided reference images to maintain exact character appearance (face, hair, costume) across every panel.\n")
	}

	if gen.IsTransitionScene {
		b.WriteString("\n## Scene type: atmospheric transition\n")
		b.WriteString("No main characters appear. All panels are environment/atmosphere shots. Do NOT add dialogue, thought bubbles, or SFX.\n")
	} else if len(gen.SceneCharacters) > 0 {
		b.WriteString("\n## Characters in this scene\n")
		for _, n := range gen.SceneCharacters {
			b.WriteString(fmt.Sprintf("- %s\n", n))
		}
	}

	if dialogueMode != "none" && !gen.IsTransitionScene {
		b.WriteString(`
## Comic lettering guide (per-panel)
For any panel where a character speaks, reacts, or has a salient inner thought, include the corresponding entry in that panel's "comicTexts" array.
Lettering types and rendering rules:
  narration  → rectangular caption box, usually top or bottom of panel; use for time/place cues or omniscient narration.
  dialogue   → speech balloon (oval) with a pointed tail toward the speaker's mouth; note tail direction in additionalNotes.
  thought    → cloud/bubble-chain outline balloon; tail is a chain of small circles toward the thinker.
  sfx        → bold, oversized onomatopoeia drawn directly on the image (e.g. 砰！ 啊？ ……); no fixed bubble shape.
Text constraints: each "text" value ≤ 12 Chinese characters. Per-panel cap: ~1 narration, 1–2 dialogue, 1 sfx, 1 thought.
Reserve negative space near balloons; do not cover eyes or key props.
Silent / purely atmospheric panels → comicTexts: [].
`)
	}

	b.WriteString(fmt.Sprintf(`
## Output format
Return ONLY a JSON object (no markdown, no commentary). Schema:
{
  "artStyle": "Executable English art-direction string covering medium, line quality, rendering technique — must match story style.",
  "lighting": "Lighting brief for the whole page; note if it shifts between panels.",
  "colorPalette": "Palette and color-grading descriptor.",
  "composition": "Describe the %d-panel grid: which rows/columns each panel occupies, which panel is wide/spotlight, gutter style.",
  "keyElements": [
    "Panel 1: [shot type + character action + setting + any key prop]",
    "Panel 2: ...",
    "... (one entry per panel)"
  ],
  "mood": "Overall emotional tone of the page.",
  "additionalNotes": "Panel border style, reading order note, balloon tail directions, SFX font treatment, any special rendering instructions.",
  "comicTexts": [
    {
      "panelIndex": 0,
      "type": "narration | dialogue | thought | sfx",
      "text": "精确中文短句（≤12字）",
      "speaker": "dialogue/thought时填角色名；其余留空",
      "position": "top-left | top-right | bottom-left | bottom-right | mid-frame | speech-bubble | thought-bubble"
    }
  ]
}

Important: "keyElements" must have exactly %d entries (one per panel). "comicTexts" entries use zero-based "panelIndex".
LENGTH: Each JSON string field should be concise (one short phrase or clause). The server will prepend the full scene narrative.`,
		opts.PanelCount, opts.PanelCount,
	))

	return b.String()
}
