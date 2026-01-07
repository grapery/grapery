# VIPPay 服务快速启动指南

## 问题：为什么返回 YOUR_GOOGLE_CLIENT_ID？

这是因为 Google OAuth Client ID 还没有配置。后端按照以下优先级读取配置：

1. **环境变量** `GOOGLE_CLIENT_ID`（最高优先级）
2. **配置文件** `cmd/vippay/vippay.json` 中的 `oauth.google.client_id`
3. **默认值** `YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com`（占位符）

## ✅ 解决方案

### 方式 1：修改配置文件（开发/测试环境）

已自动配置 ✅：`cmd/vippay/vippay.json` 已更新为正确的 Client ID

```json
{
  "oauth": {
    "google": {
      "client_id": "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com"
    }
  }
}
```

**重启服务后生效**：

```bash
# 如果是直接运行
cd grapery/cmd/vippay
./grapery-vippay

# 如果是 Docker
cd grapery
docker-compose -f docker-compose.vippay.yml restart
```

### 方式 2：使用环境变量（生产环境推荐）

#### 步骤 1：创建 .env 文件

```bash
cd grapery
cp env.vippay.example .env
```

#### 步骤 2：编辑 .env 文件

确保设置了以下关键配置：

```bash
# Google OAuth (已预填正确的 Client ID)
GOOGLE_CLIENT_ID=345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com

# Apple OAuth
APPLE_BUNDLE_ID=com.rankquantity.voyager

# JWT 密钥（必须修改为强密钥）
JWT_SECRET=$(openssl rand -base64 32)

# 数据库配置（根据实际情况修改）
DB_DATABASE=grapery
DB_USERNAME=root
DB_PASSWORD=your-actual-password
DB_ADDRESS=localhost:3306
```

#### 步骤 3：启动服务

```bash
# 使用 Docker Compose
docker-compose -f docker-compose.vippay.yml up -d

# 或者直接运行（会自动读取 .env）
cd cmd/vippay
export $(cat ../../.env | xargs)
./grapery-vippay
```

## 验证配置是否生效

### 测试 API 端点

```bash
curl http://127.0.0.1:8081/api/vippay/google-oauth/config
```

**正确的响应**（不再是 YOUR_GOOGLE_CLIENT_ID）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "clientId": "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com",
    "enabled": true,
    "isAvailable": true
  }
}
```

### iOS 应用验证

重新运行 iOS 应用，应该看到：

```
✅ Got Google OAuth config
✅ GoogleSignIn configured with clientID: 345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com
```

而不再是：

```
❌ Response data is null
❌ Failed to get Google OAuth config
```

## 🔍 关于 Client ID 的说明

### 为什么使用这个 Client ID？

`345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com` 是 **Web/Server Client ID**，不是 iOS 或 Android Client ID。

### 三种 Client ID 的区别

| 类型 | Client ID | 用途 |
|------|-----------|------|
| **iOS** | `345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb` | 在 Google Cloud Console 配置 iOS 应用 |
| **Android** | `345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev` | 在 Google Cloud Console 配置 Android 应用 |
| **Web/Server** ✅ | `345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf` | **后端验证 ID Token 的 audience** |

### 为什么后端必须使用 Server Client ID？

1. iOS/Android 应用使用 Server Client ID 请求 Google ID Token
2. 返回的 ID Token 的 `audience` 字段将是 Server Client ID
3. 后端验证 ID Token 时，会检查 `audience` 是否匹配 `GOOGLE_CLIENT_ID`
4. 如果使用 iOS/Android Client ID，验证会失败（audience 不匹配）

## 📚 相关文档

- [OAuth 配置要求](docs/oauth_config_requirements.md)
- [Google OAuth Client ID 详解](docs/GOOGLE_OAUTH_CLIENT_IDS.md)
- [Google OAuth 流程说明](docs/google_oauth_client_id_flow.md)
- [VIPPay 配置指南](cmd/vippay/CONFIG.md)

## ❓ 常见问题

### Q1: 我修改了配置，为什么还是返回 YOUR_GOOGLE_CLIENT_ID？

**A:** 需要重启服务。配置只在服务启动时读取一次。

### Q2: 环境变量和配置文件都设置了，哪个会生效？

**A:** 环境变量优先级更高，会覆盖配置文件中的值。

### Q3: 我能在配置文件中直接填 iOS Client ID 吗？

**A:** 不能。后端必须使用 Web/Server Client ID，否则 ID Token 验证会失败。

### Q4: 如何生成强 JWT_SECRET？

**A:** 使用以下命令：

```bash
openssl rand -base64 32
```

### Q5: Docker 部署时如何传递环境变量？

**A:** 在 `docker-compose.vippay.yml` 中已配置使用 `.env` 文件：

```yaml
services:
  vippay:
    env_file:
      - .env  # 自动加载 .env 文件中的所有环境变量
```

## 🎉 快速测试

```bash
# 1. 确认配置文件已更新
cat cmd/vippay/vippay.json | grep client_id

# 2. 重启服务
docker-compose -f docker-compose.vippay.yml restart

# 3. 测试 API
curl http://127.0.0.1:8081/api/vippay/google-oauth/config | jq '.data.clientId'

# 4. 应该输出正确的 Client ID（不是 YOUR_GOOGLE_CLIENT_ID）
# "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com"
```

