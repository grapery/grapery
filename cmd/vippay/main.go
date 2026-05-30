package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"

	gomysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	paypkg "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	pay "github.com/grapestree/fgrapery/grapery/internal/transport/pay"
	paymiddleware "github.com/grapestree/fgrapery/grapery/internal/transport/pay/middleware"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"github.com/grapestree/fgrapery/grapery/internal/version"
)

// mainDB 是主服务数据库连接，用于读取 memberships 表（token_quota / token_used）
var mainDB *gorm.DB

var printVersion = flag.Bool("version", false, "app build version")
var configPath = flag.String("config", "vippay.json", "config file")

func main() {
	flag.Parse()

	// Load WECHAT_APP_ID / WECHAT_APP_SECRET and other vars from grapery/.env when running locally
	// (matches cmd/server behavior; Docker can still inject env via env_file).
	utils.LoadDotEnvFiles()

	if *printVersion {
		version.PrintFullVersionInfo()
		return
	}

	// Get host information
	hostname := utils.GetHostname()
	hostIP := utils.GetHostIP()

	// Load configuration
	var cfg config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.LoadFromFile(*configPath, "vippay")
		if err != nil {
			// If config file doesn't exist, fall back to environment variables
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "Config file %s not found, falling back to environment variables\n", *configPath)
				cfg = config.Load("vippay")
			} else {
				// For other errors (e.g., invalid YAML), exit
				fmt.Fprintf(os.Stderr, "Failed to load config file: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		fmt.Println("loading config from environment variables")
		cfg = config.Load("vippay")
	}

	// Initialize telemetry manager
	telemetryConfig := telemetry.TelemetryManagerConfig{
		LogLevel: cfg.LogLevel,
	}
	cfg.Telemetry.SLS.Enabled = false // Disable SLS by default for vippay service
	// Configure SLS if enabled
	if cfg.Telemetry.SLS.Enabled && cfg.Telemetry.SLS.Endpoint != "" {
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
				"appname": "vippay",
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

	logger.Info("starting grapery vip payment service",
		zap.String("env", cfg.Env),
		zap.String("addr", cfg.Addr()),
	)

	// JWT must match Grapery API: same HS256 secret as cmd/server (internal/auth.ParseToken).
	logger.Info("========== JWT Configuration ==========")

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		jwtSecret = strings.TrimSpace(cfg.JWT.Secret)
		if jwtSecret != "" {
			logger.Info("JWT secret loaded from VipPay config file (jwt.secret)")
		}
	} else {
		logger.Info("JWT secret loaded from JWT_SECRET environment variable")
	}
	if jwtSecret == "" {
		for _, p := range config.SharedJWTConfigCandidatePaths() {
			if sec, ok := config.ReadJWTSecretFromConfigFile(p); ok {
				jwtSecret = sec
				logger.Info("JWT secret loaded from shared Grapery API config file",
					zap.String("path", p),
					zap.String("hint_env", "GRAPH_API_CONFIG_PATH | GRAPERY_CONFIG_PATH | JWT_FALLBACK_CONFIG_PATH"))
				break
			}
		}
	}
	if jwtSecret == "" {
		logger.Error("JWT_SECRET is empty: VipPay authenticated routes will reject Bearer tokens from the app",
			zap.String("fix", "Set JWT_SECRET to the same value as the Grapery API, or set jwt.secret in vippay config, or point JWT_FALLBACK_CONFIG_PATH at the API config YAML that contains jwt.secret"))
	} else {
		logger.Info("JWT secret ready for VipPay (must match Grapery API)")
	}

	// 记录JWT Secret的长度和预览（安全性考虑不记录完整值）
	logger.Info("JWT Secret configuration",
		zap.Int("secret_length", len(jwtSecret)),
		zap.String("secret_preview", jwtSecret[:min(len(jwtSecret), 10)]+"..."),
		zap.Bool("from_env", os.Getenv("JWT_SECRET") != ""),
		zap.Bool("is_default", jwtSecret == "grapery-secret-key-change-in-production"),
	)

	auth.SetJWTSecret(jwtSecret)
	logger.Info("========== End JWT Configuration ==========")

	// 初始化数据库
	err = initializeServices(logger)
	if err != nil {
		logger.Fatal("initialize services failed", zap.Error(err))
	}

	// 创建 Gin 引擎
	router := createGinEngine(cfg, logger, telemetryManager)

	// 注册路由
	registerRoutes(router)

	// 创建 HTTP 服务器
	port := getVipPayPort()
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

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

	// 启动服务器
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("VIP pay server goroutine panic recovered", zap.Any("panic", r))
			}
		}()
		logger.Info("VIP payment server listening",
			zap.String("addr", ":"+port),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("start server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	logger.Info("shutdown signal received")

	// 优雅关闭
	gracefulShutdown(ctx, server, logger)
}

// initializeServices 初始化服务
func initializeServices(logger *zap.Logger) error {
	// 从环境变量获取数据库配置
	dbUser := os.Getenv("DB_USERNAME")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = ""
	}
	dbAddr := os.Getenv("DB_ADDRESS")
	if dbAddr == "" {
		dbAddr = "localhost"
	}
	dbName := os.Getenv("DB_DATABASE")
	if dbName == "" {
		dbName = "grapery"
	}

	// 初始化支付数据库
	err := paymodels.Init(dbUser, dbPass, dbAddr, dbName)
	if err != nil {
		logger.Fatal("init vippay database failed", zap.Error(err))
		return err
	}

	// iap_products 表未跑 Grapery API 迁移时可能不存在；对齐 VipPay 商品列表所用表结构
	if mErr := paymodels.DataBase().AutoMigrate(&paymodels.IAPProduct{}); mErr != nil {
		logger.Warn("iap_products AutoMigrate skipped or failed", zap.Error(mErr))
	}
	if seedErr := paymodels.SeedGraperyIAPAppleProductsIfMissing(logger); seedErr != nil {
		logger.Warn("seed Grapery IAP products failed", zap.Error(seedErr))
	}

	// 初始化 Web 支付表
	err = paymodels.AutoMigrateWebPayments()
	if err != nil {
		logger.Warn("failed to auto-migrate web payments table", zap.Error(err))
		// 不阻断启动，只记录警告
	} else {
		logger.Info("web payments table migrated successfully")
	}

	// 初始化主库连接（用于读取 memberships 的 token_quota / token_used）
	mainDSN := fmt.Sprintf("%s:%s@(%s:3306)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local",
		dbUser, dbPass, dbAddr, dbName)
	rawDB, rawErr := sql.Open("mysql", mainDSN)
	if rawErr != nil {
		logger.Warn("failed to open main DB connection, credit info will be unavailable", zap.Error(rawErr))
	} else {
		rawDB.SetMaxIdleConns(5)
		rawDB.SetMaxOpenConns(20)
		rawDB.SetConnMaxLifetime(5 * time.Minute)
		gormDB, gormErr := gorm.Open(gomysql.New(gomysql.Config{Conn: rawDB}), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		})
		if gormErr != nil {
			logger.Warn("failed to init main gorm DB, credit info will be unavailable", zap.Error(gormErr))
		} else {
			mainDB = gormDB
			logger.Info("main DB connection for credit info initialized successfully")
		}
	}

	logger.Info("init vippay database success")
	return nil
}

