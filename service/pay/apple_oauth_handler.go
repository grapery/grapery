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

// AppleSignInRequest 前端发送的 Apple Sign-In 请求结构
type AppleSignInRequest struct {
	IdentityToken string `json:"identity_token" binding:"required"` // Apple Identity Token
	FullName      string `json:"full_name,omitempty"`               // 用户全名（仅首次登录时有）
}

// AppleSignInResponse 后端响应结构
type AppleSignInResponse struct {
	Code        int              `json:"code"`
	Message     string           `json:"msg"`
	Success     bool             `json:"success"`
	UserID      string           `json:"user_id,omitempty"`
	Email       string           `json:"email,omitempty"`
	AccessToken string           `json:"access_token,omitempty"` // 你的应用 token
	Data        *AppleSignInData `json:"data,omitempty"`
}

// AppleSignInData 登录数据
type AppleSignInData struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email,omitempty"`
	IsNewUser    bool   `json:"is_new_user"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    int64  `json:"expires_at"`
	SystemUserID int64  `json:"system_user_id"`
}

// AppleOAuthHandler Apple OAuth2 处理器
type AppleOAuthHandler struct {
	verifier   *paypkg.AppleSignInVerifier
	jwtWrapper *jwt.JwtWrapper
}

// NewAppleOAuthHandler 创建新的 Apple OAuth2 处理器
func NewAppleOAuthHandler() *AppleOAuthHandler {
	// 从配置中创建 Apple OAuth2 配置
	appleOAuthConfig := createAppleOAuthConfig()

	// 创建 Apple Sign-In 验证器
	verifier := paypkg.NewAppleSignInVerifier(appleOAuthConfig)

	// 创建 JWT wrapper（使用与系统一致的密钥和过期时间）
	jwtWrapper := jwt.NewJwtWrapper(utils.SecretKey, utils.ExpirationHours) // 使用系统统一的密钥和过期时间
	jwtWrapper.Issuer = "grapery"

	return &AppleOAuthHandler{
		verifier:   verifier,
		jwtWrapper: jwtWrapper,
	}
}

// HandleAppleSignIn 处理 Apple Sign-In 请求
func (h *AppleOAuthHandler) HandleAppleSignIn(c *gin.Context) {
	var req AppleSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, AppleSignInResponse{
			Code:    400,
			Message: "Invalid request body",
			Success: false,
		})
		return
	}

	if req.IdentityToken == "" {
		c.JSON(http.StatusBadRequest, AppleSignInResponse{
			Code:    400,
			Message: "Identity token is required",
			Success: false,
		})
		return
	}

	// 检查验证器是否有效
	if !h.verifier.IsValid() {
		logrus.Error("Apple OAuth2 verifier is not properly configured")
		c.JSON(http.StatusInternalServerError, AppleSignInResponse{
			Code:    500,
			Message: "Apple OAuth2 service is not available",
			Success: false,
		})
		return
	}

	// 验证 Apple Identity Token
	var claims *paypkg.AppleIdentityTokenClaims
	var err error

	logrus.Infof("开始验证 Apple Identity Token，Bundle ID: %s", h.verifier.GetBundleID())
	claims, err = h.verifier.VerifyToken(req.IdentityToken)

	if err != nil {
		logrus.Errorf("Apple Sign-In verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, AppleSignInResponse{
			Code:    401,
			Message: "Invalid identity token",
			Success: false,
		})
		return
	}

	// 获取用户标识符
	userID, err := h.verifier.GetUserIdentifier(req.IdentityToken)
	if err != nil {
		logrus.Errorf("Failed to get user ID: %v", err)
		c.JSON(http.StatusUnauthorized, AppleSignInResponse{
			Code:    401,
			Message: "Invalid identity token",
			Success: false,
		})
		return
	}

	// TODO: 这里可以：
	// 1. 检查用户是否已存在
	// 2. 创建新用户或更新现有用户
	// 3. 生成你自己的 JWT token 或 session

	// 处理第三方登录逻辑
	systemUserID, isNewUser, err := h.handleThirdPartyLogin(c.Request.Context(), userID, claims)
	if err != nil {
		logrus.Errorf("Failed to handle third party login: %v", err)
		c.JSON(http.StatusInternalServerError, AppleSignInResponse{
			Code:    500,
			Message: "Failed to process login",
			Success: false,
		})
		return
	}

	// 生成应用自己的访问令牌
	appToken, err := h.generateAppToken(userID, claims.FullName, claims.Email, systemUserID)
	if err != nil {
		logrus.Errorf("Failed to generate app token: %v", err)
		c.JSON(http.StatusInternalServerError, AppleSignInResponse{
			Code:    500,
			Message: "Failed to generate access token",
			Success: false,
		})
		return
	}

	response := AppleSignInResponse{
		Code:    0,
		Message: "success",
		Success: true,
		Data: &AppleSignInData{
			SystemUserID: systemUserID,
			UserID:       userID,
			Email:        claims.Email,
			IsNewUser:    isNewUser,
			AccessToken:  appToken,
			ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(), // 24小时后过期
		},
	}
	responseData, _ := json.Marshal(response)
	logrus.Infof("Apple Sign-In successful for user: %s, email: %s, response: %s", userID, claims.Email, string(responseData))
	logrus.Infof("Apple Sign-In successful for user: %s, email: %s", userID, claims.Email)
	c.JSON(http.StatusOK, response)
}

// HandleAppleSignInStatus 检查 Apple Sign-In 状态
func (h *AppleOAuthHandler) HandleAppleSignInStatus(c *gin.Context) {
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
func (h *AppleOAuthHandler) generateAppToken(thirdPartyUserID, thirdPartyName, email string, systemUserID int64) (string, error) {
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
func (h *AppleOAuthHandler) handleThirdPartyLogin(ctx context.Context, appleUserID string, claims *paypkg.AppleIdentityTokenClaims) (int64, bool, error) {
	// 1. 首先尝试通过 Apple 用户ID查找现有的第三方登录记录
	systemUserID, err := models.GetUserIDByThirdPartyLogin(ctx, models.ProviderApple, appleUserID)
	if err == nil {
		// 找到现有记录，更新登录信息
		userInfo := map[string]interface{}{
			"email":            claims.Email,
			"full_name":        claims.FullName,
			"email_verified":   claims.EmailVerified,
			"is_private_email": claims.IsPrivateEmail,
			"auth_time":        claims.AuthTime,
		}

		// 更新第三方登录记录
		_, err = models.CreateOrUpdateThirdPartyLogin(ctx, models.ProviderApple, appleUserID, claims.Email, claims.FullName, systemUserID, userInfo, "", "", nil)
		if err != nil {
			logrus.Errorf("Failed to update third party login record: %v", err)
			return 0, false, err
		}

		logrus.Infof("Found existing user via Apple login: systemUserID=%d, appleUserID=%s", systemUserID, appleUserID)
		return systemUserID, false, nil // 不是新用户
	}

	// 2. 如果没有找到 Apple 登录记录，尝试通过邮箱查找现有用户
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
		// 3a. 找到现有用户，关联 Apple 登录
		finalSystemUserID = int64(existingUser.ID)
		isNewUser = false

		logrus.Infof("Found existing user by email, linking Apple login: systemUserID=%d, email=%s", finalSystemUserID, claims.Email)
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
		"email":            claims.Email,
		"full_name":        claims.FullName,
		"email_verified":   claims.EmailVerified,
		"is_private_email": claims.IsPrivateEmail,
		"auth_time":        claims.AuthTime,
	}

	_, err = models.CreateOrUpdateThirdPartyLogin(ctx, models.ProviderApple, appleUserID, claims.Email, claims.FullName, finalSystemUserID, userInfo, "", "", nil)
	if err != nil {
		logrus.Errorf("Failed to create/update third party login record: %v", err)
		return 0, false, err
	}

	return finalSystemUserID, isNewUser, nil
}

// createNewUser 创建新用户
func (h *AppleOAuthHandler) createNewUser(ctx context.Context, claims *paypkg.AppleIdentityTokenClaims) (*models.User, error) {
	// 生成用户名（如果没有提供全名，使用邮箱前缀）
	username := claims.FullName
	if username == "" && claims.Email != "" {
		// 从邮箱中提取用户名部分
		emailParts := strings.Split(claims.Email, "@")
		if len(emailParts) > 0 {
			username = emailParts[0]
		} else {
			username = "AppleUser"
		}
	}
	if username == "" {
		username = "AppleUser"
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

// createAppleOAuthConfig 从全局配置创建 Apple OAuth2 配置
func createAppleOAuthConfig() *paypkg.AppleOAuthConfig {
	// 从全局配置中获取 Apple OAuth2 配置
	var appleOAuthConfig *config.AppleOAuthConfig
	if config.GlobalConfig.VipPay != nil && config.GlobalConfig.VipPay.AppleOAuth != nil {
		appleOAuthConfig = config.GlobalConfig.VipPay.AppleOAuth
	}

	// 如果没有配置，使用默认配置（需要后续补充实际配置）
	if appleOAuthConfig == nil {
		logrus.Warn("Apple OAuth2 configuration not found, using default placeholder config")
		appleOAuthConfig = &config.AppleOAuthConfig{
			BundleID:       "com.rankquantity.voyager", // TODO: 需要配置实际的Bundle ID
			TimeoutSeconds: 30,
			CacheDuration:  24,
		}
	}

	// 转换为 paypkg 的配置结构
	return &paypkg.AppleOAuthConfig{
		BundleID:       appleOAuthConfig.BundleID,
		TimeoutSeconds: appleOAuthConfig.TimeoutSeconds,
		CacheDuration:  appleOAuthConfig.CacheDuration,
	}
}

// GetAppleOAuthConfig 获取当前 Apple OAuth2 配置信息（用于调试）
func (h *AppleOAuthHandler) GetAppleOAuthConfig(c *gin.Context) {
	if !h.verifier.IsValid() {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Apple OAuth2 verifier is not properly configured",
		})
		return
	}

	config := gin.H{
		"bundle_id": h.verifier.GetBundleID(),
		"is_valid":  h.verifier.IsValid(),
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    config,
	})
}
