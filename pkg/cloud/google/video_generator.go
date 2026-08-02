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

// VeoVideoGenerator Veo视频生成器
type VeoVideoGenerator struct {
	client *genai.Client
	apiKey string
	model  string // veo2 或 veo3
}

// NewVeoVideoGenerator 创建Veo视频生成器
func NewVeoVideoGenerator(apiKey string, model string) (*VeoVideoGenerator, error) {
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// 验证模型
	if model != "veo2" && model != "veo3" {
		model = "veo2" // 默认使用veo2
	}

	// 设置API密钥环境变量
	os.Setenv("GOOGLE_API_KEY", apiKey)

	client, err := genai.NewClient(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建Gemini客户端失败: %w", err)
	}

	return &VeoVideoGenerator{
		client: client,
		apiKey: apiKey,
		model:  model,
	}, nil
}

// GenerateVideo 生成视频
func (v *VeoVideoGenerator) GenerateVideo(ctx context.Context, req *VideoGenerationRequest) (*VideoGenerationResponse, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return &VideoGenerationResponse{
			Success: false,
			Error:   err,
		}, err
	}

	// 验证视频参数
	if err := v.validateVideoRequest(req); err != nil {
		return &VideoGenerationResponse{
			Success: false,
			Error:   err,
		}, err
	}

	// 构建提示词
	prompt := v.buildVideoPrompt(req)

	// 调用Gemini API生成视频
	result, err := v.client.Models.GenerateContent(
		ctx,
		v.getModelName(),
		genai.Text(prompt),
		nil, // 使用默认配置
	)

	if err != nil {
		return &VideoGenerationResponse{
			Success: false,
			Error:   fmt.Errorf("视频生成失败: %w", err),
		}, err
	}

	// 处理响应
	response := &VideoGenerationResponse{
		Success:    true,
		MimeType:   "video/mp4", // Veo默认生成MP4格式
		Duration:   float64(req.Duration),
		Resolution: req.Resolution,
		FrameRate:  req.FrameRate,
		Metadata:   make(map[string]interface{}),
	}

	// 提取视频数据
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			// 如果有文本描述，添加到元数据
			response.Metadata["description"] = part.Text
		} else if part.InlineData != nil {
			// 提取视频数据
			response.Data = part.InlineData.Data
			response.Metadata["size"] = len(part.InlineData.Data)
			response.Metadata["format"] = part.InlineData.MIMEType
			response.Metadata["model"] = v.model
		}
	}

	return response, nil
}

// GenerateImage 生成图片（视频生成器不支持图片）
func (v *VeoVideoGenerator) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	return &ImageGenerationResponse{
		Success: false,
		Error:   fmt.Errorf("视频生成器不支持图片生成"),
	}, fmt.Errorf("视频生成器不支持图片生成")
}

// GenerateVideoToFile 生成视频并保存到文件
func (v *VeoVideoGenerator) GenerateVideoToFile(ctx context.Context, req *VideoGenerationRequest, filename string) error {
	response, err := v.GenerateVideo(ctx, req)
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
		return fmt.Errorf("保存视频失败: %w", err)
	}

	return nil
}

// GenerateImageToFile 生成图片并保存到文件（不支持）
func (v *VeoVideoGenerator) GenerateImageToFile(ctx context.Context, req *ImageGenerationRequest, filename string) error {
	return fmt.Errorf("视频生成器不支持图片生成")
}

// GenerateVideoToWriter 生成视频并写入到Writer
func (v *VeoVideoGenerator) GenerateVideoToWriter(ctx context.Context, req *VideoGenerationRequest, w io.Writer) error {
	response, err := v.GenerateVideo(ctx, req)
	if err != nil {
		return err
	}

	if !response.Success {
		return response.Error
	}

	_, err = w.Write(response.Data)
	return err
}

