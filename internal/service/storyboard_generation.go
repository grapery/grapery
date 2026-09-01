package service

import (
	"context"
	"encoding/json"
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

// ContentGenerationRequest represents a request to generate storyboard content
type ContentGenerationRequest struct {
	StoryboardID string   `json:"storyboardId"`
	RawInput     string   `json:"rawInput"`
	CharacterIDs []string `json:"characterIds"`
	SceneIDs     []string `json:"sceneIds"`
	Style        string   `json:"style"` // action, drama, comedy, mystery
}

// SceneDetailRequest represents a request to generate scene details
type SceneDetailRequest struct {
	StoryboardID     string `json:"storyboardId"`
	SceneID          string `json:"sceneId"`
	SceneTitle       string `json:"sceneTitle"`
	SceneLocation    string `json:"sceneLocation"`
	InputDescription string `json:"inputDescription"`
}

// ImageGenerationRequest represents a request to generate scene images
type ImageGenerationRequest struct {
	StoryboardID     string   `json:"storyboardId"`
	SceneID          string   `json:"sceneId"`
	SceneTitle       string   `json:"sceneTitle"`
	SceneDescription string   `json:"sceneDescription"`
	ReferenceImages  []string `json:"referenceImages"`

	// 场景关联的角色名称列表（用于判断是否为过渡场景）
	SceneCharacters []string `json:"sceneCharacters,omitempty"`
	// 场景关联的角色图片（用于 AI 生成时作为参考）
	CharacterReferenceImages []string `json:"characterReferenceImages,omitempty"`
	// AspectRatio is the user-selected output frame ratio for this generation turn.
	AspectRatio string `json:"aspectRatio,omitempty"`

	// 故事风格配置
	StoryStyle *domain.StyleConfig `json:"storyStyle,omitempty"`
	// ComicStyle 漫画/视觉风格 slug（如续写 continuation_comic_style）
	ComicStyle string `json:"comicStyle,omitempty"`
	// SkipPeerFailureGate 为 true 时跳过「同批其他分镜已失败则放弃」（用于用户主动重试）。
	SkipPeerFailureGate bool `json:"-"`
}

// RetryFailedStoryboardImageOptions optional flags for POST retry-failed-images.
type RetryFailedStoryboardImageOptions struct {
	// ForceComicPagePipeline 为 true 时对失败分镜使用漫画页管线（兼容历史行未写入 pipeline_kind）。
	ForceComicPagePipeline bool `json:"forceComicPagePipeline"`
}

// VideoGenerationRequest represents a request to generate scene videos
type VideoGenerationRequest struct {
	StoryboardID      string `json:"storyboardId"`
	SceneID           string `json:"sceneId"`
	SceneTitle        string `json:"sceneTitle"`
	InputDescription  string `json:"inputDescription"`
	ReferenceImageURL string `json:"referenceImageUrl"` // Start keyframe image
	EndFrameURL       string `json:"endFrameUrl"`       // End keyframe image for video transitions
}

// VideoSegmentInfo represents a video segment for HLS playlist generation
type VideoSegmentInfo struct {
	Index        int    `json:"index"`
	VideoURL     string `json:"videoUrl"`
	StartFrame   string `json:"startFrame"`
	EndFrame     string `json:"endFrame"`
	DurationSecs int    `json:"durationSecs"`
}

// VideoGenerationInfo represents video generation data for HLS playlist
type VideoGenerationInfo struct {
	SceneID           string             `json:"sceneId"`
	GeneratedVideoURL string             `json:"generatedVideoUrl"`
	Duration          int                `json:"duration"`
	IsSubdivided      bool               `json:"isSubdivided"`
	VideoSegments     []VideoSegmentInfo `json:"videoSegments,omitempty"`
}

type GenerationErrorCode string

const (
	GenerationErrorTimeout       GenerationErrorCode = "provider_timeout"
	GenerationErrorQuota         GenerationErrorCode = "provider_quota"
	GenerationErrorInvalidPrompt GenerationErrorCode = "invalid_prompt"
	GenerationErrorNetwork       GenerationErrorCode = "network_error"
	GenerationErrorProvider      GenerationErrorCode = "provider_error"
	GenerationErrorUnknown       GenerationErrorCode = "unknown_error"
	GenerationErrorCancelled     GenerationErrorCode = "cancelled"
)

func classifyGenerationError(err error, fallback GenerationErrorCode) GenerationErrorCode {
	if err == nil {
		return fallback
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline exceeded"):
		return GenerationErrorTimeout
	case strings.Contains(msg, "quota"), strings.Contains(msg, "rate limit"), strings.Contains(msg, "429"):
		return GenerationErrorQuota
	case strings.Contains(msg, "prompt"), strings.Contains(msg, "invalid argument"), strings.Contains(msg, "malformed"):
		return GenerationErrorInvalidPrompt
	case strings.Contains(msg, "network"), strings.Contains(msg, "connection refused"), strings.Contains(msg, "connection reset"), strings.Contains(msg, "temporary failure"):
		return GenerationErrorNetwork
	case strings.Contains(msg, "provider"), strings.Contains(msg, "upstream"), strings.Contains(msg, "api error"):
		return GenerationErrorProvider
	default:
		return fallback
	}
}

func formatGenerationError(code GenerationErrorCode, message string) string {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "generation failed"
	}
	return fmt.Sprintf("[%s] %s", code, msg)
}

// StartContentGeneration starts the AI content generation process (Step 1)
func (s *Service) StartContentGeneration(ctx context.Context, req *ContentGenerationRequest) (*domain.StoryboardContentGeneration, error) {
	s.logger.Info("starting content generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("rawInput", truncateForLog(req.RawInput, 200)),
		zap.Int("characterCount", len(req.CharacterIDs)),
		zap.Int("sceneCount", len(req.SceneIDs)),
		zap.String("style", req.Style))

	// Verify storyboard exists
	storyboard, err := s.repo.StoryboardByID(ctx, req.StoryboardID)
	if err != nil {
		s.logger.Error("storyboard not found for content generation",
			zap.String("storyboardId", req.StoryboardID),
			zap.Error(err))
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

	s.logger.Debug("storyboard verified for content generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("storyId", storyboard.StoryID),
		zap.String("title", storyboard.Title))

	// Create generation record
	gen := &domain.StoryboardContentGeneration{
		ID:           uuid.New().String(),
		StoryboardID: req.StoryboardID,
		RawInput:     req.RawInput,
		CharacterIDs: req.CharacterIDs,
		SceneIDs:     req.SceneIDs,
		Style:        req.Style,
		Status:       domain.GenerationStatusPending,
		CreatedAt:    time.Now().Unix(),
	}

	s.logger.Debug("creating content generation record",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("status", string(gen.Status)))

	if err := s.repo.CreateContentGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to create content generation record",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", req.StoryboardID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create content generation record: %w", err)
	}

	s.logger.Info("content generation record created",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID))

	// Update storyboard workflow status
	if err := s.repo.UpdateStoryboardWorkflow(ctx, req.StoryboardID, domain.WorkflowStatusDraft, 2); err != nil {
		s.logger.Warn("failed to update storyboard workflow",
			zap.String("storyboardId", req.StoryboardID),
			zap.Error(err))
	} else {
		s.logger.Debug("storyboard workflow status updated",
			zap.String("storyboardId", req.StoryboardID),
			zap.String("workflowStatus", string(domain.WorkflowStatusDraft)),
			zap.Int("currentStep", 2))
	}

	// Start async AI generation
	s.logger.Info("starting async content generation process",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID))
	go s.processContentGeneration(context.Background(), gen, storyboard)

	return gen, nil
}

// processContentGeneration processes the content generation in background
func (s *Service) processContentGeneration(ctx context.Context, gen *domain.StoryboardContentGeneration, storyboard *domain.Storyboard) {
	startTime := time.Now()
	s.logger.Info("processing content generation",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("rawInput", truncateForLog(gen.RawInput, 200)),
		zap.String("style", gen.Style))

	// Record metrics: pending -> processing
	if s.metrics != nil {
		s.metrics.RecordStoryboardContentGeneration("processing", 0)
	}

	// Update status to processing
	gen.Status = domain.GenerationStatusProcessing
	if err := s.repo.UpdateContentGeneration(ctx, gen); err != nil {
		s.logger.Warn("failed to update content generation status to processing",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("content generation status updated to processing",
			zap.String("generationId", gen.ID))
	}

	// Build context from characters and scenes
	s.logger.Debug("building generation context",
		zap.String("generationId", gen.ID),
		zap.Int("characterCount", len(gen.CharacterIDs)),
		zap.Int("sceneCount", len(gen.SceneIDs)))
	contextStr := s.buildGenerationContext(ctx, storyboard, gen.CharacterIDs, gen.SceneIDs)
	s.logger.Debug("generation context built",
		zap.String("generationId", gen.ID),
		zap.Int("contextLength", len(contextStr)))

	// Determine style: use provided style, or fallback to story's genre or style
	style := gen.Style
	if style == "" {
		s.logger.Debug("style not provided, attempting to get from story",
			zap.String("generationId", gen.ID),
			zap.String("storyId", storyboard.StoryID))
		// Try to get story's genre or style as the default style
		story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
		if err == nil {
			if story.Genre != "" {
				style = story.Genre
				s.logger.Debug("using story genre as style",
					zap.String("generationId", gen.ID),
					zap.String("style", style))
			} else if story.Style != nil && story.Style.Style != "" {
				style = story.Style.Style
				s.logger.Debug("using story style as style",
					zap.String("generationId", gen.ID),
					zap.String("style", style))
			} else {
				style = "drama" // Ultimate fallback
				s.logger.Debug("using default style fallback",
					zap.String("generationId", gen.ID),
					zap.String("style", style))
			}
		} else {
			style = "drama" // Ultimate fallback
			s.logger.Warn("failed to get story for style fallback, using default",
				zap.String("generationId", gen.ID),
				zap.String("storyId", storyboard.StoryID),
				zap.String("style", style),
				zap.Error(err))
		}
	} else {
		s.logger.Debug("using provided style",
			zap.String("generationId", gen.ID),
			zap.String("style", style))
	}

	// Generate content using AI — 火山优先，失败再 Gemini（重试/继续走同一逻辑）
	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	geminiOK := s.geminiClient != nil
	if !huoshanOK && !geminiOK {
		s.logger.Error("no AI client available for content generation",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", gen.StoryboardID))
		gen.Status = domain.GenerationStatusFailed
		gen.ErrorMessage = formatGenerationError(GenerationErrorProvider, "storyboard AI generation is not configured")
		_ = s.repo.UpdateContentGeneration(ctx, gen)
		return
	} else {
		prompt := renderPromptDSL(PromptDSL{
			Role:         "You are a creative story writer and manga/webtoon visual-story editor.",
			Task:         "Generate an engaging story chapter in narrative prose that can be converted into structured comic panels.",
			Inputs:       map[string]any{"context": contextStr, "userInput": gen.RawInput, "style": style},
			GlobalConfig: structuredStoryPanelGuidance(),
			Sections: []PromptDSLSection{
				{Title: "Rules", Kind: "text", Body: `
- Generate final narrative prose only; do not output JSON or markdown because the next pipeline step consumes this as story text.
- Before writing, internally choose a compact story spine: protagonist/subject, desire, obstacle, attempt, cost, and final hook. Do not list the spine, but make it visible through the prose.
- Expand plot logically from user input: each beat should change situation, knowledge, or stakes; avoid static mood paragraphs with no causal thread.
- Protagonists may be human, animal, anthropomorphized objects, or static-object personification—express goals/fears through observable action, not abstract labels.
- If the subject is an object or static scene, give it a visual agency path (tilt, roll, crack, drift, chase a light patch, resist being thrown away) instead of converting it into a generic human drama.
- Write with visual beats that can become panels: clear actions, readable blocking, concrete props, and spatial transitions.
- Prefer small but consequential expansions rooted in the input (a missing button, a late train, a cracked cup, a cat guarding a bowl) over unrelated epic lore.
- Leave room for gutter/closure: do not over-explain every action between two dramatic moments.
- If the user input contains fighting, impact, shouting, smashing, falling, chasing, fear, climax, or explosion-like verbs, stage at least one beat that can later trigger high contrast manga impact framing.
- Keep tone consistent with the requested style.`},
			},
		})

		s.logger.Debug("calling storyboard content text LLM (huoshan then gemini)",
			zap.String("generationId", gen.ID),
			zap.Int("promptLength", len(prompt)))

		text, inTok, outTok, totTok, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, prompt, "content", 8192, 0.7, false, 0.7, 8192)
		if err != nil {
			s.logger.Error("AI content generation failed",
				zap.String("generationId", gen.ID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("lastProvider", prov),
				zap.Error(err))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(classifyGenerationError(err, GenerationErrorUnknown), err.Error())
			if updateErr := s.repo.UpdateContentGeneration(ctx, gen); updateErr != nil {
				s.logger.Error("failed to update content generation status to failed",
					zap.String("generationId", gen.ID),
					zap.Error(updateErr))
			}
			s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "content", prov, prompt, "", 0, 0, domain.AITaskStatusFailed, err.Error())
			if s.metrics != nil {
				duration := time.Since(startTime)
				s.metrics.RecordStoryboardContentGeneration("failed", duration)
				metricProv := prov
				if metricProv == "" {
					metricProv = "text_llm"
				}
				s.metrics.RecordAIGenerationRetry("content", metricProv, 0)
			}
			return
		}

		gen.GeneratedContent = text
		gen.InputTokens = inTok
		gen.OutputTokens = outTok
		gen.TotalTokens = totTok
		if totTok > 0 {
			s.logger.Debug("content generation token usage recorded",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov),
				zap.Int("inputTokens", gen.InputTokens),
				zap.Int("outputTokens", gen.OutputTokens),
				zap.Int("totalTokens", gen.TotalTokens))
		} else {
			s.logger.Warn("no token usage recorded for content generation",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov))
		}

		s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "content", prov, prompt, text, gen.InputTokens, gen.OutputTokens, domain.AITaskStatusCompleted, "")

		s.logger.Info("content generated successfully",
			zap.String("generationId", gen.ID),
			zap.String("provider", prov),
			zap.Int("contentLength", len(gen.GeneratedContent)),
			zap.Int("totalTokens", gen.TotalTokens))
	}

	// Update status to completed
	gen.Status = domain.GenerationStatusCompleted
	now := time.Now().Unix()
	gen.CompletedAt = &now
	if err := s.repo.UpdateContentGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to update content generation status to completed",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("content generation status updated to completed",
			zap.String("generationId", gen.ID))
	}

	// Update storyboard content and token consumption
	storyboard.Content = gen.GeneratedContent
	storyboard.IsAIGenerated = true
	if err := s.repo.UpdateStoryboard(ctx, storyboard); err != nil {
		s.logger.Error("failed to update storyboard content",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.Error(err))
	} else {
		s.logger.Debug("storyboard content updated",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.Int("contentLength", len(storyboard.Content)))
	}

	// Aggregate and update token consumption
	s.logger.Debug("updating storyboard token consumption",
		zap.String("storyboardId", gen.StoryboardID))
	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	// Update workflow status
	if err := s.repo.UpdateStoryboardWorkflow(ctx, gen.StoryboardID, domain.WorkflowStatusContentReady, 2); err != nil {
		s.logger.Warn("failed to update storyboard workflow to content ready",
			zap.String("storyboardId", gen.StoryboardID),
			zap.Error(err))
	} else {
		s.logger.Debug("storyboard workflow updated to content ready",
			zap.String("storyboardId", gen.StoryboardID))
		// Record metrics: workflow milestone
		if s.metrics != nil {
			// Duration is tracked separately via workflow duration metric
			s.metrics.RecordStoryboardWorkflowCompleted("content_ready", 0)
		}
	}

	// Record metrics: completed
	if s.metrics != nil {
		duration := time.Since(startTime)
		s.metrics.RecordStoryboardContentGeneration("completed", duration)
		// Record AI generation
		s.metrics.RecordAIGeneration("gemini", "content")
		if gen.TotalTokens > 0 {
			s.metrics.RecordStoryboardTokenConsumed(gen.StoryboardID, float64(gen.TotalTokens))
		}
	}

	s.logger.Info("content generation completed successfully",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.Int("tokens", gen.TotalTokens),
		zap.Int("contentLength", len(gen.GeneratedContent)))
}

// GenerateSceneDetails generates detailed scene descriptions (Step 2)
func (s *Service) GenerateSceneDetails(ctx context.Context, req *SceneDetailRequest) (*domain.StoryboardSceneGeneration, error) {
	s.logger.Info("starting scene detail generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("sceneId", req.SceneID),
		zap.String("sceneTitle", req.SceneTitle),
		zap.String("sceneLocation", req.SceneLocation),
		zap.String("inputDescription", truncateForLog(req.InputDescription, 200)))

	// Verify storyboard exists
	storyboard, err := s.repo.StoryboardByID(ctx, req.StoryboardID)
	if err != nil {
		s.logger.Error("storyboard not found for scene detail generation",
			zap.String("storyboardId", req.StoryboardID),
			zap.String("sceneId", req.SceneID),
			zap.Error(err))
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

	s.logger.Debug("storyboard verified for scene detail generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("storyId", storyboard.StoryID))

	// Create generation record
	gen := &domain.StoryboardSceneGeneration{
		ID:               uuid.New().String(),
		StoryboardID:     req.StoryboardID,
		SceneID:          req.SceneID,
		SceneTitle:       req.SceneTitle,
		SceneLocation:    req.SceneLocation,
		InputDescription: req.InputDescription,
		Status:           domain.GenerationStatusPending,
		CreatedAt:        time.Now().Unix(),
	}

	s.logger.Debug("creating scene generation record",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID))

	if err := s.repo.CreateSceneGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to create scene generation record",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", req.StoryboardID),
			zap.String("sceneId", req.SceneID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create scene generation record: %w", err)
	}

	s.logger.Info("scene generation record created",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID))

	// Start async AI generation
	s.logger.Info("starting async scene detail generation process",
		zap.String("generationId", gen.ID),
		zap.String("sceneId", gen.SceneID))
	go s.processSceneGeneration(context.Background(), gen)

	return gen, nil
}

