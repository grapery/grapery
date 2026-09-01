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

const (
	comicPageMinPanelCount     = 1
	comicPageMaxPanelCount     = 7
	comicPageDefaultPanelCount = 3
)

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

// NormalizeComicPagePipeline applies fallback defaults for legacy/internal callers.
// GenerateStoryboardComicPage overrides panel count and layout with story-driven values.
func NormalizeComicPagePipeline(o *ComicPagePipelineOptions) {
	if o == nil {
		return
	}
	if strings.TrimSpace(o.LayoutPreset) == "" {
		o.LayoutPreset = comicPageLayoutPresetForPanelCount(o.PanelCount)
	}
	if strings.TrimSpace(o.PageAspectRatio) == "" {
		o.PageAspectRatio = "9:16"
	}
	if strings.TrimSpace(o.DialogueMode) == "" {
		o.DialogueMode = "auto"
	}
	if o.PanelCount <= 0 {
		o.PanelCount = comicPageDefaultPanelCount
	}
	if o.PanelCount < comicPageMinPanelCount {
		o.PanelCount = comicPageMinPanelCount
	}
	if o.PanelCount > comicPageMaxPanelCount {
		o.PanelCount = comicPageMaxPanelCount
	}
}

func resolveComicPagePipeline(base ComicPagePipelineOptions, plannedScene *domain.StoryboardScene, sceneDescription string, isTransitionScene bool, totalScenes int) ComicPagePipelineOptions {
	panelCount := estimateComicPagePanelCount(plannedScene, sceneDescription, isTransitionScene, totalScenes)
	return ComicPagePipelineOptions{
		LayoutPreset:    comicPageLayoutPresetForPanelCount(panelCount),
		PanelCount:      panelCount,
		PageAspectRatio: firstNonBlank(base.PageAspectRatio, "9:16"),
		DialogueMode:    firstNonBlank(base.DialogueMode, "auto"),
	}
}

func estimateComicPagePanelCount(plannedScene *domain.StoryboardScene, sceneDescription string, isTransitionScene bool, totalScenes int) int {
	desc := strings.TrimSpace(sceneDescription)
	descRunes := utf8.RuneCountInString(desc)
	if isTransitionScene {
		panelCount := 1
		if descRunes > 140 || (plannedScene != nil && strings.TrimSpace(plannedScene.CompositionPlan) != "") {
			panelCount = 2
		}
		if descRunes > 300 {
			panelCount = 3
		}
		return clampComicPagePanelCount(panelCount)
	}

	panelCount := 2
	if descRunes > 60 {
		panelCount++
	}
	if descRunes > 160 {
		panelCount++
	}
	if descRunes > 280 {
		panelCount++
	}

	comicTextCount := 0
	characterCount := 0
	impactBeat := false
	if plannedScene != nil {
		characterCount = len(plannedScene.Characters)
		for _, ct := range plannedScene.ComicTexts {
			if strings.TrimSpace(ct.Text) != "" {
				comicTextCount++
			}
		}
		beatText := strings.ToLower(strings.Join([]string{
			plannedScene.BeatPurpose,
			plannedScene.LayoutIntent,
			plannedScene.CompositionPlan,
			plannedScene.ShotType,
			plannedScene.VisualHierarchy,
			plannedScene.Mood,
			plannedScene.Title,
			plannedScene.Description,
		}, " "))
		impactBeat = containsAnyFold(beatText,
			"climax", "turning", "shock", "reveal", "conflict", "battle", "celebration",
			"高潮", "转折", "震惊", "揭示", "冲突", "战斗", "庆祝", "爆发", "危机")
		if totalScenes > 0 && plannedScene.Sequence >= totalScenes-1 {
			panelCount++
		}
	}

	if comicTextCount > 0 {
		panelCount++
	}
	if comicTextCount >= 3 {
		panelCount++
	}
	if comicTextCount >= 5 {
		panelCount++
	}
	if characterCount >= 2 {
		panelCount++
	}
	if impactBeat {
		panelCount++
	}

	// Keep ordinary scenes compact. Six or seven panels are reserved for dense or pivotal beats.
	if panelCount > 5 && !impactBeat && comicTextCount < 3 && descRunes < 260 {
		panelCount = 5
	}
	if panelCount > 6 && !(impactBeat && (comicTextCount >= 4 || descRunes > 360)) {
		panelCount = 6
	}
	return clampComicPagePanelCount(panelCount)
}

