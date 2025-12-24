package service

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	s.logger.Info("processing content generation",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("rawInput", truncateForLog(gen.RawInput, 200)),
		zap.String("style", gen.Style))

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

	// Generate content using AI
	// 直接调用 geminiClient，将结果记录到 StoryboardContentGeneration 表，不创建 AIGenerationRecord
	if s.geminiClient != nil {
		s.logger.Info("generating content with AI",
			zap.String("generationId", gen.ID),
			zap.String("style", style),
			zap.Int("contextLength", len(contextStr)),
			zap.Int("rawInputLength", len(gen.RawInput)))
		// Fallback to direct gemini client if aiGenService not available
		prompt := fmt.Sprintf(`You are a creative story writer. Based on the following context and user input, generate an engaging story chapter.

Context:
%s

User Input: %s
Style: %s

Generate a compelling narrative that continues the story naturally. Keep the tone consistent with the style requested.`, contextStr, gen.RawInput, style)

		s.logger.Debug("calling gemini client for content generation",
			zap.String("generationId", gen.ID),
			zap.Int("promptLength", len(prompt)))

		text, resp, err := s.geminiClient.GenerateText(ctx, "", prompt, nil)
		if err != nil {
			s.logger.Error("AI content generation failed",
				zap.String("generationId", gen.ID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.Error(err))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = err.Error()
			if updateErr := s.repo.UpdateContentGeneration(ctx, gen); updateErr != nil {
				s.logger.Error("failed to update content generation status to failed",
					zap.String("generationId", gen.ID),
					zap.Error(updateErr))
			}
			return
		}

		gen.GeneratedContent = text
		if resp != nil && resp.UsageMetadata != nil {
			gen.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
			gen.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			gen.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
			s.logger.Debug("content generation token usage recorded",
				zap.String("generationId", gen.ID),
				zap.Int("inputTokens", gen.InputTokens),
				zap.Int("outputTokens", gen.OutputTokens),
				zap.Int("totalTokens", gen.TotalTokens))
		} else {
			s.logger.Warn("no usage metadata in AI response",
				zap.String("generationId", gen.ID))
		}

		s.logger.Info("content generated successfully",
			zap.String("generationId", gen.ID),
			zap.Int("contentLength", len(gen.GeneratedContent)),
			zap.Int("totalTokens", gen.TotalTokens))
	} else {
		// If no AI client, use raw input as content
		s.logger.Warn("no AI client available, using raw input as content",
			zap.String("generationId", gen.ID),
			zap.String("storyboardId", gen.StoryboardID))
		gen.GeneratedContent = gen.RawInput
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
	s.logger.Info("processing scene detail generation",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.String("sceneTitle", gen.SceneTitle),
		zap.String("sceneLocation", gen.SceneLocation))

	gen.Status = domain.GenerationStatusProcessing
	if err := s.repo.UpdateSceneGeneration(ctx, gen); err != nil {
		s.logger.Warn("failed to update scene generation status to processing",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("scene generation status updated to processing",
			zap.String("generationId", gen.ID))
	}

	// 直接调用 geminiClient，将结果记录到 StoryboardSceneGeneration 表，不创建 AIGenerationRecord
	if s.geminiClient != nil {
		s.logger.Info("generating scene details with AI",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("sceneTitle", gen.SceneTitle))
		// Fallback to direct gemini client if aiGenService not available
		prompt := fmt.Sprintf(`Enhance and expand the following scene description with vivid details:

Scene Title: %s
Location: %s
Original Description: %s

Provide a rich, detailed description that includes:
- Visual details (lighting, colors, atmosphere)
- Sensory elements (sounds, smells, textures)
- Character positions and actions
- Emotional tone and mood`, gen.SceneTitle, gen.SceneLocation, gen.InputDescription)

		s.logger.Debug("calling gemini client for scene detail generation",
			zap.String("generationId", gen.ID),
			zap.Int("promptLength", len(prompt)))

		text, resp, err := s.geminiClient.GenerateText(ctx, "", prompt, nil)
		if err != nil {
			s.logger.Error("AI scene detail generation failed",
				zap.String("generationId", gen.ID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("sceneId", gen.SceneID),
				zap.Error(err))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = err.Error()
			if updateErr := s.repo.UpdateSceneGeneration(ctx, gen); updateErr != nil {
				s.logger.Error("failed to update scene generation status to failed",
					zap.String("generationId", gen.ID),
					zap.Error(updateErr))
			}
			return
		}

		gen.GeneratedDetail = text
		if resp != nil && resp.UsageMetadata != nil {
			gen.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
			gen.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			gen.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
			s.logger.Debug("scene generation token usage recorded",
				zap.String("generationId", gen.ID),
				zap.Int("inputTokens", gen.InputTokens),
				zap.Int("outputTokens", gen.OutputTokens),
				zap.Int("totalTokens", gen.TotalTokens))
		} else {
			s.logger.Warn("no usage metadata in AI response",
				zap.String("generationId", gen.ID))
		}

		s.logger.Info("scene details generated successfully",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.Int("detailLength", len(gen.GeneratedDetail)),
			zap.Int("totalTokens", gen.TotalTokens))
	} else {
		s.logger.Warn("no AI client available, using input description as generated detail",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID))
		gen.GeneratedDetail = gen.InputDescription
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
		zap.Int("referenceImageCount", len(req.ReferenceImages)))

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

	// Create generation record
	gen := &domain.StoryboardImageGeneration{
		ID:               uuid.New().String(),
		StoryboardID:     req.StoryboardID,
		SceneID:          req.SceneID,
		SceneTitle:       req.SceneTitle,
		SceneDescription: req.SceneDescription,
		ReferenceImages:  req.ReferenceImages,
		Status:           domain.GenerationStatusPending,
		CreatedAt:        time.Now().Unix(),
	}

	s.logger.Debug("creating image generation record",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID))

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
		zap.String("sceneId", gen.SceneID))

	// Start async generation
	s.logger.Info("starting async image generation process",
		zap.String("generationId", gen.ID),
		zap.String("sceneId", gen.SceneID))
	go s.processImageGeneration(context.Background(), gen)

	return gen, nil
}

