# Prometheus 监控指标使用指南

## 概述

本文档说明如何使用新增的故事板生成相关的 Prometheus 监控指标。

## 新增指标分类

### 1. 故事板生成工作流指标

#### StoryboardContentGenerations
**指标名称**: `storyboard_content_generations_total`  
**类型**: Counter  
**标签**: `status` ("pending" | "processing" | "completed" | "failed")  
**说明**: 故事板内容生成总数

**使用示例**:
```go
metrics.RecordStoryboardContentGeneration("completed", duration)
```

#### StoryboardContentGenerationTime
**指标名称**: `storyboard_content_generation_duration_seconds`  
**类型**: Histogram  
**标签**: `status`  
**说明**: 故事板内容生成耗时分布

#### StoryboardSceneGenerations
**指标名称**: `storyboard_scene_generations_total`  
**类型**: Counter  
**标签**: `status`  
**说明**: 故事板场景详情生成总数

**使用示例**:
```go
metrics.RecordStoryboardSceneGeneration("completed", duration)
```

#### StoryboardImageGenerations
**指标名称**: `storyboard_image_generations_total`  
**类型**: Counter  
**标签**: `status`, `scene_type` ("transition" | "with_characters")  
**说明**: 故事板图片生成总数

**使用示例**:
```go
sceneType := "with_characters"
if isTransitionScene {
    sceneType = "transition"
}
metrics.RecordStoryboardImageGeneration("completed", sceneType, duration)
```

#### StoryboardVideoGenerations
**指标名称**: `storyboard_video_generations_total`  
**类型**: Counter  
**标签**: `status`, `is_subdivided` ("true" | "false")  
**说明**: 故事板视频生成总数

**使用示例**:
```go
metrics.RecordStoryboardVideoGeneration("completed", isSubdivided, duration)
```

### 2. 图片生成详细指标

#### ImageGenerationWithCharacters
**指标名称**: `image_generation_with_characters_total`  
**类型**: Counter  
**标签**: `status`, `character_count` ("0" | "1" | "2+")  
**说明**: 使用角色参考的图片生成数量

**使用示例**:
```go
characterCount := len(sceneCharacters)
metrics.RecordImageGenerationWithCharacters("completed", characterCount)
```

#### ImageGenerationWithStyle
**指标名称**: `image_generation_with_style_total`  
**类型**: Counter  
**标签**: `status`, `has_style` ("true" | "false")  
**说明**: 使用故事风格配置的图片生成数量

**使用示例**:
```go
hasStyle := storyStyle != nil
metrics.RecordImageGenerationWithStyle("completed", hasStyle)
```

#### ImageGenerationPromptDetailsUsed
**指标名称**: `image_generation_prompt_details_used_total`  
**类型**: Counter  
**标签**: `has_prompt_details` ("true" | "false")  
**说明**: 使用结构化提示词的图片生成数量

**使用示例**:
```go
hasPromptDetails := promptDetails != nil
metrics.RecordImageGenerationPromptDetails(hasPromptDetails)
```

#### ImageGenerationCharacterRefs
**指标名称**: `image_generation_character_refs_count`  
**类型**: Histogram  
**标签**: `scene_type`  
**说明**: 图片生成中使用的角色参考数量分布

**使用示例**:
```go
metrics.RecordImageGenerationCharacterRefs(sceneType, float64(len(characterRefs)))
```

#### ImageGenerationTokenConsumed
**指标名称**: `image_generation_token_consumed`  
**类型**: Histogram  
**标签**: `step` ("prompt" | "image"), `scene_type`  
**说明**: 图片生成各步骤的 Token 消耗分布

**使用示例**:
```go
// 记录提示词生成的 Token
metrics.RecordImageGenerationTokenConsumed("prompt", sceneType, float64(promptTokens))

// 记录图片生成的 Token
metrics.RecordImageGenerationTokenConsumed("image", sceneType, float64(imageTokens))
```

#### ImageGenerationErrors
**指标名称**: `image_generation_errors_total`  
**类型**: Counter  
**标签**: `error_type` ("ai_error" | "image_api_error" | "parsing_error" | "timeout" | "unknown")  
**说明**: 图片生成错误总数

