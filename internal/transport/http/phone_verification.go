package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
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
	h.logger.Info("phone sms send HTTP",
		zap.String("user_id", userID),
		zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
		zap.String("client_ip", ip),
		zap.String("xff", c.GetHeader("X-Forwarded-For")),
		zap.String("x_real_ip", c.GetHeader("X-Real-IP")),
		zap.String("user_agent", c.Request.UserAgent()),
	)

	if err := h.svc.SendPhoneSMSVerificationCode(c.Request.Context(), userID, req.Phone, ip); err != nil {
		if errors.Is(err, service.ErrSMSSendRateLimited) {
			h.logger.Warn("phone sms send HTTP: rate limited",
				zap.String("user_id", userID),
				zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
				zap.String("client_ip", ip),
			)
			RateLimitExceeded(c, err.Error())
			return
		}
		h.logger.Warn("phone sms send HTTP: failed",
			zap.String("user_id", userID),
			zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
			zap.String("client_ip", ip),
			zap.Error(err),
		)
		HandleError(c, err)
		return
	}
	h.logger.Info("phone sms send HTTP: success",
		zap.String("user_id", userID),
		zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
		zap.String("client_ip", ip),
	)
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
	h.logger.Info("phone sms verify HTTP",
		zap.String("user_id", userID),
		zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
		zap.Int("code_len", len(req.Code)),
		zap.String("client_ip", c.ClientIP()),
	)

	if err := h.svc.VerifyPhoneSMSCode(c.Request.Context(), userID, req.Phone, req.Code); err != nil {
		h.logger.Warn("phone sms verify HTTP: failed",
			zap.String("user_id", userID),
			zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
			zap.Error(err),
		)
		HandleError(c, err)
		return
	}
	h.logger.Info("phone sms verify HTTP: success",
		zap.String("user_id", userID),
		zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
	)
	Success(c, gin.H{"message": "phone verified"})
}
