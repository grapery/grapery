package google

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	graperylog "github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

// WorkflowProvider 工作流提供商类型
type WorkflowProvider string

const (
	// CozeProvider Coze提供商
	CozeProvider WorkflowProvider = "coze"
	// GeminiProvider Gemini提供商
	GeminiProvider WorkflowProvider = "gemini"
)

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	Provider          WorkflowProvider `json:"provider"`           // 提供商类型
	APIKey            string           `json:"api_key"`            // API密钥
	AppID             string           `json:"app_id"`             // 应用ID（Coze需要）
	Endpoint          string           `json:"endpoint"`           // API端点
	EnableEnhancement bool             `json:"enable_enhancement"` // 是否启用提示词增强
}

// WorkflowClient 工作流客户端接口
type WorkflowClient interface {
	// 故事板相关方法
	InitStoryboard(ctx context.Context, params CozeInitStoryboardParams) (string, error)
	StoryBackgroundImage(ctx context.Context, params CozeStoryBackgroundImageParams) (string, error)
	StoryRoleBackgroundImage(ctx context.Context, params CozeStoryRoleBackgroundImageParams) (string, error)
	StoryRoleImage(ctx context.Context, params CozeStoryRoleImageParams) (string, error)
	StoryWrite(ctx context.Context, params CozeStoryWriteParams) (string, error)
	StoryboardImage(ctx context.Context, params CozeStoryboardImageParams) (string, error)
	StoryboardImageList(ctx context.Context, params CozeStoryboardImageListParams) (string, error)
	StoryboardVideo(ctx context.Context, params CozeStoryboardVideoParams) (string, error)
	StoryboardWriter(ctx context.Context, params CozeStoryboardWriterParams) (string, error)
	StoryboardContinue(ctx context.Context, params CozeStoryboardContinueParams) (string, error)
	StoryRoleDetail(ctx context.Context, params CozeStoryRoleDetailParams) (string, error)
	StoryRoleDetailContinue(ctx context.Context, params CozeStoryRoleDetailContinueParams) (string, error)
	GenerateStoryboardVideo(ctx context.Context, params CozeStoryboardVideoParams) (string, error)

	// 聊天相关方法
	ChatWithRole(ctx context.Context, params CozeChatWithRoleParams) (string, error)
	ContinueChatWithRole(ctx context.Context, params CozeChatWithRoleParams) (string, error)
	ContinueChatWithAssistant(ctx context.Context, params CozeChatWithRoleParams) (string, error)
	ChatWithRoleStream(ctx context.Context, params CozeChatWithRoleStreamParams, msgChan chan string, answerMap map[string][]AnswerOrFollowUp) error
	ContinueChatWithRoleStream(ctx context.Context, params CozeChatWithRoleStreamParams, msgChan chan string, answerMap map[string][]AnswerOrFollowUp) error
	ContinueChatWithAssistantStream(ctx context.Context, params CozeChatWithRoleStreamParams, msgChan chan string, answerMap map[string][]AnswerOrFollowUp) error

	// 资源管理
	Close() error
}

// WorkflowFactory 工作流工厂
type WorkflowFactory struct {
	config WorkflowConfig
}

// NewWorkflowFactory 创建工作流工厂
func NewWorkflowFactory(config WorkflowConfig) *WorkflowFactory {
	return &WorkflowFactory{
		config: config,
	}
}

// CreateClient 创建工作流客户端
func (f *WorkflowFactory) CreateClient() (WorkflowClient, error) {
	switch f.config.Provider {
	case CozeProvider:
		return f.createCozeClient()
	case GeminiProvider:
		return f.createGeminiClient()
	default:
		return nil, fmt.Errorf("不支持的工作流提供商: %s", f.config.Provider)
	}
}

// createCozeClient 创建Coze客户端
func (f *WorkflowFactory) createCozeClient() (WorkflowClient, error) {
	// 这里需要导入coze包，但为了避免循环依赖，我们返回一个错误
	// 实际使用时，应该在业务层处理Coze客户端的创建
	return nil, errors.New("coze客户端需要在业务层创建，请使用coze包")
}

// createGeminiClient 创建Gemini客户端
func (f *WorkflowFactory) createGeminiClient() (WorkflowClient, error) {
	if f.config.APIKey == "" {
		return nil, errors.New("Gemini API密钥不能为空")
	}

	client, err := NewCozeCompatibilityAdapterWithOptions(f.config.APIKey, f.config.EnableEnhancement)
	if err != nil {
		return nil, fmt.Errorf("创建Gemini客户端失败: %w", err)
	}

	graperylog.Log().Info("[WorkflowFactory] Gemini客户端创建成功", zap.Bool("enhancement", f.config.EnableEnhancement))
	return client, nil
}

// ==================== 便捷函数 ====================

// CreateGeminiWorkflowClient 创建Gemini工作流客户端（便捷函数）
func CreateGeminiWorkflowClient(apiKey string) (WorkflowClient, error) {
	return CreateGeminiWorkflowClientWithOptions(apiKey, true) // 默认启用提示词增强
}

// CreateGeminiWorkflowClientWithOptions 创建Gemini工作流客户端（带选项）
func CreateGeminiWorkflowClientWithOptions(apiKey string, enableEnhancement bool) (WorkflowClient, error) {
	config := WorkflowConfig{
		Provider:          GeminiProvider,
		APIKey:            apiKey,
		EnableEnhancement: enableEnhancement,
	}

	factory := NewWorkflowFactory(config)
	return factory.CreateClient()
}

