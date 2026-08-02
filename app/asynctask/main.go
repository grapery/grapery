package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/models"
	asynctaskpkg "github.com/grapery/grapery/pkg/asynctask"
	"github.com/grapery/grapery/pkg/genapi"
	asynctasksvc "github.com/grapery/grapery/service/asynctask"
	"github.com/grapery/grapery/version"
)

var (
	printVersion = flag.Bool("version", false, "app build version")
	configPath   = flag.String("config", "asynctask.json", "config file")
)

func main() {
	flag.Parse()
	if *printVersion {
		version.PrintFullVersionInfo()
		return
	}

	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	if err := config.ValiedConfig(config.GlobalConfig); err != nil {
		log.Fatalf("validate config failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	if err := models.Init(
		config.GlobalConfig.SqlDB.Username,
		config.GlobalConfig.SqlDB.Password,
		config.GlobalConfig.SqlDB.Address,
		config.GlobalConfig.SqlDB.Database,
	); err != nil {
		log.Fatalf("init database failed: %v", err)
	}
	defer func() {
		if err := models.Close(); err != nil {
			log.Errorf("close database failed: %v", err)
		}
	}()

	redisOpt, err := redisClientOpt(config.GlobalConfig)
	if err != nil {
		log.Fatalf("build redis options failed: %v", err)
	}

	// 初始化GenAPI和Providers
	genAPI := genapi.NewGenAPI()
	videoProviders, err := setupProviders(genAPI, config.GlobalConfig)
	if err != nil {
		log.Fatalf("setup providers failed: %v", err)
	}
	if len(videoProviders) == 0 {
		log.Warn("no video providers enabled, video generation will not work")
	} else {
		log.Infof("initialized %d video providers", len(videoProviders))
	}

	videoHandler := asynctaskpkg.NewVideoTaskHandler(videoProviders)
	manager, err := asynctaskpkg.NewTaskManager(redisOpt, videoHandler, asynctaskpkg.TaskManagerConfig{
		Concurrency: 4,
		Queues: map[string]int{
			asynctaskpkg.VideoQueueName: 4,
			"default":                   1,
		},
	})
	if err != nil {
		log.Fatalf("create task manager failed: %v", err)
	}
	defer manager.Shutdown()

	if err := manager.Start(ctx); err != nil {
		log.Fatalf("start task manager failed: %v", err)
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	server := asynctasksvc.New(config.GlobalConfig, manager)
	server.RegisterRoutes(router)

	port := config.GlobalConfig.HttpPort
	if config.GlobalConfig.Asynctask != nil && config.GlobalConfig.Asynctask.HttpPort != "" {
		port = config.GlobalConfig.Asynctask.HttpPort
	}
	if port == "" {
		port = "8050"
	}

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		Handler: router,
	}

	done := make(chan struct{})
	go func() {
		log.Infof("async task server listening on %s", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
		close(done)
	}()

	<-ctx.Done()
	log.Info("shutdown signal received, stopping async task service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Errorf("http server shutdown error: %v", err)
	}
	<-done
}

func redisClientOpt(cfg *config.Config) (asynq.RedisClientOpt, error) {
	if cfg.Redis == nil {
		return asynq.RedisClientOpt{}, fmt.Errorf("redis config is nil")
	}
	redisDB := 0
	if cfg.Redis.Database != "" {
		value, err := strconv.Atoi(cfg.Redis.Database)
		if err != nil {
			return asynq.RedisClientOpt{}, fmt.Errorf("parse redis db: %w", err)
		}
		redisDB = value
	}
	return asynq.RedisClientOpt{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       redisDB,
	}, nil
}

// setupProviders 从配置中初始化和注册所有video providers
func setupProviders(api *genapi.GenAPI, cfg *config.Config) (map[string]genapi.VideoProvider, error) {
	videoProviders := make(map[string]genapi.VideoProvider)

	if cfg.Asynctask == nil || cfg.Asynctask.Providers == nil {
		log.Warn("no providers configured in asynctask config")
		return videoProviders, nil
	}

	for name, providerCfg := range cfg.Asynctask.Providers {
		if providerCfg == nil || !providerCfg.Enabled {
			log.Infof("provider %s is disabled, skipping", name)
			continue
		}

		// 从环境变量解析API密钥
		apiKey := expandEnvVar(providerCfg.APIKey)
		if apiKey == "" {
			log.Warnf("provider %s has no API key, skipping", name)
			continue
		}

		// 转换为GenAPI配置
		timeout := time.Duration(providerCfg.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		genapiCfg := &genapi.Config{
			Provider:     genapi.ProviderKind(name),
			APIKey:       apiKey,
			Secret:       providerCfg.Secret,
			BaseURL:      providerCfg.BaseURL,
			ImageBaseURL: providerCfg.ImageBaseURL,
			Model:        providerCfg.Model,
			ImageModel:   providerCfg.ImageModel,
			Workflow:     providerCfg.Workflow,
			Timeout:      timeout,
		}

		if len(providerCfg.Additional) > 0 {
			genapiCfg.Additional = make(map[string]interface{})
			for k, v := range providerCfg.Additional {
				genapiCfg.Additional[k] = v
			}
		}

		// 注册到GenAPI
		provider, err := api.RegisterProviderConfig(genapiCfg)
		if err != nil {
			log.Errorf("register provider %s failed: %v", name, err)
			continue
		}

		// 检查是否支持视频生成
		if vp, ok := provider.(genapi.VideoProvider); ok {
			videoProviders[name] = vp
			log.Infof("registered video provider: %s", name)
		} else {
			log.Warnf("provider %s does not support video generation", name)
		}
	}

	return videoProviders, nil
}

// expandEnvVar 展开环境变量 ${VAR_NAME}
func expandEnvVar(s string) string {
	if s == "" {
		return s
	}

	return os.Getenv(s)
}
