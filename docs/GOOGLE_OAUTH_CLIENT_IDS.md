# Google OAuth 客户端 ID 配置清单

本文档记录了所有已配置的 Google OAuth 客户端 ID 信息。

## 📋 客户端 ID 列表

### iOS 客户端（voyager）
- **Client ID**: `345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb.apps.googleusercontent.com`
- **Bundle ID**: `com.rankquantity.voyager`
- **用途**: 在 Google Cloud Console 中配置 iOS OAuth 客户端
- **注意**: iOS 应用实际登录时应使用 Server Client ID（从后端 API 获取）

### Android 客户端（pioneer）
- **Client ID**: `345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev.apps.googleusercontent.com`
- **Package name**: `com.rankquantity.pioneer`
- **SHA-1 指纹**:
  - Debug: `01:C4:61:0E:33:E1:8A:5D:93:0E:E5:19:D8:A7:D5:77:60:EC:3C:74`
  - Release: （待配置）
- **用途**: 在 Google Cloud Console 中配置 Android OAuth 客户端
- **注意**: Android 应用实际登录时应使用 Server Client ID（从后端 API 获取）

### Web/Server 客户端（后端）
- **Client ID**: `345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com` ✅
- **用途**: 
  - 后端环境变量 `GOOGLE_CLIENT_ID` 的值
  - Android 和 iOS 客户端在请求 ID Token 时使用
  - 后端验证 ID Token 的 audience

## 🔧 Google Cloud Console 配置步骤

### 1. 配置 iOS 客户端

1. 登录 [Google Cloud Console](https://console.cloud.google.com/)
2. 进入 **APIs & Services** > **Credentials**
3. 创建或编辑 **OAuth 2.0 Client ID**（类型：iOS）
4. 配置：
   - **Name**: Grapery iOS Client
   - **Bundle ID**: `com.rankquantity.voyager`
   - **Client ID**: `345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb.apps.googleusercontent.com`

### 2. 配置 Android 客户端

1. 进入 **APIs & Services** > **Credentials**
2. 创建或编辑 **OAuth 2.0 Client ID**（类型：Android）
3. 配置：
   - **Name**: Grapery Android Client
   - **Package name**: `com.rankquantity.pioneer`
   - **SHA-1 certificate fingerprint**: `01:C4:61:0E:33:E1:8A:5D:93:0E:E5:19:D8:A7:D5:77:60:EC:3C:74`
   - **Client ID**: `345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev.apps.googleusercontent.com`

### 3. 配置 Web/Server 客户端（重要）✅

1. 进入 **APIs & Services** > **Credentials**
2. 创建 **OAuth 2.0 Client ID**（类型：Web application）
3. 配置：
   - **Name**: Grapery Server Client
   - **Client ID**: `345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com` ✅
   - **Authorized JavaScript origins**: （可选，如果需要）
   - **Authorized redirect URIs**: （可选，如果需要）
4. ✅ **已创建**，需要配置到后端环境变量 `GOOGLE_CLIENT_ID`

## ⚠️ 重要说明

### 客户端 ID 的使用

1. **iOS/Android Client ID**：
   - 仅在 Google Cloud Console 中配置 OAuth 客户端时使用
   - 用于标识和授权对应的移动应用

2. **Server Client ID**：
   - 后端环境变量 `GOOGLE_CLIENT_ID` 必须设置为 Server Client ID
   - Android 和 iOS 应用在请求 ID Token 时都应使用 Server Client ID
   - 后端使用它来验证 ID Token 的 audience

### 为什么需要 Server Client ID？

- Google ID Token 包含 `aud`（audience）字段，必须与后端配置的 Client ID 匹配
- 如果使用 iOS/Android Client ID，后端验证会失败（audience 不匹配）
- Server Client ID 是通用的，可以被所有平台使用

## 📝 配置验证

### 后端验证

```bash
# 检查环境变量
echo $GOOGLE_CLIENT_ID

# 应该显示 Web/Server Client ID，而不是 iOS 或 Android Client ID
```

### API 验证

```bash
# 测试 Google OAuth 配置 API
curl https://your-domain.com/api/vippay/google-oauth/config

# 返回的 clientId 应该是 Server Client ID
```

## 🔗 相关文档

- [OAuth 配置要求](./oauth_config_requirements.md)
- [Android SHA-1 指纹生成指南](../../pioneer/SHA1_FINGERPRINT_GUIDE.md)

