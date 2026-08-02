# GenAPI - 统一媒体生成代理层

GenAPI 是一个统一的图片/视频生成代理层，提供了一致的接口来调用多个AI媒体生成服务提供商（Hailuo、Huoshan、Gemini），使业务层无需关心底层SDK的差异。

## 特性

- 🎯 **统一接口**：统一的请求/响应格式，简化业务层调用
- 🔌 **多Provider支持**：支持 Hailuo、Huoshan (火山引擎)、Gemini (Google)
- 🎨 **图片生成**：文本生图、图生图、图片编辑
- 🎬 **视频生成**：文本生视频、图生视频、关键帧生视频
- 📊 **Token统计**：自动记录和上报Token使用量
- ⚡ **异步任务**：支持长时间运行的生成任务
- 🛡️ **类型安全**：完整的Go类型定义

## 快速开始

### 安装

```bash
go get github.com/grapery/grapery/pkg/genapi
```

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/grapery/grapery/pkg/genapi"
)

func main() {
    // 1. 创建GenAPI实例
    api := genapi.NewGenAPI()

    // 2. 配置并注册Provider
    hailuoCfg := &genapi.Config{
        Provider: genapi.ProviderHailuo,
        APIKey:   "your-hailuo-api-key",
        BaseURL:  "https://api.minimax.com",
        Model:    "MiniMax-Hailuo-02",
        Timeout:  30 * time.Second,
    }
    
    provider, err := api.RegisterProviderConfig(hailuoCfg)
    if err != nil {
        log.Fatal(err)
    }

    // 3. 生成视频
    ctx := context.Background()
    videoReq := &genapi.GenerateRequest{
        Operation:       genapi.OperationTextToVideo,
        Prompt:          "一只可爱的小猫在草地上玩耍，阳光明媚",
        DurationSeconds: 6,
        AspectRatio:     "16:9",
        Quality:         "high",
    }
    
    videoResp, err := api.GenerateVideo(ctx, "hailuo", videoReq)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("视频任务ID: %s\n", videoResp.TaskID)
    fmt.Printf("状态: %s\n", videoResp.Status)
    
    // 4. 生成图片
    imageReq := &genapi.GenerateRequest{
        Operation:   genapi.OperationTextToImage,
        Prompt:      "一幅美丽的日落风景画，油画风格",
        AspectRatio: "16:9",
        OutputCount: 4,
    }
    
    imageResp, err := api.GenerateImage(ctx, "hailuo", imageReq)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("生成了 %d 张图片\n", len(imageResp.ImageURLs))
    for i, url := range imageResp.ImageURLs {
        fmt.Printf("  图片 %d: %s\n", i+1, url)
    }
}
```

## Provider配置

### Hailuo (海螺视频)

```go
config := &genapi.Config{
    Provider:   genapi.ProviderHailuo,
    APIKey:     "your-api-key",
    BaseURL:    "https://api.minimax.com",
    Model:      "MiniMax-Hailuo-02",  // 视频模型
    ImageModel: "image-01",            // 图片模型
    Timeout:    30 * time.Second,
}
```

**支持的操作**：
- ✅ 文本生视频 (text_to_video)
- ✅ 图生视频 (image_to_video)
- ✅ 关键帧生视频 (keyframe_to_video)
- ✅ 文本生图 (text_to_image)

### Huoshan (火山引擎 - 豆包)

```go
config := &genapi.Config{
    Provider:     genapi.ProviderHuoshan,
    APIKey:       "your-api-key",
    BaseURL:      "https://open.byteplusapi.com",
    ImageBaseURL: "https://ark.cn-beijing.volces.com/api/v3",
    Workflow:     "doubao-seedance-1-0-pro-250528",  // 视频模型
    ImageModel:   "doubao-seedream-4-0-250828",      // 图片模型
    Timeout:      30 * time.Second,
}
```

**支持的操作**：
- ✅ 文本生视频 (text_to_video)
- ✅ 图生视频 (image_to_video)
- ✅ 关键帧生视频 (keyframe_to_video)
- ✅ 文本生图 (text_to_image)
- ✅ 图生图 (image_to_image)

### Gemini (Google)

```go
config := &genapi.Config{
    Provider: genapi.ProviderGemini,
    APIKey:   "your-google-api-key",
    Model:    "veo-001",           // 视频模型
    Timeout:  60 * time.Second,
}
```

**支持的操作**：
- ✅ 文本生视频 (text_to_video)
- ✅ 图生视频 (image_to_video)
- ✅ 文本生图 (text_to_image)
- ✅ 图生图 (image_to_image) - 通过conversational API

## API参考

### GenerateRequest

统一的生成请求参数：

```go
type GenerateRequest struct {
    // 操作类型（自动推断或手动指定）
    Operation OperationType
    
    // 文本提示词
    Prompt string
    NegativePrompt string
    
    // 视频参数
    DurationSeconds int      // 视频时长（秒）
    AspectRatio     string   // 宽高比 "16:9", "9:16", "1:1"
    Resolution      string   // 分辨率
    Quality         string   // 质量 "high", "medium", "low"
    
    // 图片参数
    Size           string    // 图片尺寸 "1024x1024"
    Width          int       // 宽度
    Height         int       // 高度
    OutputCount    int       // 输出数量
    ResponseFormat string    // 响应格式 "url", "b64_json"
    
    // 引用资源
    ReferenceImageURL string   // 参考图片URL
    ReferenceImages   []string // 多个参考图片
    FirstFrameURL     string   // 首帧图片
    LastFrameURL      string   // 尾帧图片
    ImageData         []byte   // 图片字节数据
    ImageMIMEType     string   // 图片MIME类型
    
    // 高级参数
    Model         string    // 模型名称
    Style         string    // 风格
    Seed          int       // 随机种子
    GuidanceScale float64   // 引导强度
    Watermark     *bool     // 是否添加水印
    CallbackURL   string    // 回调URL
    
    // 扩展参数
    Mode     GenerationMode           // 生成模式
    Metadata map[string]interface{}   // 元数据
    Options  map[string]interface{}   // 额外选项
}
```

### GenerateResponse

统一的响应格式：

```go
type GenerateResponse struct {
    Provider    string        // Provider名称
    Operation   OperationType // 操作类型
    MediaType   MediaType     // 媒体类型 image/video
    
    // 任务信息
    TaskID   string    // 任务ID
    Status   string    // 状态
    Message  string    // 消息
    Progress int       // 进度 (0-100)
    
    // 结果
    ImageURLs    []string  // 图片URL列表
    VideoURL     string    // 视频URL
    ThumbnailURL string    // 缩略图URL
    
    // 错误信息
    Error     string    // 错误消息
    ErrorCode string    // 错误码
    
    // 使用统计
    Usage *Usage    // Token使用量
    
    // 时间信息
    StartedAt   time.Time    // 开始时间
    CompletedAt time.Time    // 完成时间
    
    // 原始数据
    Metadata map[string]interface{}    // 元数据
    Raw      map[string]interface{}    // 原始响应
}
```

### Usage

Token使用统计：

```go
type Usage struct {
    InputTokens     int    // 输入Token数
    OutputTokens    int    // 输出Token数
    TotalTokens     int    // 总Token数
    ImageCount      int    // 图片数量
    VideoCount      int    // 视频数量
    DurationSeconds int    // 视频时长（秒）
    Additional      map[string]interface{}    // 额外信息
}
```

## 使用示例

### 示例1：文本生成视频

```go
req := &genapi.GenerateRequest{
    Prompt:          "夕阳下的海滩，海浪轻拍岸边，海鸥在天空飞翔",
    DurationSeconds: 6,
    AspectRatio:     "16:9",
    Quality:         "high",
}