// processSceneGeneration processes scene detail generation in background
func (s *Service) processSceneGeneration(ctx context.Context, gen *domain.StoryboardSceneGeneration) {
	startTime := time.Now()
	s.logger.Info("processing scene detail generation",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.String("sceneTitle", gen.SceneTitle),
		zap.String("sceneLocation", gen.SceneLocation))

	// Record metrics: pending -> processing
	if s.metrics != nil {
		s.metrics.RecordStoryboardSceneGeneration("processing", 0)
	}

	gen.Status = domain.GenerationStatusProcessing
	if err := s.repo.UpdateSceneGeneration(ctx, gen); err != nil {
		s.logger.Warn("failed to update scene generation status to processing",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("scene generation status updated to processing",
			zap.String("generationId", gen.ID))
	}

	// 分镜扩写：火山优先，失败再 Gemini（重试同路径）
	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	geminiOK := s.geminiClient != nil
	if !huoshanOK && !geminiOK {
		s.logger.Warn("no AI client available, using input description as generated detail",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID))
		gen.GeneratedDetail = gen.InputDescription
	} else {
		prompt := fmt.Sprintf(`Transform the following scene description into a highly detailed, cinematic script ideal for high-end text-to-video or text-to-image AI generators:

Scene Title: %s
Location: %s
Original Description: %s

You must enrich the scene using the following exhaustive dimensions:
1. **Camera & Cinematography**: Specify shot size (Extreme Close-Up, Wide Shot, Medium Full Shot), Camera Angle (Low angle, High angle, Dutch angle), Camera Movement (Slow push-in, Dolly track, Handheld panning, Drone shot), and Lens type (e.g. 50mm, anamorphic lens, telephoto, macro).
2. **Subject & Performance**: Detail micro-expressions (e.g. slight furrowed brows, tear welling up), Body language & Posture, specific eye direction/contact, and dynamic action/interaction.
3. **Wardrobe & Textures**: Describe fabric types (silk, distressed leather, heavy wool, etc.), clothing layering, flowing physics (e.g. cape blowing in the wind), and specific accessories.
4. **Lighting Setup**: Define the lighting strictly (e.g. Key light, volumetric god rays, rim light, neon glow, chiaroscuro, bounce light, harsh shadows, soft cinematic lighting).
5. **Environment & Atmosphere**: Include atmospheric effects (fog, dust motes, rain, atmospheric perspective), depth of field (shallow/deep focus), specific time of day nuances, and weather dynamics.

Return a cohesive, extremely vivid and descriptive narrative paragraph. Do not use bullet points in the final output.`, gen.SceneTitle, gen.SceneLocation, gen.InputDescription)

		s.logger.Debug("calling storyboard scene detail text LLM (huoshan then gemini)",
			zap.String("generationId", gen.ID),
			zap.Int("promptLength", len(prompt)))

		text, inTok, outTok, totTok, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, prompt, "scene", 8192, 0.5, false, 0.5, 8192)
		if err != nil {
			s.logger.Error("AI scene detail generation failed",
				zap.String("generationId", gen.ID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("sceneId", gen.SceneID),
				zap.Error(err))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(classifyGenerationError(err, GenerationErrorUnknown), err.Error())
			if updateErr := s.repo.UpdateSceneGeneration(ctx, gen); updateErr != nil {
				s.logger.Error("failed to update scene generation status to failed",
					zap.String("generationId", gen.ID),
					zap.Error(updateErr))
			}
			s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "scene", prov, prompt, "", 0, 0, domain.AITaskStatusFailed, err.Error())
			if s.metrics != nil {
				duration := time.Since(startTime)
				s.metrics.RecordStoryboardSceneGeneration("failed", duration)
			}
			return
		}

		gen.GeneratedDetail = text
		gen.InputTokens = inTok
		gen.OutputTokens = outTok
		gen.TotalTokens = totTok
		if totTok > 0 {
			s.logger.Debug("scene generation token usage recorded",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov),
				zap.Int("inputTokens", gen.InputTokens),
				zap.Int("outputTokens", gen.OutputTokens),
				zap.Int("totalTokens", gen.TotalTokens))
		} else {
			s.logger.Warn("no token usage recorded for scene generation",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov))
		}

		s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "scene", prov, prompt, text, gen.InputTokens, gen.OutputTokens, domain.AITaskStatusCompleted, "")

		s.logger.Info("scene details generated successfully",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("provider", prov),
			zap.Int("detailLength", len(gen.GeneratedDetail)),
			zap.Int("totalTokens", gen.TotalTokens))
	}

	gen.Status = domain.GenerationStatusCompleted
	now := time.Now().Unix()
	gen.CompletedAt = &now
	if err := s.repo.UpdateSceneGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to update scene generation status to completed",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("scene generation status updated to completed",
			zap.String("generationId", gen.ID))
	}

	s.logger.Debug("updating storyboard token consumption",
		zap.String("storyboardId", gen.StoryboardID))
	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	// Record metrics: completed
	if s.metrics != nil {
		duration := time.Since(startTime)
		s.metrics.RecordStoryboardSceneGeneration("completed", duration)
		// Record AI generation
		s.metrics.RecordAIGeneration("gemini", "scene")
		if gen.TotalTokens > 0 {
			s.metrics.RecordStoryboardTokenConsumed(gen.StoryboardID, float64(gen.TotalTokens))
		}
	}

	s.logger.Info("scene generation completed successfully",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.Int("totalTokens", gen.TotalTokens))
}

// GenerateSceneImage generates an image for a scene (Step 3)
func (s *Service) GenerateSceneImage(ctx context.Context, req *ImageGenerationRequest) (*domain.StoryboardImageGeneration, error) {
	s.logger.Info("starting scene image generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("sceneId", req.SceneID),
		zap.String("sceneTitle", req.SceneTitle),
		zap.String("sceneDescription", truncateForLog(req.SceneDescription, 200)),
		zap.Int("referenceImageCount", len(req.ReferenceImages)),
		zap.Int("sceneCharacterCount", len(req.SceneCharacters)),
		zap.Int("characterReferenceImageCount", len(req.CharacterReferenceImages)))

	// Verify storyboard exists
	storyboard, err := s.repo.StoryboardByID(ctx, req.StoryboardID)
	if err != nil {
		s.logger.Error("storyboard not found for image generation",
			zap.String("storyboardId", req.StoryboardID),
			zap.String("sceneId", req.SceneID),
			zap.Error(err))
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

	s.logger.Debug("storyboard verified for image generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("storyId", storyboard.StoryID))

	plannedScene := s.lookupStoryboardSceneForComicPage(ctx, req.StoryboardID, req.SceneID)
	mergedSceneDescription := s.MergedStoryboardSceneDescriptionForImage(ctx, req.StoryboardID, req.SceneID, req.SceneDescription)
	mergedSceneDescription = mergeStoryboardPlannedSceneForImage(mergedSceneDescription, plannedScene)

	// 获取故事信息和风格配置
	var storyStyle *domain.StyleConfig
	if req.StoryStyle != nil {
		storyStyle = req.StoryStyle
		s.logger.Debug("using provided story style",
			zap.String("style", storyStyle.Style),
			zap.String("description", storyStyle.Description))
	} else if storyboard.StoryID != "" {
		// 尝试从故事获取风格配置
		story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
		if err == nil && story.Style != nil {
			storyStyle = story.Style
			s.logger.Debug("fetched story style from story",
				zap.String("storyId", storyboard.StoryID),
				zap.String("style", storyStyle.Style))
		}
	}

	// 获取场景关联的角色图片
	characterRefImages := req.CharacterReferenceImages
	if len(characterRefImages) == 0 && len(req.SceneCharacters) > 0 && storyboard.StoryID != "" {
		// 如果没有提供角色参考图片，但有角色名称，尝试从故事中获取角色图片
		characterRefImages = s.getCharacterImagesForScene(ctx, storyboard.StoryID, req.SceneCharacters)
		s.logger.Debug("fetched character reference images",
			zap.String("storyId", storyboard.StoryID),
			zap.Strings("sceneCharacters", req.SceneCharacters),
			zap.Int("imageCount", len(characterRefImages)))
	}

	// 判断是否为过渡场景（没有角色出现）
	isTransitionScene := len(req.SceneCharacters) == 0 && len(characterRefImages) == 0

	panelURL := strings.TrimSpace(s.previousStoryboardScenePanelImageURL(ctx, req.StoryboardID, req.SceneID))

	// Build a typed reference manifest. URL ordering and prompt semantics now
	// come from the same source of truth instead of positional guesses.
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

	// Create generation record
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
		PipelineKind:             domain.StoryboardImagePipelineScene,
		SkipPeerFailureGate:      req.SkipPeerFailureGate,
		PlannedScene:             plannedScene,
		ReferenceManifest:        referenceManifest,
		ContentLanguage:          s.storyboardSceneContentLanguage(ctx, plannedScene, req.SceneTitle+"\n"+mergedSceneDescription),
		Status:                   domain.GenerationStatusPending,
		CreatedAt:                time.Now().Unix(),
	}

	s.logger.Debug("creating image generation record",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.Bool("isTransitionScene", isTransitionScene),
		zap.Int("totalReferenceImages", len(allReferenceImages)))

	if err := s.repo.CreateImageGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to create image generation record",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", req.StoryboardID),
			zap.String("sceneId", req.SceneID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create image generation record: %w", err)
	}

	s.logger.Info("image generation record created",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.Bool("hasStoryStyle", storyStyle != nil),
		zap.Bool("isTransitionScene", isTransitionScene))

	// Start async generation
	s.logger.Info("starting async image generation process",
		zap.String("generationId", gen.ID),
		zap.String("sceneId", gen.SceneID))
	go s.processImageGeneration(context.Background(), gen, req.AspectRatio)

	return gen, nil
}

// getCharacterImagesForScene 根据场景中的角色名称获取角色图片
// 限制最多5个主要角色，超过的角色将被忽略
func (s *Service) getCharacterImagesForScene(ctx context.Context, storyID string, characterNames []string) []string {
	if len(characterNames) == 0 {
		return nil
	}

	// 限制主要角色数量最多5个
	maxMainCharacters := 5
	mainCharacterNames := characterNames
	if len(mainCharacterNames) > maxMainCharacters {
		mainCharacterNames = mainCharacterNames[:maxMainCharacters]
		s.logger.Debug("limiting main characters to 5",
			zap.String("storyId", storyID),
			zap.Int("originalCount", len(characterNames)),
			zap.Int("limitedCount", len(mainCharacterNames)))
	}

	// 获取故事的所有角色
	characters, err := s.repo.CharactersByStory(ctx, storyID)
	if err != nil {
		s.logger.Warn("failed to fetch characters for story",
			zap.String("storyId", storyID),
			zap.Error(err))
		return nil
	}

	// 创建角色名称到角色的映射
	charMap := make(map[string]*domain.Character)
	for _, char := range characters {
		charMap[char.Name] = char
	}

	// 收集匹配角色的图片（仅主要角色，最多5个）
	// 只使用Portrait，如果没有Portrait则跳过该角色（该角色只会在文本中描述，不参与图片生成）
	var images []string
	for _, name := range mainCharacterNames {
		if char, ok := charMap[name]; ok {
			// 只使用Portrait（完整角色形象图），如果没有Portrait则跳过该角色
			if char.Portrait != "" {
				images = append(images, char.Portrait)
			} else {
				s.logger.Debug("character has no portrait, skipping from image generation",
					zap.String("storyId", storyID),
					zap.String("characterName", name),
					zap.String("characterId", char.ID))
			}
		}
	}

	return images
}

// latestStoryboardImageGenerationByScene keeps the newest row per scene (ties broken by ID).
func latestStoryboardImageGenerationByScene(list []*domain.StoryboardImageGeneration) map[string]*domain.StoryboardImageGeneration {
	out := make(map[string]*domain.StoryboardImageGeneration)
	for _, g := range list {
		if g == nil || g.SceneID == "" {
			continue
		}
		cur, ok := out[g.SceneID]
		if !ok || g.CreatedAt > cur.CreatedAt || (g.CreatedAt == cur.CreatedAt && g.ID > cur.ID) {
			out[g.SceneID] = g
		}
	}
	return out
}

// storyboardPeerSceneHasLatestFailedImageGen is true when another scene's latest image attempt is failed
// (or completed without URL). Used to stop the rest of a parallel batch when one panel fails.
func (s *Service) storyboardPeerSceneHasLatestFailedImageGen(ctx context.Context, storyboardID, excludeSceneID string) bool {
	if s.repo == nil {
		return false
	}
	list, err := s.repo.ListImageGenerations(ctx, storyboardID)
	if err != nil || len(list) == 0 {
		return false
	}
	byScene := latestStoryboardImageGenerationByScene(list)
	for sid, g := range byScene {
		if sid == excludeSceneID {
			continue
		}
		if g.Status == domain.GenerationStatusFailed {
			return true
		}
		if g.Status == domain.GenerationStatusCompleted && strings.TrimSpace(g.GeneratedImageURL) == "" {
			return true
		}
	}
	return false
}

// storyboardImagePhaseHasLatestFailure is true if any scene's latest image generation is in a failed terminal state.
func (s *Service) storyboardImagePhaseHasLatestFailure(ctx context.Context, storyboardID string) bool {
	if s.repo == nil {
		return false
	}
	list, err := s.repo.ListImageGenerations(ctx, storyboardID)
	if err != nil {
		return false
	}
	for _, g := range latestStoryboardImageGenerationByScene(list) {
		if g.Status == domain.GenerationStatusFailed {
			return true
		}
		if g.Status == domain.GenerationStatusCompleted && strings.TrimSpace(g.GeneratedImageURL) == "" {
			return true
		}
	}
	return false
}

// cancelInFlightSiblingStoryboardImageGenerations marks pending/processing image jobs (other than exceptGenID) failed
// when one panel fails in the same storyboard batch. Skipped for user-driven retries (SkipPeerFailureGate on the failing gen)
// so parallel retry jobs are not torn down when one scene fails.
// so parallel work stops after one panel fails in the same storyboard batch.
func (s *Service) cancelInFlightSiblingStoryboardImageGenerations(ctx context.Context, storyboardID, exceptGenID string) int {
	if s.repo == nil {
		return 0
	}
	list, err := s.repo.ListImageGenerations(ctx, storyboardID)
	if err != nil {
		return 0
	}
	msg := formatGenerationError(GenerationErrorCancelled, "stopped because another panel image failed in the same batch")
	now := time.Now().Unix()
	cancelled := 0
	for _, g := range list {
		if g == nil || g.ID == exceptGenID {
			continue
		}
		if g.Status != domain.GenerationStatusPending && g.Status != domain.GenerationStatusProcessing {
			continue
		}
		g.Status = domain.GenerationStatusFailed
		g.ErrorMessage = msg
		g.CompletedAt = &now
		if err := s.repo.UpdateImageGeneration(ctx, g); err != nil {
			s.logger.Warn("failed to cascade-cancel sibling image generation",
				zap.String("storyboardId", storyboardID),
				zap.String("generationId", g.ID),
				zap.Error(err))
			continue
		}
		cancelled++
	}
	if cancelled > 0 {
		s.logger.Info("cascade-cancelled in-flight sibling image generations after a panel failure",
			zap.String("storyboardId", storyboardID),
			zap.String("exceptGenerationId", exceptGenID),
			zap.Int("cancelledCount", cancelled))
	}
	return cancelled
}