// GenerateImageToWriter 生成图片并写入到Writer（不支持）
func (v *VeoVideoGenerator) GenerateImageToWriter(ctx context.Context, req *ImageGenerationRequest, w io.Writer) error {
	return fmt.Errorf("视频生成器不支持图片生成")
}

// buildVideoPrompt 构建视频生成提示词
func (v *VeoVideoGenerator) buildVideoPrompt(req *VideoGenerationRequest) string {
	var parts []string

	// 基础提示词
	parts = append(parts, req.Prompt)

	// 添加风格
	if req.Style != "" {
		parts = append(parts, req.Style+" style")
	}

	// 添加时长信息
	if req.Duration > 0 {
		parts = append(parts, fmt.Sprintf("%d seconds long", req.Duration))
	}

	// 添加分辨率信息
	if req.Resolution != "" {
		parts = append(parts, req.Resolution+" resolution")
	}

	// 添加帧率信息
	if req.FrameRate > 0 {
		parts = append(parts, fmt.Sprintf("%d fps", req.FrameRate))
	}

	// 添加技术细节
	parts = append(parts, "high quality, smooth motion, professional")

	// 添加负面提示词
	if req.NegativePrompt != "" {
		parts = append(parts, "avoiding "+req.NegativePrompt)
	}

	// 添加通用负面提示词
	parts = append(parts, "avoiding blurry, low quality, choppy motion, artifacts")

	return strings.Join(parts, ", ")
}

// validateVideoRequest 验证视频生成请求
func (v *VeoVideoGenerator) validateVideoRequest(req *VideoGenerationRequest) error {
	// 验证时长
	if req.Duration <= 0 {
		req.Duration = 5 // 默认5秒
	} else if req.Duration > 60 {
		return ErrInvalidDuration
	}

	// 验证分辨率
	if req.Resolution == "" {
		req.Resolution = "1920x1080" // 默认1080p
	} else {
		validResolutions := v.GetSupportedResolutions()
		valid := false
		for _, res := range validResolutions {
			if req.Resolution == res {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidResolution
		}
	}

	// 验证帧率
	if req.FrameRate <= 0 {
		req.FrameRate = 24 // 默认24fps
	} else if req.FrameRate > 60 {
		return ErrInvalidFrameRate
	}

	return nil
}

// getModelName 获取模型名称
func (v *VeoVideoGenerator) getModelName() string {
	switch v.model {
	case "veo3":
		return "veo-3"
	case "veo2":
		return "veo-2"
	default:
		return "veo-2"
	}
}

// GetSupportedResolutions 获取支持的分辨率
func (v *VeoVideoGenerator) GetSupportedResolutions() []string {
	return []string{
		"1280x720",  // 720p
		"1920x1080", // 1080p
		"2560x1440", // 1440p
		"3840x2160", // 4K
	}
}

// GetSupportedDurations 获取支持的时长范围
func (v *VeoVideoGenerator) GetSupportedDurations() (int, int) {
	return 1, 60 // 1秒到60秒
}

// GetSupportedFrameRates 获取支持的帧率
func (v *VeoVideoGenerator) GetSupportedFrameRates() []int {
	return []int{24, 25, 30, 50, 60}
}

// GetSupportedFormats 获取支持的视频格式
func (v *VeoVideoGenerator) GetSupportedFormats() []string {
	return []string{"mp4", "webm", "mov"}
}

// GetModelInfo 获取模型信息
func (v *VeoVideoGenerator) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"model":        v.model,
		"name":         fmt.Sprintf("Veo %s", strings.ToUpper(v.model)),
		"type":         "video_generation",
		"max_duration": 60,
		"min_duration": 1,
		"resolutions":  v.GetSupportedResolutions(),
		"frame_rates":  v.GetSupportedFrameRates(),
		"formats":      v.GetSupportedFormats(),
	}
}

// Close 关闭客户端
func (v *VeoVideoGenerator) Close() error {
	// Gemini客户端没有Close方法，这里只是占位符
	return nil
}
