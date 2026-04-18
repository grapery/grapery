# GenAPI - 统一媒体生成代理层

GenAPI 是一个统一的图片/视频生成代理层，提供了一致的接口来调用多个AI媒体生成服务提供商（Hailuo、Huoshan、Gemini），使业务层无需关心底层SDK的差异。

## 特性

- 🎯 **统一接口**：统一的请求/响应格式，简化业务层调用
- 🔌 **多Provider支持**：支持 Hailuo、Huoshan (火山引擎)、Gemini (Google)、Qwen (阿里云)
- 🎨 **图片生成**：文本生图、图生图、图片编辑
- 🎬 **视频生成**：文本生视频、图生视频、关键帧生视频、分镜生视频
- 📊 **Token统计**：自动记录和上报Token使用量
- ⚡ **异步任务**：支持长时间运行的生成任务，含轮询等待机制
- 🛡️ **类型安全**：完整的Go类型定义
- 📥 **视频下载**：统一的视频下载接口
- 🔄 **状态标准化**：跨Provider的统一状态表示

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Business Layer (业务层)                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ Story Service│  │Video Service│  │Image Service│  │ Callback Handler       │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘ │
└─────────┼────────────────┼────────────────┼─────────────────────┼───────────────┘
          │                │                │                     │
          ▼                ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                               GenAPI (统一代理层)                                │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                              GenAPI Struct                                │  │
│  │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────────────────┐ │  │
│  │  │ providers       │ │ imageProviders  │ │ videoProviders              │ │  │
│  │  │ map[string]     │ │ map[string]     │ │ map[string]VideoProvider    │ │  │
│  │  │ Provider        │ │ ImageProvider   │ │                             │ │  │
│  │  └─────────────────┘ └─────────────────┘ └─────────────────────────────┘ │  │
│  │  ┌─────────────────────────────────────────────────────────────────────┐ │  │
│  │  │                    TokenUsageRecorder                               │ │  │
│  │  └─────────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                         Public Methods                                   │   │
│  │  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────────────┐ │   │
│  │  │ GenerateImage()  │ │ GenerateVideo()  │ │ GetVideoStatus()         │ │   │
│  │  └──────────────────┘ └──────────────────┘ └──────────────────────────┘ │   │
│  │  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────────────┐ │   │
│  │  │ DownloadVideo()  │ │ WaitForVideo()   │ │ RegisterProvider()       │ │   │
│  │  └──────────────────┘ └──────────────────┘ └──────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────┘
          │                │                │                     │
          ▼                ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           Provider Adapters (适配器层)                           │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌────────────────┐   │
│  │ hailuoProvider │ │huoshanProvider │ │ geminiProvider │ │ qwenProvider   │   │
│  │                │ │                │ │                │ │                │   │
│  │ ├─Name()       │ │ ├─Name()       │ │ ├─Name()       │ │ ├─Name()       │   │
│  │ ├─Generate()   │ │ ├─Generate()   │ │ ├─Generate()   │ │ ├─Generate()   │   │
│  │ ├─GenerateImg()│ │ ├─GenerateImg()│ │ ├─GenerateImg()│ │ ├─GenerateImg()│   │
│  │ ├─GetStatus()  │ │ ├─GetStatus()  │ │ ├─GetStatus()  │ │ ├─GetStatus()  │   │
│  │ └─Download()   │ │ └─Download()   │ │ └─Download()   │ │ └─Download()   │   │
│  └───────┬────────┘ └───────┬────────┘ └───────┬────────┘ └───────┬────────┘   │
└──────────┼──────────────────┼──────────────────┼──────────────────┼─────────────┘
           │                  │                  │                  │
           ▼                  ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         Provider Clients (HTTP客户端层)                          │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌────────────────┐   │
│  │ hailuo.Client  │ │huoshan.Client  │ │ gemini.Client  │ │  qwen.Client   │   │
│  │                │ │                │ │                │ │                │   │
│  │ HTTP + JSON    │ │ ArkRuntime SDK │ │ GenAI SDK      │ │ HTTP + JSON    │   │
│  └───────┬────────┘ └───────┬────────┘ └───────┬────────┘ └───────┬────────┘   │
└──────────┼──────────────────┼──────────────────┼──────────────────┼─────────────┘
           │                  │                  │                  │
           ▼                  ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            External APIs (外部API)                               │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌────────────────┐   │
