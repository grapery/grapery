package pay

import (
	"context"
	"encoding/json"
	"time"
)

// IAPProductStatus IAP 产品状态
type IAPProductStatus int

const (
	IAPProductStatusActive   IAPProductStatus = iota + 1 // 上架
	IAPProductStatusInactive                             // 下架
	IAPProductStatusDeleted                              // 删除
)

// IAPProductType IAP 产品类型
type IAPProductType int

const (
	IAPProductTypeSubscription IAPProductType = iota + 1 // 订阅类型
	IAPProductTypeOneTime                                // 一次性购买
	IAPProductTypeConsumable                             // 消耗品
)

// IAPProductPlatform IAP 产品平台
type IAPProductPlatform int

const (
	IAPProductPlatformApple  IAPProductPlatform = iota + 1 // Apple App Store
	IAPProductPlatformGoogle                               // Google Play Store
)

// IAPProduct IAP 产品信息模型
type IAPProduct struct {
	IDBase
	ProductID         string             `gorm:"type:varchar(255);uniqueIndex;not null;comment:产品唯一标识符" json:"product_id"` // 产品唯一标识符
	Platform          IAPProductPlatform `gorm:"type:tinyint;not null;index;comment:平台（Apple/Google）" json:"platform"`     // 平台（Apple/Google）
	ProductType       IAPProductType     `gorm:"type:tinyint;not null;index;comment:产品类型" json:"product_type"`             // 产品类型
	Status            IAPProductStatus   `gorm:"type:tinyint;default:1;index;comment:产品状态" json:"status"`                  // 产品状态
	Name              string             `gorm:"type:varchar(255);not null;comment:产品名称" json:"name"`                      // 产品名称
	Description       string             `gorm:"type:text;comment:产品描述" json:"description"`                                // 产品描述
	Price             float64            `gorm:"type:decimal(10,2);not null;comment:价格" json:"price"`                      // 价格
	Currency          string             `gorm:"type:varchar(10);default:'USD';not null;comment:货币" json:"currency"`       // 货币
	Duration          *string            `gorm:"type:varchar(100);comment:订阅周期（如：P1M、P3M、P1Y）" json:"duration"`            // 订阅周期（增加长度冗余）
	TrialPeriod       *string            `gorm:"type:varchar(100);comment:试用期（如：P7D、P14D）" json:"trial_period"`            // 试用期（增加长度冗余）
	IntroOffer        *string            `gorm:"type:varchar(512);comment:介绍优惠信息" json:"intro_offer"`                      // 介绍优惠信息（增加长度冗余）
	SubscriptionGroup *string            `gorm:"type:varchar(255);index;comment:订阅组（用于升级/降级）" json:"subscription_group"`   // 订阅组（用于升级/降级）
	FamilyShareable   bool               `gorm:"default:false;index;comment:是否支持家庭共享" json:"family_shareable"`             // 是否支持家庭共享
	// 平台特定信息
	AppleSKU        *string `gorm:"type:varchar(255);index;comment:Apple SKU" json:"apple_sku"`          // Apple SKU
	GoogleSKU       *string `gorm:"type:varchar(255);index;comment:Google SKU" json:"google_sku"`        // Google SKU
	AppleProductID  *string `gorm:"type:varchar(255);index;comment:Apple产品ID" json:"apple_product_id"`   // Apple 产品ID
	GoogleProductID *string `gorm:"type:varchar(255);index;comment:Google产品ID" json:"google_product_id"` // Google 产品ID
	// 同步信息
	LastSyncTime *time.Time `gorm:"column:last_sync_time;index;comment:最后同步时间" json:"last_sync_time"`         // 最后同步时间
	SyncStatus   string     `gorm:"type:varchar(50);default:'pending';index;comment:同步状态" json:"sync_status"` // 同步状态
	SyncError    *string    `gorm:"type:text;comment:同步错误信息" json:"sync_error"`                               // 同步错误信息
	// 本地配置
	IsActive     bool   `gorm:"default:true;index;comment:是否激活" json:"is_active"`      // 是否激活
	DisplayOrder int    `gorm:"default:0;index;comment:显示顺序" json:"display_order"`     // 显示顺序
	Featured     bool   `gorm:"default:false;index;comment:是否为推荐产品" json:"featured"`   // 是否为推荐产品
	MaxRoles     int    `gorm:"default:2;not null;comment:最大角色数" json:"max_roles"`     // 最大角色数
	MaxContexts  int    `gorm:"default:5;not null;comment:最大上下文数" json:"max_contexts"` // 最大上下文数
	QuotaLimit   int    `gorm:"default:1000;not null;comment:额度限制" json:"quota_limit"` // 额度限制
	Metadata     string `gorm:"type:json;comment:元数据（JSON格式）" json:"metadata"`         // 元数据（JSON）
}

func (p IAPProduct) TableName() string {
	return "iap_products"
}

