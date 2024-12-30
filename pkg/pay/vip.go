package pay

import "golang.org/x/oauth2"

// 实现vip的支付
// VipPay 实现vip的支付
type VipPay struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}
