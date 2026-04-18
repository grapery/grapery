package pay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
}

// WebPaymentService provides web payment operations
type WebPaymentService struct {
	repo   *paymodels.WebPaymentRepository
	logger *logrus.Logger
	config *WebPaymentConfig
}

// NewWebPaymentService creates a new web payment service
func NewWebPaymentService(logger *logrus.Logger, config *WebPaymentConfig) *WebPaymentService {
	if config == nil {
		config = &WebPaymentConfig{}
	}
	return &WebPaymentService{
		repo:   paymodels.NewWebPaymentRepository(),
		logger: logger,
		config: config,
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

	// Create Payment Intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(req.Amount)),
		Currency: stripe.String(string(req.Currency)),
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

// createAlipayPayment creates an Alipay payment
func (s *WebPaymentService) createAlipayPayment(ctx context.Context, payment *paymodels.WebPayment, req *CreatePaymentRequest) (string, error) {
	// Generate unique out trade no
	outTradeNo := fmt.Sprintf("PAY_%s_%d", payment.ID, time.Now().UnixNano())

	// TODO: Integrate with Alipay SDK
	// For now, return a placeholder QR code URL
	qrCodeURL := fmt.Sprintf("https://qr.alipay.com/demo/%s", outTradeNo)

	payment.AlipayOutTradeNo = outTradeNo
	payment.Status = paymodels.WebPaymentStatusProcessing

	s.logger.WithFields(logrus.Fields{
		"payment_id":   payment.ID,
		"out_trade_no": outTradeNo,
	}).Info("Alipay payment created (demo mode)")

	return qrCodeURL, nil
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

func getStripePublishableKey() string {
	// Get from environment variable
	if key := os.Getenv("STRIPE_PUBLISHABLE_KEY"); key != "" {
		return key
	}
	return ""
}