resp, err := api.GenerateVideo(ctx, "hailuo", req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("任务ID: %s\n", resp.TaskID)
fmt.Printf("视频URL: %s\n", resp.VideoURL)
```

### 示例2：图片生成视频

```go
req := &genapi.GenerateRequest{
    Operation:         genapi.OperationImageToVideo,
    ReferenceImageURL: "https://example.com/image.jpg",
    Prompt:            "让图片中的人物动起来，微笑并挥手",
    DurationSeconds:   6,
}

resp, err := api.GenerateVideo(ctx, "huoshan", req)
```

### 示例3：关键帧生成视频

```go
req := &genapi.GenerateRequest{
    Operation:     genapi.OperationKeyframeToVideo,
    FirstFrameURL: "https://example.com/frame1.jpg",
    LastFrameURL:  "https://example.com/frame2.jpg",
    Prompt:        "平滑过渡，添加动态效果",
    DurationSeconds: 6,
}

resp, err := api.GenerateVideo(ctx, "hailuo", req)
```

### 示例4：文本生成图片

```go
req := &genapi.GenerateRequest{
    Prompt:      "一只戴着墨镜的猫，赛博朋克风格",
    AspectRatio: "1:1",
    OutputCount: 4,
    Style:       "cyberpunk",
}

resp, err := api.GenerateImage(ctx, "gemini", req)

for i, url := range resp.ImageURLs {
    fmt.Printf("图片%d: %s\n", i+1, url)
}
```

### 示例5：图片编辑（图生图）

```go
req := &genapi.GenerateRequest{
    Operation:         genapi.OperationImageToImage,
    ReferenceImageURL: "https://example.com/base.jpg",
    Prompt:            "将背景改成海滩，保持主体不变",
    AspectRatio:       "16:9",
}

resp, err := api.GenerateImage(ctx, "huoshan", req)
```

### 示例6：使用字节数据

```go
imageData, _ := os.ReadFile("input.jpg")

req := &genapi.GenerateRequest{
    Operation:     genapi.OperationImageToVideo,
    ImageData:     imageData,
    ImageMIMEType: "image/jpeg",
    Prompt:        "添加动态效果",
}

resp, err := api.GenerateVideo(ctx, "gemini", req)
```

## Token使用统计

### 设置全局Token记录器

```go
// 在应用启动时设置
genapi.SetTokenUsageRecorder(genapi.TokenUsageRecorderFunc(
    func(ctx context.Context, provider string, usage *genapi.Usage) {
        log.Printf("Provider: %s, Tokens: %d, Images: %d, Videos: %d",
            provider, usage.TotalTokens, usage.ImageCount, usage.VideoCount)
        
        // 保存到数据库
        saveUsageToDatabase(ctx, provider, usage)
    },
))
```

### 从响应中获取使用统计

```go
resp, err := api.GenerateVideo(ctx, "hailuo", req)
if err != nil {
    log.Fatal(err)
}

if resp.Usage != nil {
    fmt.Printf("Token使用量: %d\n", resp.Usage.TotalTokens)
    fmt.Printf("生成时长: %.2fs\n", resp.Duration().Seconds())
}
```

## 高级用法

### 自定义Provider选项

```go
req := &genapi.GenerateRequest{
    Prompt:  "示例提示词",
    Options: map[string]interface{}{
        "prompt_optimizer": true,           // Hailuo: 启用提示词优化
        "fast_pretreatment": true,          // Hailuo: 快速预处理
        "workflow": "custom-workflow",       // Huoshan: 自定义工作流
        "sequential_image_generation": "enabled",  // Huoshan: 连续生图
    },
}
```

### 批量注册Provider

```go
configs := []*genapi.Config{
    {Provider: genapi.ProviderHailuo, APIKey: "key1"},
    {Provider: genapi.ProviderHuoshan, APIKey: "key2"},
    {Provider: genapi.ProviderGemini, APIKey: "key3"},
}

for _, cfg := range configs {
    if _, err := api.RegisterProviderConfig(cfg); err != nil {
        log.Printf("注册 %s 失败: %v", cfg.Provider, err)
    }
}
```

### 动态选择Provider

```go
func generateWithFallback(api *genapi.GenAPI, req *genapi.GenerateRequest) (*genapi.GenerateResponse, error) {
    providers := []string{"hailuo", "huoshan", "gemini"}
    
    for _, provider := range providers {
        resp, err := api.GenerateVideo(ctx, provider, req)
        if err == nil {
            return resp, nil
        }
        log.Printf("Provider %s 失败: %v, 尝试下一个", provider, err)
    }
    
    return nil, fmt.Errorf("所有Provider都失败")
}
```

## 操作类型

GenAPI 支持以下操作类型，会根据请求参数自动推断：

| OperationType | 说明 | 自动推断条件 |
|--------------|------|------------|
| `OperationTextToImage` | 文本生图 | 有Prompt，无参考图 |
| `OperationImageToImage` | 图生图 | 有Prompt + ReferenceImages |
| `OperationTextToVideo` | 文本生视频 | 有Prompt，无参考图/帧 |
| `OperationImageToVideo` | 图生视频 | 有ReferenceImageURL或ImageData |
| `OperationKeyframeToVideo` | 关键帧生视频 | 有FirstFrameURL + LastFrameURL |
| `OperationStoryboardToVideo` | 分镜生视频 | 有Storyboard |

也可以手动指定：

```go
req := &genapi.GenerateRequest{
    Operation: genapi.OperationImageToVideo,  // 明确指定操作类型
    // ... 其他参数
}
```

## 错误处理

```go
resp, err := api.GenerateVideo(ctx, "hailuo", req)
if err != nil {
    // 处理调用错误
    log.Printf("调用失败: %v", err)
    return err
}

if resp.Error != "" {
    // 处理Provider返回的错误
    log.Printf("生成失败: %s (code: %s)", resp.Error, resp.ErrorCode)
    return fmt.Errorf("generation failed: %s", resp.Error)
}

// 成功
fmt.Printf("任务ID: %s, 状态: %s\n", resp.TaskID, resp.Status)
```

## 最佳实践

1. **配置管理**：将API密钥存储在环境变量或配置文件中
2. **错误处理**：始终检查错误和响应中的Error字段
3. **超时设置**：根据生成任务的复杂度设置合适的超时时间
4. **Token监控**：使用TokenUsageRecorder监控和控制成本
5. **异步任务**：对于长时间运行的任务，使用TaskID进行状态查询
6. **Provider切换**：实现fallback机制提高可用性

## 常见问题

**Q: 如何查询异步任务状态？**

A: 保存返回的`TaskID`，然后使用Provider的原生SDK查询状态（后续版本会在GenAPI中添加统一的查询接口）。

**Q: 支持哪些图片格式？**

A: 支持常见的图片格式（JPEG、PNG、WebP等），具体取决于Provider。

**Q: 如何控制生成成本？**

A: 使用`TokenUsageRecorder`监控使用量，并在业务层实现配额控制。

**Q: 可以同时使用多个Provider吗？**

A: 可以，注册多个Provider后通过provider名称选择使用。

## 许可证

本项目采用 MIT 许可证。

## 贡献

欢迎提交Issue和Pull Request！

---

**注意**: 使用前请确保已获取各Provider的有效API密钥，并遵守相应的使用条款。

