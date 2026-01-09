# Android Client ID 配置位置说明

## 📋 问题：Android Client ID 应该写在哪里？

**答案：Android Client ID 不需要在代码或配置文件中硬编码，它只在 Google Cloud Console 中使用。**

## 🔍 Android Client ID 的用途

### 1. Google Cloud Console 配置（唯一用途）

Android Client ID (`345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev.apps.googleusercontent.com`) **仅在 Google Cloud Console 中配置 Android OAuth 客户端时使用**：

1. 登录 [Google Cloud Console](https://console.cloud.google.com/)
2. 进入 **APIs & Services** > **Credentials**
3. 创建或编辑 **OAuth 2.0 Client ID**（类型：Android）
4. 配置：
   - **Name**: Grapery Android Client
   - **Package name**: `com.rankquantity.pioneer`
   - **SHA-1 certificate fingerprint**: `01:C4:61:0E:33:E1:8A:5D:93:0E:E5:19:D8:A7:D5:77:60:EC:3C:74`
   - **Client ID**: `345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev.apps.googleusercontent.com`

### 2. 为什么不需要在代码中硬编码？

**Android 应用实际登录时应该使用 Server Client ID，而不是 Android Client ID。**

#### 正确的流程：

1. **Android 应用启动时**，调用后端 API 获取配置：
   ```kotlin
   GET /api/vippay/google-oauth/config
   ```

2. **后端返回 Server Client ID**：
   ```json
   {
     "data": {
       "client_id": "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com"
     }
   }
   ```

3. **Android 应用使用 Server Client ID 请求 Google ID Token**：
   ```kotlin
   val gso = GoogleSignInOptions.Builder(GoogleSignInOptions.DEFAULT_SIGN_IN)
       .requestIdToken(serverClientId)  // 使用 Server Client ID，不是 Android Client ID
       .requestEmail()
       .build()
   ```

#### 为什么不能使用 Android Client ID？

- 如果使用 Android Client ID 请求 ID Token，Token 的 `audience`（aud）字段将是 Android Client ID
- 后端验证时，会检查 Token 的 `aud` 是否等于后端配置的 `GOOGLE_CLIENT_ID`（Server Client ID）
- 如果 `aud` 不匹配，后端验证会失败 ❌

## 📝 Android Client ID 的记录位置

### 1. 文档记录（已记录）✅

Android Client ID 已在以下文档中记录，**仅作为参考信息**：

- `grapery/docs/GOOGLE_OAUTH_CLIENT_IDS.md` - 所有 Client ID 的清单
- `grapery/docs/oauth_config_requirements.md` - OAuth 配置要求文档

### 2. 不需要在以下位置配置

- ❌ **不需要**在 `vippay.json` 配置文件中
- ❌ **不需要**在后端环境变量中
- ❌ **不需要**在 Android 应用的代码中硬编码
- ❌ **不需要**在 `google-services.json` 文件中（如果使用 Firebase）

### 3. 实际需要配置的位置

#### 后端配置（必需）✅

后端需要配置的是 **Server Client ID**，不是 Android Client ID：

1. **环境变量**：
   ```bash
   export GOOGLE_CLIENT_ID=345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com
   ```

2. **vippay.json 配置文件**（可选）：
   ```json
   {
     "oauth": {
       "google": {
         "client_id": "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com"
       }
     }
   }
   ```

#### Android 应用配置（必需）✅

Android 应用需要从后端获取 Server Client ID：

1. **调用后端 API**：
   ```kotlin
   // 从后端获取 Server Client ID
   val config = fetchGoogleOAuthConfig()
   val serverClientId = config.client_id
   ```

2. **使用 Server Client ID 配置 Google Sign-In**：
   ```kotlin
   val gso = GoogleSignInOptions.Builder(GoogleSignInOptions.DEFAULT_SIGN_IN)
       .requestIdToken(serverClientId)  // Server Client ID
       .requestEmail()
       .build()
   ```

## 🎯 总结

| 项目 | Android Client ID | Server Client ID |
|------|------------------|------------------|
| **用途** | Google Cloud Console 配置 | 后端验证 + 应用登录 |
| **配置位置** | Google Cloud Console | 后端环境变量/配置文件 |
| **应用代码** | ❌ 不需要 | ✅ 从后端 API 获取 |
| **硬编码** | ❌ 不需要 | ❌ 不应该硬编码 |

**Android Client ID 只需要在 Google Cloud Console 中配置，不需要在代码或配置文件中硬编码。**

## 📚 相关文档

- [Google OAuth Client ID 配置清单](./GOOGLE_OAUTH_CLIENT_IDS.md)
- [Google OAuth Client ID 获取流程](./google_oauth_client_id_flow.md)
- [OAuth 配置要求](./oauth_config_requirements.md)

