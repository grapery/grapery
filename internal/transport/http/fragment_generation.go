package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/handler"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// parseIntParam safely parses a string to int
func parseIntParam(s string) (int, error) {
	return strconv.Atoi(s)
}

type FragmentGenerationHandler struct {
	fragmentGenService *service.FragmentGenerationService
	fragmentHandler    *handler.FragmentHandler
	generationRuntime  *service.GenerationRuntimeService
	logger             *zap.Logger
}

func NewFragmentGenerationHandler(fragmentGenService *service.FragmentGenerationService, fragmentHandler *handler.FragmentHandler, generationRuntime *service.GenerationRuntimeService, logger *zap.Logger) *FragmentGenerationHandler {
	return &FragmentGenerationHandler{
		fragmentGenService: fragmentGenService,
		fragmentHandler:    fragmentHandler,
		generationRuntime:  generationRuntime,
		logger:             logger,
	}
}

func fragmentGenerationHTTPError(err error) (int, string, string) {
	var genErr *domain.FragmentGenerationError
	if errors.As(err, &genErr) {
		switch genErr.Code {
		case domain.FragmentGenerationErrorInvalidRequest:
			return http.StatusBadRequest, genErr.Code, genErr.Message
		case domain.FragmentGenerationErrorForbidden:
			return http.StatusForbidden, genErr.Code, genErr.Message
		case domain.FragmentGenerationErrorNotFound:
			return http.StatusNotFound, genErr.Code, genErr.Message
		case domain.FragmentGenerationErrorConflict, domain.FragmentGenerationErrorCancelled:
			return http.StatusConflict, genErr.Code, genErr.Message
		case domain.FragmentGenerationErrorInsufficient:
			return http.StatusPaymentRequired, genErr.Code, genErr.Message
		default:
			return http.StatusBadRequest, genErr.Code, genErr.Message
		}
	}
	return http.StatusInternalServerError, "internal_error", "failed to create generation task"
}

// GenerateFragmentRequest 生成碎片故事的请求
type GenerateFragmentRequest struct {
	UserInput  string   `json:"userInput" binding:"required,min=1,max=2000"`
	ImageUrls  []string `json:"imageUrls" binding:"max=10"`
	ImageCount int      `json:"imageCount" binding:"min=0,max=8"`
	Style      string   `json:"style" binding:"omitempty,max=64"`
	Mood       string   `json:"mood" binding:"omitempty,oneof=happy sad mysterious romantic"`
	Length     string   `json:"length" binding:"omitempty,oneof=short medium long"`
	Language   string   `json:"language" binding:"required,oneof=zh-Hans en ja"`
	Visibility string   `json:"visibility" binding:"required,oneof=public followers followers_only private"`
	// AspectRatio 配图长宽比；空表示由多模态（有参考图时）推断，否则默认 16:9
	AspectRatio             string                         `json:"aspectRatio" binding:"omitempty,oneof=1:1 16:9 9:16 3:4 4:3"`
	ConsistencyLevel        string                         `json:"consistencyLevel" binding:"omitempty,oneof=off standard strong"`
	EnableReferenceAssets   *bool                          `json:"enableReferenceAssets"`
	IncludeGenerationTrace  bool                           `json:"includeGenerationTrace"`
	ReferenceSlots          []domain.FragmentReferenceSlot `json:"referenceSlots" binding:"max=10"`
	TargetDraftFragmentID   string                         `json:"targetDraftFragmentId"`
	ReplaceImageIndex       int                            `json:"replaceImageIndex" binding:"min=0,max=99"`
	ClientMessageID         string                         `json:"clientMessageId" binding:"omitempty,max=128"`
	WorkflowReleaseID       string                         `json:"workflowReleaseId,omitempty"`
	WorkflowRunID           string                         `json:"workflowRunId,omitempty"`
	WorkflowSystemPrompt    string                         `json:"workflowSystemPrompt,omitempty"`
	WorkflowUserPrompt      string                         `json:"workflowUserPrompt,omitempty"`
	WorkflowModelConfig     map[string]any                 `json:"workflowModelConfig,omitempty"`
	WorkflowOutputSchema    map[string]any                 `json:"workflowOutputSchema,omitempty"`
	WorkflowPromptVersionID string                         `json:"workflowPromptVersionId,omitempty"`
}

