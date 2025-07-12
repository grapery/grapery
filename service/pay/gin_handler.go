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

// GetProducts Gin 获取商品列表处理器
func (h *GinPaymentHandler) GetProducts(c *gin.Context) {
	// 获取查询参数
	productTypeStr := c.Query("type")
	category := c.Query("category")
	limitStr := c.Query("limit")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	_ = limit
	var products []*models.Product
	var err error

	if productTypeStr != "" {
		if productType, err := strconv.Atoi(productTypeStr); err == nil {
			products, err = h.paymentService.GetProductsByType(c.Request.Context(), models.ProductType(productType))
		}
	} else if category != "" {
		products, err = h.paymentService.GetProductsByCategory(c.Request.Context(), category)
	} else {
		products, err = h.paymentService.GetActiveProducts(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	// 转换为响应格式
	productInfos := make([]gin.H, 0, len(products))
	for _, product := range products {
		productInfos = append(productInfos, gin.H{
			"id":           product.ID,
			"name":         product.Name,
			"description":  product.Description,
			"price":        product.Price,
			"currency":     product.Currency,
			"type":         product.ProductType,
			"status":       product.Status,
			"sku":          product.SKU,
			"duration":     product.Duration,
			"quota_limit":  product.QuotaLimit,
			"max_roles":    product.MaxRoles,
			"max_contexts": product.MaxContexts,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"products": productInfos,
			"total":    len(productInfos),
		},
	})
}

// GetProduct Gin 获取单个商品处理器
func (h *GinPaymentHandler) GetProduct(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid product id",
		})
		return
	}

	product, err := h.paymentService.GetProduct(c.Request.Context(), uint(productID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	if product == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "product not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"id":               product.ID,
			"name":             product.Name,
			"description":      product.Description,
			"price":            product.Price,
			"currency":         product.Currency,
			"type":             product.ProductType,
			"status":           product.Status,
			"sku":              product.SKU,
			"duration":         product.Duration,
			"quota_limit":      product.QuotaLimit,
			"max_roles":        product.MaxRoles,
			"max_contexts":     product.MaxContexts,
			"available_models": product.AvailableModels,
		},
	})
}

// RefundPayment Gin 退款处理器
func (h *GinPaymentHandler) RefundPayment(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}
	_ = userID
	// 解析请求
	var req struct {
		OrderID      uint   `json:"order_id"`
		RefundAmount int64  `json:"refund_amount"`
		RefundReason string `json:"refund_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request body",
		})
		return
	}

	// 执行退款
	err = h.paymentService.RefundPayment(c.Request.Context(), req.OrderID, req.RefundAmount, req.RefundReason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "refund processed successfully",
	})
}

// GetPaymentRecords Gin 获取支付记录处理器
func (h *GinPaymentHandler) GetPaymentRecords(c *gin.Context) {
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
	statusStr := c.Query("status")
	methodStr := c.Query("method")

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

	var records []*models.PaymentRecord
	var err2 error

	if statusStr != "" {
		if status, err := strconv.Atoi(statusStr); err == nil {
			records, err2 = h.paymentService.GetUserPaymentRecordsByStatus(c.Request.Context(), userID, models.PaymentStatus(status), offset, limit)
		}
	} else if methodStr != "" {
		if method, err := strconv.Atoi(methodStr); err == nil {
			records, err2 = h.paymentService.GetUserPaymentRecordsByMethod(c.Request.Context(), userID, models.PaymentMethod(method), offset, limit)
		}
	} else {
		records, err2 = h.paymentService.GetUserPaymentRecords(c.Request.Context(), userID, offset, limit)
	}

	if err2 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err2.Error(),
		})
		return
	}

	// 转换为响应格式
	recordInfos := make([]gin.H, 0, len(records))
	for _, record := range records {
		var paymentTimeStr *string
		if record.PaymentTime != nil {
			timeStr := record.PaymentTime.Format("2006-01-02 15:04:05")
			paymentTimeStr = &timeStr
		}

		recordInfos = append(recordInfos, gin.H{
			"id":               record.ID,
			"order_id":         record.OrderID,
			"amount":           record.Amount,
			"currency":         record.Currency,
			"status":           record.Status,
			"payment_method":   record.PaymentMethod,
			"payment_provider": record.PaymentProvider,
			"transaction_id":   record.TransactionID,
			"payment_time":     paymentTimeStr,
			"refund_amount":    record.RefundAmount,
			"refund_reason":    record.RefundReason,
			"created_at":       record.CreateAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"records": recordInfos,
			"total":   len(recordInfos),
		},
	})
}

// GetUserSubscriptions Gin 获取用户订阅列表处理器
func (h *GinPaymentHandler) GetUserSubscriptions(c *gin.Context) {
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

	subscriptions, err := h.paymentService.GetUserSubscriptions(c.Request.Context(), userID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	// 转换为响应格式
	subscriptionInfos := make([]gin.H, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		subscriptionInfos = append(subscriptionInfos, gin.H{
			"id":           subscription.ID,
			"product_id":   subscription.ProductID,
			"order_id":     subscription.OrderID,
			"status":       subscription.Status,
			"start_time":   subscription.StartTime.Format("2006-01-02 15:04:05"),
			"end_time":     subscription.EndTime.Format("2006-01-02 15:04:05"),
			"auto_renew":   subscription.AutoRenew,
			"amount":       subscription.Amount,
			"currency":     subscription.Currency,
			"quota_used":   subscription.QuotaUsed,
			"quota_limit":  subscription.QuotaLimit,
			"max_roles":    subscription.MaxRoles,
			"max_contexts": subscription.MaxContexts,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"subscriptions": subscriptionInfos,
			"total":         len(subscriptionInfos),
		},
	})
}

// UpdateSubscriptionQuota Gin 更新订阅额度处理器
func (h *GinPaymentHandler) UpdateSubscriptionQuota(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}
	_ = userID
	// 解析请求
	var req struct {
		SubscriptionID uint `json:"subscription_id"`
		QuotaUsed      int  `json:"quota_used"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request body",
		})
		return
	}

	// 更新订阅额度
	err = h.paymentService.UpdateSubscriptionQuota(c.Request.Context(), req.SubscriptionID, req.QuotaUsed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "subscription quota updated successfully",
	})
}

