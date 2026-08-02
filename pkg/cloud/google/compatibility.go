package google

import (
	"context"
	"fmt"
	"time"

	api "github.com/grapery/common-protoc/gen"
	graperylog "github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

// ==================== 兼容性适配器 ====================

// CozeCompatibilityAdapter Coze兼容性适配器
// 这个适配器让上层业务可以无缝从Coze切换到Gemini
type CozeCompatibilityAdapter struct {
	geminiClient *GeminiWorkflowClient
}

// NewCozeCompatibilityAdapter 创建Coze兼容性适配器
func NewCozeCompatibilityAdapter(apiKey string) (*CozeCompatibilityAdapter, error) {
	return NewCozeCompatibilityAdapterWithOptions(apiKey, true) // 默认启用提示词增强
}

// NewCozeCompatibilityAdapterWithOptions 创建Coze兼容性适配器（带选项）
func NewCozeCompatibilityAdapterWithOptions(apiKey string, enableEnhancement bool) (*CozeCompatibilityAdapter, error) {
	geminiClient, err := NewGeminiWorkflowClientWithOptions(apiKey, enableEnhancement)
	if err != nil {
		return nil, err
	}

	return &CozeCompatibilityAdapter{
		geminiClient: geminiClient,
	}, nil
}

// Close 关闭适配器
func (a *CozeCompatibilityAdapter) Close() error {
	return a.geminiClient.Close()
}

// ==================== 完全兼容Coze接口的方法 ====================

// InitStoryboard 初始化故事板（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) InitStoryboard(ctx context.Context, params CozeInitStoryboardParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiInitStoryboardParams{
		Title:       params.Title,
		Description: params.Description,
		Background:  params.Background,
		Roles:       convertCozeRolesToGemini(params.Roles),
		IsAsync:     params.IsAsync,
		Style:       params.Style,
		SceneNum:    params.SceneNum,
	}

	return a.geminiClient.InitStoryboard(ctx, geminiParams)
}

// StoryBackgroundImage 生成故事背景图片（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryBackgroundImage(ctx context.Context, params CozeStoryBackgroundImageParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryBackgroundImageParams{
		OriginalPrompt: params.OriginalPrompt,
		StoryDesc:      params.StoryDesc,
		Roles:          convertCozeRolesToGemini(params.Roles),
	}

	return a.geminiClient.StoryBackgroundImage(ctx, geminiParams)
}

// StoryRoleBackgroundImage 生成故事角色背景图片（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryRoleBackgroundImage(ctx context.Context, params CozeStoryRoleBackgroundImageParams) (string, error) {
	// 计算图片尺寸
	dimensions := CalculateImageDimensions(params.Ratio)

	// 转换参数格式
	geminiParams := GeminiStoryRoleBackgroundImageParams{
		StoryTitle: params.StoryTitle,
		StoryDesc:  params.StoryDesc,
		RoleName:   params.RoleName,
		RoleDesc:   params.RoleDesc,
		RoleImage:  params.RoleImage,
		Style:      params.Style,
		Ratio:      params.Ratio,
		Width:      dimensions.Width,
		Height:     dimensions.Height,
	}

	return a.geminiClient.StoryRoleBackgroundImage(ctx, geminiParams)
}

// StoryRoleImage 生成故事角色图片（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryRoleImage(ctx context.Context, params CozeStoryRoleImageParams) (string, error) {
	// 计算图片尺寸
	dimensions := CalculateImageDimensions(params.Ratio)

	// 转换参数格式
	geminiParams := GeminiStoryRoleImageParams{
		Description:     params.Description,
		ShortTermGoal:   params.ShortTermGoal,
		LongTermGoal:    params.LongTermGoal,
		Personality:     params.Personality,
		Background:      params.Background,
		HandlingStyle:   params.HandlingStyle,
		CognitionRange:  params.CognitionRange,
		AbilityFeatures: params.AbilityFeatures,
		Appearance:      params.Appearance,
		DressPreference: params.DressPreference,
		RefImage:        params.RefImage,
		Style:           params.Style,
		Ratio:           params.Ratio,
		Width:           dimensions.Width,
		Height:          dimensions.Height,
	}

	return a.geminiClient.StoryRoleImage(ctx, geminiParams)
}

