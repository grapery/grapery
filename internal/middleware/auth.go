package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT 鉴权中间件
// 验证 JWT token 并将 user_id 设置到 context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// 检查 Bearer token 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]

		// 验证 token（这里需要你实现 JWT 验证逻辑）
		userID, err := validateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// 将 user_id 设置到 context，供后续 handler 使用
		c.Set("user_id", userID)

		c.Next()
	}
}

// validateToken 验证 JWT token 并返回 user_id
// 这是一个示例实现，你需要根据你的 JWT 库进行调整
func validateToken(token string) (string, error) {
	// TODO: 实现实际的 JWT 验证逻辑
	//
	// 示例代码（使用 github.com/golang-jwt/jwt）:
	//
	// claims := &Claims{}
	// tokenParser, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
	//     return []byte(yourSecretKey), nil
	// })
	// if err != nil || !tokenParser.Valid {
	//     return "", fmt.Errorf("invalid token")
	// }
	// return claims.UserID, nil
	//
	// 或者使用其他 JWT 库...

	// 临时实现：仅用于测试
	// 在生产环境中，请删除这段代码并实现真正的 JWT 验证
	if token == "test_token" {
		return "test_user_id", nil
	}

	return "", nil // 返回空会触发鉴权失败
}

// OptionalAuthMiddleware 可选鉴权中间件
// 如果提供了 token 则验证，否则继续处理
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 没有 token，继续处理（用户未登录）
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]

		userID, err := validateToken(token)
		if err == nil {
			// token 有效，设置 user_id
			c.Set("user_id", userID)
		}

		c.Next()
	}
}

// RequireAuth 要求用户已登录的辅助函数
// 可以在单个路由中使用
func RequireAuth() gin.HandlerFunc {
	return AuthMiddleware()
}

// GetUserID 从 context 获取当前用户 ID
// 这是一个辅助函数，可以在 handler 中使用
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	userIDStr, ok := userID.(string)
	if !ok {
		return "", false
	}
	return userIDStr, true
}

// MustGetUserID 从 context 获取当前用户 ID
// 如果不存在则返回 401 错误
func MustGetUserID(c *gin.Context) string {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		c.Abort()
		return ""
	}
	return userID
}

// ============================================
// 使用示例
// ============================================

/*
// 示例 1: 全局应用鉴权
func SetupRoutes(router *gin.Engine) {
    api := router.Group("/api")
    api.Use(middleware.AuthMiddleware()) // 所有路由都需要鉴权

    // 设置用户配额路由
    handler.SetupCurrentUserQuotaRoutes(api.Group("/v1"), quotaHandler)
}

// 示例 2: 部分路由需要鉴权
func SetupRoutes(router *gin.Engine) {
    api := router.Group("/api")

    // 公开路由
    api.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // 需要鉴权的路由
    authenticated := api.Group("")
    authenticated.Use(middleware.AuthMiddleware())

    handler.SetupCurrentUserQuotaRoutes(authenticated.Group("/v1"), quotaHandler)
}

// 示例 3: 单个路由使用鉴权
func SetupRoutes(router *gin.Engine) {
    api := router.Group("/api")

    // 公开路由
    api.GET("/public", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "public"})
    })

    // 需要鉴权的单个路由
    api.GET("/private", middleware.AuthMiddleware(), func(c *gin.Context) {
        userID := middleware.MustGetUserID(c)
        c.JSON(200, gin.H{"user_id": userID})
    })
}

// 示例 4: 在 handler 中获取用户 ID
func SomeHandler(c *gin.Context) {
    // 方式 1: 使用 MustGetUserID（会自动处理错误）
    userID := middleware.MustGetUserID(c)

    // 方式 2: 使用 GetUserID（手动处理错误）
    userID, ok := middleware.GetUserID(c)
    if !ok {
        c.JSON(401, gin.H{"error": "unauthorized"})
        return
    }

    // 业务逻辑...
    c.JSON(200, gin.H{"user_id": userID})
}
*/
