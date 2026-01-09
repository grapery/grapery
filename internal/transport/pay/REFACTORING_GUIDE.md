# Pay Handler 重构指南

## 概述

本次重构统一了 pay 目录下的参数校验和错误处理机制，并集成了 Prometheus 错误监控。

## 改造内容

### 1. 统一的辅助函数（helpers.go）

#### 参数绑定和校验
- `BindJSON(c *gin.Context, obj interface{}) bool` - JSON参数绑定
- `BindQuery(c *gin.Context, obj interface{}) bool` - Query参数绑定
- `BindURI(c *gin.Context, obj interface{}) bool` - URI参数绑定

#### 用户ID获取
- `RequireUserID(c *gin.Context) (string, bool)` - 必需用户ID（字符串，未认证时返回错误）
- `RequireUserIDInt64(c *gin.Context) (int64, bool)` - 必需用户ID（int64，用于兼容旧代码）
- `GetUserID(c *gin.Context) string` - 可选用户ID（可能为空）

#### 参数获取
- `RequireParam(c *gin.Context, paramName string) (string, bool)` - 必需的路径参数
- `RequireQueryInt(c *gin.Context, paramName string) (int, bool)` - 必需的查询参数（int类型）
- `RequireQueryTime(c *gin.Context, paramName string) (*time.Time, bool)` - 必需的查询参数（时间类型）

#### 错误处理
- `HandleError(c *gin.Context, err error)` - 统一错误处理，自动判断错误类型

### 2. 统一的响应函数（response.go）

#### 成功响应
- `Success(c *gin.Context, data interface{})` - 使用 VipPayAPIResponse 格式
- `SuccessWithMessage(c *gin.Context, message string, data interface{})` - 自定义消息
- `SuccessLegacy(c *gin.Context, data interface{})` - 使用 gin.H 格式（兼容旧代码）

#### 错误响应
- `Error(c *gin.Context, code int, message string)` - 使用 VipPayAPIResponse 格式，自动记录 Prometheus metrics
- `ErrorWithData(c *gin.Context, code int, message string, data interface{})` - 错误响应（带数据）
- `ErrorLegacy(c *gin.Context, code int, message string)` - 使用 gin.H 格式（兼容旧代码）

#### 快捷错误响应函数
- `InvalidParams(c *gin.Context, message string)` - 参数错误 (-1)
- `Unauthorized(c *gin.Context, message string)` - 认证失败 (-2)
- `Forbidden(c *gin.Context, message string)` - 权限不足 (-3)
- `NotFound(c *gin.Context, message string)` - 资源不存在 (-4)
- `InternalError(c *gin.Context, message string)` - 服务器错误 (-5)
- `DuplicateEntry(c *gin.Context, message string)` - 重复记录 (-6)
- `TokenExpired(c *gin.Context)` - Token 过期 (-8)
- `InvalidToken(c *gin.Context)` - Token 无效 (-9)

### 3. 改造模式

#### 参数校验改造

**改造前：**
```go
var req struct {
    LogIDs    []uint `json:"log_ids" binding:"required"`
    BillingID string `json:"billing_id" binding:"required"`
}
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
        "code": 400,
        "msg":  "Invalid request",
    })
    return
}
```

**改造后：**
```go
var req struct {
    LogIDs    []uint `json:"log_ids" binding:"required"`
    BillingID string `json:"billing_id" binding:"required"`
}
if !BindJSON(c, &req) {
    return
}
```

#### 用户ID获取改造

**改造前：**
```go
userIDInterface, exists := c.Get("user_id")
if !exists {
    c.JSON(http.StatusUnauthorized, gin.H{
        "code": 401,
        "msg":  "User not authenticated",
    })
    return
}
userID, ok := userIDInterface.(int64)
if !ok {
    c.JSON(http.StatusInternalServerError, gin.H{
        "code": 500,
        "msg":  "Invalid user ID",
    })
    return
}
```

**改造后：**
```go
userID, ok := RequireUserIDInt64(c)
if !ok {
    return
}
```

#### 错误处理改造

**改造前：**
```go
logs, total, err := h.logService.QueryLogs(...)
if err != nil {
    h.logger.WithError(err).Error("Failed to query token usage logs")
    c.JSON(http.StatusInternalServerError, gin.H{
        "code": 500,
        "msg":  "Failed to query logs",
    })
    return
}
```

**改造后：**
```go
logs, total, err := h.logService.QueryLogs(...)
if err != nil {
    h.logger.WithError(err).Error("Failed to query token usage logs")
    HandleError(c, err)
    return
}
```

#### 成功响应改造

**改造前：**
```go
c.JSON(http.StatusOK, gin.H{
    "code": 0,
    "msg":  "success",
    "data": data,
})
```

**改造后：**
```go
SuccessLegacy(c, data)
```

或使用 VipPayAPIResponse 格式：
```go
Success(c, data)
```

### 4. Prometheus 监控集成

所有错误响应函数都会自动记录到 Prometheus metrics：
- `HTTPErrorTotal` - 错误总数
- `HTTPErrorDuration` - 错误发生时间分布
- `HTTPErrorDistribution` - 错误分布分位数
- `HTTPErrorPercentiles` - 按错误码聚合的分位数

## 已改造的文件

- ✅ `helpers.go` - 统一的辅助函数
- ✅ `response.go` - 统一的响应函数和 Prometheus 集成
- ✅ `token_usage_log_handler.go` - Token用量日志处理器（完整改造）

## 待改造的文件

以下文件需要按照相同模式进行改造：

- `google_oauth_handler.go` - Google OAuth 处理器
- `apple_oauth_handler.go` - Apple OAuth 处理器
- `iap_handler.go` - IAP 处理器
- `badge_handler.go` - Badge 处理器
- `legal_handler.go` - Legal 处理器

## 注意事项

1. **响应格式兼容**：
   - 使用 `Success` / `Error` 函数（VipPayAPIResponse 格式）
   - 使用 `SuccessLegacy` / `ErrorLegacy` 函数（gin.H 格式，兼容旧代码）

2. **用户ID获取**：
   - 字符串类型：使用 `RequireUserID(c)`
   - int64类型：使用 `RequireUserIDInt64(c)`（兼容旧代码）

3. **错误监控**：所有错误都会自动记录到 Prometheus，无需手动调用

4. **保持向后兼容**：错误响应格式保持一致，不影响现有客户端
