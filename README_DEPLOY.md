# Grapery 容器部署指南

本文档详细说明如何使用 Docker Compose 直接部署 Grapery 支付服务系统。

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Nginx (80)    │    │   Nginx (443)   │    │   MySQL (3306)  │
│   反向代理       │    │   HTTPS代理     │    │   数据库        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Grapes (8080)  │    │   MCPs (8081)   │    │ Vippay (8082)   │
│   主应用服务     │    │   MCP服务       │    │   VIP支付服务   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Redis (6379)   │
                    │   缓存服务      │
                    └─────────────────┘
```

## 前置要求

### 1. 系统要求
- Linux 服务器（推荐 Ubuntu 20.04+ 或 CentOS 8+）
- Docker 20.10+
- Docker Compose 2.0+
- 至少 4GB RAM
- 至少 20GB 磁盘空间

### 2. 安装 Docker 和 Docker Compose

#### Ubuntu/Debian:
```bash
# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

#### CentOS/RHEL:
```bash
# 安装 Docker
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install docker-ce docker-ce-cli containerd.io
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

## 快速部署

### 1. 克隆项目
```bash
git clone https://github.com/your-username/grapery.git
cd grapery
```

### 2. 配置环境变量
```bash
# 复制环境变量示例文件
cp env.example .env

# 编辑环境变量文件
nano .env
```

**重要配置项：**
- 数据库密码
- JWT密钥
- 支付服务商密钥
- 第三方API密钥

### 3. 启动服务
```bash
# 给部署脚本执行权限
chmod +x deploy.sh

# 启动所有服务
./deploy.sh start
```

### 4. 检查服务状态
```bash
# 查看所有服务状态
./deploy.sh status

# 查看服务日志
./deploy.sh logs
```

## 详细部署步骤

### 1. 环境准备

#### 创建必要目录
```bash
mkdir -p ssl logs backup
```

#### 配置SSL证书（可选）
```bash
# 将SSL证书文件放入ssl目录
cp your-domain.crt ssl/
cp your-domain.key ssl/
```

### 2. 数据库初始化

系统会自动创建数据库和表结构，包括：
- 用户表 (users)
- 商品表 (products)
- 订单表 (orders)
- 支付记录表 (payment_records)
- 订阅表 (subscriptions)
- VIP用户表 (vip_users)

### 3. 服务配置

#### 主应用服务 (Grapes)
- 端口：8080
- 功能：用户管理、内容管理、API接口

#### MCP服务 (MCPs)
- 端口：8081
- 功能：模型上下文协议服务

#### VIP支付服务 (Vippay)
- 端口：8082
- 功能：支付处理、订阅管理

#### 数据库服务 (MySQL)
- 端口：3306
- 数据持久化存储

#### 缓存服务 (Redis)
- 端口：6379
- 会话和缓存存储

#### 反向代理 (Nginx)
- 端口：80/443
- 负载均衡和SSL终止

## 服务管理

### 常用命令

```bash
# 启动所有服务
./deploy.sh start

# 停止所有服务
./deploy.sh stop

# 重启所有服务
./deploy.sh restart

# 查看服务状态
./deploy.sh status

# 查看所有日志
./deploy.sh logs

# 查看特定服务日志
./deploy.sh logs grapes
./deploy.sh logs vippay

# 构建镜像
./deploy.sh build

# 清理资源
./deploy.sh cleanup

# 备份数据
./deploy.sh backup

# 恢复数据
./deploy.sh restore backup/20231201_120000
```

### 手动Docker Compose命令

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 查看日志
docker-compose logs -f

# 重启特定服务
docker-compose restart grapes

# 进入容器
docker-compose exec grapes bash
docker-compose exec mysql mysql -u root -p
```

## 监控和维护

### 1. 日志监控
```bash
# 实时查看日志
./deploy.sh logs

# 查看特定服务日志
./deploy.sh logs vippay

# 查看错误日志
docker-compose logs --tail=100 | grep ERROR
```

