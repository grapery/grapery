package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// CreateInvitationCode 创建邀请码
// POST /api/invitation-codes
func (h *Handler) CreateInvitationCode(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req service.CreateInvitationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	code, err := h.svc.CreateInvitationCode(c.Request.Context(), userID, req)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, code)
}

// ListInvitationCodes 列出邀请码
// GET /api/invitation-codes
func (h *Handler) ListInvitationCodes(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	codes, err := h.svc.ListInvitationCodes(c.Request.Context(), userID, limit, offset)
	if err != nil {
		Error(c, CodeError, err.Error())
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
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	codeID := c.Param("id")
	if codeID == "" {
		InvalidParams(c, "invitation code id is required")
		return
	}

	code, err := h.svc.GetInvitationCode(c.Request.Context(), codeID)
	if err != nil {
		if err.Error() == "invitation code not found" {
			NotFound(c, "invitation code not found")
			return
		}
		Error(c, CodeError, err.Error())
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
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	codeID := c.Param("id")
	if codeID == "" {
		InvalidParams(c, "invitation code id is required")
		return
	}

	var req service.UpdateInvitationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	code, err := h.svc.UpdateInvitationCode(c.Request.Context(), userID, codeID, req)
	if err != nil {
		if err.Error() == "invitation code not found" {
			NotFound(c, "invitation code not found")
			return
		}
		if err.Error() == "unauthorized: you can only update your own invitation codes" {
			Forbidden(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, code)
}

// DeleteInvitationCode 删除邀请码
// DELETE /api/invitation-codes/:id
func (h *Handler) DeleteInvitationCode(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	codeID := c.Param("id")
	if codeID == "" {
		InvalidParams(c, "invitation code id is required")
		return
	}

	err := h.svc.DeleteInvitationCode(c.Request.Context(), userID, codeID)
	if err != nil {
		if err.Error() == "invitation code not found" {
			NotFound(c, "invitation code not found")
			return
		}
		if err.Error() == "unauthorized: you can only delete your own invitation codes" {
			Forbidden(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
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
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	err := h.svc.ValidateInvitationCode(c.Request.Context(), req.Code)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"valid": true, "message": "invitation code is valid"})
}

