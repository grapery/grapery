# Vidu Go 客户端功能总结

## 已实现的功能

### 1. 图片转视频 (Image-to-Video)
- **API 端点**: `/image-to-video`
- **功能**: 根据单张图片生成动态视频
- **主要参数**:
  - `image_url`: 图片 URL
  - `prompt`: 视频描述提示词
  - `duration`: 视频时长 (1-30秒)
  - `resolution`: 视频分辨率
  - `frame_rate`: 帧率
  - `style`: 视频风格
  - `negative_prompt`: 负面提示词

### 2. 视频风格转换 (Video Style Transfer)
- **API 端点**: `/video-style-transfer`
- **功能**: 将现有视频转换为不同风格
- **主要参数**:
  - `video_url`: 视频 URL
  - `prompt`: 风格描述提示词
  - `resolution`: 视频分辨率
  - `frame_rate`: 帧率
  - `style`: 视频风格
  - `negative_prompt`: 负面提示词

### 3. 参考视频生成 (Reference-to-Video)
- **API 端点**: `/reference-to-video`
- **功能**: 基于参考视频生成新的视频
- **主要参数**:
  - `reference_video_url`: 参考视频 URL
  - `prompt`: 视频描述提示词
  - `duration`: 视频时长 (1-30秒)
  - `similarity`: 相似度控制 (0.0-1.0)
  - `motion_strength`: 运动强度 (0.0-1.0)

### 4. 开始结束图片转视频 (Start-End-to-Video) ✨ 新增
- **API 端点**: `/start-end-to-video`
- **功能**: 根据开始和结束图片生成过渡视频
- **主要参数**:
  - `start_image_url`: 开始图片 URL
  - `end_image_url`: 结束图片 URL
  - `prompt`: 视频描述提示词
  - `duration`: 视频时长 (1-30秒)
  - `transition_type`: 过渡类型 (smooth, morph, dissolve, fade)
  - `transition_speed`: 过渡速度 (0.1-2.0)
  - `motion_intensity`: 运动强度 (0.0-1.0)

## 支持的参数

### 分辨率
- 640x360 (360p)
- 1280x720 (720p)
- 1920x1080 (1080p)
- 2560x1440 (1440p)
- 3840x2160 (4K)

### 视频风格
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

### 过渡类型 (开始结束图片转视频)
- smooth (平滑过渡)
- morph (变形过渡)
- dissolve (溶解过渡)
- fade (淡入淡出)

### 视频格式
- MP4
- WebM
- MOV

### 帧率
- 24, 25, 30, 50, 60 fps

## 核心特性

### 1. 完整的参数验证
- 自动验证所有输入参数
- 自动修正无效参数为默认值
- 详细的错误信息

### 2. 异步任务处理
- 支持长时间运行的视频生成任务
- 任务状态查询和轮询
- 任务完成等待机制

### 3. 错误处理
- 详细的错误类型定义
- 友好的错误消息
- 完整的错误处理示例

### 4. 视频下载
- 自动下载生成的视频到本地
- 支持自定义文件名
- 支持流式下载

### 5. 工具方法
- 获取支持的参数选项
- 任务状态检查方法
- 客户端配置方法

## 使用示例

### 基本使用
```go
// 创建客户端
client := vidu.NewViduClient("your-api-key")

// 开始结束图片转视频
req := &vidu.StartEndToVideoRequest{
    StartImageURL:    "https://example.com/start.jpg",
    EndImageURL:      "https://example.com/end.jpg",
    Prompt:           "从春天过渡到秋天",
    Duration:         10,
    Resolution:       "1920x1080",
    FrameRate:        30,
    TransitionType:   "smooth",
    TransitionSpeed:  1.0,
    MotionIntensity:  0.5,
}

// 提交任务
task, err := client.GenerateVideoFromStartEnd(ctx, req)
if err != nil {
    log.Fatal(err)
}

// 等待完成
status, err := client.WaitForTaskCompletion(ctx, task.TaskID, 5*time.Second)
if err != nil {
    log.Fatal(err)
}

// 下载视频
if status.IsTaskSuccessful() {
    client.DownloadVideo(ctx, status.Result.VideoURL, "output.mp4")
}
```

## 测试覆盖

- ✅ 客户端创建测试
- ✅ 参数验证测试
- ✅ 错误处理测试
- ✅ 任务状态处理测试
- ✅ 所有四种视频生成模式的测试

## 文档

- 完整的 API 文档
- 详细的使用示例
- 错误处理指南
- 参数说明文档

## 相关链接

- [Vidu 官方文档](https://platform.vidu.cn/docs)
- [开始结束图片转视频 API](https://platform.vidu.cn/docs/start-end-to-video)
- [参考视频生成 API](https://platform.vidu.cn/docs/reference-to-video)
- [图片转视频 API](https://platform.vidu.cn/docs/image-to-video)
