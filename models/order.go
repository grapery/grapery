package models

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type OrderStatus int

const (
	OrderStatusPending OrderStatus = iota
	OrderStatusPaid
	OrderStatusCanceled
	OrderStatusRefunded
	OrderStatusPartialRefunded // 部分退款
	OrderStatusFailed          // 支付失败
	OrderStatusExpired         // 订单过期
	OrderStatusProcessing      // 处理中
)

// Order 订单/交易记录
type Order struct {
	IDBase
	UserID      int64      `gorm:"column:user_id" json:"user_id,omitempty"`       // 用户ID
	ProductID   int64      `gorm:"column:product_id" json:"product_id,omitempty"` // 商品ID
	SKUID       *uint      `gorm:"column:sku_id" json:"sku_id,omitempty"`         // SKU ID（可选）
	Amount      int64      `gorm:"column:amount" json:"amount,omitempty"`         // 金额
	Status      int        `gorm:"column:status" json:"status,omitempty"`         // 状态
	OrderNo     string     `gorm:"column:order_no" json:"order_no,omitempty"`     // 订单号
	PaymentTime *time.Time `json:"payment_time"`                                  // 支付时间
	Description string     `json:"description"`                                   // 订单描述
	Metadata    string     `json:"metadata"`                                      // JSON string for additional data
	// 新增字段
	Currency     string     `gorm:"column:currency;size:10;default:'CNY'" json:"currency"` // 货币类型
	Quantity     int        `gorm:"column:quantity;default:1" json:"quantity"`             // 数量
	UnitPrice    int64      `gorm:"column:unit_price" json:"unit_price"`                   // 单价（分）
	Discount     int64      `gorm:"column:discount;default:0" json:"discount"`             // 折扣金额（分）
	Tax          int64      `gorm:"column:tax;default:0" json:"tax"`                       // 税费（分）
	ShippingFee  int64      `gorm:"column:shipping_fee;default:0" json:"shipping_fee"`     // 运费（分）
	TotalAmount  int64      `gorm:"column:total_amount" json:"total_amount"`               // 总金额（分）
	RefundAmount int64      `gorm:"column:refund_amount;default:0" json:"refund_amount"`   // 退款金额（分）
	RefundTime   *time.Time `json:"refund_time"`                                           // 退款时间
	RefundReason string     `gorm:"column:refund_reason;size:255" json:"refund_reason"`    // 退款原因
	ExpireTime   *time.Time `json:"expire_time"`                                           // 订单过期时间
	CancelTime   *time.Time `json:"cancel_time"`                                           // 取消时间
	CancelReason string     `gorm:"column:cancel_reason;size:255" json:"cancel_reason"`    // 取消原因
	IPAddress    string     `gorm:"column:ip_address;size:45" json:"ip_address"`           // 下单IP地址
	UserAgent    string     `gorm:"column:user_agent;size:500" json:"user_agent"`          // 用户代理
	Source       string     `gorm:"column:source;size:50" json:"source"`                   // 订单来源（web, app, api等）
	Channel      string     `gorm:"column:channel;size:50" json:"channel"`                 // 推广渠道
	PromoCode    string     `gorm:"column:promo_code;size:50" json:"promo_code"`           // 优惠码
	Notes        string     `gorm:"column:notes;type:text" json:"notes"`                   // 订单备注
}

// OrderItem 订单项
type OrderItem struct {
	IDBase
	OrderID     uint   `gorm:"column:order_id;not null;index" json:"order_id"`   // 订单ID
	ProductID   uint   `gorm:"column:product_id;not null" json:"product_id"`     // 商品ID
	SKUID       *uint  `gorm:"column:sku_id" json:"sku_id"`                      // SKU ID（可选）
	ProductName string `gorm:"column:product_name;size:255" json:"product_name"` // 商品名称
	SKUName     string `gorm:"column:sku_name;size:255" json:"sku_name"`         // SKU名称
	Quantity    int    `gorm:"column:quantity;default:1" json:"quantity"`        // 数量
	UnitPrice   int64  `gorm:"column:unit_price" json:"unit_price"`              // 单价（分）
	TotalPrice  int64  `gorm:"column:total_price" json:"total_price"`            // 总价（分）
	Attributes  string `gorm:"column:attributes;type:text" json:"attributes"`    // 属性（JSON对象）
}

func (o Order) TableName() string {
	return "orders"
}

func (oi OrderItem) TableName() string {
	return "order_items"
}

