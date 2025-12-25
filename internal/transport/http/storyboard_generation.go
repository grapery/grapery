package http

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
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
func (h *Handler) GenerateStoryboardImage(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		Error(c, CodeInvalidParams, "storyboard id required")
		return
	}

	var req struct {
		SceneID          string   `json:"sceneId" binding:"required"`
		SceneTitle       string   `json:"sceneTitle"`
		SceneDescription string   `json:"sceneDescription" binding:"required"`
		ReferenceImages  []string `json:"referenceImages"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}

	genReq := &service.ImageGenerationRequest{
		StoryboardID:     storyboardID,
		SceneID:          req.SceneID,
		SceneTitle:       req.SceneTitle,
		SceneDescription: req.SceneDescription,
		ReferenceImages:  req.ReferenceImages,
	}

	gen, err := h.svc.GenerateSceneImage(c.Request.Context(), genReq)
	if err != nil {
		Error(c, CodeInternalError, err.Error())
		return
	}

	Success(c, gen)
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