// processImageGeneration processes image generation in background
func (s *Service) processImageGeneration(ctx context.Context, gen *domain.StoryboardImageGeneration, requestedAspectRatio string) {
	startTime := time.Now()
	s.logger.Info("processing image generation",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.String("sceneTitle", gen.SceneTitle),
		zap.Int("referenceImageCount", len(gen.ReferenceImages)))

	// Determine scene type for metrics
	sceneType := "with_characters"
	characterCount := len(gen.SceneCharacters)
	if gen.IsTransitionScene || characterCount == 0 {
		sceneType = "transition"
	}

	// Record metrics: pending -> processing
	if s.metrics != nil {
		s.metrics.RecordStoryboardImageGeneration("processing", sceneType, 0)
		// Record character references
		if characterCount > 0 {
			s.metrics.RecordImageGenerationCharacterRefs(sceneType, float64(characterCount))
			s.metrics.RecordImageGenerationWithCharacters("processing", characterCount)
		}
		// Record style usage
		hasStyle := gen.StoryStyle != nil
		s.metrics.RecordImageGenerationWithStyle("processing", hasStyle)
		if hasStyle {
			s.metrics.RecordStoryStyleConfigUsage(gen.StoryStyle.ID, "image_generation")
		}
	}

	gen.Status = domain.GenerationStatusProcessing
	if err := s.repo.UpdateImageGeneration(ctx, gen); err != nil {
		s.logger.Warn("failed to update image generation status to processing",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("image generation status updated to processing",
			zap.String("generationId", gen.ID))
	}

	// 同批其它分镜仍显示失败、但属于用户主动「重试失败项」时置为 true，避免误杀新任务。
	// First, generate image prompt using text AI — 火山优先，失败再 Gemini（JSON 提示词）
	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	geminiOK := s.geminiClient != nil
	if compileStoryboardImagePromptFromPlan(s, gen) {
		s.logger.Info("compiled image prompt directly from persisted scene plan",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("promptContract", visualSceneContractVersion))
	} else if huoshanOK || geminiOK {
		s.logger.Info("generating image prompt with AI",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("sceneTitle", gen.SceneTitle),
			zap.Bool("hasStoryStyle", gen.StoryStyle != nil),
			zap.Bool("isTransitionScene", gen.IsTransitionScene),
			zap.Int("characterCount", len(gen.SceneCharacters)))

		promptGen := s.buildImageGenerationPrompt(gen)

		s.logger.Debug("calling storyboard image prompt LLM (huoshan then gemini)",
			zap.String("generationId", gen.ID),
			zap.Int("promptLength", len(promptGen)))

		text, inTok, outTok, totTok, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, promptGen, "image_prompt", 4096, 0.35, true, 0.35, 4096)
		if err != nil {
			s.logger.Error("AI image prompt generation failed",
				zap.String("generationId", gen.ID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("sceneId", gen.SceneID),
				zap.Error(err))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(classifyGenerationError(err, GenerationErrorProvider), err.Error())
			if updateErr := s.repo.UpdateImageGeneration(ctx, gen); updateErr != nil {
				s.logger.Error("failed to update image generation status to failed",
					zap.String("generationId", gen.ID),
					zap.Error(updateErr))
			}
			s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "image_prompt", prov, promptGen, "", 0, 0, domain.AITaskStatusFailed, err.Error())
			if s.metrics != nil {
				duration := time.Since(startTime)
				s.metrics.RecordStoryboardImageGeneration("failed", sceneType, duration)
				s.metrics.RecordImageGenerationError("ai_error")
				if characterCount > 0 {
					s.metrics.RecordImageGenerationWithCharacters("failed", characterCount)
				}
			}
			return
		}

		plannedScene := gen.PlannedScene
		if plannedScene == nil {
			plannedScene = s.lookupStoryboardSceneForComicPage(ctx, gen.StoryboardID, gen.SceneID)
		}
		promptDetails, _ := s.parseImagePromptDetails(text, gen.SceneTitle, gen.SceneDescription)
		if promptDetails != nil {
			applyPlannedStoryboardComicTextsToDetails(promptDetails, plannedScene, gen.ContentLanguage)
			gen.PromptDetails = promptDetails
			gen.GeneratedPrompt = s.combineImagePrompt(promptDetails, gen.SceneTitle, gen.SceneDescription, gen.ContentLanguage)
			s.logger.Debug("structured prompt details parsed successfully",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov),
				zap.String("artStyle", promptDetails.ArtStyle),
				zap.String("mood", promptDetails.Mood))
			if s.metrics != nil {
				s.metrics.RecordImageGenerationPromptDetails(true)
			}
		} else if plannedScene != nil && len(plannedScene.ComicTexts) > 0 {
			fallback := &domain.ImagePromptDetails{
				ComicTexts: normalizeStoryboardComicTextsForLanguage(plannedScene.ComicTexts, gen.ContentLanguage),
			}
			gen.PromptDetails = fallback
			gen.GeneratedPrompt = s.combineImagePrompt(fallback, gen.SceneTitle, gen.SceneDescription, gen.ContentLanguage)
			s.logger.Warn("failed to parse structured prompt; using scene comicTexts only",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov))
			if s.metrics != nil {
				s.metrics.RecordImageGenerationError("parsing_error")
				s.metrics.RecordImageGenerationPromptDetails(false)
			}
		} else {
			nb := storyboardSceneNarrativeBlock(gen.SceneTitle, gen.SceneDescription)
			gen.GeneratedPrompt = prependStoryboardImageNarrativeBlock(nb, capGeminiImageBeautyOutput(strings.TrimSpace(text)))
			s.logger.Warn("failed to parse structured prompt, using raw text",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov),
				zap.String("rawText", truncateForLog(text, 200)))
			if s.metrics != nil {
				s.metrics.RecordImageGenerationError("parsing_error")
				s.metrics.RecordImageGenerationPromptDetails(false)
			}
		}
		gen.InputTokens = inTok
		gen.OutputTokens = outTok
		gen.TotalTokens = totTok
		if totTok > 0 {
			s.logger.Debug("image prompt generation token usage recorded",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov),
				zap.Int("inputTokens", gen.InputTokens),
				zap.Int("outputTokens", gen.OutputTokens),
				zap.Int("totalTokens", gen.TotalTokens))
			if s.metrics != nil {
				s.metrics.RecordImageGenerationTokenConsumed("prompt", sceneType, float64(gen.TotalTokens))
			}
		} else {
			s.logger.Warn("no token usage recorded for image prompt",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov))
		}

		s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "image_prompt", prov, promptGen, text, gen.InputTokens, gen.OutputTokens, domain.AITaskStatusCompleted, "")

		s.logger.Info("image prompt generated successfully",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("provider", prov),
			zap.String("prompt", truncateForLog(gen.GeneratedPrompt, 200)))
	} else {
		// If no AI client, use scene description as prompt
		s.logger.Warn("no AI client available, using scene description as prompt",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID))
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

	// Inject AI-chosen panel-shape composition hint so the image model frames the subject
	// correctly when the app assembles the multi-panel collage cover.
	// We look up the storyboard scene (same lookup used for comicTexts above) to read PanelShape.
	if gen.GeneratedPrompt != "" {
		if ps := s.lookupStoryboardSceneForComicPage(ctx, gen.StoryboardID, gen.SceneID); ps != nil {
			if shape := strings.TrimSpace(ps.PanelShape); shape != "" {
				gen.GeneratedPrompt = appendStoryboardPanelShapeCompositionHint(gen.GeneratedPrompt, shape)
			}
		}
	}

	narrativeRunes := utf8.RuneCountInString(strings.TrimSpace(gen.SceneDescription))
	refURLs, imgOp, refPrimary := selectStoryboardImageRefsAndOperation(gen, narrativeRunes)
	useRefs := len(refURLs) > 0 && imgOp == genapi.OperationImageToImage
	finalPrompt := appendStoryboardImageToImageConstraints(strings.TrimSpace(gen.GeneratedPrompt), useRefs)

	// Generate actual image using genAPI directly
	// TokenUsageRecorder 会自动将 token 消耗和成功/失败记录到 AIGenerationRecord
	if s.genAPI != nil && finalPrompt != "" {
		aspectRatio := domain.NormalizeFragmentAspectRatio(requestedAspectRatio)
		if aspectRatio == "" {
			aspectRatio = "16:9"
		}
		genReq := &genapi.GenerateRequest{
			Prompt:          finalPrompt,
			AspectRatio:     aspectRatio,
			Quality:         "high",
			OutputCount:     1,
			ReferenceImages: refURLs,
			Metadata:        s.usageRecordMetadataForStoryboard(ctx, gen.StoryboardID),
		}

		genReq.Operation = imgOp
		genReq.ReferenceImageURL = refPrimary
		if imgOp == genapi.OperationImageToImage {
			s.logger.Debug("using image-to-image operation",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID),
				zap.String("referenceImageURL", genReq.ReferenceImageURL),
				zap.Int("referenceImageCount", len(refURLs)))
		} else {
			s.logger.Debug("using text-to-image operation",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID),
				zap.Int("narrativeRunes", narrativeRunes))
		}

		imageProvider := s.effectiveImageProvider()

		s.logger.Info("generating scene image",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("provider", imageProvider),
			zap.String("operation", string(genReq.Operation)),
			zap.String("prompt", truncateForLog(finalPrompt, 200)))

		if strings.EqualFold(imageProvider, "huoshan") {
			PrepareHuoshanGenAPIImageRequest(genReq)
		}

		resp, err := s.genAPI.GenerateImage(ctx, imageProvider, genReq)
		if err != nil {
			s.logger.Warn("AI image generation failed, keeping prompt only",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("provider", imageProvider),
				zap.String("operation", string(genReq.Operation)),
				zap.Error(err))
			// Record metrics: image API error
			if s.metrics != nil {
				s.metrics.RecordImageGenerationError("image_api_error")
			}
		} else if resp != nil && len(resp.ImageURLs) > 0 {
			s.logger.Info("image generation API call succeeded",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID),
				zap.String("provider", imageProvider),
				zap.Int("imageCount", len(resp.ImageURLs)))
			// 上传图片到OSS并替换URL
			originalImageURL := resp.ImageURLs[0]
			ossClient := aliyun.GetGlobalClient()
			if ossClient != nil {
				// 生成OSS object key
				objectKey := fmt.Sprintf("storyboard/%s/scenes/%s.jpg", gen.StoryboardID, gen.SceneID)

				// 上传到OSS
				ossURL, err := ossClient.UploadFileFromURL(objectKey, originalImageURL)
				if err != nil {
					s.logger.Warn("failed to upload image to OSS, using original URL",
						zap.String("sceneId", gen.SceneID),
						zap.String("originalURL", originalImageURL),
						zap.Error(err))
					// 上传失败时使用原始URL
					gen.GeneratedImageURL = originalImageURL
				} else {
					// 清理URL：移除查询参数，确保HTTPS
					ossURL = strings.Split(ossURL, "?")[0]
					ossURL = strings.ReplaceAll(ossURL, "http://", "https://")
					gen.GeneratedImageURL = ossURL
					s.logger.Debug("scene image uploaded to OSS",
						zap.String("sceneId", gen.SceneID),
						zap.String("ossURL", ossURL))
				}
			} else {
				s.logger.Warn("OSS client not available, using original image URL",
					zap.String("sceneId", gen.SceneID))
				gen.GeneratedImageURL = originalImageURL
			}

			// Add image generation tokens to total (if available)
			if resp.Usage != nil {
				gen.TotalTokens += resp.Usage.TotalTokens
				// Record metrics: token consumption for image generation
				if s.metrics != nil && resp.Usage.TotalTokens > 0 {
					s.metrics.RecordImageGenerationTokenConsumed("image", sceneType, float64(resp.Usage.TotalTokens))
				}
			}

			s.logger.Info("scene image generated",
				zap.String("sceneId", gen.SceneID),
				zap.String("provider", imageProvider),
				zap.String("imageURL", gen.GeneratedImageURL))
		} else if resp != nil && resp.Metadata != nil {
			s.logger.Debug("checking for image bytes in metadata",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID))
			// Check for image bytes in metadata (for conversational image generation)
			if imageBytes, ok := resp.Metadata["image_bytes"].([][]byte); ok && len(imageBytes) > 0 {
				s.logger.Info("found image bytes in metadata",
					zap.String("generationId", gen.ID),
					zap.String("sceneId", gen.SceneID),
					zap.Int("imageByteCount", len(imageBytes)))
				// Upload image bytes to OSS
				ossClient := aliyun.GetGlobalClient()
				if ossClient != nil {
					objectKey := fmt.Sprintf("storyboard/%s/scenes/%s.png", gen.StoryboardID, gen.SceneID)
					imageURL, uploadErr := ossClient.UploadBytes(objectKey, imageBytes[0])
					if uploadErr != nil {
						s.logger.Warn("failed to upload generated image to OSS",
							zap.String("sceneId", gen.SceneID),
							zap.Error(uploadErr))
					} else {
						// Clean URL: remove query params and ensure HTTPS
						imageURL = strings.Split(imageURL, "?")[0]
						imageURL = strings.ReplaceAll(imageURL, "http://", "https://")
						gen.GeneratedImageURL = imageURL

						s.logger.Info("scene image uploaded to OSS",
							zap.String("sceneId", gen.SceneID),
							zap.String("provider", imageProvider),
							zap.String("imageURL", gen.GeneratedImageURL))
					}
				} else {
					s.logger.Warn("OSS client not available, cannot upload generated image",
						zap.String("sceneId", gen.SceneID))
				}

				// Add image generation tokens to total (if available)
				if resp.Usage != nil {
					gen.TotalTokens += resp.Usage.TotalTokens
				}
			}
		}
	}

	// 并行任务中：若兄弟任务已先将本行标为失败（级联取消），则不再写入成功。
	if latest, err := s.repo.GetImageGeneration(ctx, gen.ID); err == nil && latest != nil && latest.Status == domain.GenerationStatusFailed {
		s.logger.Info("image generation record already failed in DB (cascade), skipping final write",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", gen.StoryboardID))
		return
	}

	if gen.GeneratedImageURL == "" {
		gen.Status = domain.GenerationStatusFailed
		if gen.ErrorMessage == "" {
			gen.ErrorMessage = formatGenerationError(GenerationErrorProvider, "image generation failed: empty generated image url")
		}
	} else {
		gen.Status = domain.GenerationStatusCompleted
		gen.ErrorMessage = ""
	}
	now := time.Now().Unix()
	gen.CompletedAt = &now
	if err := s.repo.UpdateImageGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to update image generation final status",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("image generation final status updated",
			zap.String("generationId", gen.ID),
			zap.String("status", gen.Status))
	}

	s.logger.Debug("updating storyboard token consumption",
		zap.String("storyboardId", gen.StoryboardID))
	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	// Sync generated image URL to storyboard scene
	if gen.GeneratedImageURL != "" && gen.SceneID != "" {
		s.logger.Debug("syncing image URL to storyboard scene",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("imageURL", gen.GeneratedImageURL))
		if err := s.repo.UpdateStoryboardSceneImage(ctx, gen.SceneID, gen.GeneratedImageURL); err != nil {
			s.logger.Warn("failed to sync image to storyboard scene",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID),
				zap.String("imageURL", gen.GeneratedImageURL),
				zap.Error(err))
		} else {
			s.logger.Info("image URL synced to storyboard scene",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID),
				zap.String("imageURL", gen.GeneratedImageURL))
		}
	} else {
		s.logger.Debug("skipping image URL sync - missing image URL or scene ID",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("imageURL", gen.GeneratedImageURL))
	}

	s.afterSceneImageWritten(ctx, gen)

	// Record metrics: completed
	if s.metrics != nil {
		duration := time.Since(startTime)
		status := gen.Status
		s.metrics.RecordStoryboardImageGeneration(status, sceneType, duration)
		if characterCount > 0 {
			s.metrics.RecordImageGenerationWithCharacters(status, characterCount)
		}
		hasStyle := gen.StoryStyle != nil
		s.metrics.RecordImageGenerationWithStyle(status, hasStyle)
		// Record AI generation
		s.metrics.RecordAIGeneration(s.effectiveImageProvider(), "image")
		if gen.TotalTokens > 0 {
			s.metrics.RecordStoryboardTokenConsumed(gen.StoryboardID, float64(gen.TotalTokens))
		}
	}

	s.logger.Info("image generation finished",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.String("status", gen.Status),
		zap.String("errorMessage", gen.ErrorMessage),
		zap.String("imageURL", gen.GeneratedImageURL),
		zap.Int("totalTokens", gen.TotalTokens))
}

// videoSceneHasInFlightJob reports whether there is a pending or processing video generation for the scene.
func videoSceneHasInFlightJob(list []*domain.StoryboardVideoGeneration, sceneID string) bool {
	for _, v := range list {
		if v == nil || v.SceneID != sceneID {
			continue
		}
		if v.Status == domain.GenerationStatusPending || v.Status == domain.GenerationStatusProcessing {
			return true
		}
	}
	return false
}

// hasInFlightVideoGeneration uses persisted video generation rows to avoid duplicate CreateVideoGeneration.
func (s *Service) hasInFlightVideoGeneration(ctx context.Context, storyboardID, sceneID string) bool {
	if s.repo == nil {
		return false
	}
	list, err := s.repo.ListVideoGenerations(ctx, storyboardID)
	if err != nil {
		s.logger.Debug("hasInFlightVideoGeneration: list failed",
			zap.String("storyboardId", storyboardID),
			zap.String("sceneId", sceneID),
			zap.Error(err))
		return false
	}
	return videoSceneHasInFlightJob(list, sceneID)
}

