package genapi_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/grapery/grapery/pkg/genapi"
)

// Example_basic 展示基本的使用方式
func Example_basic() {
	// 创建GenAPI实例
	api := genapi.NewGenAPI()

	// 配置Hailuo provider
	config := &genapi.Config{
		Provider: genapi.ProviderHailuo,
		APIKey:   os.Getenv("HAILUO_API_KEY"),
		BaseURL:  "https://api.minimax.com",
		Model:    "MiniMax-Hailuo-02",
		Timeout:  30 * time.Second,
	}

	// 注册provider
	_, err := api.RegisterProviderConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// 生成视频
	ctx := context.Background()
	req := &genapi.GenerateRequest{
		Prompt:          "一只可爱的小猫在草地上玩耍",
		DurationSeconds: 6,
		AspectRatio:     "16:9",
	}

	resp, err := api.GenerateVideo(ctx, "hailuo", req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("任务ID: %s\n", resp.TaskID)
	fmt.Printf("状态: %s\n", resp.Status)
}

// Example_textToVideo 展示文本生成视频
func Example_textToVideo() {
	api := genapi.NewGenAPI()

	// 注册provider
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHailuo,
		APIKey:   os.Getenv("HAILUO_API_KEY"),
	})

	ctx := context.Background()

	// 文本生成视频
	req := &genapi.GenerateRequest{
		Operation:       genapi.OperationTextToVideo,
		Prompt:          "夕阳下的海滩，海浪轻拍岸边，海鸥在天空飞翔",
		DurationSeconds: 6,
		AspectRatio:     "16:9",
		Quality:         "high",
		Metadata: map[string]interface{}{
			"user_id": 12345,
			"scene":   "beach",
		},
	}

	resp, err := api.GenerateVideo(ctx, "hailuo", req)
	if err != nil {
		log.Fatalf("生成视频失败: %v", err)
	}

	fmt.Printf("视频生成任务已创建\n")
	fmt.Printf("任务ID: %s\n", resp.TaskID)
	fmt.Printf("状态: %s\n", resp.Status)
	fmt.Printf("Provider: %s\n", resp.Provider)

	if resp.Usage != nil {
		fmt.Printf("Token使用: %d\n", resp.Usage.TotalTokens)
	}
}

// Example_imageToVideo 展示图片生成视频
func Example_imageToVideo() {
	api := genapi.NewGenAPI()
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHuoshan,
		APIKey:   os.Getenv("HUOSHAN_API_KEY"),
	})

	ctx := context.Background()

	// 从URL生成视频
	req := &genapi.GenerateRequest{
		Operation:         genapi.OperationImageToVideo,
		ReferenceImageURL: "https://example.com/input.jpg",
		Prompt:            "让图片中的人物动起来，微笑并挥手",
		DurationSeconds:   6,
		AspectRatio:       "16:9",
	}

	resp, err := api.GenerateVideo(ctx, "huoshan", req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("图生视频任务ID: %s\n", resp.TaskID)
}

// Example_keyframeToVideo 展示关键帧生成视频
func Example_keyframeToVideo() {
	api := genapi.NewGenAPI()
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHailuo,
		APIKey:   os.Getenv("HAILUO_API_KEY"),
	})

	ctx := context.Background()

	// 关键帧生成视频
	req := &genapi.GenerateRequest{
		Operation:       genapi.OperationKeyframeToVideo,
		FirstFrameURL:   "https://example.com/frame1.jpg",
		LastFrameURL:    "https://example.com/frame2.jpg",
		Prompt:          "平滑过渡，保持连贯性",
		DurationSeconds: 6,
	}

	resp, err := api.GenerateVideo(ctx, "hailuo", req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("关键帧视频任务ID: %s\n", resp.TaskID)
}

