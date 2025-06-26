# 支付系统使用说明

## 系统概述

本支付系统是一个完整的应用内支付解决方案，支持多种支付方式，包括订阅支付、商品SKU管理、VIP用户管理等核心功能。

## 核心功能

### 1. 支付方式支持
- **Apple Pay**: 苹果支付
- **Google Pay**: 谷歌支付  
- **微信支付**: 微信支付
- **支付宝**: 支付宝支付
- **Stripe**: 国际支付

### 2. 商品管理
- 支持订阅类型商品
- 支持一次性购买商品
- 支持消耗品（如积分、额度）
- 商品状态管理（上架/下架/删除）

### 3. 订阅管理
- 自动续费管理
- 订阅状态跟踪
- 订阅升级/降级
- 订阅取消

### 4. VIP用户管理
- VIP等级管理（基础VIP、专业VIP）
- 权限控制（AI使用、角色数量、上下文数量等）
- 额度管理和使用统计
- 自动续费管理

## 数据库表结构

### 1. products 商品表
```sql
CREATE TABLE products (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    product_type INT NOT NULL, -- 1:订阅 2:一次性 3:消耗品
    status INT DEFAULT 1, -- 1:上架 2:下架 3:删除
    price BIGINT NOT NULL, -- 价格（分）
    currency VARCHAR(10) DEFAULT 'CNY',
    duration BIGINT DEFAULT 0, -- 有效期（秒）
    max_roles INT DEFAULT 2,
    max_contexts INT DEFAULT 5,
    quota_limit INT DEFAULT 1000,
    available_models TEXT, -- JSON数组
    features TEXT, -- JSON对象
    sort_order INT DEFAULT 0,
    created_by BIGINT NOT NULL,
    updated_by BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### 2. subscriptions 订阅表
```sql
CREATE TABLE subscriptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    status INT DEFAULT 1, -- 1:活跃 2:过期 3:取消 4:待支付 5:失败
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    auto_renew BOOLEAN DEFAULT TRUE,
    payment_method VARCHAR(50),
    payment_provider VARCHAR(50),
    provider_sub_id VARCHAR(255),
    amount BIGINT NOT NULL,
    currency VARCHAR(10) DEFAULT 'CNY',
    quota_limit INT DEFAULT 1000,
    quota_used INT DEFAULT 0,
    max_roles INT DEFAULT 2,
    max_contexts INT DEFAULT 5,
    available_models TEXT,
    cancel_reason VARCHAR(255),
    canceled_at TIMESTAMP NULL,
    canceled_by BIGINT,
    next_billing_date TIMESTAMP NULL,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### 3. payment_records 支付记录表
```sql
CREATE TABLE payment_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    subscription_id BIGINT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) DEFAULT 'CNY',
    status INT DEFAULT 1, -- 1:待支付 2:成功 3:失败 4:取消 5:退款 6:部分退款
    payment_method INT NOT NULL, -- 1:Apple 2:Google 3:微信 4:支付宝 5:Stripe
    payment_provider VARCHAR(50) NOT NULL,
    provider_order_id VARCHAR(255),
    provider_payment_id VARCHAR(255),
    transaction_id VARCHAR(255) UNIQUE,
    payment_time TIMESTAMP NULL,
    refund_amount BIGINT DEFAULT 0,
    refund_time TIMESTAMP NULL,
    refund_reason VARCHAR(255),
    error_code VARCHAR(100),
    error_message VARCHAR(500),
    callback_data TEXT,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

## API接口

### 1. 创建订单
```http
POST /api/payment/orders
Content-Type: application/json
Authorization: Bearer <token>

{
    "product_id": 1,
    "payment_method": 5
}
```

响应：
```json
{
    "code": 0,
    "msg": "success",
    "data": {
        "order_id": 123,
        "order_no": "ORDER1234567890",
        "amount": 999,
        "status": 1,
        "payment_url": "https://checkout.stripe.com/pay/...",
        "qr_code_url": "https://...",
        "transaction_id": "pi_1234567890"
    }
}
```

### 2. 查询支付状态
```http
POST /api/payment/query
Content-Type: application/json
Authorization: Bearer <token>

{
    "order_id": 123
}
```

响应：
```json
{
    "code": 0,
    "msg": "success",
    "data": {
        "order_id": 123,
        "status": 2,
        "amount": 999,
        "payment_time": "2024-01-01 12:00:00",
        "provider_order_id": "pi_1234567890"
    }
}
```

### 3. 获取用户订单列表
```http
GET /api/payment/orders?offset=0&limit=20
Authorization: Bearer <token>
```

响应：
```json
{
    "code": 0,
    "msg": "success",
    "data": {
        "orders": [
            {
                "id": 123,
                "order_no": "ORDER1234567890",
                "amount": 999,
                "status": 2,
                "description": "基础VIP月付",
                "created_at": "2024-01-01 12:00:00"
            }
        ],
        "total": 1
    }
}
```

### 4. 获取用户VIP信息
```http
GET /api/payment/vip/info
Authorization: Bearer <token>
```

响应：
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

### 5. 取消订阅
```http
POST /api/payment/subscription/cancel
Content-Type: application/json
Authorization: Bearer <token>

