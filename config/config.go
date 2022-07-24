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

type ElasticConfig struct {
	Address []string
}

//Config define common config struct
type Config struct {
	SqlDB    *DBConfig      `json:"sql_db,omitempty"`
	Redis    *RedisConfig   `json:"redis,omitempty"`
	GraphDB  *GraphDBConfig `json:"graph,omitempty"`
	Elastic  *ElasticConfig `json:"elastic,omitempty"`
	LogLevel string         `json:"log_level,omitempty"`
	Port     string         `json:"port,omitempty"`
	S3Store  *S3Store       `json:"s3store,omitempty"`
}

type S3Store struct {
	Token   string
	Secret  string
	Bucket  string
	Address string
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

	// if cfg.GraphDB.Address == "" || cfg.GraphDB.Database == "" {
	// 	log.Info("graph database not init")
	// }

	// if len(cfg.Elastic.Address) == "" {
	// 	log.Info("elastic not init")
	// }
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
