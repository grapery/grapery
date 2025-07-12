# VIP 支付服务

## 概述

VIP 支付服务是基于 Gin 框架构建的现代化支付系统，支持多种支付方式，包括支付宝、微信支付、Stripe 等。该服务提供了完整的会员订阅管理、订单处理、支付回调等功能。

## 特性

- 🚀 **高性能**: 基于 Gin 框架，支持高并发处理
- 🔒 **安全可靠**: 支持支付签名验证、风险控制
- 💳 **多支付方式**: 支持支付宝、微信支付、Stripe 等主流支付方式
- 📊 **完整监控**: 详细的日志记录和性能监控
- 🔄 **自动续费**: 支持订阅自动续费功能
- 🎯 **权限管理**: 基于会员等级的权限控制系统

## 快速开始

### 1. 启动服务

```bash
# 编译
go build -o vippay app/vippay/main.go

# 启动服务
./vippay -config vippay.json
```

### 2. 健康检查

```bash
curl http://localhost:8082/health
```

响应示例：
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "service": "vip-payment",
  "version": "1.0.0"
}
```

## API 接口

### 订单管理

#### 创建订单
```http
POST /api/v1/payment/orders
Content-Type: application/json
Authorization: Bearer <token>

{
  "product_id": 1,
  "payment_method": 1
}
```

#### 获取用户订单
```http
GET /api/v1/payment/orders?offset=0&limit=20
Authorization: Bearer <token>
```

#### 查询支付状态
```http
POST /api/v1/payment/query
Content-Type: application/json
Authorization: Bearer <token>

{
  "order_id": 123
}
```

### VIP 会员管理

#### 获取用户VIP信息
```http
GET /api/v1/vip/info
Authorization: Bearer <token>
```

#### 取消订阅
```http
POST /api/v1/subscription/cancel
Content-Type: application/json
Authorization: Bearer <token>

{
  "subscription_id": 456,
  "reason": "用户主动取消"
}
```

### 支付回调

#### 处理支付回调
```http
POST /api/v1/payment/callback
Content-Type: application/json
X-Signature: <signature>

{
  "provider": "alipay",
  "data": {
    "out_trade_no": "ORDER123456789",
    "trade_status": "TRADE_SUCCESS",
    "total_amount": "99.00"
  }
}
```

## 配置说明

### 基础配置 (vippay.json)

```json
{
  "sql_db": {
    "database": "grapery",
    "username": "root",
    "password": "password",
    "address": "localhost"
  },
  "redis": {
    "address": "localhost:6379",
    "password": "",
    "database": "0",
    "ping_interval": 30
  },
  "log_level": "debug",
  "rpc_port": "8081",
  "http_port": "8080",
  "vippay": {
    "http_port": "8082"
  }
}
```

### 支付配置

支付配置在代码中通过 `PaymentConfig` 结构体定义：

```go
paymentConfig := &paypkg.PaymentConfig{
    DefaultCurrency:   "CNY",
    ReturnURL:         "https://your-domain.com/payment/return",
    NotifyURL:         "https://your-domain.com/api/v1/payment/callback",
    OrderExpireTime:   30, // 30分钟
    PaymentExpireTime: 15, // 15分钟
    MaxRetryCount:     3,
    EnableTestMode:    true,
    EnableRiskCheck:   true,
    RiskThreshold:     0.8,
}
```

## 中间件

### 已集成的中间件

1. **恢复中间件 (Recovery)**: 自动处理 panic，确保服务稳定性
2. **日志中间件 (Logger)**: 记录所有请求的详细信息
3. **CORS 中间件**: 支持跨域请求
4. **认证中间件**: 基于 JWT 的用户认证

### 自定义日志格式

```go
gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
    return fmt.Sprintf("[VIP-PAY] %v | %3d | %13v | %15s | %-7s %s\n%s",
        param.TimeStamp.Format("2006/01/02 - 15:04:05"),
        param.StatusCode,
        param.Latency,
        param.ClientIP,
        param.Method,
        param.Path,
        param.ErrorMessage,
    )
})
```

## 支付提供商

### 支付宝

```go
alipayProvider := paypkg.NewAlipayProvider(paypkg.AlipayConfig{
    AppID:      "your_app_id",
    PrivateKey: "your_private_key",
    PublicKey:  "alipay_public_key",
    Gateway:    "https://openapi.alipay.com/gateway.do",
    NotifyURL:  "https://your-domain.com/callback/alipay",
    ReturnURL:  "https://your-domain.com/payment/return",
})
```

### 微信支付

```go
wechatProvider := paypkg.NewWechatPayProvider(struct {
    AppID  string `json:"app_id"`
    APIKey string `json:"api_key"`
}{
    AppID:  "your_app_id",
    APIKey: "your_api_key",
})
```

### Stripe

```go
stripeProvider := paypkg.NewStripeProvider("sk_test_your_stripe_secret_key")
```

## 错误处理

### 常见错误码

- `400`: 请求参数错误
- `401`: 未授权访问
- `500`: 服务器内部错误

### 错误响应格式

```json
{
  "code": 400,
  "msg": "invalid request body"
}
```

## 监控和日志

### 日志级别

- `debug`: 开发环境，详细日志
- `info`: 生产环境，关键信息
- `warn`: 警告信息
- `error`: 错误信息

### 性能监控

服务集成了详细的性能监控：

- 请求响应时间
- 错误率统计
- 并发连接数
- 内存使用情况

## 部署

### Docker 部署

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o vippay app/vippay/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/vippay .
COPY --from=builder /app/vippay.json .
EXPOSE 8082
CMD ["./vippay", "-config", "vippay.json"]
```

### 环境变量

```bash
export DEPLOY_ENV=production
export REDIS_SERVER=redis:6379
export DB_NAME=grapery
export DB_USER=root
export DB_PASSWORD=password
export DB_ADDR=mysql:3306
```

## 开发

### 本地开发

1. 克隆项目
```bash
git clone <repository-url>
cd grapery
```

2. 安装依赖
```bash
go mod download
```

3. 启动服务
```bash
go run app/vippay/main.go -config vippay.json
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test ./pkg/pay -v
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License 