// Example_textToImage 展示文本生成图片
func Example_textToImage() {
	api := genapi.NewGenAPI()
	api.RegisterProviderConfig(&genapi.Config{
		Provider:   genapi.ProviderGemini,
		APIKey:     os.Getenv("GEMINI_API_KEY"),
		ImageModel: "gemini-2.5-flash-image",
	})

	ctx := context.Background()

	// 文本生成图片
	req := &genapi.GenerateRequest{
		Operation:   genapi.OperationTextToImage,
		Prompt:      "一只戴着墨镜的猫，赛博朋克风格，霓虹灯背景",
		AspectRatio: "1:1",
		OutputCount: 4,
		Style:       "cyberpunk",
	}

	resp, err := api.GenerateImage(ctx, "gemini", req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("生成了 %d 张图片\n", len(resp.ImageURLs))
	for i, url := range resp.ImageURLs {
		fmt.Printf("图片%d: %s\n", i+1, url)
	}

	if resp.Usage != nil {
		fmt.Printf("图片数量: %d\n", resp.Usage.ImageCount)
	}
}

// Example_imageToImage 展示图片编辑/图生图
func Example_imageToImage() {
	api := genapi.NewGenAPI()
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHuoshan,
		APIKey:   os.Getenv("HUOSHAN_API_KEY"),
	})

	ctx := context.Background()

	// 图生图
	req := &genapi.GenerateRequest{
		Operation:         genapi.OperationImageToImage,
		ReferenceImageURL: "https://example.com/base.jpg",
		Prompt:            "将背景改成雪山，保持主体不变",
		AspectRatio:       "16:9",
		OutputCount:       1,
	}

	resp, err := api.GenerateImage(ctx, "huoshan", req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("编辑后的图片: %s\n", resp.ImageURLs[0])
}

// Example_multipleProviders 展示使用多个Provider
func Example_multipleProviders() {
	api := genapi.NewGenAPI()

	// 注册多个provider
	providers := []struct {
		kind genapi.ProviderKind
		key  string
	}{
		{genapi.ProviderHailuo, os.Getenv("HAILUO_API_KEY")},
		{genapi.ProviderHuoshan, os.Getenv("HUOSHAN_API_KEY")},
		{genapi.ProviderGemini, os.Getenv("GEMINI_API_KEY")},
	}

	for _, p := range providers {
		if p.key == "" {
			continue
		}
		_, err := api.RegisterProviderConfig(&genapi.Config{
			Provider: p.kind,
			APIKey:   p.key,
		})
		if err != nil {
			log.Printf("注册 %s 失败: %v", p.kind, err)
		}
	}

	ctx := context.Background()
	req := &genapi.GenerateRequest{
		Prompt:          "一朵盛开的玫瑰花",
		DurationSeconds: 6,
	}

	// 尝试使用不同的provider
	for _, provider := range []string{"hailuo", "huoshan", "gemini"} {
		resp, err := api.GenerateVideo(ctx, provider, req)
		if err != nil {
			log.Printf("%s 失败: %v", provider, err)
			continue
		}
		fmt.Printf("使用 %s 生成成功，任务ID: %s\n", provider, resp.TaskID)
		break
	}
}

// Example_withTokenRecorder 展示Token使用统计
func Example_withTokenRecorder() {
	// 设置全局Token记录器
	genapi.SetTokenUsageRecorder(genapi.TokenUsageRecorderFunc(
		func(ctx context.Context, provider string, usage *genapi.Usage) {
			log.Printf("[Token Usage] Provider: %s", provider)
			log.Printf("  Total Tokens: %d", usage.TotalTokens)
			log.Printf("  Images: %d", usage.ImageCount)
			log.Printf("  Videos: %d", usage.VideoCount)

			// 这里可以保存到数据库
			// saveUsageToDatabase(ctx, provider, usage)
		},
	))

	api := genapi.NewGenAPI()
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHailuo,
		APIKey:   os.Getenv("HAILUO_API_KEY"),
	})

	ctx := context.Background()
	req := &genapi.GenerateRequest{
		Prompt:          "测试视频",
		DurationSeconds: 6,
	}

	resp, err := api.GenerateVideo(ctx, "hailuo", req)
	if err != nil {
		log.Fatal(err)
	}

	// 从响应中也可以获取使用统计
	if resp.Usage != nil {
		fmt.Printf("本次生成使用Token: %d\n", resp.Usage.TotalTokens)
		fmt.Printf("生成耗时: %.2f秒\n", resp.Duration().Seconds())
	}
}

