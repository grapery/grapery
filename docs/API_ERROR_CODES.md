# API 错误码规范

本文档定义了 Grapery 三个服务（server、chatmcp、vippay）的统一错误码规范。

## 统一响应格式

所有服务应遵循以下响应格式：

```json
{
  "code": 1,           // 错误码（见下方定义）
  "message": "success", // 错误消息
  "data": {}           // 响应数据（成功时）
}
```

**注意**：vippay 服务目前使用 `msg` 字段，客户端应同时支持 `message` 和 `msg`。

## 错误码定义

### 成功码
- `1`: 操作成功（server/chatmcp）
- `0`: 操作成功（vippay，历史兼容）

### 业务错误码（负数）

| 错误码 | HTTP状态码 | 说明 | 客户端处理建议 |
|--------|-----------|------|----------------|
| `-1` | 200 | 参数错误 | 显示错误消息，提示用户检查输入 |
| `-2` | 401 | 认证失败 | 清除token，跳转到登录页 |
| `-3` | 403 | 权限不足 | 显示权限不足提示 |
| `-4` | 404 | 资源不存在 | 显示资源不存在提示 |
| `-5` | 500 | 服务器内部错误 | 显示服务器错误，建议稍后重试 |
| `-6` | 200 | 重复记录 | 显示重复记录提示 |
| `-7` | 200 | 超过速率限制 | 显示限流提示，建议稍后重试 |
| `-8` | 401 | Token 过期 | 清除token，跳转到登录页 |
| `-9` | 401 | Token 无效 | 清除token，跳转到登录页 |
| `0` | 200 | 通用失败 | 显示错误消息 |

### HTTP 状态码映射

某些错误码会自动映射到对应的 HTTP 状态码：

- `-2`, `-8`, `-9` → 401 Unauthorized
- `-3` → 403 Forbidden
- `-4` → 404 Not Found
- `-5` → 500 Internal Server Error

其他错误码返回 HTTP 200，通过 `code` 字段区分。

## vippay 服务特殊说明

vippay 服务目前使用以下格式：

```json
{
  "code": 0,        // 0=成功，400/401/500等=错误
  "msg": "success", // 或 "message"
  "data": {}
}
```

**计划统一**：vippay 服务应逐步迁移到统一的错误码格式（使用负数错误码）。

## 客户端错误处理建议

### 1. 网络层错误
- 网络不可用：显示"网络连接失败，请检查网络设置"
- 请求超时：显示"请求超时，请稍后重试"
- DNS解析失败：显示"无法连接到服务器"

### 2. HTTP 状态码错误
- `401`: Token 过期或无效，清除本地token，跳转登录
- `403`: 权限不足，显示权限提示
- `404`: 资源不存在，显示资源不存在提示
- `500/502/503/504`: 服务器错误，显示"服务器暂时不可用，请稍后重试"

### 3. 业务错误码处理

```kotlin
// Android 示例
when (response.code) {
    1 -> // 成功
    -1 -> showError("参数错误：${response.message}")
    -2, -8, -9 -> {
        clearToken()
        navigateToLogin()
    }
    -3 -> showError("权限不足")
    -4 -> showError("资源不存在")
    -5 -> showError("服务器错误，请稍后重试")
    -6 -> showError("记录已存在")
    -7 -> showError("请求过于频繁，请稍后重试")
    0 -> showError(response.message ?: "操作失败")
    else -> showError("未知错误：${response.code}")
}
```

```swift
// iOS 示例
switch response.code {
case 1:
    // 成功
case -1:
    showError("参数错误：\(response.message)")
case -2, -8, -9:
    clearToken()
    navigateToLogin()
case -3:
    showError("权限不足")
case -4:
    showError("资源不存在")
case -5:
    showError("服务器错误，请稍后重试")
case -6:
    showError("记录已存在")
case -7:
    showError("请求过于频繁，请稍后重试")
case 0:
    showError(response.message ?? "操作失败")
default:
    showError("未知错误：\(response.code)")
}
```

## 错误消息国际化

客户端应根据错误码显示本地化的错误消息：

| 错误码 | 中文消息 | 英文消息 |
|--------|---------|---------|
| `-1` | 参数错误 | Invalid parameters |
| `-2` | 认证失败，请重新登录 | Authentication failed, please login again |
| `-3` | 权限不足 | Insufficient permissions |
| `-4` | 资源不存在 | Resource not found |
| `-5` | 服务器错误，请稍后重试 | Server error, please try again later |
| `-6` | 记录已存在 | Record already exists |
| `-7` | 请求过于频繁，请稍后重试 | Too many requests, please try again later |
| `-8` | Token 已过期，请重新登录 | Token expired, please login again |
| `-9` | Token 无效，请重新登录 | Invalid token, please login again |
| `0` | 操作失败 | Operation failed |

## 服务特定错误码

### vippay 服务（支付相关）

vippay 服务在迁移到统一错误码之前，可能返回 HTTP 状态码作为错误码：

- `400`: 请求参数错误
- `401`: 未授权
- `404`: 资源不存在
- `500`: 服务器错误

客户端应同时处理这两种格式。

