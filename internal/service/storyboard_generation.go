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
	// Verify storyboard exists
	storyboard, err := s.repo.StoryboardByID(ctx, req.StoryboardID)
	if err != nil {
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

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

	if err := s.repo.CreateContentGeneration(ctx, gen); err != nil {
		return nil, fmt.Errorf("failed to create content generation record: %w", err)
	}

	// Update storyboard workflow status
	if err := s.repo.UpdateStoryboardWorkflow(ctx, req.StoryboardID, domain.WorkflowStatusDraft, 2); err != nil {
		s.logger.Warn("failed to update storyboard workflow", zap.Error(err))
	}

	// Start async AI generation
	go s.processContentGeneration(context.Background(), gen, storyboard)

	return gen, nil
}

// processContentGeneration processes the content generation in background
func (s *Service) processContentGeneration(ctx context.Context, gen *domain.StoryboardContentGeneration, storyboard *domain.Storyboard) {
	// Update status to processing
	gen.Status = domain.GenerationStatusProcessing
	_ = s.repo.UpdateContentGeneration(ctx, gen)

	// Build context from characters and scenes
	contextStr := s.buildGenerationContext(ctx, storyboard, gen.CharacterIDs, gen.SceneIDs)

	// Determine style: use provided style, or fallback to story's genre or style
	style := gen.Style
	if style == "" {
		// Try to get story's genre or style as the default style
		story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
		if err == nil {
			if story.Genre != "" {
				style = story.Genre
			} else if story.Style != "" {
				style = story.Style
			} else {
				style = "drama" // Ultimate fallback
			}
		} else {
			style = "drama" // Ultimate fallback
		}
	}

	// Generate content using AI
	// 直接调用 geminiClient，将结果记录到 StoryboardContentGeneration 表，不创建 AIGenerationRecord
	if s.geminiClient != nil {
		// Fallback to direct gemini client if aiGenService not available
		prompt := fmt.Sprintf(`You are a creative story writer. Based on the following context and user input, generate an engaging story chapter.

Context:
%s

User Input: %s
Style: %s

Generate a compelling narrative that continues the story naturally. Keep the tone consistent with the style requested.`, contextStr, gen.RawInput, style)

		text, resp, err := s.geminiClient.GenerateText(ctx, "", prompt, nil)
		if err != nil {
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = err.Error()
			_ = s.repo.UpdateContentGeneration(ctx, gen)
			return
		}

		gen.GeneratedContent = text
		if resp != nil && resp.UsageMetadata != nil {
			gen.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
			gen.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			gen.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
		}
	} else {
		// If no AI client, use raw input as content
		gen.GeneratedContent = gen.RawInput
	}

	// Update status to completed
	gen.Status = domain.GenerationStatusCompleted
	now := time.Now().Unix()
	gen.CompletedAt = &now
	_ = s.repo.UpdateContentGeneration(ctx, gen)

	// Update storyboard content and token consumption
	storyboard.Content = gen.GeneratedContent
	storyboard.IsAIGenerated = true
	_ = s.repo.UpdateStoryboard(ctx, storyboard)

	// Aggregate and update token consumption
	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	// Update workflow status
	_ = s.repo.UpdateStoryboardWorkflow(ctx, gen.StoryboardID, domain.WorkflowStatusContentReady, 2)

	s.logger.Info("content generation completed",
		zap.String("storyboardId", gen.StoryboardID),
		zap.Int("tokens", gen.TotalTokens))
}

// GenerateSceneDetails generates detailed scene descriptions (Step 2)
func (s *Service) GenerateSceneDetails(ctx context.Context, req *SceneDetailRequest) (*domain.StoryboardSceneGeneration, error) {
	// Verify storyboard exists
	if _, err := s.repo.StoryboardByID(ctx, req.StoryboardID); err != nil {
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

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

	if err := s.repo.CreateSceneGeneration(ctx, gen); err != nil {
		return nil, fmt.Errorf("failed to create scene generation record: %w", err)
	}

	// Start async AI generation
	go s.processSceneGeneration(context.Background(), gen)

	return gen, nil
}

// processSceneGeneration processes scene detail generation in background
func (s *Service) processSceneGeneration(ctx context.Context, gen *domain.StoryboardSceneGeneration) {
	gen.Status = domain.GenerationStatusProcessing
	_ = s.repo.UpdateSceneGeneration(ctx, gen)

	// 直接调用 geminiClient，将结果记录到 StoryboardSceneGeneration 表，不创建 AIGenerationRecord
	if s.geminiClient != nil {
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

		text, resp, err := s.geminiClient.GenerateText(ctx, "", prompt, nil)
		if err != nil {
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = err.Error()
			_ = s.repo.UpdateSceneGeneration(ctx, gen)
			return
		}

		gen.GeneratedDetail = text
		if resp != nil && resp.UsageMetadata != nil {
			gen.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
			gen.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			gen.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
		}
	} else {
		gen.GeneratedDetail = gen.InputDescription
	}

	gen.Status = domain.GenerationStatusCompleted
	now := time.Now().Unix()
	gen.CompletedAt = &now
	_ = s.repo.UpdateSceneGeneration(ctx, gen)

	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	s.logger.Info("scene generation completed",
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID))
}

