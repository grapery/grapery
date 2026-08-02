package pay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"gorm.io/gorm"
)

// WebPaymentConfig holds configuration for web payments
type WebPaymentConfig struct {
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string
	AlipayAppID          string
	AlipayPrivateKey     string
	AlipayPublicKey      string
	Wechat               *WechatPayConfig
}

// WebPaymentService provides web payment operations
type WebPaymentService struct {
	repo      *paymodels.WebPaymentRepository
	logger    *logrus.Logger
	config    *WebPaymentConfig
	wechatPay *wechatPayRuntime
}

// NewWebPaymentService creates a new web payment service
func NewWebPaymentService(logger *logrus.Logger, config *WebPaymentConfig) *WebPaymentService {
	if config == nil {
		config = &WebPaymentConfig{}
	}
	if config.Wechat == nil {
		config.Wechat = wechatPayConfigFromEnv()
	}
	return &WebPaymentService{
		repo:      paymodels.NewWebPaymentRepository(),
		logger:    logger,
		config:    config,
		wechatPay: newWechatPayRuntime(logger, config.Wechat),
	}
}

// GetStripeWebhookSecret returns the Stripe webhook secret from config
func (s *WebPaymentService) GetStripeWebhookSecret() string {
	if s.config == nil {
		return ""
	}
	return s.config.StripeWebhookSecret
}

// CreatePaymentRequest represents a request to create a payment
type CreatePaymentRequest struct {
	UserID   string                     `json:"userId"`
	PlanID   string                     `json:"planId"`
	Amount   int                        `json:"amount"`
	Currency string                     `json:"currency"`
	Method   paymodels.WebPaymentMethod `json:"method"`
	Metadata map[string]interface{}     `json:"metadata"`
}

// CreatePaymentResponse represents the response after creating a payment
type CreatePaymentResponse struct {
	PaymentID      string                     `json:"id"`
	ClientSecret   string                     `json:"clientSecret,omitempty"`
	QRCodeURL      string                     `json:"qrCodeURL,omitempty"`
	PublishableKey string                     `json:"publishableKey,omitempty"`
	Amount         int                        `json:"amount"`
	Currency       string                     `json:"currency"`
	Status         paymodels.WebPaymentStatus `json:"status"`
	Method         paymodels.WebPaymentMethod `json:"method"`
	CreatedAt      int64                      `json:"createdAt"`
	Metadata       map[string]interface{}     `json:"metadata,omitempty"`
}

// CreatePayment creates a new web payment
func (s *WebPaymentService) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	now := time.Now().UnixMilli()

	// Generate payment ID
	paymentID, err := generatePaymentID()
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate payment ID")
		return nil, fmt.Errorf("failed to generate payment ID: %w", err)
	}

	payment := &paymodels.WebPayment{
		ID:        paymentID,
		UserID:    req.UserID,
		PlanID:    req.PlanID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Status:    paymodels.WebPaymentStatusPending,
		Method:    req.Method,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  req.Metadata,
	}

	var clientSecret string
	var qrCodeURL string

	// Create payment based on method
	switch req.Method {
	case paymodels.WebPaymentMethodStripe:
		clientSecret, err = s.createStripePayment(ctx, payment, req)
		if err != nil {
			return nil, err
		}
		payment.StripeClientSecret = clientSecret

	case paymodels.WebPaymentMethodWechat:
		qrCodeURL, err = s.createWechatPayment(ctx, payment, req)
		if err != nil {
			return nil, err
		}
		payment.WechatCodeURL = qrCodeURL

	case paymodels.WebPaymentMethodAlipay:
		qrCodeURL, err = s.createAlipayPayment(ctx, payment, req)
		if err != nil {
			return nil, err
		}
		payment.AlipayQRCodeURL = qrCodeURL

	default:
		return nil, fmt.Errorf("unsupported payment method: %s", req.Method)
	}

	// Save to database
	if err := s.repo.CreatePayment(payment); err != nil {
		s.logger.WithError(err).Error("Failed to create payment in database")
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"payment_id": paymentID,
		"user_id":    req.UserID,
		"amount":     req.Amount,
		"currency":   req.Currency,
		"method":     req.Method,
	}).Info("Web payment created successfully")

	response := &CreatePaymentResponse{
		PaymentID:    paymentID,
		ClientSecret: clientSecret,
		QRCodeURL:    qrCodeURL,
		Amount:       payment.Amount,
		Currency:     payment.Currency,
		Status:       payment.Status,
		Method:       payment.Method,
		CreatedAt:    payment.CreatedAt,
		Metadata:     payment.Metadata,
	}

	// Prefer WeChat code_url when present
	if payment.WechatCodeURL != "" {
		response.QRCodeURL = payment.WechatCodeURL
	}

	// Add Stripe publishable key if applicable
	if req.Method == paymodels.WebPaymentMethodStripe {
		response.PublishableKey = getStripePublishableKey()
	}

	return response, nil
}

