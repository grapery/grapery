# Apple 和 Google OAuth 登录 - 资源配置清单

## 📌 快速检查清单

### ✅ 已完成
- [x] Google Cloud Console 配置（iOS、Android、Web/Server Client ID）
- [x] Apple Developer Portal 配置（App ID 和 Sign In with Apple capability）
- [x] 后端环境变量 `APPLE_BUNDLE_ID` 配置
- [x] 后端环境变量 `GOOGLE_CLIENT_ID` 配置（Web/Server Client ID）
- [x] **JWT_SECRET**：已设置强密钥 ✅
- [x] **数据库配置**：`DB_USERNAME`, `DB_PASSWORD`, `DB_ADDRESS`, `DB_DATABASE` ✅
- [x] **数据库表**：`third_party_logins` 等表已创建 ✅
- [x] **iOS 应用**：已添加 "Sign In with Apple" capability，已安装 Google Sign-In SDK ✅
- [x] **Android 应用**：已安装 Google Sign-In SDK，已生成 Release SHA-1 指纹 ✅
- [x] **网络访问**：服务器可以访问 Apple 和 Google 公钥端点 ✅

### ✅ 配置完成状态
所有必需的 OAuth2 配置项已完成！🎉

## 📋 实现状态

### ✅ 已实现功能
- ✅ Apple Sign-In Token 验证（JWT）
- ✅ Google Sign-In Token 验证（JWT）
- ✅ 公钥自动获取和缓存
- ✅ JWT Token 生成（Access Token + Refresh Token）
- ✅ API 端点完整
- ✅ **用户数据持久化**
- ✅ **第三方登录账户关联**
- ✅ **跨设备登录支持**（同一账户可在不同设备登录）
- ✅ **跨登录方式关联**（Google/Apple 使用相同 email 自动关联）

## 🔧 跨设备登录实现

### 账户关联策略

当用户通过 Google 或 Apple 登录时，系统会执行以下流程：

```
1. 通过 providerUserID 查找已绑定的第三方登录记录
   └── 找到 → 直接返回关联的用户（支持同设备再次登录）
   
2. 如果未找到，通过 email 查找是否有其他登录方式已绑定的用户
   └── 找到 → 创建新的第三方登录绑定，关联到现有用户
   └── 未找到 → 创建新用户 + 创建第三方登录绑定
```

### 数据模型

#### ThirdPartyLogin 表
```sql
CREATE TABLE third_party_logins (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,           -- 关联的用户ID
    provider VARCHAR(32) NOT NULL,          -- google, apple
    provider_user_id VARCHAR(255) NOT NULL, -- 第三方平台用户ID
    provider_email VARCHAR(255),            -- 第三方平台邮箱
    provider_user_name VARCHAR(255),        -- 第三方平台用户名
    provider_user_info TEXT,                -- 完整用户信息（JSON）
    status INT DEFAULT 1,                   -- 1=正常, 2=禁用
    created_at BIGINT,
    updated_at BIGINT,
    deleted_at BIGINT,
    
    UNIQUE INDEX idx_provider_user_id (provider, provider_user_id),
    INDEX idx_user_id (user_id),
    INDEX idx_provider_email (provider_email)
);
```

### 场景示例

**场景1：新用户首次登录**
- 用户在 iPhone 上使用 Apple Sign-In 登录
- 系统创建新用户 + Apple 第三方登录绑定

**场景2：同设备再次登录**
- 用户在同一 iPhone 上再次使用 Apple Sign-In
- 系统通过 Apple User ID 找到已有绑定，直接返回用户

**场景3：不同设备同方式登录**
- 用户在 iPad 上使用 Apple Sign-In（相同 Apple ID）
- 系统通过 Apple User ID 找到已有绑定，直接返回用户

**场景4：不同方式登录（账户关联）**
- 用户之前用 Apple 登录，现在用 Google 登录（相同邮箱）
- 系统通过邮箱找到已有用户，创建 Google 登录绑定
- 两种登录方式关联到同一用户

## 🔧 环境变量配置

### 必需配置

```bash
# Apple OAuth
APPLE_BUNDLE_ID=com.rankquantity.voyager

# Google OAuth (Web/Server Client ID)
GOOGLE_CLIENT_ID=345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com

# JWT 配置（生产环境必须设置强密钥）
JWT_SECRET=your-super-secret-jwt-key-change-in-production
# 生成方式：openssl rand -base64 32

# 数据库配置（用于用户持久化）
DB_USERNAME=root
DB_PASSWORD=your_password
DB_ADDRESS=localhost:3306
DB_DATABASE=grapery
```

