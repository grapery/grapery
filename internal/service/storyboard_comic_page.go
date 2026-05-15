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
	req.Pipeline = resolveComicPagePipeline(req.Pipeline, plannedScene, mergedSceneDescription, isTransitionScene, totalScenes)

	s.logger.Info("starting storyboard comic page generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("sceneId", req.SceneID),
		zap.String("layoutPreset", req.Pipeline.LayoutPreset),
		zap.Int("panelCount", req.Pipeline.PanelCount),
		zap.String("aspectRatio", req.Pipeline.PageAspectRatio))

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
	plannedScene, totalScenes := s.storyboardSceneContextForComicPage(ctx, gen.StoryboardID, gen.SceneID, nil)
	if huoshanOK || geminiOK {
		promptGen := s.buildComicPageImageGenerationLLMPrompt(gen, opts, plannedScene, totalScenes)
		text, inTok, outTok, totTok, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, promptGen, "comic_page_image_prompt", 4096, 0.35, true, 0.35, 4096)
		if err != nil {
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(classifyGenerationError(err, GenerationErrorProvider), err.Error())
			_ = s.repo.UpdateImageGeneration(ctx, gen)
			if !gen.SkipPeerFailureGate {
				s.cancelInFlightSiblingStoryboardImageGenerations(ctx, gen.StoryboardID, gen.ID)
			}
			s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "comic_page_image_prompt", prov, promptGen, "", 0, 0, domain.AITaskStatusFailed, err.Error())
			if s.metrics != nil {
				s.metrics.RecordStoryboardImageGeneration("failed", sceneType, time.Since(startTime))
			}
			return
		}

		promptDetails, _ := s.parseImagePromptDetails(text, gen.SceneTitle, gen.SceneDescription)
		if promptDetails != nil {
			mergePlannedStoryboardComicTextsIntoDetails(promptDetails, plannedScene)
			gen.PromptDetails = promptDetails
			gen.GeneratedPrompt = s.combineImagePrompt(promptDetails, gen.SceneTitle, gen.SceneDescription)
		} else if plannedScene != nil && len(plannedScene.ComicTexts) > 0 {
			fallback := &domain.ImagePromptDetails{
				ComicTexts: append([]domain.StoryboardComicText(nil), plannedScene.ComicTexts...),
			}
			gen.PromptDetails = fallback
			gen.GeneratedPrompt = s.combineImagePrompt(fallback, gen.SceneTitle, gen.SceneDescription)
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
	guard := fmt.Sprintf("[HARD OUTPUT REQUIREMENT] Generate ONE single image that is a complete comic page, NOT one cinematic illustration. The page must contain exactly %d visibly separated comic panel(s) using layout %s (%s), clear gutters/panel borders when panelCount > 1, reading order left-to-right then top-to-bottom, and page aspect ratio %s. Each panel must show a distinct beat from the same scene; do not add extra panels beyond this exact count.", panelCount, layout, comicPageLayoutDescription(layout, panelCount), aspect)
	if strings.TrimSpace(opts.DialogueMode) != "none" {
		guard += " FORBIDDEN: empty speech balloons or empty thought bubbles — either omit bubbles entirely or fill them with the exact Chinese strings required in the COMIC LETTERING section (no blank outlines)."
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

// buildComicPageImageGenerationLLMPrompt 让文本模型为「一页多格漫画」输出结构化 JSON，
// 包含每格分镜描述、漫画文字层（对白/思想泡/拟声/旁白）及整页视觉参数。
func (s *Service) buildComicPageImageGenerationLLMPrompt(gen *domain.StoryboardImageGeneration, opts ComicPagePipelineOptions, plannedScene *domain.StoryboardScene, totalScenes int) string {
	var b strings.Builder

	dialogueMode := strings.TrimSpace(opts.DialogueMode)
	dialogueModeDesc := "include speech balloons and comic lettering where the scene calls for it"
	if dialogueMode == "none" {
		dialogueModeDesc = "no speech balloons or dialogue text — purely atmospheric / visual storytelling"
	}

	b.WriteString(fmt.Sprintf(`You are a professional manga/webtoon panel director and image prompt engineer.
Create a detailed prompt for a SINGLE OUTPUT IMAGE that is one comic page containing exactly %d sequential panel(s).

Panel count    : %d (server-selected automatically from storyboard/chapter context; do NOT change it)
Layout preset  : %s — %s
Output ratio   : %s
Dialogue mode  : %s (%s)
Count rationale: %s

Page-level rules:
- Clear gutters between panels (2–4%% of page width, dark or white).
- Consistent character design, palette, and line quality across all panels.
- Reading order: left-to-right, top-to-bottom (webtoon / manga right-to-left only if style demands).
- Lettering must be drawn inside the image by the image model — NO app-side overlay.
- Use structured manga controls: panel grid, bubble shape, SFX typography, effect lines, screentone/shading, and gutter closure must each appear in the JSON when relevant.
- The exact panel count is part of the art direction: never add extra panels, and never collapse multiple required beats into a hidden eighth/ninth panel.

`,
		opts.PanelCount,
		opts.PanelCount,
		strings.TrimSpace(opts.LayoutPreset),
		comicPageLayoutDescription(opts.LayoutPreset, opts.PanelCount),
		strings.TrimSpace(opts.PageAspectRatio),
		dialogueMode,
		dialogueModeDesc,
		comicPagePanelCountReason(plannedScene, gen.SceneDescription, gen.IsTransitionScene, totalScenes),
	))

	// 与 internal/service 中「故事碎片」多格规划（fragment_panel_plan_prompts）及 expandScenes 配图方法论对齐的精简版，
	// 避免漫画页链路提示词显著弱于碎片生成链路。
	b.WriteString(`
## Creation methodology (align with story-fragment panel & image prompts)
Same product rules as fragment multi-panel Reference → Plan → Image pipeline; you MUST reflect them in the JSON.

`)
	b.WriteString(structuredMangaLanguageGuidance())
	b.WriteString(`

### 一致性
- Maintain cast wardrobe, face, hair, body silhouette across panels when references exist; continuity with the supplied previous-panel reference when listed.
- artStyle MUST be ONE executable English art-direction paragraph (medium, ink/paint workflow, era, lineage) reusable as the signature for every panel on this page — comparable to fragment visualBible.styleBible.artStyle.

### Narrative beats (per panel is a NEW beat within the SAME scene arc)
- Do NOT redraw the same shot with tiny tweaks panel after panel — especially after panel 1. Panel 1 may establish geography; subsequent panels MUST change framing, rhythm, blocking, emotion, or time beat.
- Atmosphere-only panels (no progression) are allowed as rhythmic breathers — still need distinct framing vs neighbors.
- Build a real manga rhythm, not a contact sheet: include clear panel roles such as setup, reaction, inner thought, turning point, shock, anticipation, release/celebration, or transition when the scene supports them.
- If the page has four or more panels and is not purely atmospheric, at least one panel should carry an emotional punctuation beat (shock / anticipation / turning_point / celebration / inner_monologue / dialogue) with matching panel language.

### Shot diversity (HARD constraints)
- **Adjacent panels:** never repeat the SAME pairing of ( shot scale + camera angle ); e.g. two consecutive panels cannot both be "medium shot + eye level".
- Ideal: avoid repeating any scale+angle pair within sliding window of THREE panels when possible.
- If the narrative contains combat, impact, shouting, smashing, falling, chasing, fear, climax, or explosion semantics, at least two panels must use impact controls such as extreme low-angle, dramatic high-angle, wide-angle distortion, radial action lines, debris/sparks, border breaking, heavy ink contrast, or dynamic screentones.

### Toolbox (use intentionally; diversify)
- Shot scale: extreme close-up, close-up, medium, full, wide, extreme wide.
- Angle: eye level, high angle, low angle, Dutch tilt, overhead, worm's-eye, POV/non-human POVs when justified.
- You may blend establishing + insert, contrast sizes on the same row, asymmetric gutters.

### Panel description quality (maps to keyElements[])
- Each entry is 1–2 Chinese sentences readable as a screenplay beat (who / doing what / vibe / narrative moment) PLUS an inline English cue for shot_scale + angle pairs, e.g. …（close_up + low_angle）.
- Comparable to fragment "sceneDesc" + shot vocabulary; NOT a bland plot summary ("第二幕发生冲突").

### English fields for downstream image synthesis
- "lighting": page-level + note if shifts per region/panel strip.
- "colorPalette": concrete grading (dominant hues, accents, saturation).
- "composition": MUST describe how the preset layout geometry maps onto the canvas: rows/columns, wide hero strip, gutters and color, silhouette reading flow — English prose referencing the preset name.

`)

	b.WriteString(fmt.Sprintf("## Scene narrative\nTitle: %s\nDescription: %s\n", gen.SceneTitle, gen.SceneDescription))

	if gen.StoryStyle != nil {
		b.WriteString("\n## Story style\n")
		b.WriteString(fmt.Sprintf("- Slug : %s\n", gen.StoryStyle.Style))
		if gen.StoryStyle.Description != "" {
			b.WriteString(fmt.Sprintf("- Notes: %s\n", gen.StoryStyle.Description))
		}
		b.WriteString("The entire comic page MUST match this style for art direction, palette, and line quality.\n")
	}

	if cs := strings.TrimSpace(gen.ComicStyle); cs != "" {
		b.WriteString("\n## Comic / visual style (same taxonomy as story fragments)\n")
		b.WriteString(fmt.Sprintf("- Style slug       : %s\n", cs))
		b.WriteString(fmt.Sprintf("- Human-readable zh: %s\n", fragmentStyleDesc(cs)))
		b.WriteString("Apply consistently to line ink, shading, saturation, lettering hand, panel border treatment.\n")
	}

	if len(gen.ReferenceImages) > 0 {
		b.WriteString("\n## Reference images (ordered)\n")
		b.WriteString("FIRST reference is usually the preceding storyboard scene panel: continuity of palette/environment/costume and shot-to-shot handoff — do not ignore it unless the narration demands a deliberate break.\n")
		b.WriteString("Following refs are MAIN CAST identity anchors: face, hairstyle, attire must match refs on every appearance.\n")
		b.WriteString("You are extending the SAME scene narrative into a comic page layout; avoid \"same pose micro-variation\" clones across grids — reference locks identity only, NOT identical framing recycle.\n")
	}

	if plannedScene != nil {
		hasComicPlan := len(plannedScene.ComicTexts) > 0 ||
			strings.TrimSpace(plannedScene.LayoutIntent) != "" ||
			strings.TrimSpace(plannedScene.CompositionPlan) != "" ||
			strings.TrimSpace(plannedScene.ShotType) != "" ||
			strings.TrimSpace(plannedScene.VisualHierarchy) != ""
		if hasComicPlan {
			b.WriteString("\n## Pre-planned comic metadata from text stage (highest priority)\n")
			if v := strings.TrimSpace(plannedScene.LayoutIntent); v != "" {
				b.WriteString(fmt.Sprintf("- layoutIntent: %s\n", v))
			}
			if v := strings.TrimSpace(plannedScene.CompositionPlan); v != "" {
				b.WriteString(fmt.Sprintf("- compositionPlan: %s\n", v))
			}
			if v := strings.TrimSpace(plannedScene.ShotType); v != "" {
				b.WriteString(fmt.Sprintf("- shotType: %s\n", v))
			}
			if v := strings.TrimSpace(plannedScene.VisualHierarchy); v != "" {
				b.WriteString(fmt.Sprintf("- visualHierarchy: %s\n", v))
			}
			if len(plannedScene.ComicTexts) > 0 {
				b.WriteString("- comicTexts:\n")
				for _, ct := range plannedScene.ComicTexts {
					b.WriteString(fmt.Sprintf("  - type=%s speaker=%s position=%s text=%s\n",
						strings.TrimSpace(ct.Type),
						strings.TrimSpace(ct.Speaker),
						strings.TrimSpace(ct.Position),
						strings.TrimSpace(ct.Text),
					))
				}
			}
			b.WriteString("You MUST treat this as pre-production comic planning decided during text/storyboard authoring. Reuse it directly in output fields and only minimally adapt wording for image-model executability.\n")
		}
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

Emotional punctuation guide:
  shock          → close-up / Dutch angle / radial shock lines / sweat drop / short interjection like 「啊？」.
  anticipation   → negative space / door-gap, hand pause, countdown, off-frame gaze / caption or thought like 「要来了」 or 「……」.
  turning_point  → large or broken-border panel, silhouette, sudden lighting shift, caption that marks the irreversible reveal.
  celebration    → warm open composition, confetti/star accents/crowd reaction, short cheering dialogue like 「太好了！」.
  inner thought  → thought bubble, low-saturation background, eye/hand detail, do not cram long prose into the bubble.
`)
	}

	b.WriteString(fmt.Sprintf(`
## Output format
Return ONLY a JSON object (no markdown, no commentary). Schema:
{
  "artStyle": "Executable English paragraph: medium + line rendering + screentone/shading + texture + palette vibe + lineage (fragment visual-bible calibre). Aim ≥ 35 English words; must unify all panels.",
  "lighting": "Lighting brief for the whole page; note if shifts between panels/regions; include high contrast/chiaroscuro/noir lighting when impact semantics exist.",
  "colorPalette": "Palette + grading; concrete hues, saturation, black ink mass strategy vs fragment image prompts.",
  "composition": "Describe the %d-panel grid %s: rows/columns, wide/spotlight band, gutter closure, border style, silhouette reading flow, and any border-breaking panel.",
  "keyElements": [
    "Panel 1 — 中文 1–2 句可读画面节拍 + （English shot_scale + angle） …",
    "Panel 2 — …",
    "... (exactly %d entries; obey shot-diversity HARD rules)"
  ],
  "mood": "Overall emotional tone of the page.",
  "additionalNotes": "Panel border style, reading order note, balloon tail directions, SFX font treatment, screentone/effect-line plan, gutter closure, impact package if triggered.",
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
LENGTH: Keep field strings tight but artStyle/keyElements dense enough that the merged English prompt survives downstream image-gen (fragment-quality bar).
The server prepends full scene narrative; each keyElements line must embed both Chinese beat-writing and embedded English lens vocabulary.`,
		opts.PanelCount,
		strings.TrimSpace(opts.LayoutPreset),
		opts.PanelCount,
		opts.PanelCount,
	))

	legacyPrompt := strings.TrimSpace(b.String())
	return renderPromptDSL(PromptDSL{
		Role: "You are a professional manga/webtoon panel director and image prompt engineer.",
		Task: "Create a structured JSON prompt for one single output image that contains a full multi-panel comic page.",
		Inputs: map[string]any{
			"sceneTitle":       gen.SceneTitle,
			"sceneDescription": gen.SceneDescription,
			"panelCount":       opts.PanelCount,
			"layoutPreset":     strings.TrimSpace(opts.LayoutPreset),
			"pageAspectRatio":  strings.TrimSpace(opts.PageAspectRatio),
			"dialogueMode":     strings.TrimSpace(opts.DialogueMode),
			"storyStyleSlug": func() string {
				if gen.StoryStyle != nil {
					return gen.StoryStyle.Style
				}
				return ""
			}(),
			"comicStyleSlug":    strings.TrimSpace(gen.ComicStyle),
			"isTransitionScene": gen.IsTransitionScene,
			"sceneCharacters":   gen.SceneCharacters,
			"referenceImages":   gen.ReferenceImages,
			"preplannedLayoutIntent": func() string {
				if plannedScene == nil {
					return ""
				}
				return strings.TrimSpace(plannedScene.LayoutIntent)
			}(),
			"preplannedCompositionPlan": func() string {
				if plannedScene == nil {
					return ""
				}
				return strings.TrimSpace(plannedScene.CompositionPlan)
			}(),
			"preplannedShotType": func() string {
				if plannedScene == nil {
					return ""
				}
				return strings.TrimSpace(plannedScene.ShotType)
			}(),
			"preplannedVisualHierarchy": func() string {
				if plannedScene == nil {
					return ""
				}
				return strings.TrimSpace(plannedScene.VisualHierarchy)
			}(),
			"preplannedComicTexts": func() []domain.StoryboardComicText {
				if plannedScene == nil {
					return nil
				}
				return plannedScene.ComicTexts
			}(),
		},
		GlobalConfig: structuredMangaLanguageGuidance(),
		Sections: []PromptDSLSection{
			{Title: "Detailed Instructions", Kind: "text", Body: legacyPrompt},
		},
	})
}
