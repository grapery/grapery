# VIP Pay 服务接口文档

## 目录
- [1. 基础服务接口](#1-基础服务接口)
- [2. IAP 内购接口](#2-iap-内购接口)
- [3. Apple OAuth2 接口](#3-apple-oauth2-接口)
- [4. Google OAuth2 接口](#4-google-oauth2-接口)
- [5. VIP 会员接口](#5-vip-会员接口)
- [6. 通用错误码](#6-通用错误码)

---

## 1. 基础服务接口

### 1.1 健康检查

**接口描述**: 检查服务健康状态

**请求路径**: `/api/vippay/health`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**: 无

**返回数据结构**:
```json
{
  "status": "healthy",
  "timestamp": "2025-10-21T12:00:00Z",
  "service": "vip-payment",
  "version": "v1.0.0"
}
```

---

### 1.2 版权信息

**接口描述**: 获取应用版权信息（用于iOS app审核）

**请求路径**: `/api/vippay/copyright`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "company": "Grapery Technology",
    "copyright": "© 2025 Grapery Technology. All rights reserved.",
    "app_name": "Grapery VIP Service",
    "version": "v1.0.0",
    "service_terms": "https://www.rankquantity.xyz/api/vippay/terms-of-service",
    "privacy_policy": "https://www.rankquantity.xyz/api/vippay/privacy-policy",
    "contact_email": "support@grapery.xyz",
    "contact_phone": "+86-18589045535",
    "address": "上海市浦东新区临港新片区环湖西二路888号C楼",
    "business_license": "沪ICP备2025137210号",
    "description": "用AI描述你想象中的故事，创造你的故事世界",
    "last_updated": "2025-10-21"
  }
}
```

---

### 1.3 服务条款

**接口描述**: 获取服务条款HTML页面

**请求路径**: `/api/vippay/terms-of-service`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**: 无

**返回数据结构**: 返回 HTML 页面 (Content-Type: text/html; charset=utf-8)

---

### 1.4 隐私政策

**接口描述**: 获取隐私政策HTML页面

**请求路径**: `/api/vippay/privacy-policy`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**: 无

**返回数据结构**: 返回 HTML 页面 (Content-Type: text/html; charset=utf-8)

---

## 2. IAP 内购接口

### 2.1 获取产品列表

**接口描述**: 获取可用的IAP产品列表

**请求路径**: `/api/vippay/iap/products`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| platform | string | 是 | 平台类型: `apple` 或 `google` |
| type | string | 否 | 产品类型: `subscription`(订阅)、`onetime`(一次性)、`consumable`(消耗品) |
| featured | boolean | 否 | 是否只获取推荐产品 |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "products": [
      {
        "id": 1,
        "product_id": "com.grapery.vip.monthly",
        "platform": "apple",
        "product_type": "subscription",
        "name": "月度会员",
        "description": "享受更多创作配额和高级功能",
        "price": 18.00,
        "currency": "CNY",
        "is_subscription": true,
        "is_available": true,
        "featured": true,
        "max_roles": 10,
        "max_contexts": 50,
        "quota_limit": 1000,
        "display_order": 1,
        "duration": "P1M",
        "trial_period": "P7D",
        "intro_offer": null
      }
    ],
    "total": 1
  }
}
```

---

### 2.2 获取产品详情

**接口描述**: 获取指定产品的详细信息

**请求路径**: `/api/vippay/iap/products/:product_id`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| product_id | string | 是 | 产品ID（路径参数） |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "product_id": "com.grapery.vip.monthly",
    "platform": "apple",
    "product_type": "subscription",
    "name": "月度会员",
    "description": "享受更多创作配额和高级功能",
    "price": 18.00,
    "currency": "CNY",
    "status": "active",
    "is_active": true,
    "is_subscription": true,
    "is_available": true,
    "featured": true,
    "family_shareable": false,
    "max_roles": 10,
    "max_contexts": 50,
    "quota_limit": 1000,
    "display_order": 1,
    "sync_status": "synced",
    "last_sync_time": "2025-10-21T12:00:00Z",
    "duration": "P1M",
    "trial_period": "P7D",
    "intro_offer": null,
    "subscription_group": "vip_group",
    "duration_days": 30
  }
}
```

**异常返回码**:
- 400: product_id is required
- 404: product not found

---

### 2.3 获取产品统计

**接口描述**: 获取产品统计信息

**请求路径**: `/api/vippay/iap/products/stats`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| platform | string | 否 | 平台类型（默认为 `apple`） |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "platform": "apple",
    "stats": {
      "total_products": 10,
      "active_products": 8,
      "subscription_products": 5,
      "onetime_products": 3,
      "consumable_products": 2
    }
  }
}
```

---

### 2.4 验证 Apple 收据

**接口描述**: 验证 Apple 内购收据

**请求路径**: `/api/vippay/iap/apple/verify`

**请求方法**: `POST`

**鉴权方式**: JWT Bearer Token

**请求参数**:
```json
{
  "receipt_data": "base64_encoded_receipt_data",
  "sandbox": false
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| receipt_data | string | 是 | Base64 编码的收据数据 |
| sandbox | boolean | 否 | 是否为沙盒环境（默认 false） |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "success": true,
    "receipt": {
      "id": "receipt_12345",
      "user_id": 100,
      "bundle_id": "com.grapery.app",
      "product_id": "com.grapery.vip.monthly",
      "transaction_id": "txn_12345",
      "purchase_date": "2025-10-21T12:00:00Z",
      "expires_date": "2025-11-21T12:00:00Z",
      "is_trial_period": false,
      "is_in_intro_offer_period": false
    }
  }
}
```

**异常返回码**:
- 400: invalid request parameters
- 401: unauthorized
- 500: failed to verify receipt

---

### 2.5 验证 Google 购买

**接口描述**: 验证 Google Play 购买

**请求路径**: `/api/vippay/iap/google/verify`

**请求方法**: `POST`

**鉴权方式**: JWT Bearer Token

**请求参数**:
```json
{
  "purchase_token": "google_purchase_token",
  "product_id": "com.grapery.vip.monthly"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| purchase_token | string | 是 | Google 购买令牌 |
| product_id | string | 是 | 产品ID |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "success": true,
    "purchase": {
      "id": "purchase_12345",
      "user_id": 100,
      "product_id": "com.grapery.vip.monthly",
      "purchase_token": "google_purchase_token",
      "purchase_date": "2025-10-21T12:00:00Z",
      "expires_date": "2025-11-21T12:00:00Z"
    }
  }
}
```

**异常返回码**:
- 400: invalid request parameters
- 401: unauthorized
- 500: failed to verify purchase

---

### 2.6 获取 Apple 订阅状态

**接口描述**: 获取 Apple 订阅状态

**请求路径**: `/api/vippay/iap/apple/subscription-status`

**请求方法**: `POST`

**鉴权方式**: JWT Bearer Token

**请求参数**:
```json
{
  "original_transaction_id": "original_txn_12345"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| original_transaction_id | string | 是 | 原始交易ID |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "success": true,
    "subscription": {
      "id": "sub_12345",
      "user_id": 100,
      "product_id": "com.grapery.vip.monthly",
      "original_transaction_id": "original_txn_12345",
      "status": "active",
      "start_date": "2025-10-21T12:00:00Z",
      "expires_date": "2025-11-21T12:00:00Z",
      "auto_renew": true,
      "is_in_billing_retry_period": false
    }
  }
}
```

**异常返回码**:
- 400: invalid request parameters
- 401: unauthorized
- 500: failed to get subscription status

---

### 2.7 获取 Google 订阅状态

**接口描述**: 获取 Google Play 订阅状态

**请求路径**: `/api/vippay/iap/google/subscription-status`

**请求方法**: `POST`

**鉴权方式**: JWT Bearer Token

**请求参数**:
```json
{
  "purchase_token": "google_purchase_token",
  "product_id": "com.grapery.vip.monthly"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| purchase_token | string | 是 | Google 购买令牌 |
| product_id | string | 是 | 产品ID |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "success": true,
    "subscription": {
      "id": "sub_12345",
      "user_id": 100,
      "product_id": "com.grapery.vip.monthly",
      "purchase_token": "google_purchase_token",
      "status": "active",
      "start_date": "2025-10-21T12:00:00Z",
      "expires_date": "2025-11-21T12:00:00Z",
      "auto_renew": true
    }
  }
}
```

**异常返回码**:
- 400: invalid request parameters
- 401: unauthorized
- 500: failed to get subscription status

---

### 2.8 确认购买（通用）

**接口描述**: 确认内购购买（支持 Apple 和 Google）

**请求路径**: `/api/vippay/iap/acknowledge`

**请求方法**: `POST`

**鉴权方式**: JWT Bearer Token

**请求参数**:
```json
{
  "platform": "apple",
  "purchase_token": "purchase_token_or_transaction_id"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| platform | string | 是 | 平台: `apple` 或 `google` |
| purchase_token | string | 是 | 购买令牌或交易ID |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "acknowledged": true
  }
}
```

**异常返回码**:
- 400: invalid request parameters / invalid platform
- 401: unauthorized
- 500: failed to acknowledge purchase

---

### 2.9 消费购买（通用）

**接口描述**: 消费内购商品（支持 Apple 和 Google）

**请求路径**: `/api/vippay/iap/consume`

**请求方法**: `POST`

**鉴权方式**: JWT Bearer Token

**请求参数**:
```json
{
  "platform": "apple",
  "purchase_token": "purchase_token_or_transaction_id"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| platform | string | 是 | 平台: `apple` 或 `google` |
| purchase_token | string | 是 | 购买令牌或交易ID |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "consumed": true
  }
}
```

**异常返回码**:
- 400: invalid request parameters / invalid platform
- 401: unauthorized
- 500: failed to consume purchase

---

### 2.10 同步订阅

**接口描述**: 同步用户的所有订阅状态

**请求路径**: `/api/vippay/iap/sync`

**请求方法**: `POST`

**鉴权方式**: JWT Bearer Token

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "synced": true
  }
}
```

**异常返回码**:
- 401: unauthorized

---

### 2.11 处理 Apple 通知（服务器回调）

**接口描述**: 接收 Apple App Store 的服务器通知

**请求路径**: `/api/vippay/iap/apple/notification`

**请求方法**: `POST`

**鉴权方式**: 无需鉴权（服务器间通信）

**请求参数**:
```json
{
  "signed_payload": "apple_signed_payload_jwt"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| signed_payload | string | 是 | Apple 签名的 JWT payload |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "notification_id": "notification_12345",
    "status": "Success"
  }
}
```

**异常返回码**:
- 400: invalid request parameters / failed to parse notification

---

### 2.12 处理 Google 通知（服务器回调）

**接口描述**: 接收 Google Play 的服务器通知

**请求路径**: `/api/vippay/iap/google/notification`

**请求方法**: `POST`

**鉴权方式**: 无需鉴权（服务器间通信）

**请求参数**:
```json
{
  "data": "google_notification_data"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| data | string | 是 | Google 通知数据 |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "notification_id": "notification_12345",
    "status": "Success"
  }
}
```

**异常返回码**:
- 400: invalid request parameters / failed to parse notification

---

## 3. Apple OAuth2 接口

### 3.1 Apple 登录

**接口描述**: 使用 Apple Sign-In 进行用户登录

**请求路径**: `/api/vippay/apple-oauth/signin`

**请求方法**: `POST`

**鉴权方式**: 无需鉴权

**请求参数**:
```json
{
  "identity_token": "apple_identity_token",
  "full_name": "张三"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| identity_token | string | 是 | Apple Identity Token |
| full_name | string | 否 | 用户全名（仅首次登录时有） |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "success": true,
  "data": {
    "system_user_id": 12345,
    "user_id": "apple_user_id",
    "email": "user@privaterelay.appleid.com",
    "is_new_user": false,
    "access_token": "jwt_access_token",
    "expires_at": 1729598400
  }
}
```

**异常返回码**:
- 400: Invalid request body / Identity token is required
- 401: Invalid identity token
- 500: Apple OAuth2 service is not available / Failed to process login / Failed to generate access token

---

### 3.2 获取 Apple 登录状态

**接口描述**: 查询用户的 Apple 登录状态

**请求路径**: `/api/vippay/apple-oauth/status`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| user_id | string | 是 | 用户ID（查询参数） |

**返回数据结构**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": "apple_user_id",
    "is_active": true,
    "last_login": "2025-10-21T12:00:00Z"
  }
}
```