### 可选配置

```bash
# JWT Token 过期时间（小时，默认24小时）
JWT_EXPIRY_HOURS=24

# 服务器端口（默认8081）
VIPPAY_PORT=8081

# 服务器域名（用于生成链接）
VIPPAY_DOMAIN=https://www.rankquantity.xyz

# 日志级别（debug, info, warn, error）
LOG_LEVEL=info
```

### ⚠️ 未使用的环境变量（可忽略）

以下环境变量在 Dockerfile 中定义，但**代码中未使用**，可以忽略：

```bash
# Apple OAuth（未使用，代码只使用 APPLE_BUNDLE_ID）
APPLE_OAUTH_CLIENT_ID=
APPLE_OAUTH_TEAM_ID=
APPLE_OAUTH_KEY_ID=
APPLE_OAUTH_PRIVATE_KEY=

# Google OAuth（未使用，代码只使用 GOOGLE_CLIENT_ID）
GOOGLE_OAUTH_CLIENT_SECRET=
```

**说明**：
- Apple OAuth 使用 JWT 公钥验证，不需要 Client ID、Team ID、Key ID、Private Key
- Google OAuth 使用 ID Token 验证，只需要 Client ID（Web/Server Client ID），不需要 Client Secret
- 这些环境变量可能是为未来的功能预留的，当前不需要配置

### Dockerfile 配置示例

```dockerfile
ENV APPLE_BUNDLE_ID=com.rankquantity.voyager
ENV GOOGLE_CLIENT_ID=345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com
ENV JWT_SECRET=your-super-secret-jwt-key
ENV DB_USERNAME=root
ENV DB_PASSWORD=password
ENV DB_ADDRESS=mysql
ENV DB_DATABASE=grapery
```

## 🔧 平台配置

### Apple Developer Portal

1. 创建 App ID（Bundle ID 必须与 iOS 应用一致）
2. 启用 "Sign In with Apple" capability
3. 配置 Associated Domains（如需要）

**配置位置：** https://developer.apple.com/account/resources/identifiers/list

### Google Cloud Console

1. 创建或选择项目
2. 启用 "Google Sign-In API"
3. 创建 OAuth 2.0 客户端 ID
   - **iOS**：创建 iOS 客户端，配置 Bundle ID（与 iOS 应用一致）
     - iOS Client ID：`345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb.apps.googleusercontent.com`
     - Bundle ID：`com.rankquantity.voyager`（需要与 iOS 应用一致）
     - 用于在 Google Cloud Console 中配置 iOS OAuth 客户端
   - **Android**：创建 Android 客户端，配置 packageName + SHA-1
     - Android Client ID：`345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev.apps.googleusercontent.com`
     - Package name：`com.rankquantity.pioneer`
     - SHA-1 指纹：`01:C4:61:0E:33:E1:8A:5D:93:0E:E5:19:D8:A7:D5:77:60:EC:3C:74`（Debug）
     - 用于在 Google Cloud Console 中配置 Android OAuth 客户端
   - **Web**：创建 Web application 客户端（**Server Client ID**，用于后端校验 ID Token 的 audience）
     - Web/Server Client ID: `345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com`
     - 这是后端环境变量 `GOOGLE_CLIENT_ID` 应该设置的值
     - **重要**：Android 和 iOS 客户端在请求 ID Token 时都应该使用这个 Server Client ID
4. 复制 **Web application Client ID**（Server Client ID）用于后端配置

**配置位置：** https://console.cloud.google.com/

**已配置的客户端 ID：**
- iOS Client ID: `345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb.apps.googleusercontent.com`
- Android Client ID: `345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev.apps.googleusercontent.com`
- Web/Server Client ID: `345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com` ✅

#### 重要：GOOGLE_CLIENT_ID 该填哪个？

- **后端环境变量 `GOOGLE_CLIENT_ID`** 必须设置为 **Web application Client ID（Server Client ID）**
  - 原因：后端在 `internal/service/pay/google_oauth.go` 用 `audience = GOOGLE_CLIENT_ID` 校验 Google ID Token
  - Android 端使用 `requestIdToken(serverClientId)`，会让返回的 ID Token 的 `aud` 等于这个 Web Client ID
  - iOS 端也需要使用 Server Client ID 来请求 ID Token
