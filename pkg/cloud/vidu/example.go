package vidu

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleBasicUsage 基本使用示例
func ExampleBasicUsage() {
	// 创建客户端
	client := NewViduClient("your-api-key-here")

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 图片转视频示例
	imageToVideoExample(ctx, client)

	// 参考视频生成示例
	referenceToVideoExample(ctx, client)

	// 开始结束图片转视频示例
	startEndToVideoExample(ctx, client)
}

// imageToVideoExample 图片转视频示例
func imageToVideoExample(ctx context.Context, client *ViduClient) {
	fmt.Println("=== 图片转视频示例 ===")

	// 创建请求
	req := &ImageToVideoRequest{
		ImageURL:       "https://example.com/beautiful-landscape.jpg",
		Prompt:         "一个美丽的风景视频，阳光明媚，微风轻拂",
		Duration:       10,          // 10秒视频
		Resolution:     "1280x720",  // 720p 分辨率
		FrameRate:      24,          // 24fps
		Style:          "realistic", // 写实风格
		NegativePrompt: "模糊，低质量，抖动",
	}

	// 提交任务
	task, err := client.GenerateVideoFromImage(ctx, req)
	if err != nil {
		log.Printf("提交图片转视频任务失败: %v", err)
		return
	}

	fmt.Printf("任务已提交，任务ID: %s\n", task.TaskID)

	// 等待任务完成
	status, err := client.WaitForTaskCompletion(ctx, task.TaskID, 5*time.Second)
	if err != nil {
		log.Printf("等待任务完成失败: %v", err)
		return
	}

	// 检查任务结果
	if status.IsTaskSuccessful() {
		fmt.Printf("视频生成成功！\n")
		fmt.Printf("视频URL: %s\n", status.Result.VideoURL)
		fmt.Printf("视频时长: %d秒\n", status.Result.Duration)
		fmt.Printf("文件大小: %d字节\n", status.Result.Size)
		fmt.Printf("视频格式: %s\n", status.Result.Format)

		// 下载视频
		err = client.DownloadVideo(ctx, status.Result.VideoURL, "output_video.mp4")
		if err != nil {
			log.Printf("下载视频失败: %v", err)
		} else {
			fmt.Println("视频已下载到 output_video.mp4")
		}
	} else {
		fmt.Printf("视频生成失败: %s\n", status.GetErrorMessage())
	}
}

// referenceToVideoExample 参考视频生成示例
func referenceToVideoExample(ctx context.Context, client *ViduClient) {
	fmt.Println("\n=== 参考视频生成示例 ===")

	// 创建请求
	req := &ReferenceToVideoRequest{
		ReferenceVideoURL: "https://example.com/reference-video.mp4",
		Prompt:            "基于参考视频生成一个类似的舞蹈视频，保持相同的动作风格",
		Duration:          15,          // 15秒视频
		Resolution:        "1920x1080", // 1080p 分辨率
		FrameRate:         30,          // 30fps
		Style:             "realistic", // 写实风格
		Similarity:        0.8,         // 相似度 80%
		MotionStrength:    0.7,         // 运动强度 70%
		NegativePrompt:    "模糊，低质量，动作不自然",
	}

	// 提交任务
	task, err := client.GenerateVideoFromReference(ctx, req)
	if err != nil {
		log.Printf("提交参考视频生成任务失败: %v", err)
		return
	}

	fmt.Printf("任务已提交，任务ID: %s\n", task.TaskID)

	// 等待任务完成
	status, err := client.WaitForTaskCompletion(ctx, task.TaskID, 5*time.Second)
	if err != nil {
		log.Printf("等待任务完成失败: %v", err)
		return
	}

	// 检查任务结果
	if status.IsTaskSuccessful() {
		fmt.Printf("参考视频生成成功！\n")
		fmt.Printf("视频URL: %s\n", status.Result.VideoURL)
		fmt.Printf("视频时长: %d秒\n", status.Result.Duration)
		fmt.Printf("文件大小: %d字节\n", status.Result.Size)
		fmt.Printf("视频格式: %s\n", status.Result.Format)

		// 下载视频
		err = client.DownloadVideo(ctx, status.Result.VideoURL, "reference_video.mp4")
		if err != nil {
			log.Printf("下载视频失败: %v", err)
		} else {
			fmt.Println("视频已下载到 reference_video.mp4")
		}
	} else {
		fmt.Printf("参考视频生成失败: %s\n", status.GetErrorMessage())
	}
}

