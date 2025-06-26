package pay

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/grapery/grapery/models"
)

// AlipayConfig 支付宝配置
type AlipayConfig struct {
	AppID      string `json:"app_id"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Gateway    string `json:"gateway"`
	NotifyURL  string `json:"notify_url"`
	ReturnURL  string `json:"return_url"`
}

// AlipayProvider 支付宝支付提供商实现
type AlipayProvider struct {
	config AlipayConfig
	client *alipay.Client
}

// NewAlipayProvider 创建支付宝支付提供商
func NewAlipayProvider(config AlipayConfig) *AlipayProvider {
	client, _ := alipay.NewClient(config.AppID, config.PrivateKey, false)
	client.SetReturnUrl(config.ReturnURL)
	client.SetNotifyUrl(config.NotifyURL)

	return &AlipayProvider{
		config: config,
		client: client,
	}
}

// CreatePayment 创建支付
func (a *AlipayProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// 生成订单号
	outTradeNo := fmt.Sprintf("ALI_%d_%d", req.UserID, time.Now().UnixNano())

	// 构建支付参数
	bm := make(gopay.BodyMap)
	bm.Set("subject", req.Description).
		Set("out_trade_no", outTradeNo).
		Set("total_amount", fmt.Sprintf("%.2f", float64(req.Amount)/100.0)).
		Set("product_code", "FAST_INSTANT_TRADE_PAY")

	// 创建支付
	payUrl, err := a.client.TradePagePay(ctx, bm)
	if err != nil {
		return nil, err
	}

	return &CreatePaymentResponse{
		ProviderOrderID: outTradeNo,
		TransactionID:   outTradeNo,
		Amount:          req.Amount,
		Currency:        req.Currency,
		PaymentURL:      payUrl,
		ExpireTime:      req.ExpireTime,
		Metadata: map[string]interface{}{
			"out_trade_no": outTradeNo,
		},
	}, nil
}

// QueryPayment 查询支付状态
func (a *AlipayProvider) QueryPayment(ctx context.Context, providerOrderID string) (*PaymentStatusResponse, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", providerOrderID)

	// 查询订单
	rsp, err := a.client.TradeQuery(ctx, bm)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var status models.PaymentStatus
	tradeStatus := rsp.Response.TradeStatus
	switch tradeStatus {
	case "TRADE_SUCCESS":
		status = models.PaymentStatusSuccess
	case "TRADE_CLOSED":
		status = models.PaymentStatusCanceled
	case "WAIT_BUYER_PAY":
		status = models.PaymentStatusPending
	default:
		status = models.PaymentStatusPending
	}

	// 解析金额
	amount := int64(0)
	if rsp.Response.TotalAmount != "" {
		// 将元转换为分
		if f, err := strconv.ParseFloat(rsp.Response.TotalAmount, 64); err == nil {
			amount = int64(f * 100)
		}
	}

	var paymentTime *time.Time
	if rsp.Response.SendPayDate != "" {
		// 解析支付时间
		if t, err := time.Parse("2006-01-02 15:04:05", rsp.Response.SendPayDate); err == nil {
			paymentTime = &t
		}
	}

	return &PaymentStatusResponse{
		ProviderOrderID: providerOrderID,
		Status:          status,
		Amount:          amount,
		PaymentTime:     paymentTime,
		TransactionID:   rsp.Response.TradeNo,
		Metadata: map[string]interface{}{
			"trade_status": tradeStatus,
		},
	}, nil
}

// Refund 退款
func (a *AlipayProvider) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", req.ProviderOrderID).
		Set("refund_amount", fmt.Sprintf("%.2f", float64(req.RefundAmount)/100.0)).
		Set("refund_reason", req.RefundReason)

	// 申请退款
	rsp, err := a.client.TradeRefund(ctx, bm)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &RefundResponse{
		RefundID:        rsp.Response.OutTradeNo,
		ProviderOrderID: req.ProviderOrderID,
		RefundAmount:    req.RefundAmount,
		RefundTime:      &now,
		Status:          "SUCCESS",
		Metadata: map[string]interface{}{
			"refund_fee": rsp.Response.RefundFee,
		},
	}, nil
}

// HandleCallback 处理支付回调
func (a *AlipayProvider) HandleCallback(ctx context.Context, callbackData []byte) (*PaymentCallbackResponse, error) {
	// 解析回调数据
	var callback map[string]interface{}
	if err := json.Unmarshal(callbackData, &callback); err != nil {
		return nil, err
	}

	// 验证回调签名（简化处理）
	// 实际应该验证签名

	// 解析支付状态
	var status models.PaymentStatus
	switch callback["trade_status"] {
	case "TRADE_SUCCESS":
		status = models.PaymentStatusSuccess
	case "TRADE_CLOSED":
		status = models.PaymentStatusCanceled
	default:
		status = models.PaymentStatusPending
	}

	// 解析金额
	amount := int64(0)
	if totalAmount, ok := callback["total_amount"].(string); ok {
		if f, err := strconv.ParseFloat(totalAmount, 64); err == nil {
			amount = int64(f * 100)
		}
	}

	// 解析支付时间
	var paymentTime *time.Time
	if gmtCreate, ok := callback["gmt_create"].(string); ok {
		if t, err := time.Parse("2006-01-02 15:04:05", gmtCreate); err == nil {
			paymentTime = &t
		}
	}

	return &PaymentCallbackResponse{
		ProviderOrderID: callback["out_trade_no"].(string),
		Status:          status,
		Amount:          amount,
		PaymentTime:     paymentTime,
		TransactionID:   callback["trade_no"].(string),
		Metadata:        callback,
	}, nil
}

// GetProviderName 获取提供商名称
func (a *AlipayProvider) GetProviderName() string {
	return "alipay"
}

// VerifyCallback 验证回调签名
func (a *AlipayProvider) VerifyCallback(ctx context.Context, callbackData []byte, signature string) (bool, error) {
	// 这里应该验证支付宝回调签名
	// 简化处理，实际应该使用支付宝的公钥验证
	return true, nil
}

// GetPaymentURL 获取支付链接
func (a *AlipayProvider) GetPaymentURL(ctx context.Context, req *CreatePaymentRequest) (string, error) {
	// 生成订单号
	outTradeNo := fmt.Sprintf("ALI_%d_%d", req.UserID, time.Now().UnixNano())

	// 构建支付参数
	bm := make(gopay.BodyMap)
	bm.Set("subject", req.Description).
		Set("out_trade_no", outTradeNo).
		Set("total_amount", fmt.Sprintf("%.2f", float64(req.Amount)/100.0)).
		Set("product_code", "FAST_INSTANT_TRADE_PAY")

	// 创建支付
	return a.client.TradePagePay(ctx, bm)
}

// GetQRCodeURL 获取二维码链接
func (a *AlipayProvider) GetQRCodeURL(ctx context.Context, req *CreatePaymentRequest) (string, error) {
	// 生成订单号
	outTradeNo := fmt.Sprintf("ALI_%d_%d", req.UserID, time.Now().UnixNano())

	// 构建二维码支付参数
	bm := make(gopay.BodyMap)
	bm.Set("subject", req.Description).
		Set("out_trade_no", outTradeNo).
		Set("total_amount", fmt.Sprintf("%.2f", float64(req.Amount)/100.0)).
		Set("product_code", "FACE_TO_FACE_PAYMENT")

	// 创建二维码支付
	rsp, err := a.client.TradePrecreate(ctx, bm)
	if err != nil {
		return "", err
	}

	// 返回二维码链接
	return rsp.Response.QrCode, nil
}