- 如果误填 Android Client ID / iOS Client ID，后端会报 `invalid_token`（audience 不匹配）

#### 重要：Bundle ID 和 Package name 不一样是正常的

- **iOS Bundle ID**: `com.rankquantity.voyager`（iOS 应用标识）
- **Android Package name**: `com.rankquantity.pioneer`（Android 应用标识）
- **这是完全正常的**，因为它们是两个不同的应用（iOS 和 Android）
- **后端 OAuth 验证不检查 Bundle ID 或 Package name**，只验证 ID Token 的 `audience`（Client ID）
- Bundle ID 和 Package name 仅在 Google Cloud Console 中配置 OAuth 客户端时使用，用于标识和授权对应的应用

#### iOS 客户端说明（voyager）

- **iOS Client ID**: `345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb.apps.googleusercontent.com`
- **用途**：
  - 在 Google Cloud Console 中配置 iOS OAuth 客户端
  - 在 iOS 应用中配置 Google Sign-In SDK（如果需要）
- **重要**：实际登录时，iOS 应用应该使用 **Server Client ID**（从后端 API 获取）来请求 ID Token
  - iOS 应用调用 `GET /api/vippay/google-oauth/config` 获取 `clientId`
  - 使用返回的 `clientId`（Server Client ID）来请求 Google ID Token
  - 这样后端才能正确验证 ID Token 的 audience

#### Android 客户端说明（pioneer）

- **Android Client ID**: `345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev.apps.googleusercontent.com`
- **Package name**: `com.rankquantity.pioneer`
- **SHA-1 指纹**:
  - Debug: `01:C4:61:0E:33:E1:8A:5D:93:0E:E5:19:D8:A7:D5:77:60:EC:3C:74`
  - Release: （需要从 Release keystore 生成）
- **配置方式**：
  - Android 应用不会硬编码 clientId，而是从后端获取：`GET /api/vippay/google-oauth/config`（字段 `clientId`）
  - 使用返回的 `clientId`（Server Client ID）来请求 Google ID Token
- **Google Cloud Console 配置**：
  - 需要在 Google Cloud Console 中创建 Android OAuth 客户端
  - 配置 package name: `com.rankquantity.pioneer`
  - 添加 SHA-1 指纹（Debug 和 Release）

## 📝 配置检查清单

### 后端配置（必需）

#### 环境变量配置
- [x] `APPLE_BUNDLE_ID` 环境变量已设置
  - 值：`com.rankquantity.voyager`
  - 用途：验证 Apple ID Token 的 audience
  - 配置位置：Dockerfile、docker-compose.yml、环境变量
- [x] `GOOGLE_CLIENT_ID` 环境变量已设置
  - 值：`345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com`（Web/Server Client ID）
  - 用途：验证 Google ID Token 的 audience
  - 配置位置：Dockerfile、docker-compose.yml、环境变量
- [x] `JWT_SECRET` 环境变量已设置（**生产环境必须使用强密钥**）✅
  - 要求：至少32字符的随机字符串
  - 用途：签名和验证 Access Token 和 Refresh Token
  - 配置位置：Dockerfile、docker-compose.yml、环境变量（**不要提交到代码仓库**）
  - ✅ **已配置强密钥**

#### 数据库配置（已完成 ✅）
- [x] `DB_USERNAME` 环境变量已设置 ✅
- [x] `DB_PASSWORD` 环境变量已设置 ✅
- [x] `DB_ADDRESS` 环境变量已设置（例如：`localhost:3306` 或 `mysql:3306`）✅
- [x] `DB_DATABASE` 环境变量已设置（值：`grapery`）✅
- [x] 数据库连接正常（可以连接到 MySQL）✅
- [x] 数据库表已创建（通过迁移或手动创建）✅
  - `users` 表 ✅
  - `user_settings` 表 ✅
  - `memberships` 表 ✅
  - `third_party_logins` 表（**必需**，用于存储第三方登录绑定）✅

#### 代码配置
- [x] 代码中已调用 `auth.SetJWTSecret()`（在 `main.go:145`）
- [x] OAuthRepository 已注入到 OAuth Handlers（在 `main.go:309-325`）

### Apple 配置

#### Apple Developer Portal（已完成 ✅）
- [x] Apple Developer Portal 中已创建 App ID
  - Bundle ID: `com.rankquantity.voyager`
  - 配置位置：https://developer.apple.com/account/resources/identifiers/list
