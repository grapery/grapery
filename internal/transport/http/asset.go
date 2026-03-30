package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// ListAssets 获取资源列表
// GET /api/assets
func (h *Handler) ListAssets(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	_ = c.Query("type") // assetType: image, audio, video - TODO: use for filtering

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// TODO: 实现从数据库获取资源列表
	// assets, total, err := h.svc.ListAssets(c.Request.Context(), userID, assetType, page, pageSize)

	Success(c, gin.H{
		"assets": []interface{}{},
		"page":   page,
		"pageSize": pageSize,
		"total":  0,
	})
}

// GetAsset 获取单个资源详情
// GET /api/assets/:id
func (h *Handler) GetAsset(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	assetID := c.Param("id")
	if assetID == "" {
		InvalidParams(c, "asset id is required")
		return
	}

	// TODO: 实现从数据库获取资源详情
	// asset, err := h.svc.GetAsset(c.Request.Context(), assetID, userID)

	Success(c, gin.H{
		"id": assetID,
		// "asset": asset,
	})
}

// CreateAsset 创建资源记录
// POST /api/assets
func (h *Handler) CreateAsset(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		URL      string `json:"url" binding:"required"`
		Type     string `json:"type" binding:"required"` // image, audio, video
		Name     string `json:"name"`
		MimeType string `json:"mimeType"`
		Size     int64  `json:"size"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// TODO: 实现创建资源记录
	// asset, err := h.svc.CreateAsset(c.Request.Context(), &req, userID)

	Success(c, gin.H{
		"message": "asset created",
		// "asset": asset,
	})
}

// UpdateAsset 更新资源
// PUT /api/assets/:id
func (h *Handler) UpdateAsset(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	assetID := c.Param("id")
	if assetID == "" {
		InvalidParams(c, "asset id is required")
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// TODO: 实现更新资源
	// asset, err := h.svc.UpdateAsset(c.Request.Context(), assetID, &req, userID)

	Success(c, gin.H{
		"message": "asset updated",
		"id":      assetID,
	})
}

// DeleteAsset 删除资源
// DELETE /api/assets/:id
func (h *Handler) DeleteAsset(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	assetID := c.Param("id")
	if assetID == "" {
		InvalidParams(c, "asset id is required")
		return
	}

	// TODO: 实现删除资源
	// err := h.svc.DeleteAsset(c.Request.Context(), assetID, userID)

	Success(c, gin.H{
		"message": "asset deleted",
		"id":      assetID,
	})
}
