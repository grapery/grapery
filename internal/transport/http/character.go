package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// CreateCharacter 创建角色
func (h *Handler) CreateCharacter(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req service.CreateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	character, err := h.svc.CreateCharacter(c.Request.Context(), userID, req)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "you can only add characters to your own stories" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "character with same name already exists in this story" {
			Error(c, CodeError, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, character)
}

func (h *Handler) StartCharacterGenerationTask(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}
	var req service.CharacterGenerationTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}
	task, err := h.svc.StartCharacterGenerationTask(c.Request.Context(), userID, req)
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "you can only generate characters for your own stories" {
			Forbidden(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}
	Success(c, task)
}

func (h *Handler) GetCharacterGenerationTask(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}
	task, err := h.svc.GetCharacterGenerationTask(c.Request.Context(), userID, c.Param("taskId"))
	if err != nil {
		if err == domain.ErrNotFound || err.Error() == "record not found" {
			NotFound(c, "task not found")
			return
		}
		if err.Error() == "unauthorized" {
			Forbidden(c, "unauthorized")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}
	Success(c, task)
}

func (h *Handler) ListCharacterGenerationTasks(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	tasks, err := h.svc.ListCharacterGenerationTasks(c.Request.Context(), userID, c.Query("status"), limit, offset)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}
	Success(c, gin.H{"tasks": tasks, "count": len(tasks), "limit": limit, "offset": offset})
}

func (h *Handler) RetryCharacterGenerationTask(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}
	task, err := h.svc.RetryCharacterGenerationTask(c.Request.Context(), userID, c.Param("taskId"))
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}
	Success(c, task)
}

func (h *Handler) DismissCharacterGenerationTaskFromDrafts(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}
	task, err := h.svc.DismissCharacterGenerationTaskFromDrafts(c.Request.Context(), userID, c.Param("taskId"))
	if err != nil {
		if err == domain.ErrNotFound || err.Error() == "record not found" {
			NotFound(c, "task not found")
			return
		}
		if err.Error() == "unauthorized" {
			Forbidden(c, "unauthorized")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}
	Success(c, task)
}

func (h *Handler) PreviewFragmentCharactersForStory(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}
	resp, err := h.svc.PreviewFragmentCharactersForStory(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if err.Error() == "story not found" {
			NotFound(c, "story not found")
			return
		}
		if err.Error() == "you can only generate characters for your own stories" {
			Forbidden(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}
	Success(c, resp)
}

// GetCharacter 获取角色详情
func (h *Handler) GetCharacter(c *gin.Context) {
	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	// 获取当前用户ID（可能为空，如果未登录）
	userID := ""
	if uid, exists := c.Get("userID"); exists {
		userID = uid.(string)
	}

	character, err := h.svc.GetCharacterWithUserContext(c.Request.Context(), characterID, userID)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	shareGrant := h.ShareGrantFromRequest(c, service.ShareKindCharacter, characterID)
	if !h.svc.CanViewerSeeCharacter(c.Request.Context(), userID, character, shareGrant) {
		HandleError(c, domain.ErrForbidden)
		return
	}
	if shareGrant {
		h.recordShareEvent(c.Request.Context(), domain.ShareEventOpen, service.ShareKindCharacter, characterID, userID, service.SharePlatformWeb, service.ShareSourceContentGet)
	}

	Success(c, character)
}

// ListCharacters 获取角色列表
func (h *Handler) ListCharacters(c *gin.Context) {
	var req service.CharacterListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	characters, err := h.svc.ListCharacters(c.Request.Context(), req)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"characters": characters,
		"count":      len(characters),
		"limit":      req.Limit,
		"offset":     req.Offset,
	})
}

// UpdateCharacter 更新角色
func (h *Handler) UpdateCharacter(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	var req service.UpdateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	character, err := h.svc.UpdateCharacter(c.Request.Context(), userID, characterID, req)
	if err != nil {
		if err.Error() == "unauthorized" {
			Forbidden(c, "you can only update your own characters")
			return
		}
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, character)
}

// DeleteCharacter 删除角色
func (h *Handler) DeleteCharacter(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	err := h.svc.DeleteCharacter(c.Request.Context(), userID, characterID)
	if err != nil {
		if err.Error() == "unauthorized" {
			Forbidden(c, "you can only delete your own characters")
			return
		}
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "character deleted successfully"})
}

// FollowCharacter 关注角色
func (h *Handler) FollowCharacter(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	err := h.svc.FollowCharacter(c.Request.Context(), userID, characterID)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "character followed successfully"})
}

// UnfollowCharacter 取消关注角色
func (h *Handler) UnfollowCharacter(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	err := h.svc.UnfollowCharacter(c.Request.Context(), userID, characterID)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "character unfollowed successfully"})
}

// REMOVED: Character Skills Management - not in StoryCreationAppUI design

// ========== Character Analytics ==========

// GetCharacterAnalytics 获取角色分析数据
// GET /api/characters/:id/analytics
func (h *Handler) GetCharacterAnalytics(c *gin.Context) {
	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	analytics, err := h.svc.GetCharacterAnalytics(c.Request.Context(), characterID)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, analytics)
}

// REMOVED: Character Posters - not in StoryCreationAppUI design

