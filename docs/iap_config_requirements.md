# IAP (In-App Purchase) 配置指南

本文档详细说明了在 Apple App Store 和 Google Play Store 实现应用内购买所需的配置。

## 目录
1. [Apple App Store IAP 配置](#apple-app-store-iap-配置)
2. [Google Play Store IAP 配置](#google-play-store-iap-配置)
3. [环境变量配置](#环境变量配置)
4. [Webhook 配置](#webhook-配置)
5. [产品配置](#产品配置)

---

## Apple App Store IAP 配置

### 1. App Store Connect 配置

#### 1.1 创建 App Store Connect API Key
1. 登录 [App Store Connect](https://appstoreconnect.apple.com/)
2. 进入 **Users and Access** → **Keys** → **App Store Connect API**
3. 点击 **+** 创建新的 API Key
4. 选择 **Admin** 或 **App Manager** 权限
5. 下载 `.p8` 私钥文件（只能下载一次！）
6. 记录以下信息：
   - **Issuer ID**: 页面顶部显示的 Issuer ID
   - **Key ID**: 创建的 Key ID
   - **Private Key**: `.p8` 文件内容

#### 1.2 配置 App 内购产品
1. 在 App Store Connect 中选择你的 App
2. 进入 **Features** → **In-App Purchases**
3. 创建订阅产品或一次性购买产品
4. 配置产品 ID、价格、描述等

#### 1.3 配置 Server Notifications (Webhook)
1. 进入 **App Information** → **App Store Server Notifications**
2. 设置 **Production Server URL**: `https://your-domain.com/api/iap/apple/notification`
3. 设置 **Sandbox Server URL**: `https://your-domain.com/api/iap/apple/notification`
4. 选择 **Version 2** 通知格式

### 2. 必需的环境变量

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `APPLE_BUNDLE_ID` | App 的 Bundle ID | `com.rankquantity.voyager` |
| `APPLE_ISSUER_ID` | App Store Connect API Issuer ID | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| `APPLE_KEY_ID` | App Store Connect API Key ID | `XXXXXXXXXX` |
| `APPLE_PRIVATE_KEY` | `.p8` 私钥内容（保留换行符） | `-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----` |

#### Sandbox 环境配置（可选）
| 环境变量 | 说明 |
|---------|------|
| `APPLE_SANDBOX_BUNDLE_ID` | Sandbox Bundle ID（默认使用生产配置） |
| `APPLE_SANDBOX_ISSUER_ID` | Sandbox Issuer ID |
| `APPLE_SANDBOX_KEY_ID` | Sandbox Key ID |
| `APPLE_SANDBOX_PRIVATE_KEY` | Sandbox 私钥 |

---

## Google Play Store IAP 配置

### 1. Google Play Console 配置

#### 1.1 创建服务账号
1. 登录 [Google Cloud Console](https://console.cloud.google.com/)
2. 选择或创建项目
3. 进入 **IAM & Admin** → **Service Accounts**
4. 点击 **Create Service Account**
5. 填写服务账号信息，创建完成后点击该账号
6. 进入 **Keys** 标签页，点击 **Add Key** → **Create new key**
7. 选择 **JSON** 格式，下载密钥文件

#### 1.2 关联服务账号到 Google Play
1. 登录 [Google Play Console](https://play.google.com/console/)
2. 进入 **Setup** → **API access**
3. 点击 **Link** 关联 Google Cloud 项目
4. 在 **Service accounts** 部分，找到刚创建的服务账号
5. 点击 **Grant access**，授予以下权限：
   - **View app information and download bulk reports**
   - **View financial data, orders, and cancellation survey responses**
   - **Manage orders and subscriptions**

#### 1.3 配置 App 内购产品
1. 在 Google Play Console 选择你的 App
2. 进入 **Monetize** → **Products** → **Subscriptions** 或 **In-app products**
3. 创建并配置产品

#### 1.4 配置 Real-time Developer Notifications (RTDN)
1. 进入 **Monetize** → **Monetization setup**
2. 在 **Real-time developer notifications** 部分
3. 配置 **Topic name**: 创建一个 Cloud Pub/Sub Topic
4. 配置推送订阅，将消息推送到: `https://your-domain.com/api/iap/google/notification`

### 2. 必需的环境变量

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `GOOGLE_PACKAGE_NAME` | App 的包名（须与 Play 控制台一致） | `com.rankquantity.voyager` |
| `GOOGLE_SERVICE_ACCOUNT_KEY` | 服务账号 JSON 密钥内容 | `{"type":"service_account",...}` |

#### Sandbox 环境配置（可选）
| 环境变量 | 说明 |
|---------|------|
| `GOOGLE_SANDBOX_PACKAGE_NAME` | Sandbox 包名（默认使用生产配置） |
| `GOOGLE_SANDBOX_SERVICE_ACCOUNT_KEY` | Sandbox 服务账号密钥 |

---

## 环境变量配置

### Docker 部署示例

```bash
docker run -d \
  -e APPLE_BUNDLE_ID="com.rankquantity.voyager" \
  -e APPLE_ISSUER_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \
  -e APPLE_KEY_ID="XXXXXXXXXX" \
  -e APPLE_PRIVATE_KEY="$(cat /path/to/AuthKey.p8)" \
  -e GOOGLE_PACKAGE_NAME="com.rankquantity.voyager" \
  -e GOOGLE_SERVICE_ACCOUNT_KEY="$(cat /path/to/service-account.json)" \
  grapery/vippay
```

### Kubernetes Secret 示例

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: iap-secrets
type: Opaque
stringData:
  APPLE_ISSUER_ID: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  APPLE_KEY_ID: "XXXXXXXXXX"
  APPLE_PRIVATE_KEY: |
    -----BEGIN PRIVATE KEY-----
    MIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg...
    -----END PRIVATE KEY-----
  GOOGLE_SERVICE_ACCOUNT_KEY: |
    {
      "type": "service_account",
      "project_id": "your-project-id",
      ...
    }
```

---

## Webhook 配置

### API 端点

| 平台 | 端点 | 说明 |
|-----|------|------|
| Apple | `POST /api/iap/apple/notification` | App Store Server Notifications V2 |
| Google | `POST /api/iap/google/notification` | Real-time Developer Notifications |

### Apple Notification Types
- `INITIAL_BUY` - 首次购买
- `DID_RENEW` - 订阅续费
- `DID_CHANGE_RENEWAL_STATUS` - 续订状态变更
- `DID_FAIL_TO_RENEW` - 续费失败
- `DID_RECOVER` - 恢复购买
- `CANCEL` - 取消订阅
- `REFUND` - 退款

### Google Notification Types
- `SUBSCRIPTION_RECOVERED` - 订阅恢复
- `SUBSCRIPTION_RENEWED` - 订阅续费
- `SUBSCRIPTION_CANCELED` - 订阅取消
- `SUBSCRIPTION_PURCHASED` - 新订阅
- `SUBSCRIPTION_DEFERRED` - 订阅延期
- `SUBSCRIPTION_PAUSED` - 订阅暂停
- `SUBSCRIPTION_EXPIRED` - 订阅过期

---

## 产品配置

> **App Store Connect 逐步配置（SKU、价格、中英文元数据、审核清单）**见 [app-store-connect-iap-setup.md](./app-store-connect-iap-setup.md)。

### 建议的产品 ID 命名规范

| 类型 | 命名格式 | 示例 |
|-----|---------|------|
| 月度订阅 | `{app}.subscription.monthly.{tier}` | `voyager.subscription.monthly.premium` |
| 年度订阅 | `{app}.subscription.yearly.{tier}` | `voyager.subscription.yearly.premium` |
| 代币包 | `{app}.tokens.{amount}` | `voyager.tokens.1000` |
| 一次性购买 | `{app}.onetime.{feature}` | `voyager.onetime.unlock_all` |

### 订阅等级建议

| 等级 | 月价格 | 年价格 | 特权 |
|-----|-------|-------|------|
| Free | $0 | $0 | 基础功能，10,000 Token/月 |
| Premium | $9.99 | $99.99 | 高级功能，100,000 Token/月 |
| Pro | $19.99 | $199.99 | 专业功能，无限 Token |

---

## 测试指南

### Apple Sandbox 测试
1. 在 App Store Connect 创建 Sandbox 测试账号
2. 在设备上登出 App Store，使用 Sandbox 账号登录
3. 在 App 中进行购买测试

### Google 测试
1. 在 Google Play Console 添加测试账号（License Testers）
2. 创建内部测试轨道或封闭测试
3. 使用测试账号进行购买测试

---

## 配置检查清单

### Apple IAP
- [ ] 已创建 App Store Connect API Key
- [ ] 已下载并保存 `.p8` 私钥文件
- [ ] 已配置 `APPLE_BUNDLE_ID` 环境变量
- [ ] 已配置 `APPLE_ISSUER_ID` 环境变量
- [ ] 已配置 `APPLE_KEY_ID` 环境变量
- [ ] 已配置 `APPLE_PRIVATE_KEY` 环境变量
- [ ] 已在 App Store Connect 配置 Server Notifications URL
- [ ] 已创建 In-App Purchase 产品

### Google IAP
- [ ] 已创建 Google Cloud 服务账号
- [ ] 已下载服务账号 JSON 密钥
- [ ] 已在 Google Play Console 关联服务账号并授权
- [ ] 已配置 `GOOGLE_PACKAGE_NAME` 环境变量
- [ ] 已配置 `GOOGLE_SERVICE_ACCOUNT_KEY` 环境变量
- [ ] 已配置 Real-time Developer Notifications
- [ ] 已创建 In-App Purchase 产品

---

## 故障排查

### 常见问题

1. **Apple 验证失败 (21007)**
   - 原因：使用生产环境验证 Sandbox 收据
   - 解决：确保在 Sandbox 模式下使用 Sandbox 验证 URL

2. **Google 401 Unauthorized**
   - 原因：服务账号权限不足或未关联
   - 解决：检查 Google Play Console 中的服务账号权限

3. **Webhook 未收到通知**
   - 检查防火墙/安全组是否允许 Apple/Google 的 IP
   - 确保 Webhook URL 使用 HTTPS
   - 检查服务日志中的错误信息

### 日志查看
```bash
# 查看 IAP 相关日志
kubectl logs -f deployment/vippay | grep -i iap
```

