package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/service/llmchat"
	llmchathandler "github.com/grapery/grapery/service/llmchat/handler"
	"github.com/grapery/grapery/version"
)

var printVersion = flag.Bool("version", false, "app build version")
var configPath = flag.String("config", "config.json", "config file")

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
	// 初始化Gin
	r := gin.Default()

	// 注册llmchat相关路由
	llmchathandler.RegisterLLMChatRoutes(r)

	// 启动服务
	r.Run(":8060")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	<-sc
	log.Println("llmchat server stopped")
}
