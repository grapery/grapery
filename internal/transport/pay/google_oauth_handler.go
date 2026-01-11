package pay

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	payservice "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	paymiddleware "github.com/grapestree/fgrapery/grapery/internal/transport/pay/middleware"
	"github.com/sirupsen/logrus"
)

// GoogleSignInRequest 前端发送的 Google Sign-In 请求结构 (匹配 iOS 客户端)
type GoogleSignInRequest struct {
	IDToken      string `json:"idToken" binding:"required"` // Google ID Token
	AccessToken  string `json:"accessToken,omitempty"`      // Access Token
	RefreshToken string `json:"refreshToken,omitempty"`     // Refresh Token
}

// GoogleOAuthHandler Google OAuth2 处理器
type GoogleOAuthHandler struct {
	verifier *payservice.GoogleSignInVerifier
	repo     OAuthRepository
}

// NewGoogleOAuthHandler 创建新的 Google OAuth2 处理器
func NewGoogleOAuthHandler() *GoogleOAuthHandler {
	// 从配置中创建 Google OAuth2 配置
	googleOAuthConfig := createGoogleOAuthConfig()

	// 创建 Google Sign-In 验证器
	verifier := payservice.NewGoogleSignInVerifier(googleOAuthConfig)

	return &GoogleOAuthHandler{
		verifier: verifier,
		repo:     nil,
	}
}

// NewGoogleOAuthHandlerWithRepo 创建带 Repository 的 Google OAuth2 处理器
func NewGoogleOAuthHandlerWithRepo(repo OAuthRepository) *GoogleOAuthHandler {
	googleOAuthConfig := createGoogleOAuthConfig()
	verifier := payservice.NewGoogleSignInVerifier(googleOAuthConfig)

	return &GoogleOAuthHandler{
		verifier: verifier,
		repo:     repo,
	}
}

// HandleGoogleSignIn 处理 Google Sign-In 请求
func (h *GoogleOAuthHandler) HandleGoogleSignIn(c *gin.Context) {
	var req GoogleSignInRequest
	if !BindJSON(c, &req) {
		return
	}

	if req.IDToken == "" {
		InvalidParams(c, "ID token is required")
		return
	}

	// 检查验证器是否有效
	if !h.verifier.IsValid() {
		logrus.Error("Google OAuth2 verifier is not properly configured")
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Google OAuth2 service is not available",
			Message: "Google OAuth2 service is not available",
			Success: false,
		})
		return
	}

	// 验证 Google ID Token
	claims, err := h.verifier.VerifyToken(req.IDToken)
	if err != nil {
		logrus.Errorf("Failed to verify Google ID token: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Invalid Google ID token",
			Message: "Invalid Google ID token",
			Success: false,
		})
		return
	}

	// Google 用户ID（sub claim）
	googleUserID := claims.Subject
	if googleUserID == "" {
		logrus.Error("Google user ID (sub) not found in token claims")
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "Invalid token: user ID not found",
			Message: "Invalid token: user ID not found",
			Success: false,
		})
		return
	}

	// 使用Google User ID和email信息
	email := claims.Email
	name := claims.Name
	avatar := claims.Picture

	logrus.WithFields(logrus.Fields{
		"google_user_id": googleUserID,
		"email":          email,
		"name":           name,
	}).Info("Google Sign-In verified successfully")

	// 查找或创建用户
	ctx := c.Request.Context()
	user, isNewUser, err := h.findOrCreateUser(ctx, googleUserID, email, claims.EmailVerified, name, avatar, "google")
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

	// 生成 JWT token
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

	// 生成 Refresh Token
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