// createGinEngine 创建 Gin 引擎
func createGinEngine(cfg config.Config, logger *zap.Logger, telemetryManager *telemetry.TelemetryManager) *gin.Engine {
	// 设置 Gin 模式
	if cfg.Env == "development" || cfg.LogLevel == "debug" {
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
	)

	// Configure CORS
	allowOrigins := cfg.AllowOrigins
	isDevelopment := cfg.Env == "development" || os.Getenv("GIN_MODE") == "debug"

	if len(allowOrigins) == 0 {
		if isDevelopment {
			// Development: allow common local development origins
			allowOrigins = []string{
				"http://localhost:3000",
				"http://localhost:5173",
				"http://localhost:8080",
				"http://127.0.0.1:3000",
				"http://127.0.0.1:5173",
				"http://127.0.0.1:8080",
			}
		} else {
			// Production: no default - must be configured via CORS_ALLOW_ORIGINS or config
			logger.Warn("No CORS origins configured for production - CORS will be restrictive")
			allowOrigins = []string{} // Empty = no origins allowed
		}
	}

	corsConfig := cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", telemetry.CorrelationIDHeader},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// In development, use AllowOriginFunc for flexibility
	if isDevelopment && len(allowOrigins) > 0 {
		corsConfig.AllowOriginFunc = func(origin string) bool {
			// In development, allow any localhost/127.0.0.1 origin
			if strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:") {
				return true
			}
			for _, allowed := range allowOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		}
	}

	router.Use(cors.New(corsConfig))

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

	return router
}

