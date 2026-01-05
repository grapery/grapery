# VIPPay Service Configuration Guide

## 配置文件说明

VIPPay 服务支持两种配置方式：
1. **配置文件** (`vippay.json`) - YAML 格式，尽管扩展名是 `.json`
2. **环境变量** - 优先级高于配置文件

## 配置优先级

环境变量 > 配置文件 > 默认值

## 配置文件位置

- **开发环境**: `grapery/cmd/vippay/vippay.json`
- **Docker 容器**: `/app/vippay.json`
- **运行时**: 服务会在工作目录查找 `vippay.json`

## 配置加载逻辑

1. 如果指定了 `-config` 参数，尝试加载指定文件
2. 如果配置文件不存在，自动回退到环境变量配置
3. 如果配置文件存在但格式错误，服务会退出并报错

## 必需的环境变量

以下环境变量**必须**通过环境变量设置（不应放在配置文件中，因为包含敏感信息）：

### 数据库配置
- `DB_DATABASE` - 数据库名称
- `DB_USERNAME` - 数据库用户名
- `DB_PASSWORD` - 数据库密码（敏感）
- `DB_ADDRESS` - 数据库地址

### JWT 配置
- `JWT_SECRET` - JWT 密钥（敏感，必须设置）
- `JWT_EXPIRY_HOURS` - Token 过期时间（小时）

### Apple IAP 配置
- `APPLE_BUNDLE_ID` - App Bundle ID
- `APPLE_ISSUER_ID` - App Store Connect Issuer ID
- `APPLE_KEY_ID` - App Store Connect Key ID
- `APPLE_PRIVATE_KEY` - App Store Connect 私钥（敏感）

### Google IAP 配置
- `GOOGLE_PACKAGE_NAME` - App 包名
- `GOOGLE_SERVICE_ACCOUNT_KEY` - Google 服务账号 JSON 密钥（敏感）

### Aliyun OSS 配置
- `ALIYUN_API_KEY` - 阿里云 API Key（敏感）
- `ALIYUN_SECRET_KEY` - 阿里云 Secret Key（敏感）
- `ALIYUN_ENDPOINT` - OSS 端点
- `ALIYUN_BUCKET` - OSS Bucket 名称
- `ALIYUN_ROLE_ARN` - STS Role ARN（可选）

### 其他配置
- `VIPPAY_PORT` - 服务端口（默认: 8081）
- `VIPPAY_DOMAIN` - 服务域名

## 使用示例

### 方式 1: 使用配置文件 + 环境变量

```bash
# 配置文件提供默认值
# 环境变量覆盖敏感信息和特定配置
export DB_PASSWORD="your-password"
export JWT_SECRET="your-jwt-secret"
export APPLE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"

# 启动服务（会自动查找 vippay.json）
./grapery-vippay
```

### 方式 2: 仅使用环境变量

```bash
# 不提供配置文件，完全使用环境变量
export DB_DATABASE=grapery
export DB_USERNAME=root
export DB_PASSWORD=password
export DB_ADDRESS=localhost
export JWT_SECRET=your-secret
# ... 其他环境变量

# 启动服务（会自动回退到环境变量）
./grapery-vippay
```

### 方式 3: 指定配置文件路径

```bash
# 使用自定义配置文件路径
./grapery-vippay -config /path/to/custom-config.yaml
```

## Docker 部署

配置文件会被自动复制到 Docker 镜像中（`/app/vippay.json`）。

在 `docker-compose.vippay.yml` 中，通过环境变量覆盖配置：

```yaml
services:
  vippay:
    image: grapery-vippay:dev
    env_file:
      - .env  # 包含所有环境变量
    # 或者直接设置环境变量
    environment:
      - DB_PASSWORD=${DB_PASSWORD}
      - JWT_SECRET=${JWT_SECRET}
      # ...
```

## 注意事项

1. **敏感信息**: 永远不要将密码、密钥等敏感信息提交到版本控制系统
2. **配置文件格式**: 虽然文件名是 `.json`，但内容必须是 YAML 格式
3. **环境变量优先**: 环境变量会覆盖配置文件中的相同配置项
4. **回退机制**: 如果配置文件不存在，服务会自动使用环境变量，不会报错

## 相关文档

- [IAP 配置要求](../docs/iap_config_requirements.md)
- [OAuth 配置要求](../docs/oauth_config_requirements.md)
- [推送通知配置](../docs/push_notification_config.md)

