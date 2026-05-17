package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
)

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`    // 1=成功, 0=失败
	Message string      `json:"message"` // 消息
	Data    interface{} `json:"data"`    // 数据
}

// 错误码定义
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

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（自定义消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
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

	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// ErrorWithData 错误响应（带数据）
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	// Record error metrics
	recordHTTPError(c, code)

	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
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

func RateLimitExceeded(c *gin.Context, message string) {
	Error(c, CodeRateLimitExceed, message)
}

func TokenExpired(c *gin.Context) {
	Error(c, CodeTokenExpired, "token expired")
}

func InvalidToken(c *gin.Context) {
	Error(c, CodeInvalidToken, "invalid token")
}

// Paginated 分页响应
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

func SuccessPaginated(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	Success(c, PaginatedResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
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
