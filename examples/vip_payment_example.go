package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/grapery/grapery/models"
	paypkg "github.com/grapery/grapery/pkg/pay"
)

// 示例：如何使用 VIP 支付服务

func main() {
	// 1. 创建支付配置
	paymentConfig := &paypkg.PaymentConfig{
		DefaultCurrency:   "CNY",
		ReturnURL:         "https://your-domain.com/payment/return",
		NotifyURL:         "https://your-domain.com/api/v1/payment/callback",
		OrderExpireTime:   30, // 30分钟
		PaymentExpireTime: 15, // 15分钟
		MaxRetryCount:     3,
		EnableTestMode:    true,
		EnableRiskCheck:   true,
		RiskThreshold:     0.8,
	}

	// 2. 创建支付服务
	paymentService, err := paypkg.NewPaymentService(paymentConfig)
	if err != nil {
		log.Fatal("Failed to create payment service:", err)
	}

	// 3. 示例：创建商品
	ctx := context.Background()
	product := &models.Product{
		Name:        "VIP 会员月卡",
		Description: "享受高级功能和无限使用",
		Price:       9900, // 99元，以分为单位
		ProductType: models.ProductTypeSubscription,
		Duration:    30 * 24 * 3600, // 30天，以秒为单位
		QuotaLimit:  1000,
		MaxRoles:    10,
		MaxContexts: 50,
		Status:      models.ProductStatusActive,
	}

	if err := paymentService.CreateProduct(ctx, product); err != nil {
		log.Printf("Failed to create product: %v", err)
	}

	// 4. 示例：创建订单
	userID := int64(12345)
	productID := uint(1)
	paymentMethod := models.PaymentMethodAlipay

	order, err := paymentService.CreateOrder(ctx, userID, productID, nil, 1, paymentMethod)
	if err != nil {
		log.Fatal("Failed to create order:", err)
	}

	fmt.Printf("订单创建成功: ID=%d, 订单号=%s, 金额=%d分\n",
		order.ID, order.OrderNo, order.Amount)

	// 5. 示例：创建支付
	paymentReq := &paypkg.CreatePaymentRequest{
		UserID:        userID,
		OrderID:       order.ID,
		Amount:        order.Amount,
		Currency:      "CNY",
		PaymentMethod: paymentMethod,
		Description:   order.Description,
		ReturnURL:     "https://your-domain.com/payment/success",
		NotifyURL:     "https://your-domain.com/api/v1/payment/callback",
		IPAddress:     "127.0.0.1",
		UserAgent:     "Example Client/1.0",
	}

	payment, err := paymentService.CreatePayment(ctx, order.ID, paymentMethod, paymentReq)
	if err != nil {
		log.Fatal("Failed to create payment:", err)
	}

	fmt.Printf("支付创建成功: 交易ID=%s, 支付链接=%s\n",
		payment.TransactionID, payment.PaymentURL)

	// 6. 示例：查询支付状态
	time.Sleep(2 * time.Second) // 模拟等待

	paymentStatus, err := paymentService.QueryPaymentStatus(ctx, order.ID)
	if err != nil {
		log.Printf("Failed to query payment status: %v", err)
	} else {
		fmt.Printf("支付状态: %s\n", paymentStatus.Status)
	}

	// 7. 示例：获取用户VIP信息
	subscription, err := paymentService.GetUserVIPInfo(ctx, userID)
	if err != nil {
		fmt.Printf("用户不是VIP: %v\n", err)
	} else {
		fmt.Printf("VIP信息: 状态=%s, 到期时间=%s, 额度使用=%d/%d\n",
			subscription.Status, subscription.EndTime.Format("2006-01-02 15:04:05"),
			subscription.QuotaUsed, subscription.QuotaLimit)
	}

	// 8. 示例：HTTP 客户端调用
	callPaymentAPI()
}

// 示例：HTTP 客户端调用支付 API
func callPaymentAPI() {
	fmt.Println("\n=== HTTP API 调用示例 ===")

	// 创建订单
	createOrderReq := map[string]interface{}{
		"product_id":     1,
		"payment_method": 1, // 支付宝
	}

	createOrderResp, err := httpCall("POST", "http://localhost:8082/api/v1/payment/orders", createOrderReq)
	if err != nil {
		log.Printf("Failed to create order via HTTP: %v", err)
		return
	}

	fmt.Printf("HTTP 创建订单响应: %+v\n", createOrderResp)

	// 查询支付状态
	queryPaymentReq := map[string]interface{}{
		"order_id": 1,
	}

	queryPaymentResp, err := httpCall("POST", "http://localhost:8082/api/v1/payment/query", queryPaymentReq)
	if err != nil {
		log.Printf("Failed to query payment via HTTP: %v", err)
		return
	}

	fmt.Printf("HTTP 查询支付响应: %+v\n", queryPaymentResp)
}

// HTTP 调用辅助函数
func httpCall(method, url string, data interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer your-jwt-token")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