// GenerateSceneImage generates an image for a scene (Step 3)
func (s *Service) GenerateSceneImage(ctx context.Context, req *ImageGenerationRequest) (*domain.StoryboardImageGeneration, error) {
	// Verify storyboard exists
	if _, err := s.repo.StoryboardByID(ctx, req.StoryboardID); err != nil {
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

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

	if err := s.repo.CreateImageGeneration(ctx, gen); err != nil {
		return nil, fmt.Errorf("failed to create image generation record: %w", err)
	}

	// Start async generation
	go s.processImageGeneration(context.Background(), gen)

	return gen, nil
}

// processImageGeneration processes image generation in background
func (s *Service) processImageGeneration(ctx context.Context, gen *domain.StoryboardImageGeneration) {
	gen.Status = domain.GenerationStatusProcessing
	_ = s.repo.UpdateImageGeneration(ctx, gen)

	// First, generate image prompt using text AI
	// 直接调用 geminiClient，将结果记录到 StoryboardImageGeneration 表，不创建 AIGenerationRecord
	if s.geminiClient != nil {
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

		text, resp, err := s.geminiClient.GenerateText(ctx, "", promptGen, nil)
		if err != nil {
			gen.Status = domain.GenerationStatusFailed
			gen.ErrorMessage = err.Error()
			_ = s.repo.UpdateImageGeneration(ctx, gen)
			return
		}

		gen.GeneratedPrompt = text
		if resp != nil && resp.UsageMetadata != nil {
			gen.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
			gen.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			gen.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
		}
	} else {
		// If no AI client, use scene description as prompt
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
		} else {
			genReq.Operation = genapi.OperationTextToImage
		}

		// Use configured image provider (default: huoshan)
		imageProvider := s.imageProvider
		if imageProvider == "" {
			imageProvider = "huoshan"
		}

		s.logger.Info("generating scene image",
			zap.String("sceneId", gen.SceneID),
			zap.String("provider", imageProvider))

		resp, err := s.genAPI.GenerateImage(ctx, imageProvider, genReq)
		if err != nil {
			s.logger.Warn("AI image generation failed, keeping prompt only",
				zap.String("sceneId", gen.SceneID),
				zap.String("provider", imageProvider),
				zap.Error(err))
			// Don't fail completely, just mark as completed without image
		} else if resp != nil && len(resp.ImageURLs) > 0 {
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
			// Check for image bytes in metadata (for conversational image generation)
			if imageBytes, ok := resp.Metadata["image_bytes"].([][]byte); ok && len(imageBytes) > 0 {
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
	_ = s.repo.UpdateImageGeneration(ctx, gen)

	s.updateStoryboardTokens(ctx, gen.StoryboardID)

	// Sync generated image URL to storyboard scene
	if gen.GeneratedImageURL != "" && gen.SceneID != "" {
		if err := s.repo.UpdateStoryboardSceneImage(ctx, gen.SceneID, gen.GeneratedImageURL); err != nil {
			s.logger.Warn("failed to sync image to storyboard scene",
				zap.String("sceneId", gen.SceneID),
				zap.Error(err))
		}
	}

	// Update workflow status
	_ = s.repo.UpdateStoryboardWorkflow(ctx, gen.StoryboardID, domain.WorkflowStatusImagesReady, 3)

	s.logger.Info("image generation completed",
		zap.String("storyboardId", gen.StoryboardID),
		zap.String("sceneId", gen.SceneID),
		zap.String("imageURL", gen.GeneratedImageURL))
}

// GenerateSceneVideo generates a video for a scene (Step 4)
func (s *Service) GenerateSceneVideo(ctx context.Context, req *VideoGenerationRequest) (*domain.StoryboardVideoGeneration, error) {
	// Verify storyboard exists
	if _, err := s.repo.StoryboardByID(ctx, req.StoryboardID); err != nil {
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

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

	if err := s.repo.CreateVideoGeneration(ctx, gen); err != nil {
		return nil, fmt.Errorf("failed to create video generation record: %w", err)
	}

	// Start async generation
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
	_ = s.repo.UpdateVideoGeneration(ctx, gen)

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
		gen.GeneratedPrompt = gen.InputDescription
		s.logger.Info("using input description as video prompt (no AI client)",
			zap.String("sceneId", gen.SceneID),
			zap.String("prompt", gen.GeneratedPrompt))
	}

	// Generate actual video using genAPI directly
	// 直接调用 genAPI，将结果记录到 StoryboardVideoGeneration 表，不创建 AIGenerationRecord
	if s.genAPI != nil && gen.GeneratedPrompt != "" {
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
		gen.Status = domain.GenerationStatusCompleted
		now := time.Now().Unix()
		gen.CompletedAt = &now
		if gen.Duration == 0 {
			gen.Duration = 5 // Default duration if not set
		}
		_ = s.repo.UpdateVideoGeneration(ctx, gen)

		s.updateStoryboardTokens(ctx, gen.StoryboardID)

		// Update workflow status
		_ = s.repo.UpdateStoryboardWorkflow(ctx, gen.StoryboardID, domain.WorkflowStatusVideoReady, 4)

		s.logger.Info("video generation process completed",
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("sceneId", gen.SceneID),
			zap.String("status", string(gen.Status)),
			zap.String("videoURL", gen.GeneratedVideoURL),
			zap.String("generatedPrompt", gen.GeneratedPrompt),
			zap.Int("duration", gen.Duration),
			zap.Int("totalTokens", gen.TotalTokens),
			zap.String("errorMessage", gen.ErrorMessage))
	} else {
		// No video URL and no task ID - mark as failed
		gen.Status = domain.GenerationStatusFailed
		gen.ErrorMessage = "video generation failed: no video URL or task ID returned"
		_ = s.repo.UpdateVideoGeneration(ctx, gen)
		s.logger.Warn("video generation process failed - no video URL or task ID",
			zap.String("storyboardId", gen.StoryboardID),
			zap.String("sceneId", gen.SceneID),
			zap.String("status", string(gen.Status)),
			zap.String("errorMessage", gen.ErrorMessage))
	}
}

// GetGenerationProgress returns the complete generation progress for a storyboard
func (s *Service) GetGenerationProgress(ctx context.Context, storyboardID string) (*domain.StoryboardGenerationProgress, error) {
	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return nil, fmt.Errorf("storyboard not found: %w", err)
	}

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
		progress.ContentGeneration = contentGen
		if contentGen.Status == domain.GenerationStatusProcessing {
			isGenerating = true
			statusMessages = append(statusMessages, "正在生成内容")
		} else if contentGen.Status == domain.GenerationStatusPending {
			hasPending = true
			statusMessages = append(statusMessages, "内容生成待处理")
		}
	}

	// Get scene generations
	if sceneGens, err := s.repo.ListSceneGenerations(ctx, storyboardID); err == nil {
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
	}

	// Get image generations
	if imageGens, err := s.repo.ListImageGenerations(ctx, storyboardID); err == nil {
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
	}

	// Get video generations
	if videoGens, err := s.repo.ListVideoGenerations(ctx, storyboardID); err == nil {
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

	return progress, nil
}

// PublishStoryboard publishes a storyboard (Step 5)
func (s *Service) PublishStoryboard(ctx context.Context, storyboardID string) error {
	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return fmt.Errorf("storyboard not found: %w", err)
	}

	// Update workflow status to published using the dedicated workflow update method
	if err := s.repo.UpdateStoryboardWorkflow(ctx, storyboardID, domain.WorkflowStatusPublished, 5); err != nil {
		return fmt.Errorf("failed to publish storyboard: %w", err)
	}

	s.logger.Info("storyboard published",
		zap.String("storyboardId", storyboardID),
		zap.String("storyId", storyboard.StoryID))

	return nil
}

// buildGenerationContext builds comprehensive context string for storyboard generation
func (s *Service) buildGenerationContext(ctx context.Context, storyboard *domain.Storyboard, characterIDs, sceneIDs []string) string {
	var parts []string

	// 1. Get ancestor storyboards (up to 5) along the parent chain
	ancestors := s.getAncestorStoryboards(ctx, storyboard, 5)

	// Get story information for root storyboard
	story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
	if err != nil {
		s.logger.Warn("failed to get story", zap.String("storyId", storyboard.StoryID), zap.Error(err))
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
		parts = append(parts, "\n## Current Storyboard Characters")

		for _, charID := range characterIDs {
			char, err := s.repo.CharacterByID(ctx, charID)
			if err != nil {
				s.logger.Warn("failed to get character", zap.String("characterId", charID), zap.Error(err))
				continue
			}

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
		parts = append(parts, "\n## Storyboard Scenes")

		for _, sceneID := range sceneIDs {
			scene, err := s.repo.StorySceneByID(ctx, storyboard.StoryID, sceneID)
			if err != nil {
				s.logger.Warn("failed to get scene", zap.String("sceneId", sceneID), zap.Error(err))
				continue
			}

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

	return strings.Join(parts, "\n")
}

// updateStoryboardTokens aggregates and updates token consumption
func (s *Service) updateStoryboardTokens(ctx context.Context, storyboardID string) {
	tokens, err := s.repo.GetStoryboardTotalTokens(ctx, storyboardID)
	if err != nil {
		s.logger.Warn("failed to get total tokens", zap.Error(err))
		return
	}
	if err := s.repo.UpdateStoryboardTokens(ctx, storyboardID, tokens); err != nil {
		s.logger.Warn("failed to update storyboard tokens", zap.Error(err))
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