// registerRoutes 注册路由
func registerRoutes(router *gin.Engine) {
	// 获取配置的域名，如果未配置则使用默认值
	domain := getVipPayDomain()
	if domain == "" {
		domain = "https://www.rankquantity.xyz"
	}

	// 创建 IAP 配置
	iapConfig := createIAPConfig()

	// 创建 IAP 服务工厂
	iapServiceFactory := paypkg.NewIAPServiceFactory(iapConfig)

	// 创建复合 IAP 服务
	iapService := iapServiceFactory.CreateCompositeService()

	// 创建产品服务
	productService := paypkg.NewIAPProductServiceFromIAPConfig(iapConfig)

	// 创建 IAP 处理器，传入产品服务及主库（用于购买后充值 tokens）
	iapHandler := pay.NewIAPHandler(iapService, productService, mainDB)

	// 兜底扫描：已过期但仍标记 Active/WillExpire 的 Apple 订阅
	go func() {
		ctx := context.Background()
		iapHandler.ExpireStaleAppleSubscriptions(ctx)
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			iapHandler.ExpireStaleAppleSubscriptions(ctx)
		}
	}()

	// 创建 OAuth Repository（用于持久化用户数据和第三方登录绑定）
	oauthRepo := paymodels.NewOAuthRepository()

	// 创建 Apple OAuth2 处理器（带 Repository 支持跨设备登录和账户关联）
	var appleOAuthHandler *pay.AppleOAuthHandler
	if oauthRepo != nil {
		appleOAuthHandler = pay.NewAppleOAuthHandlerWithRepo(oauthRepo)
	} else {
		appleOAuthHandler = pay.NewAppleOAuthHandler()
	}

	// 创建 Google OAuth2 处理器（带 Repository 支持跨设备登录和账户关联）
	var googleOAuthHandler *pay.GoogleOAuthHandler
	if oauthRepo != nil {
		googleOAuthHandler = pay.NewGoogleOAuthHandlerWithRepo(oauthRepo)
	} else {
		googleOAuthHandler = pay.NewGoogleOAuthHandler()
	}

	// 创建 WeChat OAuth2 处理器（网站应用扫码登录；需 WECHAT_APP_ID / WECHAT_APP_SECRET）
	var wechatOAuthHandler *pay.WeChatOAuthHandler
	if oauthRepo != nil {
		wechatOAuthHandler = pay.NewWeChatOAuthHandlerWithRepo(oauthRepo)
	} else {
		wechatOAuthHandler = pay.NewWeChatOAuthHandler()
	}

	// 创建徽章 Repository 和处理器
	badgeRepo := paymodels.NewBadgeRepository()
	var badgeHandler *pay.BadgeHandler
	if badgeRepo != nil {
		badgeHandler = pay.NewBadgeHandler(badgeRepo)
	}

	// API 路由组
	api := router.Group("/api/vippay")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"timestamp": time.Now().Format(time.RFC3339),
				"service":   "vip-payment",
				"version":   version.GetVersion(),
			})
		})

		// 版权信息接口 - 用于iOS app审核，无需鉴权
		api.GET("/copyright", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": gin.H{
					"company":          "Grapery Technology",
					"copyright":        "© 2025 Grapery Technology. All rights reserved.",
					"app_name":         "Grapery VIP Service",
					"version":          version.GetVersion(),
					"service_terms":    domain + "/api/vippay/terms-of-service",
					"privacy_policy":   domain + "/api/vippay/privacy-policy",
					"contact_email":    "support@grapery.xyz",
					"contact_phone":    "+86-18589045535",
					"address":          "上海市浦东新区临港新片区环湖西二路888号C楼",
					"business_license": "沪ICP备2025137210号",
					"description":      "用AI描述你想象中的故事，创造你的故事世界",
					"last_updated":     time.Now().Format("2006-01-02"),
				},
			})
		})

		// 服务条款接口 - 用于iOS app审核，无需鉴权
		api.GET("/terms-of-service", func(c *gin.Context) {
			termsHTML := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>服务条款 - rankquantity(未择)</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1, h2 { color: #333; }
        .last-updated { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <h1>rankquantity(未择) 服务条款</h1>
    <p class="last-updated">最后更新：` + time.Now().Format("2006年01月02日") + `</p>
    
    <h2>1. 服务简介</h2>
    <p>欢迎使用rankquantity(未择)服务。本服务由Rankquantity Technology提供，用AI帮助您描述想象中的故事，创造属于您的故事世界。</p>
    
    <h2>2. 用户协议</h2>
    <p>通过使用我们的服务，您同意遵守以下条款：</p>
    <ul>
        <li>您必须年满18岁或在监护人陪同下使用本服务</li>
        <li>您承诺提供真实、准确的个人信息</li>
        <li>您不得使用本服务进行任何非法或有害活动</li>
        <li>您不得上传包含违法、有害、威胁、辱骂等内容</li>
    </ul>
    
    <h2>3. VIP会员服务</h2>
    <p>我们提供VIP会员服务，包括：</p>
    <ul>
        <li>更多的故事创作配额</li>
        <li>优先的服务支持</li>
        <li>高级功能访问权限</li>
        <li>无广告体验</li>
    </ul>
    
    <h2>4. 付费与退款</h2>
    <p>关于付费服务和退款政策：</p>
    <ul>
        <li>所有价格以人民币计价</li>
        <li>订阅费用按照您选择的周期进行收费</li>
        <li>根据Apple App Store和Google Play的政策处理退款</li>
        <li>特殊情况下的退款将根据具体情况处理</li>
    </ul>
    
    <h2>5. 知识产权</h2>
    <p>您通过本服务创建的内容归您所有，但您授权我们使用这些内容来改进我们的服务。</p>
    
    <h2>6. 免责声明</h2>
    <p>我们努力提供稳定可靠的服务，但不保证服务不会中断或完全无错误。</p>
    
    <h2>7. 联系我们</h2>
    <p>如有任何问题，请联系我们：</p>
    <p>putaoshuyunying@grapery.xyz<br>
    电话：+86-18589045535<br>
    地址：上海市浦东新区临港新片区环湖西二路888号C楼</p>
</body>
</html>`
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, termsHTML)
		})

		// 隐私政策接口 - 用于iOS app审核，无需鉴权
		api.GET("/privacy-policy", func(c *gin.Context) {
			privacyHTML := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>隐私政策 - rankquantity(未择)</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1, h2 { color: #333; }
        .last-updated { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <h1>rankquantity(未择) 隐私政策</h1>
    <p class="last-updated">最后更新：` + time.Now().Format("2006年01月02日") + `</p>
    
    <h2>1. 信息收集</h2>
    <p>我们可能收集以下信息：</p>
    <ul>
        <li>账户信息：用户名、邮箱地址等注册信息</li>
        <li>设备信息：设备型号、操作系统版本、设备标识符</li>
        <li>使用数据：应用使用情况、功能偏好、操作记录</li>
        <li>内容数据：您创建的故事内容和相关数据</li>
    </ul>
    
    <h2>2. 信息使用</h2>
    <p>我们使用收集的信息用于：</p>
    <ul>
        <li>提供和改进我们的服务</li>
        <li>处理您的请求和交易</li>
        <li>发送服务相关通知</li>
        <li>分析服务使用情况以优化用户体验</li>
        <li>防止欺诈和滥用行为</li>
    </ul>
    
    <h2>3. 信息分享</h2>
    <p>我们不会向第三方出售您的个人信息。但在以下情况下可能分享信息：</p>
    <ul>
        <li>获得您的明确同意</li>
        <li>法律要求或政府要求</li>
        <li>保护我们的权利和财产</li>
        <li>与可信的服务提供商合作（如云存储服务）</li>
    </ul>
    
    <h2>4. 信息安全</h2>
    <p>我们采取合理的安全措施保护您的信息：</p>
    <ul>
        <li>数据传输加密</li>
        <li>访问权限控制</li>
        <li>定期安全审计</li>
        <li>员工隐私培训</li>
    </ul>
    
    <h2>5. Cookie和跟踪技术</h2>
    <p>我们使用Cookie和类似技术来改善服务体验，您可以通过浏览器设置管理Cookie。</p>
    
    <h2>6. 儿童隐私</h2>
    <p>我们的服务面向18岁以上用户。如果我们发现收集了13岁以下儿童的信息，将立即删除。</p>
    
    <h2>7. 您的权利</h2>
    <p>您有权：</p>
    <ul>
        <li>访问和更新您的个人信息</li>
        <li>删除您的账户和数据</li>
        <li>撤回同意</li>
        <li>导出您的数据</li>
    </ul>
    
    <h2>8. 政策更新</h2>
    <p>我们可能会更新此隐私政策。重大更改将通过应用内通知或邮件告知您。</p>
    
    <h2>9. 联系我们</h2>
    <p>关于隐私政策的任何问题，请联系：</p>
    <p>putaoshuyunying@grapery.xyz<br>
    电话：+86-18589045535<br>
    地址：上海市浦东新区临港新片区环湖西二路888号C楼</p>
</body>
</html>`
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, privacyHTML)
		})

		// Legal docs (JSON + markdown, localized)
		api.GET("/legal/terms-of-service", pay.GetTermsOfService)
		api.GET("/legal/privacy-policy", pay.GetPrivacyPolicy)

		// IAP 相关路由
		iap := api.Group("/iap")
		{
			// 公开接口 - 无需鉴权
			public := iap.Group("")
			{
				// Apple 通知处理（服务器间通信）
				public.POST("/apple/notification", iapHandler.HandleAppleNotification)
				// Google 通知处理（服务器间通信）
				public.POST("/google/notification", iapHandler.HandleGoogleNotification)

				// 产品查询接口
				public.GET("/products", iapHandler.GetProducts)
				public.GET("/products/:id", iapHandler.GetProductDetail)
				public.GET("/products/stats", iapHandler.GetProductStats)
			}

			// 需要鉴权的接口
			auth := iap.Group("")
			auth.Use(paymiddleware.AuthMiddleware())
			{
				// Apple IAP
				auth.POST("/apple/verify", iapHandler.VerifyAppleReceipt)
				auth.POST("/apple/subscription-status", iapHandler.GetAppleSubscriptionStatus)
				auth.POST("/apple/subscription-notice/ack", iapHandler.AckAppleSubscriptionNotice)
				auth.POST("/apple/acknowledge", iapHandler.AcknowledgePurchase)
				auth.POST("/apple/consume", iapHandler.ConsumePurchase)

				// Google IAP
				auth.POST("/google/verify", iapHandler.VerifyGooglePurchase)
				auth.POST("/google/subscription-status", iapHandler.GetGoogleSubscriptionStatus)
				auth.POST("/google/acknowledge", iapHandler.AcknowledgePurchase)
				auth.POST("/google/consume", iapHandler.ConsumePurchase)

				// 通用 IAP 接口
				auth.POST("/acknowledge", iapHandler.AcknowledgePurchase)
				auth.POST("/consume", iapHandler.ConsumePurchase)
				auth.POST("/sync", iapHandler.SyncSubscriptions)
			}
		}

		// Apple OAuth2 相关路由
		appleOAuth := api.Group("/apple-oauth")
		{
			// 公开接口 - 无需鉴权
			appleOAuth.POST("/signin", appleOAuthHandler.HandleAppleSignIn)
			appleOAuth.GET("/status", appleOAuthHandler.HandleAppleSignInStatus)
			appleOAuth.GET("/config", appleOAuthHandler.GetAppleOAuthConfig)

			// 需要鉴权的接口
			appleOAuthAuth := appleOAuth.Group("")
			appleOAuthAuth.Use(paymiddleware.AuthMiddleware())
			{
				appleOAuthAuth.POST("/link", appleOAuthHandler.HandleAppleLink)
				appleOAuthAuth.POST("/unlink", appleOAuthHandler.HandleAppleUnlink)
			}
		}

		// Google OAuth2 相关路由
		googleOAuth := api.Group("/google-oauth")
		{
			// 公开接口 - 无需鉴权
			googleOAuth.POST("/signin", googleOAuthHandler.HandleGoogleSignIn)
			googleOAuth.GET("/status", googleOAuthHandler.HandleGoogleSignInStatus)
			googleOAuth.GET("/config", googleOAuthHandler.GetGoogleOAuthConfig)

			// 需要鉴权的接口
			googleOAuthAuth := googleOAuth.Group("")
			googleOAuthAuth.Use(paymiddleware.AuthMiddleware())
			{
				googleOAuthAuth.POST("/link", googleOAuthHandler.HandleGoogleLink)
				googleOAuthAuth.POST("/unlink", googleOAuthHandler.HandleGoogleUnlink)
			}
		}

		// WeChat OAuth2 相关路由（开放平台网站应用 qrconnect + snsapi_login）
		wechatOAuth := api.Group("/wechat-oauth")
		{
			wechatOAuth.POST("/signin", wechatOAuthHandler.HandleWeChatSignIn)
			wechatOAuth.GET("/status", wechatOAuthHandler.HandleWeChatSignInStatus)
			wechatOAuth.GET("/config", wechatOAuthHandler.GetWeChatOAuthConfig)

			wechatOAuthAuth := wechatOAuth.Group("")
			wechatOAuthAuth.Use(paymiddleware.AuthMiddleware())
			{
				wechatOAuthAuth.POST("/link", wechatOAuthHandler.HandleWeChatLink)
				wechatOAuthAuth.POST("/unlink", wechatOAuthHandler.HandleWeChatUnlink)
			}
		}

		// VIP 会员相关路由 - 需要鉴权
		vip := api.Group("/vip")
		vip.Use(paymiddleware.AuthMiddleware())
		{
		vip.GET("/info", func(c *gin.Context) {
			userIDStr := paymiddleware.GetUserIDFromContext(c)
			userID := stringToInt64(userIDStr)
			ctx := c.Request.Context()

			// 从主库读取真实 token 余量（token_quota / token_used）
			tokenQuota, tokenUsed := getMainDBTokenInfo(ctx, userIDStr)
			creditLimit := ceilDiv(tokenQuota, common.CreditToTokenRatio)
			creditUsed := ceilDiv(tokenUsed, common.CreditToTokenRatio)

			// 查询用户活跃订阅
			subscription, err := paymodels.GetUserActiveSubscriptionByUserID(ctx, userID)
			if err != nil {
				// 没有活跃订阅，返回默认值
				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id":      userID,
						"is_vip":       false,
						"level":        0,
						"status":       0,
						"auto_renew":   false,
						"quota_used":   tokenUsed,
						"quota_limit":  tokenQuota,
						"credit_used":  creditUsed,
						"credit_limit": creditLimit,
						"max_roles":    2, // 免费用户默认值
						"max_contexts": 5, // 免费用户默认值
						"expires_at":   nil,
					},
				})
				return
			}

			// 计算VIP等级（根据订阅套餐）
			vipLevel := calculateVIPLevel(subscription.PackagePlanID)

			// 格式化过期时间
			var expiresAt *string
			if !subscription.EndTime.IsZero() {
				expiresAtStr := subscription.EndTime.Format(time.RFC3339)
				expiresAt = &expiresAtStr
			}

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": gin.H{
					"user_id":      userID,
					"is_vip":       subscription.IsActive(),
					"level":        vipLevel,
					"status":       int(subscription.Status),
					"auto_renew":   subscription.AutoRenew,
					"quota_used":   tokenUsed,
					"quota_limit":  tokenQuota,
					"credit_used":  creditUsed,
					"credit_limit": creditLimit,
					"max_roles":    subscription.MaxRoles,
					"max_contexts": subscription.MaxContexts,
					"expires_at":   expiresAt,
				},
			})
		})
			vip.GET("/check", func(c *gin.Context) {
				userIDStr := paymiddleware.GetUserIDFromContext(c)
				userID := stringToInt64(userIDStr)
				ctx := c.Request.Context()

				subscription, err := paymodels.GetUserActiveSubscriptionByUserID(ctx, userID)
				isVip := err == nil && subscription != nil && subscription.IsActive()

				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id": userID,
						"is_vip":  isVip,
					},
				})
			})
			vip.GET("/quota", func(c *gin.Context) {
				userIDStr := paymiddleware.GetUserIDFromContext(c)
				userID := stringToInt64(userIDStr)
				ctx := c.Request.Context()

				subscription, err := paymodels.GetUserActiveSubscriptionByUserID(ctx, userID)
				if err != nil {
					// 没有活跃订阅
					c.JSON(http.StatusOK, gin.H{
						"code": 0,
						"msg":  "success",
						"data": gin.H{
							"user_id":     userID,
							"quota_used":  0,
							"quota_limit": 0,
							"remaining":   0,
						},
					})
					return
				}

				remaining := subscription.GetRemainingQuota()

				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id":     userID,
						"quota_used":  subscription.QuotaUsed,
						"quota_limit": subscription.QuotaLimit,
						"remaining":   remaining,
					},
				})
			})
			vip.GET("/max-roles", func(c *gin.Context) {
				userIDStr := paymiddleware.GetUserIDFromContext(c)
				userID := stringToInt64(userIDStr)
				ctx := c.Request.Context()

				subscription, err := paymodels.GetUserActiveSubscriptionByUserID(ctx, userID)
				maxRoles := 2 // 默认值
				if err == nil && subscription != nil {
					maxRoles = subscription.MaxRoles
				}

				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id":   userID,
						"max_roles": maxRoles,
					},
				})
			})
			vip.GET("/max-contexts", func(c *gin.Context) {
				userIDStr := paymiddleware.GetUserIDFromContext(c)
				userID := stringToInt64(userIDStr)
				ctx := c.Request.Context()

				subscription, err := paymodels.GetUserActiveSubscriptionByUserID(ctx, userID)
				maxContexts := 5 // 默认值
				if err == nil && subscription != nil {
					maxContexts = subscription.MaxContexts
				}

				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id":      userID,
						"max_contexts": maxContexts,
					},
				})
			})
		}

		// 徽章相关路由
		if badgeHandler != nil {
			badges := api.Group("/badges")
			{
				// 公开接口 - 无需鉴权
				badges.GET("", badgeHandler.GetAllBadges)                           // 获取所有徽章定义
				badges.GET("/category/:category", badgeHandler.GetBadgesByCategory) // 根据类别获取徽章
				badges.GET("/user", badgeHandler.GetUserBadges)                     // 获取用户已获得的徽章（支持user_id参数）
				badges.GET("/pinned", badgeHandler.GetUserPinnedBadges)             // 获取用户置顶的徽章

				// 需要鉴权的接口
				authBadges := badges.Group("")
				authBadges.Use(paymiddleware.AuthMiddleware())
				{
					authBadges.GET("/profile", badgeHandler.GetUserBadgeProfile)   // 获取用户徽章档案
					authBadges.GET("/stats", badgeHandler.GetUserStats)            // 获取用户统计
					authBadges.GET("/progress", badgeHandler.GetBadgeProgress)     // 获取徽章进度
					authBadges.POST("/pin", badgeHandler.PinBadge)                 // 置顶徽章
					authBadges.POST("/unpin/:badge_id", badgeHandler.UnpinBadge)   // 取消置顶
					authBadges.POST("/mark-viewed", badgeHandler.MarkBadgesViewed) // 标记徽章已查看
					authBadges.POST("/check", badgeHandler.CheckAndAwardBadges)    // 检查并授予徽章
					authBadges.POST("/sync-stats", badgeHandler.SyncUserStats)     // 同步用户统计
				}
			}
		}

		// Token 用量统计相关路由 - 需要鉴权
		usage := api.Group("/usage")
		usage.Use(paymiddleware.AuthMiddleware())
		{
			// 创建 TokenUsageService 和 TokenUsageLogService
			logrusLogger := &logrus.Logger{
				Out:       os.Stderr,
				Formatter: new(logrus.TextFormatter),
				Hooks:     make(logrus.LevelHooks),
				Level:     logrus.InfoLevel,
			}
			tokenUsageService := paypkg.NewTokenUsageService(logrusLogger)
			tokenUsageLogService := paypkg.NewTokenUsageLogService(logrusLogger)
			usageHandler := paypkg.NewUsageLimitHandler(tokenUsageService, logrusLogger)
			usageLogHandler := pay.NewTokenUsageLogHandler(tokenUsageLogService, logrusLogger)

			// 获取用户总用量统计（支持 daily/weekly/monthly/yearly 周期）
			usage.GET("/stats", func(c *gin.Context) {
				userIDStr := paymiddleware.GetUserIDFromContext(c)
				userID := stringToInt64(userIDStr)
				c.Set("user_id", userID)
				usageHandler.GetUsageStats(c)
			})

			// 获取用户各类型用量统计
			usage.GET("/by-type", func(c *gin.Context) {
				userIDStr := paymiddleware.GetUserIDFromContext(c)
				userID := stringToInt64(userIDStr)
				c.Set("user_id", userID)
				usageHandler.GetUsageByType(c)
			})

			// 检查特定类型的用量限制
			usage.GET("/limit/:type", func(c *gin.Context) {
				userIDStr := paymiddleware.GetUserIDFromContext(c)
				userID := stringToInt64(userIDStr)
				c.Set("user_id", userID)
				usageHandler.CheckUsageLimit(c)
			})

			// Token 用量日志相关路由
			logs := usage.Group("/logs")
			{
				// 查询日志列表
				logs.GET("", func(c *gin.Context) {
					userIDStr := paymiddleware.GetUserIDFromContext(c)
					userID := stringToInt64(userIDStr)
					c.Set("user_id", userID)
					usageLogHandler.GetLogs(c)
				})

				// 获取汇总统计
				logs.GET("/summary", func(c *gin.Context) {
					userIDStr := paymiddleware.GetUserIDFromContext(c)
					userID := stringToInt64(userIDStr)
					c.Set("user_id", userID)
					usageLogHandler.GetSummary(c)
				})

				// 按实体类型汇总
				logs.GET("/summary/by-type", func(c *gin.Context) {
					userIDStr := paymiddleware.GetUserIDFromContext(c)
					userID := stringToInt64(userIDStr)
					c.Set("user_id", userID)
					usageLogHandler.GetSummaryByEntityType(c)
				})

				// 导出日志（CSV/JSON）
				logs.GET("/export", func(c *gin.Context) {
					userIDStr := paymiddleware.GetUserIDFromContext(c)
					userID := stringToInt64(userIDStr)
					c.Set("user_id", userID)
					usageLogHandler.ExportLogs(c)
				})

				// 按业务实体查询日志
				logs.GET("/by-entity/:entity_type/:entity_id", usageLogHandler.GetLogsByEntity)

				// 获取计费汇总
				logs.GET("/billing", func(c *gin.Context) {
					userIDStr := paymiddleware.GetUserIDFromContext(c)
					userID := stringToInt64(userIDStr)
					c.Set("user_id", userID)
					usageLogHandler.GetBilling(c)
				})

				// 标记为已计费
				logs.POST("/mark-billed", usageLogHandler.MarkAsBilled)
			}
		}

		// Web 支付相关路由 - 增量添加
		web := api.Group("/web")
		{
			// 创建日志记录器
			webLogger := &logrus.Logger{
				Out:       os.Stderr,
				Formatter: new(logrus.TextFormatter),
				Hooks:     make(logrus.LevelHooks),
				Level:     logrus.InfoLevel,
			}

			// 创建 Web 支付配置
			webPaymentConfig := &paypkg.WebPaymentConfig{
				StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
				StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
				StripePublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
				AlipayAppID:          os.Getenv("ALIPAY_APP_ID"),
				AlipayPrivateKey:     os.Getenv("ALIPAY_PRIVATE_KEY"),
				AlipayPublicKey:      os.Getenv("ALIPAY_PUBLIC_KEY"),
			}

			// 创建 Web 支付服务
			webPaymentService := paypkg.NewWebPaymentService(webLogger, webPaymentConfig)

			// 创建 Web 支付处理器
			webPaymentHandler := pay.NewWebPaymentHandler(webPaymentService, webLogger)

			// 需要鉴权的接口
			authWeb := web.Group("")
			authWeb.Use(paymiddleware.AuthMiddleware())
			{
				// 支付管理
				authWeb.POST("/payments", webPaymentHandler.CreatePayment)
				authWeb.GET("/payments/:id", webPaymentHandler.GetPayment)
				authWeb.GET("/payments/user/:userId", webPaymentHandler.GetUserPayments)
			}

			// Webhook 接口（无需鉴权，由支付服务商直接调用）
			web.POST("/webhooks/stripe", webPaymentHandler.HandleStripeWebhook)
			web.POST("/webhooks/alipay", webPaymentHandler.HandleAlipayWebhook)
		}
	}
}