│  │   MiniMax API  │ │ Volcengine API │ │   Google API   │ │  DashScope API │   │
│  │  (api.minimax  │ │ (ark.cn-beijing│ │ (generativelang│ │  (dashscope    │   │
│  │   .com)        │ │  .volces.com)  │ │  uage.google..│ │   .aliyuncs..) │   │
│  └────────────────┘ └────────────────┘ └────────────────┘ └────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 请求处理流程

```
┌───────────┐      ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│ Business  │─────▶│  GenAPI      │─────▶│  Provider    │─────▶│  External    │
│  Layer    │      │ GenerateVideo│      │   Adapter    │      │    API       │
└───────────┘      └──────────────┘      └──────────────┘      └──────────────┘
      │                   │                     │                     │
      │  GenerateRequest  │                     │                     │
      │  ┌─────────────┐  │                     │                     │
      │  │ Prompt      │  │                     │                     │
      │  │ Operation   │  │                     │                     │
      │  │ Model       │  │                     │                     │
      │  │ Options...  │  │                     │                     │
      │  └─────────────┘  │                     │                     │
      │                   │                     │                     │
      │                   ▼                     │                     │
      │            ┌──────────────┐             │                     │
      │            │ 1. Clone req │             │                     │
      │            │ 2. Auto-infer│             │                     │
      │            │    Operation │             │                     │
      │            │ 3. Lookup    │             │                     │
      │            │    Provider  │             │                     │
      │            └──────┬───────┘             │                     │
      │                   │                     │                     │
      │                   ▼                     ▼                     │
      │            ┌──────────────┐      ┌──────────────┐             │
      │            │ provider.    │─────▶│ Build native │             │
      │            │ Generate()   │      │  API payload │             │
      │            └──────────────┘      └──────┬───────┘             │
      │                                         │                     │
      │                                         ▼                     ▼
      │                                  ┌──────────────┐      ┌──────────────┐
      │                                  │  HTTP/SDK    │─────▶│   API Call   │
      │                                  │   Client     │      │              │
      │                                  └──────────────┘      └──────┬───────┘
      │                                                               │
      │                                         ┌─────────────────────┘
      │                                         ▼
      │                                  ┌──────────────┐
      │                                  │  Native Resp │
      │                                  │  (TaskID,    │
      │                                  │   Status...) │
      │                                  └──────┬───────┘
      │                                         │
      │                   ┌─────────────────────┘
      │                   ▼
      │            ┌──────────────┐
      │            │ Build        │
      │            │ GenerateResp │
      │            │ + Normalize  │
      │            │   Status     │
      │            └──────┬───────┘
      │                   │
      │                   ▼
      │            ┌──────────────┐
      │            │ Record Usage │
      │            │ (if enabled) │
      │            └──────┬───────┘
      │                   │
◀─────┴───────────────────┘
GenerateResponse
```

### 接口层次关系

```
                              ┌─────────────────┐
                              │    Provider     │
                              │  └─ Name()      │
                              └────────┬────────┘
                                       │
           ┌───────────────────────────┼───────────────────────────┐
           │                           │                           │
           ▼                           ▼                           ▼
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   ImageProvider     │    │   VideoGenerator    │    │ VideoStatusFetcher  │
│ ├─ Provider         │    │ └─ Generate()       │    │ └─ GetVideoStatus() │
│ └─ GenerateImage()  │    └─────────┬───────────┘    └─────────┬───────────┘
└─────────────────────┘              │                          │
                                     └──────────┬───────────────┘
                                                │
                                                ▼
                                    ┌─────────────────────┐
                                    │   VideoProvider     │
                                    │ ├─ Provider         │
                                    │ ├─ VideoGenerator   │
                                    │ └─ VideoStatusFetch │
                                    └─────────────────────┘

Optional Interfaces:
┌─────────────────────┐    ┌─────────────────────┐
│  VideoDownloader    │    │ ImageStatusFetcher  │
│ └─ DownloadVideo()  │    │ └─ GetImageStatus() │
└─────────────────────┘    └─────────────────────┘
```

### 功能支持矩阵

| Operation              | Hailuo | Huoshan | Gemini | Qwen |
|------------------------|:------:|:-------:|:------:|:----:|
| text_to_image          |   ✓    |    ✓    |   ✓    |  ✓   |
| image_to_image         |   ✗    |    ✓    |   ✓    |  ✓   |
| text_to_video          |   ✓    |    ✓    |   ✓    |  ✓   |
| image_to_video         |   ✓    |    ✓    |   ✓    |  ✓   |
| keyframe_to_video      |   ✓    |    ✓    |   ✗    |  ✓   |
| storyboard_to_video    |   ✓    |    ✗    |   ✗    |  ✗   |
| GetVideoStatus         |   ✓    |    ✓    |   ✓    |  ✓   |
| GetImageStatus         |   ✗    |    ✗    |   ✗    |  ✓   |
| DownloadVideo          |   ✓    |    ✓    |   ✓    |  ✓   |
| Usage Tracking         |   ✓    |    ✓    |   ✓    |  ✓   |

### 数据类型关系

```
                    ┌──────────────────────────────────────────┐
                    │            GenerateRequest               │
                    │ ┌──────────────────────────────────────┐ │
                    │ │ Operation: OperationType             │ │
                    │ │ Mode: GenerationMode                 │ │
                    │ │ Prompt, NegativePrompt               │ │
                    │ │ AspectRatio, Resolution              │ │
                    │ │ DurationSeconds, Quality             │ │
                    │ │ Model, CallbackURL                   │ │
                    │ │ ReferenceImageURL, ReferenceImages   │ │
                    │ │ FirstFrameURL, LastFrameURL          │ │
                    │ │ ImageData, VideoData                 │ │
                    │ │ Storyboard, Options, Metadata        │ │
                    │ └──────────────────────────────────────┘ │
                    └──────────────────────────────────────────┘
                                        │
                                        ▼
                    ┌──────────────────────────────────────────┐
                    │            GenerateResponse              │
                    │ ┌──────────────────────────────────────┐ │
                    │ │ Provider: string                     │ │
                    │ │ Operation: OperationType             │ │
                    │ │ MediaType: MediaType (image/video)   │ │
                    │ │ TaskID, Status (normalized)          │ │
                    │ │ Message, Progress                    │ │
                    │ │ ImageURLs, VideoURL, ThumbnailURL    │ │
                    │ │ Error, ErrorCode                     │ │
                    │ │ Usage: *Usage                        │ │
                    │ │ Metadata, Raw                        │ │
                    │ │ StartedAt, CompletedAt               │ │
                    │ └──────────────────────────────────────┘ │
                    └──────────────────────────────────────────┘

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  OperationType  │     │    MediaType    │     │   TaskStatus    │
│ ├─TextToImage   │     │ ├─image         │     │ ├─pending       │
│ ├─ImageToImage  │     │ └─video         │     │ ├─processing    │
│ ├─TextToVideo   │     └─────────────────┘     │ ├─completed     │
│ ├─ImageToVideo  │                             │ ├─failed        │
│ ├─KeyframeToV   │                             │ └─cancelled     │
│ └─StoryboardToV │                             └─────────────────┘
└─────────────────┘
```

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
    Model:    "veo-3.1-generate-preview",  // 视频模型 (Veo 3.1)
    Timeout:  60 * time.Second,
}
```

**默认模型**（未指定时使用）：
- 文本：`gemini-2.5-flash`
- 图片：`gemini-2.5-flash-image`（conversational API）
- 视频：`veo-3.1-generate-preview`

**支持的操作**：
- ✅ 文本生视频 (text_to_video)
- ✅ 图生视频 (image_to_video)
- ✅ 文本生图 (text_to_image)
- ✅ 图生图 (image_to_image) - 通过conversational API

### Qwen (阿里云通义)

```go
config := &genapi.Config{
    Provider: genapi.ProviderQwen,
    APIKey:   "your-dashscope-api-key",
    BaseURL:  "https://dashscope.aliyuncs.com",  // 可选，默认值
    Model:    "qwen-video-1",      // 视频模型
    Timeout:  60 * time.Second,
}
```

**支持的操作**：
- ✅ 文本生视频 (text_to_video)
- ✅ 图生视频 (image_to_video)
- ✅ 关键帧生视频 (keyframe_to_video)
- ✅ 文本生图 (text_to_image)
- ✅ 图生图 (image_to_image)

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

### 示例7：分镜生成视频

```go
storyboard := map[string]interface{}{
    "scenes": []map[string]interface{}{
        {
            "prompt":   "场景1：日落时分的海边",
            "duration": 3,
        },
        {
            "prompt":   "场景2：海浪拍打岩石",
            "duration": 3,
        },
    },
}