**使用示例**:
```go
if err != nil {
    errorType := "unknown"
    if strings.Contains(err.Error(), "AI") {
        errorType = "ai_error"
    } else if strings.Contains(err.Error(), "image") {
        errorType = "image_api_error"
    } else if strings.Contains(err.Error(), "parse") {
        errorType = "parsing_error"
    } else if strings.Contains(err.Error(), "timeout") {
        errorType = "timeout"
    }
    metrics.RecordImageGenerationError(errorType)
}
```

### 3. 视频生成详细指标

#### VideoGenerationSubdivided
**指标名称**: `video_generation_subdivided_total`  
**类型**: Counter  
**标签**: `is_subdivided` ("true" | "false"), `status`  
**说明**: 使用视频分段的视频生成数量

**使用示例**:
```go
metrics.RecordVideoGenerationSubdivided(isSubdivided, "completed")
```

#### VideoGenerationSegmentCount
**指标名称**: `video_generation_segment_count`  
**类型**: Histogram  
**标签**: `is_subdivided`  
**说明**: 视频分段数量分布

**使用示例**:
```go
metrics.RecordVideoGenerationSegmentCount(isSubdivided, float64(len(segments)))
```

#### VideoGenerationTokenConsumed
**指标名称**: `video_generation_token_consumed`  
**类型**: Histogram  
**标签**: `step` ("prompt" | "video")  
**说明**: 视频生成各步骤的 Token 消耗分布

**使用示例**:
```go
metrics.RecordVideoGenerationTokenConsumed("prompt", float64(promptTokens))
metrics.RecordVideoGenerationTokenConsumed("video", float64(videoTokens))
```

#### VideoGenerationErrors
**指标名称**: `video_generation_errors_total`  
**类型**: Counter  
**标签**: `error_type` ("ai_error" | "video_api_error" | "timeout" | "unknown")  
**说明**: 视频生成错误总数

### 4. 角色海报生成指标

#### CharacterPosterGenerations
**指标名称**: `character_poster_generations_total`  
**类型**: Counter  
**标签**: `status`, `has_story_reference` ("true" | "false")  
**说明**: 角色海报生成总数

**使用示例**:
```go
hasStoryRef := poster.ReferenceStoryEnabled
metrics.RecordCharacterPosterGeneration("completed", hasStoryRef)
```

#### CharacterPosterGenerationTime
**指标名称**: `character_poster_generation_duration_seconds`  
**类型**: Histogram  
**标签**: `step` ("concept" | "image")  
**说明**: 角色海报生成各步骤耗时分布

**使用示例**:
```go
// 记录概念生成耗时
metrics.RecordCharacterPosterGenerationTime("concept", conceptDuration)

// 记录图片生成耗时
metrics.RecordCharacterPosterGenerationTime("image", imageDuration)
```

#### CharacterPosterConceptTime
**指标名称**: `character_poster_concept_duration_seconds`  
**类型**: Histogram  
**标签**: `status`  
**说明**: 角色海报概念生成耗时分布

#### CharacterPosterImageTime
**指标名称**: `character_poster_image_duration_seconds`  
**类型**: Histogram  
**标签**: `status`  
**说明**: 角色海报图片生成耗时分布

#### CharacterPosterTokenConsumed
**指标名称**: `character_poster_token_consumed`  
**类型**: Histogram  
**标签**: `step` ("concept" | "image")  
**说明**: 角色海报生成各步骤的 Token 消耗分布

#### CharacterPosterErrors
**指标名称**: `character_poster_errors_total`  
**类型**: Counter  
**标签**: `error_type` ("concept_error" | "image_error" | "parsing_error" | "unknown")  
**说明**: 角色海报生成错误总数

### 5. 故事风格配置指标

#### StoryStyleConfigUsage
**指标名称**: `story_style_config_usage_total`  
**类型**: Counter  
**标签**: `style_id`, `usage_type` ("image_generation" | "video_generation" | "poster_generation")  
**说明**: 故事风格配置使用次数

