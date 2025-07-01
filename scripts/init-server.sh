#!/bin/bash

# Grapery 服务器初始化脚本
# 在云主机上运行此脚本来设置部署环境

set -e

echo "🚀 开始初始化 Grapery 服务器..."

# 检查是否为root用户
if [ "$EUID" -eq 0 ]; then
    echo "❌ 请不要使用root用户运行此脚本"
    exit 1
fi

# 更新系统
echo "📦 更新系统包..."
sudo apt update && sudo apt upgrade -y

# 安装必要的软件包
echo "📦 安装必要的软件包..."
sudo apt install -y \
    curl \
    wget \
    git \
    docker.io \
    docker-compose \
    ufw \
    fail2ban \
    htop \
    nginx \
    certbot \
    python3-certbot-nginx \
    unzip \
    jq

# 启动并启用Docker服务
echo "🐳 配置Docker服务..."
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER

# 配置防火墙
echo "🔥 配置防火墙..."
sudo ufw --force enable
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8080/tcp
sudo ufw allow 8081/tcp
sudo ufw allow 8082/tcp
sudo ufw allow 3306/tcp
sudo ufw allow 6379/tcp

# 配置fail2ban
echo "🛡️ 配置fail2ban..."
sudo systemctl start fail2ban
sudo systemctl enable fail2ban

# 创建项目目录
echo "📁 创建项目目录..."
sudo mkdir -p /opt/grapery
sudo chown $USER:$USER /opt/grapery

# 创建日志目录
echo "📝 创建日志目录..."
sudo mkdir -p /var/log/grapery
sudo chown $USER:$USER /var/log/grapery

# 创建数据目录
echo "💾 创建数据目录..."
sudo mkdir -p /opt/grapery/data
sudo chown $USER:$USER /opt/grapery/data

# 创建SSL证书目录
echo "🔒 创建SSL证书目录..."
sudo mkdir -p /etc/letsencrypt
sudo chown -R $USER:$USER /etc/letsencrypt

# 配置Nginx（简化版本，主要配置在容器内）
echo "🌐 配置Nginx..."
sudo tee /etc/nginx/sites-available/grapery << 'EOF'
# 此配置将被容器内的nginx配置覆盖
server {
    listen 80;
    server_name _;
    return 444;
}
EOF

# 启用站点
sudo ln -sf /etc/nginx/sites-available/grapery /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl restart nginx

# 创建简化的部署脚本
echo "📜 创建部署脚本..."
cat > /opt/grapery/deploy.sh << 'EOF'
#!/bin/bash

# Grapery 部署脚本
set -e

echo "🚀 开始部署 Grapery..."

# 进入项目目录
cd /opt/grapery

# 拉取最新代码
git pull origin main

# 停止现有服务
docker-compose down || true

# 构建并启动服务
docker-compose up -d --build

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 检查服务状态
echo "📊 检查服务状态..."
docker-compose ps

# 检查服务健康状态
echo "🏥 检查服务健康状态..."
for service in grapes mcps vippay nginx; do
    if docker-compose ps $service | grep -q "Up"; then
        echo "✅ 服务 $service 运行正常"
    else
        echo "❌ 服务 $service 运行异常"
    fi
done

echo "🎉 部署完成！"
EOF

chmod +x /opt/grapery/deploy.sh

# 创建SSL证书管理脚本
echo "🔒 创建SSL证书管理脚本..."
cat > /opt/grapery/ssl-setup.sh << 'EOF'
#!/bin/bash

# SSL证书管理脚本
set -e

DOMAIN_NAME=${1:-""}
SSL_EMAIL=${2:-""}

if [ -z "$DOMAIN_NAME" ] || [ -z "$SSL_EMAIL" ]; then
    echo "用法: $0 <域名> <邮箱>"
    echo "示例: $0 grapery.com admin@grapery.com"
    exit 1
fi

echo "🔒 开始配置SSL证书..."

# 停止nginx容器
docker-compose stop nginx

# 申请SSL证书
sudo certbot certonly --standalone \
    -d $DOMAIN_NAME \
    -d api.$DOMAIN_NAME \
    -d mcp.$DOMAIN_NAME \
    -d pay.$DOMAIN_NAME \
    --email $SSL_EMAIL \
    --agree-tos \
    --non-interactive

# 设置证书权限
sudo chown -R $USER:$USER /etc/letsencrypt

# 重启nginx容器
docker-compose up -d nginx

echo "✅ SSL证书配置完成！"
EOF

chmod +x /opt/grapery/ssl-setup.sh

# 创建备份脚本
echo "💾 创建备份脚本..."
cat > /opt/grapery/backup.sh << 'EOF'
#!/bin/bash

# Grapery 备份脚本
set -e

BACKUP_DIR="/opt/grapery/backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

echo "💾 开始备份..."

# 备份MySQL数据
echo "📊 备份MySQL数据..."
docker exec grapery-mysql mysqldump -u root -p$MYSQL_ROOT_PASSWORD grapery > $BACKUP_DIR/mysql_backup.sql

