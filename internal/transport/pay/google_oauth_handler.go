package pay

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	payservice "github.com/grapestree/fgrapery/grapery/internal/service/pay"
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
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, OAuthErrorResponse{
			Code:    400,
			Message: "Invalid request body",
			Success: false,
		})
		return
	}

	if req.IDToken == "" {
		c.JSON(http.StatusBadRequest, OAuthErrorResponse{
			Code:    400,
			Message: "ID token is required",
			Success: false,
		})
		return
	}

	// 检查验证器是否有效
	if !h.verifier.IsValid() {
		logrus.Error("Google OAuth2 verifier is not properly configured")
		c.JSON(http.StatusInternalServerError, OAuthErrorResponse{
			Code:    500,
			Message: "Google OAuth2 service is not available",
			Success: false,
		})
		return
	}

	// 验证 Google ID Token
	claims, err := h.verifier.VerifyToken(req.IDToken)
	if err != nil {
		logrus.Errorf("Failed to verify Google ID token: %v", err)
		c.JSON(http.StatusUnauthorized, OAuthErrorResponse{
			Code:    401,
			Message: "Invalid Google ID token",
			Success: false,
		})
		return
	}

	// Google 用户ID（sub claim）
	googleUserID := claims.Subject
	if googleUserID == "" {
		logrus.Error("Google user ID (sub) not found in token claims")
		c.JSON(http.StatusBadRequest, OAuthErrorResponse{
			Code:    400,
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
	user, isNewUser, err := h.findOrCreateUser(ctx, googleUserID, email, name, avatar, "google")
	if err != nil {
		logrus.Errorf("Failed to find or create user: %v", err)
		c.JSON(http.StatusInternalServerError, OAuthErrorResponse{
			Code:    500,
			Message: "Failed to process user account",
			Success: false,
		})
		return
	}

	// 生成 JWT token
	jwtToken, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		logrus.Errorf("Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, OAuthErrorResponse{
			Code:    500,
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

	// 返回前端期望的格式
	c.JSON(http.StatusOK, OAuthResponse{
		Token:        jwtToken,
		RefreshToken: refreshToken,
		User: &OAuthUserResponse{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Avatar:      user.Avatar,
			Bio:         user.Bio,
			Status:      user.Status,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		},
		ExpiresIn: expiresIn,
		IsNewUser: isNewUser,
	})
}

// findOrCreateUser 查找或创建 Google OAuth 用户（支持跨设备、跨登录方式的账户关联）
//
// 账户关联策略：
// 1. 首先通过 providerUserID 查找已绑定的第三方登录记录
// 2. 如果未找到，通过 email 查找是否有其他登录方式已绑定的用户
// 3. 如果找到用户，创建新的第三方登录绑定
// 4. 如果未找到，创建新用户并绑定第三方登录
func (h *GoogleOAuthHandler) findOrCreateUser(ctx context.Context, providerUserID, email, displayName, avatar, provider string) (*domain.User, bool, error) {
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

			// 更新登录时间和头像
			user.LastLoginAt = &now
			user.UpdatedAt = now
			if avatar != "" && user.Avatar == "" {
				user.Avatar = avatar
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
		if email != "" {
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
			EmailVerified: true, // OAuth 登录邮箱已验证
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
		EmailVerified: true,
		LastLoginAt:   &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, true, nil
}

// HandleGoogleSignInStatus 处理 Google Sign-In 状态查询
func (h *GoogleOAuthHandler) HandleGoogleSignInStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success",
		"success": true,
		"data": gin.H{
			"enabled": h.verifier.IsValid(),
			"message": "Google Sign-In is available",
		},
	})
}

// GetGoogleOAuthConfig 获取 Google OAuth 配置（前端需要的公开信息）
func (h *GoogleOAuthHandler) GetGoogleOAuthConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success",
		"success": true,
		"data": gin.H{
			"client_id": h.verifier.GetClientID(),
			"enabled":   h.verifier.IsValid(),
		},
	})
}

// createGoogleOAuthConfig 创建 Google OAuth2 配置
func createGoogleOAuthConfig() *payservice.GoogleOAuthConfig {
	// 从环境变量读取配置
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		clientID = "YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com" // 默认 Client ID
	}

	return &payservice.GoogleOAuthConfig{
		ClientID:       clientID,
		TimeoutSeconds: 30,
		CacheDuration:  1,
	}
}
