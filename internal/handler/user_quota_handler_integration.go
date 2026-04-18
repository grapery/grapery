package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// ============================================
// 鉴权集成示例
// ============================================
//
// 本文件展示如何将 UserQuotaHandler 与现有的鉴权系统集成
//
// 假设你有一个现有的 JWT 鉴权中间件，例如：
//   - AuthMiddleware() - 验证 JWT token 并设置 user_id 到 context
//

// SetupAuthenticatedRoutes 设置需要鉴权的用户配额路由
// 这是你应该在主路由配置中调用的方法
func SetupAuthenticatedRoutes(router *gin.Engine, quotaService *service.UserQuotaService, logger *zap.Logger) {
	// 创建 handler
	quotaHandler := NewUserQuotaHandler(quotaService, logger)

	// API v1 路由组
	apiV1 := router.Group("/api/v1")

	// ============== 应用你的鉴权中间件 ==============
	// 假设你有以下中间件之一：
	// 1. JWTAuthMiddleware - JWT 验证
	// 2. AuthMiddleware - 通用认证中间件
	// 3. RequireAuth - 需要认证

	// 方式 1: 如果你的中间件直接设置 user_id 到 context
	// apiV1.Use(YourAuthMiddleware())

	// 方式 2: 如果你的中间件需要包装
	// apiV1.Use(func(c *gin.Context) {
	//     // 你的鉴权逻辑
	//     userID := getUserIDFromToken(c)
	//     if userID == "" {
	//         c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	//         c.Abort()
	//         return
	//     }
	//     c.Set("user_id", userID)
	//     c.Next()
	// })

	// ============== 设置用户配额路由 ==============
	// 推荐使用 /me/* 路由（更简洁，自动获取当前用户）
	SetupCurrentUserQuotaRoutes(apiV1, quotaHandler)

	// 如果需要支持 /users/:userId/* 格式，也可以设置
	SetupUserQuotaRoutes(apiV1, quotaHandler)
}

// ============================================
// 现有鉴权中间件示例
// ============================================

// JWTAuthMiddleware JWT 鉴权中间件示例
// 这是你应该已经存在的中间件，这里只是示例
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// 验证 token（这里简化了，实际应该验证 JWT）
		// userID, err := validateJWTToken(authHeader)
		// if err != nil {
		//     c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		//     c.Abort()
		//     return
		// }

		// 示例：从 token 解析出 user_id
		// c.Set("user_id", userID)

		c.Next()
	}
}

// ============================================
// 路由设置示例
// ============================================

// 示例 1: 使用 /me/* 路由（推荐）
// 这种方式更简洁，前端不需要知道 user_id
//
// 前端调用：
// GET /api/v1/me/quota
// GET /api/v1/me/membership
// GET /api/v1/me/usage?period=month
func ExampleSetupMeRoutes(router *gin.Engine, quotaHandler *UserQuotaHandler) {
	apiV1 := router.Group("/api/v1")
	apiV1.Use(JWTAuthMiddleware()) // 你的 JWT 鉴权中间件

	SetupCurrentUserQuotaRoutes(apiV1, quotaHandler)
}

// 示例 2: 使用 /users/:userId/* 路由
// 这种方式需要前端传递 user_id，但会验证所有权
//
// 前端调用：
// GET /api/v1/users/{user_id}/quota
// GET /api/v1/users/{user_id}/membership
func ExampleSetupUserRoutes(router *gin.Engine, quotaHandler *UserQuotaHandler) {
	apiV1 := router.Group("/api/v1")
	apiV1.Use(JWTAuthMiddleware()) // 你的 JWT 鉴权中间件

	SetupUserQuotaRoutes(apiV1, quotaHandler)
}

// 示例 3: 完整的路由设置（包含其他路由）
func ExampleSetupAllRoutes(router *gin.Engine, quotaHandler *UserQuotaHandler) {
	// 公开路由（无需鉴权）
	public := router.Group("/api/v1")
	{
		// 会员等级信息（公开）
		public.GET("/memberships/tiers", quotaHandler.GetMembershipTiers)

		// 其他公开路由...
	}

	// 需要鉴权的路由
	authenticated := router.Group("/api/v1")
	authenticated.Use(JWTAuthMiddleware())
	{
		// 当前用户路由（推荐）
		SetupCurrentUserQuotaRoutes(authenticated, quotaHandler)

		// 其他需要鉴权的路由...
	}
}

// ============================================
// 请求/响应示例
// ============================================

// 请求示例：
//
// GET /api/v1/me/quota
// Headers:
//   Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
//
// 成功响应 (200 OK):
// {
//   "userId": "user_123",
//   "tokenBalance": 5000,
//   "tokenQuota": 10000,
//   "tokenUsed": 5000,
//   "tokenRemaining": 5000,
//   "storageUsed": 536870912,
//   "storageQuota": 107374182400,
//   "storageRemaining": 106837311488,
//   "resetDate": "2026-03-01T00:00:00Z"
// }
//
// 错误响应 (401 Unauthorized):
// {
//   "error": "unauthorized"
// }
//
// 错误响应 (403 Forbidden) - 当尝试访问其他用户信息时:
// {
//   "error": "access denied: you can only access your own information"
// }

// ============================================
// Swift 客户端示例
// ============================================

/*
 // UserProfileService.swift
 import Foundation

 struct UserProfileService {
     let baseURL = "https://api.example.com/api/v1"
     let accessToken: String

     // 获取当前用户配额信息
     func fetchQuotaInfo() async throws -> UserQuotaInfo {
         let url = URL(string: "\(baseURL)/me/quota")!
         var request = URLRequest(url: url)
         request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")

         let (data, response) = try await URLSession.shared.data(for: request)

         guard let httpResponse = response as? HTTPURLResponse,
               httpResponse.statusCode == 200 else {
             throw URLError(.badServerResponse)
         }

         return try JSONDecoder().decode(UserQuotaInfo.self, from: data)
     }

     // 获取当前用户会员信息
     func fetchMembershipInfo() async throws -> UserMembershipInfo {
         let url = URL(string: "\(baseURL)/me/membership")!
         var request = URLRequest(url: url)
         request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")

         let (data, response) = try await URLSession.shared.data(for: request)

         guard let httpResponse = response as? HTTPURLResponse,
               httpResponse.statusCode == 200 else {
             throw URLError(.badServerResponse)
         }

         return try JSONDecoder().decode(UserMembershipInfo.self, from: data)
     }

     // 获取用户使用统计
     func fetchUsageStatistics(period: String = "month") async throws -> UserUsageStatistics {
         var components = URLComponents(string: "\(baseURL)/me/usage", resolvingAgainstBaseURL: false)!
         components.queryItems = [URLQueryItem(name: "period", value: period)]
         guard let url = components.url else {
             throw URLError(.badURL)
         }

         var request = URLRequest(url: url)
         request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")

         let (data, response) = try await URLSession.shared.data(for: request)

         guard let httpResponse = response as? HTTPURLResponse,
               httpResponse.statusCode == 200 else {
             throw URLError(.badServerResponse)
         }

         return try JSONDecoder().decode(UserUsageStatistics.self, from: data)
     }
 }
*/
