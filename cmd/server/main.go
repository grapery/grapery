package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"github.com/grapestree/fgrapery/grapery/internal/handler"
	"github.com/grapestree/fgrapery/grapery/internal/middleware"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
	"github.com/grapestree/fgrapery/grapery/internal/repository/migrations"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	_ "github.com/grapestree/fgrapery/grapery/internal/repository/mysql" // Register migrations
	_ "github.com/grapestree/fgrapery/grapery/internal/repository/pay"   // Register payment migrations
	"github.com/grapestree/fgrapery/grapery/internal/server"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	transport "github.com/grapestree/fgrapery/grapery/internal/transport/http"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
)

func main() {
	utils.LoadDotEnvFiles()
	if _, err := os.Stat(".env"); err != nil {
		if _, err2 := os.Stat("grapery/.env"); err2 != nil {
			fmt.Fprintf(os.Stderr, "Warning: no .env file found in cwd or grapery/\n")
		}
	}

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

	cfg.Telemetry.SLS.Enabled = false
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

	cfg.Telemetry.Tracing.Enabled = false
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

	// Configure JWT Secret (must match vippay service)
	logger.Info("========== JWT Configuration ==========")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Info("JWT_SECRET environment variable not set")
		jwtSecret = cfg.JWT.Secret
		if jwtSecret == "" {
			logger.Error("JWT_SECRET not configured - authentication will fail")
			logger.Error("Set JWT_SECRET environment variable or configure it in the config file")
			// Continue without secret - token generation will fail with ErrSecretNotSet
		} else {
			logger.Info("JWT Secret loaded from config file")
		}
	} else {
		logger.Info("JWT Secret loaded from environment variable")
	}

	// 记录JWT Secret的长度和预览（安全性考虑不记录完整值）
	logger.Info("JWT Secret configuration",
		zap.Int("secret_length", len(jwtSecret)),
		zap.String("secret_preview", jwtSecret[:min(len(jwtSecret), 10)]+"..."),
		zap.Bool("from_env", os.Getenv("JWT_SECRET") != ""),
		zap.Bool("is_default", jwtSecret == "grapery-secret-key-change-in-production"),
	)

	authPkg.SetJWTSecret(jwtSecret)
	logger.Info("=========================================")

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
	db, err := mysql.InitDB(cfg.Database.DSN(), logger)
	if err != nil {
		logger.Fatal("failed to initialize database connection", zap.Error(err))
	}
	repo := mysql.NewRepository(db, logger, cfg.Recommendation)

	// Run database migrations
	logger.Info("running database migrations")
	registry := migrations.GetRegistry()
	ctx := context.Background()
	if err := registry.ExecuteAll(ctx, repo.DB(), logger); err != nil {
		logger.Fatal("failed to run database migrations", zap.Error(err))
	}
	logger.Info("database migrations completed successfully")

	// Initialize service
	svc := service.New(repo, logger, cfg.Recommendation)

	apnsKeyPEM, apnsKeyErr := config.APNsKeyPEM(cfg.APNs)
	if apnsKeyErr != nil {
		logger.Warn("APNs auth key not loaded; remote push disabled", zap.Error(apnsKeyErr))
	} else if strings.TrimSpace(apnsKeyPEM) != "" {
		keySource := "APNS_PRIVATE_KEY"
		if strings.TrimSpace(cfg.APNs.PrivateKey) == "" && strings.TrimSpace(cfg.APNs.PrivateKeyPath) != "" {
			keySource = "file:" + cfg.APNs.PrivateKeyPath
		}
		logger.Info("APNs key material loaded",
			zap.String("keySource", keySource),
			zap.String("keyId", cfg.APNs.KeyID),
			zap.Bool("sandbox", cfg.APNs.UseSandbox))
		apnsSvc := service.NewAPNsService(&service.APNsConfig{
			BundleID:   cfg.APNs.BundleID,
			TeamID:     cfg.APNs.TeamID,
			KeyID:      cfg.APNs.KeyID,
			PrivateKey: apnsKeyPEM,
			UseSandbox: cfg.APNs.UseSandbox,
		}, logger)
		svc.SetAPNsService(apnsSvc)
		if apnsSvc.IsEnabled() {
			logger.Info("APNs push delivery enabled",
				zap.String("bundleId", cfg.APNs.BundleID),
				zap.String("teamId", cfg.APNs.TeamID),
				zap.String("keyId", cfg.APNs.KeyID),
				zap.Bool("sandbox", cfg.APNs.UseSandbox),
			)
		} else {
			logger.Warn("APNs .p8 loaded but push remains disabled — check key parse error above, or APNS_BUNDLE_ID / APNS_TEAM_ID / APNS_KEY_ID",
				zap.String("bundleId", cfg.APNs.BundleID),
				zap.String("teamId", cfg.APNs.TeamID),
				zap.String("keyId", cfg.APNs.KeyID),
				zap.Bool("sandbox", cfg.APNs.UseSandbox),
			)
		}
	} else {
		logger.Info("APNs not configured (empty key); remote push disabled")
	}

	var redisCache cache.Cache
	var aiRedisClient *redis.Client
	redisCache, err = cache.NewRedisCache(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.Database, logger)
	if err != nil {
		logger.Warn("failed to initialize redis cache; recommendation cache disabled", zap.Error(err))
	} else {
		svc.SetCache(redisCache)
		repo.SetCache(redisCache)
		aiRedisClient = redis.NewClient(&redis.Options{
			Addr:         cfg.Redis.Address,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.Database,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
			MinIdleConns: 5,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pingErr := aiRedisClient.Ping(ctx).Err()
		cancel()
		if pingErr != nil {
			logger.Warn("failed to initialize ai redis client", zap.Error(pingErr))
			_ = aiRedisClient.Close()
			aiRedisClient = nil
		} else {
			defer func() {
				if aiRedisClient != nil {
					_ = aiRedisClient.Close()
				}
			}()
		}
	}

	// Global outbound text-LLM concurrency (Redis-backed; disables if Redis unavailable or text_max_concurrent=0).
	svc.ConfigureAITextAdmission(redisCache, cfg.AI.TextMaxConcurrent)

	// Set metrics if enabled
	if telemetryManager.Metrics != nil {
		svc.SetMetrics(telemetryManager.Metrics)
	}

	// Initialize AI clients
	initAIClients(cfg, svc, repo, logger)
	if aiSvc := svc.AIGenerationService(); aiSvc != nil && aiRedisClient != nil {
		aiSvc.SetRedisClient(aiRedisClient)
	}

	// 与 Fragment 生成等共用同一套 AI 依赖；HTTP 路由上的 /ai/*（含 enhance-prompt）必须非 nil，否则异步 goroutine 会在 nil 接收者上 panic。
	aiSvc := svc.AIService()

	// Initialize Fragment repositories and service
	fragmentGenRepo := repository.NewFragmentGenerationRepository(repo.DB())
	fragmentRepo := repository.NewFragmentRepository(repo.DB(), cfg.Recommendation, redisCache, logger)
	svc.SetAccountDeletionDeps(cfg.AccountDeletion, fragmentRepo)
	fragmentGenService := service.NewFragmentGenerationService(fragmentGenRepo, fragmentRepo, repo, aiSvc, logger)
	fragmentGenService.SetNotify(svc)
	logger.Info("fragment generation service initialized")

	panelGenRepo := repository.NewFragmentPanelGenerationRepository(repo.DB())
	panelGenService := service.NewFragmentPanelGenerationService(panelGenRepo, fragmentRepo, repo, cfg.AI.ImageProvider, svc.AIGenerationService(), aiSvc, logger)
	panelGenService.SetNotify(svc)
	logger.Info("fragment panel generation service initialized")

	// Initialize Fragment Interaction Service
	fragmentInteractionRepo := repository.NewFragmentInteractionRepository(repo.DB())
	logger.Info("fragment interaction repository initialized")

	// Initialize User Settings Repository
	userSettingsRepo := mysql.NewUserSettingsRepository(repo.DB())
	logger.Info("user settings repository initialized")

	genreCatalogRepo := mysql.NewGenreCatalogRepository(repo.DB())
	genreCatalogService := service.NewGenreCatalogService(genreCatalogRepo, svc.AIGenerationService(), logger)
	logger.Info("genre catalog service initialized")

	// Comic style batches (DB + AI backfill)
	comicStyleSvc := service.NewFragmentComicStyleService(repo.DB(), svc.AIGenerationService(), logger)
	svc.SetFragmentComicStyleService(comicStyleSvc)

	// Initialize Fragment Handler
	fragmentHandler := handler.NewFragmentHandler(fragmentRepo, userSettingsRepo, repo, svc, comicStyleSvc)
	logger.Info("fragment handler initialized")

	// Initialize Like Repository
	likeRepo := mysql.NewLikeRepository(repo.DB())
	logger.Info("like repository initialized")

	// Initialize Bookmark Repository (StoryCreationAppUI)
	bookmarkRepo := mysql.NewBookmarkRepository(repo.DB())
	logger.Info("bookmark repository initialized")

	// Initialize Interaction Service
	interactionService := service.NewInteractionService(likeRepo, bookmarkRepo, repo, svc, fragmentInteractionRepo, logger)
	logger.Info("interaction service initialized")

	// Initialize User Settings Service
	userSettingsService := service.NewUserSettingsService(userSettingsRepo, genreCatalogRepo, logger, redisCache)
	logger.Info("user settings service initialized")

	feedbackRepo := mysql.NewFeedbackRepository(repo.DB())
	feedbackService := service.NewFeedbackService(feedbackRepo, logger)
	logger.Info("feedback service initialized")

	// Initialize Storyboard Path Service
	storyboardPathService := service.NewStoryboardPathService(repo, logger)
	if redisCache != nil {
		storyboardPathService.SetCache(redisCache)
	}
	logger.Info("storyboard path service initialized")

	// Initialize HTTP handler with dependencies (V1/V2 MVP - removed WritersRoom and GroupShowcase)
	deps := &transport.HandlerDependencies{
		Service:               svc,
		AIService:             aiSvc,
		StoryboardPathService: storyboardPathService,
		InteractionService:    interactionService,
		UserSettingsService:   userSettingsService,
		GenreCatalogService:   genreCatalogService,
		FeedbackService:       feedbackService,
		Logger:                logger,
		Cache:                 redisCache,
	}
	router := transport.SetupRouter(deps)

	// Create API group for route registration
	apiGroup := router.Group("/api")

	// Register Fragment Generation routes
	fragmentGenHandler := transport.NewFragmentGenerationHandler(fragmentGenService, fragmentHandler, logger)
	// AI rate limiter for fragment generation endpoints
	fragGenGroup := apiGroup.Group("/v1/fragments")
	fragGenGroup.Use(authPkg.AuthMiddleware())
	fragGenGroup.Use(middleware.NewRateLimiter(redisCache, middleware.RateLimitAIGeneration, logger))
	fragmentGenHandler.RegisterRoutes(fragGenGroup, nil) // auth middleware already applied above
	logger.Info("fragment generation routes registered")

	panelGenHandler := transport.NewFragmentPanelGenerationHandler(panelGenService, logger)
	panelGenGroup := apiGroup.Group("/v1/fragment-panels")
	panelGenGroup.Use(authPkg.AuthMiddleware())
	panelGenGroup.Use(middleware.NewRateLimiter(redisCache, middleware.RateLimitAIGeneration, logger))
	panelGenHandler.RegisterRoutes(panelGenGroup, nil) // auth middleware already applied above
	logger.Info("fragment panel generation routes registered")

	// Register Fragment Interaction routes (likes, comments, shares)
	fragmentInteractionHandler := transport.NewFragmentInteractionHandler(fragmentInteractionRepo, fragmentRepo, svc, logger)
	fragmentInteractionHandler.RegisterRoutes(apiGroup.Group("/v1/fragments"), authPkg.AuthMiddleware())
	logger.Info("fragment interaction routes registered")

	// Register Fragment CRUD routes (list, get, create, update, delete)
	// Note: /styles must be registered BEFORE /:id to avoid route conflicts
	// Use OptionalAuthMiddleware for GET endpoints to support both authenticated and unauthenticated access
	apiGroup.GET("/v1/fragments", authPkg.OptionalAuthMiddleware(), fragmentHandler.ListFragments)
	apiGroup.GET("/v1/fragments/styles", fragmentHandler.GetFragmentStyles)
	apiGroup.POST("/v1/fragments/styles/next", authPkg.AuthMiddleware(), fragmentHandler.PostFragmentStylesNext)
	apiGroup.GET("/v1/fragments/:id", authPkg.OptionalAuthMiddleware(), fragmentHandler.GetFragment)
	apiGroup.POST("/v1/fragments", authPkg.AuthMiddleware(), fragmentHandler.CreateFragment)
	apiGroup.PUT("/v1/fragments/:id", authPkg.AuthMiddleware(), fragmentHandler.UpdateFragment)
	apiGroup.DELETE("/v1/fragments/:id", authPkg.AuthMiddleware(), fragmentHandler.DeleteFragment)
	logger.Info("fragment CRUD routes registered")

	// Register User Fragments route (optional auth for checking likes)
	apiGroup.GET("/v1/users/:id/fragments", authPkg.OptionalAuthMiddleware(), fragmentHandler.GetUserFragments)
	logger.Info("user fragments route registered")

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

	// Start phased account deletion worker (grace window completion)
	go startAccountDeletionProcessor(shutdownCtx, svc, logger)
	logger.Info("Account deletion processor started")

	// Start user statistics persistence task (runs daily at midnight)
	if svc.UserStatsService() != nil {
		go startUserStatisticsTask(shutdownCtx, svc.UserStatsService(), logger)
		logger.Info("User statistics task started")
	}

	// Start server in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("server goroutine panic recovered", zap.Any("panic", r))
			}
		}()
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

