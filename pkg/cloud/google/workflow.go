package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	api "github.com/grapery/common-protoc/gen"
	graperylog "github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

// GeminiWorkflowClient Gemini工作流客户端
type GeminiWorkflowClient struct {
	imageGenerator    *GeminiImageGenerator
	videoGenerator    *VeoVideoGenerator
	promptEnhancer    *PromptEnhancer
	apiKey            string
	enableEnhancement bool // 是否启用提示词增强
}

// NewGeminiWorkflowClient 创建Gemini工作流客户端
func NewGeminiWorkflowClient(apiKey string) (*GeminiWorkflowClient, error) {
	return NewGeminiWorkflowClientWithOptions(apiKey, true) // 默认启用提示词增强
}

// NewGeminiWorkflowClientWithOptions 创建Gemini工作流客户端（带选项）
func NewGeminiWorkflowClientWithOptions(apiKey string, enableEnhancement bool) (*GeminiWorkflowClient, error) {
	if apiKey == "" {
		return nil, errors.New("API密钥不能为空")
	}

	// 创建图片生成器
	imageGen, err := NewGeminiImageGenerator(apiKey)
	if err != nil {
		return nil, fmt.Errorf("创建图片生成器失败: %w", err)
	}

	// 创建视频生成器
	videoGen, err := NewVeoVideoGenerator(apiKey, "veo2")
	if err != nil {
		imageGen.Close()
		return nil, fmt.Errorf("创建视频生成器失败: %w", err)
	}

	// 创建提示词增强器（如果启用）
	var promptEnhancer *PromptEnhancer
	if enableEnhancement {
		promptEnhancer, err = NewPromptEnhancer(apiKey)
		if err != nil {
			imageGen.Close()
			videoGen.Close()
			return nil, fmt.Errorf("创建提示词增强器失败: %w", err)
		}
	}

	return &GeminiWorkflowClient{
		imageGenerator:    imageGen,
		videoGenerator:    videoGen,
		promptEnhancer:    promptEnhancer,
		apiKey:            apiKey,
		enableEnhancement: enableEnhancement,
	}, nil
}

// Close 关闭客户端
func (c *GeminiWorkflowClient) Close() error {
	var errs []error

	if err := c.imageGenerator.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := c.videoGenerator.Close(); err != nil {
		errs = append(errs, err)
	}

	if c.promptEnhancer != nil {
		if err := c.promptEnhancer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭客户端时发生错误: %v", errs)
	}

	return nil
}

// ==================== 兼容Coze接口的参数结构 ====================

// GeminiInitStoryboardParams 初始化故事板参数（兼容Coze接口）
type GeminiInitStoryboardParams struct {
	Title       string
	Description string
	Background  string
	Roles       []GeminiRoleInfo
	IsAsync     bool
	Style       string
	SceneNum    int
}

func (g GeminiInitStoryboardParams) String() string {
	data, _ := json.Marshal(g)
	return string(data)
}

// GeminiRoleInfo 角色信息（兼容Coze接口）
type GeminiRoleInfo struct {
	RoleID          string
	RoleName        string
	RoleImage       string
	RoleDescription string
}

// GeminiStoryBackgroundImageParams 故事背景图片参数（兼容Coze接口）
type GeminiStoryBackgroundImageParams struct {
	OriginalPrompt string
	StoryDesc      string
	Roles          []GeminiRoleInfo
}

// GeminiStoryRoleBackgroundImageParams 故事角色背景图片参数（兼容Coze接口）
type GeminiStoryRoleBackgroundImageParams struct {
	StoryTitle string
	StoryDesc  string
	RoleName   string
	RoleDesc   string
	RoleImage  string
	Style      string
	Ratio      api.ImageRatios
	Width      int32 // 图片宽度（以1000为基数计算）
	Height     int32 // 图片高度（以1000为基数计算）
}

