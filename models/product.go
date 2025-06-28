package models

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ProductType 商品类型
type ProductType int

const (
	ProductTypeSubscription ProductType = iota + 1 // 订阅类型
	ProductTypeOneTime                             // 一次性购买
	ProductTypeConsumable                          // 消耗品（如积分、额度）
)

// ProductStatus 商品状态
type ProductStatus int

const (
	ProductStatusActive   ProductStatus = iota + 1 // 上架
	ProductStatusInactive                          // 下架
	ProductStatusDeleted                           // 删除
)

// Product 商品/产品模型
type Product struct {
	IDBase
	Name            string        `gorm:"column:name;size:255;not null" json:"name"`                 // 商品名称
	Description     string        `gorm:"column:description;type:text" json:"description"`           // 商品描述
	ProductType     ProductType   `gorm:"column:product_type;not null" json:"product_type"`          // 商品类型
	Status          ProductStatus `gorm:"column:status;default:1" json:"status"`                     // 商品状态
	Price           int64         `gorm:"column:price;not null" json:"price"`                        // 价格（分）
	Currency        string        `gorm:"column:currency;size:10;default:'CNY'" json:"currency"`     // 货币类型
	Duration        int64         `gorm:"column:duration;default:0" json:"duration"`                 // 有效期（秒，0表示永久）
	MaxRoles        int           `gorm:"column:max_roles;default:2" json:"max_roles"`               // 最大角色数
	MaxContexts     int           `gorm:"column:max_contexts;default:5" json:"max_contexts"`         // 最大上下文数
	QuotaLimit      int           `gorm:"column:quota_limit;default:1000" json:"quota_limit"`        // 额度限制
	AvailableModels string        `gorm:"column:available_models;type:text" json:"available_models"` // 可用模型（JSON数组）
	Features        string        `gorm:"column:features;type:text" json:"features"`                 // 功能特性（JSON对象）
	SortOrder       int           `gorm:"column:sort_order;default:0" json:"sort_order"`             // 排序
	CreatedBy       int64         `gorm:"column:created_by;not null" json:"created_by"`              // 创建者ID
	UpdatedBy       int64         `gorm:"column:updated_by" json:"updated_by"`                       // 更新者ID
	// 新增字段
	SKU            string     `gorm:"column:sku;size:100;uniqueIndex" json:"sku"`                // 商品SKU
	Category       string     `gorm:"column:category;size:100" json:"category"`                  // 商品分类
	Tags           string     `gorm:"column:tags;type:text" json:"tags"`                         // 商品标签（JSON数组）
	ImageURL       string     `gorm:"column:image_url;size:500" json:"image_url"`                // 商品图片URL
	Stock          int        `gorm:"column:stock;default:-1" json:"stock"`                      // 库存（-1表示无限制）
	SoldCount      int        `gorm:"column:sold_count;default:0" json:"sold_count"`             // 销售数量
	ViewCount      int        `gorm:"column:view_count;default:0" json:"view_count"`             // 浏览次数
	IsHot          bool       `gorm:"column:is_hot;default:false" json:"is_hot"`                 // 是否热门
	IsRecommend    bool       `gorm:"column:is_recommend;default:false" json:"is_recommend"`     // 是否推荐
	StartTime      *time.Time `gorm:"column:start_time" json:"start_time"`                       // 上架时间
	EndTime        *time.Time `gorm:"column:end_time" json:"end_time"`                           // 下架时间
	BillingCycle   string     `gorm:"column:billing_cycle;size:20" json:"billing_cycle"`         // 计费周期（monthly, yearly等）
	FreeTrialDays  int        `gorm:"column:free_trial_days;default:0" json:"free_trial_days"`   // 免费试用天数
	RefundPolicy   string     `gorm:"column:refund_policy;type:text" json:"refund_policy"`       // 退款政策
	TermsOfService string     `gorm:"column:terms_of_service;type:text" json:"terms_of_service"` // 服务条款
}