// maybeStartBatchVideoAfterAllImages 在「全部分镜图已写入场景」且故事板开启图后视频时，一次性置 images_ready 并批量排队视频。
func (s *Service) maybeStartBatchVideoAfterAllImages(ctx context.Context, storyboardID string) {
	if s.repo == nil {
		return
	}
	sb, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil || sb == nil || !sb.GenerateVideoAfterImages {
		return
	}
	scenes, err := s.repo.StoryboardScenes(ctx, storyboardID)
	if err != nil || len(scenes) == 0 {
		return
	}
	allHaveImage := true
	for _, sc := range scenes {
		if sc == nil || strings.TrimSpace(sc.Image) == "" {
			allHaveImage = false
			break
		}
	}
	if !allHaveImage {
		return
	}
	videos, err := s.repo.ListVideoGenerations(ctx, storyboardID)
	if err != nil {
		s.logger.Warn("maybeStartBatchVideoAfterAllImages: list video generations failed",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		videos = nil
	}
	if sb.WorkflowStatus != domain.WorkflowStatusImagesReady || sb.CurrentStep < 3 {
		if err := s.repo.UpdateStoryboardWorkflow(ctx, storyboardID, domain.WorkflowStatusImagesReady, 3); err != nil {
			s.logger.Warn("failed to update storyboard workflow to images ready (batch gate)",
				zap.String("storyboardId", storyboardID),
				zap.Error(err))
		} else {
			s.logger.Debug("storyboard workflow updated to images ready (all scene images)",
				zap.String("storyboardId", storyboardID))
			if s.metrics != nil {
				s.metrics.RecordStoryboardWorkflowCompleted("images_ready", 0)
			}
		}
	}
	for _, sc := range scenes {
		if sc == nil {
			continue
		}
		if strings.TrimSpace(sc.Image) == "" {
			continue
		}
		if strings.TrimSpace(sc.VideoUrl) != "" {
			continue
		}
		if videoSceneHasInFlightJob(videos, sc.ID) {
			s.logger.Debug("skip batch video enqueue: in-flight job for scene",
				zap.String("storyboardId", storyboardID),
				zap.String("sceneId", sc.ID))
			continue
		}
		scene := *sc
		sbID := storyboardID
		desc := strings.TrimSpace(scene.Description)
		if desc == "" {
			desc = strings.TrimSpace(scene.Title)
		}
		refURL := strings.TrimSpace(scene.Image)
		s.logger.Info("batch enqueue scene video after all images ready",
			zap.String("storyboardId", sbID),
			zap.String("sceneId", scene.ID))
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					s.logger.Error("panic in batched video generation after all images",
						zap.Any("panic", rec),
						zap.String("storyboardId", sbID),
						zap.String("sceneId", scene.ID))
				}
			}()
			_, vErr := s.GenerateSceneVideo(context.Background(), &VideoGenerationRequest{
				StoryboardID:      sbID,
				SceneID:           scene.ID,
				SceneTitle:        scene.Title,
				InputDescription:  desc,
				ReferenceImageURL: refURL,
			})
			if vErr != nil {
				s.logger.Warn("failed to start batched scene video",
					zap.String("storyboardId", sbID),
					zap.String("sceneId", scene.ID),
					zap.Error(vErr))
			}
		}()
	}
}

// afterSceneImageWritten 在图片生成终态落库且（如有）同步到分镜行之后调用，驱动批量视频门禁与 images_ready。
func (s *Service) afterSceneImageWritten(ctx context.Context, gen *domain.StoryboardImageGeneration) {
	if s.repo == nil || gen == nil {
		return
	}
	s.maybeStartBatchVideoAfterAllImages(ctx, gen.StoryboardID)
	if gen.Status != domain.GenerationStatusCompleted {
		return
	}
	// 仍有任一分镜配图失败（按场景最新一条）或任一分镜尚无成图时，不把 workflow 推到 images_ready，避免「部分失败却显示整板就绪」。
	if s.storyboardImagePhaseHasLatestFailure(ctx, gen.StoryboardID) {
		return
	}
	scenes, err := s.repo.StoryboardScenes(ctx, gen.StoryboardID)
	if err != nil {
		return
	}
	for _, sc := range scenes {
		if sc == nil || strings.TrimSpace(sc.Image) == "" {
			return
		}
	}
	sb, err := s.repo.StoryboardByID(ctx, gen.StoryboardID)
	if err != nil || sb == nil {
		return
	}
	if sb.GenerateVideoAfterImages {
		return
	}
	if err := s.repo.UpdateStoryboardWorkflow(ctx, gen.StoryboardID, domain.WorkflowStatusImagesReady, 3); err != nil {
		s.logger.Warn("failed to update storyboard workflow to images ready",
			zap.String("storyboardId", gen.StoryboardID),
			zap.Error(err))
	} else {
		s.logger.Debug("storyboard workflow updated to images ready",
			zap.String("storyboardId", gen.StoryboardID))
		if s.metrics != nil {
			s.metrics.RecordStoryboardWorkflowCompleted("images_ready", 0)
		}
	}
}

// maybeFinalizeBatchVideoWorkflow 在 GenerateVideoAfterImages 模式下，当每个有图分镜的视频任务均已终态且无进行中的任务时置 video_ready。
func (s *Service) maybeFinalizeBatchVideoWorkflow(ctx context.Context, storyboardID string) {
	if s.repo == nil {
		return
	}
	sb, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil || sb == nil || !sb.GenerateVideoAfterImages {
		return
	}
	if sb.WorkflowStatus == domain.WorkflowStatusVideoReady && sb.CurrentStep >= 4 {
		return
	}
	scenes, err := s.repo.StoryboardScenes(ctx, storyboardID)
	if err != nil || len(scenes) == 0 {
		return
	}
	videos, err := s.repo.ListVideoGenerations(ctx, storyboardID)
	if err != nil {
		s.logger.Warn("maybeFinalizeBatchVideoWorkflow: list video generations failed",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return
	}
	byScene := make(map[string][]*domain.StoryboardVideoGeneration)
	for _, v := range videos {
		if v == nil {
			continue
		}
		byScene[v.SceneID] = append(byScene[v.SceneID], v)
	}
	for _, sc := range scenes {
		if sc == nil || strings.TrimSpace(sc.Image) == "" {
			continue
		}
		if videoSceneHasInFlightJob(videos, sc.ID) {
			return
		}
		if strings.TrimSpace(sc.VideoUrl) != "" {
			continue
		}
		terminal := false
		for _, v := range byScene[sc.ID] {
			if v.Status == domain.GenerationStatusCompleted || v.Status == domain.GenerationStatusFailed {
				terminal = true
				break
			}
		}
		if !terminal {
			return
		}
	}
	if err := s.repo.UpdateStoryboardWorkflow(ctx, storyboardID, domain.WorkflowStatusVideoReady, 4); err != nil {
		s.logger.Warn("failed to update storyboard workflow to video ready (batch finalize)",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	} else {
		s.logger.Debug("storyboard workflow updated to video ready (all batch videos terminal)",
			zap.String("storyboardId", storyboardID))
		if s.metrics != nil {
			s.metrics.RecordStoryboardWorkflowCompleted("video_ready", 0)
		}
	}
}

// GenerateSceneVideo generates a video for a scene (Step 4)
func (s *Service) GenerateSceneVideo(ctx context.Context, req *VideoGenerationRequest) (*domain.StoryboardVideoGeneration, error) {
	s.logger.Info("starting scene video generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("sceneId", req.SceneID),
		zap.String("sceneTitle", req.SceneTitle),
		zap.String("inputDescription", truncateForLog(req.InputDescription, 200)),
		zap.String("referenceImageURL", req.ReferenceImageURL),
		zap.String("endFrameURL", req.EndFrameURL))

	// Verify storyboard exists
	storyboard, err := s.repo.StoryboardByID(ctx, req.StoryboardID)
	if err != nil {
		s.logger.Error("storyboard not found for video generation",
			zap.String("storyboardId", req.StoryboardID),
			zap.String("sceneId", req.SceneID),
			zap.Error(err))
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

	s.logger.Debug("storyboard verified for video generation",
		zap.String("storyboardId", req.StoryboardID),
		zap.String("storyId", storyboard.StoryID))

	// Create generation record
	gen := &domain.StoryboardVideoGeneration{
		ID:                uuid.New().String(),
		StoryboardID:      req.StoryboardID,
		SceneID:           req.SceneID,
		SceneTitle:        req.SceneTitle,
		InputDescription:  req.InputDescription,
		ReferenceImageURL: req.ReferenceImageURL,
		EndFrameURL:       req.EndFrameURL,
		Status:            domain.GenerationStatusPending,
		CreatedAt:         time.Now().Unix(),
	}

	s.logger.Debug("creating video generation record",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID))

	if err := s.repo.CreateVideoGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to create video generation record",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", req.StoryboardID),
			zap.String("sceneId", req.SceneID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create video generation record: %w", err)
	}

	s.logger.Info("video generation record created",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID))

	// Start async generation
	s.logger.Info("starting async video generation process",
		zap.String("generationId", gen.ID),
		zap.String("sceneId", gen.SceneID))
	go s.processVideoGeneration(context.Background(), gen)

	return gen, nil
}

