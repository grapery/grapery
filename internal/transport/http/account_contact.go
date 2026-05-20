package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	servicepkg "github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
)

func handleAccountContactError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, servicepkg.ErrSMSSendRateLimited) {
		RateLimitExceeded(c, err.Error())
		return
	}
	if errors.Is(err, servicepkg.ErrAccountContactPhoneChangeDailyLimit) {
		RateLimitExceeded(c, err.Error())
		return
	}
	if errors.Is(err, servicepkg.ErrAccountContactCacheRequired) {
		InternalError(c, err.Error())
		return
	}
	var locked *servicepkg.AccountContactLockedError
	if errors.As(err, &locked) {
		ErrorWithData(c, CodeForbidden, locked.Error(), gin.H{"lockedUntil": locked.LockedUntil})
		return
	}
	var invalid *servicepkg.AccountContactInvalidCodeError
	if errors.As(err, &invalid) {
		ErrorWithData(c, CodeInvalidParams, invalid.Error(), gin.H{"attemptsRemaining": invalid.AttemptsRemaining})
		return
	}
	if errors.Is(err, servicepkg.ErrAccountContactEmailRegistered) {
		DuplicateEntry(c, err.Error())
		return
	}
	HandleError(c, err)
}

// SendAccountContactPhoneSMS POST /api/v1/auth/account/phone/send-sms-code
func (h *Handler) SendAccountContactPhoneSMS(c *gin.Context) {
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
	if err := h.svc.SendAccountContactPhoneSMS(c.Request.Context(), userID, req.Phone, ip); err != nil {
		handleAccountContactError(c, err)
		return
	}
	Success(c, gin.H{"message": "verification code sent"})
}

// VerifyAccountContactPhoneSMS POST /api/v1/auth/account/phone/verify-sms-code
func (h *Handler) VerifyAccountContactPhoneSMS(c *gin.Context) {
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
	if err := h.svc.VerifyAccountContactPhoneSMS(c.Request.Context(), userID, req.Phone, req.Code); err != nil {
		handleAccountContactError(c, err)
		return
	}
	Success(c, gin.H{"message": "phone verified"})
}

// SendAccountContactEmailCode POST /api/v1/auth/account/email/send-verification-code
func (h *Handler) SendAccountContactEmailCode(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if !BindJSON(c, &req) {
		return
	}
	ip := utils.GetClientIP(
		c.Request.RemoteAddr,
		c.GetHeader("X-Forwarded-For"),
		c.GetHeader("X-Real-IP"),
	)
	if err := h.svc.SendAccountContactEmailCode(c.Request.Context(), userID, req.Email, ip); err != nil {
		handleAccountContactError(c, err)
		return
	}
	Success(c, gin.H{"message": "verification code sent"})
}

// ConfirmAccountContactEmail POST /api/v1/auth/account/email/verify
func (h *Handler) ConfirmAccountContactEmail(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Code  string `json:"code" binding:"required,len=6"`
	}
	if !BindJSON(c, &req) {
		return
	}
	if err := h.svc.ConfirmAccountContactEmail(c.Request.Context(), userID, req.Email, req.Code); err != nil {
		handleAccountContactError(c, err)
		return
	}
	Success(c, gin.H{"message": "email verified"})
}