// findOrCreateUser 查找或创建 Google OAuth 用户（支持跨设备、跨登录方式的账户关联）
//
// 账户关联策略：
// 1. 首先通过 providerUserID 查找已绑定的第三方登录记录
// 2. 如果未找到，通过 email 查找是否有其他登录方式已绑定的用户
// 3. 如果找到用户，创建新的第三方登录绑定
// 4. 如果未找到，创建新用户并绑定第三方登录
func (h *GoogleOAuthHandler) findOrCreateUser(ctx context.Context, providerUserID, email string, emailVerified bool, displayName, avatar, provider string) (*domain.User, bool, error) {
	now := time.Now().Unix()
	providerType := domain.ThirdPartyProvider(provider)

	// 如果有 repository，使用完整的账户关联逻辑
	if h.repo != nil {
		// Step 1: 通过 providerUserID 查找已绑定的第三方登录
		thirdPartyLogin, err := h.repo.GetThirdPartyLoginByProviderUserID(ctx, providerType, providerUserID)
		if err == nil && thirdPartyLogin != nil {
			// 已有绑定，获取关联的用户
			user, err := h.repo.UserByID(ctx, thirdPartyLogin.UserID)
			if err != nil {
				logrus.Errorf("Failed to get user by ID from third party login: %v", err)
				return nil, false, err
			}

			// 更新登录时间、头像、邮箱验证状态（只提升，不降低）
			user.LastLoginAt = &now
			user.UpdatedAt = now
			if avatar != "" && user.Avatar == "" {
				user.Avatar = avatar
			}
			if email != "" && emailVerified && !user.EmailVerified {
				user.EmailVerified = true
			}
			_ = h.repo.UpdateUser(ctx, user)

			// 更新第三方登录记录
			thirdPartyLogin.UpdatedAt = now
			_ = h.repo.UpdateThirdPartyLogin(ctx, thirdPartyLogin)

			logrus.WithFields(logrus.Fields{
				"provider":       provider,
				"providerUserID": providerUserID,
				"userID":         user.ID,
				"email":          user.Email,
			}).Info("Existing user logged in via third party")

			return user, false, nil
		}

		// Step 2: 通过 email 查找是否有已存在的用户
		var existingUser *domain.User
		// Only allow email-based linking if Google explicitly says the email is verified.
		if email != "" && emailVerified {
			// 先查找是否有其他第三方登录使用相同 email
			existingUser, _ = h.repo.GetUserByThirdPartyEmail(ctx, email)
			if existingUser == nil {
				// 再查找是否有直接注册的用户
				existingUser, _ = h.repo.UserByEmail(ctx, email)
			}
		}

		if existingUser != nil {
			// Step 3: 用户存在，创建新的第三方登录绑定（账户关联）
			newThirdPartyLogin := &domain.ThirdPartyLogin{
				ID:               uuid.New().String(),
				UserID:           existingUser.ID,
				Provider:         providerType,
				ProviderUserID:   providerUserID,
				ProviderEmail:    email,
				ProviderUserName: displayName,
				Status:           domain.ThirdPartyLoginStatusNormal,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if err := h.repo.CreateThirdPartyLogin(ctx, newThirdPartyLogin); err != nil {
				logrus.Warnf("Failed to create third party login binding: %v", err)
				// 不阻塞登录流程
			} else {
				logrus.WithFields(logrus.Fields{
					"provider":       provider,
					"providerUserID": providerUserID,
					"userID":         existingUser.ID,
					"email":          email,
				}).Info("New third party login linked to existing user")
			}

			// 更新登录时间和头像
			existingUser.LastLoginAt = &now
			existingUser.UpdatedAt = now
			if avatar != "" && existingUser.Avatar == "" {
				existingUser.Avatar = avatar
			}
			if email != "" && emailVerified && !existingUser.EmailVerified {
				existingUser.EmailVerified = true
			}
			_ = h.repo.UpdateUser(ctx, existingUser)

			return existingUser, false, nil
		}

		// Step 4: 用户不存在，创建新用户并绑定第三方登录
		username := generateUsername(displayName, email, providerUserID, provider)
		if displayName == "" {
			displayName = username
		}

		newUser := &domain.User{
			ID:            uuid.New().String(),
			Username:      username,
			Email:         email,
			DisplayName:   displayName,
			Avatar:        avatar,
			Status:        "active",
			EmailVerified: email != "" && emailVerified,
			LastLoginAt:   &now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := h.repo.CreateUser(ctx, newUser); err != nil {
			return nil, false, err
		}

		// 创建第三方登录绑定
		newThirdPartyLogin := &domain.ThirdPartyLogin{
			ID:               uuid.New().String(),
			UserID:           newUser.ID,
			Provider:         providerType,
			ProviderUserID:   providerUserID,
			ProviderEmail:    email,
			ProviderUserName: displayName,
			Status:           domain.ThirdPartyLoginStatusNormal,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := h.repo.CreateThirdPartyLogin(ctx, newThirdPartyLogin); err != nil {
			logrus.Warnf("Failed to create third party login for new user: %v", err)
			// 不阻塞登录流程
		}

		// 创建默认用户设置
		settings := &domain.UserSettings{
			ID:                 uuid.New().String(),
			UserID:             newUser.ID,
			Language:           "en",
			Theme:              "auto",
			EmailNotifications: true,
			PushNotifications:  true,
			ShowAdultContent:   false,
			ProfileVisibility:  "public",
			AllowComments:      true,
			AllowMessages:      true,
			ShowOnlineStatus:   true,
			UpdatedAt:          now,
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
			TokenQuota:   10000,
			TokenUsed:    0,
			StorageQuota: 1024 * 1024 * 100,
			StorageUsed:  0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		_ = h.repo.CreateMembership(ctx, membership)

		logrus.WithFields(logrus.Fields{
			"provider":       provider,
			"providerUserID": providerUserID,
			"userID":         newUser.ID,
			"email":          email,
		}).Info("New user created via third party login")

		return newUser, true, nil
	}

	// 没有 repository，返回基于 OAuth 信息的临时用户（不持久化）
	username := generateUsername(displayName, email, providerUserID, provider)
	if displayName == "" {
		displayName = username
	}

	logrus.Warn("OAuth handler has no repository, user data will not be persisted")

	return &domain.User{
		ID:            providerUserID,
		Username:      username,
		Email:         email,
		DisplayName:   displayName,
		Avatar:        avatar,
		Status:        "active",
		EmailVerified: email != "" && emailVerified,
		LastLoginAt:   &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, true, nil
}

// HandleGoogleSignInStatus 处理 Google Sign-In 状态查询
func (h *GoogleOAuthHandler) HandleGoogleSignInStatus(c *gin.Context) {
	enabled := h.verifier.IsValid()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success", // legacy field (Android)
		"message": "success", // new field (iOS)
		"success": true,
		"data": gin.H{
			"enabled":     enabled,
			"isAvailable": enabled,
			"provider":    "google",
			"message":     "Google Sign-In is available",
		},
	})
}

// GetGoogleOAuthConfig 获取 Google OAuth 配置（前端需要的公开信息）
func (h *GoogleOAuthHandler) GetGoogleOAuthConfig(c *gin.Context) {
	clientID := h.verifier.GetClientID()
	enabled := h.verifier.IsValid()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success", // legacy field (Android)
		"message": "success", // new field (iOS)
		"success": true,
		"data": gin.H{
			// Canonical OAuth-style fields (iOS expects these)
			"clientId":     clientID,
			"redirectUri":  "",
			"scope":        "openid email profile",
			"responseType": "id_token",
			"state":        nil,

			// Legacy/alternate keys (Android / older clients)
			"client_id":     clientID,
			"redirect_uri":  "",
			"response_type": "id_token",
			"enabled":       enabled,
			"isAvailable":   enabled,
			"provider":      "google",
			"scopes":        []string{"openid", "email", "profile"},
			"message":       "Google OAuth config",
		},
	})
}

// HandleGoogleLink 处理 Google 账号绑定请求（需要鉴权）
func (h *GoogleOAuthHandler) HandleGoogleLink(c *gin.Context) {
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

	var req GoogleSignInRequest
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

	if req.IDToken == "" {
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "ID token is required",
			Message: "ID token is required",
			Success: false,
		})
		return
	}

	// 检查验证器是否有效
	if !h.verifier.IsValid() {
		logrus.Error("Google OAuth2 verifier is not properly configured")
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Google OAuth2 service is not available",
			Message: "Google OAuth2 service is not available",
			Success: false,
		})
		return
	}

	// 验证 Google ID Token
	claims, err := h.verifier.VerifyToken(req.IDToken)
	if err != nil {
		logrus.Errorf("Failed to verify Google ID token: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Invalid Google ID token",
			Message: "Invalid Google ID token",
			Success: false,
		})
		return
	}

	googleUserID := claims.Subject
	if googleUserID == "" {
		logrus.Error("Google user ID (sub) not found in token claims")
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "Invalid token: user ID not found",
			Message: "Invalid token: user ID not found",
			Success: false,
		})
		return
	}

	email := claims.Email
	name := claims.Name
	avatar := claims.Picture

	ctx := c.Request.Context()

	// 检查该 Google 账号是否已被其他用户绑定
	if h.repo != nil {
		existingLogin, err := h.repo.GetThirdPartyLoginByProviderUserID(ctx, domain.ProviderGoogle, googleUserID)
		if err == nil && existingLogin != nil {
			// 已存在绑定
			if existingLogin.UserID != currentUserID {
				// 绑定到其他用户，返回错误
				c.JSON(http.StatusConflict, VipPayAPIResponse{
					Code:    409,
					Msg:     "This Google account is already linked to another user",
					Message: "This Google account is already linked to another user",
					Success: false,
				})
				return
			}
			// 已绑定到当前用户，更新绑定信息（幂等）
			existingLogin.ProviderEmail = email
			existingLogin.ProviderUserName = name
			existingLogin.UpdatedAt = time.Now().Unix()
			if err := h.repo.UpdateThirdPartyLogin(ctx, existingLogin); err != nil {
				logrus.Errorf("Failed to update Google login binding: %v", err)
			}
		} else {
			// 未绑定，创建新绑定
			now := time.Now().Unix()
			newThirdPartyLogin := &domain.ThirdPartyLogin{
				ID:               uuid.New().String(),
				UserID:           currentUserID,
				Provider:         domain.ProviderGoogle,
				ProviderUserID:   googleUserID,
				ProviderEmail:    email,
				ProviderUserName: name,
				Status:           domain.ThirdPartyLoginStatusNormal,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if err := h.repo.CreateThirdPartyLogin(ctx, newThirdPartyLogin); err != nil {
				logrus.Errorf("Failed to create Google login binding: %v", err)
				c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
					Code:    500,
					Msg:     "Failed to link Google account",
					Message: "Failed to link Google account",
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

		// 更新用户头像（如果Google提供了且用户还没有头像）
		if avatar != "" && user.Avatar == "" {
			user.Avatar = avatar
			user.UpdatedAt = time.Now().Unix()
			_ = h.repo.UpdateUser(ctx, user)
		}

		// 返回用户信息（类似 signin 响应）
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
		return
	}

	c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
		Code:    500,
		Msg:     "Repository not available",
		Message: "Repository not available",
		Success: false,
	})
}

