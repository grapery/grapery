package thirdpart

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
)

// 实现line的鉴权登录
// LineLogin 实现line的OAuth2鉴权登录
type LineLogin struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}

// NewLineLogin creates a new Line OAuth2 client
func NewLineLogin(clientID, clientSecret, redirectURL string, scopes []string) *LineLogin {
	login := &LineLogin{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}

	login.Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://access.line.me/oauth2/v2.1/authorize",
			TokenURL: "https://api.line.me/oauth2/v2.1/token",
		},
	}

	return login
}

// GetAuthURL returns the Line OAuth2 authorization URL
func (l *LineLogin) GetAuthURL(state string) string {
	return l.Config.AuthCodeURL(state)
}

// Exchange exchanges the authorization code for an access token
func (l *LineLogin) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return l.Config.Exchange(ctx, code)
}

// GetUserProfile fetches the user profile from Line API
func (l *LineLogin) GetUserProfile(ctx context.Context, token *oauth2.Token) (map[string]interface{}, error) {
	client := l.Config.Client(ctx, token)
	resp, err := client.Get("https://api.line.me/v2/profile")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var profile map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	return profile, nil
}
