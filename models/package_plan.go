package models

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// PackageType 套餐类型
type PackageType int

const (
	PackageTypeSubscription PackageType = iota + 1 // 订阅类型
	PackageTypeOneTime                             // 一次性购买
	PackageTypeConsumable                          // 消耗品（如积分、额度）
)

// PackageLevel 套餐等级
type PackageLevel int

const (
	PackageLevelBasic      PackageLevel = iota + 1 // 基础套餐
	PackageLevelPremium                            // 高级套餐
	PackageLevelEnterprise                         // 企业套餐
)

// PackageStatus 套餐状态
type PackageStatus int

const (
	PackageStatusActive   PackageStatus = iota + 1 // 上架
	PackageStatusInactive                          // 下架
	PackageStatusDeleted                           // 删除
)

// BillingCycle 计费周期
type BillingCycle string

const (
	BillingCycleMonthly   BillingCycle = "monthly"   // 月付
	BillingCycleQuarterly BillingCycle = "quarterly" // 季付
	BillingCycleYearly    BillingCycle = "yearly"    // 年付
	BillingCycleOneTime   BillingCycle = "one_time"  // 一次性
)

// PackagePlan 套餐计划模型
type PackagePlan struct {
	IDBase
	Name           string        `gorm:"column:name;size:255;not null" json:"name"`                  // 套餐名称
	Description    string        `gorm:"column:description;type:text" json:"description"`            // 套餐描述
	PackageType    PackageType   `gorm:"column:package_type;not null" json:"package_type"`           // 套餐类型
	PackageLevel   PackageLevel  `gorm:"column:package_level;not null" json:"package_level"`         // 套餐等级
	Status         PackageStatus `gorm:"column:status;default:1" json:"status"`                      // 套餐状态
	BillingCycle   BillingCycle  `gorm:"column:billing_cycle;size:20;not null" json:"billing_cycle"` // 计费周期
	Price          int64         `gorm:"column:price;not null" json:"price"`                         // 价格（分）
	OriginalPrice  int64         `gorm:"column:original_price" json:"original_price"`                // 原价（分）
	Currency       string        `gorm:"column:currency;size:10;default:'CNY'" json:"currency"`      // 货币类型
	Duration       int64         `gorm:"column:duration;default:0" json:"duration"`                  // 有效期（秒，0表示永久）
	SKU            string        `gorm:"column:sku;size:100;uniqueIndex" json:"sku"`                 // 套餐SKU
	Category       string        `gorm:"column:category;size:100" json:"category"`                   // 套餐分类
	Tags           string        `gorm:"column:tags;type:text" json:"tags"`                          // 套餐标签（JSON数组）
	ImageURL       string        `gorm:"column:image_url;size:500" json:"image_url"`                 // 套餐图片URL
	SortOrder      int           `gorm:"column:sort_order;default:0" json:"sort_order"`              // 排序
	IsHot          bool          `gorm:"column:is_hot;default:false" json:"is_hot"`                  // 是否热门
	IsRecommend    bool          `gorm:"column:is_recommend;default:false" json:"is_recommend"`      // 是否推荐
	IsPopular      bool          `gorm:"column:is_popular;default:false" json:"is_popular"`          // 是否畅销
	StartTime      *time.Time    `gorm:"column:start_time" json:"start_time"`                        // 上架时间
	EndTime        *time.Time    `gorm:"column:end_time" json:"end_time"`                            // 下架时间
	FreeTrialDays  int           `gorm:"column:free_trial_days;default:0" json:"free_trial_days"`    // 免费试用天数
	RefundPolicy   string        `gorm:"column:refund_policy;type:text" json:"refund_policy"`        // 退款政策
	TermsOfService string        `gorm:"column:terms_of_service;type:text" json:"terms_of_service"`  // 服务条款
	CreatedBy      int64         `gorm:"column:created_by;not null" json:"created_by"`               // 创建者ID
	UpdatedBy      int64         `gorm:"column:updated_by" json:"updated_by"`                        // 更新者ID

	// 服务能力配置
	QuotaLimit      int    `gorm:"column:quota_limit;default:1000" json:"quota_limit"`        // 额度限制
	MaxRoles        int    `gorm:"column:max_roles;default:2" json:"max_roles"`               // 最大角色数
	MaxContexts     int    `gorm:"column:max_contexts;default:5" json:"max_contexts"`         // 最大上下文数
	AvailableModels string `gorm:"column:available_models;type:text" json:"available_models"` // 可用模型（JSON数组）
	Features        string `gorm:"column:features;type:text" json:"features"`                 // 功能特性（JSON对象）

	// 统计信息
	SoldCount   int `gorm:"column:sold_count;default:0" json:"sold_count"`     // 销售数量
	ViewCount   int `gorm:"column:view_count;default:0" json:"view_count"`     // 浏览次数
	ActiveCount int `gorm:"column:active_count;default:0" json:"active_count"` // 活跃用户数
}

