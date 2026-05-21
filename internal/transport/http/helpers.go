package http

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	svcerrs "github.com/grapestree/fgrapery/grapery/internal/service"
)

// BindJSON 统一JSON参数绑定和校验
// 如果绑定失败，自动返回参数错误响应并返回false
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		InvalidParams(c, formatBindError(err))
		return false
	}
	return true
}

// BindQuery 统一Query参数绑定和校验
// 如果绑定失败，自动返回参数错误响应并返回false
func BindQuery(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		InvalidParams(c, formatBindError(err))
		return false
	}
	return true
}

// BindURI 统一URI参数绑定和校验
// 如果绑定失败，自动返回参数错误响应并返回false
func BindURI(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindUri(obj); err != nil {
		InvalidParams(c, formatBindError(err))
		return false
	}
	return true
}

// formatBindError 格式化绑定错误信息，提取更友好的错误消息
func formatBindError(err error) string {
	errMsg := err.Error()
	// 提取关键错误信息，去掉冗余的字段路径
	if strings.Contains(errMsg, "required") {
		return "缺少必需参数"
	}
	if strings.Contains(errMsg, "invalid") {
		return "参数格式错误"
	}
	return errMsg
}

// RequireUserID 获取当前用户ID，如果未认证则返回错误响应并返回false
func RequireUserID(c *gin.Context) (string, bool) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return "", false
	}
	return userID, true
}

// GetUserID 获取当前用户ID（可能为空，用于可选认证的场景）
func GetUserID(c *gin.Context) string {
	return authPkg.GetUserID(c)
}

// RequireParam 检查必需的路径参数，如果为空则返回错误响应并返回false
func RequireParam(c *gin.Context, paramName string) (string, bool) {
	value := c.Param(paramName)
	if value == "" {
		InvalidParams(c, paramName+" is required")
		return "", false
	}
	return value, true
}

// HandleError 统一错误处理，根据错误信息自动判断错误类型
// 支持标准错误类型和字符串匹配
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	errMsg := err.Error()

	// 检查标准错误类型
	if errors.Is(err, domain.ErrNotFound) {
		NotFound(c, errMsg)
		return
	}
	if errors.Is(err, domain.ErrUnauthorized) {
		Unauthorized(c, errMsg)
		return
	}
	if errors.Is(err, domain.ErrForbidden) {
		Forbidden(c, errMsg)
		return
	}
	if errors.Is(err, domain.ErrInvalidInput) {
		InvalidParams(c, errMsg)
		return
	}
	if errors.Is(err, domain.ErrAlreadyExists) {
		DuplicateEntry(c, errMsg)
		return
	}
	if errors.Is(err, svcerrs.ErrAccountDeletionRiskAckRequired) || errors.Is(err, svcerrs.ErrAccountDeletionInvalidDeletionSMSCodeFmt) {
		InvalidParams(c, errMsg)
		return
	}
	if errors.Is(err, svcerrs.ErrAccountDeletionSMSProofMissing) ||
		errors.Is(err, svcerrs.ErrAccountDeletionVerifiedPhoneRequired) ||
		errors.Is(err, svcerrs.ErrAccountDeletionSMSOnlyForActiveAccounts) {
		Forbidden(c, errMsg)
		return
	}
	if errors.Is(err, svcerrs.ErrContactOTPExpired) ||
		errors.Is(err, svcerrs.ErrAccountContactInvalidCodeFmt) ||
		errors.Is(err, svcerrs.ErrAccountContactInvalidEmail) {
		InvalidParams(c, errMsg)
		return
	}
	if errors.Is(err, svcerrs.ErrAccountContactVerifyLocked) ||
		errors.Is(err, svcerrs.ErrAccountContactModifyLocked) {
		Forbidden(c, errMsg)
		return
	}
	if errors.Is(err, svcerrs.ErrAccountContactEmailRegistered) {
		DuplicateEntry(c, errMsg)
		return
	}
	if errors.Is(err, svcerrs.ErrAccountContactCacheRequired) {
		InternalError(c, errMsg)
		return
	}

	// 通过错误消息字符串匹配常见错误类型
	errMsgLower := strings.ToLower(errMsg)

	// 未授权相关
	if strings.Contains(errMsgLower, "unauthorized") ||
		strings.Contains(errMsgLower, "not authenticated") ||
		strings.Contains(errMsgLower, "authentication failed") {
		Unauthorized(c, errMsg)
		return
	}

	// 权限不足
	if strings.Contains(errMsgLower, "forbidden") ||
		strings.Contains(errMsgLower, "permission denied") ||
		strings.Contains(errMsgLower, "can only") ||
		strings.Contains(errMsgLower, "cannot") {
		Forbidden(c, errMsg)
		return
	}

	// 资源不存在
	if strings.Contains(errMsgLower, "not found") ||
		strings.Contains(errMsgLower, "does not exist") ||
		strings.Contains(errMsgLower, "不存在") {
		NotFound(c, errMsg)
		return
	}

	// 重复记录
	if strings.Contains(errMsgLower, "already exists") ||
		strings.Contains(errMsgLower, "duplicate") ||
		strings.Contains(errMsgLower, "already") {
		DuplicateEntry(c, errMsg)
		return
	}

	// Token相关错误
	if strings.Contains(errMsgLower, "token expired") ||
		strings.Contains(errMsgLower, "expired token") {
		TokenExpired(c)
		return
	}
	if strings.Contains(errMsgLower, "invalid token") ||
		strings.Contains(errMsgLower, "token invalid") {
		InvalidToken(c)
		return
	}

	// 速率限制
	if strings.Contains(errMsgLower, "rate limit") ||
		strings.Contains(errMsgLower, "too many requests") ||
		strings.Contains(errMsgLower, "throttle") {
		Error(c, CodeRateLimitExceed, errMsg)
		return
	}

	// Aliyun SMS / Redis OTP backend (avoid matching generic "invalid" in SDK messages).
	if strings.Contains(errMsgLower, "sms verification requires redis cache") {
		InternalError(c, errMsg)
		return
	}
	if strings.Contains(errMsgLower, "aliyun sms not configured") ||
		strings.Contains(errMsgLower, "aliyun sendsms") ||
		strings.Contains(errMsgLower, "aliyun sms failed") ||
		strings.Contains(errMsgLower, "aliyun sms:") {
		InternalError(c, "contact verification temporarily unavailable")
		return
	}

	// 参数错误
	if strings.Contains(errMsgLower, "invalid") ||
		strings.Contains(errMsgLower, "bad request") ||
		strings.Contains(errMsgLower, "missing") ||
		strings.Contains(errMsgLower, "required") {
		InvalidParams(c, errMsg)
		return
	}

	// 默认作为通用错误处理
	Error(c, CodeError, errMsg)
}
