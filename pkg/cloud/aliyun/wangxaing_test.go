package aliyun

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	// 创建客户端
	client := NewWanxiangClient("your-api-key")

	// 设置可选参数
	params := &Params{
		Duration:     ptr(5),    // 5秒视频
		PromptExtend: ptr(true), // 开启智能提示词扩展
	}

	// 创建任务
	ctx := context.Background()
	resp, err := client.CreateTask(ctx,
		"https://example.com/image.jpg", // 图片URL
		"一只猫在草地上奔跑",                     // 提示词
		params,
	)
	if err != nil {
		log.Fatal(err)
	}

	// 等待任务完成（最多等待10分钟）
	result, err := client.WaitForTask(ctx, resp.Output.TaskID, 10*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	// 获取生成的视频URL
	if len(result.Output.VideoURL) > 0 {
		fmt.Printf("Generated video URL: %s\n", result.Output.VideoURL)
	}
}

// 辅助函数：创建指针
func ptr[T any](v T) *T {
	return &v
}
