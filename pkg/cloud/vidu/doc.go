// Package vidu 提供 Vidu API 的 Go 客户端实现
//
// Vidu 是一个强大的 AI 视频生成平台，支持图片转视频、视频风格转换、参考视频生成和开始结束图片转视频功能。
// 本包提供了完整的 API 封装，包括：
//
// - 图片转视频生成
// - 视频风格转换
// - 参考视频生成
// - 开始结束图片转视频
// - 任务状态查询
// - 视频下载
// - 错误处理
//
// 基本使用示例：
//
//	// 创建客户端
//	client := vidu.NewViduClient("your-api-key")
//
//	// 图片转视频
//	req := &vidu.ImageToVideoRequest{
//		ImageURL:   "https://example.com/image.jpg",
//		Prompt:     "一个美丽的风景视频",
//		Duration:   10,
//		Resolution: "1280x720",
//		FrameRate:  24,
//		Style:      "realistic",
//	}
//
//	task, err := client.GenerateVideoFromImage(ctx, req)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 等待任务完成
//	status, err := client.WaitForTaskCompletion(ctx, task.TaskID, 5*time.Second)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 下载视频
//	if status.IsTaskSuccessful() {
//		err = client.DownloadVideo(ctx, status.Result.VideoURL, "output.mp4")
//		if err != nil {
//			log.Fatal(err)
//		}
//	}
//
//	// 参考视频生成示例
//	refReq := &vidu.ReferenceToVideoRequest{
//		ReferenceVideoURL: "https://example.com/reference.mp4",
//		Prompt:            "基于参考视频生成新视频",
//		Duration:          15,
//		Resolution:        "1920x1080",
//		FrameRate:         30,
//		Similarity:        0.8,
//		MotionStrength:    0.7,
//	}
//
//	refTask, err := client.GenerateVideoFromReference(ctx, refReq)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 等待任务完成并下载
//	refStatus, err := client.WaitForTaskCompletion(ctx, refTask.TaskID, 5*time.Second)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if refStatus.IsTaskSuccessful() {
//		client.DownloadVideo(ctx, refStatus.Result.VideoURL, "reference_output.mp4")
//	}
//
//	// 开始结束图片转视频示例
//	startEndReq := &vidu.StartEndToVideoRequest{
//		StartImageURL:    "https://example.com/start.jpg",
//		EndImageURL:      "https://example.com/end.jpg",
//		Prompt:           "从开始图片平滑过渡到结束图片",
//		Duration:         10,
//		Resolution:       "1920x1080",
//		FrameRate:        30,
//		TransitionType:   "smooth",
//		TransitionSpeed:  1.0,
//		MotionIntensity:  0.5,
//	}
//
//	startEndTask, err := client.GenerateVideoFromStartEnd(ctx, startEndReq)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 等待任务完成并下载
//	startEndStatus, err := client.WaitForTaskCompletion(ctx, startEndTask.TaskID, 5*time.Second)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if startEndStatus.IsTaskSuccessful() {
//		client.DownloadVideo(ctx, startEndStatus.Result.VideoURL, "start_end_output.mp4")
//	}
//
// 更多信息请参考：https://platform.vidu.cn/docs
package vidu