// GetCharacterStoryboards 获取角色参与的故事板列表
// GET /api/characters/:id/storyboards
func (h *Handler) GetCharacterStoryboards(c *gin.Context) {
	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	storyboards, total, err := h.svc.GetCharacterStoryboards(c.Request.Context(), characterID, limit, offset)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	h.attachStoryboardIsLikedMany(c, storyboards)
	domain.RedactStoryboardViewsUnlessCreatorMany(storyboards, authPkg.GetUserID(c))
	Success(c, gin.H{
		"storyboards": storyboards,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GenerateCharacterAttributes 使用AI生成角色属性
// POST /api/characters/generate
func (h *Handler) GenerateCharacterAttributes(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req service.GenerateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	attributes, err := h.svc.GenerateCharacterWithAI(c.Request.Context(), userID, req)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, attributes)
}

// GenerateCharacter 使用AI生成角色内容（iOS兼容端点）
// POST /api/ai/generate-character
func (h *Handler) GenerateCharacter(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// iOS app sends: { name: String, baseDescription: String? }
	var req struct {
		Name            string `json:"name"`
		BaseDescription string `json:"baseDescription"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// Map iOS request to service request
	// Use baseDescription as prompt, or empty string if not provided
	prompt := req.BaseDescription
	if prompt == "" {
		prompt = "创建一个角色"
		if req.Name != "" {
			prompt = "创建一个名为" + req.Name + "的角色"
		}
	}

	serviceReq := service.GenerateCharacterRequest{
		Name:   req.Name,
		Prompt: prompt,
	}

	attributes, err := h.svc.GenerateCharacterWithAI(c.Request.Context(), userID, serviceReq)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	// Return all fields from GeneratedCharacterAttributes
	Success(c, gin.H{
		"description":     attributes.Description,
		"personality":     attributes.Personality,
		"background":      attributes.Background,
		"shortTermGoal":   attributes.ShortTermGoal,
		"longTermGoal":    attributes.LongTermGoal,
		"handlingStyle":   attributes.HandlingStyle,
		"cognitionRange":  attributes.CognitionRange,
		"abilityFeatures": attributes.AbilityFeatures,
		"appearance":      attributes.Appearance,
		"dressPreference": attributes.DressPreference,
	})
}

// GenerateCharacterAvatar 使用AI生成角色头像
// POST /api/characters/:id/generate-avatar
func (h *Handler) GenerateCharacterAvatar(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	var req service.GenerateCharacterAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体，使用默认参数
		req = service.GenerateCharacterAvatarRequest{
			AspectRatio: "1:1",
		}
	}

	result, err := h.svc.GenerateCharacterAvatar(c.Request.Context(), userID, characterID, req)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		if err.Error() == "unauthorized" {
			Forbidden(c, "you can only generate avatars for your own characters")
			return
		}
		if err.Error() == "AI generation service not configured" {
			Error(c, CodeError, "AI service temporarily unavailable")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, result)
}

// UpdateCharacterAvatar 更新角色头像
// PUT /api/characters/:id/avatar
func (h *Handler) UpdateCharacterAvatar(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	var req struct {
		AvatarURL string `json:"avatarUrl" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	character, err := h.svc.UpdateCharacterAvatar(c.Request.Context(), userID, characterID, req.AvatarURL)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		if err.Error() == "unauthorized" {
			Forbidden(c, "you can only update your own characters")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, character)
}

// UsePortraitAsAvatar 使用portrait作为头像（仅在头像为空时）
// PUT /api/characters/:id/use-portrait-as-avatar
func (h *Handler) UsePortraitAsAvatar(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	var req struct {
		PortraitURL string `json:"portraitUrl" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	character, err := h.svc.UsePortraitAsAvatar(c.Request.Context(), userID, characterID, req.PortraitURL)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		if err.Error() == "unauthorized" {
			Forbidden(c, "you can only update your own characters")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, character)
}

// GetPortraitPrompt 获取角色形象生成推荐提示词
// GET /api/characters/:id/portrait-prompt
func (h *Handler) GetPortraitPrompt(c *gin.Context) {
	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	prompt, err := h.svc.GeneratePortraitPrompt(c.Request.Context(), characterID)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"prompt": prompt})
}

// GenerateCharacterPortrait 生成角色形象图
// POST /api/characters/:id/generate-portrait
func (h *Handler) GenerateCharacterPortrait(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	var req service.GenerateCharacterPortraitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体，使用默认参数
		req = service.GenerateCharacterPortraitRequest{
			AspectRatio: "2:3",
		}
	}

	result, err := h.svc.GenerateCharacterPortrait(c.Request.Context(), userID, characterID, req)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		if err.Error() == "unauthorized: only character creator can generate portrait" {
			Forbidden(c, "you can only generate portraits for your own characters")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, result)
}

// GenerateCharacterThreeViews 生成/更新角色三视图
// POST /api/characters/:id/generate-three-views
func (h *Handler) GenerateCharacterThreeViews(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	var req service.GenerateCharacterThreeViewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = service.GenerateCharacterThreeViewsRequest{}
	}

	result, err := h.svc.GenerateCharacterThreeViews(c.Request.Context(), userID, characterID, req)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		if err.Error() == "unauthorized: only character creator can generate three-views" {
			Forbidden(c, "you can only generate three-views for your own characters")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, result)
}

// CropAvatarFromPortrait 从形象图裁剪生成头像
// POST /api/characters/:id/crop-avatar
func (h *Handler) CropAvatarFromPortrait(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	avatarURL, err := h.svc.CropAvatarFromPortrait(c.Request.Context(), characterID)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		if err.Error() == "character has no portrait to crop from" {
			InvalidParams(c, "character has no portrait image")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"avatarUrl": avatarURL})
}