**异常返回码**:
- 400: user_id is required

---

### 3.3 获取 Apple OAuth 配置

**接口描述**: 获取当前 Apple OAuth2 配置信息（用于调试）

**请求路径**: `/api/vippay/apple-oauth/config`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "bundle_id": "com.rankquantity.voyager",
    "is_valid": true
  }
}
```

**异常返回码**:
- 500: Apple OAuth2 verifier is not properly configured

---

## 4. Google OAuth2 接口

### 4.1 Google 登录

**接口描述**: 使用 Google Sign-In 进行用户登录

**请求路径**: `/api/vippay/google-oauth/signin`

**请求方法**: `POST`

**鉴权方式**: 无需鉴权

**请求参数**:
```json
{
  "id_token": "google_id_token"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id_token | string | 是 | Google ID Token |

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "success": true,
  "data": {
    "system_user_id": 12345,
    "user_id": "google_user_id",
    "email": "user@gmail.com",
    "name": "张三",
    "picture": "https://lh3.googleusercontent.com/a/...",
    "is_new_user": false,
    "access_token": "jwt_access_token",
    "expires_at": 1729598400
  }
}
```

**异常返回码**:
- 400: Invalid request body / ID token is required
- 401: Invalid ID token
- 500: Google OAuth2 service is not available / Failed to process login / Failed to generate access token

---

### 4.2 获取 Google 登录状态

**接口描述**: 查询用户的 Google 登录状态

**请求路径**: `/api/vippay/google-oauth/status`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| user_id | string | 是 | 用户ID（查询参数） |

**返回数据结构**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": "google_user_id",
    "is_active": true,
    "last_login": "2025-10-21T12:00:00Z"
  }
}
```

