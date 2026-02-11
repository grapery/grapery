# grapery API

grapery 是一个使用 Go + Gin + MySQL + Redis 构建的故事创作平台后端服务，为 `creation/` 前端提供完整的 RESTful API。

## 技术栈

- **语言**: Go 1.22
- **Web框架**: Gin
- **数据库**: MySQL (GORM)
- **缓存**: Redis
- **日志**: Zap
- **部署**: Docker + GitHub Actions

## 功能特性

- ✅ 用户管理 - 个人资料、关注系统
- ✅ 故事管理 - 创建、编辑、查看故事和面板
- ✅ 角色管理 - 角色库、角色详情
- ✅ 群组协作 - 群组列表、活动流
- ✅ 聊天系统 - 与角色对话的线程和消息
- ✅ Storyboard - AI增强的叙事生成，支持分支和继续
- ✅ 结构化日志 - 使用 Zap 进行调试和监控
- ✅ CORS 支持 - 跨域资源共享配置
- ✅ 优雅关闭 - 信号处理和资源清理

## 快速开始

### 前置要求

- Go 1.22+
- MySQL 5.7+
- Redis (可选)

### 安装依赖

```bash
cd grapery
go mod download
```

### 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，配置数据库连接信息
```

### 运行服务

```bash
# 开发模式
make run

# 或直接运行
go run ./cmd/server
```

服务将在 `http://localhost:8080` 启动。

### 数据库初始化

服务启动时会自动执行数据库迁移（Auto Migrate），创建所需的表结构。

## API 文档

### 基础路径

所有API端点都以 `/api/v1` 为前缀。

### 端点列表

| 方法 | 路径 | 描述 |
| --- | --- | --- |
| `GET` | `/health` | 健康检查 |
| `GET` | `/users/me` | 获取当前用户信息 |
| `GET` | `/stories` | 获取故事列表 (支持过滤) |
| `GET` | `/stories/:id` | 获取故事详情 (含面板和评论) |
| `POST` | `/stories` | 创建新故事 |
| `GET` | `/characters` | 获取角色列表 |
| `GET` | `/characters/:id` | 获取角色详情 |
| `GET` | `/storyboards/compositions` | 获取故事组合列表 |
| `GET` | `/storyboards/:id` | 获取 Storyboard 详情 |
| `GET` | `/chat/threads` | 获取聊天线程列表 |
| `GET` | `/chat/threads/:id/messages` | 获取聊天消息 |
| `POST` | `/chat/threads/:id/messages` | 发送聊天消息 |

### 查询参数示例

**获取故事列表**

```
GET /api/v1/stories?status=published&authorId=1&search=fantasy
```

## 数据模型

### 核心实体

- **User** - 用户/作者
- **Story** - 故事项目
- **Panel** - 故事面板
- **Character** - 角色
- **Comment** - 评论（支持嵌套回复）
- **ChatThread** - 聊天线程
- **ChatMessage** - 聊天消息
- **StoryComposition** - 故事组合
- **Storyboard** - 故事板节点（树状结构）

### 数据库表关系

```
users
  ├── stories (author_id)
  ├── characters (author_id)
  ├── comments (author_id)
  └── chat_threads (user_id)

stories
  ├── panels (story_id)
  ├── comments (story_id)
  └── storyboards (story_id)

chat_threads
  ├── chat_messages (thread_id)
  └── characters (character_id)

storyboards (树状结构)
  └── parent_id -> storyboards.id
```

## 项目结构

```
grapery/
├── cmd/
│   └── server/          # 主程序入口
│       └── main.go
├── internal/
│   ├── config/          # 配置管理
│   ├── domain/          # 领域模型和接口
│   ├── repository/      # 数据访问层
│   │   └── mysql/       # MySQL 实现
│   ├── service/         # 业务逻辑层
│   ├── transport/       # 传输层
│   │   └── http/        # HTTP 处理器
│   ├── server/          # HTTP 服务器
│   └── telemetry/       # 日志和监控
├── Dockerfile           # Docker 镜像构建
├── Makefile             # 开发任务
├── go.mod               # Go 模块依赖
└── README.md            # 项目文档
```

