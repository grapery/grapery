package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// GetReferralCode 获取用户专属邀请码
// GET /api/v1/referrals/code
func (h *Handler) GetReferralCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	referralCode, err := h.svc.GetOrCreateReferralCode(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"referralCode": referralCode,
	})
}

// GetInviteShareContent 获取邀请分享内容
// GET /api/v1/referrals/share
func (h *Handler) GetInviteShareContent(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	content, err := h.svc.GetInviteShareContent(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, content)
}

// GetReferralStats 获取用户邀请统计
// GET /api/v1/referrals/stats
func (h *Handler) GetReferralStats(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	stats, err := h.svc.GetReferralStats(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, stats)
}

// GetReferrals 获取用户邀请列表
// GET /api/v1/referrals
func (h *Handler) GetReferrals(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	referrals, err := h.svc.GetReferrals(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"referrals": referrals,
		"count":     len(referrals),
	})
}

// UseReferralCode 使用邀请码（注册后调用）
// POST /api/v1/referrals/use
func (h *Handler) UseReferralCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req domain.CreateReferralRequest
	if !BindJSON(c, &req) {
		return
	}

	response, err := h.svc.UseReferralCode(c.Request.Context(), userID, req.ReferralCode)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, response)
}

// GetUserPoints 获取用户积分
// GET /api/v1/users/:id/points
func (h *Handler) GetUserPoints(c *gin.Context) {
	userID, ok := RequireParam(c, "id")
	if !ok {
		return
	}

	points, err := h.svc.GetUserPoints(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"points": points,
	})
}
