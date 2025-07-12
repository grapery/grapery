package pay

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/models"
	paypkg "github.com/grapery/grapery/pkg/pay"
	"github.com/grapery/grapery/service/auth"
)

// GinPaymentHandler Gin 支付处理器
type GinPaymentHandler struct {
	paymentService paypkg.PaymentService
}

// NewGinPaymentHandler 创建 Gin 支付处理器
func NewGinPaymentHandler(paymentService paypkg.PaymentService) *GinPaymentHandler {
	return &GinPaymentHandler{
		paymentService: paymentService,
	}
}

// CreateOrder Gin 创建订单处理器
func (h *GinPaymentHandler) CreateOrder(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 解析请求
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request body",
		})
		return
	}

	// 创建订单
	order, err := h.paymentService.CreateOrder(c.Request.Context(), userID, req.ProductID, nil, 1, req.PaymentMethod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	// 创建支付请求
	paymentReq := &paypkg.CreatePaymentRequest{
		UserID:        userID,
		OrderID:       order.ID,
		Amount:        order.Amount,
		Currency:      "CNY",
		PaymentMethod: req.PaymentMethod,
		Description:   order.Description,
		ReturnURL:     "https://your-domain.com/payment/success",
		NotifyURL:     "https://your-domain.com/api/v1/payment/callback",
		IPAddress:     c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
	}

	// 创建支付
	payment, err := h.paymentService.CreatePayment(c.Request.Context(), order.ID, req.PaymentMethod, paymentReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"order_id":       order.ID,
			"order_no":       order.OrderNo,
			"amount":         order.Amount,
			"status":         order.Status,
			"payment_url":    payment.PaymentURL,
			"qr_code_url":    payment.QRCodeURL,
			"transaction_id": payment.TransactionID,
		},
	})
}

// GetUserOrders Gin 获取用户订单处理器
func (h *GinPaymentHandler) GetUserOrders(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 解析请求参数
	offsetStr := c.Query("offset")
	limitStr := c.Query("limit")

	offset := 0
	limit := 20

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// 获取订单列表
	orders, err := h.paymentService.GetUserOrders(c.Request.Context(), userID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	// 转换为响应格式
	orderInfos := make([]gin.H, 0, len(orders))
	for _, order := range orders {
		orderInfos = append(orderInfos, gin.H{
			"id":          order.ID,
			"order_no":    order.OrderNo,
			"amount":      order.Amount,
			"status":      order.Status,
			"description": order.Description,
			"created_at":  order.CreateAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"orders": orderInfos,
			"total":  len(orderInfos),
		},
	})
}

// QueryPayment Gin 查询支付状态处理器
func (h *GinPaymentHandler) QueryPayment(c *gin.Context) {
	// 获取用户ID
	_, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 解析请求
	var req struct {
		OrderID uint `json:"order_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request body",
		})
		return
	}

	// 查询支付状态
	payment, err := h.paymentService.QueryPaymentStatus(c.Request.Context(), req.OrderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	// 格式化支付时间
	var paymentTimeStr *string
	if payment.PaymentTime != nil {
		timeStr := payment.PaymentTime.Format("2006-01-02 15:04:05")
		paymentTimeStr = &timeStr
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"order_id":          req.OrderID,
			"status":            payment.Status,
			"amount":            payment.Amount,
			"payment_time":      paymentTimeStr,
			"provider_order_id": payment.ProviderOrderID,
		},
	})
}

// GetUserVIPInfo Gin 获取用户VIP信息处理器
func (h *GinPaymentHandler) GetUserVIPInfo(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 获取VIP信息
	subscription, err := h.paymentService.GetUserVIPInfo(c.Request.Context(), userID)
	if err != nil {
		// 用户不是VIP
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "success",
			"data": gin.H{
				"is_vip":       false,
				"level":        0,
				"status":       0,
				"auto_renew":   false,
				"quota_used":   0,
				"quota_limit":  0,
				"max_roles":    2,
				"max_contexts": 5,
			},
		})
		return
	}

	// 返回VIP信息
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"is_vip":       subscription.Status == models.SubscriptionStatusActive,
			"level":        int(subscription.Status),
			"status":       int(subscription.Status),
			"expire_time":  subscription.EndTime.Format("2006-01-02 15:04:05"),
			"auto_renew":   subscription.AutoRenew,
			"quota_used":   subscription.QuotaUsed,
			"quota_limit":  subscription.QuotaLimit,
			"max_roles":    subscription.MaxRoles,
			"max_contexts": subscription.MaxContexts,
		},
	})
}

// CancelSubscription Gin 取消订阅处理器
func (h *GinPaymentHandler) CancelSubscription(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 解析请求
	var req struct {
		SubscriptionID uint   `json:"subscription_id"`
		Reason         string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request body",
		})
		return
	}

	// 取消订阅
	err = h.paymentService.CancelSubscription(c.Request.Context(), req.SubscriptionID, req.Reason, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "subscription canceled successfully",
	})
}

// HandlePaymentCallback Gin 处理支付回调处理器
func (h *GinPaymentHandler) HandlePaymentCallback(c *gin.Context) {
	// 获取回调数据
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "failed to read request body",
		})
		return
	}

	// 获取签名（从请求头或查询参数中）
	signature := c.GetHeader("X-Signature")
	if signature == "" {
		signature = c.Query("signature")
	}

	// 解析请求
	var req struct {
		Provider string          `json:"provider"`
		Data     json.RawMessage `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request body",
		})
		return
	}

	// 处理回调
	err = h.paymentService.ProcessPaymentCallback(c.Request.Context(), req.Provider, body, signature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "callback processed successfully",
	})
}