// createStripePayment creates a Stripe Payment Intent
func (s *WebPaymentService) createStripePayment(ctx context.Context, payment *paymodels.WebPayment, req *CreatePaymentRequest) (string, error) {
	// Initialize Stripe
	stripeKey := getStripeAPIKey()
	if stripeKey == "" {
		return "", errors.New("Stripe API key not configured")
	}

	stripe.Key = stripeKey

	currency := strings.ToLower(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "usd"
	}
	amount := int64(req.Amount)
	// Typical US/EU Stripe accounts do not accept CNY PaymentIntents.
	// Convert fen → USD cents using STRIPE_CNY_USD_RATE (major-unit rate, default 0.14).
	if currency == "cny" {
		rate := stripeCNYToUSDRate()
		yuan := float64(req.Amount) / 100.0
		amount = int64(math.Round(yuan * rate * 100))
		if amount < 50 {
			amount = 50 // Stripe USD minimum
		}
		currency = "usd"
		if payment.Metadata == nil {
			payment.Metadata = map[string]interface{}{}
		}
		payment.Metadata["original_amount"] = req.Amount
		payment.Metadata["original_currency"] = "cny"
		payment.Metadata["charged_currency"] = "usd"
		payment.Currency = "usd"
		payment.Amount = int(amount)
	}

	// Create Payment Intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		Params: stripe.Params{
			Metadata: map[string]string{
				"user_id":    req.UserID,
				"plan_id":    req.PlanID,
				"payment_id": payment.ID,
			},
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create Stripe Payment Intent")
		return "", fmt.Errorf("failed to create Stripe Payment Intent: %w", err)
	}

	payment.StripePaymentIntentID = pi.ID
	payment.Status = paymodels.WebPaymentStatusProcessing

	// Return client secret for frontend
	return pi.ClientSecret, nil
}

// createAlipayPayment — web Alipay is not productized; refuse rather than return a demo QR URL.
func (s *WebPaymentService) createAlipayPayment(ctx context.Context, payment *paymodels.WebPayment, req *CreatePaymentRequest) (string, error) {
	return "", errors.New("Alipay web payments are not available; use Stripe or WeChat Pay")
}