- [x] 已启用 "Sign In with Apple" capability
  - 在 App ID 配置中启用
- [x] Bundle ID 与 iOS 应用一致
  - iOS 应用 Bundle ID: `com.rankquantity.voyager`
  - 后端 `APPLE_BUNDLE_ID` 环境变量: `com.rankquantity.voyager`
  - ✅ 已匹配

#### iOS 应用配置（已完成 ✅）
- [x] iOS 应用已添加 "Sign In with Apple" capability ✅
  - 在 Xcode 项目的 Signing & Capabilities 中添加
- [x] iOS 应用已配置从后端获取 Server Client ID ✅
  - 调用 `GET /api/vippay/google-oauth/config` 获取 `clientId`
  - 使用返回的 `clientId`（Server Client ID）请求 Google ID Token
- [x] iOS 应用已安装 Google Sign-In SDK（如果需要 Google 登录）✅
  - 通过 CocoaPods 或 SPM 安装

### Google 配置

#### Google Cloud Console（已完成）
- [x] Google Cloud Console 中已创建项目
- [x] 已启用 "Google Sign-In API"
- [x] 已创建 OAuth 2.0 客户端 ID（iOS）
  - iOS Client ID: `345805164843-68u1r8mhm4j6ke1of1ace43qh7cit1qb.apps.googleusercontent.com`
  - Bundle ID: `com.rankquantity.voyager`
- [x] 已创建 OAuth 2.0 客户端 ID（Android）
  - Android Client ID: `345805164843-jr79m749gvs4oi0fhb2q0refs3e0q0ev.apps.googleusercontent.com`
  - Package name: `com.rankquantity.pioneer`
  - SHA-1 Debug: `01:C4:61:0E:33:E1:8A:5D:93:0E:E5:19:D8:A7:D5:77:60:EC:3C:74`
  - SHA-1 Release: （需要从 Release keystore 生成并添加到 Google Cloud Console）
- [x] 已创建 OAuth 2.0 客户端 ID（Web/Server）
  - Web/Server Client ID: `345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com`
  - ✅ 已配置到后端环境变量 `GOOGLE_CLIENT_ID`

#### iOS 应用配置（已完成 ✅）
- [x] iOS 应用已配置使用 Server Client ID（从后端 API 获取）✅
  - 调用 `GET /api/vippay/google-oauth/config` 获取 `clientId`
  - 使用返回的 `clientId`（Server Client ID）请求 Google ID Token
- [x] iOS 应用已安装 Google Sign-In SDK ✅
  - 通过 CocoaPods 或 SPM 安装 `GoogleSignIn`

#### Android 应用配置（已完成 ✅）
- [x] Android 应用已配置使用 Server Client ID（从后端 API 获取）✅
  - 调用 `GET /api/vippay/google-oauth/config` 获取 `clientId`
  - 使用返回的 `clientId`（Server Client ID）请求 Google ID Token
- [x] Android 应用已安装 Google Sign-In SDK ✅
  - 在 `build.gradle.kts` 中添加依赖：`implementation("com.google.android.gms:play-services-auth:20.x.x")`
- [x] Android Release keystore SHA-1 指纹已生成并添加到 Google Cloud Console ✅
  - 使用 `get_sha1_fingerprint.sh` 脚本生成 Release SHA-1
  - 添加到 Google Cloud Console 的 Android OAuth 客户端配置中

### 网络配置（已完成 ✅）
- [x] 服务器可以访问 `https://appleid.apple.com/auth/keys` ✅
  - 用于获取 Apple 公钥集（验证 Apple ID Token）
  - 网络连接正常
- [x] 服务器可以访问 `https://www.googleapis.com/oauth2/v3/certs` ✅
  - 用于获取 Google 公钥集（验证 Google ID Token）
  - 网络连接正常

### Docker 配置（如果使用 Docker）

#### Dockerfile 环境变量
- [x] `APPLE_BUNDLE_ID` 已定义
- [x] `GOOGLE_CLIENT_ID` 已定义
- [x] `JWT_SECRET` 已定义（需要在运行时设置）
- [x] `DB_USERNAME`, `DB_PASSWORD`, `DB_ADDRESS`, `DB_DATABASE` 已定义

#### docker-compose.yml 配置
- [ ] 环境变量已正确设置（不要使用默认值）
- [ ] 数据库服务已配置并运行
- [ ] 网络配置正确（vippay 服务可以访问数据库和外部 API）