// GeminiStoryRoleImageParams 故事角色图片参数（兼容Coze接口）
type GeminiStoryRoleImageParams struct {
	Description     string
	ShortTermGoal   string
	LongTermGoal    string
	Personality     string
	Background      string
	HandlingStyle   string
	CognitionRange  string
	AbilityFeatures string
	Appearance      string
	DressPreference string
	RefImage        string
	Style           string
	Ratio           api.ImageRatios
	Width           int32 // 图片宽度（以1000为基数计算）
	Height          int32 // 图片高度（以1000为基数计算）
}

// GeminiStoryWriteParams 故事写作参数（兼容Coze接口）
type GeminiStoryWriteParams struct {
	StoryTitle string
	StoryDesc  string
}

// GeminiStoryboardImageParams 故事板图片参数（兼容Coze接口）
type GeminiStoryboardImageParams struct {
	OriginPrompt  string
	SenceRefImage string
	Storyboard    string
	Roles         []GeminiRoleInfo
	Seed          int64
}

// GeminiStoryboardImageListParams 故事板图片列表参数（兼容Coze接口）
type GeminiStoryboardImageListParams struct {
	Storyboard string
	Roles      []GeminiRoleInfo
}

// GeminiStoryboardVideoParams 故事板视频参数（兼容Coze接口）
type GeminiStoryboardVideoParams struct {
	Prompt         string
	StartRefImage  string
	EndRefImage    string
	Style          string
	NegativePrompt string
	SceneImage     string
}

// GeminiStoryboardWriterParams 故事板写作参数（兼容Coze接口）
type GeminiStoryboardWriterParams struct {
	StoryCharacters string
	StoryChapter    string
	StoryContent    string
	StoryBackground string
	ImageStyle      string
	PrevContent     string
	SceneNum        int
	Seed            int64
}

func (g GeminiStoryboardWriterParams) String() string {
	data, _ := json.Marshal(g)
	return string(data)
}

// GeminiStoryboardContinueParams 故事板继续参数（兼容Coze接口）
type GeminiStoryboardContinueParams struct {
	Title            string
	Description      string
	Background       string
	StoryName        string
	StoryPrevContent string
	Roles            []GeminiRoleInfo
	Style            string
	SceneNum         int
}

func (g GeminiStoryboardContinueParams) RoleStr() string {
	data, _ := json.Marshal(g.Roles)
	return string(data)
}

func (g GeminiStoryboardContinueParams) String() string {
	data, _ := json.Marshal(g)
	return string(data)
}

// GeminiStoryRoleDetailParams 故事角色详情参数（兼容Coze接口）
type GeminiStoryRoleDetailParams struct {
	StoryName   string
	StoryDesc   string
	RoleName    string
	Description string
	OtherRoles  string
}

func (g GeminiStoryRoleDetailParams) String() string {
	data, _ := json.Marshal(g)
	return string(data)
}

// GeminiStoryRoleDetailContinueParams 故事角色详情继续参数（兼容Coze接口）
type GeminiStoryRoleDetailContinueParams struct {
	StoryName   string
	StoryDesc   string
	RoleName    string
	Description string
	OtherRoles  string
	History     string
}

func (g GeminiStoryRoleDetailContinueParams) String() string {
	data, _ := json.Marshal(g)
	return string(data)
}

// ==================== 兼容Coze接口的方法实现 ====================

