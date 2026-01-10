package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
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

// ========== Character Skills Management ==========

// AddCharacterSkill 添加角色技能
// POST /api/characters/:id/skills
func (h *Handler) AddCharacterSkill(c *gin.Context) {
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
		Skill string `json:"skill" binding:"required,min=1,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	character, err := h.svc.AddCharacterSkill(c.Request.Context(), userID, characterID, req.Skill)
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

// RemoveCharacterSkill 移除角色技能
// DELETE /api/characters/:id/skills/:skill
func (h *Handler) RemoveCharacterSkill(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	characterID := c.Param("id")
	skill := c.Param("skill")
	if characterID == "" || skill == "" {
		InvalidParams(c, "character id and skill are required")
		return
	}

	character, err := h.svc.RemoveCharacterSkill(c.Request.Context(), userID, characterID, skill)
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

// ========== Character Posters ==========

// CreateCharacterPoster 创建角色海报
// POST /api/characters/:id/posters
func (h *Handler) CreateCharacterPoster(c *gin.Context) {
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

	var req service.CreatePosterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	poster, err := h.svc.CreateCharacterPoster(c.Request.Context(), userID, characterID, req)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		if err.Error() == "unauthorized: not a group member" {
			Forbidden(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, poster)
}

// GetCharacterPosters 获取角色海报列表
// GET /api/characters/:id/posters
func (h *Handler) GetCharacterPosters(c *gin.Context) {
	characterID := c.Param("id")
	if characterID == "" {
		InvalidParams(c, "character id is required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	posters, err := h.svc.GetCharacterPosters(c.Request.Context(), characterID, limit, offset)
	if err != nil {
		if err.Error() == "character not found" {
			NotFound(c, "character not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"posters": posters,
		"count":   len(posters),
	})
}

// LikeCharacterPoster 点赞角色海报
// POST /api/posters/:id/like
func (h *Handler) LikeCharacterPoster(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	posterID := c.Param("id")
	if posterID == "" {
		InvalidParams(c, "poster id is required")
		return
	}

	err := h.svc.LikeCharacterPoster(c.Request.Context(), posterID)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "poster liked successfully"})
}

// ShareCharacterPoster 分享角色海报
// POST /api/posters/:id/share
func (h *Handler) ShareCharacterPoster(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	posterID := c.Param("id")
	if posterID == "" {
		InvalidParams(c, "poster id is required")
		return
	}

	err := h.svc.ShareCharacterPoster(c.Request.Context(), posterID)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "poster shared successfully"})
}

// DeleteCharacterPoster 删除角色海报
// DELETE /api/posters/:id
func (h *Handler) DeleteCharacterPoster(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	posterID := c.Param("id")
	if posterID == "" {
		InvalidParams(c, "poster id is required")
		return
	}

	err := h.svc.DeleteCharacterPoster(c.Request.Context(), userID, posterID)
	if err != nil {
		if err.Error() == "unauthorized" {
			Forbidden(c, "you can only delete your own posters")
			return
		}
		if err.Error() == "poster not found" {
			NotFound(c, "poster not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "poster deleted successfully"})
}

// GenerateCharacterPoster 生成角色海报（AI两步工作流）
// POST /api/posters/:id/generate
// 步骤1：使用LLM生成海报概念JSON（包含视觉主体、场景、构图、灯光、艺术风格、排版指令）
// 步骤2：组装最终提示词，使用图像生成AI创建海报
func (h *Handler) GenerateCharacterPoster(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	posterID := c.Param("id")
	if posterID == "" {
		InvalidParams(c, "poster id is required")
		return
	}

	var req service.GeneratePosterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体，使用默认参数
		req = service.GeneratePosterRequest{
			AspectRatio: "16:9",
		}
	}

	result, err := h.svc.GenerateCharacterPoster(c.Request.Context(), userID, posterID, req)
	if err != nil {
		if err.Error() == "poster not found" {
			NotFound(c, "poster not found")
			return
		}
		if err.Error() == "unauthorized: you can only generate your own posters" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "poster is already generating or generated" {
			Error(c, CodeError, err.Error())
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

// PublishCharacterPoster 发布角色海报
// POST /api/posters/:id/publish
func (h *Handler) PublishCharacterPoster(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	posterID := c.Param("id")
	if posterID == "" {
		InvalidParams(c, "poster id is required")
		return
	}

	poster, err := h.svc.PublishCharacterPoster(c.Request.Context(), userID, posterID)
	if err != nil {
		if err.Error() == "poster not found" {
			NotFound(c, "poster not found")
			return
		}
		if err.Error() == "unauthorized: you can only publish your own posters" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "poster must be generated before publishing" {
			Error(c, CodeError, err.Error())
			return
		}
		if err.Error() == "poster has no image" {
			Error(c, CodeError, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, poster)
}

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