**异常返回码**:
- 400: user_id is required

---

### 4.3 获取 Google OAuth 配置

**接口描述**: 获取当前 Google OAuth2 配置信息（用于调试）

**请求路径**: `/api/vippay/google-oauth/config`

**请求方法**: `GET`

**鉴权方式**: 无需鉴权

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "client_id": "345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb.apps.googleusercontent.com",
    "is_valid": true
  }
}
```

**异常返回码**:
- 500: Google OAuth2 verifier is not properly configured

---

## 5. VIP 会员接口

所有 VIP 会员接口均需要 JWT Bearer Token 鉴权。

### 5.1 获取 VIP 信息

**接口描述**: 获取当前用户的 VIP 会员信息

**请求路径**: `/api/vippay/vip/info`

**请求方法**: `GET`

**鉴权方式**: JWT Bearer Token

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": 12345,
    "is_vip": false,
    "level": 0,
    "status": 0,
    "auto_renew": false,
    "quota_used": 0,
    "quota_limit": 0,
    "max_roles": 2,
    "max_contexts": 5
  }
}
```

**异常返回码**:
- 401: unauthorized

---

### 5.2 检查 VIP 状态

**接口描述**: 检查当前用户是否为 VIP 会员

**请求路径**: `/api/vippay/vip/check`

