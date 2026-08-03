package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// GenerateContent starts content generation for a storyboard (Step 1)
// POST /api/storyboards/:id/generate/content
func (h *Handler) GenerateContent(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var req struct {
		RawInput     string   `json:"rawInput" binding:"required"`
		CharacterIDs []string `json:"characterIds"`
		SceneIDs     []string `json:"sceneIds"`
		Style        string   `json:"style"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}

	// Default style
	if req.Style == "" {
		req.Style = "drama"
	}

	genReq := &service.ContentGenerationRequest{
		StoryboardID: storyboardID,
		RawInput:     req.RawInput,
		CharacterIDs: req.CharacterIDs,
		SceneIDs:     req.SceneIDs,
		Style:        req.Style,
	}

	gen, err := h.svc.StartContentGeneration(c.Request.Context(), genReq)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, gen)
}

// GenerateStoryboardStructure reruns bible + scene-plan pipeline and persists scenes when DB has zero scenes.
// When scenes already exist, returns the storyboard synchronously (asyncAccepted=false).
// Otherwise accepts immediately (asyncAccepted=true) while work continues in-process; clients poll GET .../generation-progress.
// POST /api/storyboards/:id/generate/structure
func (h *Handler) GenerateStoryboardStructure(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	// Body is optional: legacy callers resume a stalled draft with no payload, while the
	// conversational flow sends this turn's revision instruction.
	var req struct {
		UserDirective string `json:"userDirective"`
		SceneCount    int    `json:"sceneCount"`
		ComicStyle    string `json:"comicStyle"`
	}
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			InvalidParams(c, err.Error())
			return
		}
	}

	resp, err := h.svc.StartStoryboardStructureGeneration(c.Request.Context(), userID, storyboardID,
		service.StoryboardStructureGenerationOptions{
			UserDirective: req.UserDirective,
			SceneCount:    req.SceneCount,
			ComicStyle:    req.ComicStyle,
		})
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, resp)
}

// ExecuteStoryboardWorkflowStage runs one idempotent, durable text-generation
// stage. Workflow runtimes checkpoint the small result and keep prompt/output
// snapshots inside Grapery.
// POST /api/v1/storyboards/:id/generate/stages/:stage
func (h *Handler) ExecuteStoryboardWorkflowStage(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	storyboardID := strings.TrimSpace(c.Param("id"))
	stage := strings.TrimSpace(c.Param("stage"))
	if storyboardID == "" || stage == "" {
		InvalidParams(c, "storyboard id and stage are required")
		return
	}
	var req struct {
		GenerationRunID     string `json:"generationRunId"`
		ClientRequestID     string `json:"clientRequestId"`
		RegenerateStructure bool   `json:"regenerateStructure"`
		UserDirective       string `json:"userDirective"`
		SceneCount          int    `json:"sceneCount"`
		ComicStyle          string `json:"comicStyle"`
	}
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			InvalidParams(c, err.Error())
			return
		}
	}
	result, err := h.svc.ExecuteStoryboardWorkflowStage(c.Request.Context(), userID, storyboardID, stage, service.StoryboardWorkflowStageOptions{
		GenerationRunID: req.GenerationRunID, ClientRequestID: req.ClientRequestID, RegenerateStructure: req.RegenerateStructure,
		UserDirective: req.UserDirective, SceneCount: req.SceneCount, ComicStyle: req.ComicStyle,
	})
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, result)
}

// GenerateSceneDetails generates detailed scene descriptions (Step 2)
// POST /api/storyboards/:id/generate/scene-details
func (h *Handler) GenerateSceneDetails(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var req struct {
		SceneID          string `json:"sceneId" binding:"required"`
		SceneTitle       string `json:"sceneTitle"`
		SceneLocation    string `json:"sceneLocation"`
		InputDescription string `json:"inputDescription" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}

	genReq := &service.SceneDetailRequest{
		StoryboardID:     storyboardID,
		SceneID:          req.SceneID,
		SceneTitle:       req.SceneTitle,
		SceneLocation:    req.SceneLocation,
		InputDescription: req.InputDescription,
	}

	gen, err := h.svc.GenerateSceneDetails(c.Request.Context(), genReq)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, gen)
}

