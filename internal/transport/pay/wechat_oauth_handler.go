package pay

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	appservice "github.com/grapestree/fgrapery/grapery/internal/service"
	payservice "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	paymiddleware "github.com/grapestree/fgrapery/grapery/internal/transport/pay/middleware"
	"github.com/sirupsen/logrus"
)

// WeChatSignInRequest 微信登录请求
type WeChatSignInRequest struct {
	Code string `json:"code" binding:"required"` // 微信授权code
}

// WeChatOAuthHandler 微信OAuth处理器
type WeChatOAuthHandler struct {
	client *payservice.WeChatOAuthClient
	repo   OAuthRepository
}

// NewWeChatOAuthHandler 创建微信OAuth处理器
func NewWeChatOAuthHandler() *WeChatOAuthHandler {
	config := createWeChatOAuthConfig()
	client := payservice.NewWeChatOAuthClient(config)

	return &WeChatOAuthHandler{
		client: client,
		repo:   nil,
	}
}

// NewWeChatOAuthHandlerWithRepo 创建带Repository的微信OAuth处理器
func NewWeChatOAuthHandlerWithRepo(repo OAuthRepository) *WeChatOAuthHandler {
	config := createWeChatOAuthConfig()
	client := payservice.NewWeChatOAuthClient(config)

	return &WeChatOAuthHandler{
		client: client,
		repo:   repo,
	}
}

// createWeChatOAuthConfig 创建微信OAuth配置
func createWeChatOAuthConfig() *payservice.WeChatOAuthConfig {
	appID := strings.TrimSpace(os.Getenv("WECHAT_APP_ID"))
	appSecret := strings.TrimSpace(os.Getenv("WECHAT_APP_SECRET"))

	if appID != "" {
		logrus.WithFields(logrus.Fields{
			"wechat_app_id":     appID,
			"wechat_secret_set": appSecret != "",
			"wechat_secret_len": len(appSecret),
		}).Info("WeChat OAuth configured")
	} else {
		logrus.Warn("WECHAT_APP_ID is not set; WeChat sign-in will be unavailable")
	}

	return &payservice.WeChatOAuthConfig{
		AppID:     appID,
		AppSecret: appSecret,
	}
}

// HandleWeChatSignIn 处理微信登录请求
func (h *WeChatOAuthHandler) HandleWeChatSignIn(c *gin.Context) {
	var req WeChatSignInRequest
	if !BindJSON(c, &req) {
		return
	}

	req.Code = normalizeWeChatAuthCode(req.Code)
	if req.Code == "" {
		InvalidParams(c, "code is required")
		return
	}

	// 检查客户端是否有效
	if !h.client.IsValid() {
		logrus.Error("WeChat OAuth client is not properly configured")
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "WeChat OAuth service is not available",
			Message: "WeChat OAuth service is not available",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	logrus.WithFields(logrus.Fields{
		"wechat_app_id": h.client.GetAppID(),
		"code_len":      len(req.Code),
		"client_ip":     c.ClientIP(),
		"user_agent":    c.Request.UserAgent(),
	}).Info("WeChat sign-in: exchanging authorization code")

	// 1. 用code换取access_token
	tokenResp, err := h.client.GetAccessToken(ctx, req.Code)
	if err != nil {
		userMessage := weChatTokenExchangeUserMessage(err)
		logrus.WithFields(logrus.Fields{
			"wechat_app_id": h.client.GetAppID(),
			"code_len":      len(req.Code),
			"client_ip":     c.ClientIP(),
		}).Errorf("WeChat sign-in: access token exchange failed: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     userMessage,
			Message: userMessage,
			Success: false,
		})
		return
	}

	// 2. 获取微信用户信息
	userInfo, err := h.client.GetUserInfo(ctx, tokenResp.AccessToken, tokenResp.OpenID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"wechat_app_id": h.client.GetAppID(),
			"wechat_openid": tokenResp.OpenID,
			"wechat_scope":  tokenResp.Scope,
			"has_unionid":   tokenResp.UnionID != "",
		}).Errorf("WeChat sign-in: fetch user info failed: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Failed to get WeChat user info",
			Message: "Failed to get WeChat user info",
			Success: false,
		})
		return
	}

	logrus.WithFields(logrus.Fields{
		"wechat_openid": userInfo.OpenID,
		"nickname":      userInfo.Nickname,
	}).Info("WeChat sign-in verified successfully")

	// 3. 查找或创建用户
	user, isNewUser, err := h.findOrCreateUser(ctx, userInfo)
	if err != nil {
		logrus.Errorf("Failed to find or create user: %v", err)
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Failed to process user account",
			Message: "Failed to process user account",
			Success: false,
		})
		return
	}

	// 4. 生成JWT token
	jwtToken, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		logrus.Errorf("Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Failed to generate access token",
			Message: "Failed to generate access token",
			Success: false,
		})
		return
	}

	// 5. 生成Refresh Token
	refreshToken, err := auth.GenerateRefreshToken(user.ID)
	if err != nil {
		logrus.Errorf("Failed to generate refresh token: %v", err)
		// 不阻塞登录流程
	}

	expiresIn := int64(24 * 3600) // 24小时

	data := BuildOAuthSignInData(user, "wechat", isNewUser, jwtToken, refreshToken, expiresIn)

	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    0,
		Msg:     "success",
		Message: "success",
		Success: true,
		Data:    data,
	})
}