func comicPageLayoutPresetForPanelCount(panelCount int) string {
	switch clampComicPagePanelCount(panelCount) {
	case 1:
		return "single_panel_full_page"
	case 2:
		return "two_panel_vertical_stack"
	case 3:
		return "three_panel_setup_reaction_release"
	case 4:
		return "four_panel_2x2_grid"
	case 5:
		return "strip5_top2_middle_wide_bottom2"
	case 6:
		return "six_panel_2x3_grid"
	case 7:
		return "seven_panel_story_page_top2_middle3_bottom2"
	default:
		return "three_panel_setup_reaction_release"
	}
}

func clampComicPagePanelCount(panelCount int) int {
	if panelCount < comicPageMinPanelCount {
		return comicPageMinPanelCount
	}
	if panelCount > comicPageMaxPanelCount {
		return comicPageMaxPanelCount
	}
	return panelCount
}

func firstNonBlank(v, fallback string) string {
	if trimmed := strings.TrimSpace(v); trimmed != "" {
		return trimmed
	}
	return fallback
}

func containsAnyFold(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

// GenerateStoryboardComicPage 生成「单张内含多格」的漫画页（独立流水线，不复用 GenerateSceneImage 的请求体）。
func (s *Service) GenerateStoryboardComicPage(ctx context.Context, req *ComicPageGenerationRequest) (*domain.StoryboardImageGeneration, error) {
	if req == nil {
		return nil, fmt.Errorf("nil comic page request")
	}

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
	plannedScene, totalScenes := s.storyboardSceneContextForComicPage(ctx, req.StoryboardID, req.SceneID, storyboard)
	mergedSceneDescription = mergeStoryboardPlannedSceneForImage(mergedSceneDescription, plannedScene)
	req.Pipeline = resolveComicPagePipeline(req.Pipeline, plannedScene, mergedSceneDescription, isTransitionScene, totalScenes)

	s.logger.Info("starting storyboard comic page generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("sceneId", req.SceneID),
		zap.String("layoutPreset", req.Pipeline.LayoutPreset),
		zap.Int("panelCount", req.Pipeline.PanelCount),
		zap.String("aspectRatio", req.Pipeline.PageAspectRatio))

	panelURL := strings.TrimSpace(s.previousStoryboardScenePanelImageURL(ctx, req.StoryboardID, req.SceneID))

	const maxSceneRefURLs = 6
	referenceManifest := make([]domain.StoryboardImageReference, 0, maxSceneRefURLs)
	seen := make(map[string]struct{})
	if panelURL != "" {
		referenceManifest = appendStoryboardImageReference(referenceManifest, seen, panelURL, domain.StoryboardImageReferencePreviousPanel, req.SceneID, maxSceneRefURLs)
	}
	if !isTransitionScene {
		for i, u := range characterRefImages {
			key := ""
			if len(characterRefImages) == len(req.SceneCharacters) && i < len(req.SceneCharacters) {
				key = req.SceneCharacters[i]
			}
			referenceManifest = appendStoryboardImageReference(referenceManifest, seen, u, domain.StoryboardImageReferenceCharacter, key, maxSceneRefURLs)
		}
	}
	for _, u := range req.ReferenceImages {
		referenceManifest = appendStoryboardImageReference(referenceManifest, seen, u, domain.StoryboardImageReferenceUser, "", maxSceneRefURLs)
	}
	allReferenceImages := storyboardReferenceURLs(referenceManifest)

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
		PlannedScene:             plannedScene,
		ReferenceManifest:        referenceManifest,
		ContentLanguage:          s.storyboardSceneContentLanguage(ctx, plannedScene, req.SceneTitle+"\n"+mergedSceneDescription),
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

	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	geminiOK := s.geminiClient != nil
	plannedScene, totalScenes := s.storyboardSceneContextForComicPage(ctx, gen.StoryboardID, gen.SceneID, nil)
	if gen.PlannedScene != nil {
		plannedScene = gen.PlannedScene
	}
	if huoshanOK || geminiOK {
		promptGen := s.buildComicPageImageGenerationLLMPrompt(gen, opts, plannedScene, totalScenes)
		text, inTok, outTok, totTok, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, promptGen, "comic_page_image_prompt", 4096, 0.35, true, 0.35, 4096)
		if err != nil {
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(classifyGenerationError(err, GenerationErrorProvider), err.Error())
			_ = s.repo.UpdateImageGeneration(ctx, gen)
			s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "comic_page_image_prompt", prov, promptGen, "", 0, 0, domain.AITaskStatusFailed, err.Error())
			if s.metrics != nil {
				s.metrics.RecordStoryboardImageGeneration("failed", sceneType, time.Since(startTime))
			}
			return
		}

		promptDetails, _ := s.parseImagePromptDetails(text, gen.SceneTitle, gen.SceneDescription)
		if promptDetails != nil {
			applyPlannedStoryboardComicTextsToDetails(promptDetails, plannedScene, gen.ContentLanguage)
			gen.PromptDetails = promptDetails
			gen.GeneratedPrompt = s.combineImagePrompt(promptDetails, gen.SceneTitle, gen.SceneDescription, gen.ContentLanguage)
		} else if plannedScene != nil && len(plannedScene.ComicTexts) > 0 {
			fallback := &domain.ImagePromptDetails{
				ComicTexts: normalizeStoryboardComicTextsForLanguage(plannedScene.ComicTexts, gen.ContentLanguage),
			}
			gen.PromptDetails = fallback
			gen.GeneratedPrompt = s.combineImagePrompt(fallback, gen.SceneTitle, gen.SceneDescription, gen.ContentLanguage)
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
		s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "comic_page_image_prompt", prov, promptGen, text, gen.InputTokens, gen.OutputTokens, domain.AITaskStatusCompleted, "")
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
		gen.GeneratedPrompt = prependComicPageImagePromptGuard(gen.GeneratedPrompt, opts)
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
		imageProvider := s.effectiveImageProvider()
		if strings.EqualFold(imageProvider, "huoshan") {
			PrepareHuoshanGenAPIImageRequest(genReq)
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
		s.metrics.RecordAIGeneration(s.effectiveImageProvider(), "comic_page")
		if gen.TotalTokens > 0 {
			s.metrics.RecordStoryboardTokenConsumed(gen.StoryboardID, float64(gen.TotalTokens))
		}
	}
}

