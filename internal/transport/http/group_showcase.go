package http

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// GroupShowcaseHandler 小组展示相关处理器
type GroupShowcaseHandler struct {
	service GroupShowcaseService
}

// GroupShowcaseService 小组展示服务接口
type GroupShowcaseService interface {
	AddShowcase(ctx context.Context, groupID, userID string, req *domain.AddGroupShowcaseRequest) (*domain.GroupShowcase, error)
	RemoveShowcase(ctx context.Context, showcaseID, userID string) error
	GetGroupShowcases(ctx context.Context, groupID string, contentType domain.GroupShowcaseRelationType, limit, offset int) (*domain.ListGroupShowcasesResponse, error)
	UpdateShowcaseOrder(ctx context.Context, showcaseID, userID string, sortOrder int) error
}

// NewGroupShowcaseHandler 创建小组展示处理器
func NewGroupShowcaseHandler(service GroupShowcaseService) *GroupShowcaseHandler {
	return &GroupShowcaseHandler{service: service}
}

// AddShowcase 添加展示内容
// POST /api/groups/:id/showcases
func (h *GroupShowcaseHandler) AddShowcase(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "Group ID is required")
		return
	}

	var req domain.AddGroupShowcaseRequest
	if !BindJSON(c, &req) {
		return
	}

	showcase, err := h.service.AddShowcase(c.Request.Context(), groupID, userID, &req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, showcase)
}

// RemoveShowcase 移除展示内容
// DELETE /api/groups/:id/showcases/:showcaseId
func (h *GroupShowcaseHandler) RemoveShowcase(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	showcaseID := c.Param("showcaseId")
	if showcaseID == "" {
		InvalidParams(c, "Showcase ID is required")
		return
	}

	if err := h.service.RemoveShowcase(c.Request.Context(), showcaseID, userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "Showcase removed successfully"})
}

// GetGroupShowcases 获取小组展示列表
// GET /api/groups/:id/showcases
func (h *GroupShowcaseHandler) GetGroupShowcases(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "Group ID is required")
		return
	}

	// 获取内容类型筛选
	contentType := domain.GroupShowcaseRelationType(c.Query("type"))
	if contentType != domain.GroupShowcaseTypeFragment && contentType != domain.GroupShowcaseTypeStory {
		contentType = "" // 不限制类型
	}

	// 分页参数
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	resp, err := h.service.GetGroupShowcases(c.Request.Context(), groupID, contentType, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, resp)
}

// UpdateShowcaseOrder 更新展示排序
// PUT /api/groups/:id/showcases/:showcaseId/order
func (h *GroupShowcaseHandler) UpdateShowcaseOrder(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	showcaseID := c.Param("showcaseId")
	if showcaseID == "" {
		InvalidParams(c, "Showcase ID is required")
		return
	}

	var req struct {
		SortOrder int `json:"sortOrder" binding:"required"`
	}
	if !BindJSON(c, &req) {
		return
	}

	if err := h.service.UpdateShowcaseOrder(c.Request.Context(), showcaseID, userID, req.SortOrder); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "Showcase order updated successfully"})
}