// AnalyzeFragment handles POST /fragments/analyze
func (h *FragmentGenerationHandler) AnalyzeFragment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.FragmentAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.fragmentGenService.AnalyzeFragmentStory(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GenerateFragment handles POST /fragments/generate
func (h *FragmentGenerationHandler) GenerateFragment(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req GenerateFragmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.WorkflowReleaseID) == "" || strings.TrimSpace(req.WorkflowRunID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fragment generation must be started by a configured workflow"})
		return
	}
	if h.generationRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generation runtime unavailable"})
		return
	}
	workflowRun, runErr := h.generationRuntime.GetExecution(c.Request.Context(), req.WorkflowRunID)
	if runErr != nil || workflowRun.UserID != userID || workflowRun.WorkflowReleaseID != req.WorkflowReleaseID || workflowRun.Kind != "fragment" {
		c.JSON(http.StatusForbidden, gin.H{"error": "fragment generation workflow execution is invalid"})
		return
	}

	style := strings.TrimSpace(req.Style)
	if style == "" {
		style = "fantasy"
	}

	// 转换为领域模型
	domainReq := domain.FragmentGenerationRequest{
		UserInput:               req.UserInput,
		ImageUrls:               req.ImageUrls,
		ImageCount:              req.ImageCount,
		Style:                   style,
		Mood:                    req.Mood,
		Length:                  req.Length,
		Language:                req.Language,
		Visibility:              domain.NormalizeFragmentVisibility(req.Visibility),
		AspectRatio:             strings.TrimSpace(req.AspectRatio),
		ConsistencyLevel:        strings.TrimSpace(req.ConsistencyLevel),
		EnableReferenceAssets:   req.EnableReferenceAssets,
		IncludeGenerationTrace:  req.IncludeGenerationTrace,
		ReferenceSlots:          normalizeFragmentReferenceSlots(req.ReferenceSlots),
		TargetDraftFragmentID:   strings.TrimSpace(req.TargetDraftFragmentID),
		ReplaceImageIndex:       req.ReplaceImageIndex,
		ClientMessageID:         strings.TrimSpace(req.ClientMessageID),
		WorkflowReleaseID:       strings.TrimSpace(req.WorkflowReleaseID),
		WorkflowRunID:           strings.TrimSpace(req.WorkflowRunID),
		WorkflowSystemPrompt:    strings.TrimSpace(req.WorkflowSystemPrompt),
		WorkflowUserPrompt:      strings.TrimSpace(req.WorkflowUserPrompt),
		WorkflowModelConfig:     req.WorkflowModelConfig,
		WorkflowOutputSchema:    req.WorkflowOutputSchema,
		WorkflowPromptVersionID: strings.TrimSpace(req.WorkflowPromptVersionID),
	}

	// 如果用户没有指定图片数量，默认生成1张；显式数量由后端作为 slot 任务目标，不由 agent 覆盖。
	if domainReq.ImageCount == 0 {
		domainReq.ImageCount = 1
	}
	if domainReq.ImageCount > 8 {
		domainReq.ImageCount = 8
	}
	if domainReq.ReplaceImageIndex > 0 {
		if domainReq.TargetDraftFragmentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "targetDraftFragmentId is required when replaceImageIndex is set"})
			return
		}
		domainReq.ImageCount = 1
	}

	task, draftFragmentID, err := h.fragmentGenService.GenerateFragment(c.Request.Context(), userID, domainReq)
	if err != nil {
		status, code, message := fragmentGenerationHTTPError(err)
		c.JSON(status, gin.H{"error": message, "code": code})
		return
	}

	resp := gin.H{
		"taskId":   task.ID,
		"status":   task.Status,
		"progress": task.Progress,
	}
	if draftFragmentID != "" {
		resp["draftFragmentId"] = draftFragmentID
	}
	c.JSON(http.StatusAccepted, resp)
}