// GetMetadata 获取元数据
func (o *Order) GetMetadata() (map[string]interface{}, error) {
	if o.Metadata == "" {
		return map[string]interface{}{}, nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(o.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
func (o *Order) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	o.Metadata = string(data)
	return nil
}

// GetAttributes 获取订单项属性
func (oi *OrderItem) GetAttributes() (map[string]interface{}, error) {
	if oi.Attributes == "" {
		return map[string]interface{}{}, nil
	}
	var attributes map[string]interface{}
	err := json.Unmarshal([]byte(oi.Attributes), &attributes)
	return attributes, err
}

// SetAttributes 设置订单项属性
func (oi *OrderItem) SetAttributes(attributes map[string]interface{}) error {
	data, err := json.Marshal(attributes)
	if err != nil {
		return err
	}
	oi.Attributes = string(data)
	return nil
}

// CalculateTotal 计算订单总金额
func (o *Order) CalculateTotal() int64 {
	total := o.UnitPrice * int64(o.Quantity)
	total += o.Tax + o.ShippingFee
	total -= o.Discount
	return total
}

func CreateOrder(ctx context.Context, order *Order) error {
	return DataBase().WithContext(ctx).Create(order).Error
}

func GetOrder(ctx context.Context, id uint) (*Order, error) {
	var order Order
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func GetOrderByOrderNo(ctx context.Context, orderNo string) (*Order, error) {
	order := &Order{}
	err := DataBase().Model(order).
		WithContext(ctx).
		Where("order_no = ?", orderNo).
		First(order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

func GetUserOrders(ctx context.Context, userID int64, offset, limit int) ([]*Order, error) {
	var orders []*Order
	err := DataBase().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func GetUserOrdersByStatus(ctx context.Context, userID int64, status OrderStatus, offset, limit int) ([]*Order, error) {
	var orders []*Order
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, status).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func GetExpiredOrders(ctx context.Context) ([]*Order, error) {
	var orders []*Order
	err := DataBase().WithContext(ctx).
		Where("status = ? AND expire_time <= ?", OrderStatusPending, time.Now()).
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func UpdateOrderStatus(ctx context.Context, id uint, status OrderStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	now := time.Now()
	switch status {
	case OrderStatusPaid:
		updates["payment_time"] = &now
	case OrderStatusCanceled:
		updates["cancel_time"] = &now
	case OrderStatusRefunded, OrderStatusPartialRefunded:
		updates["refund_time"] = &now
	}

	return DataBase().WithContext(ctx).
		Model(&Order{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func UpdateOrderRefund(ctx context.Context, id uint, refundAmount int64, reason string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"refund_amount": refundAmount,
		"refund_reason": reason,
		"refund_time":   &now,
	}

	// 判断是否为全额退款
	order, err := GetOrder(ctx, id)
	if err != nil {
		return err
	}

	if refundAmount >= order.TotalAmount {
		updates["status"] = OrderStatusRefunded
	} else {
		updates["status"] = OrderStatusPartialRefunded
	}

	return DataBase().WithContext(ctx).
		Model(&Order{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func CancelOrder(ctx context.Context, id uint, reason string) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&Order{}).
		Where("id = ? AND status = ?", id, OrderStatusPending).
		Updates(map[string]interface{}{
			"status":        OrderStatusCanceled,
			"cancel_time":   &now,
			"cancel_reason": reason,
		}).Error
}

// 新增：分页获取Order列表
func GetOrderList(ctx context.Context, offset, limit int) ([]*Order, error) {
	var orders []*Order
	err := DataBase().Model(&Order{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at desc").
		Find(&orders).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return orders, nil
}

// 新增：根据状态获取订单列表
func GetOrderListByStatus(ctx context.Context, status OrderStatus, offset, limit int) ([]*Order, error) {
	var orders []*Order
	err := DataBase().Model(&Order{}).
		WithContext(ctx).
		Where("status = ?", status).
		Offset(offset).
		Limit(limit).
		Order("created_at desc").
		Find(&orders).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return orders, nil
}

// 新增：获取订单统计信息
func GetOrderStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	var stats struct {
		TotalOrders    int64 `json:"total_orders"`
		PendingOrders  int64 `json:"pending_orders"`
		PaidOrders     int64 `json:"paid_orders"`
		CanceledOrders int64 `json:"canceled_orders"`
		TotalAmount    int64 `json:"total_amount"`
	}

	err := DataBase().WithContext(ctx).
		Model(&Order{}).
		Where("user_id = ?", userID).
		Select(`
			COUNT(*) as total_orders,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as pending_orders,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as paid_orders,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as canceled_orders,
			SUM(CASE WHEN status = ? THEN total_amount ELSE 0 END) as total_amount
		`, OrderStatusPending, OrderStatusPaid, OrderStatusCanceled, OrderStatusPaid).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_orders":    stats.TotalOrders,
		"pending_orders":  stats.PendingOrders,
		"paid_orders":     stats.PaidOrders,
		"canceled_orders": stats.CanceledOrders,
		"total_amount":    stats.TotalAmount,
	}, nil
}

// OrderItem相关方法
func CreateOrderItem(ctx context.Context, item *OrderItem) error {
	return DataBase().WithContext(ctx).Create(item).Error
}

func GetOrderItems(ctx context.Context, orderID uint) ([]*OrderItem, error) {
	var items []*OrderItem
	err := DataBase().WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func GetOrderItem(ctx context.Context, id uint) (*OrderItem, error) {
	var item OrderItem
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func UpdateOrderItem(ctx context.Context, item *OrderItem) error {
	return DataBase().WithContext(ctx).Save(item).Error
}

func DeleteOrderItem(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).Delete(&OrderItem{}, id).Error
}
