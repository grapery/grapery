// Package google 提供Google Gemini API的图片和视频生成功能
//
// 主要功能:
// - 使用Gemini 2.5 Flash Image (Nano Banana)模型生成高质量图片
// - 使用Veo2/Veo3模型生成视频内容
// - 提供丰富的提示词模板和技巧库
// - 支持多种图片和视频格式
// - 提供统一的抽象接口和错误处理
//
// 使用示例:
//
//	// 创建图片生成器
//	imageGen, err := google.NewGeminiImageGenerator(apiKey)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer imageGen.Close()
//
//	// 生成图片
//	req := &google.ImageGenerationRequest{
//		Prompt: "A beautiful sunset over mountains",
//		Style:  "photorealistic",
//		Quality: "high",
//	}
//	response, err := imageGen.GenerateImage(ctx, req)
//
//	// 创建视频生成器
//	videoGen, err := google.NewVeoVideoGenerator(apiKey, "veo2")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer videoGen.Close()
//
//	// 生成视频
//	videoReq := &google.VideoGenerationRequest{
//		Prompt: "A cinematic cityscape at night",
//		Duration: 10,
//		Resolution: "1920x1080",
//	}
//	videoResponse, err := videoGen.GenerateVideo(ctx, videoReq)
//
// 提示词模板:
//
//	// 使用预定义模板
//	manager := google.NewPromptManager()
//	prompt, err := manager.RenderTemplate("portrait_photo", map[string]string{
//		"subject": "a businesswoman",
//		"lighting": "soft natural lighting",
//		"style": "professional",
//	})
//
// 高级提示词构建:
//
//	// 使用技巧库构建高级提示词
//	options := map[string]string{
//		"style": "minimalist commercial",
//		"quality": "ultra high",
//		"details": "studio lighting, clean background",
//	}
//	advancedPrompt := google.BuildAdvancedPrompt("Product photo", options)
package google
