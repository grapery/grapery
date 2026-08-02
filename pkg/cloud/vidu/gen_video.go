package vidu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ViduClient Vidu API 客户端
type ViduClient struct {
	APIKey     string        // API 密钥
	BaseURL    string        // API 基础 URL
	HTTPClient *http.Client  // HTTP 客户端
	Timeout    time.Duration // 请求超时时间
}

// NewViduClient 创建新的 Vidu 客户端
func NewViduClient(apiKey string) *ViduClient {
	return &ViduClient{
		APIKey:  apiKey,
		BaseURL: "https://api.vidu.cn/v1",
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		Timeout: 60 * time.Second,
	}
}

// NewViduClientWithConfig 使用自定义配置创建 Vidu 客户端
func NewViduClientWithConfig(apiKey, baseURL string, timeout time.Duration) *ViduClient {
	return &ViduClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Timeout: timeout,
	}
}

// 错误定义
var (
	ErrAPIKeyMissing            = fmt.Errorf("API 密钥不能为空")
	ErrInvalidImageURL          = fmt.Errorf("无效的图片 URL")
	ErrInvalidVideoURL          = fmt.Errorf("无效的视频 URL")
	ErrInvalidReferenceVideoURL = fmt.Errorf("无效的参考视频 URL")
	ErrInvalidStartImageURL     = fmt.Errorf("无效的开始图片 URL")
	ErrInvalidEndImageURL       = fmt.Errorf("无效的结束图片 URL")
	ErrEmptyPrompt              = fmt.Errorf("提示词不能为空")
	ErrInvalidDuration          = fmt.Errorf("无效的视频时长")
	ErrInvalidResolution        = fmt.Errorf("无效的视频分辨率")
	ErrInvalidSimilarity        = fmt.Errorf("无效的相似度值，应在 0.0-1.0 之间")
	ErrInvalidMotionStrength    = fmt.Errorf("无效的运动强度值，应在 0.0-1.0 之间")
	ErrInvalidTransitionType    = fmt.Errorf("无效的过渡类型")
	ErrInvalidTransitionSpeed   = fmt.Errorf("无效的过渡速度值，应在 0.1-2.0 之间")
	ErrInvalidMotionIntensity   = fmt.Errorf("无效的运动强度值，应在 0.0-1.0 之间")
	ErrTaskNotFound             = fmt.Errorf("任务不存在")
	ErrTaskFailed               = fmt.Errorf("任务执行失败")
)

// 任务状态常量
const (
	TaskStatusPending    = "pending"    // 等待中
	TaskStatusProcessing = "processing" // 处理中
	TaskStatusSuccess    = "success"    // 成功
	TaskStatusFailed     = "failed"     // 失败
)

// 过渡类型常量
const (
	TransitionTypeSmooth   = "smooth"   // 平滑过渡
	TransitionTypeMorph    = "morph"    // 变形过渡
	TransitionTypeDissolve = "dissolve" // 溶解过渡
	TransitionTypeFade     = "fade"     // 淡入淡出
)

// ImageToVideoRequest 图片转视频请求
type ImageToVideoRequest struct {
	ImageURL       string `json:"image_url"`                 // 图片 URL
	Prompt         string `json:"prompt"`                    // 视频描述提示词
	Duration       int    `json:"duration"`                  // 视频时长（秒），默认 5 秒
	Resolution     string `json:"resolution"`                // 视频分辨率，默认 "1280x720"
	FrameRate      int    `json:"frame_rate"`                // 帧率，默认 24
	Style          string `json:"style"`                     // 视频风格
	NegativePrompt string `json:"negative_prompt,omitempty"` // 负面提示词
}