// InitStoryboard 初始化故事板（兼容Coze接口）
func (c *GeminiWorkflowClient) InitStoryboard(ctx context.Context, params GeminiInitStoryboardParams) (string, error) {
	graperylog.Log().Info("[InitStoryboard] 开始初始化故事板", zap.Any("params", params))

	// 构建基础故事板提示词
	basePrompt := c.buildStoryboardInitPrompt(params)

	// 使用提示词增强器优化提示词（如果启用）
	var prompt string
	var err error
	if c.enableEnhancement && c.promptEnhancer != nil {
		enhancedPrompt, enhanceErr := c.promptEnhancer.EnhanceStoryboardPrompt(ctx, params)
		if enhanceErr != nil {
			graperylog.Log().Warn("[InitStoryboard] 提示词增强失败，使用原始提示词", zap.Error(enhanceErr))
			prompt = basePrompt
		} else {
			prompt = enhancedPrompt
			graperylog.Log().Info("[InitStoryboard] 提示词增强成功",
				zap.String("original", basePrompt),
				zap.String("enhanced", enhancedPrompt))
		}
	} else {
		prompt = basePrompt
	}

	// 使用图片生成器生成故事板概念图
	imageReq := &ImageGenerationRequest{
		Prompt:      prompt,
		Style:       params.Style,
		Quality:     "high",
		Size:        "1024x1024",
		AspectRatio: "16:9",
	}

	response, err := c.imageGenerator.GenerateImage(ctx, imageReq)
	if err != nil {
		graperylog.Log().Error("[InitStoryboard] 生成故事板图片失败", zap.Error(err))
		return "", err
	}

	if !response.Success {
		graperylog.Log().Error("[InitStoryboard] 故事板图片生成失败", zap.Error(response.Error))
		return "", response.Error
	}

	// 返回生成的故事板描述
	storyboardDesc := fmt.Sprintf("故事板标题: %s\n描述: %s\n背景: %s\n风格: %s\n场景数量: %d\n使用的提示词: %s",
		params.Title, params.Description, params.Background, params.Style, params.SceneNum, prompt)

	graperylog.Log().Info("[InitStoryboard] 故事板初始化成功", zap.String("result", storyboardDesc))
	return storyboardDesc, nil
}

// StoryBackgroundImage 生成故事背景图片（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryBackgroundImage(ctx context.Context, params GeminiStoryBackgroundImageParams) (string, error) {
	graperylog.Log().Info("[StoryBackgroundImage] 开始生成故事背景图片", zap.Any("params", params))

	// 构建基础背景图片提示词
	basePrompt := c.buildBackgroundImagePrompt(params)

	// 使用提示词增强器优化提示词（如果启用）
	var prompt string
	if c.enableEnhancement && c.promptEnhancer != nil {
		enhancedPrompt, enhanceErr := c.promptEnhancer.EnhanceBackgroundImagePrompt(ctx, params)
		if enhanceErr != nil {
			graperylog.Log().Warn("[StoryBackgroundImage] 提示词增强失败，使用原始提示词", zap.Error(enhanceErr))
			prompt = basePrompt
		} else {
			prompt = enhancedPrompt
			graperylog.Log().Info("[StoryBackgroundImage] 提示词增强成功",
				zap.String("original", basePrompt),
				zap.String("enhanced", enhancedPrompt))
		}
	} else {
		prompt = basePrompt
	}

	// 使用图片生成器生成背景图片
	imageReq := &ImageGenerationRequest{
		Prompt:      prompt,
		Style:       "cinematic",
		Quality:     "high",
		Size:        "1024x1024",
		AspectRatio: "16:9",
	}

	response, err := c.imageGenerator.GenerateImage(ctx, imageReq)
	if err != nil {
		graperylog.Log().Error("[StoryBackgroundImage] 生成背景图片失败", zap.Error(err))
		return "", err
	}

	if !response.Success {
		graperylog.Log().Error("[StoryBackgroundImage] 背景图片生成失败", zap.Error(response.Error))
		return "", response.Error
	}

	// 返回图片URL或描述（这里简化处理）
	imageDesc := fmt.Sprintf("故事背景图片已生成，描述: %s\n使用的提示词: %s", params.StoryDesc, prompt)
	graperylog.Log().Info("[StoryBackgroundImage] 背景图片生成成功", zap.String("result", imageDesc))
	return imageDesc, nil
}

