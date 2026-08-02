#!/bin/bash

# 部署脚本 - 包含主页的nginx服务

set -e

echo "🚀 开始部署 RankQuantity 主页服务..."

# 检查必要文件
echo "📋 检查必要文件..."
if [ ! -f "index.html" ]; then
    echo "❌ 错误: index.html 文件不存在"
    exit 1
fi

if [ ! -f "nginx.conf" ]; then
    echo "❌ 错误: nginx.conf 文件不存在"
    exit 1
fi

if [ ! -f "Dockerfile.nginx" ]; then
    echo "❌ 错误: Dockerfile.nginx 文件不存在"
    exit 1
fi

# 检查SSL证书
echo "🔒 检查SSL证书..."
if [ ! -f "certs/rankquantity.xyz.pem" ] || [ ! -f "certs/rankquantity.xyz.key" ]; then
    echo "⚠️  警告: SSL证书文件不存在，将使用HTTP模式"
    echo "   请确保以下文件存在:"
    echo "   - certs/rankquantity.xyz.pem"
    echo "   - certs/rankquantity.xyz.key"
fi

# 停止现有服务
echo "🛑 停止现有服务..."
docker-compose down

# 构建nginx镜像
echo "🔨 构建nginx镜像..."
docker-compose build nginx

# 启动服务
echo "🚀 启动服务..."
docker-compose up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 15

# 检查服务状态
echo "📊 检查服务状态..."
docker-compose ps

# 健康检查 - 多次重试
echo "🏥 执行健康检查..."
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
    docker exec grapery-nginx nginx -t
    echo "📋 显示nginx日志..."
    docker-compose logs nginx
    echo "📋 显示容器状态..."
    docker-compose ps
    exit 1
fi

echo ""
echo "✅ 部署完成!"
echo ""
echo "🌐 访问地址:"
echo "   HTTP:  http://rankquantity.xyz"
echo "   HTTPS: https://rankquantity.xyz"
echo ""
echo "📝 服务状态:"
echo "   docker-compose ps"
echo ""
echo "📋 查看日志:"
echo "   docker-compose logs nginx"
echo ""
echo "🛑 停止服务:"
echo "   docker-compose down" 