req := &genapi.GenerateRequest{
    Operation:  genapi.OperationStoryboardToVideo,
    Storyboard: storyboard,
}

resp, err := api.GenerateVideo(ctx, "hailuo", req)
```

### 示例8：Qwen 图生视频

```go
req := &genapi.GenerateRequest{
    Operation:         genapi.OperationImageToVideo,
    ReferenceImageURL: "https://example.com/portrait.jpg",
    Prompt:            "让人物自然地微笑",
    DurationSeconds:   8,
}

resp, err := api.GenerateVideo(ctx, "qwen", req)
```

## 状态标准化

GenAPI 提供统一的任务状态表示，自动将各Provider的原始状态转换为标准状态：

| TaskStatus | 说明 | 原始状态示例 |
|------------|------|------------|
| `pending` | 等待处理 | pending, queued, waiting, submitted |
| `processing` | 处理中 | processing, running, in_progress, started |
| `completed` | 已完成 | completed, succeeded, success, done, finished |
| `failed` | 失败 | failed, error, failure |
| `cancelled` | 已取消 | cancelled, canceled, aborted |

```go
// 使用状态辅助函数
if genapi.IsTerminalStatus(resp.Status) {
    // 任务已结束（完成、失败或取消）
}

if genapi.IsPendingOrProcessing(resp.Status) {
    // 任务仍在进行中
}
```

## 视频下载与等待

### 下载视频

```go
// 下载生成的视频二进制数据
data, err := api.DownloadVideo(ctx, "hailuo", taskID)
if err != nil {
    log.Fatal(err)
}
os.WriteFile("output.mp4", data, 0644)
```

### 等待任务完成

```go
// 轮询等待视频任务完成，每5秒检查一次
resp, err := api.WaitForVideo(ctx, "hailuo", taskID, 5*time.Second)
if err != nil {
    log.Fatal(err)
}

