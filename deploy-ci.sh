#!/bin/bash

# CI/CD 部署脚本 - 支持选择性部署服务
# 用法: ./deploy-ci.sh "service1,service2,service3"
# 支持的服务: nginx, redis, grapes-app, grapes-pay, grapes-llmchat, grapes-asynctask

set -e

echo "🚀 开始 CI/CD 选择性部署..."

# 解析要部署的服务列表
DEPLOY_SERVICES="${1:-}"
if [ -z "$DEPLOY_SERVICES" ]; then
    echo "❌ 错误: 未指定要部署的服务"
    echo "💡 用法: ./deploy-ci.sh \"service1,service2\""
    echo "💡 支持的服务: nginx, redis, grapes-app, grapes-pay, grapes-llmchat, grapes-asynctask"
    exit 1
fi

echo "📋 要部署的服务: $DEPLOY_SERVICES"

# 标准化服务名称
DEPLOY_SERVICES=$(echo "$DEPLOY_SERVICES" | tr '[:upper:]' '[:lower:]' | tr ',' ' ')

# 初始化服务标志
DEPLOY_NGINX=false
DEPLOY_REDIS=false
DEPLOY_GRAPES_APP=false
DEPLOY_GRAPES_PAY=false
DEPLOY_GRAPES_LLMCHAT=false
DEPLOY_GRAPES_ASYNCTASK=false

# 解析服务列表
for service in $DEPLOY_SERVICES; do
    case "$service" in
        nginx)
            DEPLOY_NGINX=true
            ;;
        redis)
            DEPLOY_REDIS=true
            ;;
        grapes-app|grapes_app|app)
            DEPLOY_GRAPES_APP=true
            ;;
        grapes-pay|grapes_pay|vippay|pay)
            DEPLOY_GRAPES_PAY=true
            ;;
        grapes-llmchat|grapes_llmchat|llmchat)
            DEPLOY_GRAPES_LLMCHAT=true
            ;;
        grapes-asynctask|grapes_asynctask|asynctask)
            DEPLOY_GRAPES_ASYNCTASK=true
            ;;
        *)
            echo "⚠️  警告: 未知的服务名称 '$service'，跳过"
            ;;
    esac
done

echo ""
echo "✅ 部署计划:"
echo "   nginx: $DEPLOY_NGINX"
echo "   redis: $DEPLOY_REDIS"
echo "   grapes-app: $DEPLOY_GRAPES_APP"
echo "   grapes-pay: $DEPLOY_GRAPES_PAY"
echo "   grapes-llmchat: $DEPLOY_GRAPES_LLMCHAT"
echo "   grapes-asynctask: $DEPLOY_GRAPES_ASYNCTASK"
echo ""

# 检查必要文件
echo "📋 检查必要文件..."
required_files=("docker-compose.yml")
for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ 错误: $file 文件不存在"
        exit 1
    fi
done

# 检查环境变量
echo "🔧 检查环境变量..."
if [ -z "$REGISTRY" ]; then
    echo "❌ 错误: REGISTRY 环境变量未设置"
    exit 1
fi

# 检查 Docker 登录凭据（如果需要拉取镜像）
if [ "$DEPLOY_GRAPES_APP" = "true" ] || [ "$DEPLOY_GRAPES_PAY" = "true" ] || [ "$DEPLOY_GRAPES_LLMCHAT" = "true" ] || [ "$DEPLOY_GRAPES_ASYNCTASK" = "true" ]; then
    if [ -z "$ACR_USERNAME" ] || [ -z "$ACR_PASSWORD" ]; then
        echo "❌ 错误: ACR_USERNAME 或 ACR_PASSWORD 环境变量未设置"
        exit 1
    fi
    
    # 登录阿里云容器镜像仓库
    echo "🔐 登录阿里云容器镜像仓库..."
    echo "$ACR_PASSWORD" | docker login "$REGISTRY" -u "$ACR_USERNAME" --password-stdin
    if [ $? -ne 0 ]; then
        echo "❌ 错误: Docker 登录失败"
        exit 1
    fi
    echo "✅ Docker 登录成功"
fi

# 构建nginx镜像（如果需要）
if [ "$DEPLOY_NGINX" = "true" ]; then
    echo "🔨 构建nginx镜像..."
    if [ ! -f "Dockerfile.nginx" ]; then
        echo "❌ 错误: Dockerfile.nginx 文件不存在"
        exit 1
    fi
    docker build -f Dockerfile.nginx -t grapery-nginx:latest .
    echo "✅ nginx镜像构建完成"
fi

# 拉取需要更新的服务镜像
PULL_SERVICES=""
if [ "$DEPLOY_GRAPES_APP" = "true" ]; then
    PULL_SERVICES="$PULL_SERVICES grapes"
fi
if [ "$DEPLOY_GRAPES_LLMCHAT" = "true" ]; then
    PULL_SERVICES="$PULL_SERVICES grapes-llmchat"
fi
if [ "$DEPLOY_GRAPES_PAY" = "true" ]; then
    PULL_SERVICES="$PULL_SERVICES grapes-vippay"