// GenerateStoryboardImage generates an image for a scene (Step 3)
// POST /api/storyboards/:id/generate/image
//
// Request body:
//   - sceneId (required): 场景ID
//   - sceneTitle: 场景标题
//   - sceneDescription (required): 场景描述
//   - referenceImages: 额外的参考图片URL列表（可选）
//   - sceneCharacters: 场景中出现的角色名称列表（可选，用于自动获取角色参考图片）
//   - characterReferenceImages: 角色参考图片URL列表（可选，如果不提供则自动从故事中获取）
//   - storyStyleId: 故事风格配置ID（可选，如果不提供则自动从故事中获取）
//
// 注意：
//   - 如果场景有角色出现，系统会自动使用角色的海报/头像作为参考图片
//   - 如果场景是过渡场景（无角色），则只使用故事风格配置
//   - 故事风格配置会自动从故事中获取，除非明确指定 storyStyleId
func (h *Handler) GenerateStoryboardImage(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var req struct {
		SceneID                  string   `json:"sceneId" binding:"required"`
		SceneTitle               string   `json:"sceneTitle"`
		SceneDescription         string   `json:"sceneDescription" binding:"required"`
		ReferenceImages          []string `json:"referenceImages"`
		SceneCharacters          []string `json:"sceneCharacters"`          // 场景中的角色名称列表
		CharacterReferenceImages []string `json:"characterReferenceImages"` // 角色参考图片（可选）
		StoryStyleID             string   `json:"storyStyleId"`             // 故事风格配置ID（可选）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}

	// 如果提供了 storyStyleId，获取风格配置
	// 暂时不实现，因为系统会自动从故事中获取风格配置
	// 客户端可以通过 storyStyleId 来指定特定的风格，但目前系统会优先使用故事默认风格

	genReq := &service.ImageGenerationRequest{
		StoryboardID:             storyboardID,
		SceneID:                  req.SceneID,
		SceneTitle:               req.SceneTitle,
		SceneDescription:         req.SceneDescription,
		ReferenceImages:          req.ReferenceImages,
		SceneCharacters:          req.SceneCharacters,
		CharacterReferenceImages: req.CharacterReferenceImages,
		// StoryStyle 会在 service 层自动从故事中获取
	}

	gen, err := h.svc.GenerateSceneImage(c.Request.Context(), genReq)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, gen)
}

// GenerateStoryboardComicPage generates a multi-panel comic PAGE (single image file) for one scene.
// POST /api/storyboards/:id/generate/comic-page
//
// Separate pipeline from GenerateStoryboardImage: different prompt + default aspect ratio (9:16).
func (h *Handler) GenerateStoryboardComicPage(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var req struct {
		SceneID                  string   `json:"sceneId" binding:"required"`
		SceneTitle               string   `json:"sceneTitle"`
		SceneDescription         string   `json:"sceneDescription" binding:"required"`
		ReferenceImages          []string `json:"referenceImages"`
		SceneCharacters          []string `json:"sceneCharacters"`
		CharacterReferenceImages []string `json:"characterReferenceImages"`
		LayoutPreset             string   `json:"layoutPreset"`
		PanelCount               int      `json:"panelCount"`
		PageAspectRatio          string   `json:"pageAspectRatio"`
		DialogueMode             string   `json:"dialogueMode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}

	genReq := &service.ComicPageGenerationRequest{
		StoryboardID:             storyboardID,
		SceneID:                  req.SceneID,
		SceneTitle:               req.SceneTitle,
		SceneDescription:         req.SceneDescription,
		ReferenceImages:          req.ReferenceImages,
		SceneCharacters:          req.SceneCharacters,
		CharacterReferenceImages: req.CharacterReferenceImages,
		Pipeline: service.ComicPagePipelineOptions{
			LayoutPreset:    req.LayoutPreset,
			PanelCount:      req.PanelCount,
			PageAspectRatio: req.PageAspectRatio,
			DialogueMode:    req.DialogueMode,
		},
	}

	gen, err := h.svc.GenerateStoryboardComicPage(c.Request.Context(), genReq)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, gen)
}

// GenerateAllStoryboardComicPages batch comic-page generation for all scenes.
// POST /api/storyboards/:id/generate/comic-pages
func (h *Handler) GenerateAllStoryboardComicPages(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var req struct {
		RegenerateAll   bool   `json:"regenerateAll"`
		LayoutPreset    string `json:"layoutPreset"`
		PanelCount      int    `json:"panelCount"`
		PageAspectRatio string `json:"pageAspectRatio"`
		DialogueMode    string `json:"dialogueMode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req = struct {
			RegenerateAll   bool   `json:"regenerateAll"`
			LayoutPreset    string `json:"layoutPreset"`
			PanelCount      int    `json:"panelCount"`
			PageAspectRatio string `json:"pageAspectRatio"`
			DialogueMode    string `json:"dialogueMode"`
		}{}
	}

	pipeline := service.ComicPagePipelineOptions{
		LayoutPreset:    req.LayoutPreset,
		PanelCount:      req.PanelCount,
		PageAspectRatio: req.PageAspectRatio,
		DialogueMode:    req.DialogueMode,
	}
	service.NormalizeComicPagePipeline(&pipeline)

	storyboard, err := h.svc.GetStoryboard(c.Request.Context(), storyboardID)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}
	if len(storyboard.StoryboardScenes) == 0 {
		Error(c, CodeInvalidParams, "no scenes found in storyboard")
		return
	}

	latestComicPageByScene := map[string]*domain.StoryboardImageGeneration{}
	if progress, progressErr := h.svc.GetGenerationProgress(c.Request.Context(), storyboardID); progressErr == nil && progress != nil {
		for _, gen := range progress.ImageGenerations {
			if gen == nil {
				continue
			}
			if strings.TrimSpace(gen.PipelineKind) != domain.StoryboardImagePipelineComicPage {
				continue
			}
			if gen.Status != domain.GenerationStatusCompleted || strings.TrimSpace(gen.GeneratedImageURL) == "" {
				continue
			}
			latestComicPageByScene[gen.SceneID] = gen
		}
	}

	type sceneBatchResult struct {
		SceneID          string                            `json:"sceneId"`
		SceneTitle       string                            `json:"sceneTitle"`
		Status           string                            `json:"status"`
		ErrorMessage     string                            `json:"errorMessage,omitempty"`
		Generation       *domain.StoryboardImageGeneration `json:"generation,omitempty"`
		ExistingImageURL string                            `json:"existingImageUrl,omitempty"`
	}

	results := make([]sceneBatchResult, 0, len(storyboard.StoryboardScenes))
	successCount := 0
	failedCount := 0

	for _, scene := range storyboard.StoryboardScenes {
		if !req.RegenerateAll && scene.Image != "" {
			existingComicPage := latestComicPageByScene[scene.ID]
			if existingComicPage != nil && strings.TrimSpace(existingComicPage.GeneratedImageURL) == strings.TrimSpace(scene.Image) {
				results = append(results, sceneBatchResult{
					SceneID:          scene.ID,
					SceneTitle:       scene.Title,
					Status:           "success",
					ExistingImageURL: scene.Image,
				})
				successCount++
				continue
			}
			// `scene.Image` may be an older single-panel illustration. The comic-page endpoint
			// must not report that as success, otherwise the product switch still appears to
			// generate a single image instead of one image containing multiple panels.
		}

		characterPortraitMap := make(map[string]string)
		for _, charRef := range storyboard.CharacterRefs {
			if charRef.Character != nil && charRef.Character.Portrait != "" {
				characterPortraitMap[charRef.Character.Name] = charRef.Character.Portrait
			}
		}
		var referenceImages []string
		for _, charName := range scene.Characters {
			if portrait, ok := characterPortraitMap[charName]; ok {
				referenceImages = append(referenceImages, portrait)
			}
		}

		mergedDesc := h.svc.MergedStoryboardSceneDescriptionForImage(c.Request.Context(), storyboardID, scene.ID, scene.Description)
		genReq := &service.ComicPageGenerationRequest{
			StoryboardID:             storyboardID,
			SceneID:                  scene.ID,
			SceneTitle:               scene.Title,
			SceneDescription:         mergedDesc,
			ReferenceImages:          referenceImages,
			SceneCharacters:          scene.Characters,
			CharacterReferenceImages: referenceImages,
			Pipeline:                 pipeline,
		}

		gen, genErr := h.svc.GenerateStoryboardComicPage(c.Request.Context(), genReq)
		if genErr != nil {
			results = append(results, sceneBatchResult{
				SceneID:      scene.ID,
				SceneTitle:   scene.Title,
				Status:       "failed",
				ErrorMessage: genErr.Error(),
			})
			failedCount++
			continue
		}
		results = append(results, sceneBatchResult{
			SceneID:    scene.ID,
			SceneTitle: scene.Title,
			Status:     "success",
			Generation: gen,
		})
		successCount++
	}

	Success(c, gin.H{
		"results":      results,
		"total":        len(results),
		"successCount": successCount,
		"failedCount":  failedCount,
	})
}

