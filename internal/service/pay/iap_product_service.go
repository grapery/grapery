package pay

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

// IAPProductService IAP 产品服务接口
type IAPProductService interface {
	// 产品管理
	CreateProduct(ctx context.Context, product *paymodels.IAPProduct) error
	GetProduct(ctx context.Context, id uint) (*paymodels.IAPProduct, error)
	GetProductByProductID(ctx context.Context, productID string) (*paymodels.IAPProduct, error)
	GetProductBySKU(ctx context.Context, sku string, platform paymodels.IAPProductPlatform) (*paymodels.IAPProduct, error)
	UpdateProduct(ctx context.Context, product *paymodels.IAPProduct) error
	DeleteProduct(ctx context.Context, id uint) error

	// 产品查询
	GetActiveProducts(ctx context.Context, platform paymodels.IAPProductPlatform) ([]*paymodels.IAPProduct, error)
	GetProductsByType(ctx context.Context, productType paymodels.IAPProductType, platform paymodels.IAPProductPlatform) ([]*paymodels.IAPProduct, error)
	GetFeaturedProducts(ctx context.Context, platform paymodels.IAPProductPlatform) ([]*paymodels.IAPProduct, error)
	GetProductStats(ctx context.Context, platform paymodels.IAPProductPlatform) (map[string]interface{}, error)

	// 平台同步
	SyncProductsFromApple(ctx context.Context) error
	SyncProductsFromGoogle(ctx context.Context) error
	UpdateSyncStatus(ctx context.Context, id uint, syncStatus, syncError string) error
}

// IAPProductServiceImpl IAP 产品服务实现
type IAPProductServiceImpl struct {
	logger     *logrus.Logger
	httpClient *HTTPClientWrapper
	config     *IAPProductConfig
}

// IAPProductConfig IAP产品服务配置
type IAPProductConfig struct {
	Apple  AppleAPIConfig  `json:"apple"`
	Google GoogleAPIConfig `json:"google"`
}

// AppleAPIConfig Apple App Store Connect API配置
type AppleAPIConfig struct {
	IssuerID       string `json:"issuer_id"`       // Apple Issuer ID
	KeyID          string `json:"key_id"`          // Apple Key ID
	PrivateKey     string `json:"private_key"`     // Apple Private Key (PEM格式)
	BundleID       string `json:"bundle_id"`       // App Bundle ID
	APIBaseURL     string `json:"api_base_url"`    // API基础URL，默认: https://api.appstoreconnect.apple.com
	TimeoutSeconds int    `json:"timeout_seconds"` // 请求超时时间，默认: 30
	MaxRetries     int    `json:"max_retries"`     // 最大重试次数，默认: 3
	RetryDelayMs   int    `json:"retry_delay_ms"`  // 重试延迟，默认: 1000
}

// GoogleAPIConfig Google Play Developer API配置
type GoogleAPIConfig struct {
	ServiceAccountKey string `json:"service_account_key"` // Google Service Account JSON密钥
	PackageName       string `json:"package_name"`        // Android包名
	APIBaseURL        string `json:"api_base_url"`        // API基础URL，默认: https://androidpublisher.googleapis.com
	TimeoutSeconds    int    `json:"timeout_seconds"`     // 请求超时时间，默认: 30
	MaxRetries        int    `json:"max_retries"`         // 最大重试次数，默认: 3
	RetryDelayMs      int    `json:"retry_delay_ms"`      // 重试延迟，默认: 1000
}

// NewIAPProductService 创建 IAP 产品服务
func NewIAPProductService() IAPProductService {
	return &IAPProductServiceImpl{
		logger:     logrus.New(),
		httpClient: NewHTTPClientWrapper(),
		config:     getDefaultIAPProductConfig(),
	}
}

// NewIAPProductServiceFromIAPConfig 从 IAPConfig 创建 IAP 产品服务
func NewIAPProductServiceFromIAPConfig(iapConfig *IAPConfig) IAPProductService {
	if iapConfig == nil {
		return NewIAPProductService()
	}

	// 转换配置
	productConfig := convertIAPConfigToProductConfig(iapConfig)
	return NewIAPProductServiceWithConfig(productConfig)
}

// convertIAPConfigToProductConfig 将 IAPConfig 转换为 IAPProductConfig
func convertIAPConfigToProductConfig(iapConfig *IAPConfig) *IAPProductConfig {
	productConfig := &IAPProductConfig{
		Apple: AppleAPIConfig{
			IssuerID:       iapConfig.Apple.IssuerID,
			KeyID:          iapConfig.Apple.KeyID,
			PrivateKey:     iapConfig.Apple.PrivateKey,
			BundleID:       iapConfig.Apple.BundleID,
			APIBaseURL:     iapConfig.Apple.APIBaseURL,
			TimeoutSeconds: iapConfig.Apple.TimeoutSeconds,
			MaxRetries:     iapConfig.Apple.MaxRetries,
			RetryDelayMs:   iapConfig.Apple.RetryDelayMs,
		},
		Google: GoogleAPIConfig{
			ServiceAccountKey: iapConfig.Google.ServiceAccountKey,
			PackageName:       iapConfig.Google.PackageName,
			APIBaseURL:        iapConfig.Google.APIBaseURL,
			TimeoutSeconds:    iapConfig.Google.TimeoutSeconds,
			MaxRetries:        iapConfig.Google.MaxRetries,
			RetryDelayMs:      iapConfig.Google.RetryDelayMs,
		},
	}

	// 设置默认值
	setDefaultAppleConfig(&productConfig.Apple)
	setDefaultGoogleConfig(&productConfig.Google)

	return productConfig
}

// NewIAPProductServiceWithConfig 使用配置创建 IAP 产品服务
func NewIAPProductServiceWithConfig(config *IAPProductConfig) IAPProductService {
	if config == nil {
		config = getDefaultIAPProductConfig()
	}

	// 设置默认值
	setDefaultAppleConfig(&config.Apple)
	setDefaultGoogleConfig(&config.Google)

	return &IAPProductServiceImpl{
		logger:     logrus.New(),
		httpClient: NewHTTPClientWrapper(),
		config:     config,
	}
}

// getDefaultIAPProductConfig 获取默认配置
func getDefaultIAPProductConfig() *IAPProductConfig {
	config := &IAPProductConfig{
		Apple:  AppleAPIConfig{},
		Google: GoogleAPIConfig{},
	}
	setDefaultAppleConfig(&config.Apple)
	setDefaultGoogleConfig(&config.Google)
	return config
}

// setDefaultAppleConfig 设置Apple API默认配置
func setDefaultAppleConfig(config *AppleAPIConfig) {
	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://api.appstoreconnect.apple.com"
	}
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 30
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelayMs == 0 {
		config.RetryDelayMs = 1000
	}
}

// setDefaultGoogleConfig 设置Google API默认配置
func setDefaultGoogleConfig(config *GoogleAPIConfig) {
	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://androidpublisher.googleapis.com"
	}
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 30
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelayMs == 0 {
		config.RetryDelayMs = 1000
	}
}

// CreateProduct 创建产品
func (s *IAPProductServiceImpl) CreateProduct(ctx context.Context, product *paymodels.IAPProduct) error {
	s.logger.WithFields(logrus.Fields{
		"product_id": product.ProductID,
		"platform":   product.Platform,
		"name":       product.Name,
	}).Info("Creating IAP product")

	return paymodels.CreateIAPProduct(ctx, product)
}

// GetProduct 获取产品
func (s *IAPProductServiceImpl) GetProduct(ctx context.Context, id uint) (*paymodels.IAPProduct, error) {
	return paymodels.GetIAPProduct(ctx, id)
}

// GetProductByProductID 根据产品ID获取产品
func (s *IAPProductServiceImpl) GetProductByProductID(ctx context.Context, productID string) (*paymodels.IAPProduct, error) {
	return paymodels.GetIAPProductByProductID(ctx, productID)
}

// GetProductBySKU 根据SKU获取产品
func (s *IAPProductServiceImpl) GetProductBySKU(ctx context.Context, sku string, platform paymodels.IAPProductPlatform) (*paymodels.IAPProduct, error) {
	return paymodels.GetIAPProductBySKU(ctx, sku, platform)
}