func (p PackagePlan) TableName() string {
	return "package_plans"
}

// GetAvailableModels 获取可用模型列表
func (p *PackagePlan) GetAvailableModels() ([]string, error) {
	if p.AvailableModels == "" {
		return []string{"gpt-3.5-turbo"}, nil // 默认模型
	}
	var models []string
	err := json.Unmarshal([]byte(p.AvailableModels), &models)
	return models, err
}

// SetAvailableModels 设置可用模型列表
func (p *PackagePlan) SetAvailableModels(models []string) error {
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	p.AvailableModels = string(data)
	return nil
}

// GetFeatures 获取功能特性
func (p *PackagePlan) GetFeatures() (map[string]interface{}, error) {
	if p.Features == "" {
		return map[string]interface{}{}, nil
	}
	var features map[string]interface{}
	err := json.Unmarshal([]byte(p.Features), &features)
	return features, err
}

// SetFeatures 设置功能特性
func (p *PackagePlan) SetFeatures(features map[string]interface{}) error {
	data, err := json.Marshal(features)
	if err != nil {
		return err
	}
	p.Features = string(data)
	return nil
}

// GetTags 获取套餐标签
func (p *PackagePlan) GetTags() ([]string, error) {
	if p.Tags == "" {
		return []string{}, nil
	}
	var tags []string
	err := json.Unmarshal([]byte(p.Tags), &tags)
	return tags, err
}

// SetTags 设置套餐标签
func (p *PackagePlan) SetTags(tags []string) error {
	data, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	p.Tags = string(data)
	return nil
}

// IsActive 检查套餐是否活跃
func (p *PackagePlan) IsActive() bool {
	if p.Status != PackageStatusActive {
		return false
	}

	now := time.Now()
	if p.StartTime != nil && now.Before(*p.StartTime) {
		return false
	}
	if p.EndTime != nil && now.After(*p.EndTime) {
		return false
	}

	return true
}

// GetDurationInDays 获取套餐天数
func (p *PackagePlan) GetDurationInDays() int {
	if p.Duration == 0 {
		return 0 // 永久
	}
	return int(p.Duration / 86400) // 转换为天
}

// CreatePackagePlan 创建套餐计划
func CreatePackagePlan(ctx context.Context, plan *PackagePlan) error {
	return DataBase().WithContext(ctx).Create(plan).Error
}

