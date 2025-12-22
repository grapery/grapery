# Apple 和 Google OAuth 登录 - 资源配置清单

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
APPLE_BUNDLE_ID=com.grapery.app

# Google OAuth
GOOGLE_CLIENT_ID=YOUR_CLIENT_ID.apps.googleusercontent.com

# JWT 配置
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# 数据库配置（用于用户持久化）
DB_USERNAME=root
DB_PASSWORD=your_password
DB_ADDRESS=localhost
DB_DATABASE=grapery
```

### Dockerfile 配置示例

```dockerfile
ENV APPLE_BUNDLE_ID=com.grapery.app
ENV GOOGLE_CLIENT_ID=YOUR_CLIENT_ID.apps.googleusercontent.com
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
3. 创建 OAuth 2.0 客户端 ID（iOS 类型）
4. 配置 Bundle ID（与 iOS 应用一致）
5. 复制 Client ID

**配置位置：** https://console.cloud.google.com/

## 📝 配置检查清单

### 后端配置
- [x] `APPLE_BUNDLE_ID` 环境变量已设置
- [x] `GOOGLE_CLIENT_ID` 环境变量已设置
- [x] `JWT_SECRET` 环境变量已设置（生产环境使用强密钥）
- [x] 代码中已调用 `auth.SetJWTSecret()`
- [x] 数据库连接配置
- [x] OAuthRepository 已注入到 OAuth Handlers

### Apple 配置
- [ ] Apple Developer Portal 中已创建 App ID
- [ ] 已启用 "Sign In with Apple" capability
- [ ] Bundle ID 与 iOS 应用一致
- [ ] iOS 应用已添加 "Sign In with Apple" capability

### Google 配置
- [ ] Google Cloud Console 中已创建项目
- [ ] 已启用 "Google Sign-In API"
- [ ] 已创建 OAuth 2.0 客户端 ID（iOS）
- [ ] Client ID 已配置到 iOS 应用
- [ ] Google Sign-In SDK 已安装

### 网络配置
- [ ] 服务器可以访问 `https://appleid.apple.com/auth/keys`
- [ ] 服务器可以访问 `https://www.googleapis.com/oauth2/v3/certs`

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
# 设置环境变量
export APPLE_BUNDLE_ID=com.grapery.app
export GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
export JWT_SECRET=your-super-secret-key-at-least-32-chars
export DB_USERNAME=root
export DB_PASSWORD=password
export DB_ADDRESS=localhost
export DB_DATABASE=grapery

# 启动服务
go run cmd/vippay/main.go
```

### Docker 启动

```bash
docker build -t vippay -f cmd/vippay/Dockerfile .
docker run -d \
  -e APPLE_BUNDLE_ID=com.grapery.app \
  -e GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com \
  -e JWT_SECRET=your-super-secret-key \
  -e DB_USERNAME=root \
  -e DB_PASSWORD=password \
  -e DB_ADDRESS=mysql \
  -e DB_DATABASE=grapery \
  -p 8081:8081 \
  vippay
```

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
