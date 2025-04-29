package models

import (
	"context"
	"time"
)

type OrderStatus int

const (
	OrderStatusPending OrderStatus = iota
	OrderStatusPaid
	OrderStatusCanceled
	OrderStatusRefunded
)

type Order struct {
	IDBase
	UserID      int64       `json:"user_id" gorm:"index"`
	Amount      float64     `json:"amount"`
	Status      OrderStatus `json:"status"`
	PaymentTime *time.Time  `json:"payment_time"`
	Description string      `json:"description"`
	Metadata    string      `json:"metadata"` // JSON string for additional data
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
