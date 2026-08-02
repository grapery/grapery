# 环境变量配置指南

## 必需的环境变量

视频/图片生成功能需要至少配置一个Provider的API密钥。

### Hailuo (海螺视频) - MiniMax

```bash
export HAILUO_API_KEY="your-hailuo-api-key"
```

**获取方式**:
1. 访问 https://platform.minimax.com/
2. 注册/登录账号
3. 在控制台中创建API密钥

### Huoshan (火山引擎) - 豆包

```bash
export HUOSHAN_API_KEY="your-huoshan-api-key"
```

**获取方式**:
1. 访问 https://console.volcengine.com/ark
2. 注册/登录账号
3. 在API管理中创建密钥

### Gemini (Google)

```bash
export GEMINI_API_KEY="your-gemini-api-key"
```

**获取方式**:
1. 访问 https://ai.google.dev/
2. 登录Google账号
3. 创建项目并获取API密钥

## 配置Provider

编辑 `asynctask.json` 中的 `asynctask.providers` 部分：

```json
{
  "asynctask": {
    "providers": {
      "hailuo": {
        "enabled": true,
        "api_key": "${HAILUO_API_KEY}"
      },
      "huoshan": {
        "enabled": true,
        "api_key": "${HUOSHAN_API_KEY}"
      },
      "gemini": {
        "enabled": false,
        "api_key": "${GEMINI_API_KEY}"
      }
    }
  }
}
```

**注意**:
- `${VAR_NAME}` 会自动从环境变量中读取
- 将 `enabled` 设为 `false` 可以禁用某个Provider
- 未配置API密钥的Provider会被自动跳过

## 快速开始

### 1. 设置环境变量

```bash
# 创建 .env 文件（推荐）
cat > .env << EOF
export HAILUO_API_KEY="your-hailuo-key"
export HUOSHAN_API_KEY="your-huoshan-key"
EOF

# 加载环境变量
source .env
```

### 2. 验证配置

```bash
# 检查环境变量是否设置
echo $HAILUO_API_KEY
echo $HUOSHAN_API_KEY
```

### 3. 启动服务

```bash
cd app/asynctask
go run main.go -config ../../asynctask.json
```

**预期输出**:
```
INFO load config : asynctask.json
INFO registered video provider: hailuo
INFO registered video provider: huoshan
INFO initialized 2 video providers
INFO async task server listening on 0.0.0.0:8050
```

## 故障排查

### Provider未注册

**现象**: 日志显示 "no video providers enabled"

**解决**:
1. 检查环境变量是否设置：`echo $HAILUO_API_KEY`
2. 检查配置文件中 `enabled` 是否为 `true`
3. 查看日志中是否有 "has no API key" 警告

### API密钥无效

**现象**: 任务失败，错误信息包含 "401" 或 "authentication"

**解决**:
1. 确认API密钥是否正确
2. 检查API密钥是否过期
3. 验证账户余额是否充足

### Provider不支持视频生成

**现象**: 日志显示 "does not support video generation"

**解决**:
这是正常的，某些Provider可能只支持图片生成。确保至少有一个支持视频的Provider。

## 生产环境配置

### Docker环境

```dockerfile
ENV HAILUO_API_KEY=your-key
ENV HUOSHAN_API_KEY=your-key
```

### Kubernetes环境

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: provider-keys
type: Opaque
stringData:
  hailuo-key: your-hailuo-key
  huoshan-key: your-huoshan-key
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: asynctask
        env:
        - name: HAILUO_API_KEY
          valueFrom:
            secretKeyRef:
              name: provider-keys
              key: hailuo-key
        - name: HUOSHAN_API_KEY
          valueFrom:
            secretKeyRef:
              name: provider-keys
              key: huoshan-key
```

## 相关文档

- [GenAPI文档](pkg/genapi/README.md)
- [异步任务集成指南](pkg/asynctask/INTEGRATION.md)
- [配置文件示例](config/video_providers_example.yaml)

