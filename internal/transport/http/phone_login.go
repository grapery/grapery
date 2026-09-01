package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// PhoneLoginSendSMSCode POST /api/auth/phone/login/send-sms-code
// 未认证：为「手机号验证码登录」发送验证码。
func (h *Handler) PhoneLoginSendSMSCode(c *gin.Context) {
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
	h.logger.Info("phone login sms send HTTP",
		zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
		zap.String("client_ip", ip),
	)

	if err := h.svc.SendPhoneLoginSMSCode(c.Request.Context(), req.Phone, ip); err != nil {
		if errors.Is(err, service.ErrSMSSendRateLimited) {
			h.logger.Warn("phone login sms send HTTP: rate limited",
				zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
				zap.String("client_ip", ip),
			)
			RateLimitExceeded(c, err.Error())
			return
		}
		h.logger.Warn("phone login sms send HTTP: failed",
			zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
			zap.String("client_ip", ip),
			zap.Error(err),
		)
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "verification code sent"})
}

// PhoneLoginVerify POST /api/auth/phone/login/verify
// 未认证：校验验证码并按手机号登录（不存在则自动注册），返回登录令牌。
func (h *Handler) PhoneLoginVerify(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Code     string `json:"code" binding:"required,len=6"`
		DeviceID string `json:"deviceId,omitempty"`
	}
	if !BindJSON(c, &req) {
		return
	}

	userAgent := c.Request.UserAgent()
	device, os, browser := utils.ParseUserAgent(userAgent)
	loginInfo := &service.LoginInfo{
		IPAddress: utils.GetClientIP(
			c.Request.RemoteAddr,
			c.GetHeader("X-Forwarded-For"),
			c.GetHeader("X-Real-IP"),
		),
		Device:    device,
		OS:        os,
		Browser:   browser,
		UserAgent: userAgent,
		DeviceID:  req.DeviceID,
	}

	h.logger.Info("phone login verify HTTP",
		zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
		zap.Int("code_len", len(req.Code)),
		zap.String("client_ip", loginInfo.IPAddress),
	)

	resp, err := h.svc.LoginWithPhoneSMS(c.Request.Context(), req.Phone, req.Code, loginInfo)
	if err != nil {
		h.logger.Warn("phone login verify HTTP: failed",
			zap.String("phone_masked", utils.MaskChinaPhone(req.Phone)),
			zap.Error(err),
		)
		HandleError(c, err)
		return
	}

	Success(c, resp)
}