// ConsumeUserQuota Gin 消费用户额度处理器
func (h *GinPaymentHandler) ConsumeUserQuota(c *gin.Context) {
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
		Amount int `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request body",
		})
		return
	}

	// 消费用户额度
	err = h.paymentService.ConsumeUserQuota(c.Request.Context(), userID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "quota consumed successfully",
	})
}

// GetUserQuota Gin 获取用户额度处理器
func (h *GinPaymentHandler) GetUserQuota(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 获取用户额度
	used, limit, err := h.paymentService.GetUserQuota(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"quota_used":  used,
			"quota_limit": limit,
			"remaining":   limit - used,
		},
	})
}

// CheckUserPermission Gin 检查用户权限处理器
func (h *GinPaymentHandler) CheckUserPermission(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 获取权限参数
	permission := c.Query("permission")
	if permission == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "permission parameter is required",
		})
		return
	}

	// 检查用户权限
	hasPermission, err := h.paymentService.CheckUserPermission(c.Request.Context(), userID, permission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"permission":     permission,
			"has_permission": hasPermission,
		},
	})
}

// GetPaymentStats Gin 获取支付统计处理器
func (h *GinPaymentHandler) GetPaymentStats(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 获取支付统计
	stats, err := h.paymentService.GetPaymentStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": stats,
	})
}

// GetOrderStats Gin 获取订单统计处理器
func (h *GinPaymentHandler) GetOrderStats(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 获取订单统计
	stats, err := h.paymentService.GetOrderStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": stats,
	})
}

// GetUserMaxRoles Gin 获取用户最大角色数处理器
func (h *GinPaymentHandler) GetUserMaxRoles(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 获取用户最大角色数
	maxRoles, err := h.paymentService.GetUserMaxRoles(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"max_roles": maxRoles,
		},
	})
}

// GetUserMaxContexts Gin 获取用户最大上下文数处理器
func (h *GinPaymentHandler) GetUserMaxContexts(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 获取用户最大上下文数
	maxContexts, err := h.paymentService.GetUserMaxContexts(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"max_contexts": maxContexts,
		},
	})
}

// GetUserAvailableModels Gin 获取用户可用模型处理器
func (h *GinPaymentHandler) GetUserAvailableModels(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 获取用户可用模型
	models, err := h.paymentService.GetUserAvailableModels(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"available_models": models,
		},
	})
}

// IsUserVIP Gin 检查用户是否为VIP处理器
func (h *GinPaymentHandler) IsUserVIP(c *gin.Context) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	// 检查用户是否为VIP
	isVIP, err := h.paymentService.IsUserVIP(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"is_vip": isVIP,
		},
	})
}
