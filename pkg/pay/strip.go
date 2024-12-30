package pay

import "golang.org/x/oauth2"

// 实现stripe的支付
// StripePay 实现stripe的支付
type StripePay struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}
