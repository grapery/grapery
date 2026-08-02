package google

import (
	"context"
	"fmt"
	"io"
)

// MediaGeneratorType 媒体生成器类型
type MediaGeneratorType string

const (
	// ImageGeneratorType 图片生成器类型
	ImageGeneratorType MediaGeneratorType = "image"
	// VideoGeneratorType 视频生成器类型
	VideoGeneratorType MediaGeneratorType = "video"
	// CombinedGeneratorType 组合生成器类型
	CombinedGeneratorType MediaGeneratorType = "combined"
)

// MediaGeneratorFactory 媒体生成器工厂
type MediaGeneratorFactory struct {
	apiKey string
}

// NewMediaGeneratorFactory 创建媒体生成器工厂
func NewMediaGeneratorFactory(apiKey string) *MediaGeneratorFactory {
	return &MediaGeneratorFactory{
		apiKey: apiKey,
	}
}

// CreateGenerator 创建媒体生成器
func (f *MediaGeneratorFactory) CreateGenerator(generatorType MediaGeneratorType, options map[string]string) (MediaGenerator, error) {
	switch generatorType {
	case ImageGeneratorType:
		return NewGeminiImageGenerator(f.apiKey)
	case VideoGeneratorType:
		model := "veo2" // 默认使用veo2
		if m, exists := options["model"]; exists {
			model = m
		}
		return NewVeoVideoGenerator(f.apiKey, model)
	case CombinedGeneratorType:
		return NewCombinedMediaGenerator(f.apiKey, options)
	default:
		return nil, fmt.Errorf("不支持的生成器类型: %s", generatorType)
	}
}

// CombinedMediaGenerator 组合媒体生成器
type CombinedMediaGenerator struct {
	imageGenerator *GeminiImageGenerator
	videoGenerator *VeoVideoGenerator
}

// NewCombinedMediaGenerator 创建组合媒体生成器
func NewCombinedMediaGenerator(apiKey string, options map[string]string) (*CombinedMediaGenerator, error) {
	imageGen, err := NewGeminiImageGenerator(apiKey)
	if err != nil {
		return nil, fmt.Errorf("创建图片生成器失败: %w", err)
	}

	model := "veo2"
	if m, exists := options["video_model"]; exists {
		model = m
	}

	videoGen, err := NewVeoVideoGenerator(apiKey, model)
	if err != nil {
		imageGen.Close()
		return nil, fmt.Errorf("创建视频生成器失败: %w", err)
	}

	return &CombinedMediaGenerator{
		imageGenerator: imageGen,
		videoGenerator: videoGen,
	}, nil
}

// GenerateImage 生成图片
func (c *CombinedMediaGenerator) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	return c.imageGenerator.GenerateImage(ctx, req)
}

// GenerateVideo 生成视频
func (c *CombinedMediaGenerator) GenerateVideo(ctx context.Context, req *VideoGenerationRequest) (*VideoGenerationResponse, error) {
	return c.videoGenerator.GenerateVideo(ctx, req)
}

// GenerateImageToFile 生成图片并保存到文件
func (c *CombinedMediaGenerator) GenerateImageToFile(ctx context.Context, req *ImageGenerationRequest, filename string) error {
	return c.imageGenerator.GenerateImageToFile(ctx, req, filename)
}

// GenerateVideoToFile 生成视频并保存到文件
func (c *CombinedMediaGenerator) GenerateVideoToFile(ctx context.Context, req *VideoGenerationRequest, filename string) error {
	return c.videoGenerator.GenerateVideoToFile(ctx, req, filename)
}

// GenerateImageToWriter 生成图片并写入到Writer
func (c *CombinedMediaGenerator) GenerateImageToWriter(ctx context.Context, req *ImageGenerationRequest, w io.Writer) error {
	return c.imageGenerator.GenerateImageToWriter(ctx, req, w)
}

// GenerateVideoToWriter 生成视频并写入到Writer
func (c *CombinedMediaGenerator) GenerateVideoToWriter(ctx context.Context, req *VideoGenerationRequest, w io.Writer) error {
	return c.videoGenerator.GenerateVideoToWriter(ctx, req, w)
}

// Close 关闭所有生成器
func (c *CombinedMediaGenerator) Close() error {
	var errs []error

	if err := c.imageGenerator.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := c.videoGenerator.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭生成器时发生错误: %v", errs)
	}

	return nil
}

// GetImageGenerator 获取图片生成器
func (c *CombinedMediaGenerator) GetImageGenerator() *GeminiImageGenerator {
	return c.imageGenerator
}

// GetVideoGenerator 获取视频生成器
func (c *CombinedMediaGenerator) GetVideoGenerator() *VeoVideoGenerator {
	return c.videoGenerator
}
