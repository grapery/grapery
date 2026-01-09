package pay

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	paymiddleware "github.com/grapestree/fgrapery/grapery/internal/transport/pay/middleware"
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

// RequireUserID 获取当前用户ID（字符串），如果未认证则返回错误响应并返回false
func RequireUserID(c *gin.Context) (string, bool) {
	userID := paymiddleware.GetUserIDFromContext(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return "", false
	}
	return userID, true
}

// RequireUserIDInt64 获取当前用户ID（int64），如果未认证则返回错误响应并返回false
// 兼容旧代码中使用 c.Get("user_id") 的方式
func RequireUserIDInt64(c *gin.Context) (int64, bool) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		Unauthorized(c, "User not authenticated")
		return 0, false
	}

	userID, ok := userIDInterface.(int64)
	if !ok {
		InternalError(c, "Invalid user ID")
		return 0, false
	}
	return userID, true
}

// GetUserID 获取当前用户ID（可能为空，用于可选认证的场景）
func GetUserID(c *gin.Context) string {
	return paymiddleware.GetUserIDFromContext(c)
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

// RequireQueryInt 获取必需的查询参数并转换为int，如果不存在或转换失败则返回错误响应并返回false
func RequireQueryInt(c *gin.Context, paramName string) (int, bool) {
	value := c.Query(paramName)
	if value == "" {
		InvalidParams(c, paramName+" is required")
		return 0, false
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		InvalidParams(c, paramName+" must be a valid integer")
		return 0, false
	}
	return intValue, true
}

// RequireQueryTime 获取必需的查询参数并解析为时间，如果不存在或解析失败则返回错误响应并返回false
func RequireQueryTime(c *gin.Context, paramName string) (*time.Time, bool) {
	value := c.Query(paramName)
	if value == "" {
		InvalidParams(c, paramName+" is required")
		return nil, false
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		InvalidParams(c, paramName+" must be a valid RFC3339 time format")
		return nil, false
	}
	return &t, true
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
