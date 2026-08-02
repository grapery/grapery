package thirdpart

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleLogin represents the Google OAuth2 login configuration
type GoogleLogin struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}

// UserInfo represents the Google user information
type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// NewGoogleLogin creates a new GoogleLogin instance
func NewGoogleLogin(clientID, clientSecret, redirectURL string) *GoogleLogin {
	return &GoogleLogin{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/gmail.readonly",
		},
	}
}

// Init initializes the OAuth2 config
func (g *GoogleLogin) Init() {
	g.Config = &oauth2.Config{
		ClientID:     g.ClientID,
		ClientSecret: g.ClientSecret,
		RedirectURL:  g.RedirectURL,
		Scopes:       g.Scopes,
		Endpoint:     google.Endpoint,
	}
}

// GetAuthURL returns the Google OAuth2 authorization URL
func (g *GoogleLogin) GetAuthURL(state string) string {
	return g.Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange exchanges the authorization code for tokens
func (g *GoogleLogin) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Config.Exchange(ctx, code)
}

// GetUserInfo retrieves the user information using the access token
func (g *GoogleLogin) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := g.Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %v", err)
	}

	return &userInfo, nil
}

// RefreshToken refreshes the access token using the refresh token
func (g *GoogleLogin) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	newToken, err := g.Config.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %v", err)
	}

	return newToken, nil
}

// GetGmailService creates a new Gmail service client using the access token
func (g *GoogleLogin) GetGmailClient(ctx context.Context, token *oauth2.Token) *http.Client {
	return g.Config.Client(ctx, token)
}
