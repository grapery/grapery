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
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/models"
	paypkg "github.com/grapery/grapery/pkg/pay"
	"github.com/grapery/grapery/service/pay"
	"github.com/grapery/grapery/utils/cache"
	"github.com/grapery/grapery/version"
)

var printVersion = flag.Bool("version", false, "app build version")
var configPath = flag.String("config", "vippay.json", "config file")

func main() {
	flag.Parse()
	if *printVersion {
		version.PrintFullVersionInfo()
		return
	}

	// 加载配置
	err := config.LoadConfig(*configPath)
	if err != nil {
		logrus.Fatal("read config failed : ", err)
	}
	err = config.ValiedConfig(config.GlobalConfig)
	if err != nil {
		logrus.Fatal("Validate config failed : ", err)
	}

	// 初始化数据库和缓存
	err = initializeServices()
	if err != nil {
		logrus.Fatal("initialize services failed : ", err)
	}

	// 创建 Gin 引擎
	router := createGinEngine()

	// 注册路由
	registerRoutes(router)

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         ":" + config.GlobalConfig.VipPay.HttpPort,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务器
	go func() {
		logrus.Infof("Starting VIP payment server on port %s", config.GlobalConfig.VipPay.HttpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatal("start server failed", zap.Error(err))
		}
	}()

	// 优雅关闭
	gracefulShutdown(server)
}

// initializeServices 初始化服务
func initializeServices() error {
	// 初始化 Redis 缓存
	cache.NewRedisClient(config.GlobalConfig)

	// 初始化数据库
	err := models.Init(
		config.GlobalConfig.SqlDB.Username,
		config.GlobalConfig.SqlDB.Password,
		config.GlobalConfig.SqlDB.Address,
		config.GlobalConfig.SqlDB.Database,
	)
	if err != nil {
		return err
	}

	return nil
}

// createGinEngine 创建 Gin 引擎
func createGinEngine() *gin.Engine {
	// 设置 Gin 模式
	if config.GlobalConfig.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建引擎
	router := gin.New()

	// 添加中间件
	router.Use(
		// 恢复中间件 - 处理 panic
		gin.Recovery(),
		// 日志中间件
		gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[VIP-PAY] %v | %3d | %13v | %15s | %-7s %s\n%s",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
				param.ErrorMessage,
			)
		}),
		// CORS 中间件
		cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
	)

	return router
}

// registerRoutes 注册路由
func registerRoutes(router *gin.Engine) {
	// 创建支付服务
	paymentService := createPaymentService()

	// 创建 Gin 支付处理器
	paymentHandler := pay.NewGinPaymentHandler(paymentService)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "vip-payment",
			"version":   version.GetVersion(),
		})
	})

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 支付相关路由
		payment := api.Group("/payment")
		{
			// 订单管理
			payment.POST("/orders", paymentHandler.CreateOrder)
			payment.GET("/orders", paymentHandler.GetUserOrders)

			// 支付状态查询
			payment.POST("/query", paymentHandler.QueryPayment)

			// 支付回调
			payment.POST("/callback", paymentHandler.HandlePaymentCallback)
		}

		// VIP 会员相关路由
		vip := api.Group("/vip")
		{
			vip.GET("/info", paymentHandler.GetUserVIPInfo)
		}

		// 订阅管理路由
		subscription := api.Group("/subscription")
		{
			subscription.POST("/cancel", paymentHandler.CancelSubscription)
		}
	}

	// 第三方支付回调路由
	router.POST("/callback/alipay", paymentHandler.HandlePaymentCallback)
	router.POST("/callback/wechat", paymentHandler.HandlePaymentCallback)
	router.POST("/callback/stripe", paymentHandler.HandlePaymentCallback)
}

// createPaymentService 创建支付服务
func createPaymentService() paypkg.PaymentService {
	// 创建支付配置
	paymentConfig := &paypkg.PaymentConfig{
		DefaultCurrency:   "CNY",
		ReturnURL:         "https://your-domain.com/payment/return",
		NotifyURL:         "https://your-domain.com/api/v1/payment/callback",
		OrderExpireTime:   30, // 30分钟
		PaymentExpireTime: 15, // 15分钟
		MaxRetryCount:     3,
		EnableTestMode:    config.GlobalConfig.LogLevel == "debug",
		EnableRiskCheck:   true,
		RiskThreshold:     0.8,
	}

	// 创建支付服务实现
	paymentServiceImpl, err := paypkg.NewPaymentService(paymentConfig)
	if err != nil {
		logrus.Fatal("Failed to create payment service:", err)
	}

	return paymentServiceImpl
}

// registerPaymentProviders 注册支付提供商
func registerPaymentProviders(service paypkg.PaymentService) {
	// 注意：支付提供商已经在 NewPaymentService 中初始化
	// 这里只需要确保配置正确即可
	logrus.Info("Payment providers initialized in payment service")
}

// gracefulShutdown 优雅关闭
func gracefulShutdown(server *http.Server) {
	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	logrus.Info("Shutting down VIP payment server...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatal("Server forced to shutdown:", err)
	}

	logrus.Info("VIP payment server exited")
}
