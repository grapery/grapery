package config

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

var GlobalConfig *Config

type DBConfig struct {
	Database string `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type RedisConfig struct {
	Database     string `json:"database,omitempty"`
	PingInterval int    `json:"ping_interval,omitempty"`
}

//Config define common config struct
type Config struct {
	DBconfig *DBConfig    `json:"d_bconfig,omitempty"`
	Redis    *RedisConfig `json:"redis,omitempty"`
	LogLevel int          `json:"log_level,omitempty"`
	Port     string       `json:"port,omitempty"`
}

func LoadConfig(configPath string) error {
	log.Info("load config : ", configPath)
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Errorf("read config file error : %v", err)
		return err
	}
	err = json.Unmarshal(data, GlobalConfig)
	if err != nil {
		log.Errorf("config file format wrong :%v", err)
		return err
	}
	return nil
}
