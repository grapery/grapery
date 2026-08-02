# GenAPI 使用示例

本目录包含了GenAPI的各种使用示例，帮助你快速上手。

## 目录结构

```
examples/
├── simple/      # 简单示例：基础用法
├── advanced/    # 高级示例：多provider、批量处理、错误处理
└── README.md    # 本文件
```

## 运行示例

### 前置准备

1. 设置环境变量（根据需要设置一个或多个）：

```bash
export HAILUO_API_KEY="your-hailuo-api-key"
export HUOSHAN_API_KEY="your-huoshan-api-key"
export GEMINI_API_KEY="your-gemini-api-key"
```

2. 确保已安装依赖：

```bash
cd /path/to/grapery
go mod download
```

### 运行简单示例

```bash
cd pkg/genapi/examples/simple
go run main.go
```

**示例输出**：
```
✅ 已注册Provider: hailuo

=== 示例1: 文本生成视频 ===
📊 [hailuo] Token使用: 120, 图片: 0, 视频: 1
✅ 视频生成任务已创建
   任务ID: task_abc123
   状态: processing
   Provider: hailuo
   耗时: 1.23秒
   Token使用: 120

=== 示例2: 文本生成图片 ===
📊 [hailuo] Token使用: 50, 图片: 2, 视频: 0
✅ 图片生成成功
   生成数量: 2
   耗时: 0.89秒
   图片1: https://example.com/image1.jpg
   图片2: https://example.com/image2.jpg
   图片数量: 2

✅ 所有示例执行完成!
```

### 运行高级示例

```bash
cd pkg/genapi/examples/advanced
go run main.go
```

**示例输出**：
```
✅ 已注册Provider: hailuo
✅ 已注册Provider: huoshan
⚠️  跳过 gemini (未设置环境变量 GEMINI_API_KEY)

=== 示例1: 单个视频生成 (自动Fallback) ===
🎬 尝试使用 hailuo 生成视频...
✅ 使用 hailuo 生成成功

✅ 视频生成成功
   Provider: hailuo
   任务ID: task_xyz789
   状态: processing
   总耗时: 1.45秒
   Token消耗: 150

=== 示例2: 批量视频生成 ===
🎬 开始批量生成 4 个视频...
📊 [hailuo] +120 tokens
📊 [hailuo] +120 tokens
📊 [huoshan] +130 tokens
📊 [hailuo] +120 tokens

📊 批量生成完成，总耗时: 2.31秒

✅ [1] 一只猫在窗台上看雨
      Provider: hailuo, 任务ID: task_001, 耗时: 1.23s
✅ [2] 城市街道的延时摄影
      Provider: hailuo, 任务ID: task_002, 耗时: 1.45s
✅ [3] 海浪拍打礁石
      Provider: huoshan, 任务ID: task_003, 耗时: 1.67s
✅ [4] 樱花飘落的慢镜头
      Provider: hailuo, 任务ID: task_004, 耗时: 1.89s

成功率: 4/4 (100.0%)

📊 === 使用统计报告 ===
Provider: hailuo
  Tokens: 480
  Images: 0
  Videos: 3
Provider: huoshan
  Tokens: 130
  Images: 0
  Videos: 1
总Token消耗: 610

✅ 所有高级示例执行完成!
```

## 示例说明

### Simple Example (简单示例)

展示基础功能：
- ✅ 创建GenAPI实例
- ✅ 注册Provider
- ✅ 文本生成视频
- ✅ 文本生成图片
- ✅ Token使用统计

**适合场景**：快速入门、简单集成

### Advanced Example (高级示例)

展示高级功能：
- ✅ 多Provider注册
- ✅ 自动Fallback机制
- ✅ 批量并发生成
- ✅ 错误处理
- ✅ 使用统计追踪
- ✅ 多种操作类型

**适合场景**：生产环境、高可用需求

## 核心功能演示

### 1. 基础生成

```go
api := genapi.NewGenAPI()
api.RegisterProviderConfig(config)

req := &genapi.GenerateRequest{
    Prompt: "你的提示词",
    DurationSeconds: 6,
}

resp, err := api.GenerateVideo(ctx, "hailuo", req)
```

### 2. 自动Fallback

```go
func generateWithFallback(api *genapi.GenAPI, req *genapi.GenerateRequest) (*genapi.GenerateResponse, error) {
    providers := []string{"hailuo", "huoshan", "gemini"}
    
    for _, provider := range providers {
        resp, err := api.GenerateVideo(ctx, provider, req)
        if err == nil && resp.Error == "" {
            return resp, nil
        }
    }
    
    return nil, fmt.Errorf("所有provider都失败")
}
```

### 3. 批量生成

```go
var wg sync.WaitGroup
results := make([]*genapi.GenerateResponse, len(prompts))

for i, prompt := range prompts {
    wg.Add(1)
    go func(idx int, p string) {
        defer wg.Done()
        req := &genapi.GenerateRequest{Prompt: p}
        results[idx], _ = api.GenerateVideo(ctx, "hailuo", req)
    }(i, prompt)
}

wg.Wait()
```

### 4. Token统计

```go
genapi.SetTokenUsageRecorder(genapi.TokenUsageRecorderFunc(
    func(ctx context.Context, provider string, usage *genapi.Usage) {
        log.Printf("Provider: %s, Tokens: %d", provider, usage.TotalTokens)
        // 保存到数据库
    },
))
```

## 常见问题

**Q: 示例运行时报错 "没有可用的provider"**

A: 请确保至少设置了一个Provider的API密钥环境变量。

**Q: 如何测试不同的Provider？**

A: 修改`api.GenerateVideo()`的第二个参数，可选值：`"hailuo"`, `"huoshan"`, `"gemini"`。

**Q: 批量生成时如何控制并发数？**

A: 使用信号量或工作池模式限制并发：

```go
semaphore := make(chan struct{}, 5) // 最多5个并发
for i, prompt := range prompts {
    semaphore <- struct{}{}
    go func(idx int, p string) {
        defer func() { <-semaphore }()
        // 生成逻辑
    }(i, prompt)
}
```

**Q: 如何获取异步任务的最终结果？**

A: 目前需要使用Provider的原生SDK查询任务状态。后续版本会在GenAPI中添加统一的任务查询接口。

## 下一步

- 查看 [API文档](../README.md) 了解完整的API参考
- 查看 [测试示例](../example_test.go) 了解更多用法
- 集成到你的项目中

## 反馈

如有问题或建议，请提交Issue！