// GetGenerationStatus handles GET /fragments/generate/:taskId
func (h *FragmentGenerationHandler) GetGenerationStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id is required"})
		return
	}

	task, err := h.fragmentGenService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	imageSlots, imageProgress := fragmentGenerationImageSnapshot(task)
	slotMode := fragmentGenerationSlotMode(task)
	response := gin.H{
		"taskId":        task.ID,
		"status":        task.Status,
		"progress":      task.Progress,
		"currentStep":   task.CurrentStep,
		"messageKey":    fragmentGenerationStepMessageKey(task.CurrentStep),
		"stage":         fragmentGenerationStage(task),
		"createdAt":     task.CreatedAt,
		"imageSlots":    imageSlots,
		"slotMode":      slotMode,
		"imageProgress": imageProgress,
		"cost":          fragmentGenerationCostSnapshot(task, strings.TrimSpace(task.Request.TargetDraftFragmentID) != ""),
	}

	if task.Result != nil {
		response["result"] = fragmentGenerationResultResponse(task)
		response["storyText"] = task.Result.Content
		response["imagePlan"] = fragmentGenerationImagePlan(task)
		response["generatedImages"] = task.Result.ImageUrls
		response["chatMessages"] = fragmentGenerationChatMessages(task)
	}

	if task.ErrorMessage != "" {
		response["error"] = task.ErrorMessage
	}

	c.JSON(http.StatusOK, response)
}

// Append generation persists a complete slot snapshot as soon as planning has
// produced placeholders for the existing and new pages. ImageUrls can still be
// empty at that point, so it cannot be used to infer whether slots are full.
func fragmentGenerationSlotMode(task *domain.FragmentGenerationTask) string {
	if task == nil || strings.TrimSpace(task.Request.TargetDraftFragmentID) == "" || task.Request.ReplaceImageIndex > 0 {
		return "delta"
	}
	if task.Result != nil && (len(task.Result.ImageSlots) > 0 || task.Result.ExpectedImageCount > task.Request.ImageCount || len(task.Result.ImageUrls) > task.Request.ImageCount) {
		return "full"
	}
	return "delta"
}

