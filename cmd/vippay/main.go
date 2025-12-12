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
	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	paypkg "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	pay "github.com/grapestree/fgrapery/grapery/internal/transport/pay"
	paymiddleware "github.com/grapestree/fgrapery/grapery/internal/transport/pay/middleware"
	"github.com/grapestree/fgrapery/grapery/internal/version"
	"github.com/sirupsen/logrus"
)

var printVersion = flag.Bool("version", false, "app build version")
var configPath = flag.String("config", "vippay.json", "config file")

func main() {
	flag.Parse()
	if *printVersion {
		version.PrintFullVersionInfo()
		return
	}

	// 加载配置（简化版，不依赖GlobalConfig）
	logrus.Info("Starting VIP payment service...")

	// 初始化数据库
	err := initializeServices()
	if err != nil {
		logrus.Fatal("initialize services failed : ", err)
	}

	// 创建 Gin 引擎
	router := createGinEngine()

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

	// 启动服务器
	go func() {
		logrus.Infof("Starting VIP payment server on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("start server failed: %v", err)
		}
	}()

	// 优雅关闭
	gracefulShutdown(server)
}

// initializeServices 初始化服务
func initializeServices() error {
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
		logrus.Fatal("init vippay database failed : ", err)
		return err
	}

	logrus.Info("init vippay database success")
	return nil
}

// createGinEngine 创建 Gin 引擎
func createGinEngine() *gin.Engine {
	// 设置 Gin 模式
	logLevel := getLogLevel()
	if logLevel == "debug" {
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
	// 获取配置的域名，如果未配置则使用默认值
	domain := getVipPayDomain()
	if domain == "" {
		domain = "https://www.grapery.xyz"
	}

	// 创建 IAP 配置
	iapConfig := createIAPConfig()

	// 创建 IAP 服务工厂
	iapServiceFactory := paypkg.NewIAPServiceFactory(iapConfig)

	// 创建复合 IAP 服务
	iapService := iapServiceFactory.CreateCompositeService()

	// 创建产品服务
	productService := paypkg.NewIAPProductServiceFromIAPConfig(iapConfig)

	// 创建 IAP 处理器，传入产品服务
	iapHandler := pay.NewIAPHandler(iapService, productService)

	// 创建 Apple OAuth2 处理器
	appleOAuthHandler := pay.NewAppleOAuthHandler()

	// 创建 Google OAuth2 处理器
	googleOAuthHandler := pay.NewGoogleOAuthHandler()

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
		}

		// Google OAuth2 相关路由
		googleOAuth := api.Group("/google-oauth")
		{
			// 公开接口 - 无需鉴权
			googleOAuth.POST("/signin", googleOAuthHandler.HandleGoogleSignIn)
			googleOAuth.GET("/status", googleOAuthHandler.HandleGoogleSignInStatus)
			googleOAuth.GET("/config", googleOAuthHandler.GetGoogleOAuthConfig)
		}

		// VIP 会员相关路由 - 需要鉴权
		vip := api.Group("/vip")
		vip.Use(paymiddleware.AuthMiddleware())
		{
			vip.GET("/info", func(c *gin.Context) {
				userID := paymiddleware.GetUserIDFromContext(c)
				// 简单的VIP信息响应
				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id":      userID,
						"is_vip":       false,
						"level":        0,
						"status":       0,
						"auto_renew":   false,
						"quota_used":   0,
						"quota_limit":  0,
						"max_roles":    2,
						"max_contexts": 5,
					},
				})
			})
			vip.GET("/check", func(c *gin.Context) {
				userID := paymiddleware.GetUserIDFromContext(c)
				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id": userID,
						"is_vip":  false,
					},
				})
			})
			vip.GET("/quota", func(c *gin.Context) {
				userID := paymiddleware.GetUserIDFromContext(c)
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
			})
			vip.GET("/max-roles", func(c *gin.Context) {
				userID := paymiddleware.GetUserIDFromContext(c)
				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id":   userID,
						"max_roles": 2,
					},
				})
			})
			vip.GET("/max-contexts", func(c *gin.Context) {
				userID := paymiddleware.GetUserIDFromContext(c)
				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": gin.H{
						"user_id":      userID,
						"max_contexts": 5,
					},
				})
			})
		}
	}
}

// createIAPConfig 创建IAP配置
func createIAPConfig() *paypkg.IAPConfig {
	// 默认配置
	config := &paypkg.IAPConfig{
		Apple: paypkg.AppleConfig{
			BundleID:       "com.yourapp.bundleid", // 需要配置实际的Bundle ID
			SandboxURL:     "https://api.storekit-sandbox.itunes.apple.com/inApps/v1/verifyReceipt",
			ProductionURL:  "https://api.storekit.itunes.apple.com/inApps/v1/verifyReceipt",
			IssuerID:       "YOUR_APPLE_ISSUER_ID",   // 需要配置实际的Issuer ID
			KeyID:          "YOUR_APPLE_KEY_ID",      // 需要配置实际的Key ID
			PrivateKey:     "YOUR_APPLE_PRIVATE_KEY", // 需要配置实际的Private Key
			APIBaseURL:     "https://api.appstoreconnect.apple.com",
			TimeoutSeconds: 30,
			MaxRetries:     3,
			RetryDelayMs:   1000,
			// Sandbox特定配置
			SandboxBundleID:   "com.yourapp.sandbox",      // 如果需要不同的Sandbox Bundle ID
			SandboxIssuerID:   "YOUR_SANDBOX_ISSUER_ID",   // 如果需要不同的Sandbox Issuer ID
			SandboxKeyID:      "YOUR_SANDBOX_KEY_ID",      // 如果需要不同的Sandbox Key ID
			SandboxPrivateKey: "YOUR_SANDBOX_PRIVATE_KEY", // 如果需要不同的Sandbox Private Key
		},
		Google: paypkg.GoogleConfig{
			PackageName:       "com.yourapp.packagename",         // 需要配置实际的Package Name
			ServiceAccountKey: "YOUR_GOOGLE_SERVICE_ACCOUNT_KEY", // 需要配置实际的Service Account Key
			APIBaseURL:        "https://androidpublisher.googleapis.com",
			TimeoutSeconds:    30,
			MaxRetries:        3,
			RetryDelayMs:      1000,
			// Sandbox特定配置
			SandboxPackageName:       "com.yourapp.sandbox",              // 如果需要不同的Sandbox Package Name
			SandboxServiceAccountKey: "YOUR_SANDBOX_SERVICE_ACCOUNT_KEY", // 如果需要不同的Sandbox Service Account Key
		},
	}

	// TODO: 从配置文件或环境变量加载实际的配置
	// 这里可以从 config.GlobalConfig 中读取IAP相关配置
	// 或者从环境变量中读取敏感信息如私钥等

	return config
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

// Helper functions for safe config access

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

func getLogLevel() string {
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		return level
	}
	return "info"
}
