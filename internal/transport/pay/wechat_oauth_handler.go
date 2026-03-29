package pay

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
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
	appID := os.Getenv("WECHAT_APP_ID")
	appSecret := os.Getenv("WECHAT_APP_SECRET")

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

	// 1. 用code换取access_token
	tokenResp, err := h.client.GetAccessToken(ctx, req.Code)
	if err != nil {
		logrus.Errorf("Failed to get WeChat access token: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Failed to get WeChat access token",
			Message: "Failed to get WeChat access token",
			Success: false,
		})
		return
	}

	// 2. 获取微信用户信息
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

	userResp := &OAuthUserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		Bio:         user.Bio,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	data := OAuthSignInData{
		Token:        jwtToken,
		RefreshToken: refreshToken,
		User:         userResp,
		ExpiresIn:    expiresIn,
		IsNewUser:    isNewUser,

		UserID:        user.ID,
		AccessToken:   jwtToken,
		RefreshToken2: refreshToken,
		ExpiresIn2:    expiresIn,
		IsNewUser2:    isNewUser,
	}

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
		// Step 1: 通过 openid 查找已绑定的第三方登录
		thirdPartyLogin, err := h.repo.GetThirdPartyLoginByProviderUserID(ctx, domain.ProviderWechat, userInfo.OpenID)
		if err == nil && thirdPartyLogin != nil {
			// 已有绑定，获取关联的用户
			user, err := h.repo.UserByID(ctx, thirdPartyLogin.UserID)
			if err != nil {
				logrus.Errorf("Failed to get user by ID from third party login: %v", err)
				return nil, false, err
			}

			// 更新登录时间、头像
			user.LastLoginAt = &now
			user.UpdatedAt = now
			if userInfo.HeadImgURL != "" && user.Avatar == "" {
				user.Avatar = userInfo.HeadImgURL
			}
			_ = h.repo.UpdateUser(ctx, user)

			// 更新第三方登录记录
			thirdPartyLogin.UpdatedAt = now
			_ = h.repo.UpdateThirdPartyLogin(ctx, thirdPartyLogin)

			logrus.WithFields(logrus.Fields{
				"provider":       "wechat",
				"providerUserID": userInfo.OpenID,
				"userID":         user.ID,
				"nickname":       userInfo.Nickname,
			}).Info("Existing user logged in via WeChat")

			return user, false, nil
		}

		// Step 2: 创建新用户并绑定第三方登录
		username := generateUsername(userInfo.Nickname, "", userInfo.OpenID, "wechat")
		displayName := userInfo.Nickname
		if displayName == "" {
			displayName = username
		}

		newUser := &domain.User{
			BaseModel: common.BaseModel{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			SocialStats: common.SocialStats{
				Followers: 0,
				Following: 0,
			},
			Username:      username,
			Email:         "", // 微信不一定提供邮箱
			DisplayName:   displayName,
			Avatar:        userInfo.HeadImgURL,
			Status:        "active",
			EmailVerified: false,
			LastLoginAt:   &now,
		}

		if err := h.repo.CreateUser(ctx, newUser); err != nil {
			return nil, false, err
		}

		// 创建第三方登录绑定
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
			logrus.Warnf("Failed to create third party login binding: %v", err)
			// 不阻塞登录流程
		} else {
			logrus.WithFields(logrus.Fields{
				"provider":       "wechat",
				"providerUserID": userInfo.OpenID,
				"userID":         newUser.ID,
				"nickname":       userInfo.Nickname,
			}).Info("New user created via WeChat login")
		}

		// 创建默认用户设置
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

		// 创建默认会员信息
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
		Username:      username,
		DisplayName:   displayName,
		Avatar:        userInfo.HeadImgURL,
		Status:        "active",
		EmailVerified: false,
		LastLoginAt:   &now,
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

	// 验证微信授权code
	tokenResp, err := h.client.GetAccessToken(ctx, req.Code)
	if err != nil {
		logrus.Errorf("Failed to get WeChat access token: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Invalid WeChat authorization code",
			Message: "Invalid WeChat authorization code",
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

	// 返回用户信息
	userResp := &OAuthUserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		Bio:         user.Bio,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	data := OAuthSignInData{
		Token:        "", // Link 操作不需要返回新 token
		RefreshToken: "",
		User:         userResp,
		ExpiresIn:    0,
		IsNewUser:    false,

		UserID:        user.ID,
		AccessToken:   "",
		RefreshToken2: "",
		ExpiresIn2:    0,
		IsNewUser2:    false,
	}

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
