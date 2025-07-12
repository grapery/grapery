# VIP 支付服务 API 文档

## 概述

VIP 支付服务提供完整的会员订阅、支付管理、权限控制等功能。所有接口都需要用户认证，请在请求头中包含 `Authorization: Bearer <token>`。

## 基础信息

- **基础URL**: `http://localhost:8082`
- **API版本**: `v1`
- **认证方式**: Bearer Token
- **响应格式**: JSON

## 通用响应格式

```json
{
  "code": 0,           // 状态码，0表示成功
  "msg": "success",    // 消息
  "data": {}           // 数据
}
```

## 错误码说明

| 状态码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 1. 健康检查

### 检查服务状态
```http
GET /health
```

**响应示例：**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "service": "vip-payment",
  "version": "1.0.0"
}
```

---

## 2. 商品管理

### 获取商品列表
```http
GET /api/v1/products?type=1&category=vip&limit=20
```

**查询参数：**
- `type` (可选): 商品类型 (1: 订阅, 2: 一次性)
- `category` (可选): 商品分类
- `limit` (可选): 返回数量，默认20

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "products": [
      {
        "id": 1,
        "name": "基础VIP月付",
        "description": "基础会员套餐，月付",
        "price": 9900,
        "currency": "CNY",
        "type": 1,
        "status": 1,
        "sku": "BASIC_MONTHLY",
        "duration": 2592000,
        "quota_limit": 1000,
        "max_roles": 5,
        "max_contexts": 100
      }
    ],
    "total": 1
  }
}
```

### 获取单个商品
```http
GET /api/v1/products/{id}
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "name": "基础VIP月付",
    "description": "基础会员套餐，月付",
    "price": 9900,
    "currency": "CNY",
    "type": 1,
    "status": 1,
    "sku": "BASIC_MONTHLY",
    "duration": 2592000,
    "quota_limit": 1000,
    "max_roles": 5,
    "max_contexts": 100,
    "available_models": ["gpt-3.5-turbo", "gpt-4"]
  }
}
```

---

## 3. 订单管理

### 创建订单
```http
POST /api/v1/payment/orders
Content-Type: application/json

{
  "product_id": 1,
  "payment_method": 5
}
```

**请求参数：**
- `product_id`: 商品ID
- `payment_method`: 支付方式 (1: Apple Pay, 2: Google Pay, 3: 微信支付, 4: 支付宝, 5: Stripe)

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "order_id": 123,
    "order_no": "ORDER1234567890",
    "amount": 9900,
    "status": 1,
    "payment_url": "https://checkout.stripe.com/pay/...",
    "qr_code_url": "https://...",
    "transaction_id": "pi_1234567890"
  }
}
```

### 获取用户订单
```http
GET /api/v1/payment/orders?offset=0&limit=20
```

**查询参数：**
- `offset` (可选): 偏移量，默认0
- `limit` (可选): 返回数量，默认20

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "orders": [
      {
        "id": 123,
        "order_no": "ORDER1234567890",
        "amount": 9900,
        "status": 2,
        "description": "基础VIP月付",
        "created_at": "2024-01-01 12:00:00"
      }
    ],
    "total": 1
  }
}
```

### 查询支付状态
```http
POST /api/v1/payment/query
Content-Type: application/json

{
  "order_id": 123
}
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "order_id": 123,
    "status": 2,
    "amount": 9900,
    "payment_time": "2024-01-01 12:00:00",
    "provider_order_id": "pi_1234567890"
  }
}
```

---

## 4. 支付管理

### 退款
```http
POST /api/v1/payment/refund
Content-Type: application/json

{
  "order_id": 123,
  "refund_amount": 9900,
  "refund_reason": "用户申请退款"
}
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "refund processed successfully"
}
```

### 获取支付记录
```http
GET /api/v1/payment/records?offset=0&limit=20&status=2&method=5
```

**查询参数：**
- `offset` (可选): 偏移量，默认0
- `limit` (可选): 返回数量，默认20
- `status` (可选): 支付状态 (1: 待支付, 2: 成功, 3: 失败, 4: 已取消, 5: 已退款)
- `method` (可选): 支付方式

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "records": [
      {
        "id": 456,
        "order_id": 123,
        "amount": 9900,
        "currency": "CNY",
        "status": 2,
        "payment_method": 5,
        "payment_provider": "stripe",
        "transaction_id": "pi_1234567890",
        "payment_time": "2024-01-01 12:00:00",
        "refund_amount": 0,
        "refund_reason": "",
        "created_at": "2024-01-01 12:00:00"
      }
    ],
    "total": 1
  }
}
```

---

## 5. VIP 会员管理

### 获取VIP信息
```http
GET /api/v1/vip/info
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "is_vip": true,
    "level": 1,
    "status": 1,
    "expire_time": "2024-02-01 12:00:00",
    "auto_renew": true,
    "quota_used": 100,
    "quota_limit": 1000,
    "max_roles": 5,
    "max_contexts": 100
  }
}
```

### 检查是否为VIP
```http
GET /api/v1/vip/check
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "is_vip": true
  }
}
```

### 获取用户额度
```http
GET /api/v1/vip/quota
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "quota_used": 100,
    "quota_limit": 1000,
    "remaining": 900
  }
}
```

### 消费用户额度
```http
POST /api/v1/vip/quota/consume
Content-Type: application/json

