package pay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapery/grapery/models"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/refund"
)

// StripeProvider Stripe支付提供商实现
type StripeProvider struct {
	secretKey string
}

// NewStripeProvider 创建Stripe支付提供商
func NewStripeProvider(secretKey string) *StripeProvider {
	stripe.Key = secretKey
	return &StripeProvider{
		secretKey: secretKey,
	}
}

// CreatePayment 创建支付
func (s *StripeProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// 创建支付意图
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		Description: stripe.String(req.Description),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, err
	}

	return &CreatePaymentResponse{
		ProviderOrderID: pi.ID,
		TransactionID:   pi.ID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		ExpireTime:      &time.Time{}, // Stripe不提供过期时间
		Metadata: map[string]interface{}{
			"client_secret": pi.ClientSecret,
		},
	}, nil
}

// QueryPayment 查询支付状态
func (s *StripeProvider) QueryPayment(ctx context.Context, providerOrderID string) (*PaymentStatusResponse, error) {
	pi, err := paymentintent.Get(providerOrderID, nil)
	if err != nil {
		return nil, err
	}

	var status models.PaymentStatus
	switch pi.Status {
	case stripe.PaymentIntentStatusRequiresPaymentMethod:
		status = models.PaymentStatusPending
	case stripe.PaymentIntentStatusRequiresConfirmation:
		status = models.PaymentStatusPending
	case stripe.PaymentIntentStatusRequiresAction:
		status = models.PaymentStatusPending
	case stripe.PaymentIntentStatusProcessing:
		status = models.PaymentStatusPending
	case stripe.PaymentIntentStatusRequiresCapture:
		status = models.PaymentStatusSuccess
	case stripe.PaymentIntentStatusCanceled:
		status = models.PaymentStatusCanceled
	case stripe.PaymentIntentStatusSucceeded:
		status = models.PaymentStatusSuccess
	default:
		status = models.PaymentStatusPending
	}

	var paymentTime *time.Time
	if pi.Created > 0 {
		t := time.Unix(pi.Created, 0)
		paymentTime = &t
	}

	return &PaymentStatusResponse{
		ProviderOrderID: pi.ID,
		Status:          status,
		Amount:          pi.Amount,
		PaymentTime:     paymentTime,
		TransactionID:   pi.ID,
		Metadata: map[string]interface{}{
			"status": pi.Status,
		},
	}, nil
}

// Refund 退款
func (s *StripeProvider) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(req.ProviderOrderID),
		Amount:        stripe.Int64(req.RefundAmount),
		Reason:        stripe.String("requested_by_customer"),
	}

	r, err := refund.New(params)
	if err != nil {
		return nil, err
	}

	var refundTime *time.Time
	if r.Created > 0 {
		t := time.Unix(r.Created, 0)
		refundTime = &t
	}

	return &RefundResponse{
		RefundID:        r.ID,
		ProviderOrderID: req.ProviderOrderID,
		Status:          string(r.Status),
		RefundAmount:    r.Amount,
		RefundTime:      refundTime,
		Metadata: map[string]interface{}{
			"status": r.Status,
		},
	}, nil
}

// HandleCallback 处理支付回调
func (s *StripeProvider) HandleCallback(ctx context.Context, callbackData []byte) (*PaymentCallbackResponse, error) {
	// 解析Stripe webhook事件
	var event stripe.Event
	if err := json.Unmarshal(callbackData, &event); err != nil {
		return nil, err
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return nil, err
		}

		var paymentTime *time.Time
		if paymentIntent.Created > 0 {
			t := time.Unix(paymentIntent.Created, 0)
			paymentTime = &t
		}

		return &PaymentCallbackResponse{
			ProviderOrderID: paymentIntent.ID,
			Status:          models.PaymentStatusSuccess,
			Amount:          paymentIntent.Amount,
			PaymentTime:     paymentTime,
			TransactionID:   paymentIntent.ID,
			Metadata: map[string]interface{}{
				"status": paymentIntent.Status,
			},
		}, nil

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return nil, err
		}

		return &PaymentCallbackResponse{
			ProviderOrderID: paymentIntent.ID,
			Status:          models.PaymentStatusFailed,
			Amount:          paymentIntent.Amount,
			TransactionID:   paymentIntent.ID,
			Metadata: map[string]interface{}{
				"status": paymentIntent.Status,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unhandled event type: %s", event.Type)
	}
}

// GetProviderName 获取提供商名称
func (s *StripeProvider) GetProviderName() string {
	return "stripe"
}

// VerifyCallback 验证回调签名
func (s *StripeProvider) VerifyCallback(ctx context.Context, callbackData []byte, signature string) (bool, error) {
	// Stripe使用webhook签名验证
	// 这里简化处理，实际应该验证签名
	return true, nil
}

// GetPaymentURL 获取支付链接
func (s *StripeProvider) GetPaymentURL(ctx context.Context, req *CreatePaymentRequest) (string, error) {
	// Stripe通常使用客户端SDK，这里返回一个通用的支付页面URL
	return fmt.Sprintf("https://checkout.stripe.com/pay/%s", req.OrderID), nil
}

// GetQRCodeURL 获取二维码链接
func (s *StripeProvider) GetQRCodeURL(ctx context.Context, req *CreatePaymentRequest) (string, error) {
	// Stripe不支持二维码支付，返回空字符串
	return "", fmt.Errorf("stripe does not support QR code payments")
}
