package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	servicepkg "github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
)

// EnsureActiveUser rejects JWTs after auth for accounts that cannot use the API (runs after AuthMiddleware).
func (h *Handler) EnsureActiveUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := authPkg.GetUserID(c)
		if userID == "" {
			Unauthorized(c, "not authenticated")
			c.Abort()
			return
		}
		if err := h.svc.EnsureActiveSessionUser(c.Request.Context(), userID); err != nil {
			Unauthorized(c, "account no longer available")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RestrictPendingDeletionWrites blocks mutating APIs while the account waits for irrevocable deletion,
// except: cancel/re-request deletion, and minimal device-token hygiene (register/unregister/badge).
func (h *Handler) RestrictPendingDeletionWrites() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := authPkg.GetUserID(c)
		if userID == "" {
			Unauthorized(c, "not authenticated")
			c.Abort()
			return
		}
		u, err := h.svc.SessionUser(c.Request.Context(), userID)
		if err != nil || u == nil {
			Unauthorized(c, "account no longer available")
			c.Abort()
			return
		}
		if u.Status != string(common.StatusPendingDeletion) {
			c.Next()
			return
		}

		m := strings.ToUpper(c.Request.Method)
		switch m {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		path := c.Request.URL.Path
		switch {
		case m == http.MethodPost && path == "/api/v1/auth/account/deletion/cancel":
			c.Next()
			return
		case m == http.MethodDelete && path == "/api/v1/auth/account":
			c.Next()
			return
		case m == http.MethodPost &&
			(path == "/api/v1/devices/register" ||
				path == "/api/v1/devices/unregister" ||
				path == "/api/v1/devices/badge"):
			c.Next()
			return
		default:
			Forbidden(c, "account deletion pending — reads, deletion controls, and device-token updates allowed")
			c.Abort()
		}
	}
}

// RequestAccountDeletion queues account closure after grace window + risk ACK + OTP proof. DELETE /api/v1/auth/account
func (h *Handler) RequestAccountDeletion(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	var body struct {
		RiskAcknowledged bool `json:"riskAcknowledged"`
	}
	if !BindJSON(c, &body) {
		return
	}
	st, err := h.svc.RequestAccountDeletion(c.Request.Context(), userID, &servicepkg.AccountDeletionSubmit{
		RiskAcknowledged: body.RiskAcknowledged,
	})
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, st)
}

// SendAccountDeletionSMS POST /api/v1/auth/account/deletion/send-sms-code
func (h *Handler) SendAccountDeletionSMS(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	ip := utils.GetClientIP(
		c.Request.RemoteAddr,
		c.GetHeader("X-Forwarded-For"),
		c.GetHeader("X-Real-IP"),
	)
	err := h.svc.SendAccountDeletionSMS(c.Request.Context(), userID, ip)
	if err != nil {
		if errors.Is(err, servicepkg.ErrSMSSendRateLimited) {
			RateLimitExceeded(c, err.Error())
			return
		}
		if errors.Is(err, servicepkg.ErrAccountDeletionSMSCacheRequired) {
			InternalError(c, err.Error())
			return
		}
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"message": "deletion verification code sent"})
}

// VerifyAccountDeletionSMS POST /api/v1/auth/account/deletion/verify-sms-code
func (h *Handler) VerifyAccountDeletionSMS(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code" binding:"required,len=6"`
	}
	if !BindJSON(c, &req) {
		return
	}
	if err := h.svc.VerifyAccountDeletionSMS(c.Request.Context(), userID, req.Code); err != nil {
		if errors.Is(err, servicepkg.ErrAccountDeletionSMSCacheRequired) {
			InternalError(c, err.Error())
			return
		}
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"message": "deletion phone verified"})
}

// GetAccountDeletionStatus returns pending deletion timeline. GET /api/v1/auth/account/deletion
func (h *Handler) GetAccountDeletionStatus(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	st, err := h.svc.GetAccountDeletionStatus(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, st)
}

// CancelAccountDeletion restores the account within the grace window. POST /api/v1/auth/account/deletion/cancel
func (h *Handler) CancelAccountDeletion(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if err := h.svc.CancelAccountDeletion(c.Request.Context(), userID); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"message": "account deletion cancelled"})
}