**使用示例**:
```go
if storyStyle != nil {
    metrics.RecordStoryStyleConfigUsage(storyStyle.ID, "image_generation")
}
```

#### StoryStyleConfigCount
**指标名称**: `story_style_config_count`  
**类型**: Gauge  
**说明**: 故事风格配置总数

**使用示例**:
```go
count := getTotalStyleConfigCount()
metrics.RecordStoryStyleConfigCount(float64(count))
```

#### StoryStyleConfigByStyle
**指标名称**: `story_style_config_by_style`  
**类型**: GaugeVec  
**标签**: `style`  
**说明**: 按风格名称统计的风格配置数量

**使用示例**:
```go
metrics.RecordStoryStyleConfigByStyle("anime", float64(animeCount))
metrics.RecordStoryStyleConfigByStyle("realistic", float64(realisticCount))
```

### 6. AI 生成质量指标

#### AIGenerationSuccessRate
**指标名称**: `ai_generation_success_rate`  
**类型**: Gauge  
**标签**: `type` ("image" | "video" | "poster" | "content" | "scene"), `provider`  
**说明**: AI 生成成功率 (0-1)

**使用示例**:
```go
successRate := float64(successCount) / float64(totalCount)
metrics.RecordAIGenerationSuccessRate("image", "gemini", successRate)
```

#### AIGenerationAverageTokens
**指标名称**: `ai_generation_average_tokens`  
**类型**: Gauge  
**标签**: `type`, `provider`  
**说明**: AI 生成平均 Token 消耗

**使用示例**:
```go
avgTokens := float64(totalTokens) / float64(count)
metrics.RecordAIGenerationAverageTokens("image", "gemini", avgTokens)
```

#### AIGenerationAverageDuration
**指标名称**: `ai_generation_average_duration_seconds`  
**类型**: Gauge  
**标签**: `type`, `provider`  
**说明**: AI 生成平均耗时

**使用示例**:
```go
avgDuration := totalDuration.Seconds() / float64(count)
metrics.RecordAIGenerationAverageDuration("image", "gemini", avgDuration)
```

#### AIGenerationRetries
**指标名称**: `ai_generation_retries_total`  
**类型**: Counter  
**标签**: `type`, `provider`, `retry_count` ("1" | "2" | "3+")  
**说明**: AI 生成重试次数

**使用示例**:
```go
metrics.RecordAIGenerationRetry("image", "gemini", retryCount)
```

### 7. 故事板工作流完成指标

#### StoryboardWorkflowCompleted
**指标名称**: `storyboard_workflow_completed_total`  
**类型**: Counter  
**标签**: `workflow_status` ("content_ready" | "images_ready" | "video_ready" | "published")  
**说明**: 完成的故事板工作流总数

**使用示例**:
```go
duration := time.Since(workflowStartTime)
metrics.RecordStoryboardWorkflowCompleted("images_ready", duration)
```

#### StoryboardWorkflowDuration
**指标名称**: `storyboard_workflow_duration_seconds`  
**类型**: Histogram  
**标签**: `workflow_status`  
**说明**: 故事板工作流总耗时分布

#### StoryboardWorkflowAbandoned
**指标名称**: `storyboard_workflow_abandoned_total`  
**类型**: Counter  
**标签**: `abandoned_at_step` ("content" | "images" | "video")  
**说明**: 被放弃的故事板工作流总数

**使用示例**:
```go
if workflowAbandoned {
    metrics.RecordStoryboardWorkflowAbandoned("images")
}
```

## 在代码中集成

### 示例：在图片生成服务中记录指标

