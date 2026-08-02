# Vidu Go 客户端

这是 Vidu API 的 Go 客户端实现，支持图片转视频、视频风格转换、参考视频生成和开始结束图片转视频功能。

## 功能特性

- ✅ 图片转视频生成
- ✅ 视频风格转换
- ✅ 参考视频生成
- ✅ 开始结束图片转视频
- ✅ 任务状态查询
- ✅ 视频下载
- ✅ 完整的错误处理
- ✅ 参数验证
- ✅ 支持多种分辨率和帧率
- ✅ 支持多种视频风格
- ✅ 相似度和运动强度控制
- ✅ 过渡类型和速度控制

## 安装

```bash
go get github.com/grapery/grapery/pkg/cloud/vidu
```

## 快速开始

### 1. 创建客户端

```go
import "github.com/grapery/grapery/pkg/cloud/vidu"

// 使用默认配置
client := vidu.NewViduClient("your-api-key")

// 使用自定义配置
client := vidu.NewViduClientWithConfig("your-api-key", "https://api.vidu.cn/v1", 60*time.Second)
```

### 2. 图片转视频

```go
ctx := context.Background()

// 创建请求
req := &vidu.ImageToVideoRequest{
    ImageURL:       "https://example.com/image.jpg",
    Prompt:         "一个美丽的风景视频",
    Duration:       10,           // 10秒视频
    Resolution:     "1280x720",   // 720p 分辨率
    FrameRate:      24,           // 24fps
    Style:          "realistic",  // 写实风格
    NegativePrompt: "模糊，低质量",
}

// 提交任务
task, err := client.GenerateVideoFromImage(ctx, req)
if err != nil {
    log.Fatal(err)
}

// 等待任务完成
status, err := client.WaitForTaskCompletion(ctx, task.TaskID, 5*time.Second)
if err != nil {
    log.Fatal(err)
}

// 检查结果
if status.IsTaskSuccessful() {
    fmt.Printf("视频生成成功！URL: %s\n", status.Result.VideoURL)
    
    // 下载视频
    err = client.DownloadVideo(ctx, status.Result.VideoURL, "output.mp4")
    if err != nil {
        log.Fatal(err)
    }
} else {
    fmt.Printf("视频生成失败: %s\n", status.GetErrorMessage())
}
```

### 3. 视频风格转换

```go
// 创建请求
req := &vidu.VideoStyleTransferRequest{
    VideoURL:       "https://example.com/video.mp4",
    Prompt:         "转换为动漫风格",
    Resolution:     "1920x1080",
    FrameRate:      30,
    Style:          "anime",
    NegativePrompt: "写实，模糊",
}

// 提交任务
task, err := client.VideoStyleTransfer(ctx, req)
if err != nil {
    log.Fatal(err)
}

// 轮询任务状态
for {
    status, err := client.GetTaskStatus(ctx, task.TaskID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("任务状态: %s, 进度: %d%%\n", status.Status, status.Progress)
    
    if status.IsTaskCompleted() {
        if status.IsTaskSuccessful() {
            fmt.Printf("风格转换成功！URL: %s\n", status.Result.VideoURL)
        } else {
            fmt.Printf("风格转换失败: %s\n", status.GetErrorMessage())
        }
        break
    }
    
    time.Sleep(5 * time.Second)
}
```

### 4. 参考视频生成

```go
// 创建请求
req := &vidu.ReferenceToVideoRequest{
    ReferenceVideoURL: "https://example.com/reference-dance.mp4",
    Prompt:            "基于参考视频生成一个类似的舞蹈视频，保持相同的动作风格",
    Duration:          15,           // 15秒视频
    Resolution:        "1920x1080",  // 1080p 分辨率
    FrameRate:         30,           // 30fps
    Style:             "realistic",  // 写实风格
    Similarity:        0.8,          // 相似度 80%
    MotionStrength:    0.7,          // 运动强度 70%
    NegativePrompt:    "模糊，低质量，动作不自然",
}

// 提交任务
task, err := client.GenerateVideoFromReference(ctx, req)
if err != nil {
    log.Fatal(err)
}

// 等待任务完成
status, err := client.WaitForTaskCompletion(ctx, task.TaskID, 5*time.Second)
if err != nil {
    log.Fatal(err)
}

// 检查结果
if status.IsTaskSuccessful() {
    fmt.Printf("参考视频生成成功！URL: %s\n", status.Result.VideoURL)
    
    // 下载视频
    err = client.DownloadVideo(ctx, status.Result.VideoURL, "generated_dance.mp4")
    if err != nil {
        log.Fatal(err)
    }
} else {
    fmt.Printf("参考视频生成失败: %s\n", status.GetErrorMessage())
}
```

### 5. 开始结束图片转视频