// ProductSKU 商品SKU模型
type ProductSKU struct {
	IDBase
	ProductID   uint   `gorm:"column:product_id;not null;index" json:"product_id"`    // 商品ID
	SKU         string `gorm:"column:sku;size:100;uniqueIndex" json:"sku"`            // SKU编码
	Name        string `gorm:"column:name;size:255;not null" json:"name"`             // SKU名称
	Description string `gorm:"column:description;type:text" json:"description"`       // SKU描述
	Price       int64  `gorm:"column:price;not null" json:"price"`                    // 价格（分）
	Currency    string `gorm:"column:currency;size:10;default:'CNY'" json:"currency"` // 货币类型
	Stock       int    `gorm:"column:stock;default:-1" json:"stock"`                  // 库存（-1表示无限制）
	Attributes  string `gorm:"column:attributes;type:text" json:"attributes"`         // 属性（JSON对象）
	Status      int    `gorm:"column:status;default:1" json:"status"`                 // 状态（1:启用 0:禁用）
	SortOrder   int    `gorm:"column:sort_order;default:0" json:"sort_order"`         // 排序
}

func (p Product) TableName() string {
	return "products"
}

func (ps ProductSKU) TableName() string {
	return "product_skus"
}

// GetAvailableModels 获取可用模型列表
func (p *Product) GetAvailableModels() ([]string, error) {
	if p.AvailableModels == "" {
		return []string{}, nil
	}
	var models []string
	err := json.Unmarshal([]byte(p.AvailableModels), &models)
	return models, err
}

// SetAvailableModels 设置可用模型列表
func (p *Product) SetAvailableModels(models []string) error {
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	p.AvailableModels = string(data)
	return nil
}

// GetFeatures 获取功能特性
func (p *Product) GetFeatures() (map[string]interface{}, error) {
	if p.Features == "" {
		return map[string]interface{}{}, nil
	}
	var features map[string]interface{}
	err := json.Unmarshal([]byte(p.Features), &features)
	return features, err
}

// SetFeatures 设置功能特性
func (p *Product) SetFeatures(features map[string]interface{}) error {
	data, err := json.Marshal(features)
	if err != nil {
		return err
	}
	p.Features = string(data)
	return nil
}

// GetTags 获取商品标签
func (p *Product) GetTags() ([]string, error) {
	if p.Tags == "" {
		return []string{}, nil
	}
	var tags []string
	err := json.Unmarshal([]byte(p.Tags), &tags)
	return tags, err
}

// SetTags 设置商品标签
func (p *Product) SetTags(tags []string) error {
	data, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	p.Tags = string(data)
	return nil
}

// CreateProduct 创建商品
func CreateProduct(ctx context.Context, product *Product) error {
	return DataBase().WithContext(ctx).Create(product).Error
}