// findOrCreateUser 查找或创建微信OAuth用户
func (h *WeChatOAuthHandler) findOrCreateUser(ctx context.Context, userInfo *payservice.WeChatUserInfo) (*domain.User, bool, error) {
	now := time.Now().Unix()

	if h.repo != nil {
		if existing, err := h.resolveExistingWeChatUser(ctx, userInfo, now); err == nil && existing != nil {
			return existing, false, nil
		}

		username := generateUsername(userInfo.Nickname, "", userInfo.OpenID, "wechat")
		displayName := userInfo.Nickname
		if displayName == "" {
			displayName = username
		}

		newUserID := uuid.New().String()
		newUser := &domain.User{
			BaseModel: common.BaseModel{
				ID:        newUserID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			SocialStats: common.SocialStats{
				Followers: 0,
				Following: 0,
			},
			Username:             username,
			Email:                wechatPlaceholderEmail(userInfo.OpenID),
			DisplayName:          displayName,
			Avatar:               userInfo.HeadImgURL,
			Status:               "active",
			EmailVerified:        false,
			PendingOAuthPhoneSMS: true,
			ReferralCode:         appservice.GenerateUserReferralCode(newUserID),
			LastLoginAt:          &now,
		}

		if err := h.repo.CreateUser(ctx, newUser); err != nil {
			if recovered, recErr := h.resolveExistingWeChatUser(ctx, userInfo, now); recErr == nil && recovered != nil {
				return recovered, false, nil
			}
			return nil, false, err
		}

		newThirdPartyLogin := &domain.ThirdPartyLogin{
			ID:               uuid.New().String(),
			UserID:           newUser.ID,
			Provider:         domain.ProviderWechat,
			ProviderUserID:   userInfo.OpenID,
			ProviderEmail:    "",
			ProviderUserName: userInfo.Nickname,
			Status:           domain.ThirdPartyLoginStatusNormal,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := h.repo.CreateThirdPartyLogin(ctx, newThirdPartyLogin); err != nil {
			logrus.Errorf("Failed to create WeChat third-party login binding: %v", err)
			_ = h.repo.DeleteUserByID(ctx, newUser.ID)
			if recovered, recErr := h.resolveExistingWeChatUser(ctx, userInfo, now); recErr == nil && recovered != nil {
				return recovered, false, nil
			}
			return nil, false, fmt.Errorf("failed to bind WeChat account: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"provider":       "wechat",
			"providerUserID": userInfo.OpenID,
			"userID":         newUser.ID,
			"nickname":       userInfo.Nickname,
		}).Info("New user created via WeChat login")

		settings := &domain.UserSettings{
			BaseModel: common.BaseModel{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:              newUser.ID,
			Language:            "zh",
			Theme:               "auto",
			EmailNotifications:  true,
			PushNotifications:   true,
			ShowAdultContent:    false,
			ProfileVisibility:   "public",
			AllowComments:       true,
			AllowMessages:       true,
			ShowOnlineStatus:    true,
			ShowPublicStories:   true,
			ShowPublicFragments: true,
			ShowPublicBookmarks: true,
		}
		_ = h.repo.CreateUserSettings(ctx, settings)

		membership := &domain.Membership{
			ID:           uuid.New().String(),
			UserID:       newUser.ID,
			Tier:         "free",
			Status:       "active",
			StartDate:    now,
			AutoRenew:    false,
			TokenQuota:   common.DefaultFreeTierTokenQuota,
			TokenUsed:    0,
			StorageQuota: 1024 * 1024 * 100,
			StorageUsed:  0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		_ = h.repo.CreateMembership(ctx, membership)

		return newUser, true, nil
	}

	// 没有repository，返回基于OAuth信息的临时用户（不持久化）
	username := generateUsername(userInfo.Nickname, "", userInfo.OpenID, "wechat")
	displayName := userInfo.Nickname
	if displayName == "" {
		displayName = username
	}

	logrus.Warn("OAuth handler has no repository, user data will not be persisted")

	return &domain.User{
		BaseModel: common.BaseModel{
			ID:        userInfo.OpenID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		SocialStats: common.SocialStats{
			Followers: 0,
			Following: 0,
		},
		Username:             username,
		DisplayName:          displayName,
		Avatar:               userInfo.HeadImgURL,
		Status:               "active",
		EmailVerified:        false,
		PendingOAuthPhoneSMS: true,
		LastLoginAt:          &now,
	}, true, nil
}

// HandleWeChatSignInStatus 处理微信登录状态查询
func (h *WeChatOAuthHandler) HandleWeChatSignInStatus(c *gin.Context) {
	enabled := h.client.IsValid()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success",
		"message": "success",
		"success": true,
		"data": gin.H{
			"enabled":     enabled,
			"isAvailable": enabled,
			"provider":    "wechat",
			"message":     "WeChat Sign-In is available",
		},
	})
}

// GetWeChatOAuthConfig 获取微信OAuth配置（前端需要的公开信息）
func (h *WeChatOAuthHandler) GetWeChatOAuthConfig(c *gin.Context) {
	appID := h.client.GetAppID()
	enabled := h.client.IsValid()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success",
		"message": "success",
		"success": true,
		"data": gin.H{
			"appId":        appID,
			"app_id":       appID,
			"enabled":      enabled,
			"isAvailable":  enabled,
			"provider":     "wechat",
			"scope":        "snsapi_login",
			"responseType": "code",
			"message":      "WeChat OAuth config",
		},
	})
}

// HandleWeChatLink 处理微信账号绑定请求（需要鉴权）
func (h *WeChatOAuthHandler) HandleWeChatLink(c *gin.Context) {
	// 获取当前登录用户ID
	currentUserID := paymiddleware.GetUserIDFromContext(c)
	if currentUserID == "" {
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Unauthorized",
			Message: "Unauthorized",
			Success: false,
		})
		return
	}

	var req WeChatSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "Invalid request body",
			Message: "Invalid request body",
			Success: false,
		})
		return
	}

	req.Code = normalizeWeChatAuthCode(req.Code)
	if req.Code == "" {
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "Authorization code is required",
			Message: "Authorization code is required",
			Success: false,
		})
		return
	}

	// 检查客户端是否有效
	if !h.client.IsValid() {
		logrus.Error("WeChat OAuth client is not properly configured")
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "WeChat OAuth service is not available",
			Message: "WeChat OAuth service is not available",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	logrus.WithFields(logrus.Fields{
		"wechat_app_id": h.client.GetAppID(),
		"code_len":      len(req.Code),
		"user_id":       currentUserID,
		"client_ip":     c.ClientIP(),
	}).Info("WeChat link: exchanging authorization code")

	// 验证微信授权code
	tokenResp, err := h.client.GetAccessToken(ctx, req.Code)
	if err != nil {
		userMessage := weChatTokenExchangeUserMessage(err)
		logrus.WithFields(logrus.Fields{
			"wechat_app_id": h.client.GetAppID(),
			"code_len":      len(req.Code),
			"user_id":       currentUserID,
		}).Errorf("WeChat link: access token exchange failed: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     userMessage,
			Message: userMessage,
			Success: false,
		})
		return
	}

	userInfo, err := h.client.GetUserInfo(ctx, tokenResp.AccessToken, tokenResp.OpenID)
	if err != nil {
		logrus.Errorf("Failed to get WeChat user info: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Failed to get WeChat user info",
			Message: "Failed to get WeChat user info",
			Success: false,
		})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Repository not available",
			Message: "Repository not available",
			Success: false,
		})
		return
	}

	// 检查该微信账号是否已被其他用户绑定
	existingLogin, err := h.repo.GetThirdPartyLoginByProviderUserID(ctx, domain.ProviderWechat, userInfo.OpenID)
	if err == nil && existingLogin != nil {
		// 已存在绑定
		if existingLogin.UserID != currentUserID {
			// 绑定到其他用户，返回错误
			c.JSON(http.StatusConflict, VipPayAPIResponse{
				Code:    409,
				Msg:     "This WeChat account is already linked to another user",
				Message: "This WeChat account is already linked to another user",
				Success: false,
			})
			return
		}
		// 已绑定到当前用户，更新绑定信息（幂等）
		existingLogin.ProviderUserName = userInfo.Nickname
		existingLogin.UpdatedAt = time.Now().Unix()
		if err := h.repo.UpdateThirdPartyLogin(ctx, existingLogin); err != nil {
			logrus.Errorf("Failed to update WeChat login binding: %v", err)
		}
	} else {
		// 未绑定，创建新绑定
		now := time.Now().Unix()
		newThirdPartyLogin := &domain.ThirdPartyLogin{
			ID:               uuid.New().String(),
			UserID:           currentUserID,
			Provider:         domain.ProviderWechat,
			ProviderUserID:   userInfo.OpenID,
			ProviderEmail:    "",
			ProviderUserName: userInfo.Nickname,
			Status:           domain.ThirdPartyLoginStatusNormal,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := h.repo.CreateThirdPartyLogin(ctx, newThirdPartyLogin); err != nil {
			logrus.Errorf("Failed to create WeChat login binding: %v", err)
			c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
				Code:    500,
				Msg:     "Failed to link WeChat account",
				Message: "Failed to link WeChat account",
				Success: false,
			})
			return
		}
	}

	// 获取当前用户信息
	user, err := h.repo.UserByID(ctx, currentUserID)
	if err != nil {
		logrus.Errorf("Failed to get user: %v", err)
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Failed to get user information",
			Message: "Failed to get user information",
			Success: false,
		})
		return
	}

	// 更新用户头像（如果微信提供了且用户还没有头像）
	if userInfo.HeadImgURL != "" && user.Avatar == "" {
		user.Avatar = userInfo.HeadImgURL
		user.UpdatedAt = time.Now().Unix()
		_ = h.repo.UpdateUser(ctx, user)
	}

	data := BuildOAuthSignInData(user, "wechat", false, "", "", 0)

	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    0,
		Msg:     "success",
		Message: "success",
		Success: true,
		Data:    data,
	})
}

