package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware JWT鉴权中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" || !strings.HasPrefix(tokenStr, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing or invalid token"})
			return
		}

		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
		logrus.WithFields(logrus.Fields{
			"path":         c.Request.URL.Path,
			"token_length": len(tokenStr),
		}).Debug("AuthMiddleware: validating Bearer token")

		claims, err := auth.ParseToken(tokenStr)
		if err != nil {
			// Align with Grapery API AuthMiddleware (HTTP 401 + business codes for App refresh logic).
			switch {
			case errors.Is(err, auth.ErrSecretNotSet):
				logrus.Error("AuthMiddleware: JWT secret not configured on VipPay")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"msg":     "jwt secret not configured",
					"message": "jwt secret not configured",
				})
			case errors.Is(err, auth.ErrExpiredToken):
				logrus.WithError(err).Warn("AuthMiddleware: token expired")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    -8,
					"msg":     "token expired",
					"message": "token expired",
					"data":    nil,
				})
			default:
				logrus.WithError(err).Warn("AuthMiddleware: invalid token")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    -9,
					"msg":     "invalid token",
					"message": "invalid token",
					"data":    nil,
				})
			}
			return
		}

		// 将用户信息存入 gin 上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		// 将用户信息注入到 request context 中，供下游 service/repository 使用
		ctx := auth.ContextWithUserInfo(c.Request.Context(), claims.UserID, claims.Username, claims.Email)
		c.Request = c.Request.WithContext(ctx)

		logrus.WithFields(logrus.Fields{
			"user_id":  claims.UserID,
			"username": claims.Username,
		}).Debug("AuthMiddleware: user authenticated")

		c.Next()
	}
}

// GetUserIDFromContext 从上下文中获取用户ID（返回字符串）
func GetUserIDFromContext(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

// GetUserIDAsUint64 从上下文中获取用户ID（返回uint64，用于兼容）
func GetUserIDAsUint64(c *gin.Context) uint64 {
	userID := GetUserIDFromContext(c)
	if userID == "" {
		return 0
	}
	// 简单处理：如果是数字字符串，可以转换
	// 这里暂时返回0，实际使用中根据UserID格式处理
	return 0
}

// GetUsername 从上下文中获取用户名
func GetUsername(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}
	if name, ok := username.(string); ok {
		return name
	}
	return ""
}

// GetEmail 从上下文中获取邮箱
func GetEmail(c *gin.Context) string {
	email, exists := c.Get("email")
	if !exists {
		return ""
	}
	if e, ok := email.(string); ok {
		return e
	}
	return ""
}