// startEndToVideoExample 开始结束图片转视频示例
func startEndToVideoExample(ctx context.Context, client *ViduClient) {
	fmt.Println("\n=== 开始结束图片转视频示例 ===")

	// 创建请求
	req := &StartEndToVideoRequest{
		StartImageURL:   "https://example.com/start-image.jpg",
		EndImageURL:     "https://example.com/end-image.jpg",
		Prompt:          "从开始图片平滑过渡到结束图片，展现季节变化",
		Duration:        12,          // 12秒视频
		Resolution:      "1920x1080", // 1080p 分辨率
		FrameRate:       30,          // 30fps
		Style:           "realistic", // 写实风格
		TransitionType:  "smooth",    // 平滑过渡
		TransitionSpeed: 1.2,         // 过渡速度 1.2x
		MotionIntensity: 0.6,         // 运动强度 60%
		NegativePrompt:  "模糊，低质量，过渡不自然，跳跃",
	}

	// 提交任务
	task, err := client.GenerateVideoFromStartEnd(ctx, req)
	if err != nil {
		log.Printf("提交开始结束图片转视频任务失败: %v", err)
		return
	}

	fmt.Printf("任务已提交，任务ID: %s\n", task.TaskID)

	// 等待任务完成
	status, err := client.WaitForTaskCompletion(ctx, task.TaskID, 5*time.Second)
	if err != nil {
		log.Printf("等待任务完成失败: %v", err)
		return
	}

	// 检查任务结果
	if status.IsTaskSuccessful() {
		fmt.Printf("开始结束图片转视频生成成功！\n")
		fmt.Printf("视频URL: %s\n", status.Result.VideoURL)
		fmt.Printf("视频时长: %d秒\n", status.Result.Duration)
		fmt.Printf("文件大小: %d字节\n", status.Result.Size)
		fmt.Printf("视频格式: %s\n", status.Result.Format)

		// 下载视频
		err = client.DownloadVideo(ctx, status.Result.VideoURL, "start_end_video.mp4")
		if err != nil {
			log.Printf("下载视频失败: %v", err)
		} else {
			fmt.Println("视频已下载到 start_end_video.mp4")
		}
	} else {
		fmt.Printf("开始结束图片转视频生成失败: %s\n", status.GetErrorMessage())
	}
}

// ExampleVideoStyleTransfer 视频风格转换示例
func ExampleVideoStyleTransfer() {
	// 创建客户端
	client := NewViduClient("your-api-key-here")

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 创建请求
	req := &VideoStyleTransferRequest{
		VideoURL:       "https://example.com/original-video.mp4",
		Prompt:         "转换为动漫风格，色彩鲜艳，线条清晰",
		Resolution:     "1920x1080", // 1080p 分辨率
		FrameRate:      30,          // 30fps
		Style:          "anime",     // 动漫风格
		NegativePrompt: "写实，模糊，低质量",
	}

	// 提交任务
	task, err := client.VideoStyleTransfer(ctx, req)
	if err != nil {
		log.Printf("提交视频风格转换任务失败: %v", err)
		return
	}

	fmt.Printf("任务已提交，任务ID: %s\n", task.TaskID)

	// 轮询任务状态
	for {
		status, err := client.GetTaskStatus(ctx, task.TaskID)
		if err != nil {
			log.Printf("查询任务状态失败: %v", err)
			break
		}

		fmt.Printf("任务状态: %s, 进度: %d%%\n", status.Status, status.Progress)

		if status.IsTaskCompleted() {
			if status.IsTaskSuccessful() {
				fmt.Printf("风格转换成功！\n")
				fmt.Printf("视频URL: %s\n", status.Result.VideoURL)
				fmt.Printf("视频时长: %d秒\n", status.Result.Duration)
				fmt.Printf("文件大小: %d字节\n", status.Result.Size)
				fmt.Printf("视频格式: %s\n", status.Result.Format)

				// 下载视频
				err = client.DownloadVideo(ctx, status.Result.VideoURL, "styled_video.mp4")
				if err != nil {
					log.Printf("下载视频失败: %v", err)
				} else {
					fmt.Println("视频已下载到 styled_video.mp4")
				}
			} else {
				fmt.Printf("风格转换失败: %s\n", status.GetErrorMessage())
			}
			break
		}

		// 等待5秒后再次查询
		time.Sleep(5 * time.Second)
	}
}