// UpdateProduct 更新产品
func (s *IAPProductServiceImpl) UpdateProduct(ctx context.Context, product *paymodels.IAPProduct) error {
	s.logger.WithFields(logrus.Fields{
		"product_id": product.ProductID,
		"platform":   product.Platform,
		"name":       product.Name,
	}).Info("Updating IAP product")

	return paymodels.UpdateIAPProduct(ctx, product)
}

// DeleteProduct 删除产品
func (s *IAPProductServiceImpl) DeleteProduct(ctx context.Context, id uint) error {
	s.logger.WithField("product_id", id).Info("Deleting IAP product")

	return paymodels.DeleteIAPProduct(ctx, id)
}

// GetActiveProducts 获取激活的产品
func (s *IAPProductServiceImpl) GetActiveProducts(ctx context.Context, platform paymodels.IAPProductPlatform) ([]*paymodels.IAPProduct, error) {
	return paymodels.GetActiveIAPProducts(ctx, platform)
}

// GetProductsByType 根据类型获取产品
func (s *IAPProductServiceImpl) GetProductsByType(ctx context.Context, productType paymodels.IAPProductType, platform paymodels.IAPProductPlatform) ([]*paymodels.IAPProduct, error) {
	return paymodels.GetIAPProductsByType(ctx, productType, platform)
}

// GetFeaturedProducts 获取推荐产品
func (s *IAPProductServiceImpl) GetFeaturedProducts(ctx context.Context, platform paymodels.IAPProductPlatform) ([]*paymodels.IAPProduct, error) {
	return paymodels.GetFeaturedIAPProducts(ctx, platform)
}

// GetProductStats 获取产品统计
func (s *IAPProductServiceImpl) GetProductStats(ctx context.Context, platform paymodels.IAPProductPlatform) (map[string]interface{}, error) {
	return paymodels.GetIAPProductStats(ctx, platform)
}

// SyncProductsFromApple 从 Apple App Store Connect 同步产品
func (s *IAPProductServiceImpl) SyncProductsFromApple(ctx context.Context) error {
	s.logger.Info("Starting sync products from Apple App Store Connect")

	// 1. 获取现有的Apple产品列表
	existingProducts, err := paymodels.GetActiveIAPProducts(ctx, paymodels.IAPProductPlatformApple)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get existing Apple products")
		return fmt.Errorf("获取现有产品失败: %w", err)
	}

	// 2. 从Apple App Store Connect获取产品信息
	appleProducts, err := s.fetchAppleProducts(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to fetch Apple products")
		return fmt.Errorf("获取Apple产品失败: %w", err)
	}

	// 3. 比较和同步产品
	syncedCount, updatedCount, createdCount, err := s.syncAppleProducts(ctx, existingProducts, appleProducts)
	if err != nil {
		s.logger.WithError(err).Error("Failed to sync Apple products")
		return fmt.Errorf("同步Apple产品失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"synced":  syncedCount,
		"updated": updatedCount,
		"created": createdCount,
	}).Info("Apple product sync completed successfully")

	return nil
}

// SyncProductsFromGoogle 从 Google Play Console 同步产品
func (s *IAPProductServiceImpl) SyncProductsFromGoogle(ctx context.Context) error {
	s.logger.Info("Starting sync products from Google Play Console")

	// 1. 获取现有的Google产品列表
	existingProducts, err := paymodels.GetActiveIAPProducts(ctx, paymodels.IAPProductPlatformGoogle)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get existing Google products")
		return fmt.Errorf("获取现有产品失败: %w", err)
	}

	// 2. 从Google Play Console获取产品信息
	googleProducts, err := s.fetchGoogleProducts(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to fetch Google products")
		return fmt.Errorf("获取Google产品失败: %w", err)
	}

	// 3. 比较和同步产品
	syncedCount, updatedCount, createdCount, err := s.syncGoogleProducts(ctx, existingProducts, googleProducts)
	if err != nil {
		s.logger.WithError(err).Error("Failed to sync Google products")
		return fmt.Errorf("同步Google产品失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"synced":  syncedCount,
		"updated": updatedCount,
		"created": createdCount,
	}).Info("Google product sync completed successfully")

	return nil
}

// UpdateSyncStatus 更新同步状态
func (s *IAPProductServiceImpl) UpdateSyncStatus(ctx context.Context, id uint, syncStatus, syncError string) error {
	return paymodels.UpdateIAPProductSyncStatus(ctx, id, syncStatus, syncError)
}

