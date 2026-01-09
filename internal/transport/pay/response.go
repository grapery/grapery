package pay

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
)

// 错误码定义（与 http 包保持一致）
const (
	CodeSuccess         = 1  // 成功
	CodeError           = 0  // 通用失败
	CodeInvalidParams   = -1 // 参数错误
	CodeUnauthorized    = -2 // 认证失败
	CodeForbidden       = -3 // 权限不足
	CodeNotFound        = -4 // 资源不存在
	CodeInternalError   = -5 // 服务器错误
	CodeDuplicateEntry  = -6 // 重复记录
	CodeRateLimitExceed = -7 // 超过速率限制
	CodeTokenExpired    = -8 // Token 过期
	CodeInvalidToken    = -9 // Token 无效
)

// Success 成功响应（使用 VipPayAPIResponse 格式）
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    0,
		Msg:     "success",
		Message: "success",
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（自定义消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    0,
		Msg:     message,
		Message: message,
		Success: true,
		Data:    data,
	})
}

// Error 错误响应（使用 VipPayAPIResponse 格式）
func Error(c *gin.Context, code int, message string) {
	statusCode := http.StatusOK // 业务错误仍返回 200，通过 code 区分

	// 特殊错误码映射到 HTTP 状态码
	switch code {
	case CodeUnauthorized, CodeTokenExpired, CodeInvalidToken:
		statusCode = http.StatusUnauthorized
	case CodeForbidden:
		statusCode = http.StatusForbidden
	case CodeNotFound:
		statusCode = http.StatusNotFound
	case CodeInternalError:
		statusCode = http.StatusInternalServerError
	}

	// Record error metrics
	recordHTTPError(c, code)

	c.JSON(statusCode, VipPayAPIResponse{
		Code:    code,
		Msg:     message,
		Message: message,
		Success: false,
		Data:    nil,
	})
}

// ErrorWithData 错误响应（带数据）
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	// Record error metrics
	recordHTTPError(c, code)

	c.JSON(http.StatusOK, VipPayAPIResponse{
		Code:    code,
		Msg:     message,
		Message: message,
		Success: false,
		Data:    data,
	})
}

// 常用错误响应快捷方法

func InvalidParams(c *gin.Context, message string) {
	Error(c, CodeInvalidParams, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, CodeUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, CodeForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, CodeNotFound, message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, CodeInternalError, message)
}

func DuplicateEntry(c *gin.Context, message string) {
	Error(c, CodeDuplicateEntry, message)
}

func TokenExpired(c *gin.Context) {
	Error(c, CodeTokenExpired, "token expired")
}

func InvalidToken(c *gin.Context) {
	Error(c, CodeInvalidToken, "invalid token")
}

// SuccessLegacy 成功响应（使用 gin.H 格式，用于兼容旧代码）
func SuccessLegacy(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": data,
	})
}

// ErrorLegacy 错误响应（使用 gin.H 格式，用于兼容旧代码）
func ErrorLegacy(c *gin.Context, code int, message string) {
	statusCode := http.StatusOK
	switch code {
	case CodeUnauthorized, CodeTokenExpired, CodeInvalidToken:
		statusCode = http.StatusUnauthorized
	case CodeForbidden:
		statusCode = http.StatusForbidden
	case CodeNotFound:
		statusCode = http.StatusNotFound
	case CodeInternalError:
		statusCode = http.StatusInternalServerError
	}

	// Record error metrics
	recordHTTPError(c, code)

	c.JSON(statusCode, gin.H{
		"code": code,
		"msg":  message,
	})
}

// recordHTTPError 记录 HTTP 错误到 Prometheus metrics
func recordHTTPError(c *gin.Context, errorCode int) {
	metrics := telemetry.GetDefaultMetrics()
	if metrics == nil {
		return
	}

	// 获取请求开始时间
	var duration time.Duration
	if startTime, exists := c.Get("request_start_time"); exists {
		if start, ok := startTime.(time.Time); ok {
			duration = time.Since(start)
		}
	}

	// 如果无法获取开始时间，使用当前时间（duration 为 0）
	if duration == 0 {
		duration = 0
	}

	// 获取 HTTP 方法和路径
	method := c.Request.Method
	path := c.Request.URL.Path
	if c.FullPath() != "" {
		path = c.FullPath()
	}

	// 将错误码转换为字符串
	errorCodeStr := strconv.Itoa(errorCode)

	// 记录错误 metrics
	metrics.RecordHTTPError(errorCodeStr, method, path, duration)
}