func startAccountDeletionProcessor(ctx context.Context, svc *service.Service, logger *zap.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			svc.ProcessDueAccountDeletionBatch(runCtx, 30)
			cancel()
		}
	}
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

// initAIClients initializes AI generation clients based on configuration
func initAIClients(cfg config.Config, svc *service.Service, repo domain.Repository, logger *zap.Logger) {
	logger.Info("========== AI Configuration Check ==========")

	// Set TokenUsageRecorder before creating GenAPI so all image/video generations are persisted
	genapi.SetTokenUsageRecorder(service.NewGenAPIUsageRecorder(repo, func(msg string, err error) {
		logger.Warn(msg, zap.Error(err))
	}))

	genAPI := genapi.NewGenAPI()
	var geminiClient *gemini.Client
	hasProvider := false
	var configuredProviders []string
	var missingProviders []string

	aiReqSec := cfg.AI.RequestTimeoutSeconds
	if aiReqSec <= 0 {
		aiReqSec = 180
	}
	aiHTTPTimeout := time.Duration(aiReqSec) * time.Second
	logger.Info("AI outbound HTTP client timeout",
		zap.Int("seconds", aiReqSec),
		zap.Duration("duration", aiHTTPTimeout),
	)

	// Check and register Huoshan provider (火山引擎/豆包)
	if cfg.AI.HuoshanAPIKey != "" {
		huoshanSec := cfg.AI.EffectiveHuoshanRequestTimeoutSeconds()
		huoshanHTTPTimeout := time.Duration(huoshanSec) * time.Second
		if huoshanSec != aiReqSec {
			logger.Info("Huoshan outbound HTTP timeout differs from global AI timeout (Seedream streaming / image payloads)",
				zap.Int("huoshan_seconds", huoshanSec),
				zap.Int("global_seconds", aiReqSec),
			)
		}
		huoshanCfg := &genapi.Config{
			Provider:   genapi.ProviderHuoshan,
			APIKey:     cfg.AI.HuoshanAPIKey,
			BaseURL:    cfg.AI.HuoshanBaseURL,
			ImageModel: cfg.AI.HuoshanImageModel,
			TextModel:  cfg.AI.HuoshanTextModel,
			Timeout:    huoshanHTTPTimeout,
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

	// Kling (可灵) — AccessKey + SecretKey, JWT to api-singapore.klingai.com
	if cfg.AI.KlingAccessKey != "" && cfg.AI.KlingSecretKey != "" {
		klingCfg := &genapi.Config{
			Provider: genapi.ProviderKling,
			APIKey:   cfg.AI.KlingAccessKey,
			Secret:   cfg.AI.KlingSecretKey,
			BaseURL:  cfg.AI.KlingBaseURL,
			Timeout:  aiHTTPTimeout,
		}
		if _, err := genAPI.RegisterProviderConfig(klingCfg); err != nil {
			logger.Error("❌ Kling provider registration failed",
				zap.Error(err),
				zap.String("baseURL", cfg.AI.KlingBaseURL),
			)
		} else {
			base := cfg.AI.KlingBaseURL
			if base == "" {
				base = "(default api-singapore.klingai.com)"
			}
			logger.Info("✅ Kling provider registered",
				zap.String("baseURL", base),
				zap.Int("accessKeyLength", len(cfg.AI.KlingAccessKey)),
			)
			configuredProviders = append(configuredProviders, "kling")
			hasProvider = true
		}
	} else {
		missingProviders = append(missingProviders, "kling (KLING_ACCESS_KEY + KLING_SECRET_KEY)")
	}

	// Check and register Gemini provider
	if cfg.AI.GeminiAPIKey != "" {
		geminiCfg := &genapi.Config{
			Provider: genapi.ProviderGemini,
			APIKey:   cfg.AI.GeminiAPIKey,
			BaseURL:  cfg.AI.GeminiBaseURL,
			Timeout:  aiHTTPTimeout,
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
				Timeout: aiHTTPTimeout,
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