### 2. 性能监控
```bash
# 查看容器资源使用
docker stats

# 查看磁盘使用
df -h

# 查看内存使用
free -h
```

### 3. 数据备份
```bash
# 自动备份
./deploy.sh backup

# 手动备份MySQL
docker exec grapery-mysql mysqldump -u root -proot123 grapery > backup.sql

# 手动备份Redis
docker exec grapery-redis redis-cli BGSAVE
docker cp grapery-redis:/data/dump.rdb ./redis_backup.rdb
```

### 4. 数据恢复
```bash
# 恢复备份
./deploy.sh restore backup/20231201_120000

# 手动恢复MySQL
docker exec -i grapery-mysql mysql -u root -proot123 grapery < backup.sql
```

## 故障排除

### 常见问题

#### 1. 容器启动失败
```bash
# 查看容器状态
docker ps -a

# 查看容器日志
docker logs grapery-grapes

# 检查端口占用
netstat -tlnp | grep :8080
```

#### 2. 数据库连接失败
```bash
# 检查MySQL容器状态
docker ps | grep mysql

# 检查数据库连接
docker exec grapery-mysql mysql -u grapery -pgrapery123 -e "SHOW DATABASES;"
```

#### 3. 支付服务异常
```bash
# 检查支付服务日志
./deploy.sh logs vippay

# 检查支付配置
docker exec grapery-vippay env | grep PAYMENT
```

#### 4. 网络连接问题
```bash
# 检查网络
docker network ls
docker network inspect grapery_grapery-network

# 测试容器间通信
docker exec grapery-grapes ping mysql
```

### 性能优化

#### 1. 资源限制
在 `docker-compose.yml` 中添加资源限制：
```yaml
services:
  grapes:
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '0.5'
        reservations:
          memory: 512M
          cpus: '0.25'
```

#### 2. 数据库优化
```sql
-- 优化MySQL配置
SET GLOBAL innodb_buffer_pool_size = 1073741824; -- 1GB
SET GLOBAL max_connections = 200;
```

#### 3. Redis优化
```bash
# 在redis.conf中添加
maxmemory 512mb
maxmemory-policy allkeys-lru
```

## 安全配置

### 1. 防火墙设置
```bash
# 只开放必要端口
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

### 2. SSL证书配置
```bash
# 使用Let's Encrypt获取免费证书
sudo apt install certbot
sudo certbot certonly --standalone -d your-domain.com

# 配置Nginx SSL
cp /etc/letsencrypt/live/your-domain.com/fullchain.pem ssl/
cp /etc/letsencrypt/live/your-domain.com/privkey.pem ssl/
```

### 3. 环境变量安全
```bash
# 使用强密码
JWT_SECRET=your-very-long-and-random-jwt-secret-key
DB_PASSWORD=your-strong-database-password

# 定期轮换密钥
# 建议每月更换一次JWT密钥
```

## 扩展部署

### 1. 多实例部署
```yaml
# 在docker-compose.yml中添加
services:
  grapes:
    deploy:
      replicas: 3
    environment:
      - INSTANCE_ID=${HOSTNAME}
```

### 2. 负载均衡
```nginx
# 在nginx.conf中添加
upstream grapes_backend {
    server grapes:8080 weight=1;
    server grapes2:8080 weight=1;
    server grapes3:8080 weight=1;
}
```

### 3. 高可用部署
- 使用外部数据库集群
- 使用Redis集群
- 配置自动故障转移

## 联系支持

如果遇到部署问题，请：
1. 查看日志文件
2. 检查环境配置
3. 提交Issue到GitHub
4. 联系技术支持团队

---

**注意：** 生产环境部署前请务必：
- 修改所有默认密码
- 配置SSL证书
- 设置防火墙规则
- 配置监控告警
- 制定备份策略 