func (s *Service) storyboardSceneContextForComicPage(ctx context.Context, storyboardID, sceneID string, storyboard *domain.Storyboard) (*domain.StoryboardScene, int) {
	if strings.TrimSpace(storyboardID) == "" || strings.TrimSpace(sceneID) == "" {
		return nil, 0
	}
	if storyboard != nil && len(storyboard.StoryboardScenes) > 0 {
		for i := range storyboard.StoryboardScenes {
			if storyboard.StoryboardScenes[i].ID == sceneID {
				return &storyboard.StoryboardScenes[i], len(storyboard.StoryboardScenes)
			}
		}
		return nil, len(storyboard.StoryboardScenes)
	}
	scenes, err := s.repo.StoryboardScenes(ctx, storyboardID)
	if err != nil {
		return nil, 0
	}
	for _, scene := range scenes {
		if scene != nil && scene.ID == sceneID {
			return scene, len(scenes)
		}
	}
	return nil, len(scenes)
}

func (s *Service) lookupStoryboardSceneForComicPage(ctx context.Context, storyboardID, sceneID string) *domain.StoryboardScene {
	scene, _ := s.storyboardSceneContextForComicPage(ctx, storyboardID, sceneID, nil)
	return scene
}

func comicPageLayoutDescription(layout string, panelCount int) string {
	switch strings.TrimSpace(layout) {
	case "single_panel_full_page":
		return "one full-page splash panel"
	case "two_panel_vertical_stack":
		return "two stacked horizontal panels: top setup, bottom reaction or reveal"
	case "three_panel_setup_reaction_release":
		return "three readable panels with setup, reaction, and release/turn"
	case "four_panel_2x2_grid":
		return "balanced 2x2 grid"
	case "strip5_top2_middle_wide_bottom2":
		return "five panels: two small top panels, one wide middle emphasis panel, two bottom panels"
	case "six_panel_2x3_grid":
		return "six panels in a compact 2-column by 3-row story grid"
	case "seven_panel_story_page_top2_middle3_bottom2":
		return "seven compact panels: two top panels, three middle beats, two bottom panels"
	default:
		if panelCount == 1 {
			return "one full-page splash panel"
		}
		return fmt.Sprintf("%d story-driven panels with readable gutters", clampComicPagePanelCount(panelCount))
	}
}

