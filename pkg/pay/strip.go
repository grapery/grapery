package pay

import (
	"context"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/price"
	"github.com/stripe/stripe-go/v74/subscription"
	"golang.org/x/oauth2"
)

// 实现stripe的支付
// StripePay 实现stripe的支付
type StripePay struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}

func NewStripePay(secretKey string) *StripePay {
	stripe.Key = secretKey
	return &StripePay{
		ClientSecret: secretKey,
	}
}

// CreateCustomer 创建客户
func (s *StripePay) CreateCustomer(ctx context.Context, email, name string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	return customer.New(params)
}

// CreateSubscription 创建订阅
func (s *StripePay) CreateSubscription(ctx context.Context, customerID, priceID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
	}

	return subscription.New(params)
}

// CancelSubscription 取消订阅
func (s *StripePay) CancelSubscription(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}

	return subscription.Update(subscriptionID, params)
}

// UpdateSubscription 更新订阅
func (s *StripePay) UpdateSubscription(ctx context.Context, subscriptionID, newPriceID string) (*stripe.Subscription, error) {
	// 获取当前订阅
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, err
	}

	// 更新订阅项
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(sub.Items.Data[0].ID),
				Price: stripe.String(newPriceID),
			},
		},
	}

	return subscription.Update(subscriptionID, params)
}

// GetSubscription 获取订阅信息
func (s *StripePay) GetSubscription(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	return subscription.Get(subscriptionID, nil)
}

// ListCustomerSubscriptions 列出客户的所有订阅
func (s *StripePay) ListCustomerSubscriptions(ctx context.Context, customerID string) ([]*stripe.Subscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}

	var subscriptions []*stripe.Subscription
	i := subscription.List(params)
	for i.Next() {
		subscriptions = append(subscriptions, i.Subscription())
	}

	return subscriptions, i.Err()
}

// CreatePrice 创建价格
func (s *StripePay) CreatePrice(ctx context.Context, amount int64, currency, productID string, recurring *stripe.PriceRecurringParams) (*stripe.Price, error) {
	params := &stripe.PriceParams{
		Currency:   stripe.String(currency),
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(amount),
		Recurring:  recurring,
	}

	return price.New(params)
}

// CheckSubscriptionStatus 检查订阅状态
func (s *StripePay) CheckSubscriptionStatus(ctx context.Context, subscriptionID string) (bool, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return false, err
	}

	return sub.Status == stripe.SubscriptionStatusActive, nil
}
