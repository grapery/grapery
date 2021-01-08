package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

var GlobalConfig = new(Config)

type DBConfig struct {
	Database string `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type RedisConfig struct {
	Address      string `json:"address,omitempty"`
	Password     string `json:"password,omitempty"`
	Database     string `json:"database,omitempty"`
	PingInterval int    `json:"ping_interval,omitempty"`
}

type GraphDBConfig struct {
	Address  string `json:"address,omitempty"`
	Database string `json:"database,omitempty"`
}

//Config define common config struct
type Config struct {
	SqlDB    *DBConfig      `json:"sql_db,omitempty"`
	Redis    *RedisConfig   `json:"redis,omitempty"`
	GraphDB  *GraphDBConfig `json:"graph,omitempty"`
	LogLevel string         `json:"log_level,omitempty"`
	Port     string         `json:"port,omitempty"`
}

func ValiedConfig(cfg *Config) error {
	if cfg.Port == "" {
		return fmt.Errorf("server port not set")
	}
	if cfg.SqlDB.Database == "" || cfg.SqlDB.Password == "" || cfg.SqlDB.Username == "" {
		return fmt.Errorf("sql database not set")
	}
	if cfg.Redis.Database == "" {
		return fmt.Errorf("redis cfg not set")
	}
	return nil
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
