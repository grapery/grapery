package pay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapery/grapery/models"
)

// WechatPayProvider 微信支付提供商实现
type WechatPayProvider struct {
	appID  string
	apiKey string
}

// NewWechatPayProvider 创建微信支付提供商
func NewWechatPayProvider(config struct {
	AppID  string `json:"app_id"`
	APIKey string `json:"api_key"`
}) *WechatPayProvider {
	return &WechatPayProvider{
		appID:  config.AppID,
		apiKey: config.APIKey,
	}
}

// CreatePayment 创建支付
func (s *WechatPayProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// 生成微信支付订单号
	outTradeNo := fmt.Sprintf("WX_%d_%d", req.OrderID, time.Now().UnixNano())

	// 简化实现，返回支付链接
	payURL := fmt.Sprintf("https://pay.weixin.qq.com/pay?out_trade_no=%s&total_fee=%d", outTradeNo, req.Amount)

	return &CreatePaymentResponse{
		ProviderOrderID: outTradeNo,
		PaymentURL:      payURL,
		TransactionID:   outTradeNo,
		ExpireTime:      nil,
		Metadata: map[string]interface{}{
			"pay_url": payURL,
		},
	}, nil
}

// QueryPayment 查询支付状态
func (s *WechatPayProvider) QueryPayment(ctx context.Context, providerOrderID string) (*PaymentStatusResponse, error) {
	// 简化实现，假设查询成功表示支付成功
	status := models.PaymentStatusSuccess
	now := time.Now()

	return &PaymentStatusResponse{
		ProviderOrderID: providerOrderID,
		Status:          status,
		Amount:          0, // 需要从其他地方获取
		PaymentTime:     &now,
		Metadata: map[string]interface{}{
			"query_success": true,
		},
	}, nil
}

// Refund 退款
func (s *WechatPayProvider) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	// 简化实现
	now := time.Now()
	return &RefundResponse{
		RefundID:     req.ProviderOrderID,
		Status:       "success",
		RefundAmount: req.RefundAmount,
		RefundTime:   &now,
		Metadata: map[string]interface{}{
			"refund_success": true,
		},
	}, nil
}

// HandleCallback 处理支付回调
func (s *WechatPayProvider) HandleCallback(ctx context.Context, callbackData []byte) (*PaymentCallbackResponse, error) {
	// 解析微信支付回调数据
	var callback map[string]interface{}
	if err := json.Unmarshal(callbackData, &callback); err != nil {
		return nil, err
	}

	// 简化处理
	resultCode, _ := callback["result_code"].(string)
	var status models.PaymentStatus
	if resultCode == "SUCCESS" {
		status = models.PaymentStatusSuccess
	} else {
		status = models.PaymentStatusFailed
	}

	outTradeNo, _ := callback["out_trade_no"].(string)
	totalFee, _ := callback["total_fee"].(float64)
	transactionId, _ := callback["transaction_id"].(string)

	return &PaymentCallbackResponse{
		ProviderOrderID: outTradeNo,
		Status:          status,
		Amount:          int64(totalFee),
		TransactionID:   transactionId,
		Metadata:        callback,
	}, nil
}

// GetProviderName 获取提供商名称
func (s *WechatPayProvider) GetProviderName() string {
	return "wechatpay"
}
