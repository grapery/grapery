# 故事板场景图片生成 API 使用指南

## 概述

故事板场景图片生成功能已升级，现在支持：
- 自动使用故事的风格配置
- 自动引入场景关联角色的图片作为参考
- 区分过渡场景（无角色）和有角色的场景

## API 端点

### 生成场景图片

**POST** `/api/storyboards/:id/generate/image`

#### 请求参数

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `sceneId` | string | 是 | 场景ID |
| `sceneTitle` | string | 否 | 场景标题 |
| `sceneDescription` | string | 是 | 场景描述 |
| `referenceImages` | string[] | 否 | 额外的参考图片URL列表 |
| `sceneCharacters` | string[] | 否 | 场景中出现的角色名称列表 |
| `characterReferenceImages` | string[] | 否 | 角色参考图片URL列表 |
| `storyStyleId` | string | 否 | 故事风格配置ID（暂未实现） |

#### 请求示例

##### 场景1：有角色出现的场景

```json
{
  "sceneId": "scene-001",
  "sceneTitle": "英雄登场",
  "sceneDescription": "阳光明媚的早晨，主角站在城市的天台上，眺望远方",
  "sceneCharacters": ["张三", "李四"],
  "referenceImages": []
}
```

**说明**：
- 提供了 `sceneCharacters` 字段，系统会自动从故事中获取"张三"和"李四"的海报/头像图片作为参考
- 系统会自动从故事中获取风格配置
- AI 生成时会保持角色外观一致性

##### 场景2：过渡场景（无角色）

```json
{
  "sceneId": "scene-002",
  "sceneTitle": "城市夜景",
  "sceneDescription": "夜幕降临，城市的霓虹灯逐渐亮起，街道上车水马龙",
  "sceneCharacters": [],
  "referenceImages": []
}
```

**说明**：
- `sceneCharacters` 为空，系统识别为过渡场景
- 不会使用角色参考图片
- 只使用故事风格配置生成环境/氛围图片

##### 场景3：手动指定角色参考图片

```json
{
  "sceneId": "scene-003",
  "sceneTitle": "决战时刻",
  "sceneDescription": "两位英雄并肩作战，面对强大的敌人",
  "sceneCharacters": ["张三", "李四"],
  "characterReferenceImages": [
    "https://example.com/character-zhang-san-poster.jpg",
    "https://example.com/character-li-si-poster.jpg"
  ]
}
```

**说明**：
- 手动提供 `characterReferenceImages`，系统会优先使用这些图片
- 如果不提供，系统会自动从故事中获取

#### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "img-gen-001",
    "storyboardId": "storyboard-001",
    "sceneId": "scene-001",
    "sceneTitle": "英雄登场",
    "sceneDescription": "阳光明媚的早晨，主角站在城市的天台上，眺望远方",
    "referenceImages": [
      "https://example.com/character-zhang-san-poster.jpg",
      "https://example.com/character-li-si-avatar.jpg"
    ],
    "sceneCharacters": ["张三", "李四"],
    "characterReferenceImages": [
      "https://example.com/character-zhang-san-poster.jpg",
      "https://example.com/character-li-si-avatar.jpg"
    ],
    "storyStyle": {
      "id": "style-001",
      "style": "anime",
      "description": "日式动漫风格，色彩鲜艳，线条清晰"
    },
    "isTransitionScene": false,
    "status": "pending",
    "createdAt": 1704067200
  }
}
```

### 查询生成进度

**GET** `/api/storyboards/:id/generation-progress`

#### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "storyboardId": "storyboard-001",
    "workflowStatus": "images_ready",
    "currentStep": 3,
    "totalTokens": 15000,
    "isGenerating": false,
    "hasPendingTasks": false,
    "generationMessage": "All images generated successfully",
    "imageGenerations": [
      {
        "id": "img-gen-001",
        "storyboardId": "storyboard-001",
        "sceneId": "scene-001",
        "sceneTitle": "英雄登场",
        "sceneDescription": "阳光明媚的早晨，主角站在城市的天台上，眺望远方",
        "referenceImages": [
          "https://example.com/character-zhang-san-poster.jpg"
        ],
        "sceneCharacters": ["张三", "李四"],
        "characterReferenceImages": [
          "https://example.com/character-zhang-san-poster.jpg"
        ],
        "storyStyle": {
          "id": "style-001",
          "style": "anime",
          "description": "日式动漫风格"
        },
        "isTransitionScene": false,
        "generatedPrompt": "[Art Style: anime] [Style Guide: 日式动漫风格] [Characters: 张三, 李四] 阳光明媚的早晨，主角站在城市的天台上，眺望远方",
        "promptDetails": {
          "artStyle": "anime, digital painting",
          "lighting": "bright morning sunlight",
          "colorPalette": "vibrant colors, warm tones",
          "composition": "wide angle, rule of thirds",
          "keyElements": ["城市天台", "主角剪影", "远方城市景观"],
          "mood": "hopeful, inspiring"
        },
        "generatedImageUrl": "https://oss.example.com/storyboard-001/scene-001.jpg",
        "status": "completed",
        "inputTokens": 150,
        "outputTokens": 80,
        "totalTokens": 230,
        "createdAt": 1704067200,
        "completedAt": 1704067230
      }
    ]
  }
}
```

