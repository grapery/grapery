package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds server configuration
type Config struct {
	Env          string          `yaml:"env"`
	HTTPPort     string          `yaml:"http_port"`
	ReadTimeout  time.Duration   `yaml:"read_timeout"`
	WriteTimeout time.Duration   `yaml:"write_timeout"`
	IdleTimeout  time.Duration   `yaml:"idle_timeout"`
	LogLevel     string          `yaml:"log_level"`
	AllowOrigins []string        `yaml:"allow_origins"`
	Database     DatabaseConfig  `yaml:"database"`
	Redis        RedisConfig     `yaml:"redis"`
	AI           AIConfig        `yaml:"ai"`
	JWT          JWTConfig       `yaml:"jwt"`
	Aliyun       AliyunConfig    `yaml:"aliyun"`
	Telemetry    TelemetryConfig `yaml:"telemetry"`
}

// DatabaseConfig holds MySQL database configuration
type DatabaseConfig struct {
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Address  string `yaml:"address"`
	MaxIdle  int    `yaml:"max_idle"`
	MaxOpen  int    `yaml:"max_open"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Address      string        `yaml:"address"`
	Password     string        `yaml:"password"`
	Database     int           `yaml:"database"`
	PingInterval time.Duration `yaml:"ping_interval"`
}

// AIConfig holds AI service configuration
type AIConfig struct {
	HuoshanAPIKey     string `yaml:"huoshan_api_key"`
	HuoshanBaseURL    string `yaml:"huoshan_base_url"`
	HuoshanImageModel string `yaml:"huoshan_image_model"` // Image model for Huoshan
	GeminiAPIKey      string `yaml:"gemini_api_key"`
	GeminiBaseURL     string `yaml:"gemini_base_url"`
	DefaultProvider   string `yaml:"default_provider"` // Default provider for text generation
	ImageProvider     string `yaml:"image_provider"`   // Provider for image generation (gemini, huoshan)
	VideoProvider     string `yaml:"video_provider"`   // Provider for video generation (gemini, huoshan, hailuo)
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret string        `yaml:"secret"`
	Expiry time.Duration `yaml:"expiry"` // Token 过期时间
}

// AliyunConfig holds Aliyun OSS configuration
type AliyunConfig struct {
	APIKey    string `yaml:"api_key"`
	SecretKey string `yaml:"secret_key"`
	Endpoint  string `yaml:"endpoint"`
	Bucket    string `yaml:"bucket"`
	RoleARN   string `yaml:"role_arn"` // for STS token
}

// TelemetryConfig holds telemetry configuration
type TelemetryConfig struct {
	SLS        SLSConfig        `yaml:"sls"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Tracing    TracingConfig    `yaml:"tracing"`
}

// SLSConfig holds Alibaba Cloud Log Service configuration
type SLSConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Endpoint        string `yaml:"endpoint"` // e.g., cn-shanghai.log.aliyuncs.com
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	Project         string `yaml:"project"`
	Logstore        string `yaml:"logstore"`
	Topic           string `yaml:"topic"`
	Source          string `yaml:"source"`
}

// PrometheusConfig holds Prometheus metrics configuration
type PrometheusConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Path         string `yaml:"path"`          // metrics endpoint path, default: /metrics
	PushGateway  string `yaml:"push_gateway"`  // Alibaba Cloud Prometheus push gateway URL
	PushInterval int    `yaml:"push_interval"` // push interval in seconds
	JobName      string `yaml:"job_name"`      // job name for push gateway
}

// TracingConfig holds distributed tracing configuration
type TracingConfig struct {
	Enabled        bool              `yaml:"enabled"`
	ServiceName    string            `yaml:"service_name"`
	ServiceVersion string            `yaml:"service_version"`
	Environment    string            `yaml:"environment"`
	JaegerEndpoint string            `yaml:"jaeger_endpoint"`
	OTLPEndpoint   string            `yaml:"otlp_endpoint"`
	SamplingRatio  float64           `yaml:"sampling_ratio"`
	Headers        map[string]string `yaml:"headers"`
}

// LoadFromFile loads configuration from a YAML file
// Environment variables will override file values
func LoadFromFile(configPath string) (Config, error) {
	// Start with default config
	cfg := getDefaultConfig()

	// Load from file if provided
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return cfg, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	cfg = overrideWithEnv(cfg)

	return cfg, nil
}

