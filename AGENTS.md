# Grapery Backend - Codex Rules

> 基于 Codex 团队最佳实践，持续迭代更新

## Project Overview
This is the Go backend service for Grapery, a story creation platform. It provides RESTful APIs using Gin framework, MySQL database, and Redis cache.

## Technology Stack
- **Language**: Go 1.22+
- **Framework**: Gin
- **ORM**: GORM
- **Database**: MySQL 8.0
- **Cache**: Redis 7.0
- **Authentication**: JWT

---

## Codex Best Practices

### 1. 遇事不决先 Plan
- 碰上复杂任务，起手先开 Plan Mode
- 先把计划搞细致，代码才能一遍过
- 发现苗头不对，立刻切回 Plan Mode 重新规划

### 2. 提示词技巧
- 让 Codex 担任代码审查员：「针对这些改动对我进行严厉质疑和询问」
- 拒绝凑合：「基于你现在掌握的所有信息，废弃这个方案，并实现一个最优雅的解决方案」
- 需求写详细，减少不必要的歧义

### 3. Subagents 使用
- 复杂问题时，在提示词后加「use subagents」
- 把具体的独立任务交给 subagents，保持主 agent 上下文干净

### 4. 学习模式
- 在 `/config` 中把输出风格改成 Explanatory 或 Learning
- 让 Codex 给新接触的代码库画 ASCII 架构图

---

## Code Style & Conventions

### General Rules
- Follow Go standard formatting (`gofmt`)
- Use `golangci-lint` for linting
- Write clear, self-documenting code
- Prefer composition over inheritance
- Keep functions small and focused (max 50 lines when possible)
- Use meaningful variable and function names

### Package Structure
- `cmd/` - Application entry points (server, chatmcp, vippay)
- `internal/domain/` - Domain models and business entities
- `internal/repository/` - Data access layer (MySQL, Redis)
- `internal/service/` - Business logic layer
- `internal/transport/` - HTTP handlers and routing
- `internal/config/` - Configuration management

### Error Handling
- Always handle errors explicitly, never ignore them
- Use structured errors with context: `fmt.Errorf("failed to create story: %w", err)`
- Log errors with context using Zap logger
- Return errors from functions, don't panic (except in main)

### Logging
- Use structured logging with Zap
- Include relevant context in log messages
- Use appropriate log levels:
  - `Debug`: Detailed information for debugging
  - `Info`: General informational messages
  - `Warn`: Warning messages for recoverable issues
  - `Error`: Error messages for failures

### Database
- Use GORM for database operations
- Always use transactions for multi-step operations
- Use prepared statements for queries with user input
- Implement proper indexes for frequently queried fields
- Use connection pooling

### API Design
- Follow RESTful conventions
- Use consistent response format: `{code, message, data}`
- Validate all input data
- Return appropriate HTTP status codes
- Document API endpoints

### Testing
- Write unit tests for business logic
- Use table-driven tests when appropriate
- Mock external dependencies
- Aim for >80% code coverage

### Security
- Never log sensitive information (passwords, tokens)
- Use JWT for authentication
- Validate and sanitize all user inputs
- Use parameterized queries to prevent SQL injection
- Implement rate limiting for API endpoints

---

## Common Patterns

### Service Layer Pattern
```go
type Service struct {
    repo Repository
    log  *zap.Logger
}

func (s *Service) CreateStory(ctx context.Context, req *CreateRequest) (*Story, error) {
    // Validation
    if err := validate(req); err != nil {
        return nil, err
    }

    // Business logic
    story := &Story{...}

    // Persist
    if err := s.repo.Create(ctx, story); err != nil {
        s.log.Error("failed to create story", zap.Error(err))
        return nil, err
    }

    return story, nil
}
```

### Repository Pattern
```go
type Repository interface {
    Create(ctx context.Context, story *Story) error
    GetByID(ctx context.Context, id string) (*Story, error)
    List(ctx context.Context, filter *Filter) ([]*Story, error)
}
```

---

## When Adding New Features
1. Define domain models in `internal/domain/`
2. Create database models in `internal/repository/mysql/`
3. Implement repository methods
4. Add business logic in `internal/service/`
5. Create HTTP handlers in `internal/transport/http/`
6. Add routes in `internal/transport/http/handler.go`
7. Write tests
8. Update API documentation

---

## Common Mistakes & Solutions

> 每次纠正完错误后，更新此部分

<!-- Example:
### Issue: GORM 查询性能慢
**Solution**: 为常用查询字段添加索引，使用 `db.Preload()` 预加载关联数据
-->

---

## Notes Directory

> 项目相关的笔记和文档

- `docs/` - API 文档
- `README.md` - 项目说明

---

## Dependencies
- Use `go mod` for dependency management
- Pin dependency versions in `go.mod`
- Regularly update dependencies for security patches

## Performance
- Use connection pooling for database and Redis
- Implement caching for frequently accessed data
- Use goroutines for concurrent operations when appropriate
- Profile code to identify bottlenecks

---

## References

- [Codex 官方文档](https://code.Codex.com/docs)
- [Codex 最佳实践 - 知乎](https://zhuanlan.zhihu.com/p/2001805409359520847)
- [git worktrees 并行开发](https://code.Codex.com/docs/en/common-workflows#run-parallel-Codex-sessions-with-git-worktrees)
- [Skills 扩展](https://code.Codex.com/docs/en/skills#extend-Codex-with-skills)
- [Hooks 配置](https://code.Codex.com/docs/en/hooks#permissionrequest)
- [终端配置](https://code.Codex.com/docs/en/terminal-config)
