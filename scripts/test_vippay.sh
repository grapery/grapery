#!/bin/bash

# VIP 支付服务测试脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 服务配置
SERVICE_URL="http://localhost:8082"
CONFIG_FILE="vippay.json"

echo -e "${YELLOW}=== VIP 支付服务测试 ===${NC}"

# 检查配置文件是否存在
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}错误: 配置文件 $CONFIG_FILE 不存在${NC}"
    exit 1
fi

# 检查服务是否正在运行
echo -e "${YELLOW}1. 检查服务状态...${NC}"
if curl -s "$SERVICE_URL/health" > /dev/null; then
    echo -e "${GREEN}✓ 服务正在运行${NC}"
else
    echo -e "${RED}✗ 服务未运行，请先启动服务${NC}"
    echo "启动命令: ./vippay -config $CONFIG_FILE"
    exit 1
fi

# 测试健康检查接口
echo -e "${YELLOW}2. 测试健康检查接口...${NC}"
HEALTH_RESPONSE=$(curl -s "$SERVICE_URL/health")
echo "响应: $HEALTH_RESPONSE"

# 检查响应格式
if echo "$HEALTH_RESPONSE" | grep -q '"status":"healthy"'; then
    echo -e "${GREEN}✓ 健康检查通过${NC}"
else
    echo -e "${RED}✗ 健康检查失败${NC}"
fi

# 测试 API 接口（需要认证）
echo -e "${YELLOW}3. 测试 API 接口...${NC}"

# 测试未认证访问
echo "测试未认证访问:"
UNAUTH_RESPONSE=$(curl -s -w "%{http_code}" "$SERVICE_URL/api/v1/vip/info" -o /dev/null)
if [ "$UNAUTH_RESPONSE" = "401" ]; then
    echo -e "${GREEN}✓ 认证中间件工作正常${NC}"
else
    echo -e "${RED}✗ 认证中间件异常，期望 401，实际 $UNAUTH_RESPONSE${NC}"
fi

# 测试无效路径
echo "测试无效路径:"
NOT_FOUND_RESPONSE=$(curl -s -w "%{http_code}" "$SERVICE_URL/api/v1/invalid/path" -o /dev/null)
if [ "$NOT_FOUND_RESPONSE" = "404" ]; then
    echo -e "${GREEN}✓ 404 处理正常${NC}"
else
    echo -e "${RED}✗ 404 处理异常，期望 404，实际 $NOT_FOUND_RESPONSE${NC}"
fi

# 测试 CORS 预检请求
echo -e "${YELLOW}4. 测试 CORS 支持...${NC}"
CORS_RESPONSE=$(curl -s -X OPTIONS -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: POST" \
    -H "Access-Control-Request-Headers: Content-Type" \
    -w "%{http_code}" "$SERVICE_URL/api/v1/payment/orders" -o /dev/null)

if [ "$CORS_RESPONSE" = "200" ]; then
    echo -e "${GREEN}✓ CORS 支持正常${NC}"
else
    echo -e "${RED}✗ CORS 支持异常，期望 200，实际 $CORS_RESPONSE${NC}"
fi

# 显示服务信息
echo -e "${YELLOW}5. 服务信息:${NC}"
echo "服务地址: $SERVICE_URL"
echo "配置文件: $CONFIG_FILE"
echo "API 文档: $SERVICE_URL/docs (如果配置了)"

# 显示可用的 API 端点
echo -e "${YELLOW}6. 可用的 API 端点:${NC}"
echo "  GET  /health                    - 健康检查"
echo "  POST /api/v1/payment/orders     - 创建订单"
echo "  GET  /api/v1/payment/orders     - 获取用户订单"
echo "  POST /api/v1/payment/query      - 查询支付状态"
echo "  POST /api/v1/payment/callback   - 支付回调"
echo "  GET  /api/v1/vip/info           - 获取用户VIP信息"
echo "  POST /api/v1/subscription/cancel - 取消订阅"
echo "  POST /callback/alipay           - 支付宝回调"
echo "  POST /callback/wechat           - 微信支付回调"
echo "  POST /callback/stripe           - Stripe 回调"

echo -e "${GREEN}=== 测试完成 ===${NC}"
echo -e "${YELLOW}提示: 要测试完整的支付流程，需要配置真实的支付提供商信息${NC}" 