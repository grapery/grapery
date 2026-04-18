package service

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"go.uber.org/zap"
)

// AIGenerationConfig AI 生成服务配置
type AIGenerationConfig struct {
	// Redis 配置（用于分布式锁和配额预留）
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// 配额预留配置
	EnableQuotaReservation  bool
	QuotaReservationTimeout time.Duration // 预留过期时间（默认 10 分钟）

	// 分布式锁配置
	DistributedLockTTL time.Duration // 锁的 TTL（默认 30 秒）

	// 异步视频处理配置
	EnableAsyncVideoPolling   bool
	AsyncVideoPollInterval    time.Duration // 轮询间隔（默认 30 秒）
	AsyncVideoMaxPollAttempts int           // 最大轮询次数（默认 120 次）
	AsyncVideoTaskTimeout     time.Duration // 任务超时时间（默认 90 分钟）

	// Webhook 配置
	EnableWebhook bool
	WebhookSecret string // Webhook 签名密钥（可选）
	WebhookPath   string // Webhook 路径（默认 /webhooks）
}

// DefaultAIGenerationConfig 默认配置
func DefaultAIGenerationConfig() *AIGenerationConfig {
	return &AIGenerationConfig{
		RedisAddr:                 "localhost:6379",
		RedisPassword:             "",
		RedisDB:                   0,
		EnableQuotaReservation:    true, // 生产环境建议启用
		QuotaReservationTimeout:   10 * time.Minute,
		DistributedLockTTL:        30 * time.Second,
		EnableAsyncVideoPolling:   true, // 生产环境建议启用
		AsyncVideoPollInterval:    30 * time.Second,
		AsyncVideoMaxPollAttempts: 120,
		AsyncVideoTaskTimeout:     90 * time.Minute,
		EnableWebhook:             true,
		WebhookSecret:             "", // 如果需要验证 webhook 签名，请设置密钥
		WebhookPath:               "/webhooks",
	}
}

// SetupAIGenerationService 设置 AI 生成服务
// 这是一个示例函数，展示如何在应用启动时配置 AI 生成服务
func SetupAIGenerationService(
	aiService *AIGenerationService,
	logger *zap.Logger,
	config *AIGenerationConfig,
) error {
	// 如果没有提供配置，使用默认配置
	if config == nil {
		config = DefaultAIGenerationConfig()
	}

	// 1. 设置 Redis 客户端（用于分布式锁和配额预留）
	if config.EnableQuotaReservation || config.EnableAsyncVideoPolling {
		redisClient, err := createRedisClient(config, logger)
		if err != nil {
			return err
		}
		aiService.SetRedisClient(redisClient)
	}

	// 2. 启用配额预留机制
	if config.EnableQuotaReservation {
		aiService.SetQuotaReservationEnabled(true)
		logger.Info("quota reservation mechanism enabled")
	}

	// 3. 配置异步视频轮询服务
	if config.EnableAsyncVideoPolling {
		asyncVideoService := aiService.GetAsyncVideoCompletionService()
		asyncVideoService.SetPollInterval(config.AsyncVideoPollInterval)
		asyncVideoService.SetMaxPollAttempts(config.AsyncVideoMaxPollAttempts)

		// 启动轮询服务（在单独的 goroutine 中）
		ctx := context.Background()
		go func() {
			logger.Info("starting async video completion polling service")
			asyncVideoService.StartPolling(ctx)
		}()
	}

	return nil
}

// createRedisClient 创建 Redis 客户端
func createRedisClient(config *AIGenerationConfig, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.RedisAddr,
		Password:     config.RedisPassword,
		DB:           config.RedisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", zap.Error(err))
		return nil, err
	}

	logger.Info("redis connected successfully for AI generation service",
		zap.String("addr", config.RedisAddr))

	return client, nil
}

// SetupAIGenerationServiceWithCache 使用现有的 Redis 缓存客户端设置 AI 生成服务
// 如果你的应用已经有 Redis 缓存客户端，可以使用这个方法
func SetupAIGenerationServiceWithCache(
	aiService *AIGenerationService,
	cacheClient cache.Cache,
	logger *zap.Logger,
	config *AIGenerationConfig,
) error {
	// 如果没有提供配置，使用默认配置
	if config == nil {
		config = DefaultAIGenerationConfig()
	}

	// 从缓存获取 Redis 客户端
	// 注意：这需要你的 cache.Cache 接口提供访问底层 Redis 客户端的方法
	// 或者你可以直接传递 redis.Client

	// 启用配额预留机制
	if config.EnableQuotaReservation {
		aiService.SetQuotaReservationEnabled(true)
		logger.Info("quota reservation mechanism enabled")
	}

	return nil
}

// ============================================
// 使用示例
// ============================================
//
// 在你的应用初始化代码中：
//
// func (app *App) InitializeAIGenerationService() error {
//     config := &service.AIGenerationConfig{
//         RedisAddr:                app.Config.Redis.Addr,
//         RedisPassword:            app.Config.Redis.Password,
//         RedisDB:                  app.Config.Redis.DB,
//         EnableQuotaReservation:   true,
//         EnableAsyncVideoPolling:  true,
//         EnableWebhook:            true,
//         WebhookSecret:            app.Config.Webhook.Secret,
//     }
//
//     return service.SetupAIGenerationService(
//         app.aiGenerationService,
//         app.logger,
//         config,
//     )
// }
//
// // 设置 Webhook 路由
// func (app *App) SetupWebhookRoutes(router *gin.Engine) {
//     webhookHandler := handler.NewWebhookHandler(
//         app.aiGenerationService.GetAsyncVideoCompletionService(),
//         app.logger,
//         app.Config.Webhook.Secret,
//     )
//
//     handler.SetupWebhookRoutes(router.Group("/api/v1"), webhookHandler)
// }