### 测试配置（推荐）

#### 后端 API 测试
- [ ] `GET /api/vippay/apple-oauth/config` 返回正确的配置
  - 应返回 `bundleId: "com.rankquantity.voyager"`
- [ ] `GET /api/vippay/google-oauth/config` 返回正确的配置
  - 应返回 `clientId: "345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com"`
- [ ] `GET /api/vippay/apple-oauth/status` 返回 `enabled: true`
- [ ] `GET /api/vippay/google-oauth/status` 返回 `enabled: true`

#### 端到端测试
- [ ] iOS 应用可以成功调用 Apple Sign-In
- [ ] iOS 应用可以成功调用 Google Sign-In
- [ ] Android 应用可以成功调用 Google Sign-In
- [ ] 登录后可以获取 Access Token 和 Refresh Token
- [ ] 使用 Access Token 可以访问受保护的 API

## ⚠️ 缺失配置检查

### 当前状态总结

#### ✅ 已完成的配置
1. **Google Cloud Console**
   - ✅ iOS Client ID 已创建
   - ✅ Android Client ID 已创建
   - ✅ Web/Server Client ID 已创建并配置到后端

2. **后端环境变量**
   - ✅ `APPLE_BUNDLE_ID` 已配置
   - ✅ `GOOGLE_CLIENT_ID` 已配置（Web/Server Client ID）

#### ✅ 所有配置已完成

1. **JWT Secret（已完成 ✅）**
   - ✅ **已设置强密钥**
   - ✅ `JWT_SECRET` 环境变量已配置为至少32字符的随机字符串

2. **数据库配置（已完成 ✅）**
   - ✅ 数据库连接信息已设置
   - ✅ 环境变量：`DB_USERNAME`, `DB_PASSWORD`, `DB_ADDRESS`, `DB_DATABASE` 已配置
   - ✅ 数据库表已创建（包括 `third_party_logins` 表）

3. **Apple Developer Portal（已完成 ✅）**
   - ✅ 已在 Apple Developer Portal 中配置 App ID
   - ✅ 已启用 "Sign In with Apple" capability
   - ✅ Bundle ID 与 iOS 应用一致

4. **iOS 应用配置（已完成 ✅）**
   - ✅ 已在 Xcode 中添加 "Sign In with Apple" capability
   - ✅ 已安装 Google Sign-In SDK
   - ✅ 已配置从后端获取 Server Client ID

5. **Android 应用配置（已完成 ✅）**
   - ✅ 已安装 Google Sign-In SDK
   - ✅ 已配置从后端获取 Server Client ID
   - ✅ 已生成 Release keystore SHA-1 指纹并添加到 Google Cloud Console

6. **网络访问（已完成 ✅）**
   - ✅ 服务器可以访问 Apple 和 Google 的公钥端点
   - ✅ 网络连接测试通过：
     ```bash
     curl https://appleid.apple.com/auth/keys  # ✅ 可访问
     curl https://www.googleapis.com/oauth2/v3/certs  # ✅ 可访问
     ```

### 快速配置检查命令

```bash
# 检查环境变量是否设置
echo "APPLE_BUNDLE_ID: ${APPLE_BUNDLE_ID:-未设置}"
echo "GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID:-未设置}"
echo "JWT_SECRET: ${JWT_SECRET:+已设置（隐藏）}${JWT_SECRET:-未设置}"
echo "DB_USERNAME: ${DB_USERNAME:-未设置}"
echo "DB_ADDRESS: ${DB_ADDRESS:-未设置}"

# 检查网络访问
curl -s -o /dev/null -w "%{http_code}" https://appleid.apple.com/auth/keys
curl -s -o /dev/null -w "%{http_code}" https://www.googleapis.com/oauth2/v3/certs

# 检查后端 API 配置端点
curl http://localhost:8081/api/vippay/apple-oauth/config
curl http://localhost:8081/api/vippay/google-oauth/config
```

## 🔒 安全注意事项

1. **JWT Secret**
   - 生产环境必须使用强随机密钥（至少32字符）
   - 不要将密钥提交到代码仓库
   - 使用密钥管理服务（AWS Secrets Manager、Azure Key Vault 等）

2. **Token 验证**
   - ✅ 所有验证在服务器端进行
   - ✅ 使用 HTTPS 传输（需要配置）
   - ✅ Token 过期时间合理（24小时）

