# 系统通知 API 文档

## 基础信息

- **基础路径**: `/api/llmchat/notifications`
- **认证方式**: Bearer Token (通过 AuthMiddleware 自动获取用户ID)
- **限流**: 已启用 RateLimitMiddleware

---

## 1. 获取通知列表

### 请求信息

- **方法**: `GET`
- **路径**: `/api/llmchat/notifications`
- **认证**: 必需

### 请求参数

**Query 参数**:

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| limit | int | 否 | 20 | 每页数量，范围：1-20 |

**示例**:
```
GET /api/llmchat/notifications?limit=10
```

### 处理流程

1. 从认证中间件获取用户ID
2. 解析 `limit` 参数（无效或 <= 0 时使用默认值 20）
3. 查询用户未删除的通知，按创建时间倒序
4. 返回通知列表和分页信息

### 响应结构

**成功响应** (HTTP 200):

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "notifications": [
      {
        "id": 123,
        "type": "like",
        "title": "点赞提醒",
        "content": "有人点赞了你的故事《xxx》",
        "is_read": false,
        "related_user_id": 456,
        "related_story_id": 789,
        "related_storyboard_id": null,
        "related_comment_id": null,
        "extra_data": {},
        "created_at": 1699123456,
        "updated_at": 1699123456
      }
    ],
    "total_count": 50,
    "has_more": true,
    "next_cursor": 1699123456
  }
}
```

**字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| notifications | array | 通知列表 |
| total_count | int64 | 总通知数 |
| has_more | bool | 是否有更多数据 |
| next_cursor | int64\|null | 下一页游标（最后一条通知的 created_at） |

**通知类型 (type)**:
- `like` - 点赞提醒
- `comment` - 评论提醒
- `follow` - 关注提醒
- `story_created` - 故事创建完成提醒
- `story_published` - 故事发布提醒
- `vip_subscription` - 会员订阅提醒
- `system_update` - 系统更新提醒
- `maintenance` - 维护通知
- `achievement` - 成就解锁提醒

**异常响应**:

| HTTP状态码 | code | message | 说明 |
|-----------|------|---------|------|
| 500 | 500 | error message | 服务器内部错误 |

---

## 2. 标记所有通知为已读

### 请求信息

- **方法**: `POST`
- **路径**: `/api/llmchat/notifications/read_all`
- **认证**: 必需

### 请求参数

**请求体**: 无（空对象即可）

**示例**:
```json
POST /api/llmchat/notifications/read_all
Content-Type: application/json

{}
```

### 处理流程

1. 从认证中间件获取用户ID
2. 批量更新该用户所有未读通知为已读状态
3. 返回成功响应

### 响应结构

**成功响应** (HTTP 200):

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

**异常响应**:

| HTTP状态码 | code | message | 说明 |
|-----------|------|---------|------|
| 500 | 500 | error message | 服务器内部错误 |

---

## 3. 标记单个通知为已读

### 请求信息

- **方法**: `POST`
- **路径**: `/api/llmchat/notifications/:id/read`
- **认证**: 必需

### 请求参数

**路径参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | int64 | 是 | 通知ID |

**请求体**: 无（空对象即可）

**示例**:
```json
POST /api/llmchat/notifications/123/read
Content-Type: application/json

{}
```

### 处理流程

1. 从认证中间件获取用户ID
2. 解析路径参数 `id`（无效或 <= 0 时返回 400）
3. 验证通知是否存在且属于当前用户
4. 更新通知状态为已读
5. 返回成功响应

### 响应结构

**成功响应** (HTTP 200):

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

**异常响应**:

| HTTP状态码 | code | message | 说明 |
|-----------|------|---------|------|
| 400 | 400 | invalid notification id | 通知ID无效 |
| 404 | 404 | notification not found | 通知不存在或不属于当前用户 |
| 500 | 500 | error message | 服务器内部错误 |

---

## 通用说明

### 认证

所有接口都需要在请求头中携带认证Token：

```
Authorization: Bearer <token>
```

用户ID由认证中间件自动从Token中解析，无需在请求中传递。

### 错误处理

所有错误响应都遵循统一格式：

```json
{
  "code": <HTTP状态码>,
  "message": "<错误描述>",
  "data": {}
}
```

### 注意事项

1. `limit` 参数最大值为 20，超过会自动限制为 20
2. 通知按创建时间倒序返回（最新的在前）
3. 已删除的通知不会返回（`deleted = 0`）
4. 关联ID字段（`related_user_id`、`related_story_id` 等）可能为 `null`
5. `extra_data` 字段为扩展数据，类型为对象，可能为空对象

