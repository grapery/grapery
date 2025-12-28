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
		fmt.Println("Grapery Agent Chat Service v1.0.0")
		os.Exit(0)
	}

	// Get host information
	hostname := utils.GetHostname()
	hostIP := utils.GetHostIP()

	// Load configuration
	var cfg config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.LoadFromFile(*configPath, "chatmcp")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Fallback to environment variables or default config.yaml
		if _, err := os.Stat("config.yaml"); err == nil {
			cfg, err = config.LoadFromFile("config.yaml", "chatmcp")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load config.yaml: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("loading config from environment variables")
			cfg = config.Load("chatmcp")
		}
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
				"appname": "chatmcp",
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

	logger.Info("starting grapery agent chat service",
		zap.String("env", cfg.Env),
		zap.String("addr", cfg.Addr()),
	)

	// Initialize Aliyun OSS client (optional)
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
			logger.Warn("failed to initialize Aliyun OSS client", zap.Error(err))
		} else {
			logger.Info("Aliyun OSS client initialized",
				zap.String("bucket", cfg.Aliyun.Bucket),
			)
		}
	}

	// Initialize MySQL repository
	repo, err := mysql.NewRepository(cfg.Database.DSN(), logger)
	if err != nil {
		logger.Fatal("failed to initialize repository", zap.Error(err))
	}
	logger.Info("database connection established")

	// Initialize Gemini Client for Agent Chat
	var geminiClient *gemini.Client
	if cfg.AI.GeminiAPIKey != "" {
		geminiClient, err = gemini.New(gemini.Config{
			APIKey:       cfg.AI.GeminiAPIKey,
			BaseURL:      cfg.AI.GeminiBaseURL,
			Timeout:      60 * time.Second,
			DefaultModel: "gemini-2.5-flash",
		})
		if err != nil {
			logger.Fatal("failed to initialize Gemini client", zap.Error(err))
		}
		logger.Info("initialized Gemini AI client")
	} else {
		logger.Fatal("Gemini API key is required for agent chat service")
	}

	// Initialize Agent Chat Service
	agentChatService := service.NewAgentChatService(repo, geminiClient, logger)
	logger.Info("agent chat service initialized")

	// Initialize Story Service (for storyboard chat)
	storyService := service.New(repo, logger)
	storyService.SetAIClients(nil, geminiClient)
	logger.Info("story service initialized")

	// Initialize Storyboard Chat Service
	storyboardChatService := service.NewStoryboardChatService(repo, storyService, logger)
	logger.Info("storyboard chat service initialized")

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "grapery-agent-chat",
		})
	})

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

	// API routes
	api := router.Group("/api")
	{
		// Public routes (no auth required)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"service": "grapery-agent-chat",
			})
		})

		// Setup Agent Chat Handler and register routes
		agentChatHandler := transport.NewAgentChatHandler(agentChatService, logger)
		agentChatHandler.RegisterRoutes(api, authPkg.AuthMiddleware())

		// Setup Storyboard Chat Handler and register routes
		storyboardChatHandler := transport.NewStoryboardChatHandler(storyboardChatService, logger)
		storyboardChatHandler.RegisterRoutes(api.Group("/agent"), authPkg.AuthMiddleware())
	}

	// Log all registered routes
	logger.Info("registered routes:")
	for _, route := range router.Routes() {
		logger.Info("route",
			zap.String("method", route.Method),
			zap.String("path", route.Path),
		)
	}

	// Initialize server
	srv := server.New(cfg, router)

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start metrics pusher if configured
	if telemetryManager.Metrics != nil && cfg.Telemetry.Prometheus.PushGateway != "" {
		go telemetryManager.Metrics.Start(context.Background())
		logger.Info("Metrics pusher started",
			zap.String("gateway", cfg.Telemetry.Prometheus.PushGateway),
			zap.Int("interval", cfg.Telemetry.Prometheus.PushInterval),
		)
	}

	// Start server in goroutine
	go func() {
		logger.Info("agent chat service listening",
			zap.String("addr", cfg.Addr()),
		)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	logger.Info("shutdown signal received")

	// Graceful shutdown
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}

	logger.Info("agent chat service stopped")
}
