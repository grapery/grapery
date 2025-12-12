package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Context key types for type safety
type contextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "userID"
	// UsernameKey is the context key for username
	UsernameKey contextKey = "username"
	// EmailKey is the context key for email
	EmailKey contextKey = "email"
)

// ContextWithUserID adds user ID to context
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// ContextWithUserInfo adds all user info to context
func ContextWithUserInfo(ctx context.Context, userID, username, email string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, UsernameKey, username)
	ctx = context.WithValue(ctx, EmailKey, email)
	return ctx
}

// UserIDFromContext retrieves user ID from context
func UserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// UsernameFromContext retrieves username from context
func UsernameFromContext(ctx context.Context) string {
	if username, ok := ctx.Value(UsernameKey).(string); ok {
		return username
	}
	return ""
}

// EmailFromContext retrieves email from context
func EmailFromContext(ctx context.Context) string {
	if email, ok := ctx.Value(EmailKey).(string); ok {
		return email
	}
	return ""
}

// AuthMiddleware JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 获取 Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    -2,
				"message": "missing authorization header",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查格式: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    -9,
				"message": "invalid token",
				"data":    nil,
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析 Token
		claims, err := ParseToken(tokenString)
		if err != nil {
			if err == ErrExpiredToken {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    -8,
					"message": "token expired",
					"data":    nil,
				})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    -9,
					"message": "invalid token",
					"data":    nil,
				})
			}
			c.Abort()
			return
		}

		// 将用户信息存入 gin 上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		// 将用户信息注入到 request context 中，供下游 service/repository 使用
		ctx := ContextWithUserInfo(c.Request.Context(), claims.UserID, claims.Username, claims.Email)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（不强制要求登录）
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims, err := ParseToken(tokenString)
		if err == nil {
			// 将用户信息存入 gin 上下文
			c.Set("userID", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("email", claims.Email)

			// 将用户信息注入到 request context 中
			ctx := ContextWithUserInfo(c.Request.Context(), claims.UserID, claims.Username, claims.Email)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("userID"); exists {
		return userID.(string)
	}
	return ""
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		return username.(string)
	}
	return ""
}

// GetEmail 从上下文获取邮箱
func GetEmail(c *gin.Context) string {
	if email, exists := c.Get("email"); exists {
		return email.(string)
	}
	return ""
}

// RequireAuth 检查是否已认证
func RequireAuth(c *gin.Context) bool {
	return GetUserID(c) != ""
}
