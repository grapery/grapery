package http

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

type characterRefPayload struct {
	CharacterID string `json:"characterId" binding:"required"`
	Role        string `json:"role"`
	Order       *int   `json:"order"`
	Notes       string `json:"notes"`
}

type sceneRefPayload struct {
	StorySceneID   string `json:"storySceneId" binding:"required"`
	Sequence       *int   `json:"sequence"`
	IsPrimaryScene bool   `json:"isPrimaryScene"`
}

type continueStoryboardPayload struct {
	RawInput          string   `json:"rawInput" binding:"required"`
	SceneCount        int      `json:"sceneCount"`
	Characters        []string `json:"characters,omitempty"` // Optional: specific character IDs to include
	GenerateVideo     bool     `json:"generateVideo"`        // 默认 false：仅生图；true 时生图后再基于图生视频
	ComicStyle        string   `json:"comicStyle,omitempty"` // 漫画风格 slug，写入故事板
	WorkflowReleaseID string   `json:"workflowReleaseId,omitempty"`
	WorkflowRunID     string   `json:"workflowRunId,omitempty"`
}

// truncateForLog truncates for logging; maxLen is bytes, adjusted to a UTF-8 boundary.
func truncateForLog(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	truncated := s[:maxLen]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated + "..."
}

// CreateStoryboard 创建 storyboard
func (h *Handler) CreateStoryboard(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(string)

	var req struct {
		StoryID              string                `json:"storyId" binding:"required"`
		ParentID             *string               `json:"parentId"`
		Title                string                `json:"title"`
		RawInput             string                `json:"rawInput" binding:"required"`
		Content              string                `json:"content"`
		IsStandalone         bool                  `json:"isStandalone"` // Independent plot, AI won't reference parent context
		SceneCount           int                   `json:"sceneCount"`   // Requested number of scenes to generate (2-8, default 3)
		SceneRefs            []sceneRefPayload     `json:"sceneRefs"`    // References to story-level scenes (static locations)
		CharacterRefs        []characterRefPayload `json:"characterRefs"`
		Tags                 []string              `json:"tags" binding:"omitempty,max=3,dive,min=1,max=50"` // 最多3个标签，每个标签1-50字符
		UseComicPagePipeline bool                  `json:"useComicPagePipeline"`
		ComicStyle           string                `json:"comicStyle"` // 漫画风格 slug，写入故事板并进入生成提示词
		WorkflowReleaseID    string                `json:"workflowReleaseId"`
		WorkflowRunID        string                `json:"workflowRunId"`
		ChapterContent       string                `json:"chapterContent"`
		StoryContent         string                `json:"storyContent"`
		ParentEnding         string                `json:"parentEnding"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	if strings.TrimSpace(req.WorkflowReleaseID) == "" || strings.TrimSpace(req.WorkflowRunID) == "" {
		InvalidParams(c, "storyboard generation must be started by a configured workflow")
		return
	}
	if h.generationRuntime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	workflowRun, runErr := h.generationRuntime.GetExecution(c.Request.Context(), req.WorkflowRunID)
	if runErr != nil || workflowRun.UserID != userID.(string) || workflowRun.WorkflowReleaseID != req.WorkflowReleaseID || workflowRun.Kind != "storyboard" {
		Forbidden(c, "storyboard generation workflow execution is invalid")
		return
	}
	h.logger.Info("CreateStoryboard called",
		zap.String("storyId", req.StoryID),
		zap.String("title", req.Title),
		zap.String("userId", uid))

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))

	sceneCount, err := service.NormalizeStoryboardSceneCount(req.SceneCount)
	if err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// Set parentID from request
	parentID := ""
	if req.ParentID != nil {
		parentID = strings.TrimSpace(*req.ParentID)
	}
	if parentID == "root" {
		parentID = domain.StoryboardRootMarker
	}

	// Root creation and continuation are separate capabilities. Only the Story
	// owner may start a root; a real parent uses the parent-specific contract.
	if parentID == "" || parentID == domain.StoryboardRootMarker {
		canCreate, permissionErr := h.svc.CanCreateStoryboard(c.Request.Context(), req.StoryID, uid)
		if permissionErr != nil {
			HandleError(c, permissionErr)
			return
		}
		if !canCreate {
			Forbidden(c, "root_storyboard_author_only")
			return
		}
	} else {
		permission, permissionErr := h.svc.CanForkStoryboard(c.Request.Context(), parentID, uid)
		if permissionErr != nil {
			HandleError(c, permissionErr)
			return
		}
		if !permission.Allowed {
			Forbidden(c, permission.Reason)
			return
		}
	}

	storyboard := &domain.Storyboard{
		StoryID:                req.StoryID,
		ParentID:               parentID,
		UserID:                 userID.(string),
		Title:                  req.Title,
		RawInput:               req.RawInput,
		Content:                req.Content,
		IsStandalone:           req.IsStandalone,
		SceneCount:             sceneCount,
		UseComicPagePipeline:   req.UseComicPagePipeline,
		ContinuationComicStyle: strings.TrimSpace(req.ComicStyle),
		// StoryboardScenes are generated by AI, not passed in request
	}

	if h.workflowRegistry == nil {
		InternalError(c, "workflow registry unavailable")
		return
	}
	routingInput := map[string]any{
		"storyId": req.StoryID, "rawInput": req.RawInput, "chapterContent": firstNonEmptyRoutingText(req.ChapterContent, req.Content),
		"storyContent": req.StoryContent, "parentEnding": req.ParentEnding,
		"sceneCount": sceneCount, "comicStyle": req.ComicStyle,
		"parentStoryboardId": parentID, "useComicPagePipeline": req.UseComicPagePipeline,
	}
	releaseID := strings.TrimSpace(req.WorkflowReleaseID)
	entry, promptSnapshots, err := h.workflowRegistry.ResolvePinnedPromptSnapshotsForInput(c.Request.Context(), "voyager.storyboard", "generate", "", releaseID, routingInput)
	if err != nil {
		h.logger.Warn("storyboard workflow context resolution failed", zap.String("releaseId", releaseID), zap.Error(err))
		InvalidParams(c, "storyboard workflow release changed; refresh and retry")
		return
	}
	storyboard.WorkflowReleaseID = entry.Release.ID
	storyboard.WorkflowChecksum = entry.Release.Checksum
	storyboard.PromptSnapshots = promptSnapshots
	storyboard.WorkflowManagedGeneration = workflowReleaseManagesStoryboardStages(entry.Release.Definition)

	if len(req.CharacterRefs) > 0 {
		storyboard.CharacterRefs = make([]domain.StoryboardCharacterRef, len(req.CharacterRefs))
		for i, ref := range req.CharacterRefs {
			order := i
			if ref.Order != nil {
				order = *ref.Order
			}
			storyboard.CharacterRefs[i] = domain.StoryboardCharacterRef{
				CharacterID: ref.CharacterID,
				Role:        ref.Role,
				Order:       order,
				Notes:       ref.Notes,
			}
		}
	}

	if len(req.SceneRefs) > 0 {
		storyboard.SceneRefs = make([]domain.StoryboardSceneRef, len(req.SceneRefs))
		for i, ref := range req.SceneRefs {
			seq := i
			if ref.Sequence != nil {
				seq = *ref.Sequence
			}
			storyboard.SceneRefs[i] = domain.StoryboardSceneRef{
				StorySceneID:   ref.StorySceneID,
				Sequence:       seq,
				IsPrimaryScene: ref.IsPrimaryScene,
			}
		}
	}

	// Create is idempotent when Idempotency-Key is provided.
	if err := h.svc.CreateStoryboardWithIdempotency(c.Request.Context(), storyboard, idempotencyKey); err != nil {
		h.logger.Error("CreateStoryboard failed",
			zap.String("storyId", req.StoryID),
			zap.Error(err))
		HandleError(c, err)
		return
	}

	// 添加标签（如果有）- 故事板标签添加到对应的故事上
	if len(req.Tags) > 0 {
		// 去重并规范化标签
		uniqueTags := make(map[string]bool)
		normalizedTags := make([]string, 0, len(req.Tags))
		for _, tag := range req.Tags {
			normalized := strings.TrimSpace(strings.ToLower(tag))
			if normalized != "" && !uniqueTags[normalized] {
				uniqueTags[normalized] = true
				normalizedTags = append(normalizedTags, normalized)
			}
		}
		if len(normalizedTags) > 0 {
			if err := h.svc.AddStoryTags(c.Request.Context(), storyboard.StoryID, normalizedTags); err != nil {
				h.logger.Warn("failed to add tags to story for storyboard",
					zap.String("storyboardId", storyboard.ID),
					zap.String("storyId", storyboard.StoryID),
					zap.Strings("tags", normalizedTags),
					zap.Error(err))
				// 不返回错误，标签添加失败不影响故事板创建
			}
		}
	}

	h.logger.Info("CreateStoryboard success",
		zap.String("storyboardId", storyboard.ID),
		zap.String("storyId", storyboard.StoryID),
		zap.String("parentId", storyboard.ParentID),
		zap.String("workflowStatus", storyboard.WorkflowStatus))

	h.attachStoryboardIsLiked(c, storyboard)
	domain.RedactStoryboardViewsUnlessCreator(storyboard, uid)
	Success(c, storyboard)
}

func firstNonEmptyRoutingText(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func workflowReleaseManagesStoryboardStages(definition domain.WorkflowDefinition) bool {
	required := map[string]bool{
		"storyboard.generate_bible_plan": false,
		"storyboard.generate_scene_plan": false,
		"storyboard.persist_content":     false,
	}
	for _, node := range definition.Nodes {
		if _, ok := required[strings.TrimSpace(node.Activity)]; ok {
			required[strings.TrimSpace(node.Activity)] = true
		}
	}
	for _, found := range required {
		if !found {
			return false
		}
	}
	return true
}

// GetStoryboard 获取 storyboard 详情
func (h *Handler) GetStoryboard(c *gin.Context) {
	id := c.Param("id")

	storyboard, err := h.svc.GetStoryboard(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			NotFound(c, "storyboard not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	shareGrant := h.ShareGrantFromRequest(c, service.ShareKindStoryboard, id)
	if !h.svc.CanViewerSeeStoryboard(c.Request.Context(), GetUserID(c), storyboard, shareGrant) {
		HandleError(c, domain.ErrForbidden)
		return
	}
	if shareGrant {
		h.recordShareEvent(c.Request.Context(), domain.ShareEventOpen, service.ShareKindStoryboard, id, GetUserID(c), service.SharePlatformWeb, service.ShareSourceContentGet)
	}

	if uid := GetUserID(c); uid != "" {
		go func(sbID, viewer string) {
			h.svc.RecordStoryboardFeedSeen(context.Background(), viewer, sbID)
		}(id, uid)
	}

	h.attachStoryboardIsLiked(c, storyboard)
	domain.RedactStoryboardViewsUnlessCreator(storyboard, GetUserID(c))
	Success(c, storyboard)
}

// UpdateStoryboard 更新 storyboard
func (h *Handler) UpdateStoryboard(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	var req struct {
		Title         string                `json:"title"`
		Content       string                `json:"content"`
		RawInput      string                `json:"rawInput"`
		SceneRefs     []sceneRefPayload     `json:"sceneRefs"` // References to story-level scenes
		CharacterRefs []characterRefPayload `json:"characterRefs"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	storyboard := &domain.Storyboard{
		BaseModel: common.BaseModel{
			ID: id,
		},
		Title:    req.Title,
		Content:  req.Content,
		RawInput: req.RawInput,
		// StoryboardScenes are managed separately via AI generation
	}

	if req.CharacterRefs != nil {
		storyboard.CharacterRefs = make([]domain.StoryboardCharacterRef, len(req.CharacterRefs))
		for i, ref := range req.CharacterRefs {
			order := i
			if ref.Order != nil {
				order = *ref.Order
			}
			storyboard.CharacterRefs[i] = domain.StoryboardCharacterRef{
				CharacterID: ref.CharacterID,
				Role:        ref.Role,
				Order:       order,
				Notes:       ref.Notes,
			}
		}
	}

	if req.SceneRefs != nil {
		storyboard.SceneRefs = make([]domain.StoryboardSceneRef, len(req.SceneRefs))
		for i, ref := range req.SceneRefs {
			seq := i
			if ref.Sequence != nil {
				seq = *ref.Sequence
			}
			storyboard.SceneRefs[i] = domain.StoryboardSceneRef{
				StorySceneID:   ref.StorySceneID,
				Sequence:       seq,
				IsPrimaryScene: ref.IsPrimaryScene,
			}
		}
	}

	if err := h.svc.UpdateStoryboard(c.Request.Context(), storyboard, userID.(string)); err != nil {
		InternalError(c, err.Error())
		return
	}

	updatedStoryboard, err := h.svc.GetStoryboard(c.Request.Context(), id)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	h.attachStoryboardIsLiked(c, updatedStoryboard)
	domain.RedactStoryboardViewsUnlessCreator(updatedStoryboard, userID.(string))
	Success(c, updatedStoryboard)
}

// UpdateStoryboardPlotScene PUT /storyboards/:id/scenes/:sceneId — 更新 AI 分镜叙事字段并失效/重算 continuation summary。
func (h *Handler) UpdateStoryboardPlotScene(c *gin.Context) {
	userID, ok := c.Get("userID")
	if !ok {
		Unauthorized(c, "not authenticated")
		return
	}
	uid, _ := userID.(string)
	if uid == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	storyboardID := c.Param("id")
	sceneID := c.Param("sceneId")
	if storyboardID == "" || sceneID == "" {
		InvalidParams(c, "storyboard id and scene id are required")
		return
	}

	var patch service.StoryboardPlotScenePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	scene, err := h.svc.UpdateStoryboardPlotScene(c.Request.Context(), uid, storyboardID, sceneID, patch)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			Forbidden(c, err.Error())
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			NotFound(c, "scene not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, scene)
}

// attachStoryboardIsLiked sets storyboard.IsLiked for the current viewer when authenticated.
func (h *Handler) attachStoryboardIsLiked(c *gin.Context, storyboard *domain.Storyboard) {
	if storyboard == nil {
		return
	}
	uid := GetUserID(c)
	if uid == "" {
		return
	}
	liked, err := h.svc.IsStoryboardLikedByUser(c.Request.Context(), uid, storyboard.ID)
	if err != nil {
		h.logger.Debug("attachStoryboardIsLiked skipped",
			zap.String("storyboardId", storyboard.ID),
			zap.Error(err))
		return
	}
	storyboard.IsLiked = &liked
}

// attachStoryboardIsLikedMany sets IsLiked on each item for the current viewer (batch query).
func (h *Handler) attachStoryboardIsLikedMany(c *gin.Context, boards []*domain.Storyboard) {
	uid := GetUserID(c)
	if uid == "" || len(boards) == 0 {
		return
	}
	ids := make([]string, 0, len(boards))
	for _, b := range boards {
		if b != nil && b.ID != "" {
			ids = append(ids, b.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	m, err := h.svc.BatchIsStoryboardLikedByUser(c.Request.Context(), uid, ids)
	if err != nil {
		h.logger.Debug("attachStoryboardIsLikedMany skipped", zap.Error(err))
		return
	}
	for _, b := range boards {
		if b == nil {
			continue
		}
		liked := m[b.ID]
		b.IsLiked = &liked
	}
}

// DeleteStoryboard 删除 storyboard
func (h *Handler) DeleteStoryboard(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	if err := h.svc.DeleteStoryboard(c.Request.Context(), id, userID.(string)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "storyboard deleted successfully"})
}

// GetStoryboardFeed 获取故事板 feed 流
// Query params:
//   - tab: discover（默认，全站公开 trending）；for_you / recommended（个性化推荐）；following；community（全站时间线）
//   - limit: 分页限制（默认20）
//   - offset: 分页偏移（默认0）
func (h *Handler) GetStoryboardFeed(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	tab := c.DefaultQuery("tab", "discover")

	uid := ""
	if v, ok := c.Get("userID"); ok {
		if s, ok := v.(string); ok {
			uid = s
		}
	}

	h.logger.Info("GetStoryboardFeed called",
		zap.String("tab", tab),
		zap.String("userID", uid),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// Feed 需逐个加载分镜（storyboardToDomain → StoryboardScenes），耗时较长。下拉刷新若取消旧请求，
	// Request.Context 会被取消导致中途 DB 报错。使用 WithoutCancel 保留 trace/values，仅用服务端超时约束。
	feedCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 90*time.Second)
	defer cancel()

	storyboards, total, err := h.svc.GetStoryboardFeed(feedCtx, uid, tab, limit, offset)
	if err != nil {
		h.logger.Error("GetStoryboardFeed failed", zap.Error(err))
		InternalError(c, err.Error())
		return
	}

	h.logger.Info("GetStoryboardFeed result",
		zap.Int("count", len(storyboards)),
		zap.Int64("total", total))

	h.attachStoryboardIsLikedMany(c, storyboards)
	domain.RedactStoryboardViewsUnlessCreatorMany(storyboards, uid)
	Success(c, gin.H{
		"storyboards": storyboards,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// ListStoryboards 获取故事的 storyboards 列表
// Query params:
//   - storyId: 故事ID（必填）
//   - parentId: 父级ID（可选）- 为空或 "root" 时获取根故事板，指定ID时获取该父级的子故事板
//   - limit: 分页限制（默认20）
//   - offset: 分页偏移（默认0）
func (h *Handler) ListStoryboards(c *gin.Context) {
	storyID := c.Query("storyId")
	if storyID == "" {
		InvalidParams(c, "storyId is required")
		return
	}
	story, err := h.svc.GetStory(c.Request.Context(), storyID)
	if err != nil {
		HandleError(c, err)
		return
	}
	if !h.svc.CanViewerSeeStory(c.Request.Context(), GetUserID(c), story) {
		HandleError(c, domain.ErrForbidden)
		return
	}
	includeUnpublished := h.svc.CanViewUnpublishedStoryboards(c.Request.Context(), story, GetUserID(c))

	parentID := c.Query("parentId")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	h.logger.Info("ListStoryboards called",
		zap.String("storyId", storyID),
		zap.String("parentId", parentID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	var storyboards []*domain.Storyboard
	var parentStoryboard *domain.Storyboard
	err = nil

	// 根据 parentId 参数决定获取哪些故事板
	switch parentID {
	case "", "root", domain.StoryboardRootMarker:
		// 获取根故事板（ParentID 为空或 "__root__"）
		storyboards, err = h.svc.ListRootStoryboards(c.Request.Context(), storyID, limit, offset, includeUnpublished)
	default:
		parentStoryboard, err = h.svc.GetStoryboard(c.Request.Context(), parentID)
		if err != nil {
			if err == domain.ErrNotFound {
				NotFound(c, "parent storyboard not found")
				return
			}
			h.logger.Error("failed to get parent storyboard",
				zap.String("storyId", storyID),
				zap.String("parentId", parentID),
				zap.Error(err))
			InternalError(c, err.Error())
			return
		}
		if parentStoryboard.StoryID != storyID || !h.svc.CanViewerSeeStoryboard(c.Request.Context(), GetUserID(c), parentStoryboard, false) {
			HandleError(c, domain.ErrForbidden)
			return
		}
		// 获取指定父级的子故事板
		storyboards, err = h.svc.ListStoryboardsByParent(c.Request.Context(), storyID, parentID, limit, offset, includeUnpublished)
	}

	if err != nil {
		h.logger.Error("ListStoryboards failed",
			zap.String("storyId", storyID),
			zap.Error(err))
		InternalError(c, err.Error())
		return
	}

	// Log details of each storyboard for debugging
	for _, sb := range storyboards {
		h.logger.Info("Storyboard found",
			zap.String("storyboardId", sb.ID),
			zap.String("storyId", sb.StoryID),
			zap.String("parentId", sb.ParentID),
			zap.String("workflowStatus", sb.WorkflowStatus),
			zap.String("title", sb.Title))
	}

	h.logger.Info("ListStoryboards result",
		zap.String("storyId", storyID),
		zap.Int("count", len(storyboards)))

	viewer := GetUserID(c)
	h.attachStoryboardIsLikedMany(c, storyboards)
	if parentStoryboard != nil {
		h.attachStoryboardIsLiked(c, parentStoryboard)
	}
	domain.RedactStoryboardViewsUnlessCreatorMany(storyboards, viewer)
	domain.RedactStoryboardViewsUnlessCreator(parentStoryboard, viewer)
	Success(c, gin.H{
		"storyboards":      storyboards,
		"parentStoryboard": parentStoryboard,
		"total":            len(storyboards),
		"limit":            limit,
		"offset":           offset,
	})
}

// GetStoryboardChildren 获取子 storyboards
func (h *Handler) GetStoryboardChildren(c *gin.Context) {
	parentID := c.Param("id")
	parent, err := h.svc.GetStoryboard(c.Request.Context(), parentID)
	if err != nil {
		HandleError(c, err)
		return
	}
	if !h.svc.CanViewerSeeStoryboard(c.Request.Context(), GetUserID(c), parent, false) {
		HandleError(c, domain.ErrForbidden)
		return
	}

	story, err := h.svc.GetStory(c.Request.Context(), parent.StoryID)
	if err != nil {
		HandleError(c, err)
		return
	}
	includeUnpublished := h.svc.CanViewUnpublishedStoryboards(c.Request.Context(), story, GetUserID(c))
	children, err := h.svc.GetStoryboardChildren(c.Request.Context(), parentID, includeUnpublished)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	h.attachStoryboardIsLikedMany(c, children)
	domain.RedactStoryboardViewsUnlessCreatorMany(children, GetUserID(c))
	Success(c, gin.H{
		"children": children,
		"count":    len(children),
	})
}

// GetStoryboardTree 获取完整的 storyboard 树
func (h *Handler) GetStoryboardTree(c *gin.Context) {
	rootID := c.Param("id")
	root, err := h.svc.GetStoryboard(c.Request.Context(), rootID)
	if err != nil {
		HandleError(c, err)
		return
	}
	if !h.svc.CanViewerSeeStoryboard(c.Request.Context(), GetUserID(c), root, false) {
		HandleError(c, domain.ErrForbidden)
		return
	}

	story, err := h.svc.GetStory(c.Request.Context(), root.StoryID)
	if err != nil {
		HandleError(c, err)
		return
	}
	includeUnpublished := h.svc.CanViewUnpublishedStoryboards(c.Request.Context(), story, GetUserID(c))
	tree, err := h.svc.GetStoryboardTree(c.Request.Context(), rootID, includeUnpublished)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	h.attachStoryboardIsLikedMany(c, tree)
	domain.RedactStoryboardViewsUnlessCreatorMany(tree, GetUserID(c))
	Success(c, gin.H{
		"tree":  tree,
		"count": len(tree),
	})
}

// ForkStoryboard Fork 一个 storyboard
func (h *Handler) ForkStoryboard(c *gin.Context) {
	userID, _ := c.Get("userID")
	parentID := c.Param("id")

	var req struct {
		Title             string                `json:"title" binding:"required"`
		RawInput          string                `json:"rawInput" binding:"required"`
		Content           string                `json:"content"`
		SceneCount        int                   `json:"sceneCount"` // Requested number of scenes to generate (2-8, default 3)
		SceneRefs         []sceneRefPayload     `json:"sceneRefs"`  // References to story-level scenes
		CharacterRefs     []characterRefPayload `json:"characterRefs"`
		WorkflowReleaseID string                `json:"workflowReleaseId,omitempty"`
		WorkflowRunID     string                `json:"workflowRunId,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	if strings.TrimSpace(req.WorkflowReleaseID) == "" || strings.TrimSpace(req.WorkflowRunID) == "" {
		InvalidParams(c, "storyboard fork must be started by a configured workflow")
		return
	}
	if h.generationRuntime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	workflowRun, runErr := h.generationRuntime.GetExecution(c.Request.Context(), req.WorkflowRunID)
	if runErr != nil || workflowRun.UserID != userID.(string) || workflowRun.WorkflowReleaseID != req.WorkflowReleaseID || workflowRun.Kind != "branch_batch" {
		Forbidden(c, "storyboard fork workflow execution is invalid")
		return
	}

	sceneCount, err := service.NormalizeStoryboardSceneCount(req.SceneCount)
	if err != nil {
		InvalidParams(c, err.Error())
		return
	}

	newStoryboard := &domain.Storyboard{
		Title:        req.Title,
		RawInput:     req.RawInput,
		Content:      req.Content,
		IsStandalone: false,
		SceneCount:   sceneCount,
		// StoryboardScenes are generated by AI, not passed in fork request
	}
	parent, err := h.svc.GetStoryboard(c.Request.Context(), parentID)
	if err != nil {
		HandleError(c, err)
		return
	}
	if h.workflowRegistry == nil {
		InternalError(c, "workflow registry unavailable")
		return
	}
	routingInput := map[string]any{
		"parentStoryboardId": parentID, "storyId": parent.StoryID, "seedPrompt": req.RawInput,
		"chapterContent": firstNonEmptyRoutingText(parent.Content, parent.ContinuationSummary, parent.RawInput),
		"parentEnding":   parent.ContinuationSummary, "sceneCount": sceneCount,
	}
	releaseID := strings.TrimSpace(req.WorkflowReleaseID)
	entry, snapshots, err := h.workflowRegistry.ResolvePinnedPromptSnapshotsForInput(c.Request.Context(), "voyager.storyboard", "branch", "", releaseID, routingInput)
	if err != nil {
		InvalidParams(c, "storyboard branch workflow release changed; refresh and retry")
		return
	}
	newStoryboard.WorkflowReleaseID = entry.Release.ID
	newStoryboard.WorkflowChecksum = entry.Release.Checksum
	newStoryboard.PromptSnapshots = snapshots

	if len(req.CharacterRefs) > 0 {
		newStoryboard.CharacterRefs = make([]domain.StoryboardCharacterRef, len(req.CharacterRefs))
		for i, ref := range req.CharacterRefs {
			order := i
			if ref.Order != nil {
				order = *ref.Order
			}
			newStoryboard.CharacterRefs[i] = domain.StoryboardCharacterRef{
				CharacterID: ref.CharacterID,
				Role:        ref.Role,
				Order:       order,
				Notes:       ref.Notes,
			}
		}
	}

	if len(req.SceneRefs) > 0 {
		newStoryboard.SceneRefs = make([]domain.StoryboardSceneRef, len(req.SceneRefs))
		for i, ref := range req.SceneRefs {
			seq := i
			if ref.Sequence != nil {
				seq = *ref.Sequence
			}
			newStoryboard.SceneRefs[i] = domain.StoryboardSceneRef{
				StorySceneID:   ref.StorySceneID,
				Sequence:       seq,
				IsPrimaryScene: ref.IsPrimaryScene,
			}
		}
	}

	// AI generation is handled by ForkStoryboard service when content is empty and rawInput is provided
	if err := h.svc.ForkStoryboard(c.Request.Context(), parentID, userID.(string), newStoryboard); err != nil {
		HandleError(c, err)
		return
	}

	h.attachStoryboardIsLiked(c, newStoryboard)
	domain.RedactStoryboardViewsUnlessCreator(newStoryboard, userID.(string))
	Success(c, newStoryboard)
}

// GetStoryboardForkPermission returns the server-authoritative preflight result.
// GET /api/v1/storyboards/:id/fork-permission
func (h *Handler) GetStoryboardForkPermission(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	parentID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	permission, err := h.svc.CanForkStoryboard(c.Request.Context(), parentID, userID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, permission)
}

// ContinueStoryboard 继续故事板（平行宇宙续写）
// Simplified interface: automatically handles path tracing, state synthesis, and AI generation
func (h *Handler) ContinueStoryboard(c *gin.Context) {
	userID, _ := c.Get("userID")
	parentID := c.Param("id")

	var req continueStoryboardPayload

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	if strings.TrimSpace(req.WorkflowReleaseID) == "" || strings.TrimSpace(req.WorkflowRunID) == "" {
		InvalidParams(c, "storyboard continuation must be started by a configured workflow")
		return
	}
	if h.generationRuntime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	workflowRun, runErr := h.generationRuntime.GetExecution(c.Request.Context(), req.WorkflowRunID)
	if runErr != nil || workflowRun.UserID != userID.(string) || workflowRun.WorkflowReleaseID != req.WorkflowReleaseID || workflowRun.Kind != "branch_batch" {
		Forbidden(c, "storyboard continuation workflow execution is invalid")
		return
	}

	h.logger.Info("ContinueStoryboard called",
		zap.String("parentId", parentID),
		zap.String("userId", userID.(string)),
		zap.String("rawInput", truncateForLog(req.RawInput, 200)),
		zap.Int("sceneCount", req.SceneCount),
		zap.Bool("generateVideo", req.GenerateVideo),
		zap.String("comicStyle", req.ComicStyle))

	sceneCount, err := service.NormalizeStoryboardSceneCount(req.SceneCount)
	if err != nil {
		InvalidParams(c, err.Error())
		return
	}

	parent, err := h.svc.GetStoryboard(c.Request.Context(), parentID)
	if err != nil {
		HandleError(c, err)
		return
	}
	if h.workflowRegistry == nil {
		InternalError(c, "workflow registry unavailable")
		return
	}
	routingInput := map[string]any{
		"parentStoryboardId": parentID,
		"storyId":            parent.StoryID,
		"seedPrompt":         req.RawInput,
		"chapterContent":     firstNonEmptyRoutingText(parent.Content, parent.ContinuationSummary, parent.RawInput),
		"parentEnding":       parent.ContinuationSummary,
		"sceneCount":         sceneCount,
		"characters":         req.Characters,
		"comicStyle":         req.ComicStyle,
	}
	releaseID := strings.TrimSpace(req.WorkflowReleaseID)
	entry, snapshots, err := h.workflowRegistry.ResolvePinnedPromptSnapshotsForInput(c.Request.Context(), "voyager.storyboard", "branch", "", releaseID, routingInput)
	if err != nil {
		InvalidParams(c, "storyboard branch workflow release changed; refresh and retry")
		return
	}

	// Use the narrator pipeline for parallel universe continuation
	result, err := h.svc.ContinueStoryboard(c.Request.Context(), userID.(string), &service.ContinueRequest{
		ParentStoryboardID: parentID,
		UserPrompt:         req.RawInput,
		SceneCount:         sceneCount,
		Characters:         req.Characters,
		GenerateVideo:      req.GenerateVideo,
		ComicStyle:         req.ComicStyle,
		WorkflowReleaseID:  entry.Release.ID,
		WorkflowChecksum:   entry.Release.Checksum,
		PromptSnapshots:    snapshots,
		ParentStoryboard:   parent,
	})
	if err != nil {
		h.logger.Error("ContinueStoryboard failed",
			zap.String("parentId", parentID),
			zap.Error(err))
		HandleError(c, err)
		return
	}

	if result != nil && result.NewStoryboard != nil {
		h.attachStoryboardIsLiked(c, result.NewStoryboard)
		domain.RedactStoryboardViewsUnlessCreator(result.NewStoryboard, userID.(string))
	}
	Success(c, result)
}

// LikeStoryboard 点赞 storyboard
func (h *Handler) LikeStoryboard(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	if err := h.svc.LikeStoryboard(c.Request.Context(), userID.(string), id); err != nil {
		if errors.Is(err, domain.ErrAlreadyLiked) {
			Success(c, gin.H{"message": "storyboard liked successfully"})
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "storyboard liked successfully"})
}

// UnlikeStoryboard 取消点赞 storyboard
func (h *Handler) UnlikeStoryboard(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")

	if err := h.svc.UnlikeStoryboard(c.Request.Context(), userID.(string), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			Success(c, gin.H{"message": "storyboard unliked successfully"})
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "storyboard unliked successfully"})
}

// ListStoryboardPanels 列出分镜面板
// GET /api/v1/storyboards/:id/panels
func (h *Handler) ListStoryboardPanels(c *gin.Context) {
	storyboardID := c.Param("id")
	if storyboardID == "" {
		InvalidParams(c, "storyboard id is required")
		return
	}

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	panels, total, err := h.svc.ListStoryboardPanels(c.Request.Context(), storyboardID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"panels": panels,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateStoryboardPanel 创建分镜面板
// POST /api/v1/storyboards/:id/panels
func (h *Handler) CreateStoryboardPanel(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyboardID := c.Param("id")

	var req struct {
		ImageURL  string `json:"img"`
		Text      string `json:"text" binding:"required"`
		TextPos   string `json:"textPos"`
		TextRight string `json:"textRight"`
		Sequence  *int   `json:"sequence"`
	}

	if !BindJSON(c, &req) {
		return
	}

	panel := &domain.StoryboardPanel{
		StoryboardID: storyboardID,
		ImageURL:     req.ImageURL,
		Text:         req.Text,
		TextPos:      req.TextPos,
		TextRight:    req.TextRight,
	}

	if req.Sequence != nil {
		panel.Sequence = *req.Sequence
	}

	createdPanel, err := h.svc.CreateStoryboardPanel(c.Request.Context(), userID, storyboardID, panel)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, createdPanel)
}
