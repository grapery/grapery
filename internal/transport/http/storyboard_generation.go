package http

import (
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