// normalizeApplePrivateKeyPEM 支持 .env 内 JSON 转义换行（docker compose 单行配置）。
func normalizeApplePrivateKeyPEM(pem string) string {
	pem = strings.TrimSpace(pem)
	if strings.Contains(pem, `\n`) {
		pem = strings.ReplaceAll(pem, `\n`, "\n")
	}
	return pem
}

// createIAPConfig 创建IAP配置（从环境变量加载）
func createIAPConfig() *paypkg.IAPConfig {
	// Apple 配置
	appleBundleID := getEnvWithDefault("APPLE_BUNDLE_ID", "com.rankquantity.voyager")
	appleIssuerID := os.Getenv("APPLE_ISSUER_ID")
	appleKeyID := os.Getenv("APPLE_KEY_ID")
	applePrivateKey := normalizeApplePrivateKeyPEM(os.Getenv("APPLE_PRIVATE_KEY"))
	// Apple Sandbox 配置（如不设置，使用生产配置）
	appleSandboxBundleID := getEnvWithDefault("APPLE_SANDBOX_BUNDLE_ID", appleBundleID)
	appleSandboxIssuerID := getEnvWithDefault("APPLE_SANDBOX_ISSUER_ID", appleIssuerID)
	appleSandboxKeyID := getEnvWithDefault("APPLE_SANDBOX_KEY_ID", appleKeyID)
	appleSandboxPrivateKey := getEnvWithDefault("APPLE_SANDBOX_PRIVATE_KEY", applePrivateKey)

	// Google 配置
	googlePackageName := getEnvWithDefault("GOOGLE_PACKAGE_NAME", "com.rankquantity.pioneer")
	googleServiceAccountKey := os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")

	// Google Sandbox 配置（如不设置，使用生产配置）
	googleSandboxPackageName := getEnvWithDefault("GOOGLE_SANDBOX_PACKAGE_NAME", googlePackageName)
	googleSandboxServiceAccountKey := getEnvWithDefault("GOOGLE_SANDBOX_SERVICE_ACCOUNT_KEY", googleServiceAccountKey)

	config := &paypkg.IAPConfig{
		Apple: paypkg.AppleConfig{
			BundleID:       appleBundleID,
			IssuerID:       appleIssuerID,
			KeyID:          appleKeyID,
			PrivateKey:     applePrivateKey,
			APIBaseURL:     "https://api.appstoreconnect.apple.com",
			TimeoutSeconds: 30,
			MaxRetries:     3,
			RetryDelayMs:   1000,
			// Sandbox 特定配置
			SandboxBundleID:   appleSandboxBundleID,
			SandboxIssuerID:   appleSandboxIssuerID,
			SandboxKeyID:      appleSandboxKeyID,
			SandboxPrivateKey: appleSandboxPrivateKey,
		},
		Google: paypkg.GoogleConfig{
			PackageName:       googlePackageName,
			ServiceAccountKey: googleServiceAccountKey,
			APIBaseURL:        "https://androidpublisher.googleapis.com",
			TimeoutSeconds:    30,
			MaxRetries:        3,
			RetryDelayMs:      1000,
			// Sandbox 特定配置
			SandboxPackageName:       googleSandboxPackageName,
			SandboxServiceAccountKey: googleSandboxServiceAccountKey,
		},
	}

	return config
}

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// gracefulShutdown 优雅关闭
func gracefulShutdown(ctx context.Context, server *http.Server, logger *zap.Logger) {
	logger.Info("Shutting down VIP payment server...")

	// 设置关闭超时
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}

	logger.Info("VIP payment server exited")
}

