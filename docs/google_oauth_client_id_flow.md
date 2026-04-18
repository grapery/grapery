# Google OAuth Client ID 获取流程说明

## 📋 概述

iOS 和 Android 应用**不应该硬编码** Google Client ID，而是应该从后端 API 动态获取 **Server Client ID**（Web/Server Client ID），然后使用这个 Client ID 来请求 Google ID Token。

## 🔄 工作流程

### 1. 应用启动时获取配置

iOS/Android 应用在启动时或需要 Google 登录时，调用后端 API 获取配置：

```http
GET /api/vippay/google-oauth/config
```

### 2. 后端返回的响应格式

```json
{
  "code": 0,
  "msg": "success",
  "message": "success",
  "success": true,
  "data": {
    // iOS 使用这些字段（camelCase）
    "clientId": "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com",
    "redirectUri": "",
    "scope": "openid email profile",
    "responseType": "id_token",
    "state": null,
    
    // Android 使用这些字段（snake_case）
    "client_id": "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com",
    "redirect_uri": "",
    "response_type": "id_token",
    "enabled": true,
    "isAvailable": true,
    "provider": "google",
    "scopes": ["openid", "email", "profile"]
  }
}
```

**重要字段说明：**
- `clientId` / `client_id`: **Server Client ID**（Web/Server Client ID），这是后端用于验证 ID Token 的 Client ID
- `enabled` / `isAvailable`: 表示 Google OAuth 是否可用
- `scope` / `scopes`: OAuth 请求的权限范围

### 3. iOS 应用使用流程

```swift
// 1. 从后端获取配置
func fetchGoogleOAuthConfig() async throws -> GoogleOAuthConfig {
    let url = URL(string: "https://www.rankquantity.xyz/api/vippay/google-oauth/config")!
    let (data, _) = try await URLSession.shared.data(from: url)
    let response = try JSONDecoder().decode(GoogleOAuthConfigResponse.self, from: data)
    return response.data
}

// 2. 使用返回的 clientId 请求 Google ID Token
func signInWithGoogle() async throws {
    let config = try await fetchGoogleOAuthConfig()
    
    // 使用从后端获取的 Server Client ID
    guard let clientID = config.clientId else {
        throw OAuthError.missingClientID
    }
    
    // 使用 Google Sign-In SDK 请求 ID Token
    let gidConfiguration = GIDConfiguration(clientID: clientID)
    GIDSignIn.sharedInstance.configuration = gidConfiguration
    
    // 请求 ID Token（audience 将是 Server Client ID）
    let result = try await GIDSignIn.sharedInstance.signIn(withPresenting: viewController)
    guard let idToken = result.user.idToken?.tokenString else {
        throw OAuthError.missingIDToken
    }
    
    // 3. 将 ID Token 发送到后端进行验证和登录
    try await sendIDTokenToBackend(idToken: idToken)
}
```

### 4. Android 应用使用流程

```kotlin
// 1. 从后端获取配置
suspend fun fetchGoogleOAuthConfig(): GoogleOAuthConfig {
    val response = httpClient.get("https://www.rankquantity.xyz/api/vippay/google-oauth/config")
    return response.body<GoogleOAuthConfigResponse>().data
}

// 2. 使用返回的 clientId 请求 Google ID Token
suspend fun signInWithGoogle() {
    val config = fetchGoogleOAuthConfig()
    
    // 使用从后端获取的 Server Client ID
    val serverClientId = config.client_id
        ?: throw OAuthException("Missing client ID")
    
    // 使用 Google Sign-In SDK 请求 ID Token
    val gso = GoogleSignInOptions.Builder(GoogleSignInOptions.DEFAULT_SIGN_IN)
        .requestIdToken(serverClientId)  // 使用 Server Client ID
        .requestEmail()
        .build()
    
    val googleSignInClient = GoogleSignIn.getClient(context, gso)
    val signInIntent = googleSignInClient.signInIntent
    val result = activityResultLauncher.launch(signInIntent)
    
    // 3. 从结果中获取 ID Token
    val task = GoogleSignIn.getSignedInAccountFromIntent(result.data)
    val account = task.getResult(ApiException::class.java)
    val idToken = account.idToken
    
    // 4. 将 ID Token 发送到后端进行验证和登录
    sendIDTokenToBackend(idToken)
}
```