## 开发指南

### 代码规范

```bash
# 格式化代码
make lint

# 运行测试
make test

# 构建二进制
make build
```

### 添加新功能

1. 在 `internal/domain/models.go` 定义数据模型
2. 在 `internal/repository/mysql/models.go` 定义数据库模型
3. 在 `internal/repository/mysql/repository.go` 实现数据访问方法
4. 在 `internal/service/service.go` 实现业务逻辑
5. 在 `internal/transport/http/handler.go` 添加 HTTP 端点

### 日志记录

使用结构化日志记录关键操作：

```go
s.log.Info("story created", 
    zap.String("storyId", created.ID), 
    zap.String("title", created.Title))

s.log.Error("failed to load panels", zap.Error(err))
```

## Docker 部署

### 构建镜像

```bash
make docker
```

### 运行容器

```bash
docker run -d \
  --name grapery \
  -p 8080:8080 \
  -e DB_DATABASE=grapery \
  -e DB_USERNAME=root \
  -e DB_PASSWORD=12345678 \
  -e DB_ADDRESS=host.docker.internal \
  grapery
```

### Docker Compose

```yaml
version: '3.8'
services:
  grapery:
    build:
      context: .
      dockerfile: grapery/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DB_DATABASE=grapery
      - DB_USERNAME=root
      - DB_PASSWORD=12345678
      - DB_ADDRESS=mysql
      - REDIS_ADDRESS=redis:6379
    depends_on:
      - mysql
      - redis

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: 12345678
      MYSQL_DATABASE: grapery
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  mysql_data:
```

## CI/CD

GitHub Actions 工作流会自动：

1. 运行代码格式检查 (`gofmt`)
2. 执行单元测试 (`go test`)
3. 构建二进制文件 (`go build`)
4. 构建 Docker 镜像
5. 推送到容器注册表 (可选)

## 环境变量

| 变量名 | 描述 | 默认值 |
| --- | --- | --- |
| `grapery_ENV` | 运行环境 | `development` |
| `grapery_HTTP_PORT` | HTTP 监听端口 | `8080` |
| `grapery_LOG_LEVEL` | 日志级别 | `info` |
| `grapery_ALLOW_ORIGIN` | CORS 允许的源 | `http://localhost:5173` |
| `DB_DATABASE` | 数据库名称 | `grapery` |
| `DB_USERNAME` | 数据库用户名 | `root` |
| `DB_PASSWORD` | 数据库密码 | `12345678` |
| `DB_ADDRESS` | 数据库地址 | `localhost` |
| `REDIS_ADDRESS` | Redis 地址 | `localhost:6379` |
| `REDIS_PASSWORD` | Redis 密码 | `` |
| `REDIS_DATABASE` | Redis 数据库编号 | `0` |

## 故障排查

### 数据库连接失败

```bash
# 检查 MySQL 是否运行
mysql -u root -p -h localhost

# 检查数据库是否存在
SHOW DATABASES;

# 创建数据库
CREATE DATABASE grapery CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 端口被占用

```bash
# 查看端口占用
lsof -i :8080

# 修改端口
export grapery_HTTP_PORT=8081
```

## 下一步计划

- [ ] 实现 JWT 认证中间件
- [ ] 添加 Redis 缓存层
- [ ] 实现分页和排序
- [ ] 添加全文搜索 (Elasticsearch)
- [ ] 实现文件上传 (OSS)
- [ ] 添加 WebSocket 支持 (实时聊天)
- [ ] 完善单元测试覆盖率
- [ ] 添加 API 文档 (Swagger)
- [ ] 实现速率限制
- [ ] 添加监控和追踪 (Prometheus + Jaeger)

## 许可证

MIT