if resp.Status == string(genapi.StatusCompleted) {
    fmt.Printf("视频URL: %s\n", resp.VideoURL)
} else {
    fmt.Printf("任务失败: %s\n", resp.Error)
}
```

### 图片任务状态查询

对于异步图片生成（如 Qwen），可以查询任务状态：

```go
// 查询图片生成任务状态
resp, err := api.GetImageStatus(ctx, "qwen", taskID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("状态: %s, 图片数: %d\n", resp.Status, len(resp.ImageURLs))

// 或等待任务完成
resp, err := api.WaitForImage(ctx, "qwen", taskID, 3*time.Second)
```

## 日志追踪

GenAPI 使用项目级的 zap 日志系统：

```go
import (
    "github.com/grapestree/fgrapery/grapery/internal/telemetry"
    "github.com/grapestree/fgrapery/grapery/internal/genai"
)

// 创建项目级 logger
logger, _ := telemetry.NewLogger("debug")

// 设置给 GenAPI 使用
genapi.SetLogger(logger)

// 所有 API 调用都会自动记录日志
// 输出示例:
// {"ts":"...","level":"info","msg":"GenerateVideo","provider":"hailuo","operation":"text_to_video",...}
// {"ts":"...","level":"info","msg":"GenerateVideo completed","provider":"hailuo","task_id":"xxx","duration_ms":1234}
```

不设置 logger 时，日志功能自动禁用。

## Token使用统计

### 设置全局Token记录器

```go
// 在应用启动时设置（新接口支持 req/rsp/err，可记录成功和失败）
genapi.SetTokenUsageRecorder(genapi.TokenUsageRecorderFunc(
    func(ctx context.Context, req *genapi.GenerateRequest, rsp *genapi.GenerateResponse, err error) {
        if rsp != nil && rsp.Usage != nil {
            log.Printf("Provider: %s, Tokens: %d, Images: %d, Videos: %d",
                rsp.Provider, rsp.Usage.TotalTokens, rsp.Usage.ImageCount, rsp.Usage.VideoCount)
        }
        // 保存到数据库（含 user_id、related_entity_id 等，从 req.Metadata 或 context 获取）
        saveUsageToDatabase(ctx, req, rsp, err)
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

