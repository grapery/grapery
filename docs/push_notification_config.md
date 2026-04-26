# 推送通知配置指南

本文档详细说明了 Apple APNs 和 Google FCM 推送通知的配置方法。

## 目录
1. [Apple APNs 配置](#apple-apns-配置)
2. [Google FCM 配置](#google-fcm-配置)
3. [环境变量配置](#环境变量配置)
4. [客户端集成](#客户端集成)
5. [API 端点](#api-端点)

---

## Apple APNs 配置

### 1. 创建 APNs Auth Key

1. 登录 [Apple Developer Portal](https://developer.apple.com/account/)
2. 进入 **Certificates, Identifiers & Profiles**
3. 选择 **Keys** → **+** 创建新 Key
4. 输入 Key 名称，勾选 **Apple Push Notifications service (APNs)**
5. 点击 **Continue** → **Register**
6. **下载 .p8 文件**（只能下载一次！）
7. 记录 **Key ID** 和 **Team ID**

### 2. 配置 App ID

1. 在 **Identifiers** 中选择你的 App ID
2. 确保已启用 **Push Notifications** 功能
3. 记录 **Bundle ID**

### 3. 必需的环境变量

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `APNS_BUNDLE_ID` | App Bundle ID（须与 Xcode / App ID 一致） | `com.rankquantity.voyager` |
| `APNS_TEAM_ID` | Apple Developer Team ID | `ABCDE12345` |
| `APNS_KEY_ID` | APNs Auth Key ID | `XXXXXXXXXX` |
| `APNS_PRIVATE_KEY` | .p8 文件内容 | `-----BEGIN PRIVATE KEY-----\n...` |
| `APNS_USE_SANDBOX` | 是否使用 Sandbox | `true` 或 `false` |

### 4. 私钥格式

私钥需要保留换行符。在 Shell 中设置时：

```bash
export APNS_PRIVATE_KEY=$(cat AuthKey_XXXXXXXXXX.p8)
```

在 Docker Compose 中：

```yaml
environment:
  APNS_PRIVATE_KEY: |
    -----BEGIN PRIVATE KEY-----
    MIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg...
    -----END PRIVATE KEY-----
```

---

## Google FCM 配置

### 1. 创建 Firebase 项目

1. 登录 [Firebase Console](https://console.firebase.google.com/)
2. 创建新项目或选择现有项目
3. 进入 **项目设置** → **服务账号**
4. 点击 **生成新的私钥**
5. 下载 JSON 密钥文件

### 2. 在 Google Play Console 中启用 FCM

1. 登录 [Google Play Console](https://play.google.com/console/)
2. 选择你的 App
3. 进入 **设置** → **API 访问**
4. 确保 Firebase Cloud Messaging API 已启用

### 3. 必需的环境变量

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `FCM_PROJECT_ID` | Firebase 项目 ID | `grapery-app` |
| `FCM_CREDENTIALS_JSON` | 服务账号 JSON | `{"type":"service_account",...}` |

### 4. JSON 凭证格式

```json
{
  "type": "service_account",
  "project_id": "your-project-id",
  "private_key_id": "...",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "client_email": "firebase-adminsdk-xxxxx@your-project-id.iam.gserviceaccount.com",
  "client_id": "...",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "..."
}
```

---

## 环境变量配置

### Docker 部署示例

```bash
docker run -d \
  -e APNS_BUNDLE_ID="com.rankquantity.voyager" \
  -e APNS_TEAM_ID="ABCDE12345" \
  -e APNS_KEY_ID="XXXXXXXXXX" \
  -e APNS_PRIVATE_KEY="$(cat AuthKey.p8)" \
  -e APNS_USE_SANDBOX="false" \
  -e FCM_PROJECT_ID="grapery-app" \
  -e FCM_CREDENTIALS_JSON="$(cat firebase-adminsdk.json)" \
  grapery/vippay
```

### Kubernetes Secret 示例

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: push-notification-secrets
type: Opaque
stringData:
  APNS_TEAM_ID: "ABCDE12345"
  APNS_KEY_ID: "XXXXXXXXXX"
  APNS_PRIVATE_KEY: |
    -----BEGIN PRIVATE KEY-----
    MIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg...
    -----END PRIVATE KEY-----
  FCM_PROJECT_ID: "grapery-app"
  FCM_CREDENTIALS_JSON: |
    {
      "type": "service_account",
      "project_id": "grapery-app",
      ...
    }
```

---

## 客户端集成

### iOS (Swift)

```swift
import UserNotifications
import UIKit

class AppDelegate: UIResponder, UIApplicationDelegate, UNUserNotificationCenterDelegate {
    
    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        // 请求通知权限
        UNUserNotificationCenter.current().delegate = self
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound]) { granted, error in
            if granted {
                DispatchQueue.main.async {
                    application.registerForRemoteNotifications()
                }
            }
        }
        return true
    }
    
    // 获取 device token
    func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        let token = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
        // 发送 token 到服务器
        registerDeviceToken(token: token, platform: "ios")
    }
    
    // 处理推送通知
    func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse, withCompletionHandler completionHandler: @escaping () -> Void) {
        let userInfo = response.notification.request.content.userInfo
        handleNotification(userInfo)
        completionHandler()
    }
}
```

### Android (Kotlin)

```kotlin
// 在 build.gradle 中添加
implementation 'com.google.firebase:firebase-messaging-ktx:23.4.0'

// FirebaseMessagingService
class MyFirebaseMessagingService : FirebaseMessagingService() {
    
    override fun onNewToken(token: String) {
        super.onNewToken(token)
        // 发送 token 到服务器
        sendTokenToServer(token, "android")
    }
    
    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        super.onMessageReceived(remoteMessage)
        
        // 处理通知
        remoteMessage.notification?.let { notification ->
            showNotification(notification.title, notification.body)
        }
        
        // 处理数据
        remoteMessage.data.let { data ->
            handleNotificationData(data)
        }
    }
    
    private fun sendTokenToServer(token: String, platform: String) {
        // POST /api/devices/register
        // { "deviceToken": token, "platform": platform }
    }
}
```

---

## API 端点

### 设备注册

```http
POST /api/devices/register
Authorization: Bearer <token>
Content-Type: application/json

{
  "deviceToken": "xxx",
  "platform": "ios",       // ios, android, macos, ipados
  "appVersion": "1.0.0",
  "deviceModel": "iPhone 15 Pro",
  "osVersion": "iOS 17.0"
}
```

### 设备注销

```http
POST /api/devices/unregister
Authorization: Bearer <token>
Content-Type: application/json

{
  "deviceToken": "xxx"
}
```

### 更新徽章数

```http
POST /api/devices/badge
Authorization: Bearer <token>
Content-Type: application/json

{
  "count": 5
}
```

---

## 通知类型

| 类型 | Category | 说明 |
|-----|----------|------|
| `like` | LIKE | 点赞通知 |
| `comment` | COMMENT | 评论通知 |
| `follow` | FOLLOW | 关注通知 |
| `mention` | MENTION | 提及通知 |
| `message` | MESSAGE | 私信通知 |
| `story_update` | STORY_UPDATE | 故事更新 |
| `subscription` | SUBSCRIPTION | 订阅相关 |
| `system` | SYSTEM | 系统通知 |

---

## 通知载荷格式

### APNs 载荷

```json
{
  "aps": {
    "alert": {
      "title": "New Like",
      "body": "John liked your story"
    },
    "badge": 1,
    "sound": "default",
    "category": "LIKE",
    "mutable-content": 1
  },
  "notificationId": "xxx",
  "type": "like",
  "link": "/story/123"
}
```

### FCM 载荷

```json
{
  "message": {
    "token": "device_token",
    "notification": {
      "title": "New Like",
      "body": "John liked your story"
    },
    "data": {
      "notificationId": "xxx",
      "type": "like",
      "link": "/story/123"
    },
    "android": {
      "priority": "high",
      "notification": {
        "channel_id": "LIKE"
      }
    }
  }
}
```

---

## 配置检查清单

### APNs
- [ ] 已创建 APNs Auth Key (.p8)
- [ ] 已配置 `APNS_BUNDLE_ID`
- [ ] 已配置 `APNS_TEAM_ID`
- [ ] 已配置 `APNS_KEY_ID`
- [ ] 已配置 `APNS_PRIVATE_KEY`
- [ ] 已在 App ID 中启用 Push Notifications
- [ ] iOS 客户端已集成并测试

### FCM
- [ ] 已创建 Firebase 项目
- [ ] 已下载服务账号 JSON
- [ ] 已配置 `FCM_PROJECT_ID`
- [ ] 已配置 `FCM_CREDENTIALS_JSON`
- [ ] Android 客户端已集成 Firebase SDK
- [ ] Android 客户端已测试

---

## 故障排查

### APNs 常见错误

| 错误 | 原因 | 解决方案 |
|-----|------|---------|
| `BadDeviceToken` | Token 无效或已过期 | 客户端需要重新注册 |
| `Unregistered` | App 已卸载 | 从数据库移除该设备 |
| `ExpiredProviderToken` | JWT 已过期 | 服务会自动刷新 |
| `BadCertificate` | 证书/密钥配置错误 | 检查私钥格式 |

### FCM 常见错误

| 错误 | 原因 | 解决方案 |
|-----|------|---------|
| `UNREGISTERED` | Token 无效 | 从数据库移除该设备 |
| `INVALID_ARGUMENT` | 请求参数错误 | 检查 token 格式 |
| `SENDER_ID_MISMATCH` | 项目 ID 不匹配 | 检查 FCM 配置 |
| `QUOTA_EXCEEDED` | 配额超限 | 等待或升级配额 |

### 调试日志

```bash
# 查看推送相关日志
kubectl logs -f deployment/vippay | grep -i "push\|apns\|fcm"
```

