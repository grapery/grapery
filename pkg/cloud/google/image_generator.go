package google

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/genai"
)

// GeminiImageGenerator Gemini图片生成器
type GeminiImageGenerator struct {
	client *genai.Client
	apiKey string
}

// NewGeminiImageGenerator 创建Gemini图片生成器
func NewGeminiImageGenerator(apiKey string) (*GeminiImageGenerator, error) {
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// 设置API密钥环境变量
	os.Setenv("GOOGLE_API_KEY", apiKey)

	client, err := genai.NewClient(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建Gemini客户端失败: %w", err)
	}

	return &GeminiImageGenerator{
		client: client,
		apiKey: apiKey,
	}, nil
}

// GenerateImage 生成图片
func (g *GeminiImageGenerator) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return &ImageGenerationResponse{
			Success: false,
			Error:   err,
		}, err
	}

	// 构建提示词
	prompt := g.buildImagePrompt(req)

	// 调用Gemini API生成图片
	result, err := g.client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash-image-preview", // Nano Banana模型
		genai.Text(prompt),
		nil, // 使用默认配置
	)

	if err != nil {
		return &ImageGenerationResponse{
			Success: false,
			Error:   fmt.Errorf("图片生成失败: %w", err),
		}, err
	}

	// 处理响应
	response := &ImageGenerationResponse{
		Success:  true,
		MimeType: "image/png", // Gemini默认生成PNG格式
		Metadata: make(map[string]interface{}),
	}

	// 提取图片数据
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			// 如果有文本描述，添加到元数据
			response.Metadata["description"] = part.Text
		} else if part.InlineData != nil {
			// 提取图片数据
			response.Data = part.InlineData.Data
			response.Metadata["size"] = len(part.InlineData.Data)
			response.Metadata["format"] = part.InlineData.MIMEType
		}
	}

	// 设置图片尺寸（默认1024x1024）
	response.Width = 1024
	response.Height = 1024

	return response, nil
}

// GenerateVideo 生成视频（图片生成器不支持视频）
func (g *GeminiImageGenerator) GenerateVideo(ctx context.Context, req *VideoGenerationRequest) (*VideoGenerationResponse, error) {
	return &VideoGenerationResponse{
		Success: false,
		Error:   fmt.Errorf("图片生成器不支持视频生成"),
	}, fmt.Errorf("图片生成器不支持视频生成")
}

// GenerateImageToFile 生成图片并保存到文件
func (g *GeminiImageGenerator) GenerateImageToFile(ctx context.Context, req *ImageGenerationRequest, filename string) error {
	response, err := g.GenerateImage(ctx, req)
	if err != nil {
		return err
	}

	if !response.Success {
		return response.Error
	}

	// 确保目录存在
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 保存文件
	if err := os.WriteFile(filename, response.Data, 0644); err != nil {
		return fmt.Errorf("保存图片失败: %w", err)
	}

	return nil
}

// GenerateVideoToFile 生成视频并保存到文件（不支持）
func (g *GeminiImageGenerator) GenerateVideoToFile(ctx context.Context, req *VideoGenerationRequest, filename string) error {
	return fmt.Errorf("图片生成器不支持视频生成")
}

// GenerateImageToWriter 生成图片并写入到Writer
func (g *GeminiImageGenerator) GenerateImageToWriter(ctx context.Context, req *ImageGenerationRequest, w io.Writer) error {
	response, err := g.GenerateImage(ctx, req)
	if err != nil {
		return err
	}

	if !response.Success {
		return response.Error
	}

	_, err = w.Write(response.Data)
	return err
}

// GenerateVideoToWriter 生成视频并写入到Writer（不支持）
func (g *GeminiImageGenerator) GenerateVideoToWriter(ctx context.Context, req *VideoGenerationRequest, w io.Writer) error {
	return fmt.Errorf("图片生成器不支持视频生成")
}

// buildImagePrompt 构建图片生成提示词
func (g *GeminiImageGenerator) buildImagePrompt(req *ImageGenerationRequest) string {
	var parts []string

	// 基础提示词
	parts = append(parts, req.Prompt)

	// 添加风格
	if req.Style != "" {
		parts = append(parts, req.Style+" style")
	}

	// 添加质量
	if req.Quality != "" {
		parts = append(parts, req.Quality+" quality")
	} else {
		parts = append(parts, "high quality")
	}

	// 添加尺寸信息
	if req.Size != "" {
		parts = append(parts, req.Size)
	} else if req.AspectRatio != "" {
		parts = append(parts, req.AspectRatio+" aspect ratio")
	}

	// 添加技术细节
	parts = append(parts, "detailed, realistic, professional")

	// 添加负面提示词
	if req.NegativePrompt != "" {
		parts = append(parts, "avoiding "+req.NegativePrompt)
	}

	// 添加通用负面提示词
	parts = append(parts, "avoiding blurry, low quality, distorted, deformed")

	return strings.Join(parts, ", ")
}

// GetSupportedFormats 获取支持的图片格式
func (g *GeminiImageGenerator) GetSupportedFormats() []string {
	return []string{"png", "jpeg", "webp"}
}

// GetMaxResolution 获取最大分辨率
func (g *GeminiImageGenerator) GetMaxResolution() (int, int) {
	return 1024, 1024
}

// GetSupportedAspectRatios 获取支持的宽高比
func (g *GeminiImageGenerator) GetSupportedAspectRatios() []string {
	return []string{
		"1:1",  // 正方形
		"16:9", // 宽屏
		"4:3",  // 标准
		"3:2",  // 经典
		"9:16", // 竖屏
	}
}

// ValidateImageRequest 验证图片生成请求
func (g *GeminiImageGenerator) ValidateImageRequest(req *ImageGenerationRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	// 验证尺寸
	if req.Size != "" {
		validSizes := []string{"1024x1024", "512x512", "256x256"}
		valid := false
		for _, size := range validSizes {
			if req.Size == size {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidImageSize
		}
	}

	// 验证宽高比
	if req.AspectRatio != "" {
		validRatios := g.GetSupportedAspectRatios()
		valid := false
		for _, ratio := range validRatios {
			if req.AspectRatio == ratio {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidAspectRatio
		}
	}

	return nil
}

// Close 关闭客户端
func (g *GeminiImageGenerator) Close() error {
	// Gemini客户端没有Close方法，这里只是占位符
	return nil
}