// GetMetadata 获取元数据
func (p *IAPProduct) GetMetadata() (map[string]interface{}, error) {
	if p.Metadata == "" {
		return map[string]interface{}{}, nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(p.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
func (p *IAPProduct) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	p.Metadata = string(data)
	return nil
}

// IsAvailable 检查产品是否可用
func (p *IAPProduct) IsAvailable() bool {
	return p.Status == IAPProductStatusActive && p.IsActive
}

// IsSubscription 检查是否为订阅产品
func (p *IAPProduct) IsSubscription() bool {
	return p.ProductType == IAPProductTypeSubscription
}

// GetDurationInDays 获取订阅天数
func (p *IAPProduct) GetDurationInDays() int {
	if p.Duration == nil {
		return 0
	}
	// 简化的周期解析，实际应用中可能需要更复杂的解析逻辑
	switch *p.Duration {
	case "P7D":
		return 7
	case "P14D":
		return 14
	case "P30D", "P1M":
		return 30
	case "P90D", "P3M":
		return 90
	case "P180D", "P6M":
		return 180
	case "P365D", "P1Y":
		return 365
	default:
		return 0
	}
}

// CreateIAPProduct 创建 IAP 产品
func CreateIAPProduct(ctx context.Context, product *IAPProduct) error {
	return DataBase().WithContext(ctx).Create(product).Error
}

// GetIAPProduct 获取 IAP 产品
func GetIAPProduct(ctx context.Context, id uint) (*IAPProduct, error) {
	var product IAPProduct
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetIAPProductByProductID 根据产品ID获取 IAP 产品
func GetIAPProductByProductID(ctx context.Context, productID string) (*IAPProduct, error) {
	var product IAPProduct
	err := DataBase().WithContext(ctx).Where("product_id = ?", productID).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetIAPProductBySKU 根据SKU获取 IAP 产品
func GetIAPProductBySKU(ctx context.Context, sku string, platform IAPProductPlatform) (*IAPProduct, error) {
	var product IAPProduct
	var err error

	if platform == IAPProductPlatformApple {
		err = DataBase().WithContext(ctx).Where("apple_sku = ?", sku).First(&product).Error
	} else {
		err = DataBase().WithContext(ctx).Where("google_sku = ?", sku).First(&product).Error
	}

	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetActiveIAPProducts 获取激活的 IAP 产品
func GetActiveIAPProducts(ctx context.Context, platform IAPProductPlatform) ([]*IAPProduct, error) {
	var products []*IAPProduct
	err := DataBase().WithContext(ctx).
		Where("platform = ? AND status = ? AND is_active = ?", platform, IAPProductStatusActive, true).
		Order("display_order ASC, id ASC").
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetIAPProductsByType 根据类型获取 IAP 产品
func GetIAPProductsByType(ctx context.Context, productType IAPProductType, platform IAPProductPlatform) ([]*IAPProduct, error) {
	var products []*IAPProduct
	err := DataBase().WithContext(ctx).
		Where("platform = ? AND product_type = ? AND status = ? AND is_active = ?", platform, productType, IAPProductStatusActive, true).
		Order("display_order ASC, id ASC").
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetFeaturedIAPProducts 获取推荐的 IAP 产品
func GetFeaturedIAPProducts(ctx context.Context, platform IAPProductPlatform) ([]*IAPProduct, error) {
	var products []*IAPProduct
	err := DataBase().WithContext(ctx).
		Where("platform = ? AND featured = ? AND status = ? AND is_active = ?", platform, true, IAPProductStatusActive, true).
		Order("display_order ASC, id ASC").
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// UpdateIAPProduct 更新 IAP 产品
func UpdateIAPProduct(ctx context.Context, product *IAPProduct) error {
	return DataBase().WithContext(ctx).Save(product).Error
}

// UpdateIAPProductSyncStatus 更新同步状态
func UpdateIAPProductSyncStatus(ctx context.Context, id uint, syncStatus, syncError string) error {
	updates := map[string]interface{}{
		"sync_status":    syncStatus,
		"last_sync_time": time.Now(),
	}

	if syncError != "" {
		updates["sync_error"] = syncError
	}

	return DataBase().WithContext(ctx).
		Model(&IAPProduct{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// DeleteIAPProduct 删除 IAP 产品
func DeleteIAPProduct(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&IAPProduct{}).
		Where("id = ?", id).
		Update("status", IAPProductStatusDeleted).Error
}

// GetIAPProductStats 获取 IAP 产品统计信息
func GetIAPProductStats(ctx context.Context, platform IAPProductPlatform) (map[string]interface{}, error) {
	var stats struct {
		TotalProducts     int64 `json:"total_products"`
		ActiveProducts    int64 `json:"active_products"`
		SubscriptionCount int64 `json:"subscription_count"`
		OneTimeCount      int64 `json:"one_time_count"`
		ConsumableCount   int64 `json:"consumable_count"`
		FeaturedCount     int64 `json:"featured_count"`
	}

	err := DataBase().WithContext(ctx).
		Model(&IAPProduct{}).
		Where("platform = ?", platform).
		Select(`
			COUNT(*) as total_products,
			SUM(CASE WHEN status = ? AND is_active = ? THEN 1 ELSE 0 END) as active_products,
			SUM(CASE WHEN product_type = ? THEN 1 ELSE 0 END) as subscription_count,
			SUM(CASE WHEN product_type = ? THEN 1 ELSE 0 END) as one_time_count,
			SUM(CASE WHEN product_type = ? THEN 1 ELSE 0 END) as consumable_count,
			SUM(CASE WHEN featured = ? THEN 1 ELSE 0 END) as featured_count
		`, IAPProductStatusActive, true, IAPProductTypeSubscription, IAPProductTypeOneTime, IAPProductTypeConsumable, true).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_products":     stats.TotalProducts,
		"active_products":    stats.ActiveProducts,
		"subscription_count": stats.SubscriptionCount,
		"one_time_count":     stats.OneTimeCount,
		"consumable_count":   stats.ConsumableCount,
		"featured_count":     stats.FeaturedCount,
	}, nil
}
