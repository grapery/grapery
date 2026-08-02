package config

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

var GlobalConfig = new(Config)

type DBConfig struct {
	Database string `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Address  string `json:"address,omitempty"`
}

type RedisConfig struct {
	Address      string `json:"address,omitempty"`
	Password     string `json:"password,omitempty"`
	Database     string `json:"database,omitempty"`
	PingInterval int    `json:"ping_interval,omitempty"`
}

type ElasticConfig struct {
	Address []string
}

type LLMchatConfig struct {
	HttpPort string `json:"http_port,omitempty"`
	MCPPort  string `json:"mcp_port,omitempty"`
}

// AppleOAuthConfig Apple OAuth2 配置
type AppleOAuthConfig struct {
	BundleID       string `json:"bundle_id,omitempty"`       // iOS App 的 Bundle Identifier，例如：com.yourapp.bundleid
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"` // 请求超时时间（秒），默认30秒
	CacheDuration  int    `json:"cache_duration,omitempty"`  // 公钥缓存时间（小时），默认1小时
}

// GoogleOAuthConfig Google OAuth2 配置
type GoogleOAuthConfig struct {
	ClientID       string `json:"client_id,omitempty"`       // Google OAuth2 Client ID
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"` // 请求超时时间（秒），默认30秒
	CacheDuration  int    `json:"cache_duration,omitempty"`  // 公钥缓存时间（小时），默认1小时
}

type VipPayConfig struct {
	HttpPort    string             `json:"http_port,omitempty"`
	Domain      string             `json:"domain,omitempty"`       // 支付回调域名
	AppleOAuth  *AppleOAuthConfig  `json:"apple_oauth,omitempty"`  // Apple OAuth2 配置
	GoogleOAuth *GoogleOAuthConfig `json:"google_oauth,omitempty"` // Google OAuth2 配置
}

type AdminConfig struct {
	HttpPort string `json:"http_port,omitempty"`
}

type AsynctaskConfig struct {
	HttpPort  string                     `json:"http_port,omitempty"`
	Providers map[string]*ProviderConfig `json:"providers,omitempty"`
}

// ProviderConfig 定义视频/图片生成Provider的配置
type ProviderConfig struct {
	Enabled      bool              `json:"enabled,omitempty"`
	APIKey       string            `json:"api_key,omitempty"`
	Secret       string            `json:"secret,omitempty"`
	BaseURL      string            `json:"base_url,omitempty"`
	ImageBaseURL string            `json:"image_base_url,omitempty"`
	Model        string            `json:"model,omitempty"`
	ImageModel   string            `json:"image_model,omitempty"`
	Workflow     string            `json:"workflow,omitempty"`
	Timeout      int               `json:"timeout,omitempty"` // 超时时间（秒）
	Additional   map[string]string `json:"additional,omitempty"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	WindowSeconds int            `json:"window_seconds,omitempty"` // 窗口时长（秒）
	DefaultLimit  int            `json:"default_limit,omitempty"`  // 每窗默认限制
	PerRoute      map[string]int `json:"per_route,omitempty"`      // 按接口覆盖：key 为 procedure，如 /common.TeamsAPI/Login
	PreferUser    bool           `json:"prefer_user,omitempty"`    // 优先按用户限流
	PerUser       map[string]int `json:"per_user,omitempty"`       // 指定用户覆盖：key 为用户ID字符串
}

// Config define common config struct
type Config struct {
	SqlDB     *DBConfig        `json:"sql_db,omitempty"`
	Redis     *RedisConfig     `json:"redis,omitempty"`
	Elastic   *ElasticConfig   `json:"elastic,omitempty"`
	LogLevel  string           `json:"log_level,omitempty"`
	RpcPort   string           `json:"rpc_port,omitempty"`
	HttpPort  string           `json:"http_port,omitempty"`
	LLMchat   *LLMchatConfig   `json:"llmchat,omitempty"`
	VipPay    *VipPayConfig    `json:"vippay,omitempty"`
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`
	Admin     *AdminConfig     `json:"admin,omitempty"`
	Asynctask *AsynctaskConfig `json:"asynctask,omitempty"`
}

func (c *Config) String() string {
	json, _ := json.Marshal(c)
	return string(json)
}

func ValiedConfig(cfg *Config) error {
	if cfg.RpcPort == "" {
		return fmt.Errorf("server rpc port not set")
	}
	if cfg.HttpPort == "" {
		return fmt.Errorf("server http port not set")
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
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Errorf("read config file error : %v", err)
		return err
	}
	err = json.Unmarshal(data, GlobalConfig)
	if err != nil {
		log.Errorf("config file format wrong :%v", err)
		return err
	}
	deployEnv := os.Getenv("DEPLOY_ENV")
	log.Infof("deployEnv: %s", deployEnv)
	log.Infof("before update GlobalConfig: %+v", GlobalConfig.String())
	if deployEnv == "pre" {
		log.Infof("update GlobalConfig: %+v", os.Getenv("REDIS_SERVER"))
		log.Infof("update GlobalConfig: %+v", os.Getenv("DB_NAME"))
		log.Infof("update GlobalConfig: %+v", os.Getenv("DB_USER"))
		log.Infof("update GlobalConfig: %+v", os.Getenv("DB_PASSWORD"))
		log.Infof("update GlobalConfig: %+v", os.Getenv("DB_ADDR"))
		GlobalConfig.Redis.Address = os.Getenv("REDIS_SERVER")
		GlobalConfig.SqlDB.Database = os.Getenv("DB_NAME")
		GlobalConfig.SqlDB.Username = os.Getenv("DB_USER")
		GlobalConfig.SqlDB.Password = os.Getenv("DB_PASSWORD")
		GlobalConfig.SqlDB.Address = os.Getenv("DB_ADDR")
	}
	log.Infof("after update GlobalConfig: %+v", GlobalConfig.String())
	return nil
}
