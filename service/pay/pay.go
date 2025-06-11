package pay

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/grapery/grapery/models"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/refund"
)

var (
	ErrInvalidAmount = errors.New("invalid amount")
	ErrOrderNotFound = errors.New("order not found")
)

type Order struct {
	Id        string  `json:"id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
}

type PayService interface {
	// 支付服务接口
	Pay(userId, orderId string, amount float64) error
	// 退款服务接口
	Refund(userId, orderId string, amount float64) error
	// 获取用户余额
	GetBalance(userId string) (float64, error)
	// 获取用户订单
	GetOrder(userId, orderId string) (Order, error)
	// 获取用户所有订单
	GetOrders(userId string) ([]Order, error)
	// 判断用户是否为vip
	IsVip(userId string) (bool, error)
	// 设置用户保证金
	SetBond(userId string, bond float64, storyId int64) error
}

var _ PayService = &PayServiceImpl{}

type PayServiceImpl struct {
	stripeKey string
}

func NewPayService(stripeKey string) *PayServiceImpl {
	stripe.Key = stripeKey
	return &PayServiceImpl{
		stripeKey: stripeKey,
	}
}

func (s *PayServiceImpl) Pay(userId, orderId string, amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// Create a payment intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount * 100)), // Convert to cents
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return err
	}

	// Convert userId to int64
	userID, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		return err
	}

	// Create order record
	order := &models.Order{
		UserID:      userID,
		Amount:      int64(amount),
		Status:      int(models.OrderStatusPending),
		Description: "Payment for order " + orderId,
		Metadata:    pi.ID,
	}

	return models.CreateOrder(context.Background(), order)
}

func (s *PayServiceImpl) Refund(userId, orderId string, amount float64) error {
	// Convert orderId to uint
	orderID, err := strconv.ParseUint(orderId, 10, 32)
	if err != nil {
		return err
	}

	// Get the order
	order, err := models.GetOrder(context.Background(), uint(orderID))
	if err != nil {
		return ErrOrderNotFound
	}

	if order.Status != int(models.OrderStatusPaid) {
		return errors.New("order is not paid")
	}

	// Create refund
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(order.Metadata),
		Amount:        stripe.Int64(int64(amount * 100)),
	}

	_, err = refund.New(params)
	if err != nil {
		return err
	}

	// Update order status
	return models.UpdateOrderStatus(context.Background(), order.ID, models.OrderStatusRefunded)
}

func (s *PayServiceImpl) GetBalance(userId string) (float64, error) {
	// In a real implementation, this would query the user's balance from a database
	// For now, return a mock value
	return 0, nil
}

func (s *PayServiceImpl) GetOrder(userId, orderId string) (Order, error) {
	// Convert orderId to uint
	orderID, err := strconv.ParseUint(orderId, 10, 32)
	if err != nil {
		return Order{}, err
	}

	order, err := models.GetOrder(context.Background(), uint(orderID))
	if err != nil {
		return Order{}, err
	}

	return Order{
		Id:        orderId,
		Amount:    float64(order.Amount),
		Status:    string(order.Status),
		CreatedAt: order.CreateAt.Unix(),
		UpdatedAt: order.UpdateAt.Unix(),
	}, nil
}

func (s *PayServiceImpl) GetOrders(userId string) ([]Order, error) {
	// Convert userId to int64
	userID, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		return nil, err
	}

	orders, err := models.GetUserOrders(context.Background(), userID, 0, 100)
	if err != nil {
		return nil, err
	}

	var result []Order
	for _, o := range orders {
		result = append(result, Order{
			Id:        strconv.FormatUint(uint64(o.ID), 10),
			Amount:    float64(o.Amount),
			Status:    string(o.Status),
			CreatedAt: o.CreateAt.Unix(),
			UpdatedAt: o.UpdateAt.Unix(),
		})
	}

	return result, nil
}

func (s *PayServiceImpl) IsVip(userId string) (bool, error) {
	// In a real implementation, this would check the user's VIP status
	// For now, return false
	return false, nil
}

func (s *PayServiceImpl) SetBond(userId string, bond float64, storyId int64) error {
	// Convert userId to int64
	userID, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		return err
	}

	// Create metadata
	metadata := map[string]interface{}{
		"type":     "bond",
		"story_id": storyId,
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// Create a bond payment
	order := &models.Order{
		UserID:      userID,
		Amount:      int64(bond),
		Status:      int(models.OrderStatusPending),
		Description: "Bond for story " + strconv.FormatInt(storyId, 10),
		Metadata:    string(metadataBytes),
	}

	return models.CreateOrder(context.Background(), order)
}