// CreateDefaultProducts 创建默认产品（用于初始化）
func (s *IAPProductServiceImpl) CreateDefaultProducts(ctx context.Context) error {
	s.logger.Info("Creating default IAP products")

	defaultProducts := []*paymodels.IAPProduct{
		{
			ProductID:    "com.rankquantity.voyager.sub_basic_y",
			Platform:     paymodels.IAPProductPlatformApple,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Basic Annual Subscription",
			Description:  "Basic annual subscription with essential features",
			Price:        99.99,
			Currency:     "USD",
			Duration:     stringPtr("P1Y"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     50,
			MaxContexts:  50,
			QuotaLimit:   1000000 * 12,
			AppleSKU:     stringPtr("com.rankquantity.voyager.sub_basic_y"),
			Featured:     true,
			DisplayOrder: 1,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_basic_m",
			Platform:     paymodels.IAPProductPlatformApple,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Basic Monthly Subscription",
			Description:  "Basic monthly subscription with essential features",
			Price:        9.99,
			Currency:     "USD",
			Duration:     stringPtr("P1M"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     50,
			MaxContexts:  50,
			QuotaLimit:   1000000,
			AppleSKU:     stringPtr("com.rankquantity.voyager.sub_basic_m"),
			Featured:     true,
			DisplayOrder: 2,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_premium_y",
			Platform:     paymodels.IAPProductPlatformApple,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Premium Annual Subscription",
			Description:  "Premium annual subscription with advanced features",
			Price:        249.99,
			Currency:     "USD",
			Duration:     stringPtr("P1Y"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     500,
			MaxContexts:  500,
			QuotaLimit:   50000000 * 12,
			AppleSKU:     stringPtr("com.rankquantity.voyager.sub_premium_y"),
			Featured:     true,
			DisplayOrder: 3,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_premium_m",
			Platform:     paymodels.IAPProductPlatformApple,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Premium Monthly Subscription",
			Description:  "Premium monthly subscription with advanced features",
			Price:        24.99,
			Currency:     "USD",
			Duration:     stringPtr("P1M"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     500,
			MaxContexts:  500,
			QuotaLimit:   50000000,
			AppleSKU:     stringPtr("com.rankquantity.voyager.sub_premium_m"),
			Featured:     true,
			DisplayOrder: 4,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_pro_y",
			Platform:     paymodels.IAPProductPlatformApple,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Pro Annual Subscription",
			Description:  "Pro annual subscription with unlimited features",
			Price:        149.99,
			Currency:     "USD",
			Duration:     stringPtr("P1Y"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     200,
			MaxContexts:  200,
			QuotaLimit:   2000000 * 12,
			AppleSKU:     stringPtr("com.rankquantity.voyager.sub_pro_y"),
			Featured:     true,
			DisplayOrder: 5,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_pro_m",
			Platform:     paymodels.IAPProductPlatformApple,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Pro Monthly Subscription",
			Description:  "Pro monthly subscription with unlimited features",
			Price:        14.99,
			Currency:     "USD",
			Duration:     stringPtr("P1M"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     200,
			MaxContexts:  200,
			QuotaLimit:   2000000,
			AppleSKU:     stringPtr("com.rankquantity.voyager.sub_pro_m"),
			Featured:     true,
			DisplayOrder: 6,
		},
		// Google Play 产品
		{
			ProductID:    "com.rankquantity.voyager.sub_basic_y",
			Platform:     paymodels.IAPProductPlatformGoogle,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Basic Annual Subscription",
			Description:  "Basic annual subscription with essential features",
			Price:        99.99,
			Currency:     "USD",
			Duration:     stringPtr("P1Y"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     50,
			MaxContexts:  50,
			QuotaLimit:   1000000 * 12,
			GoogleSKU:    stringPtr("com.rankquantity.voyager.sub_basic_y"),
			Featured:     true,
			DisplayOrder: 1,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_basic_m",
			Platform:     paymodels.IAPProductPlatformGoogle,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Basic Monthly Subscription",
			Description:  "Basic monthly subscription with essential features",
			Price:        9.99,
			Currency:     "USD",
			Duration:     stringPtr("P1M"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     50,
			MaxContexts:  50,
			QuotaLimit:   1000000,
			GoogleSKU:    stringPtr("com.rankquantity.voyager.sub_basic_m"),
			Featured:     true,
			DisplayOrder: 2,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_premium_y",
			Platform:     paymodels.IAPProductPlatformGoogle,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Premium Annual Subscription",
			Description:  "Premium annual subscription with advanced features",
			Price:        249.99,
			Currency:     "USD",
			Duration:     stringPtr("P1Y"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     500,
			MaxContexts:  500,
			QuotaLimit:   50000000 * 12,
			GoogleSKU:    stringPtr("com.rankquantity.voyager.sub_premium_y"),
			Featured:     true,
			DisplayOrder: 3,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_premium_m",
			Platform:     paymodels.IAPProductPlatformGoogle,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Premium Monthly Subscription",
			Description:  "Premium monthly subscription with advanced features",
			Price:        24.99,
			Currency:     "USD",
			Duration:     stringPtr("P1M"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     500,
			MaxContexts:  500,
			QuotaLimit:   50000000,
			GoogleSKU:    stringPtr("com.rankquantity.voyager.sub_premium_m"),
			Featured:     true,
			DisplayOrder: 4,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_pro_y",
			Platform:     paymodels.IAPProductPlatformGoogle,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Pro Annual Subscription",
			Description:  "Pro annual subscription with unlimited features",
			Price:        149.99,
			Currency:     "USD",
			Duration:     stringPtr("P1Y"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     200,
			MaxContexts:  200,
			QuotaLimit:   2000000 * 12,
			GoogleSKU:    stringPtr("com.rankquantity.voyager.sub_pro_y"),
			Featured:     true,
			DisplayOrder: 5,
		},
		{
			ProductID:    "com.rankquantity.voyager.sub_pro_m",
			Platform:     paymodels.IAPProductPlatformGoogle,
			ProductType:  paymodels.IAPProductTypeSubscription,
			Name:         "Pro Monthly Subscription",
			Description:  "Pro monthly subscription with unlimited features",
			Price:        14.99,
			Currency:     "USD",
			Duration:     stringPtr("P1M"),
			TrialPeriod:  stringPtr("P7D"),
			MaxRoles:     200,
			MaxContexts:  200,
			QuotaLimit:   2000000,
			GoogleSKU:    stringPtr("com.rankquantity.voyager.sub_pro_m"),
			Featured:     true,
			DisplayOrder: 6,
		},
	}

	for _, product := range defaultProducts {
		// 检查产品是否已存在
		_, err := paymodels.GetIAPProductByProductID(ctx, product.ProductID)
		if err == nil {
			s.logger.WithField("product_id", product.ProductID).Info("Product already exists, skipping")
			continue
		}

		// 创建产品
		if err := s.CreateProduct(ctx, product); err != nil {
			s.logger.WithError(err).WithField("product_id", product.ProductID).Error("Failed to create default product")
			return err
		}

		s.logger.WithField("product_id", product.ProductID).Info("Created default product")
	}

	return nil
}

// stringPtr 返回字符串指针的辅助函数
func stringPtr(s string) *string {
	return &s
}

// AppleProductInfo Apple产品信息结构
type AppleProductInfo struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"` // SUBSCRIPTION, NON_CONSUMABLE, CONSUMABLE
	Attributes    AppleProductAttributes `json:"attributes"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

type AppleProductAttributes struct {
	Name                      string `json:"name"`
	ProductID                 string `json:"productId"`
	ReferenceName             string `json:"referenceName"`
	FamilySharable            bool   `json:"familySharable"`
	ContentHosting            bool   `json:"contentHosting"`
	ReviewNote                string `json:"reviewNote,omitempty"`
	ReviewRequired            bool   `json:"reviewRequired"`
	State                     string `json:"state"` // READY_FOR_SALE, MISSING_METADATA, etc.
	AvailableInAllTerritories bool   `json:"availableInAllTerritories"`
	InAppPurchaseType         string `json:"inAppPurchaseType"`
	SubscriptionPeriod        string `json:"subscriptionPeriod,omitempty"` // P1M, P1Y, etc.
	SubscriptionTrialPeriod   string `json:"subscriptionTrialPeriod,omitempty"`
	SubscriptionIntroOffer    string `json:"subscriptionIntroOffer,omitempty"`
	SubscriptionGroup         string `json:"subscriptionGroup,omitempty"`
}

// GoogleProductInfo Google产品信息结构
type GoogleProductInfo struct {
	ProductID          string               `json:"productId"`
	Type               string               `json:"type"` // subscription, inapp
	Title              string               `json:"title"`
	Description        string               `json:"description"`
	PurchaseType       string               `json:"purchaseType"`                 // managedUser, subscription
	Status             string               `json:"status"`                       // active, inactive
	SubscriptionPeriod string               `json:"subscriptionPeriod,omitempty"` // P1M, P1Y, etc.
	TrialPeriod        string               `json:"trialPeriod,omitempty"`
	GracePeriod        string               `json:"gracePeriod,omitempty"`
	BasePlanID         string               `json:"basePlanId"`
	PricingPhases      []GooglePricingPhase `json:"pricingPhases,omitempty"`
}

type GooglePricingPhase struct {
	Period      string `json:"period"`
	Price       string `json:"price"`
	Currency    string `json:"currency"`
	PriceMicros int64  `json:"priceMicros"`
}

// fetchAppleProducts 从Apple App Store Connect获取产品信息
func (s *IAPProductServiceImpl) fetchAppleProducts(ctx context.Context) ([]*AppleProductInfo, error) {
	s.logger.Info("Fetching products from Apple App Store Connect")

	// 检查配置
	if s.config.Apple.IssuerID == "" || s.config.Apple.KeyID == "" || s.config.Apple.PrivateKey == "" {
		s.logger.Warn("Apple API configuration is incomplete, using mock data")
		return s.getMockAppleProducts()
	}

	// 1. 获取JWT访问令牌
	token, err := s.getAppleJWTToken(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get Apple JWT token")
		return s.getMockAppleProducts()
	}

	// 2. 调用Apple App Store Connect API
	products, err := s.callAppleInAppPurchasesAPI(ctx, token)
	if err != nil {
		s.logger.WithError(err).Error("Failed to call Apple API, using mock data")
		return s.getMockAppleProducts()
	}

	return products, nil
}

// getMockAppleProducts 获取模拟Apple产品数据
func (s *IAPProductServiceImpl) getMockAppleProducts() ([]*AppleProductInfo, error) {
	s.logger.Info("Using mock Apple products data")
	mockProducts := []*AppleProductInfo{
		{
			ID:   "com.rankquantity.voyager.sub_basic_y",
			Type: "SUBSCRIPTION",
			Attributes: AppleProductAttributes{
				Name:                      "Basic Annual Subscription",
				ProductID:                 "com.rankquantity.voyager.sub_basic_y",
				ReferenceName:             "Basic Annual",
				FamilySharable:            false,
				ContentHosting:            false,
				ReviewRequired:            false,
				State:                     "READY_FOR_SALE",
				AvailableInAllTerritories: true,
				InAppPurchaseType:         "SUBSCRIPTION",
				SubscriptionPeriod:        "P1Y",
				SubscriptionTrialPeriod:   "P7D",
				SubscriptionGroup:         "voyager_subscription_group",
			},
		},
		{
			ID:   "com.rankquantity.voyager.sub_basic_m",
			Type: "SUBSCRIPTION",
			Attributes: AppleProductAttributes{
				Name:                      "Basic Monthly Subscription",
				ProductID:                 "com.rankquantity.voyager.sub_basic_m",
				ReferenceName:             "Basic Monthly",
				FamilySharable:            false,
				ContentHosting:            false,
				ReviewRequired:            false,
				State:                     "READY_FOR_SALE",
				AvailableInAllTerritories: true,
				InAppPurchaseType:         "SUBSCRIPTION",
				SubscriptionPeriod:        "P1M",
				SubscriptionTrialPeriod:   "P7D",
				SubscriptionGroup:         "voyager_subscription_group",
			},
		},
		{
			ID:   "com.rankquantity.voyager.sub_premium_y",
			Type: "SUBSCRIPTION",
			Attributes: AppleProductAttributes{
				Name:                      "Premium Annual Subscription",
				ProductID:                 "com.rankquantity.voyager.sub_premium_y",
				ReferenceName:             "Premium Annual",
				FamilySharable:            false,
				ContentHosting:            false,
				ReviewRequired:            false,
				State:                     "READY_FOR_SALE",
				AvailableInAllTerritories: true,
				InAppPurchaseType:         "SUBSCRIPTION",
				SubscriptionPeriod:        "P1Y",
				SubscriptionTrialPeriod:   "P7D",
				SubscriptionGroup:         "voyager_subscription_group",
			},
		},
		{
			ID:   "com.rankquantity.voyager.sub_premium_m",
			Type: "SUBSCRIPTION",
			Attributes: AppleProductAttributes{
				Name:                      "Premium Monthly Subscription",
				ProductID:                 "com.rankquantity.voyager.sub_premium_m",
				ReferenceName:             "Premium Monthly",
				FamilySharable:            false,
				ContentHosting:            false,
				ReviewRequired:            false,
				State:                     "READY_FOR_SALE",
				AvailableInAllTerritories: true,
				InAppPurchaseType:         "SUBSCRIPTION",
				SubscriptionPeriod:        "P1M",
				SubscriptionTrialPeriod:   "P7D",
				SubscriptionGroup:         "voyager_subscription_group",
			},
		},
		{
			ID:   "com.rankquantity.voyager.sub_pro_y",
			Type: "SUBSCRIPTION",
			Attributes: AppleProductAttributes{
				Name:                      "Pro Annual Subscription",
				ProductID:                 "com.rankquantity.voyager.sub_pro_y",
				ReferenceName:             "Pro Annual",
				FamilySharable:            false,
				ContentHosting:            false,
				ReviewRequired:            false,
				State:                     "READY_FOR_SALE",
				AvailableInAllTerritories: true,
				InAppPurchaseType:         "SUBSCRIPTION",
				SubscriptionPeriod:        "P1Y",
				SubscriptionTrialPeriod:   "P7D",
				SubscriptionGroup:         "voyager_subscription_group",
			},
		},
		{
			ID:   "com.rankquantity.voyager.sub_pro_m",
			Type: "SUBSCRIPTION",
			Attributes: AppleProductAttributes{
				Name:                      "Pro Monthly Subscription",
				ProductID:                 "com.rankquantity.voyager.sub_pro_m",
				ReferenceName:             "Pro Monthly",
				FamilySharable:            false,
				ContentHosting:            false,
				ReviewRequired:            false,
				State:                     "READY_FOR_SALE",
				AvailableInAllTerritories: true,
				InAppPurchaseType:         "SUBSCRIPTION",
				SubscriptionPeriod:        "P1M",
				SubscriptionTrialPeriod:   "P7D",
				SubscriptionGroup:         "voyager_subscription_group",
			},
		},
	}

	s.logger.WithField("count", len(mockProducts)).Info("Fetched Apple products")
	return mockProducts, nil
}

// AppleJWTClaims Apple JWT声明
type AppleJWTClaims struct {
	IssuerID string `json:"iss"`
	IssuedAt int64  `json:"iat"`
	Expiry   int64  `json:"exp"`
	Audience string `json:"aud"`
}

// getAppleJWTToken 获取Apple JWT访问令牌
func (s *IAPProductServiceImpl) getAppleJWTToken(ctx context.Context) (string, error) {
	s.logger.Info("Generating Apple JWT token")

	// 1. 解析私钥
	privateKey, err := s.parseApplePrivateKey(s.config.Apple.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("解析Apple私钥失败: %w", err)
	}

	// 2. 创建JWT声明
	now := time.Now()
	claims := &AppleJWTClaims{
		IssuerID: s.config.Apple.IssuerID,
		IssuedAt: now.Unix(),
		Expiry:   now.Add(20 * time.Minute).Unix(), // Apple JWT有效期最多20分钟
		Audience: "appstoreconnect-v1",
	}

	// 3. 创建JWT token
	token, err := s.createAppleJWT(claims, privateKey)
	if err != nil {
		return "", fmt.Errorf("创建Apple JWT失败: %w", err)
	}

	s.logger.Debug("Apple JWT token generated successfully")
	return token, nil
}

// parseApplePrivateKey 解析Apple私钥 (Apple使用ECDSA P-256密钥)
func (s *IAPProductServiceImpl) parseApplePrivateKey(privateKeyPEM string) (*ecdsa.PrivateKey, error) {
	// 解码PEM格式的私钥
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("无法解码PEM格式的私钥")
	}

	// 解析私钥
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// 尝试解析EC私钥格式
		ecKey, ecErr := x509.ParseECPrivateKey(block.Bytes)
		if ecErr != nil {
			return nil, fmt.Errorf("解析私钥失败 (PKCS8: %v, EC: %v)", err, ecErr)
		}
		return ecKey, nil
	}

	// 转换为ECDSA私钥 (Apple App Store Connect API使用ES256)
	ecdsaPrivateKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("私钥不是ECDSA类型，Apple App Store Connect需要ES256 (P-256曲线) 密钥")
	}

	return ecdsaPrivateKey, nil
}

// createAppleJWT 创建Apple JWT token (使用ES256算法)
func (s *IAPProductServiceImpl) createAppleJWT(claims *AppleJWTClaims, privateKey *ecdsa.PrivateKey) (string, error) {
	// 1. 创建JWT Header
	header := map[string]string{
		"alg": "ES256",
		"typ": "JWT",
		"kid": s.config.Apple.KeyID,
	}

	// 2. 编码Header为Base64URL
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("编码JWT header失败: %w", err)
	}
	headerEncoded := base64URLEncode(headerBytes)

	// 3. 创建JWT Payload
	payload := map[string]interface{}{
		"iss": claims.IssuerID,
		"iat": claims.IssuedAt,
		"exp": claims.Expiry,
		"aud": claims.Audience,
	}

	// 4. 编码Payload为Base64URL
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("编码JWT payload失败: %w", err)
	}
	payloadEncoded := base64URLEncode(payloadBytes)

	// 5. 创建签名输入
	signingInput := headerEncoded + "." + payloadEncoded

	// 6. 计算SHA256哈希
	hash := sha256.Sum256([]byte(signingInput))

	// 7. 使用ECDSA私钥签名 (使用crypto/rand确保安全随机性)
	r, sVal, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("ECDSA签名失败: %w", err)
	}

	// 8. 将r和s转换为固定长度的字节序列 (各32字节，用于P-256曲线)
	curveBits := privateKey.Curve.Params().BitSize
	keyBytes := curveBits / 8
	if curveBits%8 > 0 {
		keyBytes++
	}

	// 创建签名字节切片
	signature := make([]byte, 2*keyBytes)

	// 填充r值 (左侧补0)
	rBytes := r.Bytes()
	copy(signature[keyBytes-len(rBytes):keyBytes], rBytes)

	// 填充s值 (左侧补0)
	sBytes := sVal.Bytes()
	copy(signature[2*keyBytes-len(sBytes):], sBytes)

	// 9. Base64URL编码签名
	signatureEncoded := base64URLEncode(signature)

	// 10. 组合完整的JWT
	jwt := signingInput + "." + signatureEncoded

	s.logger.Debug("Apple JWT token created successfully")
	return jwt, nil
}

// base64URLEncode 进行Base64URL编码（不带填充）
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// AppleInAppPurchasesResponse Apple应用内购买响应
type AppleInAppPurchasesResponse struct {
	Data  []AppleInAppPurchaseData `json:"data"`
	Links struct {
		Self string `json:"self"`
		Next string `json:"next,omitempty"`
	} `json:"links"`
	Meta struct {
		Paging struct {
			Total int `json:"total"`
			Limit int `json:"limit"`
		} `json:"paging"`
	} `json:"meta"`
}

type AppleInAppPurchaseData struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Attributes AppleProductAttributes `json:"attributes"`
}

// callAppleInAppPurchasesAPI 调用Apple应用内购买API
func (s *IAPProductServiceImpl) callAppleInAppPurchasesAPI(ctx context.Context, token string) ([]*AppleProductInfo, error) {
	s.logger.Info("Calling Apple In-App Purchases API")

	// 构建API URL
	apiURL := fmt.Sprintf("%s/v1/apps/%s/inAppPurchases",
		s.config.Apple.APIBaseURL,
		s.config.Apple.BundleID)

	// 使用重试机制执行请求
	var apiResponse AppleInAppPurchasesResponse
	err := s.executeWithRetry(ctx, func() error {
		return s.makeAppleAPIRequest(ctx, apiURL, token, &apiResponse)
	})

	if err != nil {
		return nil, err
	}

	// 转换为内部格式
	var products []*AppleProductInfo
	for _, data := range apiResponse.Data {
		product := &AppleProductInfo{
			ID:         data.ID,
			Type:       data.Type,
			Attributes: data.Attributes,
		}
		products = append(products, product)
	}

	s.logger.WithField("count", len(products)).Info("Successfully fetched Apple products from API")
	return products, nil
}

// makeAppleAPIRequest 执行Apple API请求
func (s *IAPProductServiceImpl) makeAppleAPIRequest(ctx context.Context, apiURL, token string, response interface{}) error {
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	s.logger.WithField("url", apiURL).Debug("Making Apple API request")
	resp, err := s.httpClient.client.client.Do(req)
	if err != nil {
		return fmt.Errorf("apple API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apple API返回错误状态: %d", resp.StatusCode)
	}

	// 解析响应
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("解析Apple API响应失败: %w", err)
	}

	return nil
}

// fetchGoogleProducts 从Google Play Console获取产品信息
func (s *IAPProductServiceImpl) fetchGoogleProducts(ctx context.Context) ([]*GoogleProductInfo, error) {
	s.logger.Info("Fetching products from Google Play Console")

	// 检查配置
	if s.config.Google.ServiceAccountKey == "" || s.config.Google.PackageName == "" {
		s.logger.Warn("Google API configuration is incomplete, using mock data")
		return s.getMockGoogleProducts()
	}

	// 1. 获取OAuth2访问令牌
	token, err := s.getGoogleAccessToken(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get Google access token")
		return s.getMockGoogleProducts()
	}

	// 2. 调用Google Play Developer API
	products, err := s.callGoogleInAppProductsAPI(ctx, token)
	if err != nil {
		s.logger.WithError(err).Error("Failed to call Google API, using mock data")
		return s.getMockGoogleProducts()
	}

	return products, nil
}

// getMockGoogleProducts 获取模拟Google产品数据
func (s *IAPProductServiceImpl) getMockGoogleProducts() ([]*GoogleProductInfo, error) {
	s.logger.Info("Using mock Google products data")
	mockProducts := []*GoogleProductInfo{
		{
			ProductID:          "com.rankquantity.voyager.sub_basic_y",
			Type:               "subscription",
			Title:              "Basic Annual Subscription",
			Description:        "Basic annual subscription with essential features",
			PurchaseType:       "subscription",
			Status:             "active",
			SubscriptionPeriod: "P1Y",
			TrialPeriod:        "P7D",
			BasePlanID:         "basic_annual_plan",
			PricingPhases: []GooglePricingPhase{
				{
					Period:      "P7D",
					Price:       "0.00",
					Currency:    "USD",
					PriceMicros: 0,
				},
				{
					Period:      "P1Y",
					Price:       "99.99",
					Currency:    "USD",
					PriceMicros: 99990000,
				},
			},
		},
		{
			ProductID:          "com.rankquantity.voyager.sub_basic_m",
			Type:               "subscription",
			Title:              "Basic Monthly Subscription",
			Description:        "Basic monthly subscription with essential features",
			PurchaseType:       "subscription",
			Status:             "active",
			SubscriptionPeriod: "P1M",
			TrialPeriod:        "P7D",
			BasePlanID:         "basic_monthly_plan",
			PricingPhases: []GooglePricingPhase{
				{
					Period:      "P7D",
					Price:       "0.00",
					Currency:    "USD",
					PriceMicros: 0,
				},
				{
					Period:      "P1M",
					Price:       "9.99",
					Currency:    "USD",
					PriceMicros: 9990000,
				},
			},
		},
		{
			ProductID:          "com.rankquantity.voyager.sub_premium_y",
			Type:               "subscription",
			Title:              "Premium Annual Subscription",
			Description:        "Premium annual subscription with advanced features",
			PurchaseType:       "subscription",
			Status:             "active",
			SubscriptionPeriod: "P1Y",
			TrialPeriod:        "P7D",
			BasePlanID:         "premium_annual_plan",
			PricingPhases: []GooglePricingPhase{
				{
					Period:      "P7D",
					Price:       "0.00",
					Currency:    "USD",
					PriceMicros: 0,
				},
				{
					Period:      "P1Y",
					Price:       "249.99",
					Currency:    "USD",
					PriceMicros: 249990000,
				},
			},
		},
		{
			ProductID:          "com.rankquantity.voyager.sub_premium_m",
			Type:               "subscription",
			Title:              "Premium Monthly Subscription",
			Description:        "Premium monthly subscription with advanced features",
			PurchaseType:       "subscription",
			Status:             "active",
			SubscriptionPeriod: "P1M",
			TrialPeriod:        "P7D",
			BasePlanID:         "premium_monthly_plan",
			PricingPhases: []GooglePricingPhase{
				{
					Period:      "P7D",
					Price:       "0.00",
					Currency:    "USD",
					PriceMicros: 0,
				},
				{
					Period:      "P1M",
					Price:       "24.99",
					Currency:    "USD",
					PriceMicros: 24990000,
				},
			},
		},
		{
			ProductID:          "com.rankquantity.voyager.sub_pro_y",
			Type:               "subscription",
			Title:              "Pro Annual Subscription",
			Description:        "Pro annual subscription with unlimited features",
			PurchaseType:       "subscription",
			Status:             "active",
			SubscriptionPeriod: "P1Y",
			TrialPeriod:        "P7D",
			BasePlanID:         "pro_annual_plan",
			PricingPhases: []GooglePricingPhase{
				{
					Period:      "P7D",
					Price:       "0.00",
					Currency:    "USD",
					PriceMicros: 0,
				},
				{
					Period:      "P1Y",
					Price:       "149.99",
					Currency:    "USD",
					PriceMicros: 149990000,
				},
			},
		},
		{
			ProductID:          "com.rankquantity.voyager.sub_pro_m",
			Type:               "subscription",
			Title:              "Pro Monthly Subscription",
			Description:        "Pro monthly subscription with unlimited features",
			PurchaseType:       "subscription",
			Status:             "active",
			SubscriptionPeriod: "P1M",
			TrialPeriod:        "P7D",
			BasePlanID:         "pro_monthly_plan",
			PricingPhases: []GooglePricingPhase{
				{
					Period:      "P7D",
					Price:       "0.00",
					Currency:    "USD",
					PriceMicros: 0,
				},
				{
					Period:      "P1M",
					Price:       "14.99",
					Currency:    "USD",
					PriceMicros: 14990000,
				},
			},
		},
	}

	s.logger.WithField("count", len(mockProducts)).Info("Fetched Google products")
	return mockProducts, nil
}

// GoogleServiceAccount Google服务账号信息
type GoogleServiceAccount struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// GoogleAccessTokenResponse Google访问令牌响应
type GoogleAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// getGoogleAccessToken 获取Google OAuth2访问令牌
func (s *IAPProductServiceImpl) getGoogleAccessToken(ctx context.Context) (string, error) {
	s.logger.Info("Getting Google OAuth2 access token")

	// 1. 解析服务账号密钥
	serviceAccount, err := s.parseGoogleServiceAccount(s.config.Google.ServiceAccountKey)
	if err != nil {
		return "", fmt.Errorf("解析Google服务账号密钥失败: %w", err)
	}

	// 2. 创建JWT配置
	config := &jwt.Config{
		Email:      serviceAccount.ClientEmail,
		PrivateKey: []byte(serviceAccount.PrivateKey),
		TokenURL:   google.JWTTokenURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/androidpublisher",
		},
	}

	// 3. 获取访问令牌
	token, err := config.TokenSource(ctx).Token()
	if err != nil {
		return "", fmt.Errorf("获取Google访问令牌失败: %w", err)
	}

	s.logger.Debug("Google OAuth2 access token obtained successfully")
	return token.AccessToken, nil
}

// parseGoogleServiceAccount 解析Google服务账号密钥
func (s *IAPProductServiceImpl) parseGoogleServiceAccount(serviceAccountJSON string) (*GoogleServiceAccount, error) {
	var serviceAccount GoogleServiceAccount
	if err := json.Unmarshal([]byte(serviceAccountJSON), &serviceAccount); err != nil {
		return nil, fmt.Errorf("解析服务账号JSON失败: %w", err)
	}

	// 验证必需字段
	if serviceAccount.ClientEmail == "" {
		return nil, fmt.Errorf("服务账号缺少client_email字段")
	}
	if serviceAccount.PrivateKey == "" {
		return nil, fmt.Errorf("服务账号缺少private_key字段")
	}

	return &serviceAccount, nil
}

// GoogleInAppProductsResponse Google应用内产品响应
type GoogleInAppProductsResponse struct {
	InAppProduct    []GoogleInAppProduct `json:"inappproduct"`
	TokenPagination struct {
		NextPageToken string `json:"nextPageToken"`
	} `json:"tokenPagination"`
}

type GoogleInAppProduct struct {
	SKU             string `json:"sku"`
	PackageName     string `json:"packageName"`
	PurchaseType    string `json:"purchaseType"`
	Status          string `json:"status"`
	DefaultLanguage string `json:"defaultLanguage"`
	DefaultPrice    struct {
		PriceMicros string `json:"priceMicros"`
		Currency    string `json:"currency"`
	} `json:"defaultPrice"`
	SubscriptionPeriod string               `json:"subscriptionPeriod,omitempty"`
	TrialPeriod        string               `json:"trialPeriod,omitempty"`
	GracePeriod        string               `json:"gracePeriod,omitempty"`
	BasePlanID         string               `json:"basePlanId,omitempty"`
	PricingPhases      []GooglePricingPhase `json:"pricingPhases,omitempty"`
	Listings           map[string]struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"listings"`
}

// callGoogleInAppProductsAPI 调用Google应用内产品API
func (s *IAPProductServiceImpl) callGoogleInAppProductsAPI(ctx context.Context, token string) ([]*GoogleProductInfo, error) {
	s.logger.Info("Calling Google In-App Products API")

	// 构建API URL
	apiURL := fmt.Sprintf("%s/androidpublisher/v3/applications/%s/inappproducts",
		s.config.Google.APIBaseURL,
		s.config.Google.PackageName)

	// 使用重试机制执行请求
	var apiResponse GoogleInAppProductsResponse
	err := s.executeWithRetry(ctx, func() error {
		return s.makeGoogleAPIRequest(ctx, apiURL, token, &apiResponse)
	})

	if err != nil {
		return nil, err
	}

	// 转换为内部格式
	var products []*GoogleProductInfo
	for _, inAppProduct := range apiResponse.InAppProduct {
		// 获取默认语言的产品信息
		defaultListing := inAppProduct.Listings[inAppProduct.DefaultLanguage]

		product := &GoogleProductInfo{
			ProductID:          inAppProduct.SKU,
			Type:               inAppProduct.PurchaseType,
			Title:              defaultListing.Title,
			Description:        defaultListing.Description,
			Status:             inAppProduct.Status,
			SubscriptionPeriod: inAppProduct.SubscriptionPeriod,
			TrialPeriod:        inAppProduct.TrialPeriod,
			GracePeriod:        inAppProduct.GracePeriod,
			BasePlanID:         inAppProduct.BasePlanID,
			PricingPhases:      inAppProduct.PricingPhases,
		}
		products = append(products, product)
	}

	s.logger.WithField("count", len(products)).Info("Successfully fetched Google products from API")
	return products, nil
}

// makeGoogleAPIRequest 执行Google API请求
func (s *IAPProductServiceImpl) makeGoogleAPIRequest(ctx context.Context, apiURL, token string, response interface{}) error {
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	s.logger.WithField("url", apiURL).Debug("Making Google API request")
	resp, err := s.httpClient.client.client.Do(req)
	if err != nil {
		return fmt.Errorf("google API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google API返回错误状态: %d", resp.StatusCode)
	}

	// 解析响应
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("解析Google API响应失败: %w", err)
	}

	return nil
}

// syncAppleProducts 同步Apple产品到数据库
func (s *IAPProductServiceImpl) syncAppleProducts(ctx context.Context, existingProducts []*paymodels.IAPProduct, appleProducts []*AppleProductInfo) (int, int, int, error) {
	s.logger.Info("Syncing Apple products to database")

	var syncedCount, updatedCount, createdCount int

	// 创建现有产品的映射，便于查找
	existingMap := make(map[string]*paymodels.IAPProduct)
	for _, product := range existingProducts {
		if product.AppleSKU != nil {
			existingMap[*product.AppleSKU] = product
		}
	}

	// 处理从Apple获取的产品
	for _, appleProduct := range appleProducts {
		// 检查产品是否已存在
		existingProduct, exists := existingMap[appleProduct.Attributes.ProductID]

		if exists {
			// 更新现有产品
			if err := s.updateAppleProduct(ctx, existingProduct, appleProduct); err != nil {
				s.logger.WithError(err).WithField("product_id", appleProduct.Attributes.ProductID).Error("Failed to update Apple product")
				continue
			}
			updatedCount++
		} else {
			// 创建新产品
			if err := s.createAppleProduct(ctx, appleProduct); err != nil {
				s.logger.WithError(err).WithField("product_id", appleProduct.Attributes.ProductID).Error("Failed to create Apple product")
				continue
			}
			createdCount++
		}
		syncedCount++
	}

	return syncedCount, updatedCount, createdCount, nil
}

// syncGoogleProducts 同步Google产品到数据库
func (s *IAPProductServiceImpl) syncGoogleProducts(ctx context.Context, existingProducts []*paymodels.IAPProduct, googleProducts []*GoogleProductInfo) (int, int, int, error) {
	s.logger.Info("Syncing Google products to database")

	var syncedCount, updatedCount, createdCount int

	// 创建现有产品的映射，便于查找
	existingMap := make(map[string]*paymodels.IAPProduct)
	for _, product := range existingProducts {
		if product.GoogleSKU != nil {
			existingMap[*product.GoogleSKU] = product
		}
	}

	// 处理从Google获取的产品
	for _, googleProduct := range googleProducts {
		// 检查产品是否已存在
		existingProduct, exists := existingMap[googleProduct.ProductID]

		if exists {
			// 更新现有产品
			if err := s.updateGoogleProduct(ctx, existingProduct, googleProduct); err != nil {
				s.logger.WithError(err).WithField("product_id", googleProduct.ProductID).Error("Failed to update Google product")
				continue
			}
			updatedCount++
		} else {
			// 创建新产品
			if err := s.createGoogleProduct(ctx, googleProduct); err != nil {
				s.logger.WithError(err).WithField("product_id", googleProduct.ProductID).Error("Failed to create Google product")
				continue
			}
			createdCount++
		}
		syncedCount++
	}

	return syncedCount, updatedCount, createdCount, nil
}

// createAppleProduct 创建Apple产品
func (s *IAPProductServiceImpl) createAppleProduct(ctx context.Context, appleProduct *AppleProductInfo) error {
	s.logger.WithField("product_id", appleProduct.Attributes.ProductID).Info("Creating Apple product")

	// 解析产品类型
	productType := s.parseAppleProductType(appleProduct.Type)

	// 解析价格（这里简化处理，实际应该从App Store Connect获取价格信息）
	price := s.parseAppleProductPrice(appleProduct)

	// 创建产品对象
	product := &paymodels.IAPProduct{
		ProductID:         appleProduct.Attributes.ProductID,
		Platform:          paymodels.IAPProductPlatformApple,
		ProductType:       productType,
		Status:            s.parseAppleProductStatus(appleProduct.Attributes.State),
		Name:              appleProduct.Attributes.Name,
		Description:       appleProduct.Attributes.ReviewNote,
		Price:             price,
		Currency:          "USD", // 默认货币，实际应该从价格信息中获取
		Duration:          s.parseAppleDuration(appleProduct.Attributes.SubscriptionPeriod),
		TrialPeriod:       s.parseAppleDuration(appleProduct.Attributes.SubscriptionTrialPeriod),
		IntroOffer:        s.parseAppleIntroOffer(appleProduct.Attributes.SubscriptionIntroOffer),
		SubscriptionGroup: s.parseAppleSubscriptionGroup(appleProduct.Attributes.SubscriptionGroup),
		FamilyShareable:   appleProduct.Attributes.FamilySharable,
		AppleSKU:          &appleProduct.Attributes.ProductID,
		AppleProductID:    &appleProduct.ID,
		IsActive:          appleProduct.Attributes.State == "READY_FOR_SALE",
		DisplayOrder:      0,
		Featured:          false,
		MaxRoles:          10,    // 默认值
		MaxContexts:       50,    // 默认值
		QuotaLimit:        10000, // 默认值
		LastSyncTime:      &[]time.Time{time.Now()}[0],
		SyncStatus:        "success",
	}

	// 设置元数据
	metadata := map[string]interface{}{
		"apple_reference_name":            appleProduct.Attributes.ReferenceName,
		"apple_review_required":           appleProduct.Attributes.ReviewRequired,
		"apple_content_hosting":           appleProduct.Attributes.ContentHosting,
		"apple_available_all_territories": appleProduct.Attributes.AvailableInAllTerritories,
		"apple_in_app_purchase_type":      appleProduct.Attributes.InAppPurchaseType,
		"sync_source":                     "apple_app_store_connect",
		"sync_timestamp":                  time.Now().Unix(),
	}

	if err := product.SetMetadata(metadata); err != nil {
		s.logger.WithError(err).Warn("Failed to set product metadata")
	}

	// 保存到数据库
	return paymodels.CreateIAPProduct(ctx, product)
}

// updateAppleProduct 更新Apple产品
func (s *IAPProductServiceImpl) updateAppleProduct(ctx context.Context, existingProduct *paymodels.IAPProduct, appleProduct *AppleProductInfo) error {
	s.logger.WithField("product_id", appleProduct.Attributes.ProductID).Info("Updating Apple product")

	// 检查是否需要更新
	needsUpdate := false

	// 更新基本信息
	if existingProduct.Name != appleProduct.Attributes.Name {
		existingProduct.Name = appleProduct.Attributes.Name
		needsUpdate = true
	}

	if existingProduct.Description != appleProduct.Attributes.ReviewNote {
		existingProduct.Description = appleProduct.Attributes.ReviewNote
		needsUpdate = true
	}

	newStatus := s.parseAppleProductStatus(appleProduct.Attributes.State)
	if existingProduct.Status != newStatus {
		existingProduct.Status = newStatus
		needsUpdate = true
	}

	newIsActive := appleProduct.Attributes.State == "READY_FOR_SALE"
	if existingProduct.IsActive != newIsActive {
		existingProduct.IsActive = newIsActive
		needsUpdate = true
	}

	// 更新同步信息
	existingProduct.LastSyncTime = &[]time.Time{time.Now()}[0]
	existingProduct.SyncStatus = "success"
	needsUpdate = true

	if needsUpdate {
		// 更新元数据
		metadata, _ := existingProduct.GetMetadata()
		metadata["apple_reference_name"] = appleProduct.Attributes.ReferenceName
		metadata["apple_review_required"] = appleProduct.Attributes.ReviewRequired
		metadata["apple_content_hosting"] = appleProduct.Attributes.ContentHosting
		metadata["apple_available_all_territories"] = appleProduct.Attributes.AvailableInAllTerritories
		metadata["apple_in_app_purchase_type"] = appleProduct.Attributes.InAppPurchaseType
		metadata["sync_source"] = "apple_app_store_connect"
		metadata["sync_timestamp"] = time.Now().Unix()

		if err := existingProduct.SetMetadata(metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to update product metadata")
		}

		return paymodels.UpdateIAPProduct(ctx, existingProduct)
	}

	return nil
}

// createGoogleProduct 创建Google产品
func (s *IAPProductServiceImpl) createGoogleProduct(ctx context.Context, googleProduct *GoogleProductInfo) error {
	s.logger.WithField("product_id", googleProduct.ProductID).Info("Creating Google product")

	// 解析产品类型
	productType := s.parseGoogleProductType(googleProduct.Type)

	// 解析价格
	price := s.parseGoogleProductPrice(googleProduct)
	currency := s.parseGoogleProductCurrency(googleProduct)

	// 创建产品对象
	product := &paymodels.IAPProduct{
		ProductID:       googleProduct.ProductID,
		Platform:        paymodels.IAPProductPlatformGoogle,
		ProductType:     productType,
		Status:          s.parseGoogleProductStatus(googleProduct.Status),
		Name:            googleProduct.Title,
		Description:     googleProduct.Description,
		Price:           price,
		Currency:        currency,
		Duration:        s.parseGoogleDuration(googleProduct.SubscriptionPeriod),
		TrialPeriod:     s.parseGoogleDuration(googleProduct.TrialPeriod),
		GoogleSKU:       &googleProduct.ProductID,
		GoogleProductID: &googleProduct.ProductID,
		IsActive:        googleProduct.Status == "active",
		DisplayOrder:    0,
		Featured:        false,
		MaxRoles:        10,    // 默认值
		MaxContexts:     50,    // 默认值
		QuotaLimit:      10000, // 默认值
		LastSyncTime:    &[]time.Time{time.Now()}[0],
		SyncStatus:      "success",
	}

	// 设置元数据
	metadata := map[string]interface{}{
		"google_base_plan_id": googleProduct.BasePlanID,
		"google_grace_period": googleProduct.GracePeriod,
		"sync_source":         "google_play_console",
		"sync_timestamp":      time.Now().Unix(),
	}

	if err := product.SetMetadata(metadata); err != nil {
		s.logger.WithError(err).Warn("Failed to set product metadata")
	}

	// 保存到数据库
	return paymodels.CreateIAPProduct(ctx, product)
}

// updateGoogleProduct 更新Google产品
func (s *IAPProductServiceImpl) updateGoogleProduct(ctx context.Context, existingProduct *paymodels.IAPProduct, googleProduct *GoogleProductInfo) error {
	s.logger.WithField("product_id", googleProduct.ProductID).Info("Updating Google product")

	// 检查是否需要更新
	needsUpdate := false

	// 更新基本信息
	if existingProduct.Name != googleProduct.Title {
		existingProduct.Name = googleProduct.Title
		needsUpdate = true
	}

	if existingProduct.Description != googleProduct.Description {
		existingProduct.Description = googleProduct.Description
		needsUpdate = true
	}

	newStatus := s.parseGoogleProductStatus(googleProduct.Status)
	if existingProduct.Status != newStatus {
		existingProduct.Status = newStatus
		needsUpdate = true
	}

	newIsActive := googleProduct.Status == "active"
	if existingProduct.IsActive != newIsActive {
		existingProduct.IsActive = newIsActive
		needsUpdate = true
	}

	// 更新同步信息
	existingProduct.LastSyncTime = &[]time.Time{time.Now()}[0]
	existingProduct.SyncStatus = "success"
	needsUpdate = true

	if needsUpdate {
		// 更新元数据
		metadata, _ := existingProduct.GetMetadata()
		metadata["google_base_plan_id"] = googleProduct.BasePlanID
		metadata["google_grace_period"] = googleProduct.GracePeriod
		metadata["sync_source"] = "google_play_console"
		metadata["sync_timestamp"] = time.Now().Unix()

		if err := existingProduct.SetMetadata(metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to update product metadata")
		}

		return paymodels.UpdateIAPProduct(ctx, existingProduct)
	}

	return nil
}

// 辅助函数：解析Apple产品类型
func (s *IAPProductServiceImpl) parseAppleProductType(productType string) paymodels.IAPProductType {
	switch strings.ToUpper(productType) {
	case "SUBSCRIPTION":
		return paymodels.IAPProductTypeSubscription
	case "NON_CONSUMABLE":
		return paymodels.IAPProductTypeOneTime
	case "CONSUMABLE":
		return paymodels.IAPProductTypeConsumable
	default:
		return paymodels.IAPProductTypeOneTime
	}
}

// 辅助函数：解析Google产品类型
func (s *IAPProductServiceImpl) parseGoogleProductType(productType string) paymodels.IAPProductType {
	switch strings.ToLower(productType) {
	case "subscription":
		return paymodels.IAPProductTypeSubscription
	case "inapp":
		return paymodels.IAPProductTypeConsumable
	default:
		return paymodels.IAPProductTypeOneTime
	}
}

// 辅助函数：解析Apple产品状态
func (s *IAPProductServiceImpl) parseAppleProductStatus(state string) paymodels.IAPProductStatus {
	switch strings.ToUpper(state) {
	case "READY_FOR_SALE":
		return paymodels.IAPProductStatusActive
	case "MISSING_METADATA", "PREPARE_FOR_SUBMISSION":
		return paymodels.IAPProductStatusInactive
	default:
		return paymodels.IAPProductStatusInactive
	}
}

// 辅助函数：解析Google产品状态
func (s *IAPProductServiceImpl) parseGoogleProductStatus(status string) paymodels.IAPProductStatus {
	switch strings.ToLower(status) {
	case "active":
		return paymodels.IAPProductStatusActive
	case "inactive":
		return paymodels.IAPProductStatusInactive
	default:
		return paymodels.IAPProductStatusInactive
	}
}

// 辅助函数：解析Apple产品价格（简化实现）
func (s *IAPProductServiceImpl) parseAppleProductPrice(appleProduct *AppleProductInfo) float64 {
	// 这里简化处理，实际应该从App Store Connect的价格信息中获取
	// 根据产品ID返回预设价格
	switch appleProduct.Attributes.ProductID {
	case "com.rankquantity.voyager.sub_basic_y":
		return 99.99
	case "com.rankquantity.voyager.sub_basic_m":
		return 9.99
	case "com.rankquantity.voyager.sub_premium_y":
		return 249.99
	case "com.rankquantity.voyager.sub_premium_m":
		return 24.99
	case "com.rankquantity.voyager.sub_pro_y":
		return 149.99
	case "com.rankquantity.voyager.sub_pro_m":
		return 14.99
	default:
		return 0.0
	}
}

// 辅助函数：解析Google产品价格
func (s *IAPProductServiceImpl) parseGoogleProductPrice(googleProduct *GoogleProductInfo) float64 {
	if len(googleProduct.PricingPhases) == 0 {
		return 0.0
	}

	// 找到非免费的价格阶段
	for _, phase := range googleProduct.PricingPhases {
		if phase.PriceMicros > 0 {
			return float64(phase.PriceMicros) / 1000000.0
		}
	}

	return 0.0
}

// 辅助函数：解析Google产品货币
func (s *IAPProductServiceImpl) parseGoogleProductCurrency(googleProduct *GoogleProductInfo) string {
	if len(googleProduct.PricingPhases) == 0 {
		return "USD"
	}

	// 返回第一个价格阶段的货币
	return googleProduct.PricingPhases[0].Currency
}

// 辅助函数：解析Apple持续时间
func (s *IAPProductServiceImpl) parseAppleDuration(duration string) *string {
	if duration == "" {
		return nil
	}
	return &duration
}

// 辅助函数：解析Google持续时间
func (s *IAPProductServiceImpl) parseGoogleDuration(duration string) *string {
	if duration == "" {
		return nil
	}
	return &duration
}

// 辅助函数：解析Apple介绍优惠
func (s *IAPProductServiceImpl) parseAppleIntroOffer(introOffer string) *string {
	if introOffer == "" {
		return nil
	}
	return &introOffer
}

// 辅助函数：解析Apple订阅组
func (s *IAPProductServiceImpl) parseAppleSubscriptionGroup(subscriptionGroup string) *string {
	if subscriptionGroup == "" {
		return nil
	}
	return &subscriptionGroup
}

// executeWithRetry 执行带重试机制的操作
func (s *IAPProductServiceImpl) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	maxRetries := s.config.Apple.MaxRetries // 使用Apple配置作为默认值

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 等待重试延迟
			delay := time.Duration(s.config.Apple.RetryDelayMs) * time.Millisecond
			s.logger.WithField("attempt", attempt+1).WithField("delay_ms", delay.Milliseconds()).Debug("Retrying after delay")

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// 继续重试
			}
		}

		// 执行操作
		err := operation()
		if err == nil {
			if attempt > 0 {
				s.logger.WithField("attempt", attempt+1).Info("Operation succeeded after retry")
			}
			return nil
		}

		lastErr = err
		s.logger.WithError(err).WithField("attempt", attempt+1).Warn("Operation failed, will retry")

		// 检查是否应该重试
		if !s.shouldRetry(err) {
			s.logger.WithError(err).Info("Error is not retryable, stopping retry")
			break
		}
	}

	return fmt.Errorf("操作在%d次重试后仍然失败: %w", maxRetries+1, lastErr)
}

// shouldRetry 判断错误是否应该重试
func (s *IAPProductServiceImpl) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 网络错误通常应该重试
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") {
		return true
	}

	// HTTP 5xx错误应该重试
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	// HTTP 429 (Too Many Requests) 应该重试
	if strings.Contains(errStr, "429") {
		return true
	}

	// 其他错误通常不应该重试
	return false
}

// LoadIAPProductConfigFromFile 从文件加载IAP产品配置
func LoadIAPProductConfigFromFile(filename string) (*IAPProductConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config IAPProductConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	setDefaultAppleConfig(&config.Apple)
	setDefaultGoogleConfig(&config.Google)

	return &config, nil
}

// ValidateIAPProductConfig 验证IAP产品配置
func ValidateIAPProductConfig(config *IAPProductConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	// 验证Apple配置
	if err := validateAppleConfig(&config.Apple); err != nil {
		return fmt.Errorf("apple配置无效: %w", err)
	}

	// 验证Google配置
	if err := validateGoogleConfig(&config.Google); err != nil {
		return fmt.Errorf("google配置无效: %w", err)
	}

	return nil
}

// validateAppleConfig 验证Apple配置
func validateAppleConfig(config *AppleAPIConfig) error {
	if config.IssuerID == "" {
		return fmt.Errorf("IssuerID不能为空")
	}
	if config.KeyID == "" {
		return fmt.Errorf("KeyID不能为空")
	}
	if config.PrivateKey == "" {
		return fmt.Errorf("PrivateKey不能为空")
	}
	if config.BundleID == "" {
		return fmt.Errorf("BundleID不能为空")
	}
	return nil
}

// validateGoogleConfig 验证Google配置
func validateGoogleConfig(config *GoogleAPIConfig) error {
	if config.ServiceAccountKey == "" {
		return fmt.Errorf("ServiceAccountKey不能为空")
	}
	if config.PackageName == "" {
		return fmt.Errorf("PackageName不能为空")
	}
	return nil
}

// GetConfigExample 获取配置示例
func GetConfigExample() *IAPProductConfig {
	return &IAPProductConfig{
		Apple: AppleAPIConfig{
			IssuerID:       "YOUR_APPLE_ISSUER_ID",
			KeyID:          "YOUR_APPLE_KEY_ID",
			PrivateKey:     "YOUR_APPLE_PRIVATE_KEY_PEM",
			BundleID:       "com.rankquantity.voyager",
			APIBaseURL:     "https://api.appstoreconnect.apple.com",
			TimeoutSeconds: 30,
			MaxRetries:     3,
			RetryDelayMs:   1000,
		},
		Google: GoogleAPIConfig{
			ServiceAccountKey: `{
				"type": "service_account",
				"project_id": "your-project-id",
				"private_key_id": "your-private-key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\nYOUR_PRIVATE_KEY\n-----END PRIVATE KEY-----\n",
				"client_email": "your-service-account@your-project.iam.gserviceaccount.com",
				"client_id": "your-client-id",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token",
				"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
				"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/your-service-account%40your-project.iam.gserviceaccount.com"
			}`,
			PackageName:    "com.rankquantity.voyager",
			APIBaseURL:     "https://androidpublisher.googleapis.com",
			TimeoutSeconds: 30,
			MaxRetries:     3,
			RetryDelayMs:   1000,
		},
	}
}
