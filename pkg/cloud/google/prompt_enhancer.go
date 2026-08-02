package google

import (
	"context"
	"fmt"
	"os"
	"strings"

	graperylog "github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

// PromptEnhancer 提示词增强器
type PromptEnhancer struct {
	client *genai.Client
	apiKey string
}

// NewPromptEnhancer 创建提示词增强器
func NewPromptEnhancer(apiKey string) (*PromptEnhancer, error) {
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// 设置API密钥环境变量
	os.Setenv("GOOGLE_API_KEY", apiKey)

	client, err := genai.NewClient(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建Gemini客户端失败: %w", err)
	}

	return &PromptEnhancer{
		client: client,
		apiKey: apiKey,
	}, nil
}

// Close 关闭客户端
func (p *PromptEnhancer) Close() error {
	return nil
}

// EnhanceImagePrompt 增强图片生成提示词
func (p *PromptEnhancer) EnhanceImagePrompt(ctx context.Context, basePrompt string, contextInfo map[string]interface{}) (string, error) {
	graperylog.Log().Info("[EnhanceImagePrompt] 开始增强图片提示词", zap.String("basePrompt", basePrompt))

	// 构建增强提示词的请求
	enhancementPrompt := p.buildImageEnhancementPrompt(basePrompt, contextInfo)

	// 调用Gemini LLM生成增强的提示词
	result, err := p.client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash", // 使用文本生成模型
		genai.Text(enhancementPrompt),
		nil,
	)

	if err != nil {
		graperylog.Log().Error("[EnhanceImagePrompt] 调用Gemini LLM失败", zap.Error(err))
		return basePrompt, err // 返回原始提示词
	}

	// 提取增强的提示词
	enhancedPrompt := basePrompt
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		if text := result.Candidates[0].Content.Parts[0].Text; text != "" {
			enhancedPrompt = strings.TrimSpace(text)
		}
	}

	graperylog.Log().Info("[EnhanceImagePrompt] 提示词增强完成",
		zap.String("original", basePrompt),
		zap.String("enhanced", enhancedPrompt))

	return enhancedPrompt, nil
}

// EnhanceVideoPrompt 增强视频生成提示词
func (p *PromptEnhancer) EnhanceVideoPrompt(ctx context.Context, basePrompt string, contextInfo map[string]interface{}) (string, error) {
	graperylog.Log().Info("[EnhanceVideoPrompt] 开始增强视频提示词", zap.String("basePrompt", basePrompt))

	// 构建增强提示词的请求
	enhancementPrompt := p.buildVideoEnhancementPrompt(basePrompt, contextInfo)

	// 调用Gemini LLM生成增强的提示词
	result, err := p.client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash", // 使用文本生成模型
		genai.Text(enhancementPrompt),
		nil,
	)

	if err != nil {
		graperylog.Log().Error("[EnhanceVideoPrompt] 调用Gemini LLM失败", zap.Error(err))
		return basePrompt, err // 返回原始提示词
	}

	// 提取增强的提示词
	enhancedPrompt := basePrompt
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		if text := result.Candidates[0].Content.Parts[0].Text; text != "" {
			enhancedPrompt = strings.TrimSpace(text)
		}
	}

	graperylog.Log().Info("[EnhanceVideoPrompt] 提示词增强完成",
		zap.String("original", basePrompt),
		zap.String("enhanced", enhancedPrompt))

	return enhancedPrompt, nil
}

// buildImageEnhancementPrompt 构建图片提示词增强请求
func (p *PromptEnhancer) buildImageEnhancementPrompt(basePrompt string, contextInfo map[string]interface{}) string {
	var parts []string

	parts = append(parts, "你是一个专业的AI图片生成提示词优化专家。请根据以下信息，将基础提示词优化为更适合Gemini图片生成的高质量提示词。")
	parts = append(parts, "")
	parts = append(parts, "基础提示词:")
	parts = append(parts, fmt.Sprintf("\"%s\"", basePrompt))
	parts = append(parts, "")

	// 添加上下文信息
	if len(contextInfo) > 0 {
		parts = append(parts, "上下文信息:")
		for key, value := range contextInfo {
			parts = append(parts, fmt.Sprintf("- %s: %v", key, value))
		}
		parts = append(parts, "")
	}

	parts = append(parts, "请按照以下要求优化提示词:")
	parts = append(parts, "1. 使用具体、详细的描述词汇")
	parts = append(parts, "2. 添加专业摄影和艺术术语")
	parts = append(parts, "3. 包含光照、构图、色彩等视觉元素")
	parts = append(parts, "4. 使用英文，因为Gemini对英文提示词效果更好")
	parts = append(parts, "5. 保持提示词长度在200-500字符之间")
	parts = append(parts, "6. 添加负面提示词避免不想要的效果")
	parts = append(parts, "7. 确保提示词符合Gemini 2.5 Flash Image的最佳实践")
	parts = append(parts, "")
	parts = append(parts, "请直接输出优化后的提示词，不要包含任何解释或额外文字:")

	return strings.Join(parts, "\n")
}

