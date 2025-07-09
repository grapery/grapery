package main

import (
	"github.com/gin-gonic/gin"

	llmchathandler "github.com/grapery/grapery/service/llmchat/handler"
)

func main() {
	// 初始化数据库（示例用sqlite，生产可换为mysql/postgres等）

	// 初始化Gin
	r := gin.Default()

	// 注册llmchat相关路由
	llmchathandler.RegisterLLMChatRoutes(r)

	// 启动服务
	r.Run(":8080")
}
