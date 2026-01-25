package pay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WeChatOAuthConfig 微信OAuth配置
type WeChatOAuthConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// WeChatAccessTokenResponse 微信Access Token响应
type WeChatAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"` // 需要绑定开放平台账号才有
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

// WeChatUserInfo 微信用户信息
type WeChatUserInfo struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionID    string   `json:"unionid"`
	ErrCode    int      `json:"errcode"`
	ErrMsg     string   `json:"errmsg"`
}

// WeChatOAuthClient 微信OAuth客户端
type WeChatOAuthClient struct {
	config     *WeChatOAuthConfig
	httpClient *http.Client
}

// NewWeChatOAuthClient 创建微信OAuth客户端
func NewWeChatOAuthClient(config *WeChatOAuthConfig) *WeChatOAuthClient {
	return &WeChatOAuthClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAccessToken 用code换取access_token
func (c *WeChatOAuthClient) GetAccessToken(ctx context.Context, code string) (*WeChatAccessTokenResponse, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		c.config.AppID,
		c.config.AppSecret,
		code,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var result WeChatAccessTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

// GetUserInfo 获取微信用户信息
func (c *WeChatOAuthClient) GetUserInfo(ctx context.Context, accessToken, openID string) (*WeChatUserInfo, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		accessToken,
		openID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var result WeChatUserInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

// IsValid 检查客户端配置是否有效
func (c *WeChatOAuthClient) IsValid() bool {
	return c != nil && c.config != nil && c.config.AppID != "" && c.config.AppSecret != ""
}

// GetAppID 获取AppID
func (c *WeChatOAuthClient) GetAppID() string {
	if c.config == nil {
		return ""
	}
	return c.config.AppID
}
