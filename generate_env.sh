#!/bin/bash
# 自动生成 .env 文件脚本
# 用法：bash generate_env.sh

cat > .env <<EOF
DB_HOST=${DB_HOST:-mysql}
DB_PORT=${DB_PORT:-3306}
DB_USER=${DB_USER:-grapery}
DB_PASSWORD=${DB_PASSWORD:-your_db_password}
DB_NAME=${DB_NAME:-grapery}
REDIS_HOST=${REDIS_HOST:-redis}
REDIS_PORT=${REDIS_PORT:-6379}
REDIS_PASSWORD=${REDIS_PASSWORD:-your_redis_password}
ENV=production
APP_PORT=8080
JWT_SECRET=${JWT_SECRET:-your-jwt-secret-key}
JWT_EXPIRE_HOURS=24
# 其他必要变量可继续追加
EOF

echo ".env 文件已生成。" 