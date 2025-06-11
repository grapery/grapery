package models

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type OrderStatus int

const (
	OrderStatusPending OrderStatus = iota
	OrderStatusPaid
	OrderStatusCanceled
	OrderStatusRefunded
)

// Order 订单/交易记录
type Order struct {
	IDBase
	UserID      int64      `gorm:"column:user_id" json:"user_id,omitempty"`       // 用户ID
	ProductID   int64      `gorm:"column:product_id" json:"product_id,omitempty"` // 商品ID
	Amount      int64      `gorm:"column:amount" json:"amount,omitempty"`         // 金额
	Status      int        `gorm:"column:status" json:"status,omitempty"`         // 状态
	OrderNo     string     `gorm:"column:order_no" json:"order_no,omitempty"`     // 订单号
	PaymentTime *time.Time `json:"payment_time"`
	Description string     `json:"description"`
	Metadata    string     `json:"metadata"` // JSON string for additional data
}

func (o Order) TableName() string {
	return "orders"
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

func UpdateOrderStatus(ctx context.Context, id uint, status OrderStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if status == OrderStatusPaid {
		now := time.Now()
		updates["payment_time"] = &now
	}
	return DataBase().WithContext(ctx).
		Model(&Order{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// 新增：分页获取Order列表
func GetOrderList(ctx context.Context, offset, limit int) ([]*Order, error) {
	var orders []*Order
	err := DataBase().Model(&Order{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&orders).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return orders, nil
}

// 新增：通过订单号唯一查询
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
