package pay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	payservice "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	paymiddleware "github.com/grapestree/fgrapery/grapery/internal/transport/pay/middleware"
	"github.com/sirupsen/logrus"
)

// VipPayAPIResponse is the canonical vippay envelope used by Android Retrofit and preferred by iOS APIClient.
// iOS `APIClient` will decode `{code,message,data}` first, then fall back to raw payload.
// Android `VipPayApiResponse<T>` expects `{code,msg|message,data}`.
type VipPayAPIResponse struct {
	Code    int         `json:"code"`
	Msg     string      `json:"msg,omitempty"`
	Message string      `json:"message,omitempty"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

// OAuthSignInData is a backward/forward compatible sign-in response payload.
// It intentionally contains both camelCase (iOS) and snake_case (Android) keys.
type OAuthSignInData struct {
	// iOS keys (OAuthResponse.swift)
	Token        string             `json:"token"`
	RefreshToken string             `json:"refreshToken,omitempty"`
	User         *OAuthUserResponse `json:"user,omitempty"`
	ExpiresIn    int64              `json:"expiresIn"`
	IsNewUser    bool               `json:"isNewUser"`
	// Apple / WeChat 登录需完成手机号短信验证（邮箱注册与 Google 登录不要求）
	RequiresPhoneVerification     bool `json:"requiresPhoneVerification"`
	RequiresPhoneVerificationSnake bool `json:"requires_phone_verification,omitempty"`

	// Android keys (VipPayService.kt OAuthSignInResponse)
	UserID        string `json:"user_id,omitempty"`
	AccessToken   string `json:"access_token,omitempty"`
	RefreshToken2 string `json:"refresh_token,omitempty"`
	ExpiresIn2    int64  `json:"expires_in,omitempty"`
	IsNewUser2    bool   `json:"is_new_user,omitempty"`
}

// AppleSignInRequest 前端发送的 Apple Sign-In 请求结构 (匹配 iOS 客户端)
type AppleSignInRequest struct {
	IdentityToken     string `json:"identityToken" binding:"required"` // Apple Identity Token
	AuthorizationCode string `json:"authorizationCode,omitempty"`      // Authorization Code
	User              string `json:"user,omitempty"`                   // Apple User ID
	Nonce             string `json:"nonce,omitempty"`                  // raw nonce (optional; if present it will be verified)
}

// OAuthUserResponse 用户信息响应 (匹配前端 User model)
type OAuthUserResponse struct {
	ID                          string `json:"id"`
	Username                    string `json:"username"`
	Email                       string `json:"email"`
	DisplayName                 string `json:"displayName"`
	Avatar                      string `json:"avatar,omitempty"`
	Bio                         string `json:"bio,omitempty"`
	Status                      string `json:"status"`
	Phone                       string `json:"phone,omitempty"`
	PhoneVerifiedAt             int64  `json:"phoneVerifiedAt,omitempty"`
	RequiresPhoneVerification   bool   `json:"requiresPhoneVerification"`
	PendingOAuthPhoneSMS        bool   `json:"pendingOAuthPhoneSMS"`
	CreatedAt                   int64  `json:"createdAt"`
	UpdatedAt                   int64  `json:"updatedAt"`
}

// OAuthResponse 统一的 OAuth 响应结构 (匹配前端 OAuthResponse)
type OAuthResponse struct {
	Token        string             `json:"token"`
	RefreshToken string             `json:"refreshToken,omitempty"`
	User         *OAuthUserResponse `json:"user"`
	ExpiresIn    int64              `json:"expiresIn"`
	IsNewUser    bool               `json:"isNewUser"`
}

// OAuthRepository OAuth 用户仓库接口（支持跨设备、跨登录方式的账户关联）
type OAuthRepository interface {
	// 用户基础操作
	UserByID(ctx context.Context, id string) (*domain.User, error)
	UserByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
	UpdateUser(ctx context.Context, user *domain.User) error
	CreateUserSettings(ctx context.Context, settings *domain.UserSettings) error
	CreateMembership(ctx context.Context, membership *domain.Membership) error

	// 第三方登录操作（支持 Google/Apple 跨设备登录）
	CreateThirdPartyLogin(ctx context.Context, login *domain.ThirdPartyLogin) error
	GetThirdPartyLoginByProviderUserID(ctx context.Context, provider domain.ThirdPartyProvider, providerUserID string) (*domain.ThirdPartyLogin, error)
	GetThirdPartyLoginByEmail(ctx context.Context, provider domain.ThirdPartyProvider, email string) (*domain.ThirdPartyLogin, error)
	GetThirdPartyLoginsByUserID(ctx context.Context, userID string) ([]*domain.ThirdPartyLogin, error)
	UpdateThirdPartyLogin(ctx context.Context, login *domain.ThirdPartyLogin) error
	DeleteThirdPartyLogin(ctx context.Context, userID string, provider domain.ThirdPartyProvider) error
	// 通过任意第三方登录的 email 查找关联的用户
	GetUserByThirdPartyEmail(ctx context.Context, email string) (*domain.User, error)
	IsAccountReRegistrationBlocked(ctx context.Context, emailNorm, phoneNorm string) (bool, error)
}

// AppleOAuthHandler Apple OAuth2 处理器
type AppleOAuthHandler struct {
	verifier *payservice.AppleSignInVerifier
	repo     OAuthRepository
}

// NewAppleOAuthHandler 创建新的 Apple OAuth2 处理器
func NewAppleOAuthHandler() *AppleOAuthHandler {
	// 从配置中创建 Apple OAuth2 配置
	appleOAuthConfig := createAppleOAuthConfig()

	// 创建 Apple Sign-In 验证器
	verifier := payservice.NewAppleSignInVerifier(appleOAuthConfig)

	return &AppleOAuthHandler{
		verifier: verifier,
		repo:     nil, // 可选：后续可以注入 repository
	}
}

// NewAppleOAuthHandlerWithRepo 创建带 Repository 的 Apple OAuth2 处理器
func NewAppleOAuthHandlerWithRepo(repo OAuthRepository) *AppleOAuthHandler {
	appleOAuthConfig := createAppleOAuthConfig()
	verifier := payservice.NewAppleSignInVerifier(appleOAuthConfig)

	return &AppleOAuthHandler{
		verifier: verifier,
		repo:     repo,
	}
}

// HandleAppleSignIn 处理 Apple Sign-In 请求
func (h *AppleOAuthHandler) HandleAppleSignIn(c *gin.Context) {
	var req AppleSignInRequest
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

	if req.IdentityToken == "" {
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "Identity token is required",
			Message: "Identity token is required",
			Success: false,
		})
		return
	}

	// 检查验证器是否有效
	if !h.verifier.IsValid() {
		logrus.Error("Apple OAuth2 verifier is not properly configured")
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Apple OAuth2 service is not available",
			Message: "Apple OAuth2 service is not available",
			Success: false,
		})
		return
	}

	// 验证 Apple Identity Token
	claims, err := h.verifier.VerifyToken(req.IdentityToken)
	if err != nil {
		logrus.Errorf("Failed to verify Apple identity token: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Invalid Apple identity token",
			Message: "Invalid Apple identity token",
			Success: false,
		})
		return
	}

	// Optional nonce verification (prevents replay). If client sends nonce, verify it matches token claim.
	// Apple returns the SHA256(nonce) value in the identity token's "nonce" claim when request.nonce is set.
	if req.Nonce != "" {
		expected := sha256Hex(req.Nonce)
		if claims.Nonce == "" || claims.Nonce != expected {
			logrus.WithFields(logrus.Fields{
				"expected": expected,
				"actual":   claims.Nonce,
			}).Warn("Apple Sign-In nonce mismatch")
			c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
				Code:    401,
				Msg:     "Invalid Apple nonce",
				Message: "Invalid Apple nonce",
				Success: false,
			})
			return
		}
	}

	// Apple 用户ID（sub claim）
	appleUserID := claims.Subject
	if appleUserID == "" {
		logrus.Error("Apple user ID (sub) not found in token claims")
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "Invalid token: user ID not found",
			Message: "Invalid token: user ID not found",
			Success: false,
		})
		return
	}

	// 使用Apple User ID和email信息
	email := claims.Email
	fullName := claims.FullName

	logrus.WithFields(logrus.Fields{
		"apple_user_id": appleUserID,
		"email":         email,
		"full_name":     fullName,
	}).Info("Apple Sign-In verified successfully")

	// 查找或创建用户
	ctx := c.Request.Context()
	user, isNewUser, err := h.findOrCreateUser(ctx, appleUserID, email, fullName, "apple")
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
	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
	}).Info("Generating JWT token for user")

	jwtToken, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": user.ID,
			"error":   err,
		}).Error("Failed to generate JWT token")
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Failed to generate access token",
			Message: "Failed to generate access token",
			Success: false,
		})
		return
	}

	logrus.WithFields(logrus.Fields{
		"user_id":     user.ID,
		"token_len":   len(jwtToken),
		"token_start": jwtToken[:min(len(jwtToken), 30)] + "...",
	}).Info("JWT token generated successfully")

	// 生成 Refresh Token
	refreshToken, err := auth.GenerateRefreshToken(user.ID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": user.ID,
			"error":   err,
		}).Warn("Failed to generate refresh token (continuing without it)")
		// 不阻塞登录流程，refresh token 可以为空
	} else {
		logrus.WithFields(logrus.Fields{
			"user_id":       user.ID,
			"refresh_len":   len(refreshToken),
			"refresh_start": refreshToken[:min(len(refreshToken), 30)] + "...",
		}).Info("Refresh token generated successfully")
	}

	expiresIn := int64(24 * 3600) // 24小时

	data := BuildOAuthSignInData(user, "apple", isNewUser, jwtToken, refreshToken, expiresIn)

	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    0,
		Msg:     "success",
		Message: "success",
		Success: true,
		Data:    data,
	})
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// findOrCreateUser 查找或创建 OAuth 用户（支持跨设备、跨登录方式的账户关联）
//
// 账户关联策略：
// 1. 首先通过 providerUserID 查找已绑定的第三方登录记录
// 2. 如果未找到，通过 email 查找是否有其他登录方式已绑定的用户
// 3. 如果找到用户，创建新的第三方登录绑定
// 4. 如果未找到，创建新用户并绑定第三方登录
func (h *AppleOAuthHandler) findOrCreateUser(ctx context.Context, providerUserID, email, displayName, provider string) (*domain.User, bool, error) {
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

			// 更新登录时间
			user.LastLoginAt = &now
			user.UpdatedAt = now
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

		// Step 2: 通过 email 查找是否有已存在的用户（可能通过其他方式注册）
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

			// 更新登录时间
			existingUser.LastLoginAt = &now
			existingUser.UpdatedAt = now
			_ = h.repo.UpdateUser(ctx, existingUser)

			return existingUser, false, nil
		}

		// Step 4: 用户不存在，创建新用户并绑定第三方登录
		username := generateUsername(displayName, email, providerUserID, provider)
		if displayName == "" {
			displayName = username
		}

		if email != "" {
			emailNorm := strings.ToLower(strings.TrimSpace(email))
			blocked, err := h.repo.IsAccountReRegistrationBlocked(ctx, emailNorm, "")
			if err != nil {
				return nil, false, err
			}
			if blocked {
				return nil, false, fmt.Errorf("this email cannot be used within 30 days after account deletion")
			}
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
			Username:               username,
			Email:                  email,
			DisplayName:            displayName,
			Status:                 "active",
			EmailVerified:          true, // OAuth 登录邮箱已验证
			PendingOAuthPhoneSMS:   true, // 首次 Apple 注册需短信验证（中国大陆）
			LastLoginAt:            &now,
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
			BaseModel: common.BaseModel{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:              newUser.ID,
			Language:            "en",
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
		BaseModel: common.BaseModel{
			ID:        providerUserID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		SocialStats: common.SocialStats{
			Followers: 0,
			Following: 0,
		},
		Username:             username,
		Email:                email,
		DisplayName:          displayName,
		Status:               "active",
		EmailVerified:        true,
		PendingOAuthPhoneSMS: true,
		LastLoginAt:          &now,
	}, true, nil
}

// generateUsername 生成用户名
func generateUsername(displayName, email, providerUserID, provider string) string {
	if displayName != "" {
		return displayName
	}
	if email != "" {
		return email
	}
	if len(providerUserID) > 8 {
		return provider + "_user_" + providerUserID[:8]
	}
	return provider + "_user_" + providerUserID
}

// HandleAppleSignInStatus 处理 Apple Sign-In 状态查询
func (h *AppleOAuthHandler) HandleAppleSignInStatus(c *gin.Context) {
	enabled := h.verifier.IsValid()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success", // legacy field (Android)
		"message": "success", // new field (iOS)
		"success": true,
		"data": gin.H{
			// Keep both names for compatibility across clients.
			"enabled":     enabled,
			"isAvailable": enabled,
			"provider":    "apple",
			"message":     "Apple Sign-In is available",
		},
	})
}

// GetAppleOAuthConfig 获取 Apple OAuth 配置（前端需要的公开信息）
func (h *AppleOAuthHandler) GetAppleOAuthConfig(c *gin.Context) {
	bundleID := h.verifier.GetBundleID()
	enabled := h.verifier.IsValid()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success", // legacy field (Android)
		"message": "success", // new field (iOS)
		"success": true,
		"data": gin.H{
			// Canonical OAuth-style fields (iOS expects these)
			"clientId":     bundleID,
			"redirectUri":  "",
			"scope":        "name email",
			"responseType": "id_token",
			"state":        nil,

			// Legacy/alternate keys (Android / older clients)
			"client_id":     bundleID,
			"redirect_uri":  "",
			"response_type": "id_token",
			"bundleId":      bundleID,
			"bundle_id":     bundleID,
			"enabled":       enabled,
			"isAvailable":   enabled,
			"provider":      "apple",
			"scopes":        []string{"name", "email"},
			"message":       "Apple OAuth config",
		},
	})
}

// HandleAppleLink 处理 Apple 账号绑定请求（需要鉴权）
func (h *AppleOAuthHandler) HandleAppleLink(c *gin.Context) {
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

	var req AppleSignInRequest
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

	if req.IdentityToken == "" {
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "Identity token is required",
			Message: "Identity token is required",
			Success: false,
		})
		return
	}

	// 检查验证器是否有效
	if !h.verifier.IsValid() {
		logrus.Error("Apple OAuth2 verifier is not properly configured")
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Apple OAuth2 service is not available",
			Message: "Apple OAuth2 service is not available",
			Success: false,
		})
		return
	}

	// 验证 Apple Identity Token
	claims, err := h.verifier.VerifyToken(req.IdentityToken)
	if err != nil {
		logrus.Errorf("Failed to verify Apple identity token: %v", err)
		c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
			Code:    401,
			Msg:     "Invalid Apple identity token",
			Message: "Invalid Apple identity token",
			Success: false,
		})
		return
	}

	// Optional nonce verification
	if req.Nonce != "" {
		expected := sha256Hex(req.Nonce)
		if claims.Nonce == "" || claims.Nonce != expected {
			logrus.WithFields(logrus.Fields{
				"expected": expected,
				"actual":   claims.Nonce,
			}).Warn("Apple Sign-In nonce mismatch")
			c.JSON(http.StatusUnauthorized, VipPayAPIResponse{
				Code:    401,
				Msg:     "Invalid Apple nonce",
				Message: "Invalid Apple nonce",
				Success: false,
			})
			return
		}
	}

	appleUserID := claims.Subject
	if appleUserID == "" {
		logrus.Error("Apple user ID (sub) not found in token claims")
		c.JSON(http.StatusBadRequest, VipPayAPIResponse{
			Code:    400,
			Msg:     "Invalid token: user ID not found",
			Message: "Invalid token: user ID not found",
			Success: false,
		})
		return
	}

	email := claims.Email
	fullName := claims.FullName

	ctx := c.Request.Context()

	// 检查该 Apple 账号是否已被其他用户绑定
	if h.repo != nil {
		existingLogin, err := h.repo.GetThirdPartyLoginByProviderUserID(ctx, domain.ProviderApple, appleUserID)
		if err == nil && existingLogin != nil {
			// 已存在绑定
			if existingLogin.UserID != currentUserID {
				// 绑定到其他用户，返回错误
				c.JSON(http.StatusConflict, VipPayAPIResponse{
					Code:    409,
					Msg:     "This Apple account is already linked to another user",
					Message: "This Apple account is already linked to another user",
					Success: false,
				})
				return
			}
			// 已绑定到当前用户，更新绑定信息（幂等）
			existingLogin.ProviderEmail = email
			existingLogin.ProviderUserName = fullName
			existingLogin.UpdatedAt = time.Now().Unix()
			if err := h.repo.UpdateThirdPartyLogin(ctx, existingLogin); err != nil {
				logrus.Errorf("Failed to update Apple login binding: %v", err)
			}
		} else {
			// 未绑定，创建新绑定
			now := time.Now().Unix()
			newThirdPartyLogin := &domain.ThirdPartyLogin{
				ID:               uuid.New().String(),
				UserID:           currentUserID,
				Provider:         domain.ProviderApple,
				ProviderUserID:   appleUserID,
				ProviderEmail:    email,
				ProviderUserName: fullName,
				Status:           domain.ThirdPartyLoginStatusNormal,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if err := h.repo.CreateThirdPartyLogin(ctx, newThirdPartyLogin); err != nil {
				logrus.Errorf("Failed to create Apple login binding: %v", err)
				c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
					Code:    500,
					Msg:     "Failed to link Apple account",
					Message: "Failed to link Apple account",
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

		data := BuildOAuthSignInData(user, "apple", false, "", "", 0)

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

// HandleAppleUnlink 处理 Apple 账号解绑请求（需要鉴权）
func (h *AppleOAuthHandler) HandleAppleUnlink(c *gin.Context) {
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

	// 删除 Apple 绑定
	if err := h.repo.DeleteThirdPartyLogin(ctx, currentUserID, domain.ProviderApple); err != nil {
		logrus.Errorf("Failed to unlink Apple account: %v", err)
		c.JSON(http.StatusInternalServerError, VipPayAPIResponse{
			Code:    500,
			Msg:     "Failed to unlink Apple account",
			Message: "Failed to unlink Apple account",
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    0,
		Msg:     "success",
		Message: "success",
		Success: true,
		Data:    gin.H{"message": "Apple account unlinked successfully"},
	})
}

// createAppleOAuthConfig 创建 Apple OAuth2 配置
func createAppleOAuthConfig() *payservice.AppleOAuthConfig {
	// 从环境变量读取配置
	bundleID := os.Getenv("APPLE_BUNDLE_ID")
	if bundleID == "" {
		bundleID = "com.rankquantity.voyager" // 默认 Bundle ID
	}

	return &payservice.AppleOAuthConfig{
		BundleID:       bundleID,
		TimeoutSeconds: 30,
		CacheDuration:  1,
	}
}
