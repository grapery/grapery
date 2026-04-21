package http

import (
	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// EnsureActiveUser rejects JWTs for deleted, suspended, or missing users (runs after AuthMiddleware).
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

// DeleteMyAccount permanently closes the current account (soft-delete + anonymize). DELETE /api/v1/auth/account
func (h *Handler) DeleteMyAccount(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteMyAccount(c.Request.Context(), userID); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"message": "account deleted"})
}
