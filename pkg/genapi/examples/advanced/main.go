package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/grapery/grapery/pkg/genapi"
)

// VideoService 封装视频生成服务
type VideoService struct {
	api           *genapi.GenAPI
	providers     []string
	usageRecorder *UsageTracker
}

// UsageTracker 跟踪使用统计
type UsageTracker struct {
	mu    sync.Mutex
	usage map[string]*genapi.Usage
}

func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		usage: make(map[string]*genapi.Usage),
	}
}

func (t *UsageTracker) Record(provider string, usage *genapi.Usage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if existing, ok := t.usage[provider]; ok {
		existing.TotalTokens += usage.TotalTokens
		existing.ImageCount += usage.ImageCount
		existing.VideoCount += usage.VideoCount
	} else {
		t.usage[provider] = usage.Clone()
	}
}

func (t *UsageTracker) Report() {
	t.mu.Lock()
	defer t.mu.Unlock()

	fmt.Println("\n📊 === 使用统计报告 ===")
	total := 0
	for provider, usage := range t.usage {
		fmt.Printf("Provider: %s\n", provider)
		fmt.Printf("  Tokens: %d\n", usage.TotalTokens)
		fmt.Printf("  Images: %d\n", usage.ImageCount)
		fmt.Printf("  Videos: %d\n", usage.VideoCount)
		total += usage.TotalTokens
	}
	fmt.Printf("总Token消耗: %d\n", total)
}

// NewVideoService 创建视频服务实例
func NewVideoService() (*VideoService, error) {
	api := genapi.NewGenAPI()
	tracker := NewUsageTracker()

	// 设置Token记录器
	genapi.SetTokenUsageRecorder(genapi.TokenUsageRecorderFunc(
		func(ctx context.Context, provider string, usage *genapi.Usage) {
			tracker.Record(provider, usage)
			log.Printf("📊 [%s] +%d tokens", provider, usage.TotalTokens)
		},
	))

	// 注册所有可用的provider
	providers := []struct {
		kind   genapi.ProviderKind
		envKey string
	}{
		{genapi.ProviderHailuo, "HAILUO_API_KEY"},
		{genapi.ProviderHuoshan, "HUOSHAN_API_KEY"},
		{genapi.ProviderGemini, "GEMINI_API_KEY"},
	}

	var activeProviders []string
	for _, p := range providers {
		apiKey := os.Getenv(p.envKey)
		if apiKey == "" {
			log.Printf("⚠️  跳过 %s (未设置环境变量 %s)", p.kind, p.envKey)
			continue
		}

		config := &genapi.Config{
			Provider: p.kind,
			APIKey:   apiKey,
			Timeout:  30 * time.Second,
		}

		_, err := api.RegisterProviderConfig(config)
		if err != nil {
			log.Printf("❌ 注册 %s 失败: %v", p.kind, err)
			continue
		}

		activeProviders = append(activeProviders, string(p.kind))
		log.Printf("✅ 已注册Provider: %s", p.kind)
	}

	if len(activeProviders) == 0 {
		return nil, fmt.Errorf("没有可用的provider")
	}

	return &VideoService{
		api:           api,
		providers:     activeProviders,
		usageRecorder: tracker,
	}, nil
}

// GenerateVideoWithFallback 使用fallback机制生成视频
func (s *VideoService) GenerateVideoWithFallback(ctx context.Context, req *genapi.GenerateRequest) (*genapi.GenerateResponse, error) {
	var lastErr error

	for _, provider := range s.providers {
		log.Printf("🎬 尝试使用 %s 生成视频...", provider)

		resp, err := s.api.GenerateVideo(ctx, provider, req)
		if err != nil {
			log.Printf("❌ %s 失败: %v", provider, err)
			lastErr = err
			continue
		}

		if resp.Error != "" {
			log.Printf("❌ %s 返回错误: %s", provider, resp.Error)
			lastErr = fmt.Errorf("%s: %s", provider, resp.Error)
			continue
		}

		log.Printf("✅ 使用 %s 生成成功", provider)
		return resp, nil
	}

	return nil, fmt.Errorf("所有provider都失败: %w", lastErr)
}

// BatchGenerate 批量生成视频
func (s *VideoService) BatchGenerate(ctx context.Context, prompts []string) []BatchResult {
	results := make([]BatchResult, len(prompts))
	var wg sync.WaitGroup

	for i, prompt := range prompts {
		wg.Add(1)
		go func(idx int, p string) {
			defer wg.Done()

			req := &genapi.GenerateRequest{
				Prompt:          p,
				DurationSeconds: 6,
				AspectRatio:     "16:9",
			}

			startTime := time.Now()
			resp, err := s.GenerateVideoWithFallback(ctx, req)
			elapsed := time.Since(startTime)

			results[idx] = BatchResult{
				Index:    idx,
				Prompt:   p,
				Response: resp,
				Error:    err,
				Duration: elapsed,
			}
		}(i, prompt)
	}

	wg.Wait()
	return results
}

