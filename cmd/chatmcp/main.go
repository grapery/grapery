package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"github.com/grapestree/fgrapery/grapery/internal/server"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	transport "github.com/grapestree/fgrapery/grapery/internal/transport/http"
	"go.uber.org/zap"
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
			cfg = config.Load("chatmcp")
		}
	}

	// Initialize logger
	logger, err := telemetry.NewLogger(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("starting grapery agent chat service",
		zap.String("env", cfg.Env),
		zap.String("addr", cfg.Addr()),
	)

	// Initialize Aliyun OSS client (optional)
	if cfg.Aliyun.APIKey != "" && cfg.Aliyun.SecretKey != "" {
		aliyunCfg := &aliyun.Config{
			APIKey:    cfg.Aliyun.APIKey,
			SecretKey: cfg.Aliyun.SecretKey,
			Endpoint:  cfg.Aliyun.Endpoint,
			Bucket:    cfg.Aliyun.Bucket,
			RoleARN:   cfg.Aliyun.RoleARN,
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
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

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