// ExampleGetSupportedOptions 获取支持的选项示例
func ExampleGetSupportedOptions() {
	fmt.Println("=== 支持的选项示例 ===")

	client := NewViduClient("your-api-key-here")

	// 获取支持的分辨率
	resolutions := client.GetSupportedResolutions()
	fmt.Printf("支持的分辨率: %v\n", resolutions)

	// 获取支持的时长范围
	minDur, maxDur := client.GetSupportedDurations()
	fmt.Printf("支持的时长范围: %d-%d秒\n", minDur, maxDur)

	// 获取支持的帧率
	frameRates := client.GetSupportedFrameRates()
	fmt.Printf("支持的帧率: %v\n", frameRates)

	// 获取支持的格式
	formats := client.GetSupportedFormats()
	fmt.Printf("支持的格式: %v\n", formats)

	// 获取支持的风格
	styles := client.GetSupportedStyles()
	fmt.Printf("支持的风格: %v\n", styles)
}

// ExampleReferenceToVideo 参考视频生成示例
func ExampleReferenceToVideo() {
	// 创建客户端
	client := NewViduClient("your-api-key-here")

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 创建请求
	req := &ReferenceToVideoRequest{
		ReferenceVideoURL: "https://example.com/dance-reference.mp4",
		Prompt:            "基于参考视频生成一个类似的舞蹈视频，保持相同的动作风格和节奏",
		Duration:          20,          // 20秒视频
		Resolution:        "1920x1080", // 1080p 分辨率
		FrameRate:         30,          // 30fps
		Style:             "realistic", // 写实风格
		Similarity:        0.9,         // 高相似度 90%
		MotionStrength:    0.8,         // 高运动强度 80%
		NegativePrompt:    "模糊，低质量，动作不自然，节奏不一致",
	}

	// 提交任务
	task, err := client.GenerateVideoFromReference(ctx, req)
	if err != nil {
		log.Printf("提交参考视频生成任务失败: %v", err)
		return
	}

	fmt.Printf("任务已提交，任务ID: %s\n", task.TaskID)

	// 轮询任务状态
	for {
		status, err := client.GetTaskStatus(ctx, task.TaskID)
		if err != nil {
			log.Printf("查询任务状态失败: %v", err)
			break
		}

		fmt.Printf("任务状态: %s, 进度: %d%%\n", status.Status, status.Progress)

		if status.IsTaskCompleted() {
			if status.IsTaskSuccessful() {
				fmt.Printf("参考视频生成成功！\n")
				fmt.Printf("视频URL: %s\n", status.Result.VideoURL)
				fmt.Printf("视频时长: %d秒\n", status.Result.Duration)
				fmt.Printf("文件大小: %d字节\n", status.Result.Size)
				fmt.Printf("视频格式: %s\n", status.Result.Format)

				// 下载视频
				err = client.DownloadVideo(ctx, status.Result.VideoURL, "generated_dance.mp4")
				if err != nil {
					log.Printf("下载视频失败: %v", err)
				} else {
					fmt.Println("视频已下载到 generated_dance.mp4")
				}
			} else {
				fmt.Printf("参考视频生成失败: %s\n", status.GetErrorMessage())
			}
			break
		}

		// 等待5秒后再次查询
		time.Sleep(5 * time.Second)
	}
}