// BatchResult 批量生成结果
type BatchResult struct {
	Index    int
	Prompt   string
	Response *genapi.GenerateResponse
	Error    error
	Duration time.Duration
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	// 创建服务
	service, err := NewVideoService()
	if err != nil {
		log.Fatalf("创建服务失败: %v", err)
	}

	ctx := context.Background()

	// 示例1: 单个视频生成（带fallback）
	fmt.Println("\n=== 示例1: 单个视频生成 (自动Fallback) ===")
	singleVideoExample(ctx, service)

	// 示例2: 批量生成
	fmt.Println("\n=== 示例2: 批量视频生成 ===")
	batchVideoExample(ctx, service)

	// 示例3: 不同操作类型
	fmt.Println("\n=== 示例3: 多种操作类型 ===")
	multiOperationExample(ctx, service)

	// 输出使用统计
	service.usageRecorder.Report()

	fmt.Println("\n✅ 所有高级示例执行完成!")
}

func singleVideoExample(ctx context.Context, service *VideoService) {
	req := &genapi.GenerateRequest{
		Prompt:          "科幻场景：一艘太空飞船在星云中穿梭，引擎发出蓝色光芒",
		DurationSeconds: 6,
		AspectRatio:     "16:9",
		Quality:         "high",
		Style:           "cinematic",
	}

	startTime := time.Now()
	resp, err := service.GenerateVideoWithFallback(ctx, req)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("❌ 生成失败: %v", err)
		return
	}

	fmt.Printf("\n✅ 视频生成成功\n")
	fmt.Printf("   Provider: %s\n", resp.Provider)
	fmt.Printf("   任务ID: %s\n", resp.TaskID)
	fmt.Printf("   状态: %s\n", resp.Status)
	fmt.Printf("   总耗时: %.2f秒\n", elapsed.Seconds())

	if resp.Usage != nil {
		fmt.Printf("   Token消耗: %d\n", resp.Usage.TotalTokens)
	}
}

func batchVideoExample(ctx context.Context, service *VideoService) {
	prompts := []string{
		"一只猫在窗台上看雨",
		"城市街道的延时摄影",
		"海浪拍打礁石",
		"樱花飘落的慢镜头",
	}

	fmt.Printf("🎬 开始批量生成 %d 个视频...\n", len(prompts))
	startTime := time.Now()

	results := service.BatchGenerate(ctx, prompts)
	elapsed := time.Since(startTime)

	fmt.Printf("\n📊 批量生成完成，总耗时: %.2f秒\n\n", elapsed.Seconds())

	successCount := 0
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("❌ [%d] %s\n", result.Index+1, result.Prompt)
			fmt.Printf("      错误: %v\n", result.Error)
		} else {
			fmt.Printf("✅ [%d] %s\n", result.Index+1, result.Prompt)
			fmt.Printf("      Provider: %s, 任务ID: %s, 耗时: %.2fs\n",
				result.Response.Provider,
				result.Response.TaskID,
				result.Duration.Seconds())
			successCount++
		}
	}

	fmt.Printf("\n成功率: %d/%d (%.1f%%)\n",
		successCount, len(results),
		float64(successCount)/float64(len(results))*100)
}

func multiOperationExample(ctx context.Context, service *VideoService) {
	operations := []struct {
		name string
		req  *genapi.GenerateRequest
	}{
		{
			name: "文本生图",
			req: &genapi.GenerateRequest{
				Operation:   genapi.OperationTextToImage,
				Prompt:      "一幅抽象艺术画，色彩丰富",
				AspectRatio: "1:1",
			},
		},
		{
			name: "文本生视频",
			req: &genapi.GenerateRequest{
				Operation:       genapi.OperationTextToVideo,
				Prompt:          "森林中的小溪流水",
				DurationSeconds: 6,
			},
		},
	}

	for _, op := range operations {
		fmt.Printf("\n🎨 执行操作: %s\n", op.name)

		var resp *genapi.GenerateResponse
		var err error

		if op.req.Operation.MediaType() == genapi.MediaTypeImage {
			resp, err = service.api.GenerateImage(ctx, service.providers[0], op.req)
		} else {
			resp, err = service.GenerateVideoWithFallback(ctx, op.req)
		}

		if err != nil {
			log.Printf("❌ %s 失败: %v", op.name, err)
			continue
		}

		fmt.Printf("✅ %s 成功\n", op.name)
		fmt.Printf("   Provider: %s\n", resp.Provider)
		fmt.Printf("   任务ID: %s\n", resp.TaskID)

		if len(resp.ImageURLs) > 0 {
			fmt.Printf("   图片数量: %d\n", len(resp.ImageURLs))
		}
		if resp.VideoURL != "" {
			fmt.Printf("   视频URL: %s\n", resp.VideoURL)
		}
	}
}