// VideoStyleTransferRequest 视频风格转换请求
type VideoStyleTransferRequest struct {
	VideoURL       string `json:"video_url"`                 // 视频 URL
	Prompt         string `json:"prompt"`                    // 风格描述提示词
	Resolution     string `json:"resolution"`                // 视频分辨率，默认 "1280x720"
	FrameRate      int    `json:"frame_rate"`                // 帧率，默认 24
	Style          string `json:"style"`                     // 视频风格
	NegativePrompt string `json:"negative_prompt,omitempty"` // 负面提示词
}

// ReferenceToVideoRequest 参考视频生成请求
type ReferenceToVideoRequest struct {
	ReferenceVideoURL string  `json:"reference_video_url"`       // 参考视频 URL
	Prompt            string  `json:"prompt"`                    // 视频描述提示词
	Duration          int     `json:"duration"`                  // 视频时长（秒），默认 5 秒
	Resolution        string  `json:"resolution"`                // 视频分辨率，默认 "1280x720"
	FrameRate         int     `json:"frame_rate"`                // 帧率，默认 24
	Style             string  `json:"style"`                     // 视频风格
	NegativePrompt    string  `json:"negative_prompt,omitempty"` // 负面提示词
	Similarity        float64 `json:"similarity,omitempty"`      // 相似度控制 (0.0-1.0)，默认 0.8
	MotionStrength    float64 `json:"motion_strength,omitempty"` // 运动强度 (0.0-1.0)，默认 0.5
}

// StartEndToVideoRequest 开始结束图片转视频请求
type StartEndToVideoRequest struct {
	StartImageURL   string  `json:"start_image_url"`            // 开始图片 URL
	EndImageURL     string  `json:"end_image_url"`              // 结束图片 URL
	Prompt          string  `json:"prompt"`                     // 视频描述提示词
	Duration        int     `json:"duration"`                   // 视频时长（秒），默认 5 秒
	Resolution      string  `json:"resolution"`                 // 视频分辨率，默认 "1280x720"
	FrameRate       int     `json:"frame_rate"`                 // 帧率，默认 24
	Style           string  `json:"style"`                      // 视频风格
	NegativePrompt  string  `json:"negative_prompt,omitempty"`  // 负面提示词
	TransitionType  string  `json:"transition_type,omitempty"`  // 过渡类型：smooth, morph, dissolve, fade
	TransitionSpeed float64 `json:"transition_speed,omitempty"` // 过渡速度 (0.1-2.0)，默认 1.0
	MotionIntensity float64 `json:"motion_intensity,omitempty"` // 运动强度 (0.0-1.0)，默认 0.5
}

// TaskResponse 任务响应
type TaskResponse struct {
	RequestID string `json:"request_id"` // 请求 ID
	TaskID    string `json:"task_id"`    // 任务 ID
	Status    string `json:"status"`     // 任务状态
	Message   string `json:"message"`    // 状态消息
	CreatedAt string `json:"created_at"` // 创建时间
	UpdatedAt string `json:"updated_at"` // 更新时间
}

// TaskStatusResponse 任务状态查询响应
type TaskStatusResponse struct {
	RequestID string `json:"request_id"` // 请求 ID
	TaskID    string `json:"task_id"`    // 任务 ID
	Status    string `json:"status"`     // 任务状态
	Progress  int    `json:"progress"`   // 进度百分比 (0-100)
	Message   string `json:"message"`    // 状态消息
	Result    struct {
		VideoURL string `json:"video_url"` // 生成的视频 URL
		Duration int    `json:"duration"`  // 视频时长（秒）
		Size     int64  `json:"size"`      // 文件大小（字节）
		Format   string `json:"format"`    // 视频格式
	} `json:"result"` // 结果信息
	Error struct {
		Code    string `json:"code"`    // 错误代码
		Message string `json:"message"` // 错误消息
	} `json:"error"` // 错误信息
	CreatedAt string `json:"created_at"` // 创建时间
	UpdatedAt string `json:"updated_at"` // 更新时间
}