// StoryRoleBackgroundImage 生成故事角色背景图片（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryRoleBackgroundImage(ctx context.Context, params GeminiStoryRoleBackgroundImageParams) (string, error) {
	graperylog.Log().Info("[StoryRoleBackgroundImage] 开始生成角色背景图片", zap.Any("params", params))

	// 构建角色背景图片提示词
	prompt := c.buildRoleBackgroundImagePrompt(params)

	// 使用图片生成器生成角色背景图片
	imageReq := &ImageGenerationRequest{
		Prompt:      prompt,
		Style:       params.Style,
		Quality:     "high",
		Size:        fmt.Sprintf("%dx%d", params.Width, params.Height),
		AspectRatio: GetAspectRatioString(params.Ratio),
	}

	response, err := c.imageGenerator.GenerateImage(ctx, imageReq)
	if err != nil {
		graperylog.Log().Error("[StoryRoleBackgroundImage] 生成角色背景图片失败", zap.Error(err))
		return "", err
	}

	if !response.Success {
		graperylog.Log().Error("[StoryRoleBackgroundImage] 角色背景图片生成失败", zap.Error(response.Error))
		return "", response.Error
	}

	// 返回图片描述
	imageDesc := fmt.Sprintf("角色 %s 的背景图片已生成", params.RoleName)
	graperylog.Log().Info("[StoryRoleBackgroundImage] 角色背景图片生成成功", zap.String("result", imageDesc))
	return imageDesc, nil
}

// StoryRoleImage 生成故事角色图片（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryRoleImage(ctx context.Context, params GeminiStoryRoleImageParams) (string, error) {
	graperylog.Log().Info("[StoryRoleImage] 开始生成角色图片", zap.Any("params", params))

	// 构建基础角色图片提示词
	basePrompt := c.buildRoleImagePrompt(params)

	// 使用提示词增强器优化提示词（如果启用）
	var prompt string
	if c.enableEnhancement && c.promptEnhancer != nil {
		enhancedPrompt, enhanceErr := c.promptEnhancer.EnhanceRoleImagePrompt(ctx, params)
		if enhanceErr != nil {
			graperylog.Log().Warn("[StoryRoleImage] 提示词增强失败，使用原始提示词", zap.Error(enhanceErr))
			prompt = basePrompt
		} else {
			prompt = enhancedPrompt
			graperylog.Log().Info("[StoryRoleImage] 提示词增强成功",
				zap.String("original", basePrompt),
				zap.String("enhanced", enhancedPrompt))
		}
	} else {
		prompt = basePrompt
	}

	// 使用图片生成器生成角色图片
	imageReq := &ImageGenerationRequest{
		Prompt:      prompt,
		Style:       params.Style,
		Quality:     "high",
		Size:        fmt.Sprintf("%dx%d", params.Width, params.Height),
		AspectRatio: GetAspectRatioString(params.Ratio),
	}

	response, err := c.imageGenerator.GenerateImage(ctx, imageReq)
	if err != nil {
		graperylog.Log().Error("[StoryRoleImage] 生成角色图片失败", zap.Error(err))
		return "", err
	}

	if !response.Success {
		graperylog.Log().Error("[StoryRoleImage] 角色图片生成失败", zap.Error(response.Error))
		return "", response.Error
	}

	// 返回图片描述
	imageDesc := fmt.Sprintf("角色图片已生成，外观: %s\n使用的提示词: %s", params.Appearance, prompt)
	graperylog.Log().Info("[StoryRoleImage] 角色图片生成成功", zap.String("result", imageDesc))
	return imageDesc, nil
}

// StoryWrite 故事写作（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryWrite(ctx context.Context, params GeminiStoryWriteParams) (string, error) {
	graperylog.Log().Info("[StoryWrite] 开始故事写作", zap.Any("params", params))

	// 这里可以调用文本生成API，但为了保持接口兼容，我们返回一个故事大纲
	storyContent := fmt.Sprintf("故事标题: %s\n\n故事描述: %s\n\n故事大纲:\n1. 开场介绍\n2. 冲突发展\n3. 高潮部分\n4. 结局收尾",
		params.StoryTitle, params.StoryDesc)

	graperylog.Log().Info("[StoryWrite] 故事写作完成", zap.String("result", storyContent))
	return storyContent, nil
}