**请求方法**: `GET`

**鉴权方式**: JWT Bearer Token

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": 12345,
    "is_vip": false
  }
}
```

**异常返回码**:
- 401: unauthorized

---

### 5.3 获取配额信息

**接口描述**: 获取当前用户的配额使用情况

**请求路径**: `/api/vippay/vip/quota`

**请求方法**: `GET`

**鉴权方式**: JWT Bearer Token

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": 12345,
    "quota_used": 0,
    "quota_limit": 0,
    "remaining": 0
  }
}
```

**异常返回码**:
- 401: unauthorized

---

### 5.4 获取最大角色数

**接口描述**: 获取当前用户可创建的最大角色数

**请求路径**: `/api/vippay/vip/max-roles`

**请求方法**: `GET`

**鉴权方式**: JWT Bearer Token

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": 12345,
    "max_roles": 2
  }
}
```

**异常返回码**:
- 401: unauthorized

---

### 5.5 获取最大上下文数

**接口描述**: 获取当前用户可使用的最大上下文数

**请求路径**: `/api/vippay/vip/max-contexts`

**请求方法**: `GET`

**鉴权方式**: JWT Bearer Token

**请求参数**: 无

**返回数据结构**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": 12345,
    "max_contexts": 5
  }
}
```

**异常返回码**:
- 401: unauthorized

