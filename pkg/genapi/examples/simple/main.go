package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/grapery/grapery/pkg/genapi"
)

func main() {
	// 从环境变量读取API密钥
	hailuoKey := os.Getenv("HAILUO_API_KEY")
	if hailuoKey == "" {
		log.Fatal("请设置 HAILUO_API_KEY 环境变量")
	}

	// 1. 创建GenAPI实例
	api := genapi.NewGenAPI()

	// 2. 配置Token使用记录器（可选）
	genapi.SetTokenUsageRecorder(genapi.TokenUsageRecorderFunc(
		func(ctx context.Context, provider string, usage *genapi.Usage) {
			log.Printf("📊 [%s] Token使用: %d, 图片: %d, 视频: %d",
				provider, usage.TotalTokens, usage.ImageCount, usage.VideoCount)
		},
	))

	// 3. 注册Provider
	config := &genapi.Config{
		Provider:   genapi.ProviderHailuo,
		APIKey:     hailuoKey,
		BaseURL:    "https://api.minimax.com",
		Model:      "MiniMax-Hailuo-02",
		ImageModel: "image-01",
		Timeout:    30 * time.Second,
	}

	provider, err := api.RegisterProviderConfig(config)
	if err != nil {
		log.Fatalf("注册Provider失败: %v", err)
	}
	log.Printf("✅ 已注册Provider: %s", provider.Name())

	ctx := context.Background()

	// 示例1: 文本生成视频
	fmt.Println("\n=== 示例1: 文本生成视频 ===")
	if err := textToVideoExample(ctx, api); err != nil {
		log.Printf("❌ 文本生视频失败: %v", err)
	}

	// 示例2: 文本生成图片
	fmt.Println("\n=== 示例2: 文本生成图片 ===")
	if err := textToImageExample(ctx, api); err != nil {
		log.Printf("❌ 文本生图失败: %v", err)
	}

	fmt.Println("\n✅ 所有示例执行完成!")
}

// textToVideoExample 文本生成视频示例
func textToVideoExample(ctx context.Context, api *genapi.GenAPI) error {
	req := &genapi.GenerateRequest{
		Prompt:          "一只可爱的小猫在草地上追逐蝴蝶，阳光明媚，画面温馨",
		DurationSeconds: 6,
		AspectRatio:     "16:9",
		Quality:         "high",
		Metadata: map[string]interface{}{
			"scene":  "outdoor",
			"animal": "cat",
		},
	}

	startTime := time.Now()
	resp, err := api.GenerateVideo(ctx, "hailuo", req)
	if err != nil {
		return fmt.Errorf("调用失败: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("生成失败: %s (code: %s)", resp.Error, resp.ErrorCode)
	}

	elapsed := time.Since(startTime)

	fmt.Printf("✅ 视频生成任务已创建\n")
	fmt.Printf("   任务ID: %s\n", resp.TaskID)
	fmt.Printf("   状态: %s\n", resp.Status)
	fmt.Printf("   Provider: %s\n", resp.Provider)
	fmt.Printf("   耗时: %.2f秒\n", elapsed.Seconds())

	if resp.Usage != nil {
		fmt.Printf("   Token使用: %d\n", resp.Usage.TotalTokens)
	}

	if resp.VideoURL != "" {
		fmt.Printf("   视频URL: %s\n", resp.VideoURL)
	}

	return nil
}

// textToImageExample 文本生成图片示例
func textToImageExample(ctx context.Context, api *genapi.GenAPI) error {
	req := &genapi.GenerateRequest{
		Operation:   genapi.OperationTextToImage,
		Prompt:      "一幅美丽的日落风景画，油画风格，暖色调",
		AspectRatio: "16:9",
		OutputCount: 2,
		Style:       "oil painting",
	}

	startTime := time.Now()
	resp, err := api.GenerateImage(ctx, "hailuo", req)
	if err != nil {
		return fmt.Errorf("调用失败: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("生成失败: %s", resp.Error)
	}

	elapsed := time.Since(startTime)

	fmt.Printf("✅ 图片生成成功\n")
	fmt.Printf("   生成数量: %d\n", len(resp.ImageURLs))
	fmt.Printf("   耗时: %.2f秒\n", elapsed.Seconds())

	for i, url := range resp.ImageURLs {
		fmt.Printf("   图片%d: %s\n", i+1, url)
	}

	if resp.Usage != nil {
		fmt.Printf("   图片数量: %d\n", resp.Usage.ImageCount)
	}

	return nil
}
