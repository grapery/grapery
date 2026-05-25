package pay

import (
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ProviderRequiresPhoneVerification is true for OAuth providers that must pass SMS gate in China.
func ProviderRequiresPhoneVerification(provider string) bool {
	p := strings.ToLower(strings.TrimSpace(provider))
	return p == "apple" || p == "wechat"
}

// UserPhoneVerified reports whether the user has completed phone verification (timestamp set).
func UserPhoneVerified(u *domain.User) bool {
	return u != nil && u.PhoneVerifiedAt != nil && *u.PhoneVerifiedAt > 0
}

// RequiresPhoneVerification is true for Apple/WeChat OAuth when the account still has the
// first-login SMS pending flag (new signup via Apple/WeChat only; cleared after verify).
func RequiresPhoneVerification(provider string, u *domain.User) bool {
	if u == nil || !ProviderRequiresPhoneVerification(provider) {
		return false
	}
	if UserPhoneVerified(u) {
		return false
	}
	return u.PendingOAuthPhoneSMS
}

// BuildOAuthUserResponse maps domain user + OAuth provider to the vippay user payload.
func BuildOAuthUserResponse(u *domain.User, provider string) *OAuthUserResponse {
	if u == nil {
		return nil
	}
	req := RequiresPhoneVerification(provider, u)
	resp := &OAuthUserResponse{
		ID:                        u.ID,
		Username:                  u.Username,
		Email:                     u.Email,
		DisplayName:               u.DisplayName,
		Avatar:                    u.Avatar,
		Bio:                       u.Bio,
		Status:                    u.Status,
		Phone:                     u.Phone,
		RequiresPhoneVerification: req,
		PendingOAuthPhoneSMS:      u.PendingOAuthPhoneSMS,
		CreatedAt:                 u.CreatedAt,
		UpdatedAt:                 u.UpdatedAt,
	}
	if u.PhoneVerifiedAt != nil && *u.PhoneVerifiedAt > 0 {
		resp.PhoneVerifiedAt = *u.PhoneVerifiedAt
	}
	return resp
}

// BuildOAuthSignInData builds the canonical OAuth sign-in / link envelope.
func BuildOAuthSignInData(u *domain.User, provider string, isNewUser bool, token, refreshToken string, expiresIn int64) OAuthSignInData {
	userResp := BuildOAuthUserResponse(u, provider)
	req := RequiresPhoneVerification(provider, u)
	return OAuthSignInData{
		Token:                         token,
		RefreshToken:                  refreshToken,
		User:                          userResp,
		ExpiresIn:                     expiresIn,
		IsNewUser:                     isNewUser,
		RequiresPhoneVerification:     req,
		RequiresPhoneVerificationSnake: req,
		UserID:                        u.ID,
		AccessToken:                   token,
		RefreshToken2:                 refreshToken,
		ExpiresIn2:                    expiresIn,
		IsNewUser2:                    isNewUser,
	}
}
