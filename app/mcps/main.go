package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/service/mcps"
	"github.com/grapery/grapery/version"
)

var printVersion = flag.Bool("version", false, "app build version")
var configPath = flag.String("config", "config.json", "config file")
var serverAddr = flag.String("addr", ":8080", "server address")

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
		log.Fatal("Validate config failed : ", err)
	}

	// Create and initialize MCP service
	service := mcps.NewMcpService()
	err = service.Initialize(config.GlobalConfig)
	if err != nil {
		log.Fatal("initialize service failed : ", err)
	}

	// Create and start MCP server
	server := mcps.NewServer(service)
	go func() {
		if err := server.Start(*serverAddr); err != nil {
			log.Fatal("start server failed : ", err)
		}
	}()

	// Handle shutdown signals
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	select {
	case s := <-sc:
		log.Info("Received signal: ", s.String())
		if err := server.Stop(); err != nil {
			log.Error("Error stopping server: ", err)
		}
		if err := service.Shutdown(); err != nil {
			log.Error("Error shutting down service: ", err)
		}
	}
}