// Validate 验证图片转视频请求
func (req *ImageToVideoRequest) Validate() error {
	if req.ImageURL == "" {
		return ErrInvalidImageURL
	}
	if req.Prompt == "" {
		return ErrEmptyPrompt
	}
	if req.Duration <= 0 {
		req.Duration = 5 // 默认 5 秒
	}
	if req.Duration > 30 {
		return ErrInvalidDuration
	}
	if req.Resolution == "" {
		req.Resolution = "1280x720" // 默认 720p
	}
	if req.FrameRate <= 0 {
		req.FrameRate = 24 // 默认 24fps
	}
	return nil
}

// Validate 验证视频风格转换请求
func (req *VideoStyleTransferRequest) Validate() error {
	if req.VideoURL == "" {
		return ErrInvalidVideoURL
	}
	if req.Prompt == "" {
		return ErrEmptyPrompt
	}
	if req.Resolution == "" {
		req.Resolution = "1280x720" // 默认 720p
	}
	if req.FrameRate <= 0 {
		req.FrameRate = 24 // 默认 24fps
	}
	return nil
}

// Validate 验证参考视频生成请求
func (req *ReferenceToVideoRequest) Validate() error {
	if req.ReferenceVideoURL == "" {
		return ErrInvalidReferenceVideoURL
	}
	if req.Prompt == "" {
		return ErrEmptyPrompt
	}
	if req.Duration <= 0 {
		req.Duration = 5 // 默认 5 秒
	}
	if req.Duration > 30 {
		return ErrInvalidDuration
	}
	if req.Resolution == "" {
		req.Resolution = "1280x720" // 默认 720p
	}
	if req.FrameRate <= 0 {
		req.FrameRate = 24 // 默认 24fps
	}
	if req.Similarity < 0 || req.Similarity > 1 {
		req.Similarity = 0.8 // 默认相似度
	}
	if req.MotionStrength < 0 || req.MotionStrength > 1 {
		req.MotionStrength = 0.5 // 默认运动强度
	}
	return nil
}

// Validate 验证开始结束图片转视频请求
func (req *StartEndToVideoRequest) Validate() error {
	if req.StartImageURL == "" {
		return ErrInvalidStartImageURL
	}
	if req.EndImageURL == "" {
		return ErrInvalidEndImageURL
	}
	if req.Prompt == "" {
		return ErrEmptyPrompt
	}
	if req.Duration <= 0 {
		req.Duration = 5 // 默认 5 秒
	}
	if req.Duration > 30 {
		return ErrInvalidDuration
	}
	if req.Resolution == "" {
		req.Resolution = "1280x720" // 默认 720p
	}
	if req.FrameRate <= 0 {
		req.FrameRate = 24 // 默认 24fps
	}
	if req.TransitionType == "" {
		req.TransitionType = TransitionTypeSmooth // 默认平滑过渡
	} else {
		// 验证过渡类型
		validTypes := []string{TransitionTypeSmooth, TransitionTypeMorph, TransitionTypeDissolve, TransitionTypeFade}
		valid := false
		for _, t := range validTypes {
			if req.TransitionType == t {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidTransitionType
		}
	}
	if req.TransitionSpeed < 0.1 || req.TransitionSpeed > 2.0 {
		req.TransitionSpeed = 1.0 // 默认过渡速度
	}
	if req.MotionIntensity < 0 || req.MotionIntensity > 1 {
		req.MotionIntensity = 0.5 // 默认运动强度
	}
	return nil
}

// GenerateVideoFromImage 根据图片生成视频
func (c *ViduClient) GenerateVideoFromImage(ctx context.Context, req *ImageToVideoRequest) (*TaskResponse, error) {
	// 验证请求参数
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 验证 API 密钥
	if c.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"image_url":  req.ImageURL,
		"prompt":     req.Prompt,
		"duration":   req.Duration,
		"resolution": req.Resolution,
		"frame_rate": req.FrameRate,
	}

	// 添加可选参数
	if req.Style != "" {
		requestBody["style"] = req.Style
	}
	if req.NegativePrompt != "" {
		requestBody["negative_prompt"] = req.NegativePrompt
	}

	// 发送请求
	return c.sendRequest(ctx, "POST", "/image-to-video", requestBody)
}