# 备份Redis数据
echo "🔴 备份Redis数据..."
docker exec grapery-redis redis-cli BGSAVE
sleep 2
docker cp grapery-redis:/data/dump.rdb $BACKUP_DIR/redis_backup.rdb

# 备份配置文件
echo "⚙️ 备份配置文件..."
cp /opt/grapery/.env $BACKUP_DIR/env_backup
cp /opt/grapery/docker-compose.yml $BACKUP_DIR/docker-compose_backup.yml

# 备份SSL证书
echo "🔒 备份SSL证书..."
sudo cp -r /etc/letsencrypt $BACKUP_DIR/ssl_backup

echo "✅ 备份完成: $BACKUP_DIR"
EOF

chmod +x /opt/grapery/backup.sh

# 创建监控脚本
echo "📊 创建监控脚本..."
cat > /opt/grapery/monitor.sh << 'EOF'
#!/bin/bash

# Grapery 监控脚本
echo "📊 Grapery 服务状态监控"
echo "================================"

# 检查容器状态
echo "🐳 容器状态:"
docker-compose ps

echo ""
echo "💾 磁盘使用情况:"
df -h /opt/grapery

echo ""
echo "🧠 内存使用情况:"
free -h

echo ""
echo "🌐 网络连接:"
netstat -tlnp | grep -E ':(80|443|8080|8081|8082|3306|6379)'

echo ""
echo "📝 服务日志 (最近5行):"
for service in grapes mcps vippay nginx; do
    echo "--- $service ---"
    docker-compose logs --tail=5 $service
done

echo ""
echo "🔒 SSL证书状态:"
if [ -d "/etc/letsencrypt/live" ]; then
    for domain in $(ls /etc/letsencrypt/live/); do
        if [ -f "/etc/letsencrypt/live/$domain/fullchain.pem" ]; then
            echo "✅ $domain: 证书有效"
        else
            echo "❌ $domain: 证书无效"
        fi
    done
else
    echo "⚠️  SSL证书目录不存在"
fi
EOF

chmod +x /opt/grapery/monitor.sh

# 创建日志清理脚本
echo "🧹 创建日志清理脚本..."
cat > /opt/grapery/cleanup.sh << 'EOF'
#!/bin/bash

# 日志清理脚本
echo "🧹 开始清理日志..."

# 清理Docker日志
docker system prune -f

# 清理应用日志
find /var/log/grapery -name "*.log" -mtime +7 -delete

# 清理备份文件（保留最近7天）
find /opt/grapery/backups -type d -mtime +7 -exec rm -rf {} \;

echo "✅ 日志清理完成！"
EOF

chmod +x /opt/grapery/cleanup.sh

# 创建定时任务
echo "⏰ 创建定时任务..."
(crontab -l 2>/dev/null; echo "0 2 * * * /opt/grapery/backup.sh") | crontab -
(crontab -l 2>/dev/null; echo "0 3 * * * /opt/grapery/cleanup.sh") | crontab -
(crontab -l 2>/dev/null; echo "*/5 * * * * /opt/grapery/monitor.sh >> /var/log/grapery/monitor.log 2>&1") | crontab -

# 设置日志轮转
echo "📝 配置日志轮转..."
sudo tee /etc/logrotate.d/grapery << 'EOF'
/var/log/grapery/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 644 $USER $USER
}
EOF

# 创建快速部署脚本
echo "⚡ 创建快速部署脚本..."
cat > /opt/grapery/quick-deploy.sh << 'EOF'
#!/bin/bash

# 快速部署脚本
set -e

echo "⚡ 快速部署 Grapery..."

cd /opt/grapery

# 拉取代码并部署
git pull origin main
docker-compose up -d --build

echo "✅ 快速部署完成！"
EOF

chmod +x /opt/grapery/quick-deploy.sh

echo "✅ 服务器初始化完成！"
echo ""
echo "📋 下一步操作："
echo "1. 配置GitHub Secrets（包括DOMAIN_NAME和SSL_EMAIL）"
echo "2. 在GitHub仓库中设置SSH密钥"
echo "3. 推送代码到指定分支触发部署"
echo ""
echo "🔧 常用命令："
echo "  部署: /opt/grapery/deploy.sh"
echo "  快速部署: /opt/grapery/quick-deploy.sh"
echo "  SSL配置: /opt/grapery/ssl-setup.sh <域名> <邮箱>"
echo "  备份: /opt/grapery/backup.sh"
echo "  监控: /opt/grapery/monitor.sh"
echo "  清理: /opt/grapery/cleanup.sh"
echo ""
echo "🌐 服务访问地址："
echo "  主应用: https://api.your-domain.com"
echo "  MCP服务: https://mcp.your-domain.com"
echo "  支付服务: https://pay.your-domain.com"
echo ""
echo "⚠️  请重启服务器以应用Docker组权限更改"
echo "   执行: sudo reboot" 