// GetProduct 获取商品信息
func GetProduct(ctx context.Context, id uint) (*Product, error) {
	var product Product
	err := DataBase().WithContext(ctx).Where("id = ? AND status != ?", id, ProductStatusDeleted).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetProductBySKU 根据SKU获取商品
func GetProductBySKU(ctx context.Context, sku string) (*Product, error) {
	var product Product
	err := DataBase().WithContext(ctx).Where("sku = ? AND status != ?", sku, ProductStatusDeleted).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetActiveProducts 获取所有上架商品
func GetActiveProducts(ctx context.Context) ([]*Product, error) {
	var products []*Product
	err := DataBase().WithContext(ctx).
		Where("status = ?", ProductStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetProductsByType 根据类型获取商品
func GetProductsByType(ctx context.Context, productType ProductType) ([]*Product, error) {
	var products []*Product
	err := DataBase().WithContext(ctx).
		Where("product_type = ? AND status = ?", productType, ProductStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetProductsByCategory 根据分类获取商品
func GetProductsByCategory(ctx context.Context, category string) ([]*Product, error) {
	var products []*Product
	err := DataBase().WithContext(ctx).
		Where("category = ? AND status = ?", category, ProductStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetHotProducts 获取热门商品
func GetHotProducts(ctx context.Context, limit int) ([]*Product, error) {
	var products []*Product
	err := DataBase().WithContext(ctx).
		Where("is_hot = ? AND status = ?", true, ProductStatusActive).
		Order("sold_count DESC, view_count DESC").
		Limit(limit).
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetRecommendProducts 获取推荐商品
func GetRecommendProducts(ctx context.Context, limit int) ([]*Product, error) {
	var products []*Product
	err := DataBase().WithContext(ctx).
		Where("is_recommend = ? AND status = ?", true, ProductStatusActive).
		Order("sort_order ASC, id ASC").
		Limit(limit).
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// UpdateProduct 更新商品
func UpdateProduct(ctx context.Context, product *Product) error {
	return DataBase().WithContext(ctx).Save(product).Error
}

// DeleteProduct 删除商品（软删除）
func DeleteProduct(ctx context.Context, id uint, updatedBy int64) error {
	return DataBase().WithContext(ctx).
		Model(&Product{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     ProductStatusDeleted,
			"updated_by": updatedBy,
		}).Error
}

// IncrementSoldCount 增加销售数量
func IncrementSoldCount(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&Product{}).
		Where("id = ?", id).
		UpdateColumn("sold_count", gorm.Expr("sold_count + ?", 1)).Error
}

// IncrementViewCount 增加浏览次数
func IncrementViewCount(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&Product{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// CheckStock 检查库存
func (p *Product) CheckStock(quantity int) bool {
	if p.Stock == -1 {
		return true // 无限制
	}
	return p.Stock >= quantity
}

// DecreaseStock 减少库存
func DecreaseStock(ctx context.Context, id uint, quantity int) error {
	return DataBase().WithContext(ctx).
		Model(&Product{}).
		Where("id = ? AND (stock = -1 OR stock >= ?)", id, quantity).
		UpdateColumn("stock", gorm.Expr("CASE WHEN stock = -1 THEN -1 ELSE stock - ? END", quantity)).Error
}

// IncreaseStock 增加库存
func IncreaseStock(ctx context.Context, id uint, quantity int) error {
	return DataBase().WithContext(ctx).
		Model(&Product{}).
		Where("id = ?", id).
		UpdateColumn("stock", gorm.Expr("CASE WHEN stock = -1 THEN -1 ELSE stock + ? END", quantity)).Error
}

// ProductSKU相关方法
func CreateProductSKU(ctx context.Context, sku *ProductSKU) error {
	return DataBase().WithContext(ctx).Create(sku).Error
}

func GetProductSKU(ctx context.Context, id uint) (*ProductSKU, error) {
	var sku ProductSKU
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&sku).Error
	if err != nil {
		return nil, err
	}
	return &sku, nil
}

func GetProductSKUBySKU(ctx context.Context, skuCode string) (*ProductSKU, error) {
	var sku ProductSKU
	err := DataBase().WithContext(ctx).Where("sku = ?", skuCode).First(&sku).Error
	if err != nil {
		return nil, err
	}
	return &sku, nil
}

func GetProductSKUs(ctx context.Context, productID uint) ([]*ProductSKU, error) {
	var skus []*ProductSKU
	err := DataBase().WithContext(ctx).
		Where("product_id = ? AND status = ?", productID, 1).
		Order("sort_order ASC, id ASC").
		Find(&skus).Error
	if err != nil {
		return nil, err
	}
	return skus, nil
}

func UpdateProductSKU(ctx context.Context, sku *ProductSKU) error {
	return DataBase().WithContext(ctx).Save(sku).Error
}

func DeleteProductSKU(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).Delete(&ProductSKU{}, id).Error
}

// GetAttributes 获取SKU属性
func (ps *ProductSKU) GetAttributes() (map[string]interface{}, error) {
	if ps.Attributes == "" {
		return map[string]interface{}{}, nil
	}
	var attributes map[string]interface{}
	err := json.Unmarshal([]byte(ps.Attributes), &attributes)
	return attributes, err
}

// SetAttributes 设置SKU属性
func (ps *ProductSKU) SetAttributes(attributes map[string]interface{}) error {
	data, err := json.Marshal(attributes)
	if err != nil {
		return err
	}
	ps.Attributes = string(data)
	return nil
}
