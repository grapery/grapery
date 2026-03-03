package pay

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	paypkg "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhook"
)

// WebPaymentHandler handles web payment HTTP requests
type WebPaymentHandler struct {
	service *paypkg.WebPaymentService
	logger  *logrus.Logger
}

// NewWebPaymentHandler creates a new web payment handler
func NewWebPaymentHandler(service *paypkg.WebPaymentService, logger *logrus.Logger) *WebPaymentHandler {
	return &WebPaymentHandler{
		service: service,
		logger:  logger,
	}
}

// CreatePayment handles payment creation requests
// POST /api/vippay/web/payments
func (h *WebPaymentHandler) CreatePayment(c *gin.Context) {
	var req paypkg.CreatePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"msg":     "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if req.UserID == "" || req.PlanID == "" || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Missing required fields: userId, planId, amount",
		})
		return
	}

	// Set default currency if not provided
	if req.Currency == "" {
		req.Currency = "USD"
	}

	// Set default metadata if nil
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	// Create payment
	resp, err := h.service.CreatePayment(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create web payment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"msg":     "Failed to create payment",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "Payment created successfully",
		"data": resp,
	})
}

// GetPayment retrieves a payment by ID
// GET /api/vippay/web/payments/:id
func (h *WebPaymentHandler) GetPayment(c *gin.Context) {
	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Payment ID is required",
		})
		return
	}

	payment, err := h.service.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		if err.Error() == "payment not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"code": 404,
				"msg":  "Payment not found",
			})
			return
		}

		h.logger.WithError(err).WithField("payment_id", paymentID).Error("Failed to get payment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"msg":     "Failed to get payment",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Success",
		"data": payment,
	})
}

// GetUserPayments retrieves payment history for a user
// GET /api/vippay/web/payments/user/:userId
func (h *WebPaymentHandler) GetUserPayments(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "User ID is required",
		})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	status := c.Query("status")
	method := c.Query("method")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	var statusPtr *paymodels.WebPaymentStatus
	if status != "" {
		s := paymodels.WebPaymentStatus(status)
		statusPtr = &s
	}

	var methodPtr *paymodels.WebPaymentMethod
	if method != "" {
		m := paymodels.WebPaymentMethod(method)
		methodPtr = &m
	}

	payments, total, err := h.service.GetUserPayments(c.Request.Context(), userID, limit, offset, statusPtr, methodPtr)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user payments")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"msg":     "Failed to get payment history",
			"details": err.Error(),
		})
		return
	}

	page := (offset / limit) + 1

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Success",
		"data": gin.H{
			"payments": payments,
			"total":    total,
			"page":     page,
			"limit":    limit,
		},
	})
}

// HandleStripeWebhook handles Stripe webhook events
// POST /api/vippay/web/webhooks/stripe
func (h *WebPaymentHandler) HandleStripeWebhook(c *gin.Context) {
	// Read request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read webhook payload")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Failed to read payload",
		})
		return
	}

	// Get Stripe webhook secret from the service config
	webhookSecret := h.service.GetStripeWebhookSecret()
	if webhookSecret == "" {
		h.logger.Warn("Stripe webhook secret not configured, skipping signature verification")
	} else {
		// Verify webhook signature
		sigHeader := c.GetHeader("Stripe-Signature")
		if sigHeader == "" {
			h.logger.Error("Missing Stripe-Signature header")
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  "Missing signature header",
			})
			return
		}

		event, err := webhook.ConstructEvent(payload, sigHeader, webhookSecret)
		if err != nil {
			h.logger.WithError(err).Error("Failed to verify Stripe webhook signature")
				c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  "Invalid signature",
			})
			return
		}

		// Handle the verified event
		if err := h.service.HandleStripeWebhook(c.Request.Context(), event); err != nil {
			h.logger.WithError(err).WithField("event_type", event.Type).Error("Failed to handle webhook")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "Failed to handle webhook",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "Webhook processed successfully",
		})
		return
	}

	// Fallback: parse without verification when webhook secret is not configured
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		h.logger.WithError(err).Error("Failed to unmarshal Stripe event")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Invalid webhook payload",
		})
		return
	}

	// Handle the event
	if err := h.service.HandleStripeWebhook(c.Request.Context(), event); err != nil {
		h.logger.WithError(err).WithField("event_type", event.Type).Error("Failed to handle webhook")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Failed to handle webhook",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Webhook processed successfully",
	})
}

// HandleAlipayWebhook handles Alipay webhook events
// POST /api/vippay/web/webhooks/alipay
func (h *WebPaymentHandler) HandleAlipayWebhook(c *gin.Context) {
	var notification map[string]interface{}
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Invalid notification",
		})
		return
	}

	if err := h.service.HandleAlipayWebhook(c.Request.Context(), notification); err != nil {
		h.logger.WithError(err).Error("Failed to handle Alipay webhook")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Failed to handle webhook",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Notification processed successfully",
	})
}