// buildVideoEnhancementPrompt 构建视频提示词增强请求
func (p *PromptEnhancer) buildVideoEnhancementPrompt(basePrompt string, contextInfo map[string]interface{}) string {
	var parts []string

	parts = append(parts, "你是一个专业的AI视频生成提示词优化专家。请根据以下信息，将基础提示词优化为更适合Gemini Veo视频生成的高质量提示词。")
	parts = append(parts, "")
	parts = append(parts, "基础提示词:")
	parts = append(parts, fmt.Sprintf("\"%s\"", basePrompt))
	parts = append(parts, "")

	// 添加上下文信息
	if len(contextInfo) > 0 {
		parts = append(parts, "上下文信息:")
		for key, value := range contextInfo {
			parts = append(parts, fmt.Sprintf("- %s: %v", key, value))
		}
		parts = append(parts, "")
	}

	parts = append(parts, "请按照以下要求优化提示词:")
	parts = append(parts, "1. 使用具体的动作和场景描述")
	parts = append(parts, "2. 添加电影摄影术语（镜头运动、景别等）")
	parts = append(parts, "3. 包含时间、节奏、氛围等动态元素")
	parts = append(parts, "4. 使用英文，因为Gemini对英文提示词效果更好")
	parts = append(parts, "5. 保持提示词长度在200-400字符之间")
	parts = append(parts, "6. 添加负面提示词避免不想要的效果")
	parts = append(parts, "7. 确保提示词符合Veo2/Veo3的最佳实践")
	parts = append(parts, "")
	parts = append(parts, "请直接输出优化后的提示词，不要包含任何解释或额外文字:")

	return strings.Join(parts, "\n")
}

// EnhanceStoryboardPrompt 增强故事板提示词
func (p *PromptEnhancer) EnhanceStoryboardPrompt(ctx context.Context, params GeminiInitStoryboardParams) (string, error) {
	// 构建基础故事板提示词
	basePrompt := p.buildBasicStoryboardPrompt(params)

	// 构建上下文信息
	contextInfo := map[string]interface{}{
		"故事标题": params.Title,
		"故事描述": params.Description,
		"背景设定": params.Background,
		"艺术风格": params.Style,
		"场景数量": params.SceneNum,
		"角色数量": len(params.Roles),
		"是否异步": params.IsAsync,
	}

	// 添加角色信息
	if len(params.Roles) > 0 {
		roles := make([]string, len(params.Roles))
		for i, role := range params.Roles {
			roles[i] = fmt.Sprintf("%s: %s", role.RoleName, role.RoleDescription)
		}
		contextInfo["角色列表"] = strings.Join(roles, "; ")
	}

	return p.EnhanceImagePrompt(ctx, basePrompt, contextInfo)
}

// EnhanceRoleImagePrompt 增强角色图片提示词
func (p *PromptEnhancer) EnhanceRoleImagePrompt(ctx context.Context, params GeminiStoryRoleImageParams) (string, error) {
	// 构建基础角色图片提示词
	basePrompt := p.buildBasicRoleImagePrompt(params)

	// 构建上下文信息
	contextInfo := map[string]interface{}{
		"角色描述": params.Description,
		"个性特征": params.Personality,
		"背景故事": params.Background,
		"外观描述": params.Appearance,
		"服装偏好": params.DressPreference,
		"短期目标": params.ShortTermGoal,
		"长期目标": params.LongTermGoal,
		"处理风格": params.HandlingStyle,
		"认知范围": params.CognitionRange,
		"能力特征": params.AbilityFeatures,
		"艺术风格": params.Style,
		"图片比例": string(params.Ratio),
	}

	return p.EnhanceImagePrompt(ctx, basePrompt, contextInfo)
}