// Helper functions for safe config access

// stringToInt64 将字符串用户ID转换为int64
// 使用 FNV-1a 哈希将字符串UUID转换为稳定的数值ID
func stringToInt64(s string) int64 {
	if s == "" {
		return 0
	}
	h := fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}

// calculateVIPLevel 根据套餐计划ID计算VIP等级
// 这里可以根据实际的套餐计划配置来映射等级
func calculateVIPLevel(packagePlanID uint) int {
	// 简单的映射逻辑，实际应该从数据库查询套餐计划配置
	// 这里可以根据 packagePlanID 映射到不同的VIP等级
	// 例如：1=免费, 2=基础, 3=高级, 4=专业
	if packagePlanID == 0 {
		return 0 // 免费用户
	}
	// 可以根据实际业务逻辑调整
	return int(packagePlanID)
}

func getVipPayPort() string {
	// 尝试从环境变量获取
	if port := os.Getenv("VIPPAY_PORT"); port != "" {
		return port
	}
	// 默认端口
	return "8081"
}

func getVipPayDomain() string {
	// 尝试从环境变量获取
	if domain := os.Getenv("VIPPAY_DOMAIN"); domain != "" {
		return domain
	}
	return "https://www.grapery.xyz"
}

// ceilDiv 向上取整除法：ceil(a/b)
func ceilDiv(a, b int) int {
	if b <= 0 {
		return 0
	}
	return (a + b - 1) / b
}

// mainDBMembership 用于从主库读取 membership 行
type mainDBMembership struct {
	TokenQuota int `gorm:"column:token_quota"`
	TokenUsed  int `gorm:"column:token_used"`
}

// getMainDBTokenInfo 从主库 memberships 表读取 token_quota 和 token_used。
// 如果 mainDB 未配置或查询失败，返回 (0, 0)，调用方应降级处理。
func getMainDBTokenInfo(ctx context.Context, userIDStr string) (quota int, used int) {
	if mainDB == nil || userIDStr == "" {
		return 0, 0
	}
	var m mainDBMembership
	if err := mainDB.WithContext(ctx).Table("memberships").
		Select("token_quota, token_used").
		Where("user_id = ?", userIDStr).
		First(&m).Error; err != nil {
		return 0, 0
	}
	return m.TokenQuota, m.TokenUsed
}
