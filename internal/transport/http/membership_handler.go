package http

import (
	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ========== Membership Handlers ==========

// ListMembershipPlans 列出可用会员方案
// GET /api/v1/membership/plans
func (h *Handler) ListMembershipPlans(c *gin.Context) {
	plans, err := h.svc.ListMembershipPlans(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"plans": plans,
	})
}

// GetCurrentMembership 获取当前用户会员信息
// GET /api/v1/membership/current
func (h *Handler) GetCurrentMembership(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	membership, err := h.svc.GetUserMembership(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, membership)
}

// SubscribeMembership 订阅会员方案
// POST /api/v1/membership/subscribe
func (h *Handler) SubscribeMembership(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req domain.SubscribeRequest
	if !BindJSON(c, &req) {
		return
	}

	response, err := h.svc.SubscribeMembership(c.Request.Context(), userID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, response)
}

// CancelMembership 取消会员订阅
// POST /api/v1/membership/cancel
func (h *Handler) CancelMembership(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	err := h.svc.CancelMembership(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "subscription cancelled successfully"})
}

// GetMembershipUsage 获取会员使用量
// GET /api/v1/membership/usage
func (h *Handler) GetMembershipUsage(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	usage, err := h.svc.GetMembershipUsage(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, usage)
}