// EnhanceBackgroundImagePrompt 增强背景图片提示词
func (p *PromptEnhancer) EnhanceBackgroundImagePrompt(ctx context.Context, params GeminiStoryBackgroundImageParams) (string, error) {
	// 构建基础背景图片提示词
	basePrompt := p.buildBasicBackgroundImagePrompt(params)

	// 构建上下文信息
	contextInfo := map[string]interface{}{
		"原始提示": params.OriginalPrompt,
		"故事描述": params.StoryDesc,
		"角色数量": len(params.Roles),
	}

	// 添加角色信息
	if len(params.Roles) > 0 {
		roles := make([]string, len(params.Roles))
		for i, role := range params.Roles {
			roles[i] = fmt.Sprintf("%s: %s", role.RoleName, role.RoleDescription)
		}
		contextInfo["角色列表"] = strings.Join(roles, "; ")
	}

	return p.EnhanceImagePrompt(ctx, basePrompt, contextInfo)
}

// EnhanceStoryboardVideoPrompt 增强故事板视频提示词
func (p *PromptEnhancer) EnhanceStoryboardVideoPrompt(ctx context.Context, params GeminiStoryboardVideoParams) (string, error) {
	// 构建基础视频提示词
	basePrompt := p.buildBasicStoryboardVideoPrompt(params)

	// 构建上下文信息
	contextInfo := map[string]interface{}{
		"视频提示":  params.Prompt,
		"开始参考图": params.StartRefImage,
		"结束参考图": params.EndRefImage,
		"艺术风格":  params.Style,
		"负面提示":  params.NegativePrompt,
		"场景图片":  params.SceneImage,
	}

	return p.EnhanceVideoPrompt(ctx, basePrompt, contextInfo)
}

// ==================== 基础提示词构建方法 ====================

// buildBasicStoryboardPrompt 构建基础故事板提示词
func (p *PromptEnhancer) buildBasicStoryboardPrompt(params GeminiInitStoryboardParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Create a storyboard for: %s", params.Title))
	parts = append(parts, fmt.Sprintf("Description: %s", params.Description))
	parts = append(parts, fmt.Sprintf("Background: %s", params.Background))
	parts = append(parts, fmt.Sprintf("Style: %s", params.Style))
	parts = append(parts, fmt.Sprintf("Number of scenes: %d", params.SceneNum))

	if len(params.Roles) > 0 {
		parts = append(parts, "Characters:")
		for _, role := range params.Roles {
			parts = append(parts, fmt.Sprintf("- %s: %s", role.RoleName, role.RoleDescription))
		}
	}

	return strings.Join(parts, ", ")
}

// buildBasicRoleImagePrompt 构建基础角色图片提示词
func (p *PromptEnhancer) buildBasicRoleImagePrompt(params GeminiStoryRoleImageParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Character portrait: %s", params.Description))
	parts = append(parts, fmt.Sprintf("Appearance: %s", params.Appearance))
	parts = append(parts, fmt.Sprintf("Personality: %s", params.Personality))
	parts = append(parts, fmt.Sprintf("Background: %s", params.Background))
	parts = append(parts, fmt.Sprintf("Style: %s", params.Style))

	return strings.Join(parts, ", ")
}

// buildBasicBackgroundImagePrompt 构建基础背景图片提示词
func (p *PromptEnhancer) buildBasicBackgroundImagePrompt(params GeminiStoryBackgroundImageParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Story background: %s", params.StoryDesc))
	parts = append(parts, fmt.Sprintf("Original prompt: %s", params.OriginalPrompt))

	if len(params.Roles) > 0 {
		parts = append(parts, "Including characters:")
		for _, role := range params.Roles {
			parts = append(parts, fmt.Sprintf("- %s", role.RoleName))
		}
	}

	return strings.Join(parts, ", ")
}

// buildBasicStoryboardVideoPrompt 构建基础故事板视频提示词
func (p *PromptEnhancer) buildBasicStoryboardVideoPrompt(params GeminiStoryboardVideoParams) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Storyboard video: %s", params.Prompt))
	parts = append(parts, fmt.Sprintf("Style: %s", params.Style))

	if params.NegativePrompt != "" {
		parts = append(parts, fmt.Sprintf("Avoid: %s", params.NegativePrompt))
	}

	return strings.Join(parts, ", ")
}