// VideoStyleTransfer 视频风格转换
func (c *ViduClient) VideoStyleTransfer(ctx context.Context, req *VideoStyleTransferRequest) (*TaskResponse, error) {
	// 验证请求参数
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 验证 API 密钥
	if c.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"video_url":  req.VideoURL,
		"prompt":     req.Prompt,
		"resolution": req.Resolution,
		"frame_rate": req.FrameRate,
	}

	// 添加可选参数
	if req.Style != "" {
		requestBody["style"] = req.Style
	}
	if req.NegativePrompt != "" {
		requestBody["negative_prompt"] = req.NegativePrompt
	}

	// 发送请求
	return c.sendRequest(ctx, "POST", "/video-style-transfer", requestBody)
}

// GenerateVideoFromReference 根据参考视频生成视频
func (c *ViduClient) GenerateVideoFromReference(ctx context.Context, req *ReferenceToVideoRequest) (*TaskResponse, error) {
	// 验证请求参数
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 验证 API 密钥
	if c.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"reference_video_url": req.ReferenceVideoURL,
		"prompt":              req.Prompt,
		"duration":            req.Duration,
		"resolution":          req.Resolution,
		"frame_rate":          req.FrameRate,
		"similarity":          req.Similarity,
		"motion_strength":     req.MotionStrength,
	}

	// 添加可选参数
	if req.Style != "" {
		requestBody["style"] = req.Style
	}
	if req.NegativePrompt != "" {
		requestBody["negative_prompt"] = req.NegativePrompt
	}

	// 发送请求
	return c.sendRequest(ctx, "POST", "/reference-to-video", requestBody)
}

// GenerateVideoFromStartEnd 根据开始和结束图片生成视频
func (c *ViduClient) GenerateVideoFromStartEnd(ctx context.Context, req *StartEndToVideoRequest) (*TaskResponse, error) {
	// 验证请求参数
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 验证 API 密钥
	if c.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"start_image_url":  req.StartImageURL,
		"end_image_url":    req.EndImageURL,
		"prompt":           req.Prompt,
		"duration":         req.Duration,
		"resolution":       req.Resolution,
		"frame_rate":       req.FrameRate,
		"transition_type":  req.TransitionType,
		"transition_speed": req.TransitionSpeed,
		"motion_intensity": req.MotionIntensity,
	}

	// 添加可选参数
	if req.Style != "" {
		requestBody["style"] = req.Style
	}
	if req.NegativePrompt != "" {
		requestBody["negative_prompt"] = req.NegativePrompt
	}

	// 发送请求
	return c.sendRequest(ctx, "POST", "/start-end-to-video", requestBody)
}

// GetTaskStatus 查询任务状态
func (c *ViduClient) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatusResponse, error) {
	// 验证 API 密钥
	if c.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// 验证任务 ID
	if taskID == "" {
		return nil, fmt.Errorf("任务 ID 不能为空")
	}

	// 构建 URL
	url := fmt.Sprintf("%s/tasks/%s", c.BaseURL, taskID)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		var apiErr APIErrorResponse
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return nil, fmt.Errorf("API 错误: %s", apiErr.Error.Message)
		}
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var taskStatus TaskStatusResponse
	if err := json.Unmarshal(body, &taskStatus); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &taskStatus, nil
}

// sendRequest 发送 HTTP 请求的通用方法
func (c *ViduClient) sendRequest(ctx context.Context, method, endpoint string, body interface{}) (*TaskResponse, error) {
	// 构建 URL
	url := c.BaseURL + endpoint

	// 序列化请求体
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var apiErr APIErrorResponse
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return nil, fmt.Errorf("API 错误: %s", apiErr.Error.Message)
		}
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var taskResp TaskResponse
	if err := json.Unmarshal(respBody, &taskResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &taskResp, nil
}