// Example_advancedOptions 展示高级选项
func Example_advancedOptions() {
	api := genapi.NewGenAPI()
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHailuo,
		APIKey:   os.Getenv("HAILUO_API_KEY"),
	})

	ctx := context.Background()

	// 使用高级选项
	req := &genapi.GenerateRequest{
		Prompt:          "科幻场景，太空飞船",
		DurationSeconds: 6,
		AspectRatio:     "16:9",
		Quality:         "high",
		Seed:            42, // 固定随机种子
		Options: map[string]interface{}{
			"prompt_optimizer":  true, // 启用提示词优化
			"fast_pretreatment": true, // 快速预处理
		},
		Metadata: map[string]interface{}{
			"project_id": "proj_123",
			"user_id":    456,
			"tag":        "sci-fi",
		},
	}

	resp, err := api.GenerateVideo(ctx, "hailuo", req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("任务ID: %s\n", resp.TaskID)
	fmt.Printf("元数据: %v\n", resp.Metadata)
}

// Example_errorHandling 展示错误处理
func Example_errorHandling() {
	api := genapi.NewGenAPI()
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHailuo,
		APIKey:   os.Getenv("HAILUO_API_KEY"),
	})

	ctx := context.Background()
	req := &genapi.GenerateRequest{
		Prompt:          "测试提示词",
		DurationSeconds: 6,
	}

	resp, err := api.GenerateVideo(ctx, "hailuo", req)
	if err != nil {
		// 处理调用错误
		log.Printf("调用失败: %v", err)
		return
	}

	// 检查响应中的错误
	if resp.Error != "" {
		log.Printf("生成失败: %s", resp.Error)
		if resp.ErrorCode != "" {
			log.Printf("错误码: %s", resp.ErrorCode)
		}
		return
	}

	// 成功
	fmt.Printf("生成成功，任务ID: %s\n", resp.TaskID)
}

// Example_withFallback 展示Provider切换/容错
func Example_withFallback() {
	api := genapi.NewGenAPI()

	// 注册多个provider作为备份
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHailuo,
		APIKey:   os.Getenv("HAILUO_API_KEY"),
	})
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHuoshan,
		APIKey:   os.Getenv("HUOSHAN_API_KEY"),
	})

	ctx := context.Background()
	req := &genapi.GenerateRequest{
		Prompt:          "备份测试",
		DurationSeconds: 6,
	}

	// 按优先级尝试provider
	providers := []string{"hailuo", "huoshan"}
	var resp *genapi.GenerateResponse
	var err error

	for _, provider := range providers {
		resp, err = api.GenerateVideo(ctx, provider, req)
		if err == nil && resp.Error == "" {
			fmt.Printf("使用 %s 生成成功\n", provider)
			break
		}
		log.Printf("%s 失败，尝试下一个: %v", provider, err)
	}

	if err != nil || resp.Error != "" {
		log.Fatal("所有provider都失败了")
	}

	fmt.Printf("最终任务ID: %s\n", resp.TaskID)
}

// Example_batchGeneration 展示批量生成
func Example_batchGeneration() {
	api := genapi.NewGenAPI()
	api.RegisterProviderConfig(&genapi.Config{
		Provider: genapi.ProviderHailuo,
		APIKey:   os.Getenv("HAILUO_API_KEY"),
	})

	ctx := context.Background()

	// 批量生成多个视频
	prompts := []string{
		"一只猫在玩毛线球",
		"海边的日落",
		"城市夜景延时",
	}

	type result struct {
		prompt string
		taskID string
		err    error
	}

	results := make([]result, len(prompts))

	// 并发生成
	for i, prompt := range prompts {
		go func(idx int, p string) {
			req := &genapi.GenerateRequest{
				Prompt:          p,
				DurationSeconds: 6,
			}
			resp, err := api.GenerateVideo(ctx, "hailuo", req)
			if err != nil {
				results[idx] = result{prompt: p, err: err}
			} else {
				results[idx] = result{prompt: p, taskID: resp.TaskID}
			}
		}(i, prompt)
	}

	// 等待所有任务完成（实际应用中应使用WaitGroup）
	time.Sleep(5 * time.Second)

	// 输出结果
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("❌ %s: %v\n", r.prompt, r.err)
		} else {
			fmt.Printf("✅ %s: %s\n", r.prompt, r.taskID)
		}
	}
}