func comicPagePanelCountReason(plannedScene *domain.StoryboardScene, sceneDescription string, isTransitionScene bool, totalScenes int) string {
	descRunes := utf8.RuneCountInString(strings.TrimSpace(sceneDescription))
	if isTransitionScene {
		return fmt.Sprintf("atmospheric transition scene; description length=%d runes; compact pacing", descRunes)
	}
	comicTextCount := 0
	characterCount := 0
	beatPurpose := ""
	sequence := 0
	if plannedScene != nil {
		characterCount = len(plannedScene.Characters)
		beatPurpose = strings.TrimSpace(plannedScene.BeatPurpose)
		sequence = plannedScene.Sequence
		for _, ct := range plannedScene.ComicTexts {
			if strings.TrimSpace(ct.Text) != "" {
				comicTextCount++
			}
		}
	}
	return fmt.Sprintf("story-driven auto count from chapter context: scene sequence=%d/%d, description length=%d runes, characters=%d, comicTexts=%d, beatPurpose=%s",
		sequence, totalScenes, descRunes, characterCount, comicTextCount, beatPurpose)
}

func prependComicPageImagePromptGuard(prompt string, opts ComicPagePipelineOptions) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return prompt
	}
	panelCount := opts.PanelCount
	if panelCount <= 0 {
		panelCount = comicPageDefaultPanelCount
	}
	layout := strings.TrimSpace(opts.LayoutPreset)
	if layout == "" {
		layout = comicPageLayoutPresetForPanelCount(panelCount)
	}
	aspect := strings.TrimSpace(opts.PageAspectRatio)
	if aspect == "" {
		aspect = "9:16"
	}
	guard := fmt.Sprintf("[HARD OUTPUT REQUIREMENT] Generate ONE single image that is a complete comic page, NOT one cinematic illustration. The page must contain exactly %d visibly separated comic panel(s) using layout %s (%s), clear gutters/panel borders between panels when panelCount > 1, reading order left-to-right then top-to-bottom, and page aspect ratio %s. Each panel must show a distinct beat from the same scene; do not add extra panels beyond this exact count. %s", panelCount, layout, comicPageLayoutDescription(layout, panelCount), aspect, fullBleedCanvasDirective())
	if strings.TrimSpace(opts.DialogueMode) != "none" {
		guard += " FORBIDDEN: empty speech balloons or empty thought bubbles — either omit bubbles entirely or fill them with the exact supplied lettering-language strings required by the scene plan (no blank outlines)."
	}
	return guard + "\n\n" + prompt
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
