package pay

import "golang.org/x/oauth2"

// 实现支付宝的支付
// AlipayPay 实现支付宝的支付
type AlipayPay struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}