// CreateWorkflowClientFromEnv 从环境变量创建工作流客户端（便捷函数）
func CreateWorkflowClientFromEnv() (WorkflowClient, error) {
	provider := os.Getenv("WORKFLOW_PROVIDER")
	if provider == "" {
		provider = string(GeminiProvider) // 默认使用Gemini
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
	}

	if apiKey == "" {
		return nil, errors.New("未找到API密钥，请设置GOOGLE_API_KEY或API_KEY环境变量")
	}

	config := WorkflowConfig{
		Provider: WorkflowProvider(provider),
		APIKey:   apiKey,
		AppID:    os.Getenv("APP_ID"),
		Endpoint: os.Getenv("WORKFLOW_ENDPOINT"),
	}

	factory := NewWorkflowFactory(config)
	return factory.CreateClient()
}

// ==================== 工作流管理器 ====================

// WorkflowManager 工作流管理器
type WorkflowManager struct {
	client WorkflowClient
	config WorkflowConfig
}

// NewWorkflowManager 创建工作流管理器
func NewWorkflowManager(config WorkflowConfig) (*WorkflowManager, error) {
	factory := NewWorkflowFactory(config)
	client, err := factory.CreateClient()
	if err != nil {
		return nil, err
	}

	return &WorkflowManager{
		client: client,
		config: config,
	}, nil
}

// GetClient 获取工作流客户端
func (m *WorkflowManager) GetClient() WorkflowClient {
	return m.client
}

// SwitchProvider 切换提供商
func (m *WorkflowManager) SwitchProvider(provider WorkflowProvider, apiKey string) error {
	// 关闭当前客户端
	if m.client != nil {
		m.client.Close()
	}

	// 更新配置
	m.config.Provider = provider
	m.config.APIKey = apiKey

	// 创建新客户端
	factory := NewWorkflowFactory(m.config)
	client, err := factory.CreateClient()
	if err != nil {
		return err
	}

	m.client = client
	graperylog.Log().Info("[WorkflowManager] 提供商切换成功", zap.String("provider", string(provider)))
	return nil
}

// Close 关闭管理器
func (m *WorkflowManager) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// ==================== 示例和测试 ====================

// ExampleWorkflowFactoryUsage 工作流工厂使用示例
func ExampleWorkflowFactoryUsage() {
	// 方式1: 直接创建Gemini客户端
	client, err := CreateGeminiWorkflowClient("your-api-key")
	if err != nil {
		graperylog.Log().Error("创建客户端失败", zap.Error(err))
		return
	}
	defer client.Close()

	// 方式2: 从环境变量创建
	client2, err := CreateWorkflowClientFromEnv()
	if err != nil {
		graperylog.Log().Error("从环境变量创建客户端失败", zap.Error(err))
		return
	}
	defer client2.Close()

	// 方式3: 使用管理器
	config := WorkflowConfig{
		Provider: GeminiProvider,
		APIKey:   "your-api-key",
	}

	manager, err := NewWorkflowManager(config)
	if err != nil {
		graperylog.Log().Error("创建管理器失败", zap.Error(err))
		return
	}
	defer manager.Close()

	// 使用客户端
	ctx := context.Background()
	params := CozeInitStoryboardParams{
		Title:       "测试故事",
		Description: "这是一个测试故事",
		Background:  "现代都市",
		Style:       "现实主义",
		SceneNum:    5,
	}

	result, err := manager.GetClient().InitStoryboard(ctx, params)
	if err != nil {
		graperylog.Log().Error("初始化故事板失败", zap.Error(err))
		return
	}

	graperylog.Log().Info("故事板初始化成功", zap.String("result", result))
}

// TestWorkflowCompatibility 测试工作流兼容性
func TestWorkflowCompatibility() {
	// 创建测试客户端
	client, err := CreateGeminiWorkflowClient("test-api-key")
	if err != nil {
		graperylog.Log().Error("[TestWorkflowCompatibility] 创建客户端失败", zap.Error(err))
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试各种接口
	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "InitStoryboard",
			test: func() error {
				params := CozeInitStoryboardParams{
					Title:       "测试故事",
					Description: "测试描述",
					Background:  "测试背景",
					Style:       "测试风格",
					SceneNum:    3,
				}
				_, err := client.InitStoryboard(ctx, params)
				return err
			},
		},
		{
			name: "StoryWrite",
			test: func() error {
				params := CozeStoryWriteParams{
					StoryTitle: "测试故事标题",
					StoryDesc:  "测试故事描述",
				}
				_, err := client.StoryWrite(ctx, params)
				return err
			},
		},
		{
			name: "ChatWithRole",
			test: func() error {
				params := CozeChatWithRoleParams{
					StoryName: "测试故事",
					StoryDesc: "测试对话",
					RoleName:  "测试角色",
				}
				_, err := client.ChatWithRole(ctx, params)
				return err
			},
		},
	}

	// 运行测试
	for _, test := range tests {
		if err := test.test(); err != nil {
			graperylog.Log().Error("[TestWorkflowCompatibility] 测试失败",
				zap.String("test", test.name), zap.Error(err))
		} else {
			graperylog.Log().Info("[TestWorkflowCompatibility] 测试通过",
				zap.String("test", test.name))
		}
	}
}
