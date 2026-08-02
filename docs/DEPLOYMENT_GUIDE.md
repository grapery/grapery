# CI/CD 选择性部署使用说明

## 概述

CI/CD 流程已更新为支持选择性部署服务。默认情况下，推送代码不会触发部署，需要明确指定要部署的服务才会进行部署。

## 支持的服务

- `nginx` - Nginx 反向代理服务
- `redis` - Redis 缓存服务（如果使用外部 Redis，此选项可能无效）
- `grapes-app` - 主应用服务（也可使用 `app`）
- `grapes-pay` - 支付服务（也可使用 `vippay` 或 `pay`）
- `grapes-llmchat` - LLM 聊天服务（也可使用 `llmchat`）
- `grapes-asynctask` - 异步任务服务（也可使用 `asynctask`）

## 使用方法

### 方法 1: 通过 Commit Message（推荐）

在 commit message 中使用 `[deploy:service1,service2]` 格式指定要部署的服务：

```bash
git commit -m "修复登录bug [deploy:grapes-app]"
git commit -m "更新支付接口 [deploy:grapes-pay,grapes-app]"
git commit -m "更新nginx配置 [deploy:nginx]"
```

**示例**:
- `[deploy:grapes-app]` - 只部署主应用
- `[deploy:grapes-app,grapes-llmchat]` - 部署主应用和 LLM 聊天服务
- `[deploy:nginx]` - 只部署 Nginx
- `[deploy:grapes-app,grapes-pay,grapes-llmchat,grapes-asynctask]` - 部署所有应用服务

### 方法 2: 通过 GitHub Actions 手动触发

1. 在 GitHub 仓库中，进入 **Actions** 标签页
2. 选择 **CI/CD Deploy** workflow
3. 点击 **Run workflow** 按钮
4. 在 **services** 输入框中填写要部署的服务（逗号分隔）
5. 点击 **Run workflow** 执行部署

**示例输入**:
```
grapes-app,grapes-llmchat
```

## 部署流程

1. **检测服务**: 从 commit message 或手动输入中解析要部署的服务
2. **构建镜像**: 只构建指定服务的 Docker 镜像
3. **推送镜像**: 将构建的镜像推送到阿里云容器镜像仓库
4. **上传文件**: 上传必要的配置文件到 ECS
5. **部署服务**: 在 ECS 上拉取最新镜像并重启指定服务

## 注意事项

1. **默认行为**: 如果没有指定服务，CI/CD 流程会跳过部署步骤
2. **服务隔离**: 只有指定的服务会被更新，其他服务保持运行状态不变
3. **Nginx 特殊处理**: 如果部署 nginx，会自动运行 nginx 配置修复脚本
4. **健康检查**: 如果部署了 nginx，会自动执行健康检查
5. **Redis**: 如果 Redis 是外部服务（不在 docker-compose 中），redis 选项可能无效

## 服务名称映射

为了兼容性，支持多种服务名称格式：

| 标准名称 | 支持的别名 |
|---------|-----------|
| `grapes-app` | `app`, `grapes_app` |
| `grapes-pay` | `pay`, `vippay`, `grapes_pay` |
| `grapes-llmchat` | `llmchat`, `grapes_llmchat` |
| `grapes-asynctask` | `asynctask`, `grapes_asynctask` |

## 示例场景

### 场景 1: 只更新主应用
```bash
git commit -m "修复用户登录问题 [deploy:grapes-app]"
git push
```

### 场景 2: 更新多个服务
```bash
git commit -m "更新支付和聊天功能 [deploy:grapes-pay,grapes-llmchat]"
git push
```

### 场景 3: 更新 Nginx 配置
```bash
git commit -m "更新nginx反向代理配置 [deploy:nginx]"
git push
```

### 场景 4: 更新所有应用服务（不包括 nginx 和 redis）
```bash
git commit -m "发布新版本 [deploy:grapes-app,grapes-pay,grapes-llmchat,grapes-asynctask]"
git push
```

## 故障排查

### 问题: 部署没有触发

**原因**: Commit message 中没有 `[deploy:...]` 标记

**解决**: 在 commit message 中添加部署标记，或使用 GitHub Actions 手动触发

### 问题: 服务名称不识别

**原因**: 使用了不支持的服务名称

**解决**: 使用标准服务名称，参考上面的服务名称映射表

### 问题: 部署失败

**检查**:
1. 查看 GitHub Actions 日志
2. 检查 ECS 上的服务状态: `docker-compose ps`
3. 查看服务日志: `docker-compose logs [service_name]`

## 技术细节

- **部署脚本**: `deploy-ci.sh` 支持选择性部署
- **Workflow 文件**: `.github/workflows/deploy.yml`
- **服务重启**: 使用 `docker-compose up -d --no-deps` 只重启指定服务，不影响依赖服务