{
    "subscription_id": 456,
    "reason": "用户主动取消"
}
```

### 6. 支付回调处理
```http
POST /api/payment/callback
Content-Type: application/json

{
    "provider": "stripe",
    "data": {
        "id": "evt_1234567890",
        "type": "payment_intent.succeeded",
        "data": {
            "object": {
                "id": "pi_1234567890",
                "amount": 999,
                "status": "succeeded"
            }
        }
    }
}
```

## 使用示例

### 1. 初始化支付服务
```go
package main

import (
    "github.com/grapery/grapery/pkg/pay"
    "github.com/grapery/grapery/service/pay"
)

func main() {
    // 加载配置
    config := &pay.PaymentConfig{
        StripeConfig: pay.PaymentConfig.StripeConfig{
            SecretKey: "sk_test_...",
            PublishableKey: "pk_test_...",
            WebhookSecret: "whsec_...",
        },
        DefaultCurrency: "CNY",
        ReturnURL: "https://yourdomain.com/payment/return",
        NotifyURL: "https://yourdomain.com/payment/notify",
    }

    // 创建支付服务
    paymentService, err := pay.NewPaymentService(config)
    if err != nil {
        panic(err)
    }

    // 创建HTTP处理器
    handler := pay.NewPaymentHandler(paymentService)

    // 设置路由
    http.HandleFunc("/api/payment/orders", handler.CreateOrder)
    http.HandleFunc("/api/payment/query", handler.QueryPayment)
    http.HandleFunc("/api/payment/orders", handler.GetUserOrders)
    http.HandleFunc("/api/payment/vip/info", handler.GetUserVIPInfo)
    http.HandleFunc("/api/payment/subscription/cancel", handler.CancelSubscription)
    http.HandleFunc("/api/payment/callback", handler.HandlePaymentCallback)

    http.ListenAndServe(":8080", nil)
}
```

### 2. 权限检查示例
```go
// 检查用户是否可以使用AI
func checkAIPermission(ctx context.Context, userID int64) error {
    canUse, err := paymentService.CheckUserPermission(ctx, userID, "use_ai")
    if err != nil {
        return err
    }
    if !canUse {
        return errors.New("需要VIP才能使用AI功能")
    }
    return nil
}

// 消费用户额度
func consumeUserQuota(ctx context.Context, userID int64, amount int) error {
    return paymentService.ConsumeUserQuota(ctx, userID, amount)
}

// 获取用户额度信息
func getUserQuota(ctx context.Context, userID int64) (used, limit int, err error) {
    return paymentService.GetUserQuota(ctx, userID)
}
```

## 配置说明

### 1. Stripe配置
- `secret_key`: Stripe私钥
- `publishable_key`: Stripe公钥
- `webhook_secret`: Webhook签名密钥

### 2. 支付宝配置
- `app_id`: 支付宝应用ID
- `private_key`: 支付宝私钥
- `public_key`: 支付宝公钥

### 3. 微信支付配置
- `app_id`: 微信应用ID
- `mch_id`: 商户号
- `api_key`: API密钥
- `cert_path`: 证书路径
- `key_path`: 私钥路径

### 4. Apple Pay配置
- `bundle_id`: 应用Bundle ID
- `key_id`: 密钥ID
- `key_path`: 密钥文件路径

### 5. Google Pay配置
- `merchant_id`: 商户ID
- `key_path`: 密钥文件路径

## 注意事项

1. **安全性**: 所有敏感配置信息应通过环境变量或安全的配置管理系统管理
2. **错误处理**: 支付过程中可能出现各种错误，需要完善的错误处理机制
3. **日志记录**: 所有支付操作都应记录详细日志，便于问题排查
4. **数据一致性**: 支付状态更新需要保证数据一致性，建议使用事务
5. **回调验证**: 支付回调需要验证签名，确保数据安全
6. **测试环境**: 建议先在测试环境验证所有支付流程
7. **监控告警**: 设置支付失败率监控和告警机制

## 扩展功能

1. **优惠券系统**: 支持优惠券和折扣码
2. **积分系统**: 支持积分支付和积分奖励
3. **发票系统**: 支持电子发票生成
4. **对账系统**: 支持支付对账和财务报表
5. **风控系统**: 支持支付风险控制
6. **多语言支持**: 支持多语言界面和错误信息 