## 工作流程

### 1. 客户端调用图片生成API

客户端在生成故事板章节后，为每个场景调用图片生成API。

### 2. 服务端处理流程

1. **获取故事风格配置**
   - 如果请求中没有提供 `storyStyleId`，系统会自动从故事中获取风格配置
   - 风格配置包含艺术风格、描述等信息

2. **获取角色参考图片**
   - 如果请求中提供了 `sceneCharacters` 但没有提供 `characterReferenceImages`
   - 系统会根据角色名称从故事中查找对应的角色
   - 优先使用角色的海报图片（`poster`），其次使用头像（`avatar`）

3. **判断场景类型**
   - 如果 `sceneCharacters` 为空且没有角色参考图片，标记为过渡场景
   - 过渡场景不使用角色参考图片

4. **生成AI提示词**
   - 在提示词中包含故事风格配置
   - 如果是有角色的场景，说明需要保持角色一致性
   - 如果是过渡场景，说明不要出现人物

5. **调用图片生成API**
   - 如果有参考图片，使用 image-to-image 模式
   - 如果没有参考图片，使用 text-to-image 模式

### 3. 客户端轮询进度

客户端定期调用 `GET /api/storyboards/:id/generation-progress` 查询生成进度。

## 最佳实践

### 1. 场景角色信息

**推荐做法**：
```json
{
  "sceneCharacters": ["张三", "李四"]
}
```

系统会自动获取角色图片，无需手动提供。

### 2. 过渡场景

对于没有角色出现的场景（如环境描写、远景等），将 `sceneCharacters` 设为空数组：

```json
{
  "sceneCharacters": []
}
```

### 3. 风格一致性

不需要在每次请求中指定风格配置，系统会自动从故事中获取并应用。

### 4. 批量生成

为故事板的多个场景批量生成图片时，建议：
- 按顺序发起请求
- 每个请求间隔 100-200ms，避免过载
- 使用进度查询API监控所有场景的生成状态

## 错误处理

### 常见错误

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| `invalid_params` | 缺少必填参数 | 检查 `sceneId` 和 `sceneDescription` |
| `storyboard_not_found` | 故事板不存在 | 检查故事板ID是否正确 |
| `story_not_found` | 故事不存在 | 确保故事板关联了有效的故事 |
| `character_not_found` | 角色不存在 | 检查 `sceneCharacters` 中的角色名称 |

### 生成失败

如果图片生成失败，响应中的 `status` 会是 `failed`，`errorMessage` 会包含失败原因。

## 注意事项

1. **角色名称匹配**：`sceneCharacters` 中的角色名称必须与故事中定义的角色名称完全一致
2. **图片格式**：生成的图片默认为 16:9 比例的 JPG 格式
3. **生成时间**：图片生成是异步的，通常需要 10-30 秒
4. **Token消耗**：每次生成会消耗AI tokens，具体数量在响应的 `totalTokens` 字段中
5. **风格配置**：如果故事没有设置风格配置，系统会使用默认风格

## 更新日志

### 2024-01-07
- 新增 `sceneCharacters` 字段，支持自动获取角色参考图片
- 新增 `characterReferenceImages` 字段，支持手动指定角色参考图片
- 新增 `storyStyleId` 字段（预留，暂未实现）
- 新增 `isTransitionScene` 字段，标识过渡场景
- 响应中新增 `storyStyle`、`sceneCharacters`、`characterReferenceImages` 等字段
- 优化AI提示词生成逻辑，包含风格配置和场景类型信息