// WaitForTaskCompletion 等待任务完成
func (c *ViduClient) WaitForTaskCompletion(ctx context.Context, taskID string, checkInterval time.Duration) (*TaskStatusResponse, error) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := c.GetTaskStatus(ctx, taskID)
			if err != nil {
				return nil, err
			}

			switch status.Status {
			case TaskStatusSuccess:
				return status, nil
			case TaskStatusFailed:
				return status, ErrTaskFailed
			case TaskStatusPending, TaskStatusProcessing:
				// 继续等待
				continue
			default:
				return status, fmt.Errorf("未知的任务状态: %s", status.Status)
			}
		}
	}
}

// DownloadVideo 下载视频到本地文件
func (c *ViduClient) DownloadVideo(ctx context.Context, videoURL, filename string) error {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err != nil {
		return fmt.Errorf("创建下载请求失败: %w", err)
	}

	// 发送请求
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 复制数据
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	return nil
}

// GetSupportedResolutions 获取支持的分辨率
func (c *ViduClient) GetSupportedResolutions() []string {
	return []string{
		"640x360",   // 360p
		"1280x720",  // 720p
		"1920x1080", // 1080p
		"2560x1440", // 1440p
		"3840x2160", // 4K
	}
}

// GetSupportedDurations 获取支持的时长范围
func (c *ViduClient) GetSupportedDurations() (int, int) {
	return 1, 30 // 1秒到30秒
}

// GetSupportedFrameRates 获取支持的帧率
func (c *ViduClient) GetSupportedFrameRates() []int {
	return []int{24, 25, 30, 50, 60}
}

// GetSupportedFormats 获取支持的视频格式
func (c *ViduClient) GetSupportedFormats() []string {
	return []string{"mp4", "webm", "mov"}
}

// GetSupportedStyles 获取支持的视频风格
func (c *ViduClient) GetSupportedStyles() []string {
	return []string{
		"realistic",    // 写实风格
		"anime",        // 动漫风格
		"cartoon",      // 卡通风格
		"oil_painting", // 油画风格
		"watercolor",   // 水彩风格
		"sketch",       // 素描风格
		"cyberpunk",    // 赛博朋克风格
		"vintage",      // 复古风格
		"fantasy",      // 奇幻风格
		"documentary",  // 纪录片风格
	}
}

// GetSupportedTransitionTypes 获取支持的过渡类型
func (c *ViduClient) GetSupportedTransitionTypes() []string {
	return []string{
		TransitionTypeSmooth,   // 平滑过渡
		TransitionTypeMorph,    // 变形过渡
		TransitionTypeDissolve, // 溶解过渡
		TransitionTypeFade,     // 淡入淡出
	}
}

// IsTaskCompleted 检查任务是否已完成
func (status *TaskStatusResponse) IsTaskCompleted() bool {
	return status.Status == TaskStatusSuccess || status.Status == TaskStatusFailed
}

// IsTaskSuccessful 检查任务是否成功
func (status *TaskStatusResponse) IsTaskSuccessful() bool {
	return status.Status == TaskStatusSuccess
}

// HasError 检查任务是否有错误
func (status *TaskStatusResponse) HasError() bool {
	return status.Status == TaskStatusFailed || status.Error.Code != ""
}

// GetErrorMessage 获取错误消息
func (status *TaskStatusResponse) GetErrorMessage() string {
	if status.Error.Message != "" {
		return status.Error.Message
	}
	return status.Message
}

// APIErrorResponse API 错误响应
type APIErrorResponse struct {
	Error struct {
		Code    string `json:"code"`    // 错误代码
		Message string `json:"message"` // 错误消息
		Type    string `json:"type"`    // 错误类型
	} `json:"error"`
}
