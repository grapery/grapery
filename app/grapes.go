package main

import "fmt"
import "flag"
import "github.com/grapery/grapery/version"
import "github.com/grapery/grapery/config"
import log "github.com/sirupsen/logrus"

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
}