// HandleWeChatUnlink 处理微信账号解绑请求（需要鉴权）
func (h *WeChatOAuthHandler) HandleWeChatUnlink(c *gin.Context) {
	// 获取当前登录用户ID
	currentUserID := paymiddleware.GetUserIDFromContext(c)
	if currentUserID == "" {
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Unauthorized",
			Message: "Unauthorized",
			Success: false,
		})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Repository not available",
			Message: "Repository not available",
			Success: false,
		})
		return
	}

	ctx := c.Request.Context()

	// 删除微信绑定
	if err := h.repo.DeleteThirdPartyLogin(ctx, currentUserID, domain.ProviderWechat); err != nil {
		logrus.Errorf("Failed to unlink WeChat account: %v", err)
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Failed to unlink WeChat account",
			Message: "Failed to unlink WeChat account",
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    0,
		Msg:     "success",
		Message: "success",
		Success: true,
		Data:    gin.H{"message": "WeChat account unlinked successfully"},
	})
}

func normalizeWeChatAuthCode(code string) string {
	return strings.TrimSpace(code)
}

func weChatTokenExchangeUserMessage(err error) string {
	if err == nil {
		return "Failed to get WeChat access token"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "40029"):
		return "WeChat authorization code is invalid or expired. Please sign in again."
	case strings.Contains(msg, "40125"):
		return "WeChat OAuth server configuration is invalid (check WECHAT_APP_SECRET)."
	default:
		return "Failed to get WeChat access token"
	}
}
