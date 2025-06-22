package pay

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// 实现支付宝的支付
// AlipayPay 实现支付宝的支付
type AlipayPay struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}

// 订阅
func (a *AlipayPay) Subscribe(userID int64, planID string) (string, error) {
	client, err := a.getClient()
	if err != nil {
		return "", err
	}
	// 生成唯一订单号
	outTradeNo := fmt.Sprintf("%s_%d", uuid.New().String(), time.Now().UnixNano())
	// 这里的 subject/total_amount/product_code 需根据你的订阅套餐配置
	bm := make(gopay.BodyMap)
	bm.Set("subject", "订阅服务").
		Set("out_trade_no", outTradeNo).
		Set("total_amount", "9.99"). // 这里应查 planID 得到价格
		Set("product_code", "FAST_INSTANT_TRADE_PAY")
	payUrl, err := client.TradePagePay(context.Background(), bm)
	if err != nil {
		return "", err
	}
	// 返回支付跳转链接
	return payUrl, nil
}

// 取消订阅
func (a *AlipayPay) CancelSubscription(userID int64, subscriptionID string) error {
	client, err := a.getClient()
	if err != nil {
		return err
	}
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", subscriptionID)
	_, err = client.TradeClose(context.Background(), bm)
	return err
}

// 续订
func (a *AlipayPay) RenewSubscription(userID int64, subscriptionID string) error {
	// 续订本质上是创建新订单，和 Subscribe 类似
	// 你可以查找原订阅套餐，重新调用 Subscribe
	// 或者直接调用 Subscribe(userID, planID)
	return nil // 具体实现可参考 Subscribe
}

// 升级订阅
func (a *AlipayPay) UpgradeSubscription(userID int64, subscriptionID, newPlanID string) error {
	// 升级通常是先关闭原订单，再创建新订单
	err := a.CancelSubscription(userID, subscriptionID)
	if err != nil {
		return err
	}
	// 创建新订单
	_, err = a.Subscribe(userID, newPlanID)
	return err
}

// 查询支付记录
func (a *AlipayPay) ListPayments(userID int64) ([]PaymentRecord, error) {
	// 实际业务应从你自己的数据库查找支付记录
	// 这里仅做演示，调用支付宝API查询单笔订单
	// 推荐你维护自己的支付记录表
	return []PaymentRecord{
		{
			OrderID:     "sub_123_456",
			Amount:      9.99,
			Status:      "TRADE_SUCCESS",
			CreatedAt:   time.Now().Unix(),
			Description: "订阅服务",
		},
	}, nil
}

// PaymentRecord 支付记录结构体
// 可根据实际业务扩展
type PaymentRecord struct {
	OrderID     string
	Amount      float64
	Status      string
	CreatedAt   int64
	Description string
}

func (a *AlipayPay) getClient() (*alipay.Client, error) {
	// 这里需要你配置好 appid、私钥、证书等
	client, err := alipay.NewClient(a.ClientID, a.ClientSecret, false)
	if err != nil {
		return nil, err
	}
	// 可选：设置回调、证书等
	// client.SetReturnUrl(...)
	// client.SetNotifyUrl(...)
	return client, nil
}