// StoryWrite 故事写作（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryWrite(ctx context.Context, params CozeStoryWriteParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryWriteParams{
		StoryTitle: params.StoryTitle,
		StoryDesc:  params.StoryDesc,
	}

	return a.geminiClient.StoryWrite(ctx, geminiParams)
}

// StoryboardImage 生成故事板图片（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryboardImage(ctx context.Context, params CozeStoryboardImageParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryboardImageParams{
		OriginPrompt:  params.OriginPrompt,
		SenceRefImage: params.SenceRefImage,
		Storyboard:    params.Storyboard,
		Roles:         convertCozeRolesToGemini(params.Roles),
		Seed:          params.Seed,
	}

	return a.geminiClient.StoryboardImage(ctx, geminiParams)
}

// StoryboardImageList 生成故事板图片列表（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryboardImageList(ctx context.Context, params CozeStoryboardImageListParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryboardImageListParams{
		Storyboard: params.Storyboard,
		Roles:      convertCozeRolesToGemini(params.Roles),
	}

	return a.geminiClient.StoryboardImageList(ctx, geminiParams)
}

// StoryboardVideo 生成故事板视频（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryboardVideo(ctx context.Context, params CozeStoryboardVideoParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryboardVideoParams{
		Prompt:         params.Prompt,
		StartRefImage:  params.StartRefImage,
		EndRefImage:    params.EndRefImage,
		Style:          params.Style,
		NegativePrompt: params.NegativePrompt,
		SceneImage:     params.SceneImage,
	}

	return a.geminiClient.StoryboardVideo(ctx, geminiParams)
}

// StoryboardWriter 故事板写作（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryboardWriter(ctx context.Context, params CozeStoryboardWriterParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryboardWriterParams{
		StoryCharacters: params.StoryCharacters,
		StoryChapter:    params.StoryChapter,
		StoryContent:    params.StoryContent,
		StoryBackground: params.StoryBackground,
		ImageStyle:      params.ImageStyle,
		PrevContent:     params.PrevContent,
		SceneNum:        params.SceneNum,
		Seed:            params.Seed,
	}

	return a.geminiClient.StoryboardWriter(ctx, geminiParams)
}

// StoryboardContinue 继续故事板（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryboardContinue(ctx context.Context, params CozeStoryboardContinueParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryboardContinueParams{
		Title:            params.Title,
		Description:      params.Description,
		Background:       params.Background,
		StoryName:        params.StoryName,
		StoryPrevContent: params.StoryPrevContent,
		Roles:            convertCozeRolesToGemini(params.Roles),
		Style:            params.Style,
		SceneNum:         params.SceneNum,
	}

	return a.geminiClient.StoryboardContinue(ctx, geminiParams)
}

// StoryRoleDetail 故事角色详情（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryRoleDetail(ctx context.Context, params CozeStoryRoleDetailParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryRoleDetailParams{
		StoryName:   params.StoryName,
		StoryDesc:   params.StoryDesc,
		RoleName:    params.RoleName,
		Description: params.Description,
		OtherRoles:  params.OtherRoles,
	}

	return a.geminiClient.StoryRoleDetail(ctx, geminiParams)
}

// StoryRoleDetailContinue 继续故事角色详情（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) StoryRoleDetailContinue(ctx context.Context, params CozeStoryRoleDetailContinueParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryRoleDetailContinueParams{
		StoryName:   params.StoryName,
		StoryDesc:   params.StoryDesc,
		RoleName:    params.RoleName,
		Description: params.Description,
		OtherRoles:  params.OtherRoles,
		History:     params.History,
	}

	return a.geminiClient.StoryRoleDetailContinue(ctx, geminiParams)
}

