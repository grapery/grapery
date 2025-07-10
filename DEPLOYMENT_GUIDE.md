# Grapery 部署指南

## 概述

本项目使用 Docker Compose 进行容器化部署，通过 Nginx 反向代理提供 HTTPS 访问。

## 架构

```
Internet -> Nginx (80/443) -> Grapes App (8080) + LLM Chat (8070)
```

### 服务说明

- **grapes**: 主应用服务，运行在 8080 端口
- **grapes-llmchat**: LLM 聊天服务，运行在 8070 端口  
- **nginx**: 反向代理服务，提供 HTTPS 访问

## SSL 证书配置

项目使用自定义 SSL 证书：
- 证书文件：`certs/rankquantity.xyz.pem`
- 私钥文件：`certs/rankquantity.xyz.key`
- 域名：`api.rankquantity.xyz`

## 访问地址

- 主应用：`https://api.rankquantity.xyz`
- LLM Chat：`https://api.rankquantity.xyz/llmchat/`
- 健康检查：`https://api.rankquantity.xyz/health`

## 部署步骤

### 1. 环境准备

确保服务器已安装：
- Docker
- Docker Compose
- Git

### 2. 克隆代码

```bash
git clone <repository-url>
cd grapery
```

### 3. 配置环境变量

复制环境变量模板：
```bash
cp env.example .env
```

编辑 `.env` 文件，配置必要的环境变量。

### 4. 部署服务

```bash
# 构建并启动所有服务
docker-compose up -d --build

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 5. 验证部署

```bash
# 检查服务健康状态
curl https://api.rankquantity.xyz/health

# 检查 LLM Chat 服务
curl https://api.rankquantity.xyz/llmchat/health
```

## 自动化部署

项目配置了 GitHub Actions 自动化部署：

1. 推送代码到 `main`、`develop` 或 `feature/coze` 分支
2. GitHub Actions 自动构建 Docker 镜像
3. 推送镜像到阿里云容器镜像服务
4. 自动部署到 ECS 服务器

### 所需 GitHub Secrets

确保在 GitHub 仓库中配置以下 Secrets：

- `ACR_USERNAME`: 阿里云容器镜像服务用户名
- `ACR_PASSWORD`: 阿里云容器镜像服务密码
- `SSH_KEY`: 服务器 SSH 私钥
- `DB_PASSWORD`: 数据库密码

### 所需 GitHub Variables

确保在 GitHub 仓库中配置以下 Variables：

- `SSH_USER`: 服务器 SSH 用户名
- `DOMAIN_NAME`: 域名
- 其他环境变量（参考 `env.example`）

## 维护操作

### 重启服务

```bash
docker-compose restart
```

### 更新服务

```bash
# 拉取最新镜像
docker-compose pull

# 重启服务
docker-compose up -d
```

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f grapes
docker-compose logs -f grapes-llmchat
docker-compose logs -f nginx
```

### 备份数据

```bash
# 备份数据库（如果有）
docker exec grapery-mysql mysqldump -u root -p database_name > backup.sql
```

## 故障排除

### 常见问题

1. **SSL 证书问题**
   - 检查证书文件是否存在：`ls -la certs/`
   - 检查证书权限：`chmod 600 certs/rankquantity.xyz.key`

2. **服务无法启动**
   - 检查端口占用：`netstat -tlnp | grep :80`
   - 查看服务日志：`docker-compose logs`

3. **域名解析问题**
   - 确保域名 `api.rankquantity.xyz` 正确解析到服务器 IP
   - 检查防火墙设置

### 日志位置

- Nginx 日志：`docker-compose logs nginx`
- 应用日志：`docker-compose logs grapes`
- LLM Chat 日志：`docker-compose logs grapes-llmchat`

## 安全建议

1. 定期更新 SSL 证书
2. 使用强密码和密钥
3. 定期备份数据
4. 监控服务状态
5. 及时更新依赖包

## 联系信息

如有问题，请联系开发团队。 