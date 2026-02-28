package http

import (
	"fmt"
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
	var generations []*domain.StoryboardImageGeneration
	for _, scene := range storyboard.StoryboardScenes {
		// Skip if already has image and not regenerating
		if !req.RegenerateAll && scene.Image != "" {
			h.logger.Debug("skipping scene with existing image",
				zap.String("sceneId", scene.ID))
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

		genReq := &service.ImageGenerationRequest{
			StoryboardID:             storyboardID,
			SceneID:                  scene.ID,
			SceneTitle:               scene.Title,
			SceneDescription:         scene.Description,
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
			continue
		}

		generations = append(generations, gen)
	}

	h.logger.Info("GenerateAllStoryboardImages completed",
		zap.String("storyboardId", storyboardID),
		zap.Int("scenesProcessed", len(generations)))

	Success(c, generations)
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
