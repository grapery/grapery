package pay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	payservice "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	"github.com/sirupsen/logrus"
)

func wechatPlaceholderEmail(openID string) string {
	openID = strings.TrimSpace(openID)
	sum := sha256.Sum256([]byte(openID))
	return fmt.Sprintf("wechat_%s@oauth.grapery.local", hex.EncodeToString(sum[:16]))
}

// syncOAuthPhoneVerificationGate clears PendingOAuthPhoneSMS once phone is verified.
func syncOAuthPhoneVerificationGate(ctx context.Context, repo OAuthRepository, user *domain.User) {
	if repo == nil || user == nil || !UserPhoneVerified(user) || !user.PendingOAuthPhoneSMS {
		return
	}
	user.PendingOAuthPhoneSMS = false
	user.UpdatedAt = time.Now().Unix()
	if err := repo.UpdateUser(ctx, user); err != nil {
		logrus.WithError(err).WithField("user_id", user.ID).Warn("failed to clear pending OAuth phone SMS flag")
	}
}

func (h *WeChatOAuthHandler) resolveExistingWeChatUser(ctx context.Context, userInfo *payservice.WeChatUserInfo, now int64) (*domain.User, error) {
	if h.repo == nil {
		return nil, domain.ErrNotFound
	}

	tryLogin := func(login *domain.ThirdPartyLogin, restore bool) (*domain.User, error) {
		if login == nil {
			return nil, domain.ErrNotFound
		}
		if restore {
			if err := h.repo.RestoreThirdPartyLogin(ctx, login.ID); err != nil {
				logrus.WithError(err).WithField("login_id", login.ID).Warn("failed to restore soft-deleted WeChat binding")
			}
		}
		user, err := h.repo.UserByID(ctx, login.UserID)
		if err != nil || user == nil {
			return nil, err
		}
		user.LastLoginAt = &now
		user.UpdatedAt = now
		if userInfo.HeadImgURL != "" && user.Avatar == "" {
			user.Avatar = userInfo.HeadImgURL
		}
		syncOAuthPhoneVerificationGate(ctx, h.repo, user)
		_ = h.repo.UpdateUser(ctx, user)

		login.ProviderUserName = userInfo.Nickname
		login.UpdatedAt = now
		_ = h.repo.UpdateThirdPartyLogin(ctx, login)

		logrus.WithFields(logrus.Fields{
			"provider":       "wechat",
			"providerUserID": userInfo.OpenID,
			"userID":         user.ID,
			"restored":       restore,
		}).Info("Existing user logged in via WeChat")
		return user, nil
	}

	if login, err := h.repo.GetThirdPartyLoginByProviderUserID(ctx, domain.ProviderWechat, userInfo.OpenID); err == nil && login != nil {
		return tryLogin(login, false)
	}

	if login, err := h.repo.GetThirdPartyLoginByProviderUserIDUnscoped(ctx, domain.ProviderWechat, userInfo.OpenID); err == nil && login != nil {
		return tryLogin(login, true)
	}

	return nil, domain.ErrNotFound
}
