package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// GetStyleConfigs 获取风格配置列表
func (h *Handler) GetStyleConfigs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	styleConfigs, total, err := h.svc.ListStyleConfigs(c.Request.Context(), limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	page := offset/limit + 1
	if limit == 0 {
		page = 1
	}

	SuccessPaginated(c, styleConfigs, total, page, limit)
}

// GetStyleConfigByID 根据ID获取风格配置
func (h *Handler) GetStyleConfigByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		InvalidParams(c, "style config ID is required")
		return
	}

	styleConfig, err := h.svc.GetStyleConfigByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "style config not found: "+id {
			NotFound(c, "Style config not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, styleConfig)
}

// GetStyleConfigByStyle 根据风格名称获取风格配置
func (h *Handler) GetStyleConfigByStyle(c *gin.Context) {
	styleName := c.Param("style")
	if styleName == "" {
		InvalidParams(c, "style name is required")
		return
	}

	styleConfig, err := h.svc.GetStyleConfigByStyle(c.Request.Context(), styleName)
	if err != nil {
		if err.Error() == "style config not found for style: "+styleName {
			NotFound(c, "Style config not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, styleConfig)
}

// SearchStyleConfigs 搜索风格配置
// 支持 groupId 参数，当提供时优先返回该小组的风格
func (h *Handler) SearchStyleConfigs(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		InvalidParams(c, "search keyword is required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	styleConfigs, total, err := h.svc.SearchStyleConfigs(c.Request.Context(), keyword, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	page := offset/limit + 1
	if limit == 0 {
		page = 1
	}

	SuccessPaginated(c, styleConfigs, total, page, limit)
}

// CreateStyleConfig 创建风格配置
func (h *Handler) CreateStyleConfig(c *gin.Context) {
	var req struct {
		Style          string `json:"style" binding:"required"`
		Description    string `json:"description"`
		SampleImageURL string `json:"sampleImageUrl"`
		UserID         string `json:"userId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	styleConfig := &domain.StyleConfig{
		Style:          req.Style,
		Description:    req.Description,
		SampleImageURL: req.SampleImageURL,
		UserID:         req.UserID,
	}

	if err := h.svc.CreateStyleConfig(c.Request.Context(), styleConfig); err != nil {
		if err.Error() == "style config with name '"+req.Style+"' already exists" {
			DuplicateEntry(c, "Style config with this name already exists")
			return
		}
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "Style config created successfully", styleConfig)
}

// UpdateStyleConfig 更新风格配置
func (h *Handler) UpdateStyleConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		InvalidParams(c, "style config ID is required")
		return
	}

	var req struct {
		Style          string `json:"style" binding:"required"`
		Description    string `json:"description"`
		SampleImageURL string `json:"sampleImageUrl"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	styleConfig := &domain.StyleConfig{
		ID:             id,
		Style:          req.Style,
		Description:    req.Description,
		SampleImageURL: req.SampleImageURL,
	}

	if err := h.svc.UpdateStyleConfig(c.Request.Context(), styleConfig); err != nil {
		if err.Error() == "style config not found: "+id {
			NotFound(c, "Style config not found")
			return
		}
		if err.Error() == "style config with name '"+req.Style+"' already exists" {
			DuplicateEntry(c, "Style config with this name already exists")
			return
		}
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "Style config updated successfully", styleConfig)
}

// DeleteStyleConfig 删除风格配置
func (h *Handler) DeleteStyleConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		InvalidParams(c, "style config ID is required")
		return
	}

	if err := h.svc.DeleteStyleConfig(c.Request.Context(), id); err != nil {
		if err.Error() == "style config not found: "+id {
			NotFound(c, "Style config not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "Style config deleted successfully", nil)
}

// GetStyleOptions 获取风格选项列表（简化版）
func (h *Handler) GetStyleOptions(c *gin.Context) {
	styleOptions, err := h.svc.GetStyleOptions(c.Request.Context())
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"styles": styleOptions})
}

// InitializeDefaultStyles 初始化默认风格配置
func (h *Handler) InitializeDefaultStyles(c *gin.Context) {
	if err := h.svc.InitializeDefaultStyles(c.Request.Context()); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "Default styles initialized successfully", nil)
}