// GetPackagePlan 获取套餐计划
func GetPackagePlan(ctx context.Context, id uint) (*PackagePlan, error) {
	var plan PackagePlan
	err := DataBase().WithContext(ctx).Where("id = ? AND status != ?", id, PackageStatusDeleted).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// GetPackagePlanBySKU 根据SKU获取套餐计划
func GetPackagePlanBySKU(ctx context.Context, sku string) (*PackagePlan, error) {
	var plan PackagePlan
	err := DataBase().WithContext(ctx).Where("sku = ? AND status != ?", sku, PackageStatusDeleted).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// GetActivePackagePlans 获取所有活跃套餐计划
func GetActivePackagePlans(ctx context.Context) ([]*PackagePlan, error) {
	var plans []*PackagePlan
	err := DataBase().WithContext(ctx).
		Where("status = ?", PackageStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetPackagePlansByType 根据类型获取套餐计划
func GetPackagePlansByType(ctx context.Context, packageType PackageType) ([]*PackagePlan, error) {
	var plans []*PackagePlan
	err := DataBase().WithContext(ctx).
		Where("package_type = ? AND status = ?", packageType, PackageStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetPackagePlansByLevel 根据等级获取套餐计划
func GetPackagePlansByLevel(ctx context.Context, level PackageLevel) ([]*PackagePlan, error) {
	var plans []*PackagePlan
	err := DataBase().WithContext(ctx).
		Where("package_level = ? AND status = ?", level, PackageStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetPackagePlansByBillingCycle 根据计费周期获取套餐计划
func GetPackagePlansByBillingCycle(ctx context.Context, billingCycle BillingCycle) ([]*PackagePlan, error) {
	var plans []*PackagePlan
	err := DataBase().WithContext(ctx).
		Where("billing_cycle = ? AND status = ?", billingCycle, PackageStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetPackagePlansByCategory 根据分类获取套餐计划
func GetPackagePlansByCategory(ctx context.Context, category string) ([]*PackagePlan, error) {
	var plans []*PackagePlan
	err := DataBase().WithContext(ctx).
		Where("category = ? AND status = ?", category, PackageStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetHotPackagePlans 获取热门套餐计划
func GetHotPackagePlans(ctx context.Context, limit int) ([]*PackagePlan, error) {
	var plans []*PackagePlan
	err := DataBase().WithContext(ctx).
		Where("is_hot = ? AND status = ?", true, PackageStatusActive).
		Order("sold_count DESC, view_count DESC").
		Limit(limit).
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetRecommendPackagePlans 获取推荐套餐计划
func GetRecommendPackagePlans(ctx context.Context, limit int) ([]*PackagePlan, error) {
	var plans []*PackagePlan
	err := DataBase().WithContext(ctx).
		Where("is_recommend = ? AND status = ?", true, PackageStatusActive).
		Order("sort_order ASC, id ASC").
		Limit(limit).
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetPopularPackagePlans 获取畅销套餐计划
func GetPopularPackagePlans(ctx context.Context, limit int) ([]*PackagePlan, error) {
	var plans []*PackagePlan
	err := DataBase().WithContext(ctx).
		Where("is_popular = ? AND status = ?", true, PackageStatusActive).
		Order("sold_count DESC, active_count DESC").
		Limit(limit).
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// UpdatePackagePlan 更新套餐计划
func UpdatePackagePlan(ctx context.Context, plan *PackagePlan) error {
	return DataBase().WithContext(ctx).Save(plan).Error
}

// DeletePackagePlan 删除套餐计划（软删除）
func DeletePackagePlan(ctx context.Context, id uint, updatedBy int64) error {
	return DataBase().WithContext(ctx).
		Model(&PackagePlan{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     PackageStatusDeleted,
			"updated_by": updatedBy,
		}).Error
}

// IncrementPackagePlanSoldCount 增加套餐销售数量
func IncrementPackagePlanSoldCount(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&PackagePlan{}).
		Where("id = ?", id).
		Update("sold_count", gorm.Expr("sold_count + ?", 1)).Error
}

// IncrementPackagePlanViewCount 增加套餐浏览次数
func IncrementPackagePlanViewCount(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&PackagePlan{}).
		Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// IncrementActiveCount 增加活跃用户数
func IncrementActiveCount(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&PackagePlan{}).
		Where("id = ?", id).
		Update("active_count", gorm.Expr("active_count + ?", 1)).Error
}

// DecrementActiveCount 减少活跃用户数
func DecrementActiveCount(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&PackagePlan{}).
		Where("id = ?", id).
		Update("active_count", gorm.Expr("active_count - ?", 1)).Error
}

// GetPackagePlanStats 获取套餐计划统计信息
func GetPackagePlanStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalPlans     int64 `json:"total_plans"`
		ActivePlans    int64 `json:"active_plans"`
		HotPlans       int64 `json:"hot_plans"`
		RecommendPlans int64 `json:"recommend_plans"`
		PopularPlans   int64 `json:"popular_plans"`
		TotalSold      int64 `json:"total_sold"`
		TotalViews     int64 `json:"total_views"`
		TotalActive    int64 `json:"total_active"`
	}

	err := DataBase().WithContext(ctx).
		Model(&PackagePlan{}).
		Select(`
			COUNT(*) as total_plans,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as active_plans,
			SUM(CASE WHEN is_hot = 1 THEN 1 ELSE 0 END) as hot_plans,
			SUM(CASE WHEN is_recommend = 1 THEN 1 ELSE 0 END) as recommend_plans,
			SUM(CASE WHEN is_popular = 1 THEN 1 ELSE 0 END) as popular_plans,
			SUM(sold_count) as total_sold,
			SUM(view_count) as total_views,
			SUM(active_count) as total_active
		`).
		Where("status != ?", PackageStatusDeleted).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_plans":     stats.TotalPlans,
		"active_plans":    stats.ActivePlans,
		"hot_plans":       stats.HotPlans,
		"recommend_plans": stats.RecommendPlans,
		"popular_plans":   stats.PopularPlans,
		"total_sold":      stats.TotalSold,
		"total_views":     stats.TotalViews,
		"total_active":    stats.TotalActive,
	}, nil
}
