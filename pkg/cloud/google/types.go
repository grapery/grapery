package google

import (
	"context"
	"io"
)

// MediaGenerationRequest 媒体生成请求接口
type MediaGenerationRequest interface {
	GetPrompt() string
	GetOptions() map[string]interface{}
	Validate() error
}

// MediaGenerationResponse 媒体生成响应接口
type MediaGenerationResponse interface {
	GetData() []byte
	GetMimeType() string
	GetMetadata() map[string]interface{}
	IsSuccess() bool
	GetError() error
}

// ImageGenerationRequest 图片生成请求
type ImageGenerationRequest struct {
	Prompt         string                 `json:"prompt"`                    // 图片描述提示词
	Style          string                 `json:"style,omitempty"`           // 图片风格
	Quality        string                 `json:"quality,omitempty"`         // 图片质量
	Size           string                 `json:"size,omitempty"`            // 图片尺寸
	AspectRatio    string                 `json:"aspect_ratio,omitempty"`    // 宽高比
	NegativePrompt string                 `json:"negative_prompt,omitempty"` // 负面提示词
	Options        map[string]interface{} `json:"options,omitempty"`         // 其他选项
}

// VideoGenerationRequest 视频生成请求
type VideoGenerationRequest struct {
	Prompt         string                 `json:"prompt"`                    // 视频描述提示词
	Duration       int                    `json:"duration,omitempty"`        // 视频时长（秒）
	Resolution     string                 `json:"resolution,omitempty"`      // 视频分辨率
	FrameRate      int                    `json:"frame_rate,omitempty"`      // 帧率
	Style          string                 `json:"style,omitempty"`           // 视频风格
	NegativePrompt string                 `json:"negative_prompt,omitempty"` // 负面提示词
	Options        map[string]interface{} `json:"options,omitempty"`         // 其他选项
}

// ImageGenerationResponse 图片生成响应
type ImageGenerationResponse struct {
	Data     []byte                 `json:"data"`      // 图片数据
	MimeType string                 `json:"mime_type"` // MIME类型
	Width    int                    `json:"width"`     // 图片宽度
	Height   int                    `json:"height"`    // 图片高度
	Metadata map[string]interface{} `json:"metadata"`  // 元数据
	Success  bool                   `json:"success"`   // 是否成功
	Error    error                  `json:"error"`     // 错误信息
}

// VideoGenerationResponse 视频生成响应
type VideoGenerationResponse struct {
	Data       []byte                 `json:"data"`       // 视频数据
	MimeType   string                 `json:"mime_type"`  // MIME类型
	Duration   float64                `json:"duration"`   // 视频时长（秒）
	Resolution string                 `json:"resolution"` // 视频分辨率
	FrameRate  int                    `json:"frame_rate"` // 帧率
	Metadata   map[string]interface{} `json:"metadata"`   // 元数据
	Success    bool                   `json:"success"`    // 是否成功
	Error      error                  `json:"error"`      // 错误信息
}

// MediaGenerator 媒体生成器接口
type MediaGenerator interface {
	// GenerateImage 生成图片
	GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error)

	// GenerateVideo 生成视频
	GenerateVideo(ctx context.Context, req *VideoGenerationRequest) (*VideoGenerationResponse, error)

	// GenerateImageToFile 生成图片并保存到文件
	GenerateImageToFile(ctx context.Context, req *ImageGenerationRequest, filename string) error

	// GenerateVideoToFile 生成视频并保存到文件
	GenerateVideoToFile(ctx context.Context, req *VideoGenerationRequest, filename string) error

	// GenerateImageToWriter 生成图片并写入到Writer
	GenerateImageToWriter(ctx context.Context, req *ImageGenerationRequest, w io.Writer) error

	// GenerateVideoToWriter 生成视频并写入到Writer
	GenerateVideoToWriter(ctx context.Context, req *VideoGenerationRequest, w io.Writer) error
}

// PromptTemplate 提示词模板
type PromptTemplate struct {
	Name        string          `json:"name"`        // 模板名称
	Description string          `json:"description"` // 模板描述
	Template    string          `json:"template"`    // 模板内容
	Variables   []string        `json:"variables"`   // 变量列表
	Category    string          `json:"category"`    // 分类
	Tags        []string        `json:"tags"`        // 标签
	Examples    []PromptExample `json:"examples"`    // 示例
}

// PromptExample 提示词示例
type PromptExample struct {
	Title       string `json:"title"`       // 示例标题
	Description string `json:"description"` // 示例描述
	Prompt      string `json:"prompt"`      // 示例提示词
	Result      string `json:"result"`      // 预期结果描述
}

// PromptManager 提示词管理器接口
type PromptManager interface {
	// GetTemplate 获取模板
	GetTemplate(name string) (*PromptTemplate, error)

	// ListTemplates 列出所有模板
	ListTemplates(category string) ([]*PromptTemplate, error)

	// CreateTemplate 创建模板
	CreateTemplate(template *PromptTemplate) error

	// UpdateTemplate 更新模板
	UpdateTemplate(template *PromptTemplate) error

	// DeleteTemplate 删除模板
	DeleteTemplate(name string) error

	// RenderTemplate 渲染模板
	RenderTemplate(templateName string, variables map[string]string) (string, error)
}

// 实现MediaGenerationRequest接口
func (r *ImageGenerationRequest) GetPrompt() string {
	return r.Prompt
}

func (r *ImageGenerationRequest) GetOptions() map[string]interface{} {
	return r.Options
}

func (r *ImageGenerationRequest) Validate() error {
	if r.Prompt == "" {
		return ErrEmptyPrompt
	}
	return nil
}

func (r *VideoGenerationRequest) GetPrompt() string {
	return r.Prompt
}

func (r *VideoGenerationRequest) GetOptions() map[string]interface{} {
	return r.Options
}

func (r *VideoGenerationRequest) Validate() error {
	if r.Prompt == "" {
		return ErrEmptyPrompt
	}
	return nil
}

// 实现MediaGenerationResponse接口
func (r *ImageGenerationResponse) GetData() []byte {
	return r.Data
}

func (r *ImageGenerationResponse) GetMimeType() string {
	return r.MimeType
}

func (r *ImageGenerationResponse) GetMetadata() map[string]interface{} {
	return r.Metadata
}

func (r *ImageGenerationResponse) IsSuccess() bool {
	return r.Success
}

func (r *ImageGenerationResponse) GetError() error {
	return r.Error
}

func (r *VideoGenerationResponse) GetData() []byte {
	return r.Data
}

func (r *VideoGenerationResponse) GetMimeType() string {
	return r.MimeType
}

func (r *VideoGenerationResponse) GetMetadata() map[string]interface{} {
	return r.Metadata
}

func (r *VideoGenerationResponse) IsSuccess() bool {
	return r.Success
}

func (r *VideoGenerationResponse) GetError() error {
	return r.Error
}
