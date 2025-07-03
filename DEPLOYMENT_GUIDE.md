# Grapery 部署指南

## 概述

本指南介绍如何将 Grapery 项目部署到云主机，支持多分支自动部署、HTTPS、SSL证书自动配置和容器化部署。

## 系统要求

- Ubuntu 20.04+ 或 CentOS 8+
- 2GB+ RAM
- 20GB+ 磁盘空间
- 域名（用于HTTPS）
- 云主机（阿里云、腾讯云等）

## 快速部署

### 1. 服务器初始化

在云主机上运行初始化脚本：

```bash
# 下载并运行初始化脚本
curl -fsSL https://raw.githubusercontent.com/your-repo/grapery/main/scripts/init-server.sh | bash

# 重启服务器以应用Docker组权限
sudo reboot
```

### 2. 配置GitHub Secrets

在GitHub仓库设置中添加以下Secrets：

#### 基础配置
- `SERVER_HOST`: 云主机IP地址
- `SERVER_USER`: SSH用户名
- `SERVER_PORT`: SSH端口（默认22）
- `SSH_PRIVATE_KEY`: SSH私钥

#### 数据库配置
- `DB_USER`: MySQL用户名
- `DB_PASSWORD`: MySQL密码
- `DB_NAME`: 数据库名
- `MYSQL_ROOT_PASSWORD`: MySQL root密码
- `REDIS_PASSWORD`: Redis密码

#### 应用配置
- `JWT_SECRET`: JWT密钥
- `CORS_ALLOWED_ORIGINS`: 允许的跨域域名

#### 支付配置
- `STRIPE_SECRET_KEY`: Stripe密钥
- `STRIPE_PUBLISHABLE_KEY`: Stripe公钥
- `STRIPE_WEBHOOK_SECRET`: Stripe Webhook密钥
- `ALIPAY_APP_ID`: 支付宝应用ID
- `ALIPAY_PRIVATE_KEY`: 支付宝私钥
- `ALIPAY_PUBLIC_KEY`: 支付宝公钥
- `ALIPAY_NOTIFY_URL`: 支付宝回调URL
- `ALIPAY_RETURN_URL`: 支付宝返回URL
- `WECHAT_APP_ID`: 微信应用ID
- `WECHAT_MCH_ID`: 微信商户ID
- `WECHAT_API_KEY`: 微信API密钥
- `WECHAT_NOTIFY_URL`: 微信回调URL

#### 第三方服务配置
- `OPENAI_API_KEY`: OpenAI API密钥
- `OPENAI_BASE_URL`: OpenAI API地址
- `ALIYUN_ACCESS_KEY_ID`: 阿里云AccessKey ID
- `ALIYUN_ACCESS_KEY_SECRET`: 阿里云AccessKey Secret
- `ALIYUN_REGION`: 阿里云区域
- `ALIYUN_OSS_BUCKET`: 阿里云OSS存储桶
- `ALIYUN_OSS_ENDPOINT`: 阿里云OSS端点
- `COZE_API_KEY`: Coze API密钥
- `COZE_BOT_ID`: Coze机器人ID
- `COZE_WORKSPACE_ID`: Coze工作空间ID
- `TENCENT_SECRET_ID`: 腾讯云Secret ID
- `TENCENT_SECRET_KEY`: 腾讯云Secret Key
- `TENCENT_REGION`: 腾讯云区域
- `ZHIPU_API_KEY`: 智谱AI API密钥

#### 邮件配置
- `SMTP_HOST`: SMTP服务器地址
- `SMTP_PORT`: SMTP端口
- `SMTP_USERNAME`: SMTP用户名
- `SMTP_PASSWORD`: SMTP密码

#### HTTPS配置
- `DOMAIN_NAME`: 主域名（如：grapery.com）
- `SSL_EMAIL`: SSL证书邮箱

### 3. 部署触发

#### 自动部署
推送代码到以下分支会自动触发部署：
- `main`: 生产环境
- `develop`: 开发环境
- `staging`: 测试环境

#### 手动部署
在GitHub Actions页面手动触发部署，可选择部署分支。

### 4. SSL证书配置

首次部署后，配置SSL证书：

```bash
# 在服务器上运行
cd /opt/grapery
./ssl-setup.sh your-domain.com your-email@domain.com
```

## 服务架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Nginx (80/443)│    │   MySQL (3306)  │    │   Redis (6379)  │
│   (反向代理)     │    │   (数据库)      │    │   (缓存)        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Grapes (8080) │    │   MCPS (8081)   │    │  VIPPay (8082)  │
│   (主应用)       │    │   (MCP服务)     │    │   (支付服务)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 服务访问地址

- **主应用**: `https://api.your-domain.com`
- **MCP服务**: `https://mcp.your-domain.com`
- **支付服务**: `https://pay.your-domain.com`
- **健康检查**: `https://api.your-domain.com/health`

## 管理命令

### 部署管理
```bash
# 完整部署
/opt/grapery/deploy.sh

# 快速部署
/opt/grapery/quick-deploy.sh

# 查看服务状态
docker-compose ps

# 查看服务日志
docker-compose logs -f [service_name]
```

### SSL证书管理
```bash
# 配置SSL证书
/opt/grapery/ssl-setup.sh <域名> <邮箱>

# 续期SSL证书
sudo certbot renew

# 查看证书状态
sudo certbot certificates
```

### 数据管理
```bash
# 备份数据
/opt/grapery/backup.sh

# 恢复数据
docker exec -i grapery-mysql mysql -u root -p$MYSQL_ROOT_PASSWORD grapery < backup.sql

# 清理日志
/opt/grapery/cleanup.sh
```

### 监控管理
```bash
# 查看服务监控
/opt/grapery/monitor.sh

# 查看系统资源
htop

# 查看网络连接
netstat -tlnp
```

## 安全配置

### 防火墙
- SSH: 22
- HTTP: 80
- HTTPS: 443
- 应用端口: 8080, 8081, 8082
- 数据库端口: 3306, 6379

### SSL/TLS
- 使用Let's Encrypt免费证书
- 自动续期
- 强制HTTPS重定向
- 安全头配置

### 访问控制
- fail2ban防暴力破解
- 日志轮转
- 定期备份
- 监控告警

## 故障排除

### 常见问题

1. **服务无法启动**
   ```bash
   # 检查日志
   docker-compose logs [service_name]
   
   # 检查配置
   docker-compose config
   ```

2. **SSL证书问题**
   ```bash
   # 检查证书状态
   sudo certbot certificates
   
   # 重新申请证书
   sudo certbot certonly --standalone -d your-domain.com
   ```

3. **数据库连接问题**
   ```bash
   # 检查MySQL状态
   docker exec grapery-mysql mysqladmin ping -u root -p
   
   # 检查Redis状态
   docker exec grapery-redis redis-cli ping
   ```

4. **网络连接问题**
   ```bash
   # 检查端口监听
   netstat -tlnp
   
   # 检查防火墙
   sudo ufw status
   ```

### 日志位置
- 应用日志: `/var/log/grapery/`
- Nginx日志: `/var/log/nginx/`
- Docker日志: `docker-compose logs`

## 性能优化

### 系统优化
- 启用Gzip压缩
- 配置缓存策略
- 优化数据库查询
- 使用CDN加速

### 监控指标
- CPU使用率
- 内存使用率
- 磁盘使用率
- 网络流量
- 响应时间

## 备份策略

### 自动备份
- 每日凌晨2点自动备份
- 保留最近7天备份
- 备份内容包括：
  - MySQL数据
  - Redis数据
  - 配置文件
  - SSL证书

### 手动备份
```bash
# 创建备份
/opt/grapery/backup.sh

# 恢复备份
# 1. 停止服务
docker-compose down

# 2. 恢复数据
docker exec -i grapery-mysql mysql -u root -p$MYSQL_ROOT_PASSWORD grapery < backup.sql

# 3. 重启服务
docker-compose up -d
```

## 更新升级

### 代码更新
```bash
# 拉取最新代码
git pull origin main

# 重新构建并启动
docker-compose up -d --build
```

### 配置更新
1. 修改GitHub Secrets
2. 推送代码触发重新部署
3. 或手动更新.env文件

## 联系支持

如遇到问题，请：
1. 查看日志文件
2. 检查服务状态
3. 参考故障排除部分
4. 提交Issue到GitHub仓库 