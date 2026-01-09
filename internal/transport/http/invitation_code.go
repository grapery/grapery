package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// CreateInvitationCode 创建邀请码
// POST /api/invitation-codes
func (h *Handler) CreateInvitationCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req service.CreateInvitationCodeRequest
	if !BindJSON(c, &req) {
		return
	}

	code, err := h.svc.CreateInvitationCode(c.Request.Context(), userID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, code)
}

// ListInvitationCodes 列出邀请码
// GET /api/invitation-codes
func (h *Handler) ListInvitationCodes(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	codes, err := h.svc.ListInvitationCodes(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"codes": codes,
		"count": len(codes),
	})
}

// GetInvitationCode 获取邀请码信息
// GET /api/invitation-codes/:id
func (h *Handler) GetInvitationCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	codeID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	code, err := h.svc.GetInvitationCode(c.Request.Context(), codeID)
	if err != nil {
		HandleError(c, err)
		return
	}

	// 验证权限（只有创建者可以查看）
	if code.CreatedBy != userID {
		Forbidden(c, "you can only view your own invitation codes")
		return
	}

	Success(c, code)
}

// UpdateInvitationCode 更新邀请码
// PUT /api/invitation-codes/:id
func (h *Handler) UpdateInvitationCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	codeID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	var req service.UpdateInvitationCodeRequest
	if !BindJSON(c, &req) {
		return
	}

	code, err := h.svc.UpdateInvitationCode(c.Request.Context(), userID, codeID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, code)
}

// DeleteInvitationCode 删除邀请码
// DELETE /api/invitation-codes/:id
func (h *Handler) DeleteInvitationCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	codeID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	err := h.svc.DeleteInvitationCode(c.Request.Context(), userID, codeID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "invitation code deleted successfully"})
}

// ValidateInvitationCode 验证邀请码（公开接口）
// POST /api/invitation-codes/validate
func (h *Handler) ValidateInvitationCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if !BindJSON(c, &req) {
		return
	}

	err := h.svc.ValidateInvitationCode(c.Request.Context(), req.Code)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"valid": true, "message": "invitation code is valid"})
}
