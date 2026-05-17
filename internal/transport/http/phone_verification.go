package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
)

// SendPhoneSMSVerificationCode POST /api/v1/auth/phone/send-sms-code
func (h *Handler) SendPhoneSMSVerificationCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if !BindJSON(c, &req) {
		return
	}
	ip := utils.GetClientIP(
		c.Request.RemoteAddr,
		c.GetHeader("X-Forwarded-For"),
		c.GetHeader("X-Real-IP"),
	)
	if err := h.svc.SendPhoneSMSVerificationCode(c.Request.Context(), userID, req.Phone, ip); err != nil {
		if errors.Is(err, service.ErrSMSSendRateLimited) {
			RateLimitExceeded(c, err.Error())
			return
		}
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"message": "verification code sent"})
}

// VerifyPhoneSMSCode POST /api/v1/auth/phone/verify-sms-code
func (h *Handler) VerifyPhoneSMSCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required,len=6"`
	}
	if !BindJSON(c, &req) {
		return
	}
	if err := h.svc.VerifyPhoneSMSCode(c.Request.Context(), userID, req.Phone, req.Code); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"message": "phone verified"})
}
