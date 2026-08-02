package pay

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/models"
	paypkg "github.com/grapery/grapery/pkg/pay"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/jwt"

	api "github.com/grapery/common-protoc/gen"
)

// GoogleSignInRequest 前端发送的 Google Sign-In 请求结构
type GoogleSignInRequest struct {
	IDToken string `json:"id_token" binding:"required"` // Google ID Token
}

// GoogleSignInResponse 后端响应结构
type GoogleSignInResponse struct {
	Code        int               `json:"code"`
	Message     string            `json:"msg"`
	Success     bool              `json:"success"`
	UserID      string            `json:"user_id,omitempty"`
	Email       string            `json:"email,omitempty"`
	AccessToken string            `json:"access_token,omitempty"` // 你的应用 token
	Data        *GoogleSignInData `json:"data,omitempty"`
}

// GoogleSignInData 登录数据
type GoogleSignInData struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email,omitempty"`
	Name         string `json:"name,omitempty"`
	Picture      string `json:"picture,omitempty"`
	IsNewUser    bool   `json:"is_new_user"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    int64  `json:"expires_at"`
	SystemUserID int64  `json:"system_user_id"`
}

// GoogleOAuthHandler Google OAuth2 处理器
type GoogleOAuthHandler struct {
	verifier   *paypkg.GoogleSignInVerifier
	jwtWrapper *jwt.JwtWrapper
}

// NewGoogleOAuthHandler 创建新的 Google OAuth2 处理器
func NewGoogleOAuthHandler() *GoogleOAuthHandler {
	// 从配置中创建 Google OAuth2 配置
	googleOAuthConfig := createGoogleOAuthConfig()

	// 创建 Google Sign-In 验证器
	verifier := paypkg.NewGoogleSignInVerifier(googleOAuthConfig)

	// 创建 JWT wrapper（使用与系统一致的密钥和过期时间）
	jwtWrapper := jwt.NewJwtWrapper(utils.SecretKey, utils.ExpirationHours)
	jwtWrapper.Issuer = "grapery"

	return &GoogleOAuthHandler{
		verifier:   verifier,
		jwtWrapper: jwtWrapper,
	}
}

// HandleGoogleSignIn 处理 Google Sign-In 请求
func (h *GoogleOAuthHandler) HandleGoogleSignIn(c *gin.Context) {
	var req GoogleSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, GoogleSignInResponse{
			Code:    400,
			Message: "Invalid request body",
			Success: false,
		})
		return
	}

	if req.IDToken == "" {
		c.JSON(http.StatusBadRequest, GoogleSignInResponse{
			Code:    400,
			Message: "ID token is required",
			Success: false,
		})
		return
	}

	// 检查验证器是否有效
	if !h.verifier.IsValid() {
		logrus.Error("Google OAuth2 verifier is not properly configured")
		c.JSON(http.StatusInternalServerError, GoogleSignInResponse{
			Code:    500,
			Message: "Google OAuth2 service is not available",
			Success: false,
		})
		return
	}

	// 验证 Google ID Token
	var claims *paypkg.GoogleIdentityTokenClaims
	var err error

	logrus.Infof("开始验证 Google ID Token，Client ID: %s", h.verifier.GetClientID())
	claims, err = h.verifier.VerifyToken(req.IDToken)

	if err != nil {
		logrus.Errorf("Google Sign-In verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, GoogleSignInResponse{
			Code:    401,
			Message: "Invalid ID token",
			Success: false,
		})
		return
	}

	// 获取用户标识符
	userID, err := h.verifier.GetUserIdentifier(req.IDToken)
	if err != nil {
		logrus.Errorf("Failed to get user ID: %v", err)
		c.JSON(http.StatusUnauthorized, GoogleSignInResponse{
			Code:    401,
			Message: "Invalid ID token",
			Success: false,
		})
		return
	}

	// 处理第三方登录逻辑
	systemUserID, isNewUser, err := h.handleThirdPartyLogin(c.Request.Context(), userID, claims)
	if err != nil {
		logrus.Errorf("Failed to handle third party login: %v", err)
		c.JSON(http.StatusInternalServerError, GoogleSignInResponse{
			Code:    500,
			Message: "Failed to process login",
			Success: false,
		})
		return
	}

	// 生成应用自己的访问令牌
	appToken, err := h.generateAppToken(userID, claims.Name, claims.Email, systemUserID)
	if err != nil {
		logrus.Errorf("Failed to generate app token: %v", err)
		c.JSON(http.StatusInternalServerError, GoogleSignInResponse{
			Code:    500,
			Message: "Failed to generate access token",
			Success: false,
		})
		return
	}

	response := GoogleSignInResponse{
		Code:    0,
		Message: "success",
		Success: true,
		Data: &GoogleSignInData{
			SystemUserID: systemUserID,
			UserID:       userID,
			Email:        claims.Email,
			Name:         claims.Name,
			Picture:      claims.Picture,
			IsNewUser:    isNewUser,
			AccessToken:  appToken,
			ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(), // 24小时后过期
		},
	}
	responseData, _ := json.Marshal(response)
	logrus.Infof("Google Sign-In successful for user: %s, email: %s, response: %s", userID, claims.Email, string(responseData))
	c.JSON(http.StatusOK, response)
}

// HandleGoogleSignInStatus 检查 Google Sign-In 状态
func (h *GoogleOAuthHandler) HandleGoogleSignInStatus(c *gin.Context) {
	// 获取用户ID（从认证中间件或请求参数）
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "user_id is required",
		})
		return
	}

	// TODO: 从数据库查询用户状态
	// 这里返回简单的状态信息
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"user_id":    userID,
			"is_active":  true,
			"last_login": time.Now().Format(time.RFC3339),
		},
	})
}

// generateAppToken 生成应用自己的 JWT token
func (h *GoogleOAuthHandler) generateAppToken(thirdPartyUserID, thirdPartyName, email string, systemUserID int64) (string, error) {
	// 创建用户信息结构体
	userInfo := &api.UserInfo{
		UserId: systemUserID,
		Email:  email,
		Name:   thirdPartyName, // 使用第三方用户名
	}

	// 使用现有的 JWT 系统生成 token
	token, err := h.jwtWrapper.GenerateToken(userInfo)
	if err != nil {
		logrus.Errorf("Failed to generate JWT token: %v", err)
		return "", err
	}
	logrus.Infof("生成应用自己的 JWT token: %s", token)
	return token, nil
}

// handleThirdPartyLogin 处理第三方登录逻辑
func (h *GoogleOAuthHandler) handleThirdPartyLogin(ctx context.Context, googleUserID string, claims *paypkg.GoogleIdentityTokenClaims) (int64, bool, error) {
	// 1. 首先尝试通过 Google 用户ID查找现有的第三方登录记录
	systemUserID, err := models.GetUserIDByThirdPartyLogin(ctx, models.ProviderGoogle, googleUserID)
	if err == nil {
		// 找到现有记录，更新登录信息
		userInfo := map[string]interface{}{
			"email":          claims.Email,
			"name":           claims.Name,
			"given_name":     claims.GivenName,
			"family_name":    claims.FamilyName,
			"picture":        claims.Picture,
			"locale":         claims.Locale,
			"email_verified": claims.EmailVerified,
			"hosted_domain":  claims.HostedDomain,
		}

		// 更新第三方登录记录
		_, err = models.CreateOrUpdateThirdPartyLogin(ctx, models.ProviderGoogle, googleUserID, claims.Email, claims.Name, systemUserID, userInfo, "", "", nil)
		if err != nil {
			logrus.Errorf("Failed to update third party login record: %v", err)
			return 0, false, err
		}

		logrus.Infof("Found existing user via Google login: systemUserID=%d, googleUserID=%s", systemUserID, googleUserID)
		return systemUserID, false, nil // 不是新用户
	}

	// 2. 如果没有找到 Google 登录记录，尝试通过邮箱查找现有用户
	var existingUser *models.User
	if claims.Email != "" {
		existingUser, err = models.GetUserByEmail(ctx, claims.Email)
		if err != nil {
			logrus.Errorf("Failed to get user by email: %v", err)
			return 0, false, err
		}
	}

	var finalSystemUserID int64
	var isNewUser bool

	if existingUser != nil {
		// 3a. 找到现有用户，关联 Google 登录
		finalSystemUserID = int64(existingUser.ID)
		isNewUser = false

		logrus.Infof("Found existing user by email, linking Google login: systemUserID=%d, email=%s", finalSystemUserID, claims.Email)
	} else {
		// 3b. 没有找到现有用户，创建新用户
		newUser, err := h.createNewUser(ctx, claims)
		if err != nil {
			logrus.Errorf("Failed to create new user: %v", err)
			return 0, false, err
		}

		finalSystemUserID = int64(newUser.ID)
		isNewUser = true

		logrus.Infof("Created new user: systemUserID=%d, email=%s", finalSystemUserID, claims.Email)
	}

	// 4. 创建或更新第三方登录记录
	userInfo := map[string]interface{}{
		"email":          claims.Email,
		"name":           claims.Name,
		"given_name":     claims.GivenName,
		"family_name":    claims.FamilyName,
		"picture":        claims.Picture,
		"locale":         claims.Locale,
		"email_verified": claims.EmailVerified,
		"hosted_domain":  claims.HostedDomain,
	}

	_, err = models.CreateOrUpdateThirdPartyLogin(ctx, models.ProviderGoogle, googleUserID, claims.Email, claims.Name, finalSystemUserID, userInfo, "", "", nil)
	if err != nil {
		logrus.Errorf("Failed to create/update third party login record: %v", err)
		return 0, false, err
	}

	return finalSystemUserID, isNewUser, nil
}

// createNewUser 创建新用户
func (h *GoogleOAuthHandler) createNewUser(ctx context.Context, claims *paypkg.GoogleIdentityTokenClaims) (*models.User, error) {
	// 生成用户名（优先使用全名，其次使用邮箱前缀）
	username := claims.Name
	if username == "" && claims.Email != "" {
		// 从邮箱中提取用户名部分
		emailParts := strings.Split(claims.Email, "@")
		if len(emailParts) > 0 {
			username = emailParts[0]
		} else {
			username = "GoogleUser"
		}
	}
	if username == "" {
		username = "GoogleUser"
	}

	// 创建新用户
	newUser := &models.User{
		Name:   username,
		Email:  claims.Email,
		Status: api.UserStatus_Rest, // 使用默认状态
	}

	err := newUser.Create()
	if err != nil {
		return nil, err
	}

	return newUser, nil
}

// createGoogleOAuthConfig 从全局配置创建 Google OAuth2 配置
func createGoogleOAuthConfig() *paypkg.GoogleOAuthConfig {
	// 从全局配置中获取 Google OAuth2 配置
	var googleOAuthConfig *config.GoogleOAuthConfig
	if config.GlobalConfig.VipPay != nil && config.GlobalConfig.VipPay.GoogleOAuth != nil {
		googleOAuthConfig = config.GlobalConfig.VipPay.GoogleOAuth
	}

	// 如果没有配置，使用默认配置（需要后续补充实际配置）
	if googleOAuthConfig == nil {
		logrus.Warn("Google OAuth2 configuration not found, using default placeholder config")
		googleOAuthConfig = &config.GoogleOAuthConfig{
			ClientID:       "345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb.apps.googleusercontent.com", // TODO: 需要配置实际的 Client ID
			TimeoutSeconds: 60,
			CacheDuration:  24,
		}
	}

	// 转换为 paypkg 的配置结构
	return &paypkg.GoogleOAuthConfig{
		ClientID:       googleOAuthConfig.ClientID,
		TimeoutSeconds: googleOAuthConfig.TimeoutSeconds,
		CacheDuration:  googleOAuthConfig.CacheDuration,
	}
}

// GetGoogleOAuthConfig 获取当前 Google OAuth2 配置信息（用于调试）
func (h *GoogleOAuthHandler) GetGoogleOAuthConfig(c *gin.Context) {
	if !h.verifier.IsValid() {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Google OAuth2 verifier is not properly configured",
		})
		return
	}

	config := gin.H{
		"client_id": h.verifier.GetClientID(),
		"is_valid":  h.verifier.IsValid(),
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    config,
	})
}