---

## 6. 通用错误码

### 6.1 HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 未授权/Token无效 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

### 6.2 业务错误码

| code | msg | 说明 |
|------|-----|------|
| 0 | success | 请求成功 |
| 400 | invalid request parameters | 请求参数无效 |
| 401 | unauthorized | 未授权 |
| 401 | missing or invalid token | Token缺失或无效 |
| 401 | invalid token | Token无效 |
| 401 | user not found | 用户不存在 |
| 401 | Invalid identity token | Apple Identity Token 无效 |
| 401 | Invalid ID token | Google ID Token 无效 |
| 404 | product not found | 产品不存在 |
| 500 | failed to verify receipt | 收据验证失败 |
| 500 | failed to verify purchase | 购买验证失败 |
| 500 | failed to get subscription status | 获取订阅状态失败 |
| 500 | failed to acknowledge purchase | 确认购买失败 |
| 500 | failed to consume purchase | 消费购买失败 |
| 500 | failed to get products | 获取产品列表失败 |
| 500 | failed to get product stats | 获取产品统计失败 |
| 500 | Apple OAuth2 service is not available | Apple OAuth2 服务不可用 |
| 500 | Google OAuth2 service is not available | Google OAuth2 服务不可用 |
| 500 | Failed to process login | 登录处理失败 |
| 500 | Failed to generate access token | 生成访问令牌失败 |

---

## 7. 鉴权说明

### 7.1 JWT Bearer Token

需要鉴权的接口需要在 HTTP Header 中添加 Authorization 字段：

```
Authorization: Bearer <your_jwt_token>
```

### 7.2 获取 Token

通过以下任一方式获取 JWT Token：
1. Apple OAuth2 登录 (`/api/vippay/apple-oauth/signin`)
2. Google OAuth2 登录 (`/api/vippay/google-oauth/signin`)

### 7.3 Token 有效期

- Token 有效期：24小时
- Token 过期后需重新登录获取新的 Token

---

## 8. 环境说明

### 8.1 基础 URL

- 生产环境: `https://www.rankquantity.xyz`
- 完整接口地址示例: `https://www.rankquantity.xyz/api/vippay/health`

### 8.2 IAP 环境

- Apple 沙盒环境：用于开发测试
- Apple 生产环境：用于正式发布
- Google 测试环境：通过 Google Play Console 配置
- Google 生产环境：用于正式发布

---

## 9. 注意事项

1. **跨域支持**: 所有接口支持 CORS 跨域请求
2. **超时时间**: 
   - 读取超时: 120秒
   - 写入超时: 120秒
   - 空闲超时: 120秒
3. **内容类型**: 
   - 请求: `application/json`（除HTML接口外）
   - 响应: `application/json`（除HTML接口外）
4. **字符编码**: UTF-8
5. **时间格式**: ISO 8601 (RFC3339)，如 `2025-10-21T12:00:00Z`
6. **Apple 通知和 Google 通知接口**仅用于服务器间通信，不应被客户端调用

---

## 10. 联系方式

- 技术支持邮箱: support@grapery.xyz
- 运营邮箱: putaoshuyunying@grapery.xyz
- 联系电话: +86-18589045535
- 公司地址: 上海市浦东新区临港新片区环湖西二路888号C楼

---

**文档版本**: v1.0.0  
**最后更新**: 2025-10-21  
**维护团队**: Grapery Technology