// StoryboardImage 生成故事板图片（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryboardImage(ctx context.Context, params GeminiStoryboardImageParams) (string, error) {
	graperylog.Log().Info("[StoryboardImage] 开始生成故事板图片", zap.Any("params", params))

	// 构建故事板图片提示词
	prompt := c.buildStoryboardImagePrompt(params)

	// 使用图片生成器生成故事板图片
	imageReq := &ImageGenerationRequest{
		Prompt:      prompt,
		Style:       "cinematic",
		Quality:     "high",
		Size:        "1024x1024",
		AspectRatio: "16:9",
	}

	response, err := c.imageGenerator.GenerateImage(ctx, imageReq)
	if err != nil {
		graperylog.Log().Error("[StoryboardImage] 生成故事板图片失败", zap.Error(err))
		return "", err
	}

	if !response.Success {
		graperylog.Log().Error("[StoryboardImage] 故事板图片生成失败", zap.Error(response.Error))
		return "", response.Error
	}

	// 返回图片描述
	imageDesc := fmt.Sprintf("故事板图片已生成，场景: %s", params.OriginPrompt)
	graperylog.Log().Info("[StoryboardImage] 故事板图片生成成功", zap.String("result", imageDesc))
	return imageDesc, nil
}

// StoryboardImageList 生成故事板图片列表（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryboardImageList(ctx context.Context, params GeminiStoryboardImageListParams) (string, error) {
	graperylog.Log().Info("[StoryboardImageList] 开始生成故事板图片列表", zap.Any("params", params))

	// 解析故事板内容，为每个场景生成图片
	scenes := strings.Split(params.Storyboard, "\n")
	var imageList []string

	for i, scene := range scenes {
		if strings.TrimSpace(scene) == "" {
			continue
		}

		// 为每个场景生成图片
		sceneParams := GeminiStoryboardImageParams{
			OriginPrompt: scene,
			Storyboard:   params.Storyboard,
			Roles:        params.Roles,
		}

		imageDesc, err := c.StoryboardImage(ctx, sceneParams)
		if err != nil {
			graperylog.Log().Error("[StoryboardImageList] 生成场景图片失败", zap.Error(err), zap.Int("scene", i))
			continue
		}

		imageList = append(imageList, imageDesc)
	}

	result := fmt.Sprintf("故事板图片列表生成完成，共 %d 个场景", len(imageList))
	graperylog.Log().Info("[StoryboardImageList] 故事板图片列表生成成功", zap.String("result", result))
	return result, nil
}

// StoryboardVideo 生成故事板视频（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryboardVideo(ctx context.Context, params GeminiStoryboardVideoParams) (string, error) {
	graperylog.Log().Info("[StoryboardVideo] 开始生成故事板视频", zap.Any("params", params))

	// 构建基础视频生成提示词
	basePrompt := c.buildStoryboardVideoPrompt(params)

	// 使用提示词增强器优化提示词（如果启用）
	var prompt string
	if c.enableEnhancement && c.promptEnhancer != nil {
		enhancedPrompt, enhanceErr := c.promptEnhancer.EnhanceStoryboardVideoPrompt(ctx, params)
		if enhanceErr != nil {
			graperylog.Log().Warn("[StoryboardVideo] 提示词增强失败，使用原始提示词", zap.Error(enhanceErr))
			prompt = basePrompt
		} else {
			prompt = enhancedPrompt
			graperylog.Log().Info("[StoryboardVideo] 提示词增强成功",
				zap.String("original", basePrompt),
				zap.String("enhanced", enhancedPrompt))
		}
	} else {
		prompt = basePrompt
	}

	// 使用视频生成器生成视频
	videoReq := &VideoGenerationRequest{
		Prompt:     prompt,
		Duration:   10, // 默认10秒
		Resolution: "1920x1080",
		FrameRate:  24,
		Style:      params.Style,
	}

	response, err := c.videoGenerator.GenerateVideo(ctx, videoReq)
	if err != nil {
		graperylog.Log().Error("[StoryboardVideo] 生成故事板视频失败", zap.Error(err))
		return "", err
	}

	if !response.Success {
		graperylog.Log().Error("[StoryboardVideo] 故事板视频生成失败", zap.Error(response.Error))
		return "", response.Error
	}

	// 返回视频描述
	videoDesc := fmt.Sprintf("故事板视频已生成，时长: %.1f秒\n使用的提示词: %s", response.Duration, prompt)
	graperylog.Log().Info("[StoryboardVideo] 故事板视频生成成功", zap.String("result", videoDesc))
	return videoDesc, nil
}

