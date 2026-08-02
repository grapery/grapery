package thirdpart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// 实现微信的鉴权登录
// WechatLogin 实现微信的OAuth2鉴权登录
type WechatLogin struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}

// ... existing WechatLogin struct ...

// NewWechatLogin 创建新的微信登录实例
func NewWechatLogin(clientID, clientSecret, redirectURL string) *WechatLogin {
	wl := &WechatLogin{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"snsapi_userinfo"}, // 微信的默认scope
	}

	wl.Config = &oauth2.Config{
		ClientID:     wl.ClientID,
		ClientSecret: wl.ClientSecret,
		RedirectURL:  wl.RedirectURL,
		Scopes:       wl.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://open.weixin.qq.com/connect/qrconnect",
			TokenURL: "https://api.weixin.qq.com/sns/oauth2/access_token",
		},
	}

	return wl
}

// GetAuthURL 获取微信授权URL
func (w *WechatLogin) GetAuthURL(state string) string {
	return w.Config.AuthCodeURL(state) + "#wechat_redirect"
}

// Exchange 通过授权码获取访问令牌
func (w *WechatLogin) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return w.Config.Exchange(ctx, code)
}

// WechatUserInfo 微信用户信息结构体
type WechatUserInfo struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	UnionID    string   `json:"unionid"`
	Privilege  []string `json:"privilege"`
}

// GetUserInfo 获取微信用户信息
func (w *WechatLogin) GetUserInfo(ctx context.Context, token *oauth2.Token) (*WechatUserInfo, error) {
	// 微信需要同时使用access_token和openid来获取用户信息
	userInfoURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		token.AccessToken,
		token.Extra("openid"),
	)

	client := &http.Client{}
	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info, status code: %d", resp.StatusCode)
	}

	var userInfo WechatUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func WeixinLogin() {
	wechatLogin := NewWechatLogin(
		"your-app-id",
		"your-app-secret",
		"http://your-domain/callback",
	)

	// 1. 获取授权URL
	authURL := wechatLogin.GetAuthURL("random-state")

	// 2. 重定向用户到authURL
	_ = authURL
	// 3. 在回调处理中：
	token, err := wechatLogin.Exchange(context.Background(), "authorization-code")
	if err != nil {
		// 处理错误
		log.Log().Error("wechat login exchange error", zap.Error(err))
	}

	// 4. 获取用户信息
	userInfo, err := wechatLogin.GetUserInfo(context.Background(), token)
	if err != nil {
		// 处理错误
		log.Log().Error("wechat login get user info error", zap.Error(err))
	}

	// 使用用户信息
	log.Log().Info("wechat login get user info", zap.Any("userInfo", userInfo))
}