## ✅ 当前配置支持情况

### 后端实现 ✅

1. **API 端点已实现**
   - 路由：`GET /api/vippay/google-oauth/config`
   - 处理器：`GoogleOAuthHandler.GetGoogleOAuthConfig()`
   - 位置：`grapery/internal/transport/pay/google_oauth_handler.go:423`

2. **配置读取逻辑 ✅**
   - 优先级：环境变量 `GOOGLE_CLIENT_ID` > `vippay.json` 配置文件 > 默认值
   - 代码位置：`grapery/internal/transport/pay/google_oauth_handler.go:692-729`

3. **返回的 Client ID 类型 ✅**
   - 返回的是 **Server Client ID**（Web/Server Client ID）
   - 值：`345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com`
   - 这是后端用于验证 ID Token 的 Client ID

### 配置来源

后端会按以下优先级读取 `GOOGLE_CLIENT_ID`：

1. **环境变量**（最高优先级）
   ```bash
   export GOOGLE_CLIENT_ID=345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com
   ```

2. **vippay.json 配置文件**
   ```json
   {
     "oauth": {
       "google": {
         "client_id": "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com"
       }
     }
   }
   ```

3. **默认值**（如果都未设置）
   - `YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com`

## 🔍 为什么使用 Server Client ID？

### 问题：为什么不能使用 iOS/Android Client ID？

1. **后端验证需要**
   - 后端在验证 Google ID Token 时，会检查 Token 的 `audience`（aud）字段
   - 后端配置的 `GOOGLE_CLIENT_ID` 必须是 Server Client ID
   - 如果 iOS/Android 使用各自的 Client ID 请求 Token，Token 的 `aud` 将是 iOS/Android Client ID
   - 后端验证时会失败，因为 `aud` 不匹配

2. **统一管理**
   - 使用 Server Client ID 可以统一管理所有平台的 OAuth 配置
   - 后端可以动态返回配置，无需客户端硬编码
   - 便于后续更换或更新 Client ID

### 解决方案：使用 Server Client ID

1. **iOS/Android 都使用 Server Client ID**
   - iOS 应用调用 `requestIdToken(serverClientId)` 时，返回的 ID Token 的 `aud` 将是 Server Client ID
   - Android 应用调用 `requestIdToken(serverClientId)` 时，返回的 ID Token 的 `aud` 将是 Server Client ID
   - 后端验证时，`aud` 匹配，验证成功 ✅

2. **从后端动态获取**
   - 应用启动时从后端获取 Server Client ID
   - 使用获取到的 Client ID 请求 Google ID Token
   - 这样后端可以统一管理配置，客户端无需硬编码

## 📝 验证步骤

### 1. 测试后端 API

```bash
# 测试配置端点
curl https://www.rankquantity.xyz/api/vippay/google-oauth/config

# 预期响应
{
  "code": 0,
  "success": true,
  "data": {
    "clientId": "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com",
    "enabled": true,
    ...
  }
}
```

### 2. 检查配置是否正确

```bash
# 检查环境变量
echo $GOOGLE_CLIENT_ID

# 或检查配置文件
cat vippay.json | jq '.oauth.google.client_id'
```

### 3. iOS/Android 应用集成

- ✅ iOS 应用调用 `GET /api/vippay/google-oauth/config` 获取 `clientId`
- ✅ 使用返回的 `clientId`（Server Client ID）请求 Google ID Token
- ✅ 将 ID Token 发送到 `POST /api/vippay/google-oauth/signin` 进行登录

## 🎯 总结

**当前配置完全支持** iOS 和 Android 应用从后端获取 Server Client ID：

1. ✅ 后端 API `/api/vippay/google-oauth/config` 已实现
2. ✅ 返回的 `clientId` 是 Server Client ID（Web/Server Client ID）
3. ✅ 配置可以从环境变量或 `vippay.json` 读取
4. ✅ iOS/Android 应用可以使用返回的 Client ID 请求 Google ID Token
5. ✅ 后端可以正确验证使用 Server Client ID 请求的 ID Token

**下一步**：确保 iOS 和 Android 应用已实现从后端获取配置的逻辑，而不是硬编码 Client ID。

