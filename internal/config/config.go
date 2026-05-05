package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds server configuration
type Config struct {
	Env            string               `yaml:"env"`
	HTTPPort       string               `yaml:"http_port"`
	ReadTimeout    time.Duration        `yaml:"read_timeout"`
	WriteTimeout   time.Duration        `yaml:"write_timeout"`
	IdleTimeout    time.Duration        `yaml:"idle_timeout"`
	LogLevel       string               `yaml:"log_level"`
	AllowOrigins   []string             `yaml:"allow_origins"`
	Database       DatabaseConfig       `yaml:"database"`
	Redis          RedisConfig          `yaml:"redis"`
	Recommendation RecommendationConfig `yaml:"recommendation"`
	AI             AIConfig             `yaml:"ai"`
	JWT            JWTConfig            `yaml:"jwt"`
	Aliyun         AliyunConfig         `yaml:"aliyun"`
	APNs           APNsConfig           `yaml:"apns"`
	Telemetry      TelemetryConfig      `yaml:"telemetry"`
}

// APNsConfig Apple Push Notification service (HTTP/2 API, .p8 auth key).
type APNsConfig struct {
	BundleID       string `yaml:"bundle_id"`        // apns-topic; must match iOS app Bundle ID
	TeamID         string `yaml:"team_id"`          // Apple Developer Team ID
	KeyID          string `yaml:"key_id"`           // APNs Auth Key ID
	PrivateKey     string `yaml:"private_key"`      // PEM contents (optional; prefer file in deployment)
	PrivateKeyPath string `yaml:"private_key_path"` // Path to AuthKey_xxx.p8
	UseSandbox     bool   `yaml:"use_sandbox"`      // true → api.sandbox.push.apple.com
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

// RecommendationConfig holds recommendation and feed cache configuration
type RecommendationConfig struct {
	FragmentGenreRatio      int `yaml:"fragment_genre_ratio"`
	FragmentFallbackRatio   int `yaml:"fragment_fallback_ratio"`
	StoryboardGenreRatio    int `yaml:"storyboard_genre_ratio"`
	StoryboardFallbackRatio int `yaml:"storyboard_fallback_ratio"`
	CandidateMultiplier     int `yaml:"candidate_multiplier"`
	CacheTTLSeconds         int `yaml:"cache_ttl_seconds"`
	// SeenMaxEntries caps per-user Redis ZSET for for_you "already seen" storyboard/fragment IDs (0 = default 5000).
	SeenMaxEntries int `yaml:"seen_max_entries"`
	// SeenTTLDays sets Redis key TTL for those ZSETs; 0 = no expiration on the key.
	SeenTTLDays int `yaml:"seen_ttl_days"`
}

// AIConfig holds AI service configuration
type AIConfig struct {
	HuoshanAPIKey     string `yaml:"huoshan_api_key"`
	HuoshanBaseURL    string `yaml:"huoshan_base_url"`
	HuoshanImageModel string `yaml:"huoshan_image_model"` // Image model for Huoshan
	// HuoshanTextModel 火山方舟对话/多模态接入点 ID（如 ep-xxxx），用于国内用户文本与分镜规划。
	HuoshanTextModel string `yaml:"huoshan_text_model"`
	GeminiAPIKey     string `yaml:"gemini_api_key"`
	GeminiBaseURL    string `yaml:"gemini_base_url"`
	KlingAccessKey   string `yaml:"kling_access_key"`
	KlingSecretKey   string `yaml:"kling_secret_key"`
	KlingBaseURL     string `yaml:"kling_base_url"`
	DefaultProvider  string `yaml:"default_provider"` // Default provider for text generation
	ImageProvider    string `yaml:"image_provider"`   // Provider for image generation (gemini, huoshan, kling)
	VideoProvider    string `yaml:"video_provider"`   // Provider for video generation (gemini, huoshan, hailuo, kling)
	// RequestTimeoutSeconds is the HTTP client timeout (seconds) for outbound AI provider calls registered in initAIClients
	// (Gemini, Huoshan, Kling). 0 or negative means default 180 (multimodal / 分镜规划常需更久).
	RequestTimeoutSeconds int `yaml:"request_timeout_seconds"`
	// TextMaxConcurrent caps simultaneous outbound text-LLM calls cluster-wide (in-flight until response completes).
	// Implemented via Redis; 0 disables the gate. Typical value matches provider throughput (e.g. 5).
	TextMaxConcurrent int `yaml:"text_max_concurrent"`
}

// NormalizeAITextDefaultProvider coerces default_provider / AI_DEFAULT_PROVIDER to a text-LLM-capable name (huoshan or gemini).
// Values like kling or hailuo are not valid for chat/planning; they map to huoshan. warn is true when a non-empty invalid value was replaced.
func NormalizeAITextDefaultProvider(p string) (normalized string, warn bool) {
	s := strings.ToLower(strings.TrimSpace(p))
	switch s {
	case "gemini", "huoshan":
		return s, false
	case "":
		return "huoshan", false
	default:
		return "huoshan", true
	}
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
	// OSS STS credentials (RAM user for AssumeRole)
	OSSAccessKeyID     string `yaml:"oss_access_key_id"`
	OSSAccessKeySecret string `yaml:"oss_access_key_secret"`
	OSSRoleARN         string `yaml:"oss_role_arn"`
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
// app parameter identifies which backend service is running (e.g., "api-server", "vippay")
func LoadFromFile(configPath string, app string) (Config, error) {
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

		if cfg.APNs.PrivateKeyPath != "" && !filepath.IsAbs(cfg.APNs.PrivateKeyPath) {
			if absConfig, err := filepath.Abs(configPath); err == nil {
				cfg.APNs.PrivateKeyPath = filepath.Join(filepath.Dir(absConfig), cfg.APNs.PrivateKeyPath)
			}
		}
	}

	// Override with environment variables
	cfg = overrideWithEnv(cfg, app)

	return cfg, nil
}

// Load builds a Config from environment variables (backward compatible)
// app parameter identifies which backend service is running (e.g., "api-server", "vippay")
func Load(app string) Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DATABASE", "0"))
	pingInterval, _ := strconv.Atoi(getEnv("REDIS_PING_INTERVAL", "30"))
	jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))

	cfg := Config{
		Env:          getEnv("GRAPERY_ENV", "development"),
		HTTPPort:     getEnv("GRAPERY_HTTP_PORT", "8080"),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second, // 增加超时以支持 AI 生成等长时间操作
		IdleTimeout:  120 * time.Second,
		LogLevel:     getEnv("GRAPERY_LOG_LEVEL", "info"),
		AllowOrigins: []string{
			getEnv("GRAPERY_ALLOW_ORIGIN", "http://localhost:5173"),
		},
		Database: DatabaseConfig{
			Database: getEnv("DB_DATABASE", "grapery"),
			Username: getEnv("DB_USERNAME", "root"),
			Password: getEnv("DB_PASSWORD", ""), // SECURITY: No default - must be set via env
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
		Recommendation: RecommendationConfig{
			FragmentGenreRatio:      getEnvInt("RECO_FRAGMENT_GENRE_RATIO", 3),
			FragmentFallbackRatio:   getEnvInt("RECO_FRAGMENT_FALLBACK_RATIO", 2),
			StoryboardGenreRatio:    getEnvInt("RECO_STORYBOARD_GENRE_RATIO", 3),
			StoryboardFallbackRatio: getEnvInt("RECO_STORYBOARD_FALLBACK_RATIO", 2),
			CandidateMultiplier:     getEnvInt("RECO_CANDIDATE_MULTIPLIER", 4),
			CacheTTLSeconds:         getEnvInt("RECO_CACHE_TTL_SECONDS", 180),
			SeenMaxEntries:          getEnvInt("RECO_SEEN_MAX_ENTRIES", 5000),
			SeenTTLDays:             getEnvInt("RECO_SEEN_TTL_DAYS", 30),
		},
		AI: AIConfig{
			HuoshanAPIKey:         getEnv("HUOSHAN_API_KEY", ""),
			HuoshanBaseURL:        getEnv("HUOSHAN_BASE_URL", ""),
			HuoshanImageModel:     getEnv("HUOSHAN_IMAGE_MODEL", ""),
			HuoshanTextModel:      getEnv("HUOSHAN_TEXT_MODEL", ""),
			GeminiAPIKey:          getEnv("GEMINI_API_KEY", ""),
			GeminiBaseURL:         getEnv("GEMINI_BASE_URL", ""),
			KlingAccessKey:        getEnv("KLING_ACCESS_KEY", ""),
			KlingSecretKey:        getEnv("KLING_SECRET_KEY", ""),
			KlingBaseURL:          getEnv("KLING_BASE_URL", ""),
			DefaultProvider:       getEnv("AI_DEFAULT_PROVIDER", "huoshan"),
			ImageProvider:         getEnv("AI_IMAGE_PROVIDER", "huoshan"), // Default to huoshan for image generation
			VideoProvider:         getEnv("AI_VIDEO_PROVIDER", "huoshan"), // Default to huoshan for video generation
			RequestTimeoutSeconds: normalizeAIRequestTimeoutSeconds(getEnvInt("AI_REQUEST_TIMEOUT_SECONDS", 180)),
			TextMaxConcurrent:     normalizeAITextMaxConcurrent(getEnvInt("AI_TEXT_MAX_CONCURRENT", 5)),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""), // SECURITY: No default - must be set via env
			Expiry: time.Duration(jwtExpiry) * time.Hour,
		},
		Aliyun: AliyunConfig{
			APIKey:    getEnv("ALIYUN_API_KEY", ""),
			SecretKey: getEnv("ALIYUN_SECRET_KEY", ""),
			Endpoint:  getEnv("ALIYUN_ENDPOINT", "oss-cn-shanghai.aliyuncs.com"),
			Bucket:    getEnv("ALIYUN_BUCKET", "grapery-dev"),
			RoleARN:   getEnv("ALIYUN_ROLE_ARN", ""),
			// OSS STS credentials (RAM user for AssumeRole)
			OSSAccessKeyID:     getEnv("ALIYUN_OSS_ACCESS_KEY_ID", ""),
			OSSAccessKeySecret: getEnv("ALIYUN_OSS_ACCESS_KEY_SECRET", ""),
			OSSRoleARN:         getEnv("ALIYUN_OSS_ROLE_ARN", ""),
		},
		APNs: mergeAPNsEmptyFields(APNsConfig{
			BundleID:       getEnv("APNS_BUNDLE_ID", ""),
			TeamID:         getEnv("APNS_TEAM_ID", ""),
			KeyID:          getEnv("APNS_KEY_ID", ""),
			PrivateKey:     getEnv("APNS_PRIVATE_KEY", ""),
			PrivateKeyPath: getEnv("APNS_PRIVATE_KEY_PATH", ""),
			UseSandbox:     true,
		}),
		Telemetry: TelemetryConfig{
			SLS: SLSConfig{
				Enabled:         true,
				Endpoint:        "cn-hangzhou.log.aliyuncs.com",
				AccessKeyID:     os.Getenv("ALIYUN_ACCESS_KEY_ID"),
				AccessKeySecret: os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
				Project:         "grapery-dev",
				Logstore:        "apiservice",
				Topic:           "api-backend",
				Source:          app,
			},
			Prometheus: PrometheusConfig{
				Enabled:      true,
				Path:         "/metrics",
				PushGateway:  "https://workspace-default-cms-1866841989078847-cn-hangzhou.cn-hangzhou.log.aliyuncs.com/prometheus/workspace-default-cms-1866841989078847-cn-hangzhou/aliyun-prom-s8l4110ylj/api/v1/pushgateway",
				PushInterval: 60,
				JobName:      "grapery",
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

	if _, ok := os.LookupEnv("APNS_USE_SANDBOX"); ok {
		cfg.APNs.UseSandbox = getEnvBool("APNS_USE_SANDBOX", cfg.APNs.UseSandbox)
	}

	return cfg
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

// DSN returns MySQL connection string
func (d DatabaseConfig) DSN() string {
	// collation=utf8mb4_unicode_ci: ensures each pooled connection uses utf8mb4 end-to-end (avoids 1366 with Chinese prompts).
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local",
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

// normalizeAITextMaxConcurrent returns n if non-negative; negative values are treated as 0 (admission gate off).
func normalizeAITextMaxConcurrent(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// normalizeAIRequestTimeoutSeconds returns sec if positive, otherwise default 180 (seconds per AI HTTP call).
func normalizeAIRequestTimeoutSeconds(sec int) int {
	if sec <= 0 {
		return 180
	}
	return sec
}

// getSLSSourceWithDefault returns the SLS source, using provided default or app parameter
func getSLSSourceWithDefault(defaultSource string, app string) string {
	if source := getEnv("TELEMETRY_SLS_SOURCE", ""); source != "" {
		return source
	}
	if defaultSource != "" {
		return defaultSource
	}
	if app != "" {
		return app
	}
	return "unknown"
}

// getDefaultConfig returns default configuration
func getDefaultConfig() Config {
	return Config{
		Env:          "development",
		HTTPPort:     "8080",
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second, // 增加超时以支持 AI 生成等长时间操作
		IdleTimeout:  120 * time.Second,
		LogLevel:     "info",
		AllowOrigins: []string{"http://localhost:8080"},
		Database: DatabaseConfig{
			Database: "grapery",
			Username: "root",
			Password: "", // SECURITY: No default password - must be set via DB_PASSWORD env var
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
		Recommendation: RecommendationConfig{
			FragmentGenreRatio:      3,
			FragmentFallbackRatio:   2,
			StoryboardGenreRatio:    3,
			StoryboardFallbackRatio: 2,
			CandidateMultiplier:     4,
			CacheTTLSeconds:         180,
			SeenMaxEntries:          5000,
			SeenTTLDays:             30,
		},
		AI: AIConfig{
			HuoshanAPIKey:         "",
			HuoshanBaseURL:        "",
			GeminiAPIKey:          "",
			GeminiBaseURL:         "",
			DefaultProvider:       "huoshan",
			RequestTimeoutSeconds: 120,
			TextMaxConcurrent:     5,
		},
		JWT: JWTConfig{
			Secret: "", // SECURITY: No default secret - must be set via JWT_SECRET env var
			Expiry: 24 * time.Hour,
		},
		Aliyun: AliyunConfig{
			APIKey:    "",
			SecretKey: "",
			Endpoint:  "oss-cn-shanghai.aliyuncs.com",
			Bucket:    "grapery-dev",
			RoleARN:   "",
		},
		APNs: mergeAPNsEmptyFields(APNsConfig{}),
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
func overrideWithEnv(cfg Config, app string) Config {
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

	// Recommendation config
	cfg.Recommendation.FragmentGenreRatio = getEnvInt("RECO_FRAGMENT_GENRE_RATIO", cfg.Recommendation.FragmentGenreRatio)
	cfg.Recommendation.FragmentFallbackRatio = getEnvInt("RECO_FRAGMENT_FALLBACK_RATIO", cfg.Recommendation.FragmentFallbackRatio)
	cfg.Recommendation.StoryboardGenreRatio = getEnvInt("RECO_STORYBOARD_GENRE_RATIO", cfg.Recommendation.StoryboardGenreRatio)
	cfg.Recommendation.StoryboardFallbackRatio = getEnvInt("RECO_STORYBOARD_FALLBACK_RATIO", cfg.Recommendation.StoryboardFallbackRatio)
	cfg.Recommendation.CandidateMultiplier = getEnvInt("RECO_CANDIDATE_MULTIPLIER", cfg.Recommendation.CandidateMultiplier)
	cfg.Recommendation.CacheTTLSeconds = getEnvInt("RECO_CACHE_TTL_SECONDS", cfg.Recommendation.CacheTTLSeconds)
	cfg.Recommendation.SeenMaxEntries = getEnvInt("RECO_SEEN_MAX_ENTRIES", cfg.Recommendation.SeenMaxEntries)
	cfg.Recommendation.SeenTTLDays = getEnvInt("RECO_SEEN_TTL_DAYS", cfg.Recommendation.SeenTTLDays)

	// AI config
	cfg.AI.HuoshanAPIKey = getEnv("HUOSHAN_API_KEY", cfg.AI.HuoshanAPIKey)
	cfg.AI.HuoshanBaseURL = getEnv("HUOSHAN_BASE_URL", cfg.AI.HuoshanBaseURL)
	cfg.AI.HuoshanImageModel = getEnv("HUOSHAN_IMAGE_MODEL", cfg.AI.HuoshanImageModel)
	cfg.AI.HuoshanTextModel = getEnv("HUOSHAN_TEXT_MODEL", cfg.AI.HuoshanTextModel)
	cfg.AI.GeminiAPIKey = getEnv("GEMINI_API_KEY", cfg.AI.GeminiAPIKey)
	cfg.AI.GeminiBaseURL = getEnv("GEMINI_BASE_URL", cfg.AI.GeminiBaseURL)
	cfg.AI.KlingAccessKey = getEnv("KLING_ACCESS_KEY", cfg.AI.KlingAccessKey)
	cfg.AI.KlingSecretKey = getEnv("KLING_SECRET_KEY", cfg.AI.KlingSecretKey)
	cfg.AI.KlingBaseURL = getEnv("KLING_BASE_URL", cfg.AI.KlingBaseURL)
	cfg.AI.DefaultProvider = getEnv("AI_DEFAULT_PROVIDER", cfg.AI.DefaultProvider)
	origDefaultAI := cfg.AI.DefaultProvider
	if norm, warn := NormalizeAITextDefaultProvider(cfg.AI.DefaultProvider); warn {
		log.Printf("config: AI_DEFAULT_PROVIDER=%q is not valid for text LLM (use huoshan or gemini); coerced to %q",
			strings.TrimSpace(origDefaultAI), norm)
		cfg.AI.DefaultProvider = norm
	} else {
		cfg.AI.DefaultProvider = norm
	}
	cfg.AI.ImageProvider = getEnv("AI_IMAGE_PROVIDER", cfg.AI.ImageProvider)
	cfg.AI.VideoProvider = getEnv("AI_VIDEO_PROVIDER", cfg.AI.VideoProvider)
	cfg.AI.RequestTimeoutSeconds = normalizeAIRequestTimeoutSeconds(getEnvInt("AI_REQUEST_TIMEOUT_SECONDS", cfg.AI.RequestTimeoutSeconds))
	cfg.AI.TextMaxConcurrent = normalizeAITextMaxConcurrent(getEnvInt("AI_TEXT_MAX_CONCURRENT", cfg.AI.TextMaxConcurrent))

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
	// OSS STS credentials (RAM user for AssumeRole)
	cfg.Aliyun.OSSAccessKeyID = getEnv("ALIYUN_OSS_ACCESS_KEY_ID", cfg.Aliyun.OSSAccessKeyID)
	cfg.Aliyun.OSSAccessKeySecret = getEnv("ALIYUN_OSS_ACCESS_KEY_SECRET", cfg.Aliyun.OSSAccessKeySecret)
	cfg.Aliyun.OSSRoleARN = getEnv("ALIYUN_OSS_ROLE_ARN", cfg.Aliyun.OSSRoleARN)

	// APNs push
	cfg.APNs.BundleID = getEnv("APNS_BUNDLE_ID", cfg.APNs.BundleID)
	cfg.APNs.TeamID = getEnv("APNS_TEAM_ID", cfg.APNs.TeamID)
	cfg.APNs.KeyID = getEnv("APNS_KEY_ID", cfg.APNs.KeyID)
	if v := getEnv("APNS_PRIVATE_KEY", ""); v != "" {
		cfg.APNs.PrivateKey = v
	}
	cfg.APNs.PrivateKeyPath = getEnv("APNS_PRIVATE_KEY_PATH", cfg.APNs.PrivateKeyPath)
	if _, ok := os.LookupEnv("APNS_USE_SANDBOX"); ok {
		cfg.APNs.UseSandbox = getEnvBool("APNS_USE_SANDBOX", cfg.APNs.UseSandbox)
	}

	cfg.APNs = mergeAPNsEmptyFields(cfg.APNs)

	// Telemetry SLS config
	cfg.Telemetry.SLS.Enabled = getEnvBool("TELEMETRY_SLS_ENABLED", cfg.Telemetry.SLS.Enabled)
	cfg.Telemetry.SLS.Endpoint = getEnv("TELEMETRY_SLS_ENDPOINT", cfg.Telemetry.SLS.Endpoint)
	cfg.Telemetry.SLS.AccessKeyID = getEnv("TELEMETRY_SLS_ACCESS_KEY_ID", cfg.Telemetry.SLS.AccessKeyID)
	cfg.Telemetry.SLS.AccessKeySecret = getEnv("TELEMETRY_SLS_ACCESS_KEY_SECRET", cfg.Telemetry.SLS.AccessKeySecret)
	cfg.Telemetry.SLS.Project = getEnv("TELEMETRY_SLS_PROJECT", cfg.Telemetry.SLS.Project)
	cfg.Telemetry.SLS.Logstore = getEnv("TELEMETRY_SLS_LOGSTORE", cfg.Telemetry.SLS.Logstore)
	cfg.Telemetry.SLS.Topic = getEnv("TELEMETRY_SLS_TOPIC", cfg.Telemetry.SLS.Topic)
	cfg.Telemetry.SLS.Source = getSLSSourceWithDefault(cfg.Telemetry.SLS.Source, app)

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