// GenerateAllStoryboardImages generates images for all scenes (Batch operation)
// POST /api/storyboards/:id/generate/images
//
// Request body:
//   - regenerateAll (optional): If true, regenerate images even if they exist
//   - storyStyleId (optional): Story style configuration ID
//
// Response: Array of image generation records
func (h *Handler) GenerateAllStoryboardImages(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var req struct {
		RegenerateAll bool   `json:"regenerateAll"`
		StoryStyleID  string `json:"storyStyleId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to empty request (regenerateAll = false)
		req = struct {
			RegenerateAll bool   `json:"regenerateAll"`
			StoryStyleID  string `json:"storyStyleId"`
		}{}
	}

	h.logger.Info("GenerateAllStoryboardImages called",
		zap.String("storyboardId", storyboardID),
		zap.Bool("regenerateAll", req.RegenerateAll))

	// Get storyboard to access scenes
	storyboard, err := h.svc.GetStoryboard(c.Request.Context(), storyboardID)
	if err != nil {
		h.logger.Error("failed to get storyboard for batch image generation",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		Error(c, CodeInternalError, err.Error())
		return
	}

	if len(storyboard.StoryboardScenes) == 0 {
		Error(c, CodeInvalidParams, "no scenes found in storyboard")
		return
	}

	// Generate images for all scenes
	type sceneBatchResult struct {
		SceneID          string                            `json:"sceneId"`
		SceneTitle       string                            `json:"sceneTitle"`
		Status           string                            `json:"status"`
		ErrorMessage     string                            `json:"errorMessage,omitempty"`
		Generation       *domain.StoryboardImageGeneration `json:"generation,omitempty"`
		ExistingImageURL string                            `json:"existingImageUrl,omitempty"`
	}

	results := make([]sceneBatchResult, 0, len(storyboard.StoryboardScenes))
	successCount := 0
	failedCount := 0
	for _, scene := range storyboard.StoryboardScenes {
		// Skip if already has image and not regenerating
		if !req.RegenerateAll && scene.Image != "" {
			h.logger.Debug("skipping scene with existing image",
				zap.String("sceneId", scene.ID))
			results = append(results, sceneBatchResult{
				SceneID:          scene.ID,
				SceneTitle:       scene.Title,
				Status:           "success",
				ExistingImageURL: scene.Image,
			})
			successCount++
			continue
		}

		// Build character reference images for this scene
		// First collect all character portraits from storyboard character refs
		characterPortraitMap := make(map[string]string)
		for _, charRef := range storyboard.CharacterRefs {
			if charRef.Character != nil && charRef.Character.Portrait != "" {
				characterPortraitMap[charRef.Character.Name] = charRef.Character.Portrait
			}
		}

		// Match scene characters with portraits
		var referenceImages []string
		for _, charName := range scene.Characters {
			if portrait, ok := characterPortraitMap[charName]; ok {
				referenceImages = append(referenceImages, portrait)
			}
		}

		mergedDesc := h.svc.MergedStoryboardSceneDescriptionForImage(c.Request.Context(), storyboardID, scene.ID, scene.Description)
		genReq := &service.ImageGenerationRequest{
			StoryboardID:             storyboardID,
			SceneID:                  scene.ID,
			SceneTitle:               scene.Title,
			SceneDescription:         mergedDesc,
			ReferenceImages:          referenceImages,
			SceneCharacters:          scene.Characters,
			CharacterReferenceImages: referenceImages,
			// StoryStyle 由服务层自动从故事中获取
		}

		gen, genErr := h.svc.GenerateSceneImage(c.Request.Context(), genReq)
		if genErr != nil {
			h.logger.Warn("failed to generate image for scene",
				zap.String("sceneId", scene.ID),
				zap.Error(genErr))
			results = append(results, sceneBatchResult{
				SceneID:      scene.ID,
				SceneTitle:   scene.Title,
				Status:       "failed",
				ErrorMessage: genErr.Error(),
			})
			failedCount++
			continue
		}

		results = append(results, sceneBatchResult{
			SceneID:    scene.ID,
			SceneTitle: scene.Title,
			Status:     "success",
			Generation: gen,
		})
		successCount++
	}

	h.logger.Info("GenerateAllStoryboardImages completed",
		zap.String("storyboardId", storyboardID),
		zap.Int("scenesProcessed", len(results)),
		zap.Int("successCount", successCount),
		zap.Int("failedCount", failedCount))

	Success(c, gin.H{
		"results":      results,
		"total":        len(results),
		"successCount": successCount,
		"failedCount":  failedCount,
	})
}

// GenerateStoryboardVideo generates a video for a scene (Step 4)
// POST /api/storyboards/:id/generate/video
func (h *Handler) GenerateStoryboardVideo(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var req struct {
		SceneID           string `json:"sceneId" binding:"required"`
		SceneTitle        string `json:"sceneTitle"`
		InputDescription  string `json:"inputDescription" binding:"required"`
		ReferenceImageURL string `json:"referenceImageUrl"` // Start keyframe image
		EndFrameURL       string `json:"endFrameUrl"`       // End keyframe image for keyframe video generation
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}

	genReq := &service.VideoGenerationRequest{
		StoryboardID:      storyboardID,
		SceneID:           req.SceneID,
		SceneTitle:        req.SceneTitle,
		InputDescription:  req.InputDescription,
		ReferenceImageURL: req.ReferenceImageURL,
		EndFrameURL:       req.EndFrameURL,
	}

	gen, err := h.svc.GenerateSceneVideo(c.Request.Context(), genReq)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, gen)
}

// GetGenerationProgress returns all generation records for a storyboard
// GET /api/storyboards/:id/generation-progress
func (h *Handler) GetGenerationProgress(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	progress, err := h.svc.GetGenerationProgress(c.Request.Context(), storyboardID)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, progress)
}

// RetryFailedStoryboardImages retries failed image generations for a storyboard.
// POST /api/storyboards/:id/retry-failed-images
func (h *Handler) RetryFailedStoryboardImages(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var opts service.RetryFailedStoryboardImageOptions
	raw, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		Error(c, CodeInvalidParams, readErr.Error())
		return
	}
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &opts); err != nil {
			Error(c, CodeInvalidParams, err.Error())
			return
		}
	}

	retried, remainingFailed, err := h.svc.RetryFailedStoryboardImages(c.Request.Context(), storyboardID, &opts)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, gin.H{
		"storyboardId":    storyboardID,
		"retriedCount":    retried,
		"remainingFailed": remainingFailed,
	})
}

// CancelStoryboardGeneration cancels all pending/processing generation tasks for a storyboard.
// POST /api/storyboards/:id/cancel-generation
func (h *Handler) CancelStoryboardGeneration(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	userIDValue, ok := c.Get("userID")
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}
	userID := userIDValue.(string)

	cancelledCount, err := h.svc.CancelStoryboardGeneration(c.Request.Context(), storyboardID, userID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "permission denied") {
			Error(c, CodeForbidden, err.Error())
			return
		}
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, gin.H{
		"storyboardId":   storyboardID,
		"cancelledCount": cancelledCount,
	})
}

// PublishStoryboard publishes a storyboard (Step 5)
// POST /api/storyboards/:id/publish
func (h *Handler) PublishStoryboard(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	h.logger.Info("PublishStoryboard called",
		zap.String("storyboardId", storyboardID))

	// Get storyboard first to log its storyId
	storyboard, err := h.svc.GetStoryboard(c.Request.Context(), storyboardID)
	if err != nil {
		h.logger.Error("PublishStoryboard: failed to get storyboard",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		Error(c, CodeInternalError, err.Error())
		return
	}

	h.logger.Info("PublishStoryboard: storyboard details",
		zap.String("storyboardId", storyboardID),
		zap.String("storyId", storyboard.StoryID),
		zap.String("parentId", storyboard.ParentID),
		zap.String("currentStatus", storyboard.WorkflowStatus))

	if err := h.svc.PublishStoryboard(c.Request.Context(), storyboardID); err != nil {
		h.logger.Error("PublishStoryboard failed",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		Error(c, CodeInternalError, err.Error())
		return
	}

	h.logger.Info("PublishStoryboard success",
		zap.String("storyboardId", storyboardID),
		zap.String("storyId", storyboard.StoryID))

	Success(c, gin.H{"message": "storyboard published successfully"})
}

// GetSceneVideoPlaylist returns an HLS playlist (.m3u8) for a scene's video segments
// GET /api/storyboards/:id/scenes/:sceneId/playlist.m3u8
func (h *Handler) GetSceneVideoPlaylist(c *gin.Context) {
	storyboardID := c.Param("id")
	sceneID := c.Param("sceneId")

	if storyboardID == "" || sceneID == "" {
		c.String(400, "#EXTM3U\n#EXT-X-ERROR:Missing storyboard or scene ID")
		return
	}

	// Get video generation for this scene
	videoGen, err := h.svc.GetVideoGenerationBySceneID(c.Request.Context(), storyboardID, sceneID)
	if err != nil {
		h.logger.Warn("GetSceneVideoPlaylist: failed to get video generation",
			zap.String("storyboardId", storyboardID),
			zap.String("sceneId", sceneID),
			zap.Error(err))
		c.String(404, "#EXTM3U\n#EXT-X-ERROR:Video not found")
		return
	}

	// Build HLS playlist
	playlist := h.buildHLSPlaylist(videoGen)

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "no-cache")
	c.String(200, playlist)
}

// GetStoryboardVideoPlaylist returns an HLS playlist for all scenes in a storyboard
// GET /api/storyboards/:id/playlist.m3u8
func (h *Handler) GetStoryboardVideoPlaylist(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		c.String(400, "#EXTM3U\n#EXT-X-ERROR:Missing storyboard ID")
		return
	}

	// Get all video generations for this storyboard
	videoGens, err := h.svc.GetVideoGenerationsByStoryboardID(c.Request.Context(), storyboardID)
	if err != nil {
		h.logger.Warn("GetStoryboardVideoPlaylist: failed to get video generations",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		c.String(404, "#EXTM3U\n#EXT-X-ERROR:Videos not found")
		return
	}

	// Build combined HLS playlist from all scenes
	playlist := h.buildCombinedHLSPlaylist(videoGens)

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "no-cache")
	c.String(200, playlist)
}

// buildHLSPlaylist creates an HLS m3u8 playlist from video generation data
func (h *Handler) buildHLSPlaylist(videoGen *service.VideoGenerationInfo) string {
	var sb strings.Builder

	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:3\n")
	sb.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")

	// Check if we have video segments (from subdivision)
	if len(videoGen.VideoSegments) > 0 {
		// Calculate max duration for target duration
		maxDuration := 0
		for _, seg := range videoGen.VideoSegments {
			if seg.DurationSecs > maxDuration {
				maxDuration = seg.DurationSecs
			}
		}
		if maxDuration == 0 {
			maxDuration = 5 // Default 5 seconds
		}

		sb.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", maxDuration))
		sb.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")

		// Add each segment
		for _, seg := range videoGen.VideoSegments {
			duration := seg.DurationSecs
			if duration == 0 {
				duration = 5
			}
			sb.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", float64(duration)))
			sb.WriteString(seg.VideoURL + "\n")
		}
	} else if videoGen.GeneratedVideoURL != "" {
		// Single video fallback
		duration := videoGen.Duration
		if duration == 0 {
			duration = 5
		}
		sb.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", duration))
		sb.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
		sb.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", float64(duration)))
		sb.WriteString(videoGen.GeneratedVideoURL + "\n")
	}

	sb.WriteString("#EXT-X-ENDLIST\n")
	return sb.String()
}

// buildCombinedHLSPlaylist creates an HLS playlist from multiple video generations
func (h *Handler) buildCombinedHLSPlaylist(videoGens []*service.VideoGenerationInfo) string {
	var sb strings.Builder

	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:3\n")
	sb.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")

	// Calculate max segment duration
	maxDuration := 5
	for _, gen := range videoGens {
		if len(gen.VideoSegments) > 0 {
			for _, seg := range gen.VideoSegments {
				if seg.DurationSecs > maxDuration {
					maxDuration = seg.DurationSecs
				}
			}
		} else if gen.Duration > maxDuration {
			maxDuration = gen.Duration
		}
	}

	sb.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", maxDuration))
	sb.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")

	// Add segments from all video generations in order
	for _, gen := range videoGens {
		if len(gen.VideoSegments) > 0 {
			// Add all segments from subdivision
			for _, seg := range gen.VideoSegments {
				duration := seg.DurationSecs
				if duration == 0 {
					duration = 5
				}
				sb.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", float64(duration)))
				sb.WriteString(seg.VideoURL + "\n")
			}
		} else if gen.GeneratedVideoURL != "" {
			// Single video
			duration := gen.Duration
			if duration == 0 {
				duration = 5
			}
			sb.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", float64(duration)))
			sb.WriteString(gen.GeneratedVideoURL + "\n")
		}
	}

	sb.WriteString("#EXT-X-ENDLIST\n")
	return sb.String()
}

// AnalyzeStoryboard handles POST /api/v1/storyboards/analyze
func (h *Handler) AnalyzeStoryboard(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	var req domain.StoryboardAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	resp, err := h.svc.AnalyzeStoryboardDirection(c.Request.Context(), userID, req)
	if err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	Success(c, resp)
}

type syncStoryboardConversationRequest struct {
	Messages []syncStoryboardConversationMessage `json:"messages" binding:"required,min=1,dive"`
}

type syncStoryboardConversationMessage struct {
	ClientMessageID string `json:"clientMessageId" binding:"required,max=64"`
	Role            string `json:"role" binding:"required,oneof=user assistant status"`
	Type            string `json:"type" binding:"omitempty,max=40"`
	Text            string `json:"text" binding:"required,max=8000"`
	TaskID          string `json:"taskId" binding:"omitempty,max=36"`
	CreatedAt       int64  `json:"createdAt"`
}

// GetStoryboardConversation handles GET /api/v1/storyboards/:id/conversation (creator only).
func (h *Handler) GetStoryboardConversation(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	storyboardID := strings.TrimSpace(c.Param("id"))
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}
	sb, err := h.svc.GetStoryboard(c.Request.Context(), storyboardID)
	if err != nil || sb == nil {
		HandleError(c, err)
		return
	}
	if sb.UserID != userID {
		Error(c, CodeForbidden, "forbidden")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	var beforeCreatedAt int64
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			beforeCreatedAt = n
		}
	}
	messages, hasMore, err := h.svc.ListStoryboardConversationPage(c.Request.Context(), storyboardID, limit, beforeCreatedAt)
	if err != nil {
		Error(c, CodeInternalError, "failed to load conversation")
		return
	}
	out := make([]gin.H, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		out = append(out, gin.H{
			"id":              msg.ID,
			"clientMessageId": msg.ClientMessageID,
			"role":            msg.Role,
			"type":            msg.MessageType,
			"text":            msg.Text,
			"taskId":          msg.TaskID,
			"createdAt":       msg.CreatedAt,
		})
	}
	Success(c, gin.H{
		"storyboardId": storyboardID,
		"messages":     out,
		"hasMore":      hasMore,
	})
}

// SyncStoryboardConversationMessages handles PUT /api/v1/storyboards/:id/conversation/messages (creator only).
func (h *Handler) SyncStoryboardConversationMessages(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	storyboardID := strings.TrimSpace(c.Param("id"))
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}
	var req syncStoryboardConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	sb, err := h.svc.GetStoryboard(c.Request.Context(), storyboardID)
	if err != nil || sb == nil {
		HandleError(c, err)
		return
	}
	if sb.UserID != userID {
		Error(c, CodeForbidden, "forbidden")
		return
	}
	toSave := make([]*domain.StoryboardConversationMessage, 0, len(req.Messages))
	for _, item := range req.Messages {
		msgType := strings.TrimSpace(item.Type)
		if msgType == "" {
			msgType = domain.StoryboardConversationTypeStatus
		}
		toSave = append(toSave, &domain.StoryboardConversationMessage{
			StoryboardID:    storyboardID,
			UserID:          userID,
			Role:            strings.TrimSpace(item.Role),
			MessageType:     msgType,
			Text:            strings.TrimSpace(item.Text),
			TaskID:          strings.TrimSpace(item.TaskID),
			ClientMessageID: strings.TrimSpace(item.ClientMessageID),
			CreatedAt:       item.CreatedAt,
		})
	}
	if err := h.svc.SyncStoryboardConversationMessages(c.Request.Context(), toSave); err != nil {
		Error(c, CodeInternalError, "failed to sync conversation")
		return
	}
	Success(c, gin.H{"ok": true, "synced": len(toSave)})
}
