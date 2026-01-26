package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"github.com/grapestree/fgrapery/grapery/internal/server"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	transport "github.com/grapestree/fgrapery/grapery/internal/transport/http"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (YAML)")
	version := flag.Bool("version", false, "Print version information")
	flag.Parse()

	// Print version and exit if requested
	if *version {
		fmt.Println("Grapery Server v1.0.0")
		os.Exit(0)
	}
	var appname = "api-server"
	if os.Getenv("APP_NAME") != "" {
		appname = os.Getenv("APP_NAME")
	}

	// Get host information
	hostname := utils.GetHostname()
	hostIP := utils.GetHostIP()

	// Load configuration
	var cfg config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.LoadFromFile(*configPath, appname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("loading config from environment variables")
		// Fallback to environment variables only
		cfg = config.Load(appname)
	}

	// Initialize telemetry manager
	telemetryConfig := telemetry.TelemetryManagerConfig{
		LogLevel: cfg.LogLevel,
	}

	// Configure SLS if enabled
	if cfg.Telemetry.SLS.Enabled {
		telemetryConfig.SLS = &telemetry.SLSConfig{
			Endpoint:        cfg.Telemetry.SLS.Endpoint,
			AccessKeyID:     cfg.Telemetry.SLS.AccessKeyID,
			AccessKeySecret: cfg.Telemetry.SLS.AccessKeySecret,
			Project:         cfg.Telemetry.SLS.Project,
			Logstore:        cfg.Telemetry.SLS.Logstore,
			Topic:           cfg.Telemetry.SLS.Topic,
			Source:          cfg.Telemetry.SLS.Source,
		}
	}

	// Configure Prometheus if enabled
	if cfg.Telemetry.Prometheus.Enabled {
		fmt.Println("telemetry Prometheus enable")
		prometheusConfigData, _ := json.Marshal(cfg.Telemetry.Prometheus)
		fmt.Println("prometheus config:", string(prometheusConfigData))
		telemetryConfig.Prometheus = &telemetry.PrometheusConfig{
			Enabled:      cfg.Telemetry.Prometheus.Enabled,
			Path:         cfg.Telemetry.Prometheus.Path,
			PushGateway:  cfg.Telemetry.Prometheus.PushGateway,
			PushInterval: cfg.Telemetry.Prometheus.PushInterval,
			JobName:      cfg.Telemetry.Prometheus.JobName,
			AccessKey:    cfg.Telemetry.SLS.AccessKeyID,
			SecretKey:    cfg.Telemetry.SLS.AccessKeySecret,
			Grouping: map[string]string{
				"appname": appname,
				"host":    hostname,
				"ip":      hostIP,
			},
		}
	} else {
		fmt.Println("telemetry Prometheus disable")
	}

	// Configure tracing if enabled
	if cfg.Telemetry.Tracing.Enabled {
		telemetryConfig.Tracing = &telemetry.TracingConfig{
			Enabled:        cfg.Telemetry.Tracing.Enabled,
			ServiceName:    cfg.Telemetry.Tracing.ServiceName,
			ServiceVersion: cfg.Telemetry.Tracing.ServiceVersion,
			Environment:    cfg.Telemetry.Tracing.Environment,
			JaegerEndpoint: cfg.Telemetry.Tracing.JaegerEndpoint,
			OTLPEndpoint:   cfg.Telemetry.Tracing.OTLPEndpoint,
			SamplingRatio:  cfg.Telemetry.Tracing.SamplingRatio,
		}
	} else {
		fmt.Println("telemetry tracing disable")
	}

	telemetryManager, err := telemetry.NewTelemetryManager(telemetryConfig)
	if err != nil {
		panic(err)
	}
	defer telemetryManager.Close()

	logger := telemetryManager.Logger

	logger.Info("starting grapery api",
		zap.String("env", cfg.Env),
		zap.String("addr", cfg.Addr()),
	)

	// Initialize Aliyun OSS client (optional, graceful degradation to local storage)
	if cfg.Aliyun.APIKey != "" && cfg.Aliyun.SecretKey != "" {
		aliyunCfg := &aliyun.Config{
			APIKey:             cfg.Aliyun.APIKey,
			SecretKey:          cfg.Aliyun.SecretKey,
			Endpoint:           cfg.Aliyun.Endpoint,
			Bucket:             cfg.Aliyun.Bucket,
			RoleARN:            cfg.Aliyun.RoleARN,
			OSSAccessKeyID:     cfg.Aliyun.OSSAccessKeyID,
			OSSAccessKeySecret: cfg.Aliyun.OSSAccessKeySecret,
			OSSRoleARN:         cfg.Aliyun.OSSRoleARN,
		}
		if err := aliyun.InitGlobalClient(aliyunCfg, logger); err != nil {
			logger.Warn("failed to initialize Aliyun OSS client, falling back to local storage", zap.Error(err))
		} else {
			logger.Info("Aliyun OSS client initialized",
				zap.String("bucket", cfg.Aliyun.Bucket),
				zap.String("endpoint", cfg.Aliyun.Endpoint),
			)
		}
	} else {
		logger.Info("Aliyun OSS not configured, using local storage")
	}

	// Initialize MySQL repository
	repo, err := mysql.NewRepository(cfg.Database.DSN(), logger)
	if err != nil {
		logger.Fatal("failed to initialize repository", zap.Error(err))
	}

	// Initialize service
	svc := service.New(repo, logger)

	// Set metrics if enabled
	if telemetryManager.Metrics != nil {
		svc.SetMetrics(telemetryManager.Metrics)
		// Start metrics collection
		ctx := context.Background()
		telemetryManager.Metrics.Start(ctx)
	}

	// Initialize AI clients
	initAIClients(cfg, svc, logger)

	// Initialize Storyboard Chat Service
	storyboardChatService := service.NewStoryboardChatService(repo, svc, logger)
	logger.Info("storyboard chat service initialized")
	// Initialize Writers Room Service
	writersRoomService := service.NewWritersRoomService(repo, logger)
	logger.Info("writers room service initialized")
	// Initialize HTTP handler
	handler := transport.NewHandler(svc, nil, writersRoomService, logger)
	router := transport.SetupRouter(handler, logger)

	// Register Storyboard Chat routes
	storyboardChatHandler := transport.NewStoryboardChatHandler(storyboardChatService, logger)
	apiGroup := router.Group("/api")
	storyboardChatHandler.RegisterRoutes(apiGroup.Group("/agent"), authPkg.AuthMiddleware())
	logger.Info("storyboard chat routes registered")

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", telemetry.CorrelationIDHeader},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	// Add telemetry middleware
	router.Use(telemetry.GinCorrelationMiddleware(logger))
	router.Use(telemetry.GinRequestIDMiddleware(logger))
	if telemetryManager.Tracer != nil {
		router.Use(telemetryManager.Tracer.GinTraceMiddleware())
	}
	if telemetryManager.Metrics != nil {
		router.Use(telemetry.GinHTTPMiddleware(logger, telemetryManager.Metrics))
	}

	// Add metrics endpoint if Prometheus is enabled
	if telemetryManager.Metrics != nil && cfg.Telemetry.Prometheus.Enabled {
		router.GET(cfg.Telemetry.Prometheus.Path, gin.WrapH(telemetryManager.Metrics.Handler()))
		logger.Info("Metrics endpoint registered", zap.String("path", cfg.Telemetry.Prometheus.Path))
	}

	// Setup Writers Room Handler and register routes
	writersRoomHandler := transport.NewHandler(nil, nil, writersRoomService, logger)
	writersRoomHandler.RegisterWritersRoomRoutes(apiGroup.Group("/stories"), apiGroup.Group("/writers-rooms"))
	// Setup graceful shutdown
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize server
	srv := server.New(cfg, router)

	// Start metrics pusher if configured
	if telemetryManager.Metrics != nil && cfg.Telemetry.Prometheus.PushGateway != "" {
		go telemetryManager.Metrics.Start(context.Background())
		logger.Info("Metrics pusher started",
			zap.String("gateway", cfg.Telemetry.Prometheus.PushGateway),
			zap.Int("interval", cfg.Telemetry.Prometheus.PushInterval),
		)
	}

	// Start user statistics persistence task (runs daily at midnight)
	if svc.UserStatsService() != nil {
		go startUserStatisticsTask(shutdownCtx, svc.UserStatsService(), logger)
		logger.Info("User statistics task started")
	}

	// Start invitation expiry check task (runs hourly)
	go startInvitationExpiryTask(shutdownCtx, svc, logger)
	logger.Info("Invitation expiry check task started")

	// Start server in goroutine
	go func() {
		logger.Info("server listening", zap.String("addr", cfg.Addr()))
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	<-shutdownCtx.Done()
	logger.Info("shutdown signal received")

	// Graceful shutdown
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}

	logger.Info("server stopped")
}

