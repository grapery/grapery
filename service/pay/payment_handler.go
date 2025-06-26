package pay

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/pay"
	"github.com/grapery/grapery/service/auth"
)

// PaymentHandler 支付处理器
type PaymentHandler struct {
	paymentService pay.PaymentService
}

// NewPaymentHandler 创建支付处理器
func NewPaymentHandler(paymentService pay.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	ProductID     uint                 `json:"product_id"`
	PaymentMethod models.PaymentMethod `json:"payment_method"`
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data *OrderData `json:"data"`
}

// OrderData 订单数据
type OrderData struct {
	OrderID       uint   `json:"order_id"`
	OrderNo       string `json:"order_no"`
	Amount        int64  `json:"amount"`
	Status        int    `json:"status"`
	PaymentURL    string `json:"payment_url,omitempty"`
	QRCodeURL     string `json:"qr_code_url,omitempty"`
	TransactionID string `json:"transaction_id,omitempty"`
}

// CreateOrder 创建订单
func (h *PaymentHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 解析请求
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 创建订单
	order, err := h.paymentService.CreateOrder(r.Context(), userID, req.ProductID, req.PaymentMethod)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 创建支付
	payment, err := h.paymentService.CreatePayment(r.Context(), order.ID, req.PaymentMethod)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回响应
	response := &CreateOrderResponse{
		Code: 0,
		Msg:  "success",
		Data: &OrderData{
			OrderID:       order.ID,
			OrderNo:       order.OrderNo,
			Amount:        order.Amount,
			Status:        order.Status,
			PaymentURL:    payment.PaymentURL,
			QRCodeURL:     payment.QRCodeURL,
			TransactionID: payment.TransactionID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// QueryPaymentRequest 查询支付状态请求
type QueryPaymentRequest struct {
	OrderID uint `json:"order_id"`
}

// QueryPaymentResponse 查询支付状态响应
type QueryPaymentResponse struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data *PaymentData `json:"data"`
}

// PaymentData 支付数据
type PaymentData struct {
	OrderID         uint                 `json:"order_id"`
	Status          models.PaymentStatus `json:"status"`
	Amount          int64                `json:"amount"`
	PaymentTime     *string              `json:"payment_time,omitempty"`
	ProviderOrderID string               `json:"provider_order_id"`
}

// QueryPayment 查询支付状态
func (h *PaymentHandler) QueryPayment(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID
	_, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 解析请求
	var req QueryPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 查询支付状态
	payment, err := h.paymentService.QueryPaymentStatus(r.Context(), req.OrderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 格式化支付时间
	var paymentTimeStr *string
	if payment.PaymentTime != nil {
		timeStr := payment.PaymentTime.Format("2006-01-02 15:04:05")
		paymentTimeStr = &timeStr
	}

	// 返回响应
	response := &QueryPaymentResponse{
		Code: 0,
		Msg:  "success",
		Data: &PaymentData{
			OrderID:         req.OrderID,
			Status:          payment.Status,
			Amount:          payment.Amount,
			PaymentTime:     paymentTimeStr,
			ProviderOrderID: payment.ProviderOrderID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserOrdersRequest 获取用户订单请求
type GetUserOrdersRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// GetUserOrdersResponse 获取用户订单响应
type GetUserOrdersResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data *OrdersData `json:"data"`
}

// OrdersData 订单列表数据
type OrdersData struct {
	Orders []*OrderInfo `json:"orders"`
	Total  int          `json:"total"`
}

// OrderInfo 订单信息
type OrderInfo struct {
	ID          uint   `json:"id"`
	OrderNo     string `json:"order_no"`
	Amount      int64  `json:"amount"`
	Status      int    `json:"status"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// GetUserOrders 获取用户订单
func (h *PaymentHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 解析请求参数
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

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
	orders, err := h.paymentService.GetUserOrders(r.Context(), userID, offset, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 转换为响应格式
	orderInfos := make([]*OrderInfo, 0, len(orders))
	for _, order := range orders {
		orderInfos = append(orderInfos, &OrderInfo{
			ID:          order.ID,
			OrderNo:     order.OrderNo,
			Amount:      order.Amount,
			Status:      order.Status,
			Description: order.Description,
			CreatedAt:   order.CreateAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 返回响应
	response := &GetUserOrdersResponse{
		Code: 0,
		Msg:  "success",
		Data: &OrdersData{
			Orders: orderInfos,
			Total:  len(orderInfos),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserVIPInfoResponse 获取用户VIP信息响应
type GetUserVIPInfoResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data *VIPData `json:"data"`
}

// VIPData VIP数据
type VIPData struct {
	IsVIP       bool   `json:"is_vip"`
	Level       int    `json:"level"`
	Status      int    `json:"status"`
	ExpireTime  string `json:"expire_time,omitempty"`
	AutoRenew   bool   `json:"auto_renew"`
	QuotaUsed   int    `json:"quota_used"`
	QuotaLimit  int    `json:"quota_limit"`
	MaxRoles    int    `json:"max_roles"`
	MaxContexts int    `json:"max_contexts"`
}

// GetUserVIPInfo 获取用户VIP信息
func (h *PaymentHandler) GetUserVIPInfo(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 获取VIP信息
	subscription, err := h.paymentService.GetUserVIPInfo(r.Context(), userID)
	if err != nil {
		// 用户不是VIP
		response := &GetUserVIPInfoResponse{
			Code: 0,
			Msg:  "success",
			Data: &VIPData{
				IsVIP:       false,
				Level:       0,
				Status:      0,
				AutoRenew:   false,
				QuotaUsed:   0,
				QuotaLimit:  0,
				MaxRoles:    2,
				MaxContexts: 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 返回VIP信息
	response := &GetUserVIPInfoResponse{
		Code: 0,
		Msg:  "success",
		Data: &VIPData{
			IsVIP:       subscription.Status == models.SubscriptionStatusActive,
			Level:       int(subscription.Status),
			Status:      int(subscription.Status),
			ExpireTime:  subscription.EndTime.Format("2006-01-02 15:04:05"),
			AutoRenew:   subscription.AutoRenew,
			QuotaUsed:   subscription.QuotaUsed,
			QuotaLimit:  subscription.QuotaLimit,
			MaxRoles:    subscription.MaxRoles,
			MaxContexts: subscription.MaxContexts,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelSubscriptionRequest 取消订阅请求
type CancelSubscriptionRequest struct {
	SubscriptionID uint   `json:"subscription_id"`
	Reason         string `json:"reason"`
}

// CancelSubscriptionResponse 取消订阅响应
type CancelSubscriptionResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// CancelSubscription 取消订阅
func (h *PaymentHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	// 获取用户ID
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 解析请求
	var req CancelSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 取消订阅
	err = h.paymentService.CancelSubscription(r.Context(), req.SubscriptionID, req.Reason, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回响应
	response := &CancelSubscriptionResponse{
		Code: 0,
		Msg:  "subscription canceled successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PaymentCallbackRequest 支付回调请求
type PaymentCallbackRequest struct {
	Provider string          `json:"provider"`
	Data     json.RawMessage `json:"data"`
}

// PaymentCallbackResponse 支付回调响应
type PaymentCallbackResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// HandlePaymentCallback 处理支付回调
func (h *PaymentHandler) HandlePaymentCallback(w http.ResponseWriter, r *http.Request) {
	// 解析请求
	var req PaymentCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 处理回调
	err := h.paymentService.ProcessPaymentCallback(r.Context(), req.Provider, req.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回响应
	response := &PaymentCallbackResponse{
		Code: 0,
		Msg:  "callback processed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