// createWechatPayment creates a WeChat Native (QR) order. Amount must be CNY fen.
func (s *WebPaymentService) createWechatPayment(ctx context.Context, payment *paymodels.WebPayment, req *CreatePaymentRequest) (string, error) {
	if s.wechatPay == nil {
		return "", errors.New("WeChat Pay is not available")
	}

	currency := strings.ToLower(strings.TrimSpace(req.Currency))
	amountFen := int64(req.Amount)
	// Plans may be quoted in USD; convert to CNY fen for WeChat (inverse of Stripe path).
	if currency == "usd" || currency == "" {
		rate := stripeCNYToUSDRate()
		if rate <= 0 {
			rate = 0.14
		}
		usd := float64(req.Amount) / 100.0
		amountFen = int64(math.Round(usd / rate * 100))
		if amountFen < 1 {
			amountFen = 1
		}
		payment.Currency = "CNY"
		payment.Amount = int(amountFen)
		if payment.Metadata == nil {
			payment.Metadata = map[string]interface{}{}
		}
		payment.Metadata["original_amount"] = req.Amount
		payment.Metadata["original_currency"] = strings.ToUpper(firstNonEmpty(currency, "usd"))
		payment.Metadata["charged_currency"] = "cny"
	} else if currency != "cny" {
		return "", fmt.Errorf("WeChat Pay only supports CNY (got %s)", req.Currency)
	} else {
		payment.Currency = "CNY"
	}

	outTradeNo := wechatOutTradeNoFromPaymentID(payment.ID)
	desc := fmt.Sprintf("Membership %s", req.PlanID)
	codeURL, err := s.wechatPay.CreateNativeOrder(ctx, outTradeNo, desc, amountFen)
	if err != nil {
		return "", err
	}

	payment.WechatOutTradeNo = outTradeNo
	payment.WechatCodeURL = codeURL
	payment.Status = paymodels.WebPaymentStatusProcessing
	return codeURL, nil
}

// HandleWechatNotify processes WeChat Pay payment result notifications.
func (s *WebPaymentService) HandleWechatNotify(ctx context.Context, request *http.Request) error {
	if s.wechatPay == nil {
		return errors.New("WeChat Pay is not available")
	}
	_, transaction, err := s.wechatPay.ParseNotify(ctx, request)
	if err != nil {
		return fmt.Errorf("parse WeChat notify: %w", err)
	}
	if transaction == nil || transaction.OutTradeNo == nil {
		return errors.New("WeChat notify missing out_trade_no")
	}

	outTradeNo := *transaction.OutTradeNo
	payment, err := s.repo.GetPaymentByWechatOutTradeNo(outTradeNo)
	if err != nil {
		return fmt.Errorf("payment not found for WeChat out_trade_no %s: %w", outTradeNo, err)
	}

	tradeState := ""
	if transaction.TradeState != nil {
		tradeState = *transaction.TradeState
	}

	switch tradeState {
	case "SUCCESS":
		return s.UpdatePaymentStatus(ctx, payment.ID, paymodels.WebPaymentStatusSucceeded, map[string]interface{}{
			"wechat_out_trade_no": outTradeNo,
		})
	case "CLOSED", "REVOKED", "PAYERROR":
		return s.UpdatePaymentStatus(ctx, payment.ID, paymodels.WebPaymentStatusFailed, map[string]interface{}{
			"failure_reason": tradeState,
		})
	default:
		s.logger.WithFields(logrus.Fields{
			"payment_id":  payment.ID,
			"trade_state": tradeState,
		}).Info("WeChat notify ignored (non-terminal state)")
		return nil
	}
}

// GetPayment retrieves a payment by ID
func (s *WebPaymentService) GetPayment(ctx context.Context, paymentID string) (*paymodels.WebPayment, error) {
	payment, err := s.repo.GetPaymentByID(paymentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("payment not found")
		}
		s.logger.WithError(err).WithField("payment_id", paymentID).Error("Failed to get payment")
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}

// UpdatePaymentStatus updates the payment status
func (s *WebPaymentService) UpdatePaymentStatus(ctx context.Context, paymentID string, status paymodels.WebPaymentStatus, updates map[string]interface{}) error {
	// Build update map
	updateMap := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UnixMilli(),
	}

	// Merge additional updates
	for k, v := range updates {
		updateMap[k] = v
	}

	if err := s.repo.UpdatePayment(paymentID, updateMap); err != nil {
		s.logger.WithError(err).WithField("payment_id", paymentID).Error("Failed to update payment status")
		return fmt.Errorf("failed to update payment: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"payment_id": paymentID,
		"status":     status,
	}).Info("Payment status updated")

	return nil
}