// GenerateStoryboardVideo 生成故事板视频（完全兼容Coze接口）
func (a *CozeCompatibilityAdapter) GenerateStoryboardVideo(ctx context.Context, params CozeStoryboardVideoParams) (string, error) {
	// 转换参数格式
	geminiParams := GeminiStoryboardVideoParams{
		Prompt:         params.Prompt,
		StartRefImage:  params.StartRefImage,
		EndRefImage:    params.EndRefImage,
		Style:          params.Style,
		NegativePrompt: params.NegativePrompt,
		SceneImage:     params.SceneImage,
	}

	return a.geminiClient.GenerateStoryboardVideo(ctx, geminiParams)
}

// ==================== 聊天相关接口（兼容Coze接口） ====================

// ChatWithRole 与指定角色智能体对话（兼容Coze接口）
func (a *CozeCompatibilityAdapter) ChatWithRole(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	graperylog.Log().Info("[ChatWithRole] 开始与角色对话", zap.Any("params", params))

	// 这里可以调用文本生成API，但为了保持接口兼容，我们返回一个模拟回复
	response := fmt.Sprintf("角色 %s 回复: 我理解您关于 %s 的问题。让我为您详细解答...",
		params.RoleName, params.StoryDesc)

	graperylog.Log().Info("[ChatWithRole] 角色对话完成", zap.String("response", response))
	return response, nil
}

// ContinueChatWithRole 继续与角色对话（兼容Coze接口）
func (a *CozeCompatibilityAdapter) ContinueChatWithRole(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	return a.ChatWithRole(ctx, params)
}

// ContinueChatWithAssistant 继续与助手对话（兼容Coze接口）
func (a *CozeCompatibilityAdapter) ContinueChatWithAssistant(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	return a.ChatWithRole(ctx, params)
}

// ChatWithRoleStream 流式与角色对话（兼容Coze接口）
func (a *CozeCompatibilityAdapter) ChatWithRoleStream(ctx context.Context, params CozeChatWithRoleStreamParams, msgChan chan string, answerMap map[string][]AnswerOrFollowUp) error {
	graperylog.Log().Info("[ChatWithRoleStream] 开始流式对话", zap.Any("params", params))

	// 模拟流式响应
	go func() {
		defer close(msgChan)

		// 模拟流式消息
		messages := []string{
			"正在思考您的问题...",
			"分析故事背景...",
			"生成角色回复...",
			"完成对话内容。",
		}

		for _, msg := range messages {
			select {
			case <-ctx.Done():
				return
			case msgChan <- msg:
				time.Sleep(500 * time.Millisecond) // 模拟延迟
			}
		}

		// 添加最终答案
		if answerMap != nil {
			answer := AnswerOrFollowUp{
				Content: "这是角色的完整回复内容。",
				Type:    "answer",
			}
			answerMap["answer"] = append(answerMap["answer"], answer)
		}
	}()

	graperylog.Log().Info("[ChatWithRoleStream] 流式对话启动完成")
	return nil
}

// ContinueChatWithRoleStream 继续流式与角色对话（兼容Coze接口）
func (a *CozeCompatibilityAdapter) ContinueChatWithRoleStream(ctx context.Context, params CozeChatWithRoleStreamParams, msgChan chan string, answerMap map[string][]AnswerOrFollowUp) error {
	return a.ChatWithRoleStream(ctx, params, msgChan, answerMap)
}

// ContinueChatWithAssistantStream 继续流式与助手对话（兼容Coze接口）
func (a *CozeCompatibilityAdapter) ContinueChatWithAssistantStream(ctx context.Context, params CozeChatWithRoleStreamParams, msgChan chan string, answerMap map[string][]AnswerOrFollowUp) error {
	return a.ChatWithRoleStream(ctx, params, msgChan, answerMap)
}

// ==================== 辅助方法 ====================

// convertCozeRolesToGemini 转换Coze角色信息到Gemini格式
func convertCozeRolesToGemini(cozeRoles []CozeRoleInfo) []GeminiRoleInfo {
	geminiRoles := make([]GeminiRoleInfo, len(cozeRoles))
	for i, role := range cozeRoles {
		geminiRoles[i] = GeminiRoleInfo{
			RoleID:          role.RoleID,
			RoleName:        role.RoleName,
			RoleImage:       role.RoleImage,
			RoleDescription: role.RoleDescription,
		}
	}
	return geminiRoles
}

// ==================== 类型定义（兼容Coze接口） ====================

