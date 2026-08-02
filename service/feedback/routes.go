package feedback

import (
	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/service/llmchat/middleware"
)

// RegisterFeedbackRoutes 注册反馈相关路由
func RegisterFeedbackRoutes(r *gin.Engine) {
	// 创建反馈处理器
	handler := NewHandler()

	// 创建反馈API组
	api := r.Group("/api/feedback")
	api.Use(middleware.AuthMiddleware())
	// 注意：这里需要添加认证中间件，但为了简化，暂时不添加
	// 在实际使用中，应该添加类似 middleware.AuthMiddleware() 的中间件
	{
		// 创建反馈
		api.POST("", handler.CreateFeedback)

		// 获取用户反馈列表
		api.GET("", handler.GetUserFeedbackList)

		// 获取反馈详情
		api.GET("/:id", handler.GetFeedbackDetail)

		// 获取反馈类型列表
		api.GET("/types", handler.GetFeedbackTypes)

		// 获取反馈状态列表
		api.GET("/statuses", handler.GetFeedbackStatuses)

		// 健康检查
		api.GET("/health", handler.HealthCheck)
	}
}