fi
if [ "$DEPLOY_GRAPES_ASYNCTASK" = "true" ]; then
    PULL_SERVICES="$PULL_SERVICES grapes-asynctask"
fi

if [ -n "$PULL_SERVICES" ]; then
    echo "📥 拉取服务镜像: $PULL_SERVICES"
    docker-compose pull $PULL_SERVICES
    echo "✅ 镜像拉取完成"
fi

# 重启指定的服务
RESTART_SERVICES=""
if [ "$DEPLOY_NGINX" = "true" ]; then
    RESTART_SERVICES="$RESTART_SERVICES nginx"
fi
if [ "$DEPLOY_GRAPES_APP" = "true" ]; then
    RESTART_SERVICES="$RESTART_SERVICES grapes"
fi
if [ "$DEPLOY_GRAPES_LLMCHAT" = "true" ]; then
    RESTART_SERVICES="$RESTART_SERVICES grapes-llmchat"
fi
if [ "$DEPLOY_GRAPES_PAY" = "true" ]; then
    RESTART_SERVICES="$RESTART_SERVICES grapes-vippay"
fi
if [ "$DEPLOY_GRAPES_ASYNCTASK" = "true" ]; then
    RESTART_SERVICES="$RESTART_SERVICES grapes-asynctask"
fi

if [ -n "$RESTART_SERVICES" ]; then
    echo "🔄 重启服务: $RESTART_SERVICES"
    docker-compose up -d --no-deps $RESTART_SERVICES
    echo "✅ 服务重启完成"
fi

# Redis 特殊处理（如果redis是外部服务，这里可能需要重启redis容器或执行其他操作）
if [ "$DEPLOY_REDIS" = "true" ]; then
    echo "🔄 处理 Redis 服务..."
    # 如果redis在docker-compose中，重启它
    if docker-compose ps | grep -q redis; then
        docker-compose restart redis || echo "⚠️  Redis重启失败或不在docker-compose中"
    else
        echo "ℹ️  Redis不在docker-compose中，可能是外部服务，跳过重启"
    fi
fi

# 设置证书文件权限（如果部署了nginx）
if [ "$DEPLOY_NGINX" = "true" ]; then
    echo "🔒 设置SSL证书权限..."
    if [ -f "certs/rankquantity.xyz.key" ]; then
        chmod 600 certs/rankquantity.xyz.key
        chmod 644 certs/rankquantity.xyz.pem
        echo "✅ 证书权限设置完成"
    else
        echo "⚠️  警告: SSL证书文件不存在"
    fi
fi

# 等待服务启动
if [ -n "$RESTART_SERVICES" ]; then
    echo "⏳ 等待服务启动..."
    sleep 10
fi

# 检查服务状态
echo "📊 检查服务状态..."
if [ -n "$RESTART_SERVICES" ]; then
    docker-compose ps $RESTART_SERVICES
else
    docker-compose ps
fi

# 如果部署了nginx，检查nginx日志和健康状态
if [ "$DEPLOY_NGINX" = "true" ]; then
    echo "📋 检查nginx日志..."
    docker-compose logs nginx --tail=20
    
    # 健康检查 - 多次重试
    echo "🏥 执行nginx健康检查..."
    max_retries=5
    retry_count=0
    
    while [ $retry_count -lt $max_retries ]; do
        current_attempt=$((retry_count + 1))
        echo "尝试健康检查 ($current_attempt/$max_retries)..."
        
        if curl -f -s http://localhost/health > /dev/null 2>&1; then
            echo "✅ 健康检查通过"
            break
        else
            echo "❌ 健康检查失败，等待5秒后重试..."
            sleep 5
            retry_count=$((retry_count + 1))
        fi
    done
    
    if [ $retry_count -eq $max_retries ]; then
        echo "❌ 健康检查最终失败"
        echo "📋 显示nginx配置..."
        docker exec grapery-nginx nginx -t || true
        echo "📋 显示nginx日志..."
        docker-compose logs nginx --tail=50
        echo "📋 显示容器状态..."
        docker-compose ps
        exit 1
    fi
fi

echo ""
echo "✅ 选择性部署完成!"
echo ""
echo "📝 已部署的服务:"
if [ "$DEPLOY_NGINX" = "true" ]; then
    echo "   ✅ nginx"
fi
if [ "$DEPLOY_REDIS" = "true" ]; then
    echo "   ✅ redis"
fi
if [ "$DEPLOY_GRAPES_APP" = "true" ]; then
    echo "   ✅ grapes-app"
fi
if [ "$DEPLOY_GRAPES_PAY" = "true" ]; then
    echo "   ✅ grapes-pay"
fi
if [ "$DEPLOY_GRAPES_LLMCHAT" = "true" ]; then
    echo "   ✅ grapes-llmchat"
fi
if [ "$DEPLOY_GRAPES_ASYNCTASK" = "true" ]; then
    echo "   ✅ grapes-asynctask"
fi
echo ""
echo "📋 查看日志:"
echo "   docker-compose logs -f [service_name]" 