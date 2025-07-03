# Grapery 部署方案

## 🚀 快速部署

### 1. 服务器初始化
```bash
# 在云主机上运行
curl -fsSL https://raw.githubusercontent.com/your-repo/grapery/main/scripts/init-server.sh | bash
sudo reboot
```

### 2. 配置GitHub Secrets
在GitHub仓库设置中添加以下Secrets：

**基础配置**
- `SERVER_HOST`: 云主机IP
- `SERVER_USER`: SSH用户名  
- `SERVER_PORT`: SSH端口
- `SSH_PRIVATE_KEY`: SSH私钥

**数据库配置**
- `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `MYSQL_ROOT_PASSWORD`, `REDIS_PASSWORD`

**应用配置**
- `JWT_SECRET`, `CORS_ALLOWED_ORIGINS`

**支付配置**
- Stripe: `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET`
- 支付宝: `ALIPAY_APP_ID`, `ALIPAY_PRIVATE_KEY`, `ALIPAY_PUBLIC_KEY`
- 微信: `WECHAT_APP_ID`, `WECHAT_MCH_ID`, `WECHAT_API_KEY`

**第三方服务**
- OpenAI, 阿里云, Coze, 腾讯云, 智谱AI等API密钥

**HTTPS配置**
- `DOMAIN_NAME`: 主域名
- `SSL_EMAIL`: SSL证书邮箱

### 3. 自动部署
推送代码到以下分支自动触发部署：
- `main`: 生产环境
- `develop`: 开发环境  
- `staging`: 测试环境

### 4. SSL证书配置
```bash
# 首次部署后配置SSL
cd /opt/grapery
./ssl-setup.sh your-domain.com your-email@domain.com
```

## 🌐 服务访问

- **主应用**: `https://api.your-domain.com`
- **MCP服务**: `https://mcp.your-domain.com`
- **支付服务**: `https://pay.your-domain.com`
- **健康检查**: `https://api.your-domain.com/health`

## 🔧 管理命令

```bash
# 部署管理
/opt/grapery/deploy.sh start|stop|restart|status
/opt/grapery/quick-deploy.sh

# 日志查看
docker-compose logs -f [service_name]

# SSL证书
/opt/grapery/ssl-setup.sh <域名> <邮箱>
sudo certbot renew

# 数据管理
/opt/grapery/backup.sh
/opt/grapery/cleanup.sh

# 监控
/opt/grapery/monitor.sh
```

## 📋 特性

✅ **多分支自动部署** - 支持main/develop/staging分支  
✅ **HTTPS支持** - 自动SSL证书配置和续期  
✅ **安全配置** - 防火墙、fail2ban、安全头  
✅ **健康检查** - 容器和服务状态监控  
✅ **自动备份** - 每日数据备份和日志轮转  
✅ **简化操作** - 一键部署和管理脚本  
✅ **配置安全** - 通过GitHub Secrets传递敏感配置  

## 🏗️ 架构

```
GitHub Actions → 云主机 → Docker Compose → 服务容器
     ↓              ↓           ↓
  自动部署    →   SSL证书   →   Nginx反向代理
     ↓              ↓           ↓
  配置管理    →   健康检查   →   应用服务
```

## 📞 支持

- 查看详细部署指南: [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)
- 提交Issue: GitHub仓库
- 查看日志: `/var/log/grapery/` 