// Load builds a Config from environment variables (backward compatible)
func Load() Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DATABASE", "0"))
	pingInterval, _ := strconv.Atoi(getEnv("REDIS_PING_INTERVAL", "30"))
	jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))

	cfg := Config{
		Env:          getEnv("GRAPERY_ENV", "development"),
		HTTPPort:     getEnv("GRAPERY_HTTP_PORT", "8080"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		LogLevel:     getEnv("GRAPERY_LOG_LEVEL", "info"),
		AllowOrigins: []string{
			getEnv("GRAPERY_ALLOW_ORIGIN", "http://localhost:5173"),
		},
		Database: DatabaseConfig{
			Database: getEnv("DB_DATABASE", "grapery"),
			Username: getEnv("DB_USERNAME", "root"),
			Password: getEnv("DB_PASSWORD", "12345678"),
			Address:  getEnv("DB_ADDRESS", "localhost"),
			MaxIdle:  10,
			MaxOpen:  100,
		},
		Redis: RedisConfig{
			Address:      getEnv("REDIS_ADDRESS", "localhost:6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			Database:     redisDB,
			PingInterval: time.Duration(pingInterval) * time.Second,
		},
		AI: AIConfig{
			HuoshanAPIKey:     getEnv("HUOSHAN_API_KEY", ""),
			HuoshanBaseURL:    getEnv("HUOSHAN_BASE_URL", ""),
			HuoshanImageModel: getEnv("HUOSHAN_IMAGE_MODEL", ""),
			GeminiAPIKey:      getEnv("GEMINI_API_KEY", ""),
			GeminiBaseURL:     getEnv("GEMINI_BASE_URL", ""),
			DefaultProvider:   getEnv("AI_DEFAULT_PROVIDER", "huoshan"),
			ImageProvider:     getEnv("AI_IMAGE_PROVIDER", "huoshan"), // Default to huoshan for image generation
			VideoProvider:     getEnv("AI_VIDEO_PROVIDER", "huoshan"), // Default to huoshan for video generation
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "grapery-secret-key-change-in-production"),
			Expiry: time.Duration(jwtExpiry) * time.Hour,
		},
		Aliyun: AliyunConfig{
			APIKey:    getEnv("ALIYUN_API_KEY", ""),
			SecretKey: getEnv("ALIYUN_SECRET_KEY", ""),
			Endpoint:  getEnv("ALIYUN_ENDPOINT", "oss-cn-shanghai.aliyuncs.com"),
			Bucket:    getEnv("ALIYUN_BUCKET", "grapery-dev"),
			RoleARN:   getEnv("ALIYUN_ROLE_ARN", ""),
		},
		Telemetry: TelemetryConfig{
			SLS: SLSConfig{
				Enabled:         getEnv("TELEMETRY_SLS_ENABLED", "false") == "true",
				Endpoint:        getEnv("TELEMETRY_SLS_ENDPOINT", ""),
				AccessKeyID:     getEnv("TELEMETRY_SLS_ACCESS_KEY_ID", ""),
				AccessKeySecret: getEnv("TELEMETRY_SLS_ACCESS_KEY_SECRET", ""),
				Project:         getEnv("TELEMETRY_SLS_PROJECT", ""),
				Logstore:        getEnv("TELEMETRY_SLS_LOGSTORE", ""),
				Topic:           getEnv("TELEMETRY_SLS_TOPIC", "grapery"),
				Source:          getEnv("TELEMETRY_SLS_SOURCE", ""),
			},
			Prometheus: PrometheusConfig{
				Enabled:      getEnv("TELEMETRY_PROMETHEUS_ENABLED", "false") == "true",
				Path:         getEnv("TELEMETRY_PROMETHEUS_PATH", "/metrics"),
				PushGateway:  getEnv("TELEMETRY_PROMETHEUS_PUSH_GATEWAY", ""),
				PushInterval: getEnvInt("TELEMETRY_PROMETHEUS_PUSH_INTERVAL", 15),
				JobName:      getEnv("TELEMETRY_PROMETHEUS_JOB_NAME", "grapery"),
			},
			Tracing: TracingConfig{
				Enabled:        getEnv("TELEMETRY_TRACING_ENABLED", "false") == "true",
				ServiceName:    getEnv("TELEMETRY_TRACING_SERVICE_NAME", "grapery-api"),
				ServiceVersion: getEnv("TELEMETRY_TRACING_SERVICE_VERSION", "1.0.0"),
				Environment:    getEnv("TELEMETRY_TRACING_ENVIRONMENT", "development"),
				JaegerEndpoint: getEnv("TELEMETRY_TRACING_JAEGER_ENDPOINT", ""),
				OTLPEndpoint:   getEnv("TELEMETRY_TRACING_OTLP_ENDPOINT", ""),
				SamplingRatio:  getEnvFloat("TELEMETRY_TRACING_SAMPLING_RATIO", 1.0),
			},
		},
	}

	return cfg
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

// DSN returns MySQL connection string
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.Username, d.Password, d.Address, d.Database)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v == "true" || v == "1" || v == "yes"
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// getDefaultConfig returns default configuration
func getDefaultConfig() Config {
	return Config{
		Env:          "development",
		HTTPPort:     "8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		LogLevel:     "info",
		AllowOrigins: []string{"http://localhost:8080"},
		Database: DatabaseConfig{
			Database: "grapery",
			Username: "root",
			Password: "12345678",
			Address:  "localhost",
			MaxIdle:  10,
			MaxOpen:  100,
		},
		Redis: RedisConfig{
			Address:      "localhost:6379",
			Password:     "",
			Database:     0,
			PingInterval: 30 * time.Second,
		},
		AI: AIConfig{
			HuoshanAPIKey:   "",
			HuoshanBaseURL:  "",
			GeminiAPIKey:    "",
			GeminiBaseURL:   "",
			DefaultProvider: "huoshan",
		},
		JWT: JWTConfig{
			Secret: "grapery-secret-key-change-in-production",
			Expiry: 24 * time.Hour,
		},
		Aliyun: AliyunConfig{
			APIKey:    "",
			SecretKey: "",
			Endpoint:  "oss-cn-shanghai.aliyuncs.com",
			Bucket:    "grapery-dev",
			RoleARN:   "",
		},
		Telemetry: TelemetryConfig{
			SLS: SLSConfig{
				Enabled:         false,
				Endpoint:        "",
				AccessKeyID:     "",
				AccessKeySecret: "",
				Project:         "",
				Logstore:        "",
				Topic:           "grapery",
				Source:          "",
			},
			Prometheus: PrometheusConfig{
				Enabled:      false,
				Path:         "/metrics",
				PushGateway:  "",
				PushInterval: 15,
				JobName:      "grapery",
			},
			Tracing: TracingConfig{
				Enabled:        false,
				ServiceName:    "grapery-api",
				ServiceVersion: "1.0.0",
				Environment:    "development",
				SamplingRatio:  1.0,
				Headers:        make(map[string]string),
				JaegerEndpoint: "",
				OTLPEndpoint:   "",
			},
		},
	}
}