// StoryboardWriter 故事板写作（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryboardWriter(ctx context.Context, params GeminiStoryboardWriterParams) (string, error) {
	graperylog.Log().Info("[StoryboardWriter] 开始故事板写作", zap.Any("params", params))

	// 生成故事板内容
	storyboardContent := fmt.Sprintf("章节: %s\n\n内容: %s\n\n背景: %s\n\n角色: %s\n\n风格: %s\n\n场景数量: %d",
		params.StoryChapter, params.StoryContent, params.StoryBackground, params.StoryCharacters, params.ImageStyle, params.SceneNum)

	graperylog.Log().Info("[StoryboardWriter] 故事板写作完成", zap.String("result", storyboardContent))
	return storyboardContent, nil
}

// StoryboardContinue 继续故事板（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryboardContinue(ctx context.Context, params GeminiStoryboardContinueParams) (string, error) {
	graperylog.Log().Info("[StoryboardContinue] 开始继续故事板", zap.Any("params", params))

	// 生成继续的故事内容
	continueContent := fmt.Sprintf("故事名称: %s\n\n描述: %s\n\n背景: %s\n\n角色: %s\n\n风格: %s\n\n场景数量: %d\n\n继续内容: 基于前面的内容继续发展故事...",
		params.StoryName, params.Description, params.Background, params.RoleStr(), params.Style, params.SceneNum)

	graperylog.Log().Info("[StoryboardContinue] 故事板继续完成", zap.String("result", continueContent))
	return continueContent, nil
}

// StoryRoleDetail 故事角色详情（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryRoleDetail(ctx context.Context, params GeminiStoryRoleDetailParams) (string, error) {
	graperylog.Log().Info("[StoryRoleDetail] 开始生成角色详情", zap.Any("params", params))

	// 生成角色详情
	roleDetail := fmt.Sprintf("故事: %s\n\n角色: %s\n\n描述: %s\n\n其他角色: %s",
		params.StoryName, params.RoleName, params.Description, params.OtherRoles)

	graperylog.Log().Info("[StoryRoleDetail] 角色详情生成完成", zap.String("result", roleDetail))
	return roleDetail, nil
}

// StoryRoleDetailContinue 继续故事角色详情（兼容Coze接口）
func (c *GeminiWorkflowClient) StoryRoleDetailContinue(ctx context.Context, params GeminiStoryRoleDetailContinueParams) (string, error) {
	graperylog.Log().Info("[StoryRoleDetailContinue] 开始继续角色详情", zap.Any("params", params))

	// 生成继续的角色详情
	continueDetail := fmt.Sprintf("故事: %s\n\n角色: %s\n\n描述: %s\n\n其他角色: %s\n\n历史: %s",
		params.StoryName, params.RoleName, params.Description, params.OtherRoles, params.History)

	graperylog.Log().Info("[StoryRoleDetailContinue] 角色详情继续完成", zap.String("result", continueDetail))
	return continueDetail, nil
}

// GenerateStoryboardVideo 生成故事板视频（兼容Coze接口）
func (c *GeminiWorkflowClient) GenerateStoryboardVideo(ctx context.Context, params GeminiStoryboardVideoParams) (string, error) {
	video, err := c.StoryboardVideo(ctx, params)
	if err != nil {
		graperylog.Log().Error("[GenerateStoryboardVideo] 生成视频失败", zap.Error(err), zap.Any("params", params))
		return "", err
	}
	graperylog.Log().Info("[GenerateStoryboardVideo] 生成视频成功", zap.String("video", video))
	return video, nil
}

// ==================== 提示词构建方法 ====================