```go
// 创建请求
req := &vidu.StartEndToVideoRequest{
    StartImageURL:    "https://example.com/start-image.jpg",
    EndImageURL:      "https://example.com/end-image.jpg",
    Prompt:           "从开始图片平滑过渡到结束图片，展现季节变化",
    Duration:         12,           // 12秒视频
    Resolution:       "1920x1080",  // 1080p 分辨率
    FrameRate:        30,           // 30fps
    Style:            "realistic",  // 写实风格
    TransitionType:   "smooth",     // 平滑过渡
    TransitionSpeed:  1.2,          // 过渡速度 1.2x
    MotionIntensity:  0.6,          // 运动强度 60%
    NegativePrompt:   "模糊，低质量，过渡不自然，跳跃",
}

// 提交任务
task, err := client.GenerateVideoFromStartEnd(ctx, req)
if err != nil {
    log.Fatal(err)
}

// 等待任务完成
status, err := client.WaitForTaskCompletion(ctx, task.TaskID, 5*time.Second)
if err != nil {
    log.Fatal(err)
}

// 检查结果
if status.IsTaskSuccessful() {
    fmt.Printf("开始结束图片转视频生成成功！URL: %s\n", status.Result.VideoURL)
    
    // 下载视频
    err = client.DownloadVideo(ctx, status.Result.VideoURL, "transition_video.mp4")
    if err != nil {
        log.Fatal(err)
    }
} else {
    fmt.Printf("开始结束图片转视频生成失败: %s\n", status.GetErrorMessage())
}
```

## API 参考

### 客户端方法

- `NewViduClient(apiKey string) *ViduClient` - 创建客户端
- `NewViduClientWithConfig(apiKey, baseURL string, timeout time.Duration) *ViduClient` - 使用自定义配置创建客户端

### 视频生成方法

- `GenerateVideoFromImage(ctx context.Context, req *ImageToVideoRequest) (*TaskResponse, error)` - 图片转视频
- `VideoStyleTransfer(ctx context.Context, req *VideoStyleTransferRequest) (*TaskResponse, error)` - 视频风格转换
- `GenerateVideoFromReference(ctx context.Context, req *ReferenceToVideoRequest) (*TaskResponse, error)` - 参考视频生成
- `GenerateVideoFromStartEnd(ctx context.Context, req *StartEndToVideoRequest) (*TaskResponse, error)` - 开始结束图片转视频

### 任务管理方法

- `GetTaskStatus(ctx context.Context, taskID string) (*TaskStatusResponse, error)` - 查询任务状态
- `WaitForTaskCompletion(ctx context.Context, taskID string, checkInterval time.Duration) (*TaskStatusResponse, error)` - 等待任务完成

### 工具方法

- `DownloadVideo(ctx context.Context, videoURL, filename string) error` - 下载视频
- `GetSupportedResolutions() []string` - 获取支持的分辨率
- `GetSupportedDurations() (int, int)` - 获取支持的时长范围
- `GetSupportedFrameRates() []int` - 获取支持的帧率
- `GetSupportedFormats() []string` - 获取支持的格式
- `GetSupportedStyles() []string` - 获取支持的风格

## 支持的参数

### 分辨率
- 640x360 (360p)
- 1280x720 (720p)
- 1920x1080 (1080p)
- 2560x1440 (1440p)
- 3840x2160 (4K)

### 时长
- 1-30 秒

### 帧率
- 24, 25, 30, 50, 60 fps

### 格式
- MP4, WebM, MOV

### 风格
- realistic (写实)
- anime (动漫)
- cartoon (卡通)
- oil_painting (油画)
- watercolor (水彩)
- sketch (素描)
- cyberpunk (赛博朋克)
- vintage (复古)
- fantasy (奇幻)
- documentary (纪录片)

### 参考视频生成参数
- **Similarity** (0.0-1.0): 相似度控制，默认 0.8
- **MotionStrength** (0.0-1.0): 运动强度控制，默认 0.5

### 开始结束图片转视频参数
- **TransitionType**: 过渡类型
  - `smooth` (平滑过渡)
  - `morph` (变形过渡)
  - `dissolve` (溶解过渡)
  - `fade` (淡入淡出)
- **TransitionSpeed** (0.1-2.0): 过渡速度控制，默认 1.0
- **MotionIntensity** (0.0-1.0): 运动强度控制，默认 0.5

## 错误处理

客户端提供了详细的错误信息：

```go
if err != nil {
    switch err {
    case vidu.ErrAPIKeyMissing:
        log.Println("API 密钥未设置")
    case vidu.ErrInvalidImageURL:
        log.Println("无效的图片 URL")
    case vidu.ErrInvalidVideoURL:
        log.Println("无效的视频 URL")
    case vidu.ErrInvalidReferenceVideoURL:
        log.Println("无效的参考视频 URL")
    case vidu.ErrInvalidStartImageURL:
        log.Println("无效的开始图片 URL")
    case vidu.ErrInvalidEndImageURL:
        log.Println("无效的结束图片 URL")
    case vidu.ErrEmptyPrompt:
        log.Println("提示词不能为空")
    case vidu.ErrInvalidDuration:
        log.Println("无效的视频时长")
    case vidu.ErrInvalidSimilarity:
        log.Println("无效的相似度值")
    case vidu.ErrInvalidMotionStrength:
        log.Println("无效的运动强度值")
    case vidu.ErrInvalidTransitionType:
        log.Println("无效的过渡类型")
    case vidu.ErrInvalidTransitionSpeed:
        log.Println("无效的过渡速度值")
    case vidu.ErrInvalidMotionIntensity:
        log.Println("无效的运动强度值")
    default:
        log.Printf("其他错误: %v", err)
    }
}
```

## 测试

运行测试：

```bash
go test ./pkg/cloud/vidu/ -v
```

## 许可证

本项目使用 MIT 许可证。

## 相关链接

- [Vidu 官方文档](https://platform.vidu.cn/docs)
- [Vidu API 文档](https://platform.vidu.cn/docs/image-to-video)