// overrideWithEnv overrides config values with environment variables
func overrideWithEnv(cfg Config) Config {
	// Server config
	cfg.Env = getEnv("GRAPERY_ENV", cfg.Env)
	cfg.HTTPPort = getEnv("GRAPERY_HTTP_PORT", cfg.HTTPPort)
	cfg.LogLevel = getEnv("GRAPERY_LOG_LEVEL", cfg.LogLevel)

	if origin := getEnv("GRAPERY_ALLOW_ORIGIN", ""); origin != "" {
		cfg.AllowOrigins = []string{origin}
	}

	// Database config
	cfg.Database.Database = getEnv("DB_DATABASE", cfg.Database.Database)
	cfg.Database.Username = getEnv("DB_USERNAME", cfg.Database.Username)
	cfg.Database.Password = getEnv("DB_PASSWORD", cfg.Database.Password)
	cfg.Database.Address = getEnv("DB_ADDRESS", cfg.Database.Address)

	// Redis config
	cfg.Redis.Address = getEnv("REDIS_ADDRESS", cfg.Redis.Address)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", cfg.Redis.Password)
	if redisDB, _ := strconv.Atoi(getEnv("REDIS_DATABASE", "")); redisDB > 0 {
		cfg.Redis.Database = redisDB
	}
	if pingInterval, _ := strconv.Atoi(getEnv("REDIS_PING_INTERVAL", "")); pingInterval > 0 {
		cfg.Redis.PingInterval = time.Duration(pingInterval) * time.Second
	}

	// AI config
	cfg.AI.HuoshanAPIKey = getEnv("HUOSHAN_API_KEY", cfg.AI.HuoshanAPIKey)
	cfg.AI.HuoshanBaseURL = getEnv("HUOSHAN_BASE_URL", cfg.AI.HuoshanBaseURL)
	cfg.AI.GeminiAPIKey = getEnv("GEMINI_API_KEY", cfg.AI.GeminiAPIKey)
	cfg.AI.GeminiBaseURL = getEnv("GEMINI_BASE_URL", cfg.AI.GeminiBaseURL)
	cfg.AI.DefaultProvider = getEnv("AI_DEFAULT_PROVIDER", cfg.AI.DefaultProvider)

	// JWT config
	cfg.JWT.Secret = getEnv("JWT_SECRET", cfg.JWT.Secret)
	if jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "")); jwtExpiry > 0 {
		cfg.JWT.Expiry = time.Duration(jwtExpiry) * time.Hour
	}

	// Aliyun OSS config
	cfg.Aliyun.APIKey = getEnv("ALIYUN_API_KEY", cfg.Aliyun.APIKey)
	cfg.Aliyun.SecretKey = getEnv("ALIYUN_SECRET_KEY", cfg.Aliyun.SecretKey)
	cfg.Aliyun.Endpoint = getEnv("ALIYUN_ENDPOINT", cfg.Aliyun.Endpoint)
	cfg.Aliyun.Bucket = getEnv("ALIYUN_BUCKET", cfg.Aliyun.Bucket)
	cfg.Aliyun.RoleARN = getEnv("ALIYUN_ROLE_ARN", cfg.Aliyun.RoleARN)

	// Telemetry SLS config
	cfg.Telemetry.SLS.Enabled = getEnvBool("TELEMETRY_SLS_ENABLED", cfg.Telemetry.SLS.Enabled)
	cfg.Telemetry.SLS.Endpoint = getEnv("TELEMETRY_SLS_ENDPOINT", cfg.Telemetry.SLS.Endpoint)
	cfg.Telemetry.SLS.AccessKeyID = getEnv("TELEMETRY_SLS_ACCESS_KEY_ID", cfg.Telemetry.SLS.AccessKeyID)
	cfg.Telemetry.SLS.AccessKeySecret = getEnv("TELEMETRY_SLS_ACCESS_KEY_SECRET", cfg.Telemetry.SLS.AccessKeySecret)
	cfg.Telemetry.SLS.Project = getEnv("TELEMETRY_SLS_PROJECT", cfg.Telemetry.SLS.Project)
	cfg.Telemetry.SLS.Logstore = getEnv("TELEMETRY_SLS_LOGSTORE", cfg.Telemetry.SLS.Logstore)
	cfg.Telemetry.SLS.Topic = getEnv("TELEMETRY_SLS_TOPIC", cfg.Telemetry.SLS.Topic)
	cfg.Telemetry.SLS.Source = getEnv("TELEMETRY_SLS_SOURCE", cfg.Telemetry.SLS.Source)

	// Telemetry Prometheus config
	cfg.Telemetry.Prometheus.Enabled = getEnvBool("TELEMETRY_PROMETHEUS_ENABLED", cfg.Telemetry.Prometheus.Enabled)
	cfg.Telemetry.Prometheus.Path = getEnv("TELEMETRY_PROMETHEUS_PATH", cfg.Telemetry.Prometheus.Path)
	cfg.Telemetry.Prometheus.PushGateway = getEnv("TELEMETRY_PROMETHEUS_PUSH_GATEWAY", cfg.Telemetry.Prometheus.PushGateway)
	cfg.Telemetry.Prometheus.PushInterval = getEnvInt("TELEMETRY_PROMETHEUS_PUSH_INTERVAL", cfg.Telemetry.Prometheus.PushInterval)
	cfg.Telemetry.Prometheus.JobName = getEnv("TELEMETRY_PROMETHEUS_JOB_NAME", cfg.Telemetry.Prometheus.JobName)

	// Telemetry Tracing config
	cfg.Telemetry.Tracing.Enabled = getEnvBool("TELEMETRY_TRACING_ENABLED", cfg.Telemetry.Tracing.Enabled)
	cfg.Telemetry.Tracing.ServiceName = getEnv("TELEMETRY_TRACING_SERVICE_NAME", cfg.Telemetry.Tracing.ServiceName)
	cfg.Telemetry.Tracing.ServiceVersion = getEnv("TELEMETRY_TRACING_SERVICE_VERSION", cfg.Telemetry.Tracing.ServiceVersion)
	cfg.Telemetry.Tracing.Environment = getEnv("TELEMETRY_TRACING_ENVIRONMENT", cfg.Telemetry.Tracing.Environment)
	cfg.Telemetry.Tracing.JaegerEndpoint = getEnv("TELEMETRY_TRACING_JAEGER_ENDPOINT", cfg.Telemetry.Tracing.JaegerEndpoint)
	cfg.Telemetry.Tracing.OTLPEndpoint = getEnv("TELEMETRY_TRACING_OTLP_ENDPOINT", cfg.Telemetry.Tracing.OTLPEndpoint)
	cfg.Telemetry.Tracing.SamplingRatio = getEnvFloat("TELEMETRY_TRACING_SAMPLING_RATIO", cfg.Telemetry.Tracing.SamplingRatio)

	return cfg
}
