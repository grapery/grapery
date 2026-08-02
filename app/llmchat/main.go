package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/pkg/story"
	"github.com/grapery/grapery/service/feedback"
	"github.com/grapery/grapery/service/llmchat"
	llmchathandler "github.com/grapery/grapery/service/llmchat/handler"
	"github.com/grapery/grapery/service/mcps"
	"github.com/grapery/grapery/version"
)

var printVersion = flag.Bool("version", false, "app build version")
var configPath = flag.String("config", "llmchat.json", "config file")

func main() {
	flag.Parse()
	if *printVersion {
		version.PrintFullVersionInfo()
		return
	}
	err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatal("read config failed : ", err)
	}
	err = config.ValiedConfig(config.GlobalConfig)
	if err != nil {
		log.Fatal("Valied config failed : ", err)
	}
	err = llmchat.Init(config.GlobalConfig)
	if err != nil {
		log.Fatal("init llmchat failed : ", err)
	}
	mcpService := mcps.NewMcpService(story.GetStoryEngine())
	if err := mcpService.Initialize(config.GlobalConfig); err != nil {
		log.Fatal("init mcp service failed : ", err)
	}
	mcpServer := mcps.NewServer(mcpService)

	mcpPort := ""
	if config.GlobalConfig.LLMchat != nil {
		mcpPort = config.GlobalConfig.LLMchat.MCPPort
	}
	if mcpPort == "" {
		if envPort := os.Getenv("MCP_PORT"); envPort != "" {
			mcpPort = envPort
		} else {
			mcpPort = "8081"
		}
	}

	go func() {
		addr := fmt.Sprintf("0.0.0.0:%s", mcpPort)
		if err := mcpServer.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("mcp server stopped with error: %v", err)
		}
	}()
	defer func() {
		if err := mcpServer.Stop(); err != nil {
			log.Printf("failed to stop mcp server: %v", err)
		}
	}()

	// 初始化Gin
	r := gin.Default()

	// 注册llmchat相关路由
	llmchathandler.RegisterLLMChatRoutes(r)

	// 注册反馈相关路由
	feedback.RegisterFeedbackRoutes(r)

	// 启动服务
	r.Run(fmt.Sprintf("0.0.0.0:%s", config.GlobalConfig.LLMchat.HttpPort))
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	<-sc
	log.Println("llmchat server stopped")
}