// ListGenerationTasks handles GET /fragments/generate
func (h *FragmentGenerationHandler) ListGenerationTasks(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := parseIntParam(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	tasks, total, err := h.fragmentGenService.ListTasks(c.Request.Context(), userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}

	// Convert tasks to response format
	taskResponses := make([]gin.H, len(tasks))
	for i, task := range tasks {
		imageSlots, imageProgress := fragmentGenerationImageSnapshot(task)
		taskResponses[i] = gin.H{
			"taskId":        task.ID,
			"status":        task.Status,
			"progress":      task.Progress,
			"currentStep":   task.CurrentStep,
			"messageKey":    fragmentGenerationStepMessageKey(task.CurrentStep),
			"createdAt":     task.CreatedAt,
			"imageSlots":    imageSlots,
			"imageProgress": imageProgress,
		}
		if task.Result != nil {
			taskResponses[i]["result"] = fragmentGenerationResultResponse(task)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": taskResponses,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func fragmentGenerationResultResponse(task *domain.FragmentGenerationTask) gin.H {
	imageURLs := task.Result.ImageUrls
	if imageURLs == nil {
		// `result` is created before the first image is available. Keep the HTTP
		// response contract stable for clients polling this legitimate middle state.
		imageURLs = []string{}
	}
	res := gin.H{
		"content":            task.Result.Content,
		"imageUrls":          imageURLs,
		"tokensUsed":         task.Result.TokensUsed,
		"draftFragmentId":    task.Result.DraftFragmentID,
		"expectedImageCount": fragmentGenerationExpectedImageCount(task),
		"imageSlots":         task.Result.ImageSlots,
		"imageProgress":      fragmentGenerationProgressFromResult(task),
		"visualBible":        task.Result.VisualBible,
		"scenePlan":          task.Result.ScenePlan,
		"referenceAssets":    task.Result.ReferenceAssets,
		"consistencyPolicy":  task.Result.ConsistencyPolicy,
		"consistencyIssues":  task.Result.ConsistencyIssues,
		"storyElements":      task.Result.StoryElements,
	}
	if task.Result.AspectRatio != "" {
		res["aspectRatio"] = task.Result.AspectRatio
	}
	if task.Request.IncludeGenerationTrace {
		res["generationTrace"] = task.Result.GenerationTrace
	}
	return res
}

func fragmentGenerationStage(task *domain.FragmentGenerationTask) string {
	switch strings.TrimSpace(task.CurrentStep) {
	case "extracting_elements", "expanding_scenes", "planning_comic_pages":
		return "story"
	case "generating_reference_assets":
		return "style"
	case "generating_images":
		return "images"
	case "checking_consistency", "repairing_consistency":
		return "review"
	case "completed":
		return "completed"
	default:
		if task.Status == "completed" {
			return "completed"
		}
		if task.Status == "failed" || task.Status == "cancelled" {
			return task.Status
		}
		return "preparing"
	}
}

func fragmentGenerationImageSnapshot(task *domain.FragmentGenerationTask) ([]gin.H, gin.H) {
	total := fragmentGenerationExpectedImageCount(task)
	if total < 1 {
		total = 1
	}
	if task.Result != nil && len(task.Result.ImageSlots) > 0 {
		slots := make([]gin.H, 0, len(task.Result.ImageSlots))
		completed := 0
		for _, slot := range task.Result.ImageSlots {
			if slot.Status == "completed" && strings.TrimSpace(slot.ImageURL) != "" {
				completed++
			}
			slots = append(slots, gin.H{
				"id":           slot.ID,
				"taskId":       slot.TaskID,
				"fragmentId":   slot.FragmentID,
				"index":        slot.Index,
				"title":        slot.Title,
				"caption":      slot.Caption,
				"status":       slot.Status,
				"imageUrl":     slot.ImageURL,
				"assetId":      slot.AssetID,
				"errorMessage": slot.ErrorMessage,
			})
		}
		return slots, gin.H{
			"completedCount": completed,
			"totalCount":     maxInt(total, len(task.Result.ImageSlots)),
		}
	}
	var scenes []domain.FragmentScenePlan
	var urls []string
	if task.Result != nil {
		scenes = task.Result.ScenePlan
		urls = task.Result.ImageUrls
		if len(scenes) > total {
			total = len(scenes)
		}
		if len(urls) > total {
			total = len(urls)
		}
	}

	slots := make([]gin.H, 0, total)
	completed := 0
	for i := 0; i < total; i++ {
		imageURL := ""
		title := "第" + strconv.Itoa(i+1) + "页"
		caption := ""
		if i < len(scenes) {
			if strings.TrimSpace(scenes[i].SceneDesc) != "" {
				caption = strings.TrimSpace(scenes[i].SceneDesc)
			}
			if strings.TrimSpace(scenes[i].GeneratedImageURL) != "" {
				imageURL = strings.TrimSpace(scenes[i].GeneratedImageURL)
			}
		}
		if imageURL == "" && i < len(urls) {
			imageURL = strings.TrimSpace(urls[i])
		}
		status := "planned"
		if imageURL != "" {
			status = "completed"
			completed++
		} else if task.Status == "failed" {
			status = "failed"
		} else if task.CurrentStep == "generating_images" && i == completed {
			status = "generating"
		}
		slots = append(slots, gin.H{
			"index":    i + 1,
			"title":    title,
			"caption":  caption,
			"status":   status,
			"imageUrl": imageURL,
		})
	}
	return slots, gin.H{
		"completedCount": completed,
		"totalCount":     total,
	}
}

func fragmentGenerationExpectedImageCount(task *domain.FragmentGenerationTask) int {
	if task == nil {
		return 1
	}
	if task.Result != nil && task.Result.ExpectedImageCount > 0 {
		return task.Result.ExpectedImageCount
	}
	count := task.Request.ImageCount
	if count < 1 {
		count = 1
	}
	if count > 8 {
		count = 8
	}
	return count
}

func fragmentGenerationProgressFromResult(task *domain.FragmentGenerationTask) gin.H {
	if task != nil && task.Result != nil && task.Result.ImageProgress != nil {
		return gin.H{
			"completedCount": task.Result.ImageProgress.CompletedCount,
			"totalCount":     task.Result.ImageProgress.TotalCount,
		}
	}
	_, progress := fragmentGenerationImageSnapshot(task)
	return progress
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func fragmentGenerationImagePlan(task *domain.FragmentGenerationTask) []gin.H {
	if task.Result == nil || len(task.Result.ScenePlan) == 0 {
		return []gin.H{}
	}
	out := make([]gin.H, 0, len(task.Result.ScenePlan))
	for i, scene := range task.Result.ScenePlan {
		index := scene.Index
		if index <= 0 {
			index = i + 1
		}
		out = append(out, gin.H{
			"index":   index,
			"caption": strings.TrimSpace(scene.SceneDesc),
			"status":  "planned",
		})
	}
	return out
}

func fragmentGenerationChatMessages(task *domain.FragmentGenerationTask) []gin.H {
	var out []gin.H
	if task.Result != nil && strings.TrimSpace(task.Result.Content) != "" {
		out = append(out, gin.H{
			"id":   task.ID + ":story",
			"type": "story",
			"text": task.Result.Content,
		})
	}
	if task.Result != nil && len(task.Result.ScenePlan) > 0 {
		lines := make([]string, 0, len(task.Result.ScenePlan))
		for i, scene := range task.Result.ScenePlan {
			caption := strings.TrimSpace(scene.SceneDesc)
			if caption == "" {
				continue
			}
			lines = append(lines, "第"+strconv.Itoa(i+1)+"页："+caption)
		}
		if len(lines) > 0 {
			out = append(out, gin.H{
				"id":   task.ID + ":image_plan",
				"type": "image_plan",
				"text": strings.Join(lines, "\n"),
			})
		}
	}
	return out
}

func fragmentGenerationCostSnapshot(task *domain.FragmentGenerationTask, revision bool) gin.H {
	count := task.Request.ImageCount
	if count < 1 {
		count = 1
	}
	points := count * 8
	label := "本次创作消耗 "
	if revision {
		label = "本次修改消耗 "
	}
	return gin.H{
		"amount": points,
		"unit":   "点数",
		"text":   label + strconv.Itoa(points) + " 点数",
	}
}

func normalizeFragmentReferenceSlots(slots []domain.FragmentReferenceSlot) []domain.FragmentReferenceSlot {
	if len(slots) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]domain.FragmentReferenceSlot, 0, len(slots))
	for _, slot := range slots {
		key := strings.TrimSpace(slot.Key)
		label := strings.TrimSpace(slot.Label)
		kind := strings.TrimSpace(slot.Kind)
		if key == "" || label == "" || kind == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		inputType := strings.TrimSpace(slot.InputType)
		if inputType == "" {
			inputType = "image"
		}
		out = append(out, domain.FragmentReferenceSlot{
			Key:        key,
			Label:      label,
			Kind:       kind,
			Required:   slot.Required,
			InputType:  inputType,
			ImageURL:   strings.TrimSpace(slot.ImageURL),
			HelperText: strings.TrimSpace(slot.HelperText),
		})
	}
	return out
}

func fragmentGenerationStepMessageKey(step string) string {
	switch strings.TrimSpace(step) {
	case "starting":
		return "fragment_generation_starting"
	case "extracting_elements":
		return "fragment_generation_analyzing_story"
	case "expanding_scenes":
		return "fragment_generation_writing_story"
	case "planning_comic_pages":
		return "fragment_generation_planning_pages"
	case "generating_reference_assets":
		return "fragment_generation_designing_style"
	case "generating_images":
		return "fragment_generation_generating_images"
	case "checking_consistency":
		return "fragment_generation_checking_consistency"
	case "repairing_consistency":
		return "fragment_generation_repairing_consistency"
	case "completed":
		return "fragment_generation_completed"
	default:
		return ""
	}
}

// CancelGeneration handles DELETE /fragments/generate/:taskId
func (h *FragmentGenerationHandler) CancelGeneration(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id is required"})
		return
	}

	if err := h.fragmentGenService.CancelTask(c.Request.Context(), taskID, userID); err != nil {
		if err.Error() == "unauthorized: task does not belong to user" {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId": taskID,
		"status": "cancelled",
	})
}

// RegisterRoutes registers the fragment generation routes
func (h *FragmentGenerationHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	router.POST("/analyze", h.AnalyzeFragment)

	fragmentGenGroup := router.Group("/generate")
	if authMiddleware != nil {
		fragmentGenGroup.Use(authMiddleware)
	}
	{
		fragmentGenGroup.POST("", h.GenerateFragment)
		fragmentGenGroup.GET("/:taskId", h.GetGenerationStatus)
		fragmentGenGroup.GET("", h.ListGenerationTasks)
		fragmentGenGroup.DELETE("/:taskId", h.CancelGeneration)
	}
}

// GetFragmentStyles handles GET /fragments/styles
func (h *FragmentGenerationHandler) GetFragmentStyles(c *gin.Context) {
	h.fragmentHandler.GetFragmentStyles(c)
}