// processVideoGeneration processes video generation in background
func (s *Service) processVideoGeneration(ctx context.Context, gen *domain.StoryboardVideoGeneration) {
	startTime := time.Now()
	s.logger.Info("starting video generation process",
		zap.String("sceneId", gen.SceneID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneTitle", gen.SceneTitle),
		zap.String("referenceImageURL", gen.ReferenceImageURL),
		zap.String("endFrameURL", gen.EndFrameURL),
		zap.String("inputDescription", gen.InputDescription))

	// Record metrics: pending -> processing
	if s.metrics != nil {
		s.metrics.RecordStoryboardVideoGeneration("processing", false, 0)
	}

	gen.Status = domain.GenerationStatusProcessing
	if err := s.repo.UpdateVideoGeneration(ctx, gen); err != nil {
		s.logger.Warn("failed to update video generation status to processing",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("video generation status updated to processing",
			zap.String("generationId", gen.ID))
	}

	// Generate video prompt using text AI — 火山优先，失败再 Gemini（JSON）
	huoshanOK := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	geminiOK := s.geminiClient != nil
	if huoshanOK || geminiOK {
		promptGen := fmt.Sprintf(`Create a video generation prompt for the following scene:

Scene Title: %s
Scene Description: %s

Please return a structured JSON object (without markdown code blocks) with the following format:
{
  "cameraMovement": "摄像机运动，如 slow pan left, zoom in, static, dolly forward",
  "subjectMotion": "主体动作描述，如 walking slowly, turning around, gesturing",
  "atmosphere": "氛围描述，如 peaceful morning, tense night, mysterious fog",
  "transitionStyle": "转场风格，如 fade, cut, crossfade",
  "duration": "时长描述，如 5 seconds",
  "keyMoments": ["关键时刻1", "关键时刻2"],
  "additionalNotes": "其他补充说明（可选）"
}

Important: Return ONLY the JSON object, no explanations or markdown formatting.
LENGTH: When merged into one video prompt, keep the total substantive Chinese text (汉字) within 200 characters — be concise.`, gen.SceneTitle, gen.InputDescription)

		s.logger.Info("generating video prompt with AI",
			zap.String("sceneId", gen.SceneID),
			zap.String("sceneTitle", gen.SceneTitle),
			zap.String("inputDescription", gen.InputDescription))

		text, inTok, outTok, totTok, prov, err := s.storyboardLLMTextHuoshanThenGemini(ctx, promptGen, "video_prompt", 2048, 0.35, true, 0.35, 2048)
		if err != nil {
			s.logger.Error("failed to generate video prompt",
				zap.String("sceneId", gen.SceneID),
				zap.String("provider", prov),
				zap.Error(err))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(classifyGenerationError(err, GenerationErrorProvider), err.Error())
			_ = s.repo.UpdateVideoGeneration(ctx, gen)
			s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "video_prompt", prov, promptGen, "", 0, 0, domain.AITaskStatusFailed, err.Error())
			if s.metrics != nil {
				duration := time.Since(startTime)
				s.metrics.RecordStoryboardVideoGeneration("failed", false, duration)
				s.metrics.RecordVideoGenerationError("ai_error")
			}
			s.maybeFinalizeBatchVideoWorkflow(ctx, gen.StoryboardID)
			return
		}

		videoPromptDetails, combinedPrompt := s.parseVideoPromptDetails(text)
		if videoPromptDetails != nil {
			gen.PromptDetails = videoPromptDetails
			gen.GeneratedPrompt = combinedPrompt
			s.logger.Debug("structured video prompt details parsed successfully",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov),
				zap.String("cameraMovement", videoPromptDetails.CameraMovement),
				zap.String("atmosphere", videoPromptDetails.Atmosphere))
		} else {
			gen.GeneratedPrompt = text
			s.logger.Warn("failed to parse structured video prompt, using raw text",
				zap.String("generationId", gen.ID),
				zap.String("provider", prov),
				zap.String("rawText", truncateForLog(text, 200)))
		}
		gen.InputTokens = inTok
		gen.OutputTokens = outTok
		gen.TotalTokens = totTok
		if totTok > 0 && s.metrics != nil {
			s.metrics.RecordVideoGenerationTokenConsumed("prompt", float64(gen.TotalTokens))
		}

		s.recordStoryboardTextGeneration(ctx, gen.StoryboardID, "video_prompt", prov, promptGen, text, gen.InputTokens, gen.OutputTokens, domain.AITaskStatusCompleted, "")

		s.logger.Info("video prompt generated successfully",
			zap.String("sceneId", gen.SceneID),
			zap.String("provider", prov),
			zap.String("generatedPrompt", gen.GeneratedPrompt),
			zap.Int("inputTokens", gen.InputTokens),
			zap.Int("outputTokens", gen.OutputTokens),
			zap.Int("totalTokens", gen.TotalTokens))
	} else {
		// If no AI client, use scene description as prompt
		s.logger.Warn("no AI client available, using input description as video prompt",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("storyboardId", gen.StoryboardID))
		gen.GeneratedPrompt = gen.InputDescription
		s.logger.Info("using input description as video prompt (no AI client)",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("prompt", truncateForLog(gen.GeneratedPrompt, 200)))
	}

	if gen.GeneratedPrompt != "" {
		gen.GeneratedPrompt = truncateStringToMaxRunes(strings.TrimSpace(gen.GeneratedPrompt), maxStoryboardAIGeneratedPromptRunes)
	}

	// Generate actual video using genAPI directly
	// 直接调用 genAPI，将结果记录到 StoryboardVideoGeneration 表，不创建 AIGenerationRecord
	if s.genAPI == nil {
		s.logger.Warn("genAPI not available, cannot generate video",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID))
	} else if gen.GeneratedPrompt == "" {
		s.logger.Warn("generated prompt is empty, cannot generate video",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID))
	} else if s.genAPI != nil && gen.GeneratedPrompt != "" {
		// Log all input parameters
		s.logger.Info("preparing video generation request",
			zap.String("sceneId", gen.SceneID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("referenceImageURL", gen.ReferenceImageURL),
			zap.String("endFrameURL", gen.EndFrameURL),
			zap.String("generatedPrompt", gen.GeneratedPrompt))

		genReq := &genapi.GenerateRequest{
			Prompt:          gen.GeneratedPrompt,
			DurationSeconds: 5, // 5 second video clips
			AspectRatio:     "16:9",
			Metadata:        s.usageRecordMetadataForStoryboard(ctx, gen.StoryboardID),
		}

		videoProvider := s.effectiveVideoProvider()
		s.logger.Debug("using video provider",
			zap.String("generationId", gen.ID),
			zap.String("provider", videoProvider))

		// Choose operation type based on whether reference images are provided
		if gen.ReferenceImageURL != "" && gen.EndFrameURL != "" {
			// Keyframe mode: use subdivision strategy for smooth transitions
			s.logger.Info("using keyframe-to-video mode with subdivision strategy",
				zap.String("sceneId", gen.SceneID),
				zap.String("firstFrameURL", gen.ReferenceImageURL),
				zap.String("lastFrameURL", gen.EndFrameURL))

			// Use AIGenerationService for subdivision if available
			if s.aiGenService != nil {
				subdivResult, err := s.aiGenService.GenerateVideoWithSubdivision(ctx, &genapi.SubdivisionVideoRequest{
					FirstFrameURL:   gen.ReferenceImageURL,
					LastFrameURL:    gen.EndFrameURL,
					Prompt:          gen.GeneratedPrompt,
					DurationSeconds: 5,
					MaxDepth:        3,
					Provider:        videoProvider,
					AspectRatio:     "16:9",
				})
				if err != nil {
					s.logger.Warn("subdivision video generation failed, falling back to direct generation",
						zap.String("sceneId", gen.SceneID),
						zap.Error(err))
					// Fall back to direct keyframe-to-video
				} else if subdivResult != nil && len(subdivResult.Segments) > 0 {
					// Successfully generated with subdivision
					s.logger.Info("subdivision video generation succeeded",
						zap.String("sceneId", gen.SceneID),
						zap.Int("segmentCount", len(subdivResult.Segments)),
						zap.Bool("isSubdivided", subdivResult.IsSubdivided))

					// Use the first segment's video URL as the main video
					gen.GeneratedVideoURL = subdivResult.Segments[0].VideoURL

					// Store subdivision info in generation record
					if subdivResult.IsSubdivided {
						gen.VideoSegmentsJSON = s.serializeVideoSegments(subdivResult.Segments)
						gen.MiddleFrameURLsJSON = s.serializeMiddleFrames(subdivResult.MiddleFrames)
						gen.IsSubdivided = true
					}

					// Record metrics: subdivision video generation
					if s.metrics != nil {
						s.metrics.RecordVideoGenerationSubdivided(true, "processing")
						s.metrics.RecordVideoGenerationSegmentCount(true, float64(len(subdivResult.Segments)))
					}

					// Skip the normal generation flow
					goto updateGeneration
				}
			}

			// Direct keyframe-to-video generation (fallback or no aiGenService)
			genReq.Operation = genapi.OperationKeyframeToVideo
			genReq.FirstFrameURL = gen.ReferenceImageURL
			genReq.LastFrameURL = gen.EndFrameURL
		} else if gen.ReferenceImageURL != "" {
			// Image to video mode: only need ReferenceImageURL
			genReq.Operation = genapi.OperationImageToVideo
			genReq.ReferenceImageURL = gen.ReferenceImageURL
			s.logger.Info("using image-to-video mode",
				zap.String("sceneId", gen.SceneID),
				zap.String("referenceImageURL", genReq.ReferenceImageURL))
		} else {
			// Text to video mode: no images
			genReq.Operation = genapi.OperationTextToVideo
			s.logger.Info("using text-to-video mode",
				zap.String("sceneId", gen.SceneID))
		}

		// Log final request details
		s.logger.Info("generating scene video",
			zap.String("sceneId", gen.SceneID),
			zap.String("provider", videoProvider),
			zap.String("operation", string(genReq.Operation)),
			zap.String("prompt", genReq.Prompt),
			zap.Int("durationSeconds", genReq.DurationSeconds),
			zap.String("aspectRatio", genReq.AspectRatio),
			zap.String("firstFrameURL", genReq.FirstFrameURL),
			zap.String("lastFrameURL", genReq.LastFrameURL),
			zap.String("referenceImageURL", genReq.ReferenceImageURL))

		resp, err := s.genAPI.GenerateVideo(ctx, videoProvider, genReq)
		if err != nil {
			s.logger.Warn("AI video generation failed, keeping prompt only",
				zap.String("sceneId", gen.SceneID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("provider", videoProvider),
				zap.String("operation", string(genReq.Operation)),
				zap.String("firstFrameURL", genReq.FirstFrameURL),
				zap.String("lastFrameURL", genReq.LastFrameURL),
				zap.String("referenceImageURL", genReq.ReferenceImageURL),
				zap.String("prompt", genReq.Prompt),
				zap.Error(err))
			// Record metrics: video API error
			if s.metrics != nil {
				s.metrics.RecordVideoGenerationError("video_api_error")
			}
			// Don't fail completely, just mark as completed without video
		} else if resp != nil {
			s.logger.Info("video generation API call succeeded",
				zap.String("sceneId", gen.SceneID),
				zap.String("provider", videoProvider),
				zap.String("taskId", resp.TaskID),
				zap.String("status", resp.Status),
				zap.String("videoURL", resp.VideoURL),
				zap.String("message", resp.Message))
			if resp.VideoURL != "" {
				s.logger.Info("received video URL from provider",
					zap.String("sceneId", gen.SceneID),
					zap.String("originalVideoURL", resp.VideoURL))

				// 上传视频到OSS并替换URL
				originalVideoURL := resp.VideoURL
				ossClient := aliyun.GetGlobalClient()
				if ossClient != nil {
					// 生成OSS object key
					objectKey := fmt.Sprintf("storyboard/%s/scenes/%s.mp4", gen.StoryboardID, gen.SceneID)
					s.logger.Info("uploading video to OSS",
						zap.String("sceneId", gen.SceneID),
						zap.String("objectKey", objectKey),
						zap.String("sourceURL", originalVideoURL))

					// 上传到OSS
					ossURL, err := ossClient.UploadFileFromURL(objectKey, originalVideoURL)
					if err != nil {
						s.logger.Warn("failed to upload video to OSS, using original URL",
							zap.String("sceneId", gen.SceneID),
							zap.String("storyboardId", gen.StoryboardID),
							zap.String("objectKey", objectKey),
							zap.String("originalURL", originalVideoURL),
							zap.Error(err))
						// 上传失败时使用原始URL
						gen.GeneratedVideoURL = originalVideoURL
					} else {
						// 清理URL：移除查询参数，确保HTTPS
						ossURL = strings.Split(ossURL, "?")[0]
						ossURL = strings.ReplaceAll(ossURL, "http://", "https://")
						gen.GeneratedVideoURL = ossURL
						s.logger.Info("scene video uploaded to OSS successfully",
							zap.String("sceneId", gen.SceneID),
							zap.String("storyboardId", gen.StoryboardID),
							zap.String("objectKey", objectKey),
							zap.String("ossURL", ossURL),
							zap.String("originalURL", originalVideoURL))
					}
				} else {
					s.logger.Warn("OSS client not available, using original video URL",
						zap.String("sceneId", gen.SceneID),
						zap.String("storyboardId", gen.StoryboardID),
						zap.String("originalURL", originalVideoURL))
					gen.GeneratedVideoURL = originalVideoURL
				}

				gen.Duration = 5 // Default duration, may be updated by actual result
				// Add video generation tokens to total (if available)
				if resp.Usage != nil {
					gen.TotalTokens += resp.Usage.TotalTokens
					// Record metrics: token consumption for video generation
					if s.metrics != nil && resp.Usage.TotalTokens > 0 {
						s.metrics.RecordVideoGenerationTokenConsumed("video", float64(resp.Usage.TotalTokens))
					}
				}

				s.logger.Info("scene video generated",
					zap.String("sceneId", gen.SceneID),
					zap.String("provider", videoProvider),
					zap.String("videoURL", gen.GeneratedVideoURL))
			} else if resp.TaskID != "" {
				// Video generation is async, start polling in background
				s.logger.Info("scene video generation started (async), starting background polling",
					zap.String("sceneId", gen.SceneID),
					zap.String("storyboardId", gen.StoryboardID),
					zap.String("provider", videoProvider),
					zap.String("taskId", resp.TaskID),
					zap.String("status", resp.Status),
					zap.String("message", resp.Message))

				// Save taskId and provider name to database for recovery
				gen.ProviderTaskID = resp.TaskID
				gen.ProviderName = videoProvider

				// Start background polling goroutine
				// Keep status as processing, polling will update it when done
				go s.pollVideoGenerationStatus(context.Background(), gen, videoProvider, resp.TaskID)

				// Don't mark as completed here - let polling update the status
				// Just update the record to keep it in processing state with taskId saved
				_ = s.repo.UpdateVideoGeneration(ctx, gen)
				s.logger.Info("video generation process started (async), waiting for polling to complete",
					zap.String("storyboardId", gen.StoryboardID),
					zap.String("sceneId", gen.SceneID),
					zap.String("status", string(gen.Status)),
					zap.String("taskId", resp.TaskID),
					zap.String("provider", videoProvider))
				return // Exit early, polling will handle completion
			} else {
				s.logger.Warn("video generation response has no video URL or task ID",
					zap.String("sceneId", gen.SceneID),
					zap.String("storyboardId", gen.StoryboardID),
					zap.String("provider", videoProvider),
					zap.String("status", resp.Status),
					zap.String("message", resp.Message))
			}
		}
	}

updateGeneration:
	// Only mark as completed if we have a video URL (synchronous case)
	// For async tasks, polling will handle completion
	if gen.GeneratedVideoURL != "" {
		s.logger.Info("video generation completed synchronously",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("sceneId", gen.SceneID),
			zap.String("videoURL", gen.GeneratedVideoURL))
		gen.Status = domain.GenerationStatusCompleted
		now := time.Now().Unix()
		gen.CompletedAt = &now
		if gen.Duration == 0 {
			gen.Duration = 5 // Default duration if not set
			s.logger.Debug("setting default video duration",
				zap.String("generationId", gen.ID),
				zap.Int("duration", gen.Duration))
		}
		if err := s.repo.UpdateVideoGeneration(ctx, gen); err != nil {
			s.logger.Error("failed to update video generation status to completed",
				zap.String("generationId", gen.ID),
				zap.Error(err))
		} else {
			s.logger.Debug("video generation status updated to completed",
				zap.String("generationId", gen.ID))
		}

		s.logger.Debug("updating storyboard token consumption",
			zap.String("storyboardId", gen.StoryboardID))
		s.updateStoryboardTokens(ctx, gen.StoryboardID)

		s.syncVideoReadyWorkflowAfterVideoCompletion(ctx, gen.StoryboardID)

		// Record metrics: completed
		if s.metrics != nil {
			duration := time.Since(startTime)
			isSubdivided := gen.IsSubdivided
			s.metrics.RecordStoryboardVideoGeneration("completed", isSubdivided, duration)
			s.metrics.RecordVideoGenerationSubdivided(isSubdivided, "completed")
			// Record AI generation
			s.metrics.RecordAIGeneration(s.effectiveVideoProvider(), "video")
			if gen.TotalTokens > 0 {
				s.metrics.RecordStoryboardTokenConsumed(gen.StoryboardID, float64(gen.TotalTokens))
			}
		}

		s.logger.Info("video generation process completed successfully",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("sceneId", gen.SceneID),
			zap.String("status", string(gen.Status)),
			zap.String("videoURL", gen.GeneratedVideoURL),
			zap.Int("duration", gen.Duration),
			zap.Int("totalTokens", gen.TotalTokens))
	} else {
		// No video URL and no task ID - mark as failed
		s.logger.Warn("video generation failed - no video URL or task ID",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("sceneId", gen.SceneID),
			zap.String("providerTaskID", gen.ProviderTaskID))
		gen.Status = domain.GenerationStatusFailed
		gen.ErrorMessage = formatGenerationError(GenerationErrorProvider, "video generation failed: no video URL or task ID returned")
		if err := s.repo.UpdateVideoGeneration(ctx, gen); err != nil {
			s.logger.Error("failed to update video generation status to failed",
				zap.String("generationId", gen.ID),
				zap.Error(err))
		} else {
			s.logger.Debug("video generation status updated to failed",
				zap.String("generationId", gen.ID),
				zap.String("errorMessage", gen.ErrorMessage))
		}
		// Record metrics: failed
		if s.metrics != nil {
			duration := time.Since(startTime)
			s.metrics.RecordStoryboardVideoGeneration("failed", gen.IsSubdivided, duration)
			s.metrics.RecordVideoGenerationError("unknown")
		}
		s.maybeFinalizeBatchVideoWorkflow(ctx, gen.StoryboardID)
	}
}

// syncVideoReadyWorkflowAfterVideoCompletion 单视频成功落库后更新 workflow：批量图后视频模式走聚合 finalize，否则立即 video_ready。
func (s *Service) syncVideoReadyWorkflowAfterVideoCompletion(ctx context.Context, storyboardID string) {
	if s.repo == nil {
		return
	}
	sb, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil || sb == nil {
		return
	}
	if sb.GenerateVideoAfterImages {
		s.maybeFinalizeBatchVideoWorkflow(ctx, storyboardID)
		return
	}
	if err := s.repo.UpdateStoryboardWorkflow(ctx, storyboardID, domain.WorkflowStatusVideoReady, 4); err != nil {
		s.logger.Warn("failed to update storyboard workflow to video ready",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	} else {
		s.logger.Debug("storyboard workflow updated to video ready",
			zap.String("storyboardId", storyboardID))
		if s.metrics != nil {
			s.metrics.RecordStoryboardWorkflowCompleted("video_ready", 0)
		}
	}
}

// GetGenerationProgress returns the complete generation progress for a storyboard
func (s *Service) GetGenerationProgress(ctx context.Context, storyboardID string) (*domain.StoryboardGenerationProgress, error) {
	s.logger.Info("getting generation progress",
		zap.String("storyboardId", storyboardID))

	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		s.logger.Error("storyboard not found for generation progress",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

	s.logger.Debug("storyboard retrieved for generation progress",
		zap.String("storyboardId", storyboardID),
		zap.String("workflowStatus", storyboard.WorkflowStatus),
		zap.Int("currentStep", storyboard.CurrentStep))

	var latestRun *domain.StoryboardGenerationRun
	if r, err := s.repo.LatestStoryboardGenerationRun(ctx, storyboardID); err == nil && r != nil {
		latestRun = r
	}

	progress := &domain.StoryboardGenerationProgress{
		StoryboardID:   storyboardID,
		WorkflowStatus: storyboard.WorkflowStatus,
		CurrentStep:    storyboard.CurrentStep,
		TotalTokens:    storyboard.TokenConsumption,
	}

	// Track generation status
	var isGenerating, hasPending, hasFailed bool
	var statusMessages []string

	// Get content generation
	if contentGen, err := s.repo.GetContentGenerationByStoryboard(ctx, storyboardID); err == nil {
		s.logger.Debug("content generation found",
			zap.String("storyboardId", storyboardID),
			zap.String("status", string(contentGen.Status)))
		progress.ContentGeneration = contentGen
		if contentGen.Status == domain.GenerationStatusProcessing {
			isGenerating = true
			statusMessages = append(statusMessages, "正在生成内容")
		} else if contentGen.Status == domain.GenerationStatusPending {
			hasPending = true
			statusMessages = append(statusMessages, "内容生成待处理")
		} else if contentGen.Status == domain.GenerationStatusFailed {
			hasFailed = true
		}
	} else {
		s.logger.Debug("no content generation found",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	}

	// Get scene generations
	if sceneGens, err := s.repo.ListSceneGenerations(ctx, storyboardID); err == nil {
		s.logger.Debug("scene generations found",
			zap.String("storyboardId", storyboardID),
			zap.Int("count", len(sceneGens)))
		progress.SceneGenerations = sceneGens
		for _, gen := range sceneGens {
			if gen.Status == domain.GenerationStatusProcessing {
				isGenerating = true
				statusMessages = append(statusMessages, "正在生成场景描述")
				break
			} else if gen.Status == domain.GenerationStatusPending {
				hasPending = true
			} else if gen.Status == domain.GenerationStatusFailed {
				hasFailed = true
			}
		}
	} else {
		s.logger.Debug("failed to list scene generations",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	}

	// Get image generations — deduplicate to latest per scene so that old failed records
	// from previous attempts don't shadow active retry/pending records after RetryFailedImages.
	if imageGens, err := s.repo.ListImageGenerations(ctx, storyboardID); err == nil {
		s.logger.Debug("image generations found",
			zap.String("storyboardId", storyboardID),
			zap.Int("count", len(imageGens)))
		// Keep only the latest record per scene (same logic used by RetryFailedStoryboardImages).
		latestByScene := latestStoryboardImageGenerationByScene(imageGens)
		latestImageGens := make([]*domain.StoryboardImageGeneration, 0, len(latestByScene))
		for _, g := range latestByScene {
			latestImageGens = append(latestImageGens, g)
		}
		progress.ImageGenerations = latestImageGens
		processingCount := 0
		failedCount := 0
		for _, gen := range latestImageGens {
			if gen.Status == domain.GenerationStatusProcessing {
				processingCount++
			} else if gen.Status == domain.GenerationStatusPending {
				hasPending = true
			} else if gen.Status == domain.GenerationStatusFailed {
				failedCount++
			} else if gen.Status == domain.GenerationStatusCompleted && gen.GeneratedImageURL == "" {
				// Strict completion: completed status without image URL is treated as failed.
				failedCount++
			}
		}
		if processingCount > 0 {
			isGenerating = true
			statusMessages = append(statusMessages, fmt.Sprintf("正在生成图片 (%d)", processingCount))
		}
		if failedCount > 0 {
			hasFailed = true
			statusMessages = append(statusMessages, fmt.Sprintf("图片生成失败 (%d)", failedCount))
		}
	} else {
		s.logger.Debug("failed to list image generations",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	}

	// Get video generations
	if videoGens, err := s.repo.ListVideoGenerations(ctx, storyboardID); err == nil {
		s.logger.Debug("video generations found",
			zap.String("storyboardId", storyboardID),
			zap.Int("count", len(videoGens)))
		progress.VideoGenerations = videoGens
		processingCount := 0
		failedCount := 0
		for _, gen := range videoGens {
			if gen.Status == domain.GenerationStatusProcessing {
				processingCount++
			} else if gen.Status == domain.GenerationStatusPending {
				hasPending = true
			} else if gen.Status == domain.GenerationStatusFailed {
				failedCount++
			} else if gen.Status == domain.GenerationStatusCompleted && gen.GeneratedVideoURL == "" {
				// Strict completion: completed status without video URL is treated as failed.
				failedCount++
			}
		}
		if processingCount > 0 {
			isGenerating = true
			statusMessages = append(statusMessages, fmt.Sprintf("正在生成视频 (%d)", processingCount))
		}
		if failedCount > 0 {
			hasFailed = true
			statusMessages = append(statusMessages, fmt.Sprintf("视频生成失败 (%d)", failedCount))
		}
	} else {
		s.logger.Debug("failed to list video generations",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	}

	// Latest redesign pipeline run (bible + scene plan) may be active without legacy generation rows updating yet.
	if latestRun != nil {
		switch latestRun.Status {
		case domain.GenerationStatusProcessing:
			isGenerating = true
			msg := "正在生成分镜结构"
			switch latestRun.CurrentStep {
			case domain.StoryboardGenerationStepContext:
				msg = "正在准备分镜上下文"
			case domain.StoryboardGenerationStepBiblePlan:
				msg = "正在生成视觉圣经与节拍"
			case domain.StoryboardGenerationStepScenePlan:
				msg = "正在规划分镜场景"
			default:
				if step := strings.TrimSpace(latestRun.CurrentStep); step != "" {
					msg = fmt.Sprintf("正在生成分镜结构（%s）", step)
				}
			}
			statusMessages = append(statusMessages, msg)
		case domain.GenerationStatusFailed:
			hasFailed = true
			if em := strings.TrimSpace(latestRun.ErrorMessage); em != "" {
				statusMessages = append(statusMessages, "分镜结构生成失败: "+em)
			}
		}
	}

	// Set final status
	progress.IsGenerating = isGenerating
	progress.HasPendingTasks = hasPending

	if isGenerating {
		progress.GenerationMessage = strings.Join(statusMessages, ", ")
	} else if hasPending {
		progress.GenerationMessage = "有待处理的生成任务"
	} else if hasFailed {
		progress.GenerationMessage = "存在失败的生成任务"
	} else if progress.ContentGeneration == nil && len(progress.SceneGenerations) == 0 &&
		len(progress.ImageGenerations) == 0 && len(progress.VideoGenerations) == 0 {
		progress.GenerationMessage = "无生成任务"
	} else if storyboard.WorkflowStatus == domain.WorkflowStatusContentReady && len(progress.ImageGenerations) == 0 {
		// 文案/分镜已就绪但配图任务尚未落库或未开始时，旧逻辑会误报「所有生成任务已完成」。
		progress.GenerationMessage = "等待配图生成"
		progress.HasPendingTasks = true
	} else {
		progress.GenerationMessage = "所有生成任务已完成"
	}

	// Only log progress retrieval while there is ongoing work.
	// Once all tasks are completed, this endpoint can be polled frequently by clients;
	// suppressing the completed-state log reduces noise.
	if progress.IsGenerating || progress.HasPendingTasks {
		s.logger.Info("generation progress retrieved",
			zap.String("storyboardId", storyboardID),
			zap.Bool("isGenerating", progress.IsGenerating),
			zap.Bool("hasPendingTasks", progress.HasPendingTasks),
			zap.String("generationMessage", progress.GenerationMessage),
			zap.Int("totalTokens", progress.TotalTokens))
	}

	var sceneRows []*domain.StoryboardScene
	if s.repo != nil {
		if rows, err := s.repo.StoryboardScenes(ctx, storyboardID); err == nil {
			sceneRows = rows
		} else {
			s.logger.Debug("StoryboardScenes for pipeline derivation failed",
				zap.String("storyboardId", storyboardID),
				zap.Error(err))
		}
	}
	steps, suggested := deriveWizardPipelineSteps(
		storyboard,
		sceneRows,
		progress.ContentGeneration,
		progress.SceneGenerations,
		progress.ImageGenerations,
		isGenerating,
		latestRun,
	)
	if len(steps) > 0 {
		progress.PipelineSteps = steps
	}
	if suggested != "" && suggested != domain.SuggestedResumeNone {
		progress.SuggestedResumeAction = suggested
	}
	if latestRun != nil {
		progress.LatestRun = latestRun
		progress.ConsistencyIssuesJSON = latestRun.ConsistencyIssuesJSON
		if audits, auditErr := s.repo.ListAIPromptAuditRecords(ctx, latestRun.ID); auditErr == nil {
			progress.PromptAuditRecordIDs = make([]string, 0, len(audits))
			for _, audit := range audits {
				if audit != nil {
					progress.PromptAuditRecordIDs = append(progress.PromptAuditRecordIDs, audit.ID)
				}
			}
		}
	}

	enrichStoryboardProgressFineGrain(progress)

	return progress, nil
}

// RetryFailedStoryboardImages retries failed image generations for a storyboard.
// It starts new generation jobs only for scenes whose latest image generation row (by time) is failed,
// so old failed rows after a successful retry do not trigger duplicate work.
func (s *Service) RetryFailedStoryboardImages(ctx context.Context, storyboardID string, opts *RetryFailedStoryboardImageOptions) (retried int, remainingFailed int, err error) {
	forceComic := opts != nil && opts.ForceComicPagePipeline
	imageGens, err := s.repo.ListImageGenerations(ctx, storyboardID)
	if err != nil {
		return 0, 0, err
	}

	latestByScene := latestStoryboardImageGenerationByScene(imageGens)
	var toRetry []*domain.StoryboardImageGeneration
	for _, gen := range latestByScene {
		if gen == nil {
			continue
		}
		needsRetry := gen.Status == domain.GenerationStatusFailed ||
			(gen.Status == domain.GenerationStatusCompleted && strings.TrimSpace(gen.GeneratedImageURL) == "")
		if !needsRetry {
			continue
		}
		toRetry = append(toRetry, gen)
	}

	for _, gen := range toRetry {
		comicStyle := strings.TrimSpace(gen.ComicStyle)
		if comicStyle == "" {
			if sb, sbErr := s.repo.StoryboardByID(ctx, storyboardID); sbErr == nil && sb != nil {
				comicStyle = strings.TrimSpace(sb.ContinuationComicStyle)
			}
		}
		useComic := forceComic || strings.TrimSpace(gen.PipelineKind) == domain.StoryboardImagePipelineComicPage
		var retryErr error
		if useComic {
			_, retryErr = s.GenerateStoryboardComicPage(ctx, &ComicPageGenerationRequest{
				StoryboardID:             storyboardID,
				SceneID:                  gen.SceneID,
				SceneTitle:               gen.SceneTitle,
				SceneDescription:         gen.SceneDescription,
				ReferenceImages:          gen.ReferenceImages,
				SceneCharacters:          gen.SceneCharacters,
				CharacterReferenceImages: gen.CharacterReferenceImages,
				StoryStyle:               gen.StoryStyle,
				ComicStyle:               comicStyle,
				SkipPeerFailureGate:      true,
				Pipeline:                 ComicPagePipelineOptions{},
			})
		} else {
			_, retryErr = s.GenerateSceneImage(ctx, &ImageGenerationRequest{
				StoryboardID:             storyboardID,
				SceneID:                  gen.SceneID,
				SceneTitle:               gen.SceneTitle,
				SceneDescription:         gen.SceneDescription,
				ReferenceImages:          gen.ReferenceImages,
				SceneCharacters:          gen.SceneCharacters,
				CharacterReferenceImages: gen.CharacterReferenceImages,
				StoryStyle:               gen.StoryStyle,
				ComicStyle:               comicStyle,
				SkipPeerFailureGate:      true,
			})
		}
		if retryErr != nil {
			s.logger.Warn("failed to retry image generation",
				zap.String("storyboardId", storyboardID),
				zap.String("sceneId", gen.SceneID),
				zap.Bool("comicPage", useComic),
				zap.Error(retryErr))
			continue
		}
		retried++
	}

	// Count remaining failed based on latest record per scene only,
	// so newly-created pending retry records don't get masked by the old failed rows.
	allImageGens, listErr := s.repo.ListImageGenerations(ctx, storyboardID)
	if listErr != nil {
		return retried, 0, nil
	}
	postRetryLatest := latestStoryboardImageGenerationByScene(allImageGens)
	for _, gen := range postRetryLatest {
		if gen != nil && gen.Status == domain.GenerationStatusFailed {
			remainingFailed++
		}
	}

	return retried, remainingFailed, nil
}

// CancelStoryboardGeneration cancels all in-flight generation tasks for a storyboard.
func (s *Service) CancelStoryboardGeneration(ctx context.Context, storyboardID, userID string) (cancelledCount int, err error) {
	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return 0, err
	}
	if storyboard.UserID != userID {
		return 0, fmt.Errorf("permission denied: not the creator")
	}

	cancelMessage := formatGenerationError(GenerationErrorCancelled, "generation cancelled by user")

	if contentGen, getErr := s.repo.GetContentGenerationByStoryboard(ctx, storyboardID); getErr == nil && contentGen != nil {
		if contentGen.Status == domain.GenerationStatusPending || contentGen.Status == domain.GenerationStatusProcessing {
			contentGen.Status = domain.GenerationStatusFailed
			contentGen.ErrorMessage = cancelMessage
			now := time.Now().Unix()
			contentGen.CompletedAt = &now
			if updateErr := s.repo.UpdateContentGeneration(ctx, contentGen); updateErr == nil {
				cancelledCount++
			}
		}
	}

	if sceneGens, listErr := s.repo.ListSceneGenerations(ctx, storyboardID); listErr == nil {
		for _, gen := range sceneGens {
			if gen.Status != domain.GenerationStatusPending && gen.Status != domain.GenerationStatusProcessing {
				continue
			}
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = cancelMessage
			now := time.Now().Unix()
			gen.CompletedAt = &now
			if updateErr := s.repo.UpdateSceneGeneration(ctx, gen); updateErr == nil {
				cancelledCount++
			}
		}
	}

	if imageGens, listErr := s.repo.ListImageGenerations(ctx, storyboardID); listErr == nil {
		for _, gen := range imageGens {
			if gen.Status != domain.GenerationStatusPending && gen.Status != domain.GenerationStatusProcessing {
				continue
			}
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = cancelMessage
			now := time.Now().Unix()
			gen.CompletedAt = &now
			if updateErr := s.repo.UpdateImageGeneration(ctx, gen); updateErr == nil {
				cancelledCount++
			}
		}
	}

	if videoGens, listErr := s.repo.ListVideoGenerations(ctx, storyboardID); listErr == nil {
		for _, gen := range videoGens {
			if gen.Status != domain.GenerationStatusPending && gen.Status != domain.GenerationStatusProcessing {
				continue
			}
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = cancelMessage
			now := time.Now().Unix()
			gen.CompletedAt = &now
			if updateErr := s.repo.UpdateVideoGeneration(ctx, gen); updateErr == nil {
				cancelledCount++
			}
		}
	}

	return cancelledCount, nil
}

// PublishStoryboard publishes a storyboard (Step 5)
func (s *Service) PublishStoryboard(ctx context.Context, storyboardID string) error {
	s.logger.Info("publishing storyboard",
		zap.String("storyboardId", storyboardID))

	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		s.logger.Error("storyboard not found for publishing",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return fmt.Errorf("storyboard not found: %w", err)
	}

	s.logger.Debug("storyboard retrieved for publishing",
		zap.String("storyboardId", storyboardID),
		zap.String("storyId", storyboard.StoryID),
		zap.String("currentWorkflowStatus", storyboard.WorkflowStatus),
		zap.Int("currentStep", storyboard.CurrentStep))

	wasPublished := storyboard.WorkflowStatus == domain.WorkflowStatusPublished

	// Update workflow status to published using the dedicated workflow update method
	if err := s.repo.UpdateStoryboardWorkflow(ctx, storyboardID, domain.WorkflowStatusPublished, 5); err != nil {
		s.logger.Error("failed to publish storyboard",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return fmt.Errorf("failed to publish storyboard: %w", err)
	}

	if !wasPublished {
		s.onChildStoryboardPublished(ctx, storyboard)
	}

	// Record metrics: workflow completed (published)
	if s.metrics != nil {
		// Get storyboard creation time to calculate workflow duration
		workflowDuration := time.Since(time.Unix(storyboard.CreatedAt, 0))
		s.metrics.RecordStoryboardWorkflowCompleted("published", workflowDuration)
	}

	s.logger.Info("storyboard published successfully",
		zap.String("storyboardId", storyboardID),
		zap.String("storyId", storyboard.StoryID))

	storyTitle := ""
	if st, err := s.repo.StoryByID(ctx, storyboard.StoryID); err == nil && st != nil {
		storyTitle = st.Title
	} else if err != nil {
		s.logger.Warn("publish notify: failed to load story title",
			zap.String("storyId", storyboard.StoryID),
			zap.Error(err))
	}

	pubName, pubAvatar := "", ""
	if pu, err := s.repo.UserByID(ctx, storyboard.UserID); err == nil && pu != nil {
		pubName = pu.DisplayName
		pubAvatar = pu.Avatar
	}

	const followPageSize = 200
	for offset := 0; ; offset += followPageSize {
		follows, ferr := s.repo.ListStoryFollowRecordsByStory(ctx, storyboard.StoryID, followPageSize, offset)
		if ferr != nil {
			s.logger.Warn("publish notify: list story follows failed",
				zap.String("storyId", storyboard.StoryID),
				zap.Error(ferr))
			break
		}
		if len(follows) == 0 {
			break
		}
		for _, f := range follows {
			if f == nil || f.FollowerID == "" || f.FollowerID == storyboard.UserID {
				continue
			}
			if err := s.NotifyFollowedStoryPublishedStoryboard(ctx, f.FollowerID, storyboard.StoryID, storyboardID, storyboard.UserID, pubName, pubAvatar, storyTitle); err != nil {
				s.logger.Warn("publish notify: follower notification failed",
					zap.Error(err),
					zap.String("followerId", f.FollowerID))
			}
		}
		if len(follows) < followPageSize {
			break
		}
	}

	return nil
}

// buildGenerationContext builds comprehensive context string for storyboard generation
func (s *Service) buildGenerationContext(ctx context.Context, storyboard *domain.Storyboard, characterIDs, sceneIDs []string) string {
	s.logger.Debug("building generation context",
		zap.String("storyboardId", storyboard.ID),
		zap.String("storyId", storyboard.StoryID),
		zap.Int("characterCount", len(characterIDs)),
		zap.Int("sceneCount", len(sceneIDs)))

	var parts []string

	// 1. Get ancestor storyboards (up to 5) along the parent chain
	s.logger.Debug("fetching ancestor storyboards for context",
		zap.String("storyboardId", storyboard.ID),
		zap.String("parentId", storyboard.ParentID))
	ancestors := s.getAncestorStoryboards(ctx, storyboard, 5)
	s.logger.Debug("ancestor storyboards fetched",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("ancestorCount", len(ancestors)))

	// Get story information for root storyboard
	story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
	if err != nil {
		s.logger.Warn("failed to get story for context",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", storyboard.StoryID),
			zap.Error(err))
	} else {
		s.logger.Debug("story retrieved for context",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", story.ID),
			zap.String("storyTitle", story.Title))
	}

	// 2. Add previous storyboard context (chronological order)
	if len(ancestors) > 0 {
		parts = append(parts, "## Previous Storyboards Context")

		for i, ancestor := range ancestors {
			parts = append(parts, fmt.Sprintf("\n### Storyboard %d: %s", i+1, ancestor.Title))

			// If this is the first storyboard (root), add story description
			if i == 0 {
				isRoot := ancestor.ParentID == "" || ancestor.ParentID == domain.StoryboardRootMarker
				if isRoot && story != nil && story.Description != "" {
					parts = append(parts, fmt.Sprintf("Story Background: %s", story.Description))
				}
			}

			// Add storyboard content
			if ancestor.Content != "" {
				parts = append(parts, fmt.Sprintf("Content: %s", ancestor.Content))
			} else if ancestor.RawInput != "" {
				parts = append(parts, fmt.Sprintf("Raw Input: %s", ancestor.RawInput))
			}

			// Add characters that participated in this storyboard
			if len(ancestor.CharacterRefs) > 0 {
				var charNames []string
				for _, ref := range ancestor.CharacterRefs {
					if ref.Character != nil {
						charNames = append(charNames, ref.Character.Name)
					} else if ref.CharacterID != "" {
						// Try to get character name by ID
						if char, err := s.repo.CharacterByID(ctx, ref.CharacterID); err == nil {
							charNames = append(charNames, char.Name)
						} else {
							s.logger.Debug("failed to get character by ID for ancestor context",
								zap.String("storyboardId", storyboard.ID),
								zap.String("characterId", ref.CharacterID),
								zap.Error(err))
						}
					}
				}
				if len(charNames) > 0 {
					parts = append(parts, fmt.Sprintf("Participating Characters: %s", strings.Join(charNames, ", ")))
				}
			}
		}
	} else if story != nil && story.Description != "" {
		// If no ancestors, this is the first storyboard - use story description
		parts = append(parts, "## Story Background")
		parts = append(parts, story.Description)
	}

	// 3. Add current storyboard's participating characters with detailed information
	if len(characterIDs) > 0 {
		s.logger.Debug("adding current storyboard characters to context",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("characterCount", len(characterIDs)))
		parts = append(parts, "\n## Current Storyboard Characters")

		for _, charID := range characterIDs {
			char, err := s.repo.CharacterByID(ctx, charID)
			if err != nil {
				s.logger.Warn("failed to get character for context",
					zap.String("storyboardId", storyboard.ID),
					zap.String("characterId", charID),
					zap.Error(err))
				continue
			}

			s.logger.Debug("adding character to context",
				zap.String("storyboardId", storyboard.ID),
				zap.String("characterId", charID),
				zap.String("characterName", char.Name))

			var charInfo []string
			charInfo = append(charInfo, fmt.Sprintf("Character ID: %s", char.ID))
			charInfo = append(charInfo, fmt.Sprintf("Name: %s", char.Name))

			if char.Description != "" {
				charInfo = append(charInfo, fmt.Sprintf("Description: %s", char.Description))
			}
			if char.Personality != "" {
				charInfo = append(charInfo, fmt.Sprintf("Personality: %s", char.Personality))
			}
			if char.Appearance != "" {
				charInfo = append(charInfo, fmt.Sprintf("Appearance: %s", char.Appearance))
			}
			if char.DressPreference != "" {
				charInfo = append(charInfo, fmt.Sprintf("Dress Preference: %s", char.DressPreference))
			}
			if char.Background != "" {
				charInfo = append(charInfo, fmt.Sprintf("Background: %s", char.Background))
			}
			if char.HandlingStyle != "" {
				charInfo = append(charInfo, fmt.Sprintf("Handling Style: %s", char.HandlingStyle))
			}
			if char.CognitionRange != "" {
				charInfo = append(charInfo, fmt.Sprintf("Cognition Range: %s", char.CognitionRange))
			}
			if char.AbilityFeatures != "" {
				charInfo = append(charInfo, fmt.Sprintf("Ability Features: %s", char.AbilityFeatures))
			}
			if char.ShortTermGoal != "" {
				charInfo = append(charInfo, fmt.Sprintf("Short-term Goal: %s", char.ShortTermGoal))
			}
			if char.LongTermGoal != "" {
				charInfo = append(charInfo, fmt.Sprintf("Long-term Goal: %s", char.LongTermGoal))
			}
			if len(char.Traits) > 0 {
				charInfo = append(charInfo, fmt.Sprintf("Traits: %s", strings.Join(char.Traits, ", ")))
			}
			if len(char.Skills) > 0 {
				charInfo = append(charInfo, fmt.Sprintf("Skills: %s", strings.Join(char.Skills, ", ")))
			}

			parts = append(parts, strings.Join(charInfo, "\n"))
			parts = append(parts, "") // Empty line between characters
		}
	}

	// 4. Add scene information
	if len(sceneIDs) > 0 {
		s.logger.Debug("adding scenes to context",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneCount", len(sceneIDs)))
		parts = append(parts, "\n## Storyboard Scenes")

		for _, sceneID := range sceneIDs {
			scene, err := s.repo.StorySceneByID(ctx, storyboard.StoryID, sceneID)
			if err != nil {
				s.logger.Warn("failed to get scene for context",
					zap.String("storyboardId", storyboard.ID),
					zap.String("storyId", storyboard.StoryID),
					zap.String("sceneId", sceneID),
					zap.Error(err))
				continue
			}

			s.logger.Debug("adding scene to context",
				zap.String("storyboardId", storyboard.ID),
				zap.String("sceneId", sceneID),
				zap.String("sceneTitle", scene.Title))

			var sceneInfo []string
			sceneInfo = append(sceneInfo, fmt.Sprintf("Scene: %s", scene.Title))
			if scene.Location != "" {
				sceneInfo = append(sceneInfo, fmt.Sprintf("Location: %s", scene.Location))
			}
			if scene.TimeOfDay != "" {
				sceneInfo = append(sceneInfo, fmt.Sprintf("Time of Day: %s", scene.TimeOfDay))
			}
			if scene.Description != "" {
				sceneInfo = append(sceneInfo, fmt.Sprintf("Description: %s", scene.Description))
			}

			parts = append(parts, strings.Join(sceneInfo, "\n"))
			parts = append(parts, "") // Empty line between scenes
		}
	}

	contextStr := strings.Join(parts, "\n")
	s.logger.Debug("generation context built",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("contextLength", len(contextStr)),
		zap.Int("partCount", len(parts)))

	return contextStr
}

// updateStoryboardTokens aggregates and updates token consumption
func (s *Service) updateStoryboardTokens(ctx context.Context, storyboardID string) {
	s.logger.Debug("updating storyboard token consumption",
		zap.String("storyboardId", storyboardID))

	tokens, err := s.repo.GetStoryboardTotalTokens(ctx, storyboardID)
	if err != nil {
		s.logger.Warn("failed to get total tokens",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return
	}

	s.logger.Debug("total tokens retrieved",
		zap.String("storyboardId", storyboardID),
		zap.Int("totalTokens", tokens))

	if err := s.repo.UpdateStoryboardTokens(ctx, storyboardID, tokens); err != nil {
		s.logger.Warn("failed to update storyboard tokens",
			zap.String("storyboardId", storyboardID),
			zap.Int("tokens", tokens),
			zap.Error(err))
	} else {
		s.logger.Debug("storyboard tokens updated successfully",
			zap.String("storyboardId", storyboardID),
			zap.Int("tokens", tokens))

		// Record metrics
		if s.metrics != nil && tokens > 0 {
			s.metrics.RecordStoryboardTokenConsumed(storyboardID, float64(tokens))
		}
	}
}

// pollVideoGenerationStatus polls the video generation task status until completion
func (s *Service) pollVideoGenerationStatus(ctx context.Context, gen *domain.StoryboardVideoGeneration, providerName, taskID string) {
	// Recover from panic to prevent goroutine crash
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("pollVideoGenerationStatus panic recovered",
				zap.String("sceneId", gen.SceneID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("provider", providerName),
				zap.String("taskId", taskID),
				zap.Any("panic", r))
			// Update status to failed
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(GenerationErrorUnknown, fmt.Sprintf("panic: %v", r))
			_ = s.repo.UpdateVideoGeneration(context.Background(), gen)
			s.maybeFinalizeBatchVideoWorkflow(context.Background(), gen.StoryboardID)
		}
	}()

	// Log immediately when goroutine starts
	s.logger.Info("pollVideoGenerationStatus goroutine started",
		zap.String("sceneId", gen.SceneID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("provider", providerName),
		zap.String("taskId", taskID))

	const (
		maxPollAttempts = 120             // 最大轮询次数 (10分钟，每5秒一次)
		pollInterval    = 5 * time.Second // 轮询间隔
	)

	// Check if genAPI is available
	if s.genAPI == nil {
		s.logger.Error("genAPI is nil, cannot poll video status",
			zap.String("sceneId", gen.SceneID),
			zap.String("storyboardId", gen.StoryboardID))
		gen.Status = domain.GenerationStatusFailed
		gen.ErrorMessage = formatGenerationError(GenerationErrorProvider, "genAPI is not available")
		_ = s.repo.UpdateVideoGeneration(ctx, gen)
		s.maybeFinalizeBatchVideoWorkflow(ctx, gen.StoryboardID)
		return
	}

	// Reload from database to ensure we have the latest data
	latestGen, err := s.repo.GetVideoGeneration(ctx, gen.ID)
	if err != nil {
		s.logger.Warn("failed to reload video generation from database, using provided data",
			zap.String("sceneId", gen.SceneID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.Error(err))
		// Continue with provided gen
	} else {
		gen = latestGen
	}

	s.logger.Info("starting video generation polling",
		zap.String("sceneId", gen.SceneID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("provider", providerName),
		zap.String("taskId", taskID),
		zap.Int("maxAttempts", maxPollAttempts),
		zap.Duration("pollInterval", pollInterval))

	// Custom polling with detailed logging
	pollCtx, cancel := context.WithTimeout(ctx, maxPollAttempts*pollInterval)
	defer cancel()

	// Attach usage record metadata for TokenUsageRecorder (GetVideoStatus has no request)
	pollCtx = s.ctxWithUsageRecordForStoryboard(pollCtx, gen.StoryboardID)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var resp *genapi.GenerateResponse
	var pollErr error
	attempt := 0

	// Check immediately first
	s.logger.Info("checking video status immediately",
		zap.String("sceneId", gen.SceneID),
		zap.String("taskId", taskID))

	resp, pollErr = s.genAPI.GetVideoStatus(pollCtx, providerName, taskID)
	if pollErr != nil {
		s.logger.Error("initial video status check failed",
			zap.String("sceneId", gen.SceneID),
			zap.String("taskId", taskID),
			zap.Error(pollErr))
	} else {
		s.logger.Info("initial video status",
			zap.String("sceneId", gen.SceneID),
			zap.String("taskId", taskID),
			zap.String("status", resp.Status),
			zap.String("videoURL", resp.VideoURL),
			zap.String("error", resp.Error))

		if genapi.IsTerminalStatus(resp.Status) {
			s.logger.Info("video generation already completed on first check",
				zap.String("sceneId", gen.SceneID),
				zap.String("taskId", taskID),
				zap.String("status", resp.Status))
			goto processResult
		}
	}

	// Poll until completion
	for attempt = 1; attempt <= maxPollAttempts; attempt++ {
		select {
		case <-pollCtx.Done():
			s.logger.Error("video generation polling context cancelled",
				zap.String("sceneId", gen.SceneID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("provider", providerName),
				zap.String("taskId", taskID),
				zap.Int("attempts", attempt),
				zap.Error(pollCtx.Err()))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = formatGenerationError(GenerationErrorCancelled, fmt.Sprintf("polling cancelled: %v", pollCtx.Err()))
			_ = s.repo.UpdateVideoGeneration(ctx, gen)
			s.maybeFinalizeBatchVideoWorkflow(ctx, gen.StoryboardID)
			return

		case <-ticker.C:
			s.logger.Debug("polling video status",
				zap.String("sceneId", gen.SceneID),
				zap.String("taskId", taskID),
				zap.Int("attempt", attempt),
				zap.Int("maxAttempts", maxPollAttempts))

			resp, pollErr = s.genAPI.GetVideoStatus(pollCtx, providerName, taskID)
			if pollErr != nil {
				s.logger.Warn("video status check failed, will retry",
					zap.String("sceneId", gen.SceneID),
					zap.String("taskId", taskID),
					zap.Int("attempt", attempt),
					zap.Error(pollErr))
				continue
			}

			s.logger.Info("video status update",
				zap.String("sceneId", gen.SceneID),
				zap.String("taskId", taskID),
				zap.String("status", resp.Status),
				zap.String("videoURL", resp.VideoURL),
				zap.String("error", resp.Error),
				zap.Int("attempt", attempt))

			if genapi.IsTerminalStatus(resp.Status) {
				s.logger.Info("video generation reached terminal status",
					zap.String("sceneId", gen.SceneID),
					zap.String("taskId", taskID),
					zap.String("status", resp.Status),
					zap.Int("attempts", attempt))
				goto processResult
			}
		}
	}

	// Timeout - max attempts reached
	s.logger.Error("video generation polling timeout",
		zap.String("sceneId", gen.SceneID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("provider", providerName),
		zap.String("taskId", taskID),
		zap.Int("attempts", attempt))
	gen.Status = domain.GenerationStatusFailed
	gen.ErrorMessage = formatGenerationError(GenerationErrorTimeout, fmt.Sprintf("polling timeout after %d attempts", attempt))
	_ = s.repo.UpdateVideoGeneration(ctx, gen)
	s.maybeFinalizeBatchVideoWorkflow(ctx, gen.StoryboardID)
	return

processResult:
	// Check if video generation succeeded
	if resp == nil || resp.VideoURL == "" {
		errMsg := "video URL is empty after polling"
		if resp != nil && resp.Error != "" {
			errMsg = resp.Error
		}
		s.logger.Error("video generation completed but no video URL",
			zap.String("sceneId", gen.SceneID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("provider", providerName),
			zap.String("taskId", taskID),
			zap.String("status", func() string {
				if resp != nil {
					return resp.Status
				} else {
					return "nil"
				}
			}()),
			zap.String("error", func() string {
				if resp != nil {
					return resp.Error
				} else {
					return "response is nil"
				}
			}()))

		gen.Status = domain.GenerationStatusFailed
		gen.ErrorMessage = formatGenerationError(GenerationErrorProvider, errMsg)
		_ = s.repo.UpdateVideoGeneration(ctx, gen)
		s.maybeFinalizeBatchVideoWorkflow(ctx, gen.StoryboardID)
		return
	}

	s.logger.Info("video generation polling completed, uploading to OSS",
		zap.String("sceneId", gen.SceneID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("provider", providerName),
		zap.String("taskId", taskID),
		zap.String("videoURL", resp.VideoURL),
		zap.String("status", resp.Status))

	// Upload video to OSS
	originalVideoURL := resp.VideoURL
	ossClient := aliyun.GetGlobalClient()
	if ossClient != nil {
		// Generate OSS object key
		objectKey := fmt.Sprintf("storyboard/%s/scenes/%s.mp4", gen.StoryboardID, gen.SceneID)
		s.logger.Info("uploading video to OSS",
			zap.String("sceneId", gen.SceneID),
			zap.String("objectKey", objectKey),
			zap.String("sourceURL", originalVideoURL))

		// Upload to OSS
		ossURL, err := ossClient.UploadFileFromURL(objectKey, originalVideoURL)
		if err != nil {
			s.logger.Warn("failed to upload video to OSS, using original URL",
				zap.String("sceneId", gen.SceneID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("objectKey", objectKey),
				zap.String("originalURL", originalVideoURL),
				zap.Error(err))
			// Use original URL if upload fails
			gen.GeneratedVideoURL = originalVideoURL
		} else {
			// Clean URL: remove query params and ensure HTTPS
			ossURL = strings.Split(ossURL, "?")[0]
			ossURL = strings.ReplaceAll(ossURL, "http://", "https://")
			gen.GeneratedVideoURL = ossURL
			s.logger.Info("video uploaded to OSS successfully",
				zap.String("sceneId", gen.SceneID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("objectKey", objectKey),
				zap.String("ossURL", ossURL),
				zap.String("originalURL", originalVideoURL))
		}
	} else {
		s.logger.Warn("OSS client not available, using original video URL",
			zap.String("sceneId", gen.SceneID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("originalURL", originalVideoURL))
		gen.GeneratedVideoURL = originalVideoURL
	}

	// Update duration if available
	if resp.Usage != nil && resp.Usage.DurationSeconds > 0 {
		gen.Duration = resp.Usage.DurationSeconds
	} else if gen.Duration == 0 {
		gen.Duration = 5 // Default duration
	}

	// Add video generation tokens to total (if available)
	if resp.Usage != nil {
		gen.TotalTokens += resp.Usage.TotalTokens
	}

	// Update generation record
	gen.Status = domain.GenerationStatusCompleted
	now := time.Now().Unix()
	gen.CompletedAt = &now
	if err := s.repo.UpdateVideoGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to update video generation record",
			zap.String("sceneId", gen.SceneID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.Error(err))
		return
	}

	// Sync video URL and subdivision info to storyboard scene for easy access
	if gen.GeneratedVideoURL != "" && gen.SceneID != "" {
		if gen.IsSubdivided {
			// Sync with subdivision info for seamless HLS playback
			if err := s.repo.UpdateStoryboardSceneVideoWithSubdivision(
				ctx,
				gen.SceneID,
				gen.GeneratedVideoURL,
				gen.IsSubdivided,
				gen.VideoSegmentsJSON,
				gen.MiddleFrameURLsJSON,
			); err != nil {
				s.logger.Warn("failed to sync video with subdivision to storyboard scene",
					zap.String("sceneId", gen.SceneID),
					zap.String("videoURL", gen.GeneratedVideoURL),
					zap.Bool("isSubdivided", gen.IsSubdivided),
					zap.Error(err))
			} else {
				s.logger.Info("video with subdivision synced to storyboard scene",
					zap.String("sceneId", gen.SceneID),
					zap.String("videoURL", gen.GeneratedVideoURL),
					zap.Bool("isSubdivided", gen.IsSubdivided))
			}
		} else {
			// Simple video URL update
			if err := s.repo.UpdateStoryboardSceneVideo(ctx, gen.SceneID, gen.GeneratedVideoURL); err != nil {
				s.logger.Warn("failed to sync video URL to storyboard scene",
					zap.String("sceneId", gen.SceneID),
					zap.String("videoURL", gen.GeneratedVideoURL),
					zap.Error(err))
			} else {
				s.logger.Info("video URL synced to storyboard scene",
					zap.String("sceneId", gen.SceneID),
					zap.String("videoURL", gen.GeneratedVideoURL))
			}
		}
	}

	// Update storyboard tokens
	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	s.syncVideoReadyWorkflowAfterVideoCompletion(ctx, gen.StoryboardID)

	s.logger.Info("video generation polling completed successfully",
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.String("videoURL", gen.GeneratedVideoURL),
		zap.Int("duration", gen.Duration),
		zap.Int("totalTokens", gen.TotalTokens))
}

// RecoverPendingVideoGenerations recovers all pending video generation tasks on service startup
func (s *Service) RecoverPendingVideoGenerations(ctx context.Context) {
	if s.genAPI == nil {
		s.logger.Warn("genAPI not available, skipping video generation recovery")
		return
	}

	s.logger.Info("starting recovery of pending video generation tasks")

	pendingTasks, err := s.repo.ListPendingVideoGenerations(ctx)
	if err != nil {
		s.logger.Error("failed to list pending video generations",
			zap.Error(err))
		return
	}

	if len(pendingTasks) == 0 {
		s.logger.Info("no pending video generation tasks to recover")
		return
	}

	s.logger.Info("found pending video generation tasks to recover",
		zap.Int("count", len(pendingTasks)))

	for _, gen := range pendingTasks {
		if gen.ProviderTaskID == "" || gen.ProviderName == "" {
			s.logger.Warn("skipping video generation task without provider info",
				zap.String("sceneId", gen.SceneID),
				zap.String("storyboardId", gen.StoryboardID))
			continue
		}

		s.logger.Info("recovering video generation task",
			zap.String("sceneId", gen.SceneID),
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("provider", gen.ProviderName),
			zap.String("taskId", gen.ProviderTaskID))

		// Start background polling for this task
		go s.pollVideoGenerationStatus(context.Background(), gen, gen.ProviderName, gen.ProviderTaskID)
	}

	s.logger.Info("video generation recovery completed",
		zap.Int("recoveredTasks", len(pendingTasks)))
}

// ============== Keyframe Subdivision Helper Functions ==============

// serializeVideoSegments converts video segments to JSON string for storage.
// recordStoryboardTextGeneration records text generation (content/scene/image_prompt/video_prompt) to AIGenerationRecord.
func (s *Service) recordStoryboardTextGeneration(ctx context.Context, storyboardID, subtype, provider, prompt, output string, inputTokens, outputTokens int, status domain.AITaskStatus, errorMsg string) {
	if s.aiGenService == nil {
		return
	}
	meta := s.usageRecordMetadataForStoryboard(ctx, storyboardID)
	if meta == nil {
		return
	}
	userID, _ := meta["user_id"].(string)
	relID, _ := meta["related_entity_id"].(string)
	if relID == "" {
		relID = storyboardID
	}
	if provider == "" {
		provider = "text_llm"
	}
	s.aiGenService.RecordTextGenerationUsage(ctx, &RecordTextGenerationRequest{
		UserID:            userID,
		RelatedEntityID:   relID,
		RelatedEntityType: "storyboard",
		Provider:          provider,
		Model:             "",
		OriginalPrompt:    prompt,
		OutputResult:      output,
		Step:              subtype,
		InputTokens:       inputTokens,
		OutputTokens:      outputTokens,
		Status:            status,
		ErrorMessage:      errorMsg,
	})
}

// ctxWithUsageRecordForStoryboard attaches usage record metadata to ctx for GetVideoStatus/GetImageStatus.
func (s *Service) ctxWithUsageRecordForStoryboard(ctx context.Context, storyboardID string) context.Context {
	meta := s.usageRecordMetadataForStoryboard(ctx, storyboardID)
	if meta == nil {
		return ctx
	}
	userID, _ := meta["user_id"].(string)
	relID, _ := meta["related_entity_id"].(string)
	relType, _ := meta["related_entity_type"].(string)
	if relID == "" {
		relID = storyboardID
	}
	if relType == "" {
		relType = "storyboard"
	}
	return genapi.ContextWithUsageRecord(ctx, userID, relID, relType)
}

// usageRecordMetadataForStoryboard returns metadata for AIGenerationRecord attribution (user, storyboard).
func (s *Service) usageRecordMetadataForStoryboard(ctx context.Context, storyboardID string) map[string]interface{} {
	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil || storyboard == nil {
		return nil
	}
	story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
	if err != nil || story == nil {
		return map[string]interface{}{
			"related_entity_id":   storyboardID,
			"related_entity_type": "storyboard",
		}
	}
	return map[string]interface{}{
		"user_id":             story.UserID,
		"related_entity_id":   storyboardID,
		"related_entity_type": "storyboard",
	}
}

func (s *Service) serializeVideoSegments(segments []genapi.VideoSegment) string {
	if len(segments) == 0 {
		return ""
	}
	// Convert genapi.VideoSegment to domain.VideoSegmentInfo
	domainSegments := make([]domain.VideoSegmentInfo, len(segments))
	for i, seg := range segments {
		domainSegments[i] = domain.VideoSegmentInfo{
			Index:        seg.Index,
			VideoURL:     seg.VideoURL,
			StartFrame:   seg.StartFrame,
			EndFrame:     seg.EndFrame,
			DurationSecs: seg.DurationSecs,
		}
	}
	data, err := json.Marshal(domainSegments)
	if err != nil {
		s.logger.Warn("failed to serialize video segments",
			zap.Error(err))
		return ""
	}
	return string(data)
}

// serializeMiddleFrames converts middle frame URLs to JSON string for storage.
func (s *Service) serializeMiddleFrames(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	data, err := json.Marshal(urls)
	if err != nil {
		s.logger.Warn("failed to serialize middle frames",
			zap.Error(err))
		return ""
	}
	return string(data)
}

// deserializeVideoSegments parses JSON string back to video segments.
func (s *Service) deserializeVideoSegments(jsonStr string) []domain.VideoSegmentInfo {
	if jsonStr == "" {
		return nil
	}
	var segments []domain.VideoSegmentInfo
	if err := json.Unmarshal([]byte(jsonStr), &segments); err != nil {
		s.logger.Warn("failed to deserialize video segments",
			zap.Error(err))
		return nil
	}
	return segments
}

// deserializeMiddleFrames parses JSON string back to middle frame URLs.
func (s *Service) deserializeMiddleFrames(jsonStr string) []string {
	if jsonStr == "" {
		return nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(jsonStr), &urls); err != nil {
		s.logger.Warn("failed to deserialize middle frames",
			zap.Error(err))
		return nil
	}
	return urls
}

// ============== HLS Playlist Support ==============

// GetVideoGenerationBySceneID retrieves video generation info for a specific scene
func (s *Service) GetVideoGenerationBySceneID(ctx context.Context, storyboardID, sceneID string) (*VideoGenerationInfo, error) {
	// Get video generation record for this scene
	videoGens, err := s.repo.ListVideoGenerations(ctx, storyboardID)
	if err != nil {
		return nil, err
	}

	for _, gen := range videoGens {
		if gen.SceneID == sceneID && gen.Status == domain.GenerationStatusCompleted {
			return s.convertToVideoGenerationInfo(gen), nil
		}
	}

	return nil, domain.ErrNotFound
}

// GetVideoGenerationsByStoryboardID retrieves all video generations for a storyboard
func (s *Service) GetVideoGenerationsByStoryboardID(ctx context.Context, storyboardID string) ([]*VideoGenerationInfo, error) {
	videoGens, err := s.repo.ListVideoGenerations(ctx, storyboardID)
	if err != nil {
		return nil, err
	}

	result := make([]*VideoGenerationInfo, 0)
	for _, gen := range videoGens {
		if gen.Status == domain.GenerationStatusCompleted && gen.GeneratedVideoURL != "" {
			result = append(result, s.convertToVideoGenerationInfo(gen))
		}
	}

	return result, nil
}

// convertToVideoGenerationInfo converts domain model to service info type
func (s *Service) convertToVideoGenerationInfo(gen *domain.StoryboardVideoGeneration) *VideoGenerationInfo {
	info := &VideoGenerationInfo{
		SceneID:           gen.SceneID,
		GeneratedVideoURL: gen.GeneratedVideoURL,
		Duration:          gen.Duration,
		IsSubdivided:      gen.IsSubdivided,
	}

	// Convert video segments if available
	if len(gen.VideoSegments) > 0 {
		info.VideoSegments = make([]VideoSegmentInfo, len(gen.VideoSegments))
		for i, seg := range gen.VideoSegments {
			info.VideoSegments[i] = VideoSegmentInfo{
				Index:        seg.Index,
				VideoURL:     seg.VideoURL,
				StartFrame:   seg.StartFrame,
				EndFrame:     seg.EndFrame,
				DurationSecs: seg.DurationSecs,
			}
		}
	} else if gen.VideoSegmentsJSON != "" {
		// Parse from JSON if not already parsed
		segments := s.deserializeVideoSegments(gen.VideoSegmentsJSON)
		info.VideoSegments = make([]VideoSegmentInfo, len(segments))
		for i, seg := range segments {
			info.VideoSegments[i] = VideoSegmentInfo{
				Index:        seg.Index,
				VideoURL:     seg.VideoURL,
				StartFrame:   seg.StartFrame,
				EndFrame:     seg.EndFrame,
				DurationSecs: seg.DurationSecs,
			}
		}
	}

	return info
}

func (s *Service) parseImagePromptDetails(text, sceneTitle, sceneDescription string) (*domain.ImagePromptDetails, string) {
	// Clean the text - remove markdown code blocks if present
	cleanedText := strings.TrimSpace(text)
	if strings.HasPrefix(cleanedText, "```") {
		// Find the end of the first line (skip ```json or ```)
		if idx := strings.Index(cleanedText, "\n"); idx != -1 {
			cleanedText = cleanedText[idx+1:]
		}
		// Remove trailing ```
		if idx := strings.LastIndex(cleanedText, "```"); idx != -1 {
			cleanedText = strings.TrimSpace(cleanedText[:idx])
		}
	}

	// Try to find JSON object in the text
	startIdx := strings.Index(cleanedText, "{")
	endIdx := strings.LastIndex(cleanedText, "}")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		s.logger.Debug("no JSON object found in prompt response",
			zap.String("text", truncateForLog(cleanedText, 200)))
		return nil, ""
	}

	jsonText := cleanedText[startIdx : endIdx+1]
	var details domain.ImagePromptDetails
	if err := json.Unmarshal([]byte(jsonText), &details); err != nil {
		s.logger.Warn("failed to parse structured prompt JSON",
			zap.Error(err),
			zap.String("jsonText", truncateForLog(jsonText, 200)))
		return nil, ""
	}

	// Combine structured details into final prompt text
	combinedPrompt := s.combineImagePrompt(&details, sceneTitle, sceneDescription)

	return &details, combinedPrompt
}

// combineImagePrompt 将结构化的提示词详情组合成最终的文本提示词
func (s *Service) combineImagePrompt(details *domain.ImagePromptDetails, sceneTitle, sceneDescription string, languages ...string) string {
	var parts []string
	language := inferGenerationLanguage(sceneTitle + "\n" + sceneDescription)
	if len(languages) > 0 && strings.TrimSpace(languages[0]) != "" {
		language = normalizeGenerationLanguage(languages[0])
	}

	if details.ArtStyle != "" {
		parts = append(parts, fmt.Sprintf("Art style: %s", details.ArtStyle))
	}
	if details.Lighting != "" {
		parts = append(parts, fmt.Sprintf("Lighting: %s", details.Lighting))
	}
	if details.ColorPalette != "" {
		parts = append(parts, fmt.Sprintf("Color palette: %s", details.ColorPalette))
	}
	if details.Composition != "" {
		parts = append(parts, fmt.Sprintf("Composition: %s", details.Composition))
	}
	if len(details.KeyElements) > 0 {
		parts = append(parts, fmt.Sprintf("Key elements: %s", strings.Join(details.KeyElements, ", ")))
	}
	if details.Mood != "" {
		parts = append(parts, fmt.Sprintf("Mood: %s", details.Mood))
	}
	if details.AdditionalNotes != "" {
		parts = append(parts, details.AdditionalNotes)
	}
	// Inject structured comic text instructions so the image model renders them in-picture.
	if len(details.ComicTexts) > 0 {
		var letteringLines []string
		for _, ct := range details.ComicTexts {
			if strings.TrimSpace(ct.Text) == "" {
				continue
			}
			panelRef := ""
			if ct.PanelIndex != nil {
				panelRef = fmt.Sprintf(" in panel %d", *ct.PanelIndex+1)
			}
			switch ct.Type {
			case "narration":
				pos := ct.Position
				if pos == "" {
					pos = "top of panel"
				}
				letteringLines = append(letteringLines, fmt.Sprintf("Draw a rectangular narration caption box%s at %s with only the exact supplied text 「%s」 in a clean comic font", panelRef, pos, ct.Text))
			case "dialogue":
				pos := ct.Position
				if pos == "" {
					pos = "speech-bubble"
				}
				speaker := ct.Speaker
				if speaker == "" {
					speaker = "the character"
				}
				letteringLines = append(letteringLines, fmt.Sprintf("Draw a speech balloon%s (oval with pointed tail toward %s) at %s containing only the exact supplied text 「%s」", panelRef, speaker, pos, ct.Text))
			case "thought":
				pos := ct.Position
				if pos == "" {
					pos = "thought-bubble"
				}
				speaker := ct.Speaker
				if speaker == "" {
					speaker = "the character"
				}
				letteringLines = append(letteringLines, fmt.Sprintf("Draw a thought cloud%s (bubble-chain outline) near %s at %s with only the exact supplied text 「%s」", panelRef, speaker, pos, ct.Text))
			case "sfx":
				pos := ct.Position
				if pos == "" {
					pos = "mid-frame"
				}
				letteringLines = append(letteringLines, fmt.Sprintf("Render bold oversized SFX text%s 「%s」 at %s in dynamic comic lettering style", panelRef, ct.Text, pos))
			default:
				pos := ct.Position
				if pos == "" {
					pos = "mid-frame"
				}
				letteringLines = append(letteringLines, fmt.Sprintf("Render only the exact supplied text%s 「%s」 at %s", panelRef, ct.Text, pos))
			}
		}
		visualOnly := strings.Join(parts, ". ")
		visualOnly = capGeminiImageBeautyOutput(visualOnly)
		var beauty string
		if len(letteringLines) > 0 {
			// Lettering MUST appear before the capped visual block inside 【画面与镜头】 so
			// truncateStoryboardImagePromptPreservingNarrative (tail-truncates beauty) and
			// capGeminiImageBeautyOutput do not drop comic text instructions.
			visualTexts := storyboardComicTextsToVisual(details.ComicTexts, language)
			letteringBlock := "[COMIC LETTERING — use only the supplied text]\n" + strings.Join(letteringLines, "; ") + "\n" + visualSceneLetteringPolicy(language, visualTexts)
			if visualOnly != "" {
				beauty = letteringBlock + "\n\n" + visualOnly
			} else {
				beauty = letteringBlock
			}
		} else {
			beauty = visualOnly
		}
		nb := storyboardSceneNarrativeBlock(sceneTitle, sceneDescription)
		return prependStoryboardImageNarrativeBlock(nb, beauty)
	}

	parts = append(parts, visualSceneLetteringPolicy(language, nil))
	beauty := strings.Join(parts, ". ")
	beauty = capGeminiImageBeautyOutput(beauty)
	nb := storyboardSceneNarrativeBlock(sceneTitle, sceneDescription)
	return prependStoryboardImageNarrativeBlock(nb, beauty)
}

// normalizeStoryboardComicTexts enforces the same density contract used by the
// fragment pipeline. Text length and type counts are deterministic product rules,
// so they should not depend on whether a model remembered the prose instruction.
func normalizeStoryboardComicTexts(texts []domain.StoryboardComicText) []domain.StoryboardComicText {
	return normalizeStoryboardComicTextsForLanguage(texts, "zh-Hans")
}

func normalizeStoryboardComicTextsForLanguage(texts []domain.StoryboardComicText, language string) []domain.StoryboardComicText {
	counts := map[string]int{}
	limits := map[string]int{"narration": 1, "dialogue": 2, "thought": 1, "sfx": 1}
	out := make([]domain.StoryboardComicText, 0, len(texts))
	for _, item := range texts {
		typ := strings.ToLower(strings.TrimSpace(item.Type))
		if _, ok := limits[typ]; !ok || counts[typ] >= limits[typ] {
			continue
		}
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		counts[typ]++
		item.Type = typ
		item.Text = truncateRunes(text, comicTextRuneLimit(language))
		item.Speaker = strings.TrimSpace(item.Speaker)
		item.Position = strings.TrimSpace(item.Position)
		out = append(out, item)
	}
	return out
}

// applyPlannedStoryboardComicTextsToDetails makes the persisted scene plan the
// sole lettering authority. The normalizer may improve visual controls, but it
// may not invent dialogue or signs during the second text-model pass.
func applyPlannedStoryboardComicTextsToDetails(details *domain.ImagePromptDetails, planned *domain.StoryboardScene, languages ...string) {
	if details == nil {
		return
	}
	if planned == nil {
		details.ComicTexts = nil
		return
	}
	language := "zh-Hans"
	if len(languages) > 0 && strings.TrimSpace(languages[0]) != "" {
		language = normalizeGenerationLanguage(languages[0])
	}
	details.ComicTexts = normalizeStoryboardComicTextsForLanguage(planned.ComicTexts, language)
}

// parseVideoPromptDetails 解析AI返回的结构化视频提示词JSON
func (s *Service) parseVideoPromptDetails(text string) (*domain.VideoPromptDetails, string) {
	// Clean the text - remove markdown code blocks if present
	cleanedText := strings.TrimSpace(text)
	if strings.HasPrefix(cleanedText, "```") {
		// Find the end of the first line (skip ```json or ```)
		if idx := strings.Index(cleanedText, "\n"); idx != -1 {
			cleanedText = cleanedText[idx+1:]
		}
		// Remove trailing ```
		if idx := strings.LastIndex(cleanedText, "```"); idx != -1 {
			cleanedText = strings.TrimSpace(cleanedText[:idx])
		}
	}

	// Try to find JSON object in the text
	startIdx := strings.Index(cleanedText, "{")
	endIdx := strings.LastIndex(cleanedText, "}")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		s.logger.Debug("no JSON object found in video prompt response",
			zap.String("text", truncateForLog(cleanedText, 200)))
		return nil, ""
	}

	jsonText := cleanedText[startIdx : endIdx+1]
	var details domain.VideoPromptDetails
	if err := json.Unmarshal([]byte(jsonText), &details); err != nil {
		s.logger.Warn("failed to parse structured video prompt JSON",
			zap.Error(err),
			zap.String("jsonText", truncateForLog(jsonText, 200)))
		return nil, ""
	}

	// Combine structured details into final prompt text
	combinedPrompt := s.combineVideoPrompt(&details)

	return &details, combinedPrompt
}

// combineVideoPrompt 将结构化的视频提示词详情组合成最终的文本提示词
func (s *Service) combineVideoPrompt(details *domain.VideoPromptDetails) string {
	var parts []string

	if details.CameraMovement != "" {
		parts = append(parts, fmt.Sprintf("Camera: %s", details.CameraMovement))
	}
	if details.SubjectMotion != "" {
		parts = append(parts, fmt.Sprintf("Action: %s", details.SubjectMotion))
	}
	if details.Atmosphere != "" {
		parts = append(parts, fmt.Sprintf("Atmosphere: %s", details.Atmosphere))
	}
	if details.TransitionStyle != "" {
		parts = append(parts, fmt.Sprintf("Transition: %s", details.TransitionStyle))
	}
	if details.Duration != "" {
		parts = append(parts, fmt.Sprintf("Duration: %s", details.Duration))
	}
	if len(details.KeyMoments) > 0 {
		parts = append(parts, fmt.Sprintf("Key moments: %s", strings.Join(details.KeyMoments, ", ")))
	}
	if details.AdditionalNotes != "" {
		parts = append(parts, details.AdditionalNotes)
	}

	return strings.Join(parts, ". ")
}