```go
func (s *Service) processImageGeneration(ctx context.Context, gen *domain.StoryboardImageGeneration) {
    startTime := time.Now()
    
    // 判断场景类型
    sceneType := "with_characters"
    characterCount := len(gen.SceneCharacters)
    if gen.IsTransitionScene || characterCount == 0 {
        sceneType = "transition"
    }
    
    // 记录开始生成
    metrics.RecordStoryboardImageGeneration("processing", sceneType, 0)
    
    // 记录角色引用
    if characterCount > 0 {
        metrics.RecordImageGenerationCharacterRefs(sceneType, float64(characterCount))
        metrics.RecordImageGenerationWithCharacters("processing", characterCount)
    }
    
    // 记录风格配置使用
    hasStyle := gen.StoryStyle != nil
    if hasStyle {
        metrics.RecordImageGenerationWithStyle("processing", true)
        metrics.RecordStoryStyleConfigUsage(gen.StoryStyle.ID, "image_generation")
    }
    
    // 记录结构化提示词使用
    hasPromptDetails := gen.PromptDetails != nil
    if hasPromptDetails {
        metrics.RecordImageGenerationPromptDetails(true)
    }
    
    // 生成图片...
    err := s.generateImage(ctx, gen)
    
    duration := time.Since(startTime)
    
    if err != nil {
        // 记录错误
        errorType := classifyError(err)
        metrics.RecordImageGenerationError(errorType)
        metrics.RecordStoryboardImageGeneration("failed", sceneType, duration)
        return
    }
    
    // 记录成功
    metrics.RecordStoryboardImageGeneration("completed", sceneType, duration)
    metrics.RecordImageGenerationWithCharacters("completed", characterCount)
    metrics.RecordImageGenerationWithStyle("completed", hasStyle)
    
    // 记录 Token 消耗
    metrics.RecordImageGenerationTokenConsumed("prompt", sceneType, float64(gen.InputTokens+gen.OutputTokens))
    // 图片生成的 Token 通常不单独记录，因为由外部 API 处理
}
```

## Prometheus 查询示例

### 查询图片生成成功率

```promql
# 按场景类型统计成功率
sum(rate(storyboard_image_generations_total{status="completed"}[5m])) 
/ 
sum(rate(storyboard_image_generations_total[5m]))
```

### 查询平均生成耗时

```promql
# 图片生成平均耗时（按场景类型）
rate(storyboard_image_generation_duration_seconds_sum[5m]) 
/ 
rate(storyboard_image_generation_duration_seconds_count[5m])
```

### 查询 Token 消耗分布

```promql
# 图片生成 Token 消耗（按步骤）
histogram_quantile(0.95, 
  rate(image_generation_token_consumed_bucket[5m])
)
```

### 查询错误率

```promql
# 图片生成错误率
sum(rate(image_generation_errors_total[5m])) 
/ 
sum(rate(storyboard_image_generations_total[5m]))
```

### 查询工作流完成率

```promql
# 工作流完成率
sum(rate(storyboard_workflow_completed_total[5m])) 
/ 
sum(rate(storyboard_content_generations_total[5m]))
```

## Grafana 仪表板建议

### 1. 故事板生成概览
- 各步骤生成总数（Counter）
- 各步骤成功率（Rate）
- 平均耗时（Histogram）

### 2. 图片生成详情
- 过渡场景 vs 有角色场景对比
- 使用风格配置的比例
- 使用结构化提示词的比例
- Token 消耗趋势
- 错误类型分布

### 3. 视频生成详情
- 分段视频使用率
- 平均分段数量
- Token 消耗趋势

### 4. 角色海报生成
- 生成成功率
- 概念生成 vs 图片生成耗时对比
- Token 消耗分布

### 5. 风格配置使用
- 各风格使用频率
- 风格配置总数趋势

### 6. 工作流分析
- 工作流完成率
- 工作流耗时分布
- 工作流放弃率（按步骤）

## 注意事项

1. **标签值一致性**: 确保标签值使用预定义的值（如 "true"/"false" 而不是 "True"/"False"）
2. **指标命名**: 遵循 Prometheus 命名规范（小写字母、下划线分隔）
3. **Histogram Buckets**: 根据实际数据分布调整 buckets，避免过度或不足
4. **性能影响**: 指标记录应该是轻量级的，避免影响业务性能
5. **错误分类**: 统一错误类型分类，便于后续分析

## 更新日志

### 2025-01-07
- 新增故事板生成工作流指标
- 新增图片生成详细指标（角色引用、风格配置、结构化提示词）
- 新增视频生成详细指标（分段、Token消耗）
- 新增角色海报生成指标
- 新增故事风格配置指标
- 新增 AI 生成质量指标
- 新增故事板工作流完成指标

