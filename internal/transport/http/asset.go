package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// ListAssets 获取资产列表
// GET /api/assets
func (h *Handler) ListAssets(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	assetType := c.DefaultQuery("type", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	assets, err := h.svc.ListAssets(c.Request.Context(), userID, assetType, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"assets": assets,
		"count":  len(assets),
	})
}

// GetAsset 获取资产详情
// GET /api/assets/:id
func (h *Handler) GetAsset(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	assetID := c.Param("id")
	asset, err := h.svc.GetAsset(c.Request.Context(), assetID, userID)
	if err != nil {
		NotFound(c, "asset not found")
		return
	}

	Success(c, asset)
}

// CreateAsset 创建资产
// POST /api/assets
func (h *Handler) CreateAsset(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		Type      string   `json:"type" binding:"required"`
		Name      string   `json:"name" binding:"required"`
		URL       string   `json:"url" binding:"required"`
		Thumbnail string   `json:"thumbnail"`
		Size      int64    `json:"size"`
		MimeType  string   `json:"mimeType"`
		Width     int      `json:"width"`
		Height    int      `json:"height"`
		Duration  int      `json:"duration"`
		Tags      []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	asset := &domain.Asset{
		UserID:    userID,
		Type:      req.Type,
		Name:      req.Name,
		URL:       req.URL,
		Thumbnail: req.Thumbnail,
		Size:      req.Size,
		MimeType:  req.MimeType,
		Width:     req.Width,
		Height:    req.Height,
		Duration:  req.Duration,
		Tags:      req.Tags,
	}

	if err := h.svc.CreateAsset(c.Request.Context(), asset); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, asset)
}

// UpdateAsset 更新资产
// PUT /api/assets/:id
func (h *Handler) UpdateAsset(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	assetID := c.Param("id")

	var req struct {
		Name *string   `json:"name"`
		Tags *[]string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	updateReq := &service.AssetUpdateRequest{
		Name: req.Name,
		Tags: req.Tags,
	}

	asset, err := h.svc.UpdateAsset(c.Request.Context(), assetID, userID, updateReq)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, asset)
}

// DeleteAsset 删除资产
// DELETE /api/assets/:id
func (h *Handler) DeleteAsset(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	assetID := c.Param("id")
	if err := h.svc.DeleteAsset(c.Request.Context(), assetID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "asset deleted successfully"})
}