// startUserStatisticsTask 启动用户统计持久化任务（每天凌晨执行）
func startUserStatisticsTask(ctx context.Context, statsService *service.UserStatisticsService, logger *zap.Logger) {
	// 立即执行一次（用于初始化）
	go func() {
		statsCtx := context.Background()
		if err := statsService.PersistStatistics(statsCtx, time.Now()); err != nil {
			logger.Warn("failed to persist user statistics", zap.Error(err))
		}
	}()

	// 计算到下一个凌晨的时间
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	durationUntilMidnight := nextMidnight.Sub(now)

	// 等待到凌晨
	select {
	case <-time.After(durationUntilMidnight):
		// 到达凌晨，开始定时任务
	case <-ctx.Done():
		return
	}

	// 每天凌晨执行一次
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			statsCtx := context.Background()
			if err := statsService.PersistStatistics(statsCtx, time.Now()); err != nil {
				logger.Warn("failed to persist user statistics", zap.Error(err))
			} else {
				logger.Info("user statistics persisted successfully")
			}
		case <-ctx.Done():
			return
		}
	}
}

// startInvitationExpiryTask 启动邀请过期检查任务（每小时执行）
func startInvitationExpiryTask(ctx context.Context, svc *service.Service, logger *zap.Logger) {
	// 立即执行一次（用于初始化）
	if err := svc.CheckAndExpireInvitations(ctx); err != nil {
		logger.Warn("failed to check expired invitations", zap.Error(err))
	}
	// 每小时执行一次
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := svc.CheckAndExpireInvitations(ctx); err != nil {
				logger.Warn("failed to check expired invitations", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

// initAIClients initializes AI generation clients based on configuration
func initAIClients(cfg config.Config, svc *service.Service, logger *zap.Logger) {
	logger.Info("========== AI Configuration Check ==========")

	genAPI := genapi.NewGenAPI()
	var geminiClient *gemini.Client
	hasProvider := false
	var configuredProviders []string
	var missingProviders []string

	// Check and register Huoshan provider (火山引擎/豆包)
	if cfg.AI.HuoshanAPIKey != "" {
		huoshanCfg := &genapi.Config{
			Provider: genapi.ProviderHuoshan,
			APIKey:   cfg.AI.HuoshanAPIKey,
			BaseURL:  cfg.AI.HuoshanBaseURL,
		}
		if _, err := genAPI.RegisterProviderConfig(huoshanCfg); err != nil {
			logger.Error("❌ Huoshan provider registration failed",
				zap.Error(err),
				zap.String("baseURL", cfg.AI.HuoshanBaseURL),
			)
		} else {
			logger.Info("✅ Huoshan provider registered",
				zap.String("baseURL", cfg.AI.HuoshanBaseURL),
				zap.Int("apiKeyLength", len(cfg.AI.HuoshanAPIKey)),
			)
			configuredProviders = append(configuredProviders, "huoshan")
			hasProvider = true
		}
	} else {
		missingProviders = append(missingProviders, "huoshan (HUOSHAN_API_KEY)")
	}

	// Check and register Gemini provider
	if cfg.AI.GeminiAPIKey != "" {
		geminiCfg := &genapi.Config{
			Provider: genapi.ProviderGemini,
			APIKey:   cfg.AI.GeminiAPIKey,
			BaseURL:  cfg.AI.GeminiBaseURL,
		}
		if _, err := genAPI.RegisterProviderConfig(geminiCfg); err != nil {
			logger.Error("❌ Gemini provider registration failed",
				zap.Error(err),
				zap.String("baseURL", cfg.AI.GeminiBaseURL),
			)
		} else {
			baseURL := cfg.AI.GeminiBaseURL
			if baseURL == "" {
				baseURL = "(default)"
			}
			logger.Info("✅ Gemini provider registered",
				zap.String("baseURL", baseURL),
				zap.Int("apiKeyLength", len(cfg.AI.GeminiAPIKey)),
			)
			configuredProviders = append(configuredProviders, "gemini")
			hasProvider = true

			// Create direct Gemini client for AIGenerationService
			var err error
			geminiClient, err = gemini.New(gemini.Config{
				APIKey:  cfg.AI.GeminiAPIKey,
				BaseURL: cfg.AI.GeminiBaseURL,
			})
			if err != nil {
				logger.Error("❌ Failed to create Gemini client",
					zap.Error(err),
				)
				geminiClient = nil
			} else {
				logger.Info("✅ Gemini client created for AI generation service")
			}
		}
	} else {
		missingProviders = append(missingProviders, "gemini (GEMINI_API_KEY)")
	}

	// Summary log
	logger.Info("========== AI Configuration Summary ==========")

	if len(missingProviders) > 0 {
		logger.Warn("⚠️  Missing AI provider configurations",
			zap.Strings("missing", missingProviders),
		)
	}

	// Set AI clients on service
	if hasProvider {
		svc.SetAIClients(genAPI, geminiClient)
		svc.SetAIConfig(cfg.AI) // Set image/video provider configuration
		logger.Info("✅ AI generation service initialized",
			zap.Strings("providers", configuredProviders),
			zap.String("defaultProvider", cfg.AI.DefaultProvider),
			zap.String("imageProvider", cfg.AI.ImageProvider),
			zap.String("videoProvider", cfg.AI.VideoProvider),
			zap.Bool("geminiClientAvailable", geminiClient != nil),
		)
	} else {
		logger.Error("❌ No AI providers configured - AI features will be DISABLED",
			zap.String("action", "Set at least one of: GEMINI_API_KEY, HUOSHAN_API_KEY"),
		)
	}

	logger.Info("==============================================")
}
