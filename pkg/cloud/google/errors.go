package google

import "errors"

// 定义错误类型
var (
	// 基础错误
	ErrEmptyPrompt       = errors.New("提示词不能为空")
	ErrInvalidRequest    = errors.New("无效的请求参数")
	ErrGenerationFailed  = errors.New("媒体生成失败")
	ErrUnsupportedFormat = errors.New("不支持的格式")
	ErrTemplateNotFound  = errors.New("模板未找到")
	ErrInvalidTemplate   = errors.New("无效的模板")
	ErrRenderingFailed   = errors.New("模板渲染失败")

	// 图片生成相关错误
	ErrImageGenerationFailed = errors.New("图片生成失败")
	ErrInvalidImageSize      = errors.New("无效的图片尺寸")
	ErrInvalidAspectRatio    = errors.New("无效的宽高比")
	ErrImageSaveFailed       = errors.New("图片保存失败")

	// 视频生成相关错误
	ErrVideoGenerationFailed = errors.New("视频生成失败")
	ErrInvalidDuration       = errors.New("无效的视频时长")
	ErrInvalidResolution     = errors.New("无效的视频分辨率")
	ErrInvalidFrameRate      = errors.New("无效的帧率")
	ErrVideoSaveFailed       = errors.New("视频保存失败")

	// API相关错误
	ErrAPIKeyMissing    = errors.New("API密钥缺失")
	ErrAPIQuotaExceeded = errors.New("API配额已超限")
	ErrAPIRateLimited   = errors.New("API请求频率超限")
	ErrAPIServiceError  = errors.New("API服务错误")
	ErrNetworkError     = errors.New("网络连接错误")
	ErrTimeoutError     = errors.New("请求超时")
)

// GenerationError 生成错误结构
type GenerationError struct {
	Type    string `json:"type"`    // 错误类型
	Message string `json:"message"` // 错误消息
	Code    int    `json:"code"`    // 错误代码
	Details string `json:"details"` // 错误详情
}

func (e *GenerationError) Error() string {
	return e.Message
}

// NewGenerationError 创建生成错误
func NewGenerationError(errType, message string, code int) *GenerationError {
	return &GenerationError{
		Type:    errType,
		Message: message,
		Code:    code,
	}
}

// WithDetails 添加错误详情
func (e *GenerationError) WithDetails(details string) *GenerationError {
	e.Details = details
	return e
}