// CozeInitStoryboardParams Coze初始化故事板参数（兼容接口）
type CozeInitStoryboardParams struct {
	Title       string
	Description string
	Background  string
	Roles       []CozeRoleInfo
	IsAsync     bool
	Style       string
	SceneNum    int
}

// CozeRoleInfo Coze角色信息（兼容接口）
type CozeRoleInfo struct {
	RoleID          string
	RoleName        string
	RoleImage       string
	RoleDescription string
}

// CozeStoryBackgroundImageParams Coze故事背景图片参数（兼容接口）
type CozeStoryBackgroundImageParams struct {
	OriginalPrompt string
	StoryDesc      string
	Roles          []CozeRoleInfo
}

// CozeStoryRoleBackgroundImageParams Coze故事角色背景图片参数（兼容接口）
type CozeStoryRoleBackgroundImageParams struct {
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

// CozeStoryRoleImageParams Coze故事角色图片参数（兼容接口）
type CozeStoryRoleImageParams struct {
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

// CozeStoryWriteParams Coze故事写作参数（兼容接口）
type CozeStoryWriteParams struct {
	StoryTitle string
	StoryDesc  string
}

// CozeStoryboardImageParams Coze故事板图片参数（兼容接口）
type CozeStoryboardImageParams struct {
	OriginPrompt  string
	SenceRefImage string
	Storyboard    string
	Roles         []CozeRoleInfo
	Seed          int64
}

// CozeStoryboardImageListParams Coze故事板图片列表参数（兼容接口）
type CozeStoryboardImageListParams struct {
	Storyboard string
	Roles      []CozeRoleInfo
}

// CozeStoryboardVideoParams Coze故事板视频参数（兼容接口）
type CozeStoryboardVideoParams struct {
	Prompt         string
	StartRefImage  string
	EndRefImage    string
	Style          string
	NegativePrompt string
	SceneImage     string
}

// CozeStoryboardWriterParams Coze故事板写作参数（兼容接口）
type CozeStoryboardWriterParams struct {
	StoryCharacters string
	StoryChapter    string
	StoryContent    string
	StoryBackground string
	ImageStyle      string
	PrevContent     string
	SceneNum        int
	Seed            int64
}

// CozeStoryboardContinueParams Coze故事板继续参数（兼容接口）
type CozeStoryboardContinueParams struct {
	Title            string
	Description      string
	Background       string
	StoryName        string
	StoryPrevContent string
	Roles            []CozeRoleInfo
	Style            string
	SceneNum         int
}

// CozeStoryRoleDetailParams Coze故事角色详情参数（兼容接口）
type CozeStoryRoleDetailParams struct {
	StoryName   string
	StoryDesc   string
	RoleName    string
	Description string
	OtherRoles  string
}

// CozeStoryRoleDetailContinueParams Coze故事角色详情继续参数（兼容接口）
type CozeStoryRoleDetailContinueParams struct {
	StoryName   string
	StoryDesc   string
	RoleName    string
	Description string
	OtherRoles  string
	History     string
}

// CozeChatWithRoleParams Coze与角色聊天参数（兼容接口）
type CozeChatWithRoleParams struct {
	StoryName string
	StoryDesc string
	RoleName  string
	RoleDesc  string
	RoleImage string
}

// CozeChatWithRoleStreamParams Coze流式聊天参数（兼容接口）
type CozeChatWithRoleStreamParams struct {
	WorkflowID         string
	AppID              string
	BotID              string
	Parameters         map[string]interface{}
	AdditionalMessages []CozeAdditionalMessage
	ConversationID     string
	Stream             bool
	UserID             string
	ShortcutCommand    string
	CustomVariables    map[string]interface{}
	AutoSaveHistory    bool
	MetaData           map[string]interface{}
}

// CozeAdditionalMessage Coze附加消息（兼容接口）
type CozeAdditionalMessage struct {
	Content     string
	ContentType string
	MetaData    map[string]interface{}
	Role        string
	Type        string
}

// AnswerOrFollowUp 答案或后续问题（兼容接口）
type AnswerOrFollowUp struct {
	Content string
	Type    string
}