// GetUserPayments retrieves payment history for a user
func (s *WebPaymentService) GetUserPayments(ctx context.Context, userID string, limit, offset int, status *paymodels.WebPaymentStatus, method *paymodels.WebPaymentMethod) ([]*paymodels.WebPayment, int64, error) {
	payments, total, err := s.repo.GetUserPayments(userID, limit, offset, status, method)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user payments")
		return nil, 0, fmt.Errorf("failed to get user payments: %w", err)
	}

	return payments, total, nil
}

// HandleStripeWebhook handles Stripe webhook events
func (s *WebPaymentService) HandleStripeWebhook(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "payment_intent.succeeded":
		// Parse the payment intent from the raw data
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			s.logger.WithError(err).Error("Failed to unmarshal payment intent")
			return errors.New("invalid payment intent object")
		}
		return s.handleStripePaymentSucceeded(ctx, &pi)

	case "payment_intent.payment_failed":
		// Parse the payment intent from the raw data
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			s.logger.WithError(err).Error("Failed to unmarshal payment intent")
			return errors.New("invalid payment intent object")
		}
		return s.handleStripePaymentFailed(ctx, &pi)

	default:
		s.logger.WithField("event_type", event.Type).Info("Unhandled Stripe webhook event")
		return nil
	}
}

// handleStripePaymentSucceeded handles successful Stripe payment
func (s *WebPaymentService) handleStripePaymentSucceeded(ctx context.Context, pi *stripe.PaymentIntent) error {
	payment, err := s.repo.GetPaymentByStripePaymentIntentID(pi.ID)
	if err != nil {
		s.logger.WithError(err).WithField("stripe_pi", pi.ID).Error("Payment not found for Stripe PI")
		return fmt.Errorf("payment not found: %w", err)
	}

	return s.UpdatePaymentStatus(ctx, payment.ID, paymodels.WebPaymentStatusSucceeded, map[string]interface{}{
		"stripe_payment_intent_id": pi.ID,
	})
}

// handleStripePaymentFailed handles failed Stripe payment
func (s *WebPaymentService) handleStripePaymentFailed(ctx context.Context, pi *stripe.PaymentIntent) error {
	payment, err := s.repo.GetPaymentByStripePaymentIntentID(pi.ID)
	if err != nil {
		s.logger.WithError(err).WithField("stripe_pi", pi.ID).Error("Payment not found for Stripe PI")
		return fmt.Errorf("payment not found: %w", err)
	}

	failureReason := ""
	if pi.LastPaymentError != nil {
		failureReason = pi.LastPaymentError.Error()
	}

	return s.UpdatePaymentStatus(ctx, payment.ID, paymodels.WebPaymentStatusFailed, map[string]interface{}{
		"failure_reason": failureReason,
		"failure_code":   "", // pi.LastPaymentError.Code,
	})
}

// HandleAlipayWebhook handles Alipay webhook events
func (s *WebPaymentService) HandleAlipayWebhook(ctx context.Context, notification map[string]interface{}) error {
	// TODO: Implement Alipay webhook handling
	s.logger.WithField("notification", notification).Info("Alipay webhook received")
	return nil
}

// Helper functions

func generatePaymentID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "pay_" + hex.EncodeToString(b), nil
}

func getStripeAPIKey() string {
	// Get from environment variable
	if key := os.Getenv("STRIPE_SECRET_KEY"); key != "" {
		return key
	}
	// Return empty string if not configured - this will cause Stripe calls to fail
	// which is safer than using a placeholder key
	return ""
}

// stripeCNYToUSDRate returns CNY→USD rate for major units (¥1 → $rate).
func stripeCNYToUSDRate() float64 {
	if v := os.Getenv("STRIPE_CNY_USD_RATE"); v != "" {
		if rate, err := strconv.ParseFloat(v, 64); err == nil && rate > 0 {
			return rate
		}
	}
	return 0.14
}

func getStripePublishableKey() string {
	// Get from environment variable
	if key := os.Getenv("STRIPE_PUBLISHABLE_KEY"); key != "" {
		return key
	}
	return ""
}