// processImageGeneration processes image generation in background
func (s *Service) processImageGeneration(ctx context.Context, gen *domain.StoryboardImageGeneration) {
	s.logger.Info("processing image generation",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.String("sceneTitle", gen.SceneTitle),
		zap.Int("referenceImageCount", len(gen.ReferenceImages)))

	gen.Status = domain.GenerationStatusProcessing
	if err := s.repo.UpdateImageGeneration(ctx, gen); err != nil {
		s.logger.Warn("failed to update image generation status to processing",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("image generation status updated to processing",
			zap.String("generationId", gen.ID))
	}

	// First, generate image prompt using text AI
	// 直接调用 geminiClient，将结果记录到 StoryboardImageGeneration 表，不创建 AIGenerationRecord
	if s.geminiClient != nil {
		s.logger.Info("generating image prompt with AI",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("sceneTitle", gen.SceneTitle))
		// Fallback to direct gemini client if aiGenService not available
		promptGen := fmt.Sprintf(`Create a detailed image generation prompt for the following scene:

Scene Title: %s
Scene Description: %s

Generate a prompt that would create a visually stunning image. Include:
- Art style and medium
- Lighting and atmosphere
- Color palette
- Composition details
- Key visual elements

Keep the prompt concise but descriptive, suitable for an image generation AI. Output ONLY the prompt, no explanations.`, gen.SceneTitle, gen.SceneDescription)

		s.logger.Debug("calling gemini client for image prompt generation",
			zap.String("generationId", gen.ID),
			zap.Int("promptLength", len(promptGen)))

		text, resp, err := s.geminiClient.GenerateText(ctx, "", promptGen, nil)
		if err != nil {
			s.logger.Error("AI image prompt generation failed",
				zap.String("generationId", gen.ID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("sceneId", gen.SceneID),
				zap.Error(err))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = err.Error()
			if updateErr := s.repo.UpdateImageGeneration(ctx, gen); updateErr != nil {
				s.logger.Error("failed to update image generation status to failed",
					zap.String("generationId", gen.ID),
					zap.Error(updateErr))
			}
			return
		}

		gen.GeneratedPrompt = text
		if resp != nil && resp.UsageMetadata != nil {
			gen.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
			gen.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			gen.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
			s.logger.Debug("image prompt generation token usage recorded",
				zap.String("generationId", gen.ID),
				zap.Int("inputTokens", gen.InputTokens),
				zap.Int("outputTokens", gen.OutputTokens),
				zap.Int("totalTokens", gen.TotalTokens))
		} else {
			s.logger.Warn("no usage metadata in AI response",
				zap.String("generationId", gen.ID))
		}

		s.logger.Info("image prompt generated successfully",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("prompt", truncateForLog(gen.GeneratedPrompt, 200)))
	} else {
		// If no AI client, use scene description as prompt
		s.logger.Warn("no AI client available, using scene description as prompt",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID))
		gen.GeneratedPrompt = gen.SceneDescription
	}

	// Generate actual image using genAPI directly
	// 直接调用 genAPI，将结果记录到 StoryboardImageGeneration 表，不创建 AIGenerationRecord
	if s.genAPI != nil && gen.GeneratedPrompt != "" {
		genReq := &genapi.GenerateRequest{
			Prompt:          gen.GeneratedPrompt,
			AspectRatio:     "16:9",
			Quality:         "high",
			OutputCount:     1,
			ReferenceImages: gen.ReferenceImages, // Use character reference images
		}

		// Choose operation type based on whether reference images are provided
		if len(gen.ReferenceImages) > 0 {
			genReq.Operation = genapi.OperationImageToImage
			// Use first reference image as the primary reference
			genReq.ReferenceImageURL = gen.ReferenceImages[0]
			s.logger.Debug("using image-to-image operation",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID),
				zap.String("referenceImageURL", genReq.ReferenceImageURL))
		} else {
			genReq.Operation = genapi.OperationTextToImage
			s.logger.Debug("using text-to-image operation",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID))
		}

		// Use configured image provider (default: huoshan)
		imageProvider := s.imageProvider
		if imageProvider == "" {
			imageProvider = "huoshan"
			s.logger.Debug("using default image provider",
				zap.String("generationId", gen.ID),
				zap.String("provider", imageProvider))
		}

		s.logger.Info("generating scene image",
			zap.String("generationId", gen.ID),
			zap.String("sceneId", gen.SceneID),
			zap.String("provider", imageProvider),
			zap.String("operation", string(genReq.Operation)),
			zap.String("prompt", truncateForLog(gen.GeneratedPrompt, 200)))

		resp, err := s.genAPI.GenerateImage(ctx, imageProvider, genReq)
		if err != nil {
			s.logger.Warn("AI image generation failed, keeping prompt only",
				zap.String("generationId", gen.ID),
				zap.String("sceneId", gen.SceneID),
				zap.String("storyboardId", gen.StoryboardID),
				zap.String("provider", imageProvider),
				zap.String("operation", string(genReq.Operation)),
				zap.Error(err))
			// Don't fail completely, just mark as completed without image
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

	gen.Status = domain.GenerationStatusCompleted
	now := time.Now().Unix()
	gen.CompletedAt = &now
	if err := s.repo.UpdateImageGeneration(ctx, gen); err != nil {
		s.logger.Error("failed to update image generation status to completed",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("image generation status updated to completed",
			zap.String("generationId", gen.ID))
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

	// Update workflow status
	if err := s.repo.UpdateStoryboardWorkflow(ctx, gen.StoryboardID, domain.WorkflowStatusImagesReady, 3); err != nil {
		s.logger.Warn("failed to update storyboard workflow to images ready",
			zap.String("storyboardId", gen.StoryboardID),
			zap.Error(err))
	} else {
		s.logger.Debug("storyboard workflow updated to images ready",
			zap.String("storyboardId", gen.StoryboardID))
	}

	s.logger.Info("image generation completed successfully",
		zap.String("generationId", gen.ID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.String("imageURL", gen.GeneratedImageURL),
		zap.Int("totalTokens", gen.TotalTokens))
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
	s.logger.Info("starting video generation process",
		zap.String("sceneId", gen.SceneID),
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneTitle", gen.SceneTitle),
		zap.String("referenceImageURL", gen.ReferenceImageURL),
		zap.String("endFrameURL", gen.EndFrameURL),
		zap.String("inputDescription", gen.InputDescription))

	gen.Status = domain.GenerationStatusProcessing
	if err := s.repo.UpdateVideoGeneration(ctx, gen); err != nil {
		s.logger.Warn("failed to update video generation status to processing",
			zap.String("generationId", gen.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("video generation status updated to processing",
			zap.String("generationId", gen.ID))
	}

	// Generate video prompt using text AI
	// 直接调用 geminiClient，将结果记录到 StoryboardVideoGeneration 表，不创建 AIGenerationRecord
	if s.geminiClient != nil {
		// Fallback to direct gemini client if aiGenService not available
		promptGen := fmt.Sprintf(`Create a video generation prompt for the following scene:

Scene Title: %s
Scene Description: %s

Generate a prompt for a short video clip (5-10 seconds). Include:
- Camera movement (pan, zoom, static)
- Subject motion and actions
- Atmosphere and mood
- Transition style

Keep it concise and suitable for AI video generation. Output ONLY the prompt, no explanations.`, gen.SceneTitle, gen.InputDescription)

		s.logger.Info("generating video prompt with AI",
			zap.String("sceneId", gen.SceneID),
			zap.String("sceneTitle", gen.SceneTitle),
			zap.String("inputDescription", gen.InputDescription))

		text, resp, err := s.geminiClient.GenerateText(ctx, "", promptGen, nil)
		if err != nil {
			s.logger.Error("failed to generate video prompt",
				zap.String("sceneId", gen.SceneID),
				zap.Error(err))
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = err.Error()
			_ = s.repo.UpdateVideoGeneration(ctx, gen)
			return
		}

		gen.GeneratedPrompt = text
		if resp != nil && resp.UsageMetadata != nil {
			gen.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
			gen.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			gen.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
		}

		s.logger.Info("video prompt generated successfully",
			zap.String("sceneId", gen.SceneID),
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
		}

		// Choose operation type based on whether reference images are provided
		if gen.ReferenceImageURL != "" && gen.EndFrameURL != "" {
			// Keyframe mode: need both FirstFrameURL and LastFrameURL
			genReq.Operation = genapi.OperationKeyframeToVideo
			genReq.FirstFrameURL = gen.ReferenceImageURL // Map ReferenceImageURL to FirstFrameURL
			genReq.LastFrameURL = gen.EndFrameURL
			s.logger.Info("using keyframe-to-video mode",
				zap.String("sceneId", gen.SceneID),
				zap.String("firstFrameURL", genReq.FirstFrameURL),
				zap.String("lastFrameURL", genReq.LastFrameURL))
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

		// Use configured video provider (default: hailuo)
		videoProvider := s.videoProvider
		if videoProvider == "" {
			videoProvider = "hailuo"
			s.logger.Debug("using default video provider",
				zap.String("generationId", gen.ID),
				zap.String("provider", videoProvider))
		} else {
			s.logger.Debug("using configured video provider",
				zap.String("generationId", gen.ID),
				zap.String("provider", videoProvider))
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

		// Update workflow status
		if err := s.repo.UpdateStoryboardWorkflow(ctx, gen.StoryboardID, domain.WorkflowStatusVideoReady, 4); err != nil {
			s.logger.Warn("failed to update storyboard workflow to video ready",
				zap.String("storyboardId", gen.StoryboardID),
				zap.Error(err))
		} else {
			s.logger.Debug("storyboard workflow updated to video ready",
				zap.String("storyboardId", gen.StoryboardID))
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
		gen.ErrorMessage = "video generation failed: no video URL or task ID returned"
		if err := s.repo.UpdateVideoGeneration(ctx, gen); err != nil {
			s.logger.Error("failed to update video generation status to failed",
				zap.String("generationId", gen.ID),
				zap.Error(err))
		} else {
			s.logger.Debug("video generation status updated to failed",
				zap.String("generationId", gen.ID),
				zap.String("errorMessage", gen.ErrorMessage))
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

	progress := &domain.StoryboardGenerationProgress{
		StoryboardID:   storyboardID,
		WorkflowStatus: storyboard.WorkflowStatus,
		CurrentStep:    storyboard.CurrentStep,
		TotalTokens:    storyboard.TokenConsumption,
	}

	// Track generation status
	var isGenerating, hasPending bool
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
			}
		}
	} else {
		s.logger.Debug("failed to list scene generations",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	}

	// Get image generations
	if imageGens, err := s.repo.ListImageGenerations(ctx, storyboardID); err == nil {
		s.logger.Debug("image generations found",
			zap.String("storyboardId", storyboardID),
			zap.Int("count", len(imageGens)))
		progress.ImageGenerations = imageGens
		processingCount := 0
		for _, gen := range imageGens {
			if gen.Status == domain.GenerationStatusProcessing {
				processingCount++
			} else if gen.Status == domain.GenerationStatusPending {
				hasPending = true
			}
		}
		if processingCount > 0 {
			isGenerating = true
			statusMessages = append(statusMessages, fmt.Sprintf("正在生成图片 (%d)", processingCount))
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
		for _, gen := range videoGens {
			if gen.Status == domain.GenerationStatusProcessing {
				processingCount++
			} else if gen.Status == domain.GenerationStatusPending {
				hasPending = true
			}
		}
		if processingCount > 0 {
			isGenerating = true
			statusMessages = append(statusMessages, fmt.Sprintf("正在生成视频 (%d)", processingCount))
		}
	} else {
		s.logger.Debug("failed to list video generations",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
	}

	// Set final status
	progress.IsGenerating = isGenerating
	progress.HasPendingTasks = hasPending

	if isGenerating {
		progress.GenerationMessage = strings.Join(statusMessages, ", ")
	} else if hasPending {
		progress.GenerationMessage = "有待处理的生成任务"
	} else if progress.ContentGeneration == nil && len(progress.SceneGenerations) == 0 &&
		len(progress.ImageGenerations) == 0 && len(progress.VideoGenerations) == 0 {
		progress.GenerationMessage = "无生成任务"
	} else {
		progress.GenerationMessage = "所有生成任务已完成"
	}

	s.logger.Info("generation progress retrieved",
		zap.String("storyboardId", storyboardID),
		zap.Bool("isGenerating", progress.IsGenerating),
		zap.Bool("hasPendingTasks", progress.HasPendingTasks),
		zap.String("generationMessage", progress.GenerationMessage),
		zap.Int("totalTokens", progress.TotalTokens))

	return progress, nil
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

	// Update workflow status to published using the dedicated workflow update method
	if err := s.repo.UpdateStoryboardWorkflow(ctx, storyboardID, domain.WorkflowStatusPublished, 5); err != nil {
		s.logger.Error("failed to publish storyboard",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return fmt.Errorf("failed to publish storyboard: %w", err)
	}

	s.logger.Info("storyboard published successfully",
		zap.String("storyboardId", storyboardID),
		zap.String("storyId", storyboard.StoryID))

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
			gen.ErrorMessage = fmt.Sprintf("panic: %v", r)
			_ = s.repo.UpdateVideoGeneration(context.Background(), gen)
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
		gen.ErrorMessage = "genAPI is not available"
		_ = s.repo.UpdateVideoGeneration(ctx, gen)
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
			gen.ErrorMessage = fmt.Sprintf("polling cancelled: %v", pollCtx.Err())
			_ = s.repo.UpdateVideoGeneration(ctx, gen)
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
	gen.ErrorMessage = fmt.Sprintf("polling timeout after %d attempts", attempt)
	_ = s.repo.UpdateVideoGeneration(ctx, gen)
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
		gen.ErrorMessage = errMsg
		_ = s.repo.UpdateVideoGeneration(ctx, gen)
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

	// Sync video URL to storyboard scene for easy access
	if gen.GeneratedVideoURL != "" && gen.SceneID != "" {
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

	// Update storyboard tokens
	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	// Update workflow status
	_ = s.repo.UpdateStoryboardWorkflow(ctx, gen.StoryboardID, domain.WorkflowStatusVideoReady, 4)

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
