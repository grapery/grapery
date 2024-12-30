package pay

import "golang.org/x/oauth2"

// 实现微信支付的支付
// WechatPay 实现微信支付的支付
type WechatPay struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}