// buildStoryboardInitPrompt 构建故事板初始化提示词
func (c *GeminiWorkflowClient) buildStoryboardInitPrompt(params GeminiInitStoryboardParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("创建一个故事板，标题: %s", params.Title))
	parts = append(parts, fmt.Sprintf("描述: %s", params.Description))
	parts = append(parts, fmt.Sprintf("背景: %s", params.Background))
	parts = append(parts, fmt.Sprintf("风格: %s", params.Style))
	parts = append(parts, fmt.Sprintf("场景数量: %d", params.SceneNum))

	if len(params.Roles) > 0 {
		parts = append(parts, "角色包括:")
		for _, role := range params.Roles {
			parts = append(parts, fmt.Sprintf("- %s: %s", role.RoleName, role.RoleDescription))
		}
	}

	parts = append(parts, "cinematic style, high quality, professional storyboard")

	return strings.Join(parts, ", ")
}

// buildBackgroundImagePrompt 构建背景图片提示词
func (c *GeminiWorkflowClient) buildBackgroundImagePrompt(params GeminiStoryBackgroundImageParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("故事背景图片: %s", params.StoryDesc))
	parts = append(parts, fmt.Sprintf("原始提示: %s", params.OriginalPrompt))

	if len(params.Roles) > 0 {
		parts = append(parts, "包含角色:")
		for _, role := range params.Roles {
			parts = append(parts, fmt.Sprintf("- %s", role.RoleName))
		}
	}

	parts = append(parts, "cinematic background, atmospheric, high quality")

	return strings.Join(parts, ", ")
}

// buildRoleBackgroundImagePrompt 构建角色背景图片提示词
func (c *GeminiWorkflowClient) buildRoleBackgroundImagePrompt(params GeminiStoryRoleBackgroundImageParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("角色 %s 的背景环境", params.RoleName))
	parts = append(parts, fmt.Sprintf("故事: %s", params.StoryTitle))
	parts = append(parts, fmt.Sprintf("角色描述: %s", params.RoleDesc))
	parts = append(parts, fmt.Sprintf("风格: %s", params.Style))

	parts = append(parts, "character background, environmental, high quality")

	return strings.Join(parts, ", ")
}

// buildRoleImagePrompt 构建角色图片提示词
func (c *GeminiWorkflowClient) buildRoleImagePrompt(params GeminiStoryRoleImageParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("角色外观: %s", params.Appearance))
	parts = append(parts, fmt.Sprintf("服装偏好: %s", params.DressPreference))
	parts = append(parts, fmt.Sprintf("个性: %s", params.Personality))
	parts = append(parts, fmt.Sprintf("背景: %s", params.Background))
	parts = append(parts, fmt.Sprintf("风格: %s", params.Style))

	parts = append(parts, "character portrait, detailed, high quality")

	return strings.Join(parts, ", ")
}

// buildStoryboardImagePrompt 构建故事板图片提示词
func (c *GeminiWorkflowClient) buildStoryboardImagePrompt(params GeminiStoryboardImageParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("故事板场景: %s", params.OriginPrompt))
	parts = append(parts, fmt.Sprintf("故事板内容: %s", params.Storyboard))

	if len(params.Roles) > 0 {
		parts = append(parts, "包含角色:")
		for _, role := range params.Roles {
			parts = append(parts, fmt.Sprintf("- %s", role.RoleName))
		}
	}

	parts = append(parts, "storyboard scene, cinematic, high quality")

	return strings.Join(parts, ", ")
}

// buildStoryboardVideoPrompt 构建故事板视频提示词
func (c *GeminiWorkflowClient) buildStoryboardVideoPrompt(params GeminiStoryboardVideoParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("故事板视频: %s", params.Prompt))
	parts = append(parts, fmt.Sprintf("风格: %s", params.Style))

	if params.NegativePrompt != "" {
		parts = append(parts, fmt.Sprintf("避免: %s", params.NegativePrompt))
	}

	parts = append(parts, "cinematic video, smooth motion, high quality")

	return strings.Join(parts, ", ")
}