3. **账户安全**
   - 通过 providerUserID 确保同一第三方账号不会绑定到多个用户
   - 通过 email 关联确保用户不会因切换登录方式而丢失数据

## 📚 API 端点

### Apple OAuth
- `POST /api/vippay/apple-oauth/signin` - 登录
- `GET /api/vippay/apple-oauth/config` - 获取配置
- `GET /api/vippay/apple-oauth/status` - 检查状态

### Google OAuth
- `POST /api/vippay/google-oauth/signin` - 登录
- `GET /api/vippay/google-oauth/config` - 获取配置
- `GET /api/vippay/google-oauth/status` - 检查状态

## 🚀 快速开始

### 完整配置

```bash
# 1. 生成 JWT Secret（生产环境必须）
export JWT_SECRET=$(openssl rand -base64 32)

# 2. 设置 OAuth 配置
export APPLE_BUNDLE_ID=com.rankquantity.voyager
export GOOGLE_CLIENT_ID=345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com

# 3. 设置数据库配置
export DB_USERNAME=root
export DB_PASSWORD=your_password
export DB_ADDRESS=localhost:3306
export DB_DATABASE=grapery

# 4. 启动服务
go run cmd/vippay/main.go
```

### 环境变量优先级

配置加载优先级（从高到低）：
1. **环境变量**（最高优先级）
2. `vippay.json` 配置文件
3. 代码默认值（仅用于开发）

### 配置验证

启动服务后，检查配置是否正确：

```bash
# 检查 Apple OAuth 配置
curl http://localhost:8081/api/vippay/apple-oauth/config
# 应返回：{"code":0,"data":{"bundleId":"com.rankquantity.voyager",...}}

# 检查 Google OAuth 配置
curl http://localhost:8081/api/vippay/google-oauth/config
# 应返回：{"code":0,"data":{"clientId":"345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com",...}}

# 检查 OAuth 状态
curl http://localhost:8081/api/vippay/apple-oauth/status
curl http://localhost:8081/api/vippay/google-oauth/status
# 应返回：{"code":0,"data":{"enabled":true,...}}
```

### Docker 启动

```bash
# 1. 构建镜像
docker build -t vippay -f cmd/vippay/Dockerfile .

# 2. 运行容器（使用环境变量）
docker run -d \
  --name vippay \
  -e APPLE_BUNDLE_ID=com.rankquantity.voyager \
  -e GOOGLE_CLIENT_ID=345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com \
  -e JWT_SECRET=$(openssl rand -base64 32) \
  -e DB_USERNAME=root \
  -e DB_PASSWORD=your_password \
  -e DB_ADDRESS=mysql:3306 \
  -e DB_DATABASE=grapery \
  -p 8081:8081 \
  vippay
```

### Docker Compose 启动（推荐）

如果使用 `docker-compose.vippay.yml`，确保环境变量已正确设置：

```yaml
services:
  vippay:
    environment:
      APPLE_BUNDLE_ID: com.rankquantity.voyager
      GOOGLE_CLIENT_ID: 345805164843-pbd5oc8emnu03l1i0sdn7r19pmk10ajf.apps.googleusercontent.com
      JWT_SECRET: ${JWT_SECRET}  # 从 .env 文件读取
      DB_USERNAME: root
      DB_PASSWORD: ${DB_PASSWORD}
      DB_ADDRESS: mysql:3306
      DB_DATABASE: grapery
```

**注意**：`JWT_SECRET` 应该从 `.env` 文件或密钥管理服务读取，不要硬编码在配置文件中。

## 📦 代码结构

```
grapery/
├── cmd/vippay/main.go                          # 服务入口，注入 OAuthRepository
├── internal/
│   ├── domain/
│   │   ├── third_party_models.go              # ThirdPartyLogin 领域模型
│   │   └── repository.go                       # Repository 接口定义
│   ├── repository/
│   │   ├── mysql/
│   │   │   ├── models.go                       # 数据库模型（包含 ThirdPartyLogin）
│   │   │   └── third_party_login_impl.go       # ThirdPartyLogin MySQL 实现
│   │   └── pay/
│   │       └── oauth_repository.go             # vippay 专用 OAuthRepository
│   └── transport/pay/
│       ├── apple_oauth_handler.go              # Apple OAuth Handler
│       └── google_oauth_handler.go             # Google OAuth Handler
```
