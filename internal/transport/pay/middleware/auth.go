package middleware

import (
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
		logrus.WithField("token_length", len(tokenStr)).Debug("AuthMiddleware: processing token")

		// TEMP(通知联调): 打印完整 Bearer JWT，测试完成后删除本段日志。
		logrus.WithFields(logrus.Fields{
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"bearer_token": tokenStr,
		}).Info("TEMP debug bearer token (vippay) — remove this log block after tests")

		claims, err := auth.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid token"})
			logrus.WithError(err).Error("AuthMiddleware: invalid token")
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