// HandleGoogleUnlink 处理 Google 账号解绑请求（需要鉴权）
func (h *GoogleOAuthHandler) HandleGoogleUnlink(c *gin.Context) {
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

	// 删除 Google 绑定
	if err := h.repo.DeleteThirdPartyLogin(ctx, currentUserID, domain.ProviderGoogle); err != nil {
		logrus.Errorf("Failed to unlink Google account: %v", err)
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Failed to unlink Google account",
			Message: "Failed to unlink Google account",
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    0,
		Msg:     "success",
		Message: "success",
		Success: true,
		Data:    gin.H{"message": "Google account unlinked successfully"},
	})
}

// createGoogleOAuthConfig 创建 Google OAuth2 配置
func createGoogleOAuthConfig() *payservice.GoogleOAuthConfig {
	// 优先级：环境变量 > 配置文件 > 默认值
	clientID := os.Getenv("GOOGLE_CLIENT_ID")

	// 如果环境变量未设置，尝试从配置文件读取
	if clientID == "" {
		// 尝试读取 vippay.json 配置文件
		configPath := os.Getenv("VIPPAY_CONFIG_PATH")
		if configPath == "" {
			configPath = "vippay.json"
		}

		if data, err := os.ReadFile(configPath); err == nil {
			// 解析 JSON 配置文件
			var config struct {
				OAuth struct {
					Google struct {
						ClientID string `json:"client_id"`
					} `json:"google"`
				} `json:"oauth"`
			}
			if err := json.Unmarshal(data, &config); err == nil && config.OAuth.Google.ClientID != "" {
				clientID = config.OAuth.Google.ClientID
			}
		}
	}

	// 如果仍未设置，使用默认值
	if clientID == "" {
		clientID = "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com" // 默认 Client ID
	}

	return &payservice.GoogleOAuthConfig{
		ClientID:       clientID,
		TimeoutSeconds: 30,
		CacheDuration:  1,
	}
}
