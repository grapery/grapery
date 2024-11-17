package syncworker

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/service"
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
	srv := service.NewTeamsService()
	err = service.Run(srv, config.GlobalConfig)
	if err != nil {
		log.Fatal("start service failed")
	}
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	select {
	case s := <-sc:
		log.Info("signal : ", s.String())
	}
	return
}
