package thridpart

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

// 实现谷歌的鉴权登录
// GoogleLogin 实现谷歌的OAuth2鉴权登录
type GoogleLogin struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}

// NewGoogleLogin 创建新的谷歌登录实例
func NewGoogleLogin(clientID, clientSecret, redirectURL string) *GoogleLogin {
	gl := &GoogleLogin{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	gl.Config = &oauth2.Config{
		ClientID:     gl.ClientID,
		ClientSecret: gl.ClientSecret,
		RedirectURL:  gl.RedirectURL,
		Scopes:       gl.Scopes,
		Endpoint:     google.Endpoint,
	}

	return gl
}

// GetAuthURL 获取谷歌授权URL
func (g *GoogleLogin) GetAuthURL(state string) string {
	return g.Config.AuthCodeURL(state)
}

// Exchange 通过授权码获取用户信息
func (g *GoogleLogin) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Config.Exchange(ctx, code)
}

// GetUserInfo 获取用户信息
func (g *GoogleLogin) GetUserInfo(ctx context.Context, token *oauth2.Token) (*oauth2.Userinfo, error) {
	client := g.Config.Client(ctx, token)
	service, err := oauth2.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	userInfo, err := service.Userinfo.Get().Do()
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

func main() {
	googleLogin := NewGoogleLogin(
		"your-client-id",
		"your-client-secret",
		"http://localhost:8080/callback",
	)

	// 1. 获取授权URL
	authURL := googleLogin.GetAuthURL("random-state")

	// 2. 重定向用户到authURL

	// 3. 在回调处理中：
	token, err := googleLogin.Exchange(context.Background(), "authorization-code")
	if err != nil {
		// 处理错误
	}

	// 4. 获取用户信息
	userInfo, err := googleLogin.GetUserInfo(context.Background(), token)
	if err != nil {
		// 处理错误
	}

	// 使用用户信息
	fmt.Printf("用户邮箱: %s\n", userInfo.Email)
	fmt.Printf("用户名: %s\n", userInfo.Name)
}