{
  "amount": 10
}
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "quota consumed successfully"
}
```

### 获取最大角色数
```http
GET /api/v1/vip/max-roles
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "max_roles": 5
  }
}
```

### 获取最大上下文数
```http
GET /api/v1/vip/max-contexts
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "max_contexts": 100
  }
}
```

### 获取可用模型
```http
GET /api/v1/vip/models
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "available_models": ["gpt-3.5-turbo", "gpt-4"]
  }
}
```

### 检查用户权限
```http
GET /api/v1/vip/permission?permission=ai_generation
```

**权限类型：**
- `ai_generation`: AI生成权限
- `advanced_models`: 高级模型权限
- `unlimited_contexts`: 无限上下文权限
- `multiple_roles`: 多角色权限

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "permission": "ai_generation",
    "has_permission": true
  }
}
```

---

## 6. 订阅管理

### 获取用户订阅
```http
GET /api/v1/subscription?offset=0&limit=20
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "subscriptions": [
      {
        "id": 789,
        "product_id": 1,
        "order_id": 123,
        "status": 1,
        "start_time": "2024-01-01 12:00:00",
        "end_time": "2024-02-01 12:00:00",
        "auto_renew": true,
        "amount": 9900,
        "currency": "CNY",
        "quota_used": 100,
        "quota_limit": 1000,
        "max_roles": 5,
        "max_contexts": 100
      }
    ],
    "total": 1
  }
}
```

### 取消订阅
```http
POST /api/v1/subscription/cancel
Content-Type: application/json

{
  "subscription_id": 789,
  "reason": "用户主动取消"
}
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "subscription canceled successfully"
}
```

### 更新订阅额度
```http
PUT /api/v1/subscription/quota
Content-Type: application/json

{
  "subscription_id": 789,
  "quota_used": 150
}
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "subscription quota updated successfully"
}
```

---

## 7. 统计信息

### 获取支付统计
```http
GET /api/v1/stats/payment
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total_payments": 10,
    "successful_payments": 8,
    "failed_payments": 2,
    "total_amount": 99000,
    "total_refunded": 0,
    "success_rate": 80.0
  }
}
```

### 获取订单统计
```http
GET /api/v1/stats/order
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total_orders": 10,
    "pending_orders": 1,
    "paid_orders": 8,
    "canceled_orders": 1,
    "total_amount": 99000,
    "success_rate": 80.0
  }
}
```

---

## 8. 支付回调

### 处理支付回调
```http
POST /api/v1/payment/callback
Content-Type: application/json

{
  "provider": "stripe",
  "data": {
    "id": "evt_1234567890",
    "type": "payment_intent.succeeded",
    "data": {
      "object": {
        "id": "pi_1234567890",
        "amount": 9900,
        "status": "succeeded"
      }
    }
  }
}
```

**响应示例：**
```json
{
  "code": 0,
  "msg": "callback processed successfully"
}
```

### 第三方支付回调
```http
POST /callback/alipay
POST /callback/wechat
POST /callback/stripe
```

---

## 使用示例

### 1. 创建VIP订阅
```bash
# 1. 获取商品列表
curl -X GET "http://localhost:8082/api/v1/products?type=1" \
  -H "Authorization: Bearer your_token"

# 2. 创建订单
curl -X POST "http://localhost:8082/api/v1/payment/orders" \
  -H "Authorization: Bearer your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "payment_method": 5
  }'

# 3. 查询支付状态
curl -X POST "http://localhost:8082/api/v1/payment/query" \
  -H "Authorization: Bearer your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": 123
  }'
```

### 2. 检查用户权限
```bash
# 检查AI生成权限
curl -X GET "http://localhost:8082/api/v1/vip/permission?permission=ai_generation" \
  -H "Authorization: Bearer your_token"

# 获取用户额度
curl -X GET "http://localhost:8082/api/v1/vip/quota" \
  -H "Authorization: Bearer your_token"
```

### 3. 消费用户额度
```bash
# 消费10个额度
curl -X POST "http://localhost:8082/api/v1/vip/quota/consume" \
  -H "Authorization: Bearer your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 10
  }'
```

---

## 注意事项

1. **认证**: 所有接口都需要在请求头中包含有效的Bearer Token
2. **金额**: 所有金额都以分为单位（如9900表示99.00元）
3. **时间格式**: 所有时间都使用 `2006-01-02 15:04:05` 格式
4. **分页**: 支持分页的接口都使用 `offset` 和 `limit` 参数
5. **状态码**: 支付状态和订单状态使用数字编码，具体含义请参考相关模型定义
6. **错误处理**: 所有错误都会返回相应的HTTP状态码